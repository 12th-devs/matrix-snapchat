import { chromium } from "playwright";
import crypto from "node:crypto";
import fs from "node:fs/promises";
import path from "node:path";

const SNAPCHAT_URL = "https://web.snapchat.com";
const SNAPCHAT_HOST_PATTERN = /(^|\.)snapchat\.com$/i;
const profileDir = process.env.SNAPCHAT_PROFILE_DIR || path.resolve("../data/snapchat-profile");
const headless = process.env.SNAPCHAT_HEADLESS === "true";
const traceDir = process.env.SNAPCHAT_TRACE_DIR || path.resolve("../data/debug");

let context;
let page;
let inflight;
const pendingTasks = [];

const loginTextPattern = /log in|login|sign up|continue on phone|scan|qr|use mobile app|use your phone/i;
const shellTextPattern = /chat|chats|search|conversations|send a chat|new chat/i;
const ignoreChatPattern = /^(search|camera|stories|spotlight|map|settings|profile|my ai|discover|for you|click to install the desktop app|try the camera.*)$/i;
const composerPattern = /send a chat|message|type a chat|write a chat|send message/i;
const chatDetailPattern = /(opened|received|sent|replied|reply|say hi!?|typing|new|call|\d+\s*[mhdwy])/i;
const ignoreMessagePattern = /^(notifications are off|turn on|call|reply|close chat|drag & drop to upload|search|not now|enable notifications|click to install the desktop app|to always have access to your chats!)$/i;
const dateDividerPattern = /^(today|yesterday|march|april|may|june|july|august|september|october|november|december)(\s+\d{1,2})?$/i;
const unsupportedMediaLabel = "[Unsupported Snapchat snap/media]";
const unsupportedEventLabel = "[Unsupported Snapchat event]";
const snapchatErrorPattern = /oops!? something went wrong|reload the page|error id:/i;
const ignoreMessageTokenPattern = /^(opened|received|sent|say hi!?|typing|my ai|🔥|·|\d+\s*[mhdwy]|\d+)$/i;

function normalizeText(value) {
  return String(value || "").replace(/\s+/g, " ").trim();
}

function prettifyChatText(value) {
  return normalizeText(
    value
      .replace(/Close Chat/gi, "")
      .replace(/·/g, " · ")
      .replace(/(Opened|Received|Sent|Replied|Reply|Say Hi!?|Typing)/g, " $1 ")
      .replace(/(\d+\s*[mhdwy])/gi, " $1 ")
      .replace(/(🔥)/g, " $1 ")
  );
}

function stableID(parts) {
  return crypto.createHash("sha1").update(parts.join("::")).digest("hex").slice(0, 16);
}

function makeChatID(name) {
  return stableID([normalizeText(name)]);
}

function looksLikeConversationID(value) {
  return /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(String(value || "").trim());
}

function conversationURL(chatID) {
  return `${SNAPCHAT_URL}/${chatID}`;
}

function getConversationIDFromURL(value) {
  try {
    const url = new URL(value);
    const match = url.pathname.match(/^\/web\/([0-9a-f-]+)$/i);
    return match ? match[1] : "";
  } catch {
    return "";
  }
}

function inferAuthorFromEventText(text, fallbackName = "") {
  const normalized = normalizeText(text);
  if (!normalized) {
    return fallbackName;
  }
  const noPrefix = normalized.replace(/^\[Unsupported Snapchat event\]\s*/i, "");
  const youAndMatch = noPrefix.match(/^YOU AND (.+?) STARTED A SNAPSTREAK/i);
  if (youAndMatch) {
    return normalizeText(youAndMatch[1]) || fallbackName;
  }
  const triedCallMatch = noPrefix.match(/^(.+?) TRIED TO CALL YOU/i);
  if (triedCallMatch) {
    return normalizeText(triedCallMatch[1]) || fallbackName;
  }
  return fallbackName;
}

function inferEventAuthor(text, fallbackName = "") {
  const normalized = normalizeText(text);
  if (!normalized.startsWith(unsupportedEventLabel)) {
    return "";
  }
  if (/started a snapstreak|tried to call you/i.test(normalized)) {
    return "System";
  }
  return inferAuthorFromEventText(text, fallbackName);
}

function runNextTask() {
  if (inflight || pendingTasks.length === 0) {
    return;
  }

  pendingTasks.sort((a, b) => a.priority - b.priority || a.id - b.id);
  const next = pendingTasks.shift();
  const promise = Promise.resolve()
    .then(next.fn)
    .then(next.resolve, next.reject)
    .finally(() => {
      inflight = null;
      runNextTask();
    });
  inflight = promise;
}

let taskID = 0;
function withLock(fn, { priority = 1 } = {}) {
  return new Promise((resolve, reject) => {
    pendingTasks.push({
      fn,
      priority,
      resolve,
      reject,
      id: taskID++,
    });
    runNextTask();
  });
}

function isSnapchatURL(value) {
  try {
    const url = new URL(value);
    return SNAPCHAT_HOST_PATTERN.test(url.hostname);
  } catch {
    return false;
  }
}

function isBlankURL(value) {
  return !value || value === "about:blank" || value.startsWith("chrome://newtab");
}

function isConversationPath(value) {
  try {
    const url = new URL(value);
    return /^\/web\/.+/.test(url.pathname);
  } catch {
    return false;
  }
}

function pickBestPage(pages) {
  const openPages = pages.filter((candidate) => !candidate.isClosed());
  const snapchatPages = openPages.filter((candidate) => isSnapchatURL(candidate.url()));

  if (snapchatPages.length > 0) {
    return snapchatPages[snapchatPages.length - 1];
  }
  if (openPages.length > 0) {
    return openPages[openPages.length - 1];
  }
  return null;
}

async function resetSession() {
  try {
    if (context) {
      await context.close();
    }
  } catch {
  }
  context = undefined;
  page = undefined;
}

async function ensureSession() {
  if (context && page && !page.isClosed()) {
    return { context, page };
  }

  await fs.mkdir(profileDir, { recursive: true });
  await fs.mkdir(traceDir, { recursive: true });

  context = await chromium.launchPersistentContext(profileDir, {
    headless,
    viewport: { width: 1440, height: 960 },
    args: [
      "--disable-blink-features=AutomationControlled",
      "--disable-dev-shm-usage",
      "--no-first-run",
    ],
  });

  context.on("page", (newPage) => {
    page = newPage;
    page.setDefaultTimeout(15000);
  });

  page = pickBestPage(context.pages()) || (await context.newPage());
  page.setDefaultTimeout(15000);
  page.on("dialog", async (dialog) => {
    try {
      await dialog.dismiss();
    } catch {
    }
  });

  return { context, page };
}

async function gotoSnapchat({ reload = false } = {}) {
  try {
    return await gotoSnapchatOnce({ reload });
  } catch (error) {
    if (!/has been closed|target closed|browser has been closed/i.test(String(error))) {
      throw error;
    }
    await resetSession();
    return gotoSnapchatOnce({ reload });
  }
}

async function gotoSnapchatOnce({ reload = false } = {}) {
  const { context: currentContext, page: initialPage } = await ensureSession();
  const currentPage = pickBestPage(currentContext.pages()) || initialPage;

  const navigate = async (action) => {
    try {
      await action();
    } catch (error) {
      if (!isSnapchatURL(currentPage.url())) {
        throw error;
      }
    }
  };

  if (isBlankURL(currentPage.url())) {
    await navigate(() => currentPage.goto(SNAPCHAT_URL, { waitUntil: "domcontentloaded" }));
  } else if (!isSnapchatURL(currentPage.url())) {
    await navigate(() => currentPage.goto(SNAPCHAT_URL, { waitUntil: "domcontentloaded" }));
  } else if (reload && currentPage.url().startsWith(SNAPCHAT_URL)) {
    await navigate(() => currentPage.reload({ waitUntil: "domcontentloaded" }));
  }

  await currentPage.waitForTimeout(1500);
  const state = await detectState(currentPage);
  if (state === "error") {
    await navigate(() => currentPage.goto(SNAPCHAT_URL, { waitUntil: "domcontentloaded" }));
    await currentPage.waitForTimeout(1500);
    if ((await detectState(currentPage)) === "error") {
      await navigate(() => currentPage.reload({ waitUntil: "domcontentloaded" }));
      await currentPage.waitForTimeout(1500);
    }
  }
  return currentPage;
}

async function firstVisibleLocator(candidates) {
  for (const locator of candidates) {
    try {
      if ((await locator.count()) > 0 && (await locator.first().isVisible())) {
        return locator.first();
      }
    } catch {
    }
  }
  return null;
}

async function detectState(currentPage) {
  const loginHints = [
    currentPage.getByText(loginTextPattern),
    currentPage.locator("[data-testid*='login'], [aria-label*='login' i]"),
  ];
  const shellHints = [
    currentPage.getByRole("textbox"),
    currentPage.getByText(shellTextPattern),
    currentPage.locator("[aria-label*='search' i], [placeholder*='search' i]"),
  ];

  const loginVisible = Boolean(await firstVisibleLocator(loginHints));
  const shellVisible = Boolean(await firstVisibleLocator(shellHints));
  const errorVisible = await currentPage.evaluate((source) => {
    const matcher = new RegExp(source, "i");
    return matcher.test((document.body?.innerText || "").replace(/\s+/g, " ").trim());
  }, snapchatErrorPattern.source);

  if (shellVisible && !loginVisible) {
    return "ready";
  }
  if (errorVisible) {
    return "error";
  }
  if (loginVisible) {
    return "login_required";
  }
  return "loading";
}

async function isLoggedIn(currentPage) {
  return (await detectState(currentPage)) === "ready";
}

async function getSearchBox(currentPage) {
  const locator = await firstVisibleLocator([
    currentPage.locator("[aria-label*='search' i]"),
    currentPage.locator("[placeholder*='search' i]"),
    currentPage.locator("input[type='search']"),
    currentPage.locator("input[type='text']"),
    currentPage.getByRole("textbox"),
  ]);

  if (!locator) {
    throw new Error("could not find Snapchat chat search input");
  }

  return locator;
}

async function clearSearchBox(currentPage) {
  const searchBox = await firstVisibleLocator([
    currentPage.locator("input[role='searchbox']"),
    currentPage.locator("[role='searchbox'] input"),
    currentPage.locator("[aria-label*='search' i]"),
    currentPage.locator("[placeholder*='search' i]"),
  ]);

  if (!searchBox) {
    return;
  }

  try {
    await searchBox.focus();
  } catch {
  }

  try {
    await searchBox.fill("");
  } catch {
    try {
      await currentPage.keyboard.press(process.platform === "darwin" ? "Meta+A" : "Control+A");
      await currentPage.keyboard.press("Backspace");
    } catch {
    }
  }

  try {
    await currentPage.keyboard.press("Escape");
  } catch {
  }

  await currentPage.waitForTimeout(250);
}

async function dismissInterferingOverlays(currentPage) {
  const buttons = [
    currentPage.getByRole("button", { name: /^not now$/i }),
    currentPage.getByRole("button", { name: /^close$/i }),
  ];

  for (const locator of buttons) {
    try {
      if ((await locator.count()) > 0 && (await locator.first().isVisible())) {
        await locator.first().click({ force: true });
      }
    } catch {
    }
  }
}

async function findChatRow(currentPage, chatName) {
  return firstVisibleLocator([
    currentPage.locator("[data-testid*='conversation'], [data-testid*='chat']").filter({
      hasText: new RegExp(escapeRegExp(chatName), "i"),
    }),
    currentPage.getByRole("button", { name: new RegExp(escapeRegExp(chatName), "i") }),
    currentPage.getByRole("link", { name: new RegExp(escapeRegExp(chatName), "i") }),
    currentPage.locator("button, a, [role='button'], [role='link']").filter({
      hasText: new RegExp(escapeRegExp(chatName), "i"),
    }),
    currentPage.getByText(new RegExp(escapeRegExp(chatName), "i")),
  ]);
}

async function findSidebarChatTarget(currentPage, chatName) {
  return currentPage.evaluate((targetName) => {
    document.querySelectorAll("[data-codex-chat-target]").forEach((element) => {
      element.removeAttribute("data-codex-chat-target");
    });
    const normalizedTarget = targetName.replace(/\s+/g, " ").trim().toLowerCase();
    const titleSpans = Array.from(document.querySelectorAll("span[id^='title-']"));
    for (const span of titleSpans) {
      const text = (span.textContent || "").replace(/\s+/g, " ").trim();
      if (!text || text.toLowerCase() !== normalizedTarget) {
        continue;
      }

      const row = span.closest("[role='button'][aria-labelledby*='title-'], [role='link'][aria-labelledby*='title-']");
      if (!row) {
        continue;
      }
      const rect = row.getBoundingClientRect();
      if (!rect.width || !rect.height || rect.left > Math.max(window.innerWidth * 0.45, 420)) {
        continue;
      }
      row.scrollIntoView({ block: "center" });
      row.setAttribute("data-codex-chat-target", "1");
      return { name: text, exact: true };
    }

    const selectors = [
      "[data-testid*='conversation']",
      "[data-testid*='chat']",
      "[role='listitem']",
      "button",
      "a",
      "[role='button']",
      "[role='link']",
    ];
    const escaped = targetName.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
    const matcher = new RegExp(escaped, "i");
    const widthLimit = Math.max(window.innerWidth * 0.45, 420);

    const candidates = Array.from(document.querySelectorAll(selectors.join(",")))
      .map((element) => {
        const rect = element.getBoundingClientRect();
        const text = (element.textContent || "").replace(/\s+/g, " ").trim();
        return { element, rect, text };
      })
      .filter(({ element, rect, text }) =>
        rect.width &&
        rect.height &&
        rect.left < widthLimit &&
        rect.height >= 24 &&
        rect.height <= 160 &&
        matcher.test(text) &&
        !/click to install the desktop app|try the camera/i.test(text) &&
        element.getAttribute("role") !== "searchbox" &&
        element.tagName !== "INPUT" &&
        !element.querySelector("input, [role='searchbox']")
      )
      .sort((a, b) => a.rect.top - b.rect.top);

    const match = candidates[0];
    if (!match) {
      return null;
    }

    match.element.scrollIntoView({ block: "center" });
    match.element.setAttribute("data-codex-chat-target", "1");
    return { name: match.text };
  }, chatName);
}

async function getConversationState(currentPage, chatName = "") {
  return currentPage.evaluate((targetName) => {
    const rightPaneLeft = Math.max(window.innerWidth * 0.28, 300);
    const normalizedTarget = (targetName || "").trim().toLowerCase();

    const headerTexts = Array.from(document.querySelectorAll("h1, h2, h3, header, [role='heading'], [data-testid*='header']"))
      .map((element) => {
        const rect = element.getBoundingClientRect();
        const text = (element.textContent || "").replace(/\s+/g, " ").trim();
        return { text, rect };
      })
      .filter(({ text, rect }) => text && rect.width && rect.height && rect.left >= rightPaneLeft && rect.top < 220)
      .map(({ text }) => text);

    const composer = Array.from(document.querySelectorAll("textarea, input, [contenteditable='true'], [role='textbox']"))
      .map((element) => {
        const rect = element.getBoundingClientRect();
        return {
          rect,
          placeholder: element.getAttribute("placeholder") || "",
          aria: element.getAttribute("aria-label") || "",
        };
      })
      .find(({ rect, placeholder, aria }) =>
        rect.width &&
        rect.height &&
        rect.left >= rightPaneLeft &&
        rect.bottom >= window.innerHeight * 0.6 &&
        !/search/i.test(`${placeholder} ${aria}`)
      );

    return {
      url: window.location.href,
      hasConversationURL: /^\/web\/.+/.test(window.location.pathname),
      headerTexts,
      hasTargetHeader: Boolean(normalizedTarget && headerTexts.some((text) => text.toLowerCase().includes(normalizedTarget))),
      hasComposer: Boolean(composer),
    };
  }, chatName);
}

async function waitForConversationOpen(currentPage, chatName, previousURL = "", chatID = "") {
  for (let attempt = 0; attempt < 16; attempt += 1) {
    await currentPage.waitForTimeout(250);
    const state = await getConversationState(currentPage, chatName);
    const urlChanged = state.url !== previousURL && isConversationPath(state.url);
    const openedCorrectID = Boolean(chatID && state.url.includes(`/${chatID}`));
    if ((state.hasTargetHeader || urlChanged) && state.hasComposer) {
      if (!chatID || openedCorrectID) {
        return state;
      }
    }
    if (openedCorrectID && state.hasComposer) {
      return state;
    }
  }

  const state = await getConversationState(currentPage, chatName);
  throw new Error(
    `chat "${chatName}" did not open cleanly (url=${state.url}, headers=${state.headerTexts.join(" | ") || "none"}, composer=${state.hasComposer})`
  );
}

async function openChat(chatName = "", chatID = "", chatURL = "") {
  const currentPage = await gotoSnapchat();
  if (!(await isLoggedIn(currentPage))) {
    throw new Error("snapchat session is not logged in");
  }

  await dismissInterferingOverlays(currentPage);
  await clearSearchBox(currentPage);
  const previousURL = currentPage.url();

  if (chatURL && isSnapchatURL(chatURL)) {
    await currentPage.goto(chatURL, { waitUntil: "domcontentloaded" });
    const activeID = getConversationIDFromURL(chatURL) || chatID;
    await waitForConversationOpen(currentPage, chatName, previousURL, activeID);
    return currentPage;
  }

  if (looksLikeConversationID(chatID)) {
    await currentPage.goto(conversationURL(chatID), { waitUntil: "domcontentloaded" });
    await waitForConversationOpen(currentPage, chatName, previousURL, chatID);
    return currentPage;
  }

  let target = await findSidebarChatTarget(currentPage, chatName);
  if (target) {
    await currentPage.locator("[data-codex-chat-target='1']").first().click({ force: true });
    await waitForConversationOpen(currentPage, chatName, previousURL, chatID);
    return currentPage;
  }

  const searchBox = await getSearchBox(currentPage);
  try {
    await searchBox.focus();
    await searchBox.fill("");
    await searchBox.fill(chatName);
    await currentPage.waitForTimeout(900);
    target = await findSidebarChatTarget(currentPage, chatName);
    if (target) {
      await currentPage.locator("[data-codex-chat-target='1']").first().click({ force: true });
      await waitForConversationOpen(currentPage, chatName, previousURL, chatID);
      return currentPage;
    }
    await searchBox.press("Enter");
    await waitForConversationOpen(currentPage, chatName, previousURL, chatID);
    return currentPage;
  } catch {
  }
  let result = await findChatRow(currentPage, chatName);
  if (!result) {
    try {
      await searchBox.focus();
    } catch {
    }
    try {
      await searchBox.fill("");
    } catch {
      await currentPage.keyboard.press(process.platform === "darwin" ? "Meta+A" : "Control+A");
      await currentPage.keyboard.press("Backspace");
    }
    await searchBox.fill(chatName);
    await currentPage.waitForTimeout(900);
    target = await findSidebarChatTarget(currentPage, chatName);
    if (target) {
      await currentPage.locator("[data-codex-chat-target='1']").first().click({ force: true });
      await waitForConversationOpen(currentPage, chatName, previousURL, chatID);
      return currentPage;
    }
    result = await findChatRow(currentPage, chatName);
  }

  if (!result) {
    throw new Error(`could not find chat named "${chatName}"`);
  }

  await result.click({ force: true });
  await waitForConversationOpen(currentPage, chatName, previousURL, chatID);
  return currentPage;
}

function escapeRegExp(value) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

function parseChatRows(rows) {
  const deduped = [];
  const seen = new Set();

  for (const row of rows) {
    const name = normalizeText(row.name);
    if (!name || ignoreChatPattern.test(name)) {
      continue;
    }
    if (seen.has(name.toLowerCase())) {
      continue;
    }
    seen.add(name.toLowerCase());
    const detail = prettifyChatText(row.preview || row.lastMessage || "").replace(new RegExp(`^${escapeRegExp(name)}`, "i"), "").trim();
    deduped.push({
      id: row.id || makeChatID(name),
      url: row.url || (row.id ? conversationURL(row.id) : ""),
      name,
      preview: detail,
      unread: Boolean(row.unread),
      lastMessage: detail,
    });
  }

  return deduped.slice(0, 100);
}

async function extractChats(currentPage) {
  if ((await detectState(currentPage)) === "error") {
    currentPage = await gotoSnapchat({ reload: true });
  }
  const rows = await currentPage.evaluate(() => {
    const results = [];
    const widthLimit = Math.max(window.innerWidth * 0.45, 420);

    for (const row of document.querySelectorAll("[role='button'][aria-labelledby*='title-'], [role='link'][aria-labelledby*='title-']")) {
      const rect = row.getBoundingClientRect();
      if (!rect.width || !rect.height || rect.left > widthLimit || rect.height < 24 || rect.height > 160) {
        continue;
      }

      const titleSpan = row.querySelector("span[id^='title-']");
      const rowID = titleSpan?.id?.startsWith("title-") ? titleSpan.id.slice(6) : "";
      const name = (titleSpan?.textContent || "").replace(/\s+/g, " ").trim();
      const status = rowID ? (row.querySelector(`#status-${rowID}`)?.textContent || "") : "";
      const time = row.querySelector("time")?.textContent || "";
      const text = (row.textContent || "").replace(/\s+/g, " ").trim();
      const preview = [status, time].map((value) => value.replace(/\s+/g, " ").trim()).filter(Boolean).join(" · ");
      const aria = row.getAttribute("aria-label") || "";
      const className = typeof row.className === "string" ? row.className : "";

      if (!name) {
        continue;
      }

      results.push({
        id: rowID,
        url: rowID ? `${window.location.origin}/web/${rowID}` : "",
        name,
        preview,
        unread: /unread|new/i.test(`${aria} ${className} ${text}`),
        lastMessage: preview,
      });
    }

    return results;
  });

  return parseChatRows(rows);
}

async function extractMessages(currentPage, activeChatName, activeChatID = "") {
  const state = await getConversationState(currentPage, activeChatName);
  if (!state.hasComposer) {
    return [];
  }

  const messages = await currentPage.evaluate(({ activeName, activeID, unsupportedLabel, unsupportedEvent, dateDividerSource }) => {
    const normalize = (value) => String(value || "").replace(/\s+/g, " ").trim();
    const dateDivider = new RegExp(dateDividerSource, "i");
    const conversationRoot =
      (activeID && document.getElementById(`cv-${activeID}`)) ||
      document.querySelector("ul[id^='cv-']");

    if (!conversationRoot) {
      return [];
    }

    const activeLower = normalize(activeName).toLowerCase();
    const items = [];
    const seen = new Set();
    let currentTimestamp = "";

    for (const item of conversationRoot.querySelectorAll(":scope > li")) {
      const rect = item.getBoundingClientRect();
      if (!rect.width || !rect.height) {
        continue;
      }

      const timeElement = item.querySelector("time");
      if (timeElement && !item.querySelector(".KB4Aq, [dir='auto'], .ogn1z, .ijyxU")) {
        currentTimestamp = normalize(
          timeElement.getAttribute("datetime") ||
          timeElement.getAttribute("title") ||
          timeElement.textContent
        );
        continue;
      }

      const headerText = normalize(item.querySelector("header")?.textContent || "");
      const outgoing = /^me$/i.test(headerText);
      const author = outgoing ? "You" : (headerText || activeName || "");
      const textNodes = Array.from(item.querySelectorAll("[dir='auto'], .ogn1z, .ijyxU, .mZgqh"))
        .map((node) => normalize(node.textContent))
        .filter(Boolean);

      let text = textNodes.length > 0 ? textNodes[textNodes.length - 1] : "";
      const hasMedia = item.querySelectorAll("img, video, canvas, input[type='file']").length > 0;
      const hasBubble = Boolean(item.querySelector(".KB4Aq"));
      const isStatusOnly = Boolean(item.querySelector(".mZgqh")) && !item.querySelector("[dir='auto'], .ogn1z");

      if (!text && !hasMedia && !hasBubble) {
        continue;
      }
      if (/^(drag & drop to upload|reply|call)$/i.test(text)) {
        continue;
      }
      if (/^you are using snapchat for web$/i.test(text)) {
        continue;
      }
      if (/started a snapstreak|tried to call you/i.test(text)) {
        text = `${unsupportedEvent} ${text}`;
      } else if (isStatusOnly) {
        continue;
      } else if (!text && hasMedia) {
        text = unsupportedLabel;
      } else if (!text) {
        continue;
      }

      const lower = text.toLowerCase();
      if (!text || lower === activeLower || dateDivider.test(text)) {
        continue;
      }

      const key = `${outgoing ? "out" : "in"}:${Math.round(rect.top / 6)}:${text}`;
      if (seen.has(key)) {
        continue;
      }
      seen.add(key);
      items.push({ author, text, top: rect.top, outgoing, timestamp: currentTimestamp });
    }

    return items.sort((a, b) => a.top - b.top).slice(-80);
  }, {
    activeName: activeChatName,
    activeID: activeChatID,
    unsupportedLabel: unsupportedMediaLabel,
    unsupportedEvent: unsupportedEventLabel,
    dateDividerSource: dateDividerPattern.source,
  });

  return messages.map((message, index) => ({
    id: stableID([String(index), message.text, String(Math.round(message.top)), message.outgoing ? "out" : "in"]),
    author: message.author || (message.outgoing ? "You" : inferEventAuthor(message.text, activeChatName) || inferAuthorFromEventText(message.text, activeChatName) || "Unknown"),
    text: message.text,
    timestamp: message.timestamp || "",
    outgoing: message.outgoing,
  }));
}

async function resolveChat(currentPage, chatID, chatName, chatURL = "") {
  if (chatURL && isSnapchatURL(chatURL)) {
    const urlID = getConversationIDFromURL(chatURL);
    return {
      id: urlID || chatID || "",
      name: chatName || "",
      url: chatURL,
    };
  }

  if (looksLikeConversationID(chatID)) {
    return {
      id: chatID,
      name: chatName || "",
      url: conversationURL(chatID),
    };
  }

  const chats = await extractChats(currentPage);
  if (chatName) {
    const match = chats.find((chat) => chat.name === chatName);
    return {
      id: match?.id || "",
      name: chatName,
      url: match?.url || (match?.id ? conversationURL(match.id) : ""),
    };
  }
  if (!chatID) {
    throw new Error("chatId or chatName is required");
  }

  const match = chats.find((chat) => chat.id === chatID);
  if (!match) {
    throw new Error(`could not resolve chatId "${chatID}"`);
  }
  return { id: match.id, name: match.name, url: match.url || conversationURL(match.id) };
}

async function getComposer(currentPage) {
  const found = await currentPage.evaluate(() => {
    document.querySelectorAll("[data-codex-composer]").forEach((element) => {
      element.removeAttribute("data-codex-composer");
    });

    const rightPaneLeft = Math.max(window.innerWidth * 0.28, 300);
    const candidates = Array.from(document.querySelectorAll("[contenteditable='true'], textarea, input, [role='textbox']"))
      .map((element) => {
        const rect = element.getBoundingClientRect();
        return {
          element,
          rect,
          placeholder: element.getAttribute("placeholder") || "",
          aria: element.getAttribute("aria-label") || "",
          contenteditable: element.getAttribute("contenteditable") || "",
        };
      })
      .filter(({ rect, placeholder, aria, contenteditable }) =>
        rect.width &&
        rect.height &&
        rect.left >= rightPaneLeft &&
        rect.bottom >= window.innerHeight * 0.6 &&
        !/search/i.test(`${placeholder} ${aria}`) &&
        (contenteditable === "true" || /send a chat|message/i.test(`${placeholder} ${aria}`))
      )
      .sort((a, b) => {
        const aEditable = a.contenteditable === "true" ? 1 : 0;
        const bEditable = b.contenteditable === "true" ? 1 : 0;
        return bEditable - aEditable || b.rect.bottom - a.rect.bottom || b.rect.width - a.rect.width;
      });

    const match = candidates[0];
    if (!match) {
      return false;
    }

    match.element.setAttribute("data-codex-composer", "1");
    return true;
  });

  if (!found) {
    throw new Error("could not find Snapchat message composer");
  }

  return currentPage.locator("[data-codex-composer='1']").first();
}

async function getActiveConversation(currentPage, fallbackName = "", fallbackID = "") {
  const state = await getConversationState(currentPage, fallbackName);
  const urlID = getConversationIDFromURL(state.url);
  return {
    id: urlID || fallbackID || "",
    name: state.headerTexts[0] || fallbackName || "",
    url: state.url,
    hasComposer: state.hasComposer,
    headers: state.headerTexts,
  };
}

function sameConversationID(a = "", b = "") {
  const left = normalizeText(a);
  const right = normalizeText(b);
  return Boolean(left && right && left === right);
}

async function readComposerValue(currentPage) {
  return normalizeText(await currentPage.evaluate(() => {
    const target = document.querySelector("[data-codex-composer='1']");
    if (!target) {
      return "";
    }
    if ("value" in target && typeof target.value === "string") {
      return target.value;
    }
    return target.textContent || "";
  }));
}

async function setComposerText(currentPage, composer, text) {
  const value = normalizeText(text);
  await composer.click();

  try {
    await currentPage.evaluate((nextText) => {
      const target = document.querySelector("[data-codex-composer='1']");
      if (!target) {
        return;
      }
      if (target.getAttribute("contenteditable") === "true") {
        target.focus();
        target.textContent = "";
        const selection = window.getSelection();
        const range = document.createRange();
        range.selectNodeContents(target);
        range.collapse(false);
        selection?.removeAllRanges();
        selection?.addRange(range);
        document.execCommand("insertText", false, nextText);
        target.dispatchEvent(new InputEvent("input", { bubbles: true, data: nextText, inputType: "insertText" }));
      }
    }, value);
  } catch {
  }

  try {
    await composer.fill("");
    await composer.fill(value);
  } catch {
    try {
      await composer.press(process.platform === "darwin" ? "Meta+A" : "Control+A");
      await composer.press("Backspace");
    } catch {
    }
    await currentPage.keyboard.type(value);
  }

  const composerValue = await currentPage.evaluate(() => {
    const target = document.querySelector("[data-codex-composer='1']");
    if (!target) {
      return "";
    }
    if ("value" in target && typeof target.value === "string") {
      return target.value;
    }
    return target.textContent || "";
  });
  if (normalizeText(composerValue) !== value) {
    throw new Error("failed to populate Snapchat message composer");
  }
}

async function captureDebugSnapshot(currentPage) {
  const timestamp = new Date().toISOString().replace(/[:.]/g, "-");
  const htmlPath = path.join(traceDir, `snapchat-${timestamp}.html`);
  const screenshotPath = path.join(traceDir, `snapchat-${timestamp}.png`);

  await fs.writeFile(htmlPath, await currentPage.content(), "utf8");
  await currentPage.screenshot({ path: screenshotPath, fullPage: true });

  return { htmlPath, screenshotPath };
}

export async function createSession() {
  return withLock(async () => {
    const currentPage = await gotoSnapchat();
    const state = await detectState(currentPage);
    return {
      state,
      authenticated: state === "ready",
      url: currentPage.url(),
      title: await currentPage.title(),
    };
  }, { priority: 0 });
}

export async function getStatus() {
  return withLock(async () => {
    const currentPage = await gotoSnapchat();
    const state = await detectState(currentPage);
    const chats = state === "ready" ? await extractChats(currentPage) : [];

    return {
      state,
      authenticated: state === "ready",
      activeChatName: await currentPage.title(),
      url: currentPage.url(),
      visibleChatCount: chats.length,
    };
  }, { priority: 0 });
}

export async function getChats() {
  return withLock(async () => {
    const currentPage = await gotoSnapchat();
    if (!(await isLoggedIn(currentPage))) {
      throw new Error("snapchat session is not logged in");
    }

    await clearSearchBox(currentPage);
    await currentPage.waitForTimeout(1200);
    return extractChats(currentPage);
  }, { priority: 2 });
}

export async function getMessages(chatID, chatName, chatURL = "") {
  return withLock(async () => {
    const currentPage = await gotoSnapchat();
    const resolvedChat = await resolveChat(currentPage, chatID, chatName, chatURL);
    const openedPage = await openChat(resolvedChat.name, resolvedChat.id, resolvedChat.url);
    const activeChat = await getActiveConversation(openedPage, resolvedChat.name, resolvedChat.id);
    return extractMessages(openedPage, activeChat.name || resolvedChat.name, activeChat.id || resolvedChat.id);
  }, { priority: 3 });
}

export async function sendMessage(chatID, chatName, chatURL = "", text) {
  return withLock(async () => {
    console.error(`sendMessage: begin chatId=${chatID} chatName=${JSON.stringify(chatName)} chatUrl=${JSON.stringify(chatURL)}`);
    const currentPage = await gotoSnapchat();
    console.error("sendMessage: got session");
    const resolvedChat = await resolveChat(currentPage, chatID, chatName, chatURL);
    console.error(`sendMessage: resolved chat => id=${resolvedChat.id} name=${JSON.stringify(resolvedChat.name)} url=${JSON.stringify(resolvedChat.url)}`);
    const openedPage = await openChat(resolvedChat.name, resolvedChat.id, resolvedChat.url);
    console.error(`sendMessage: opened chat => url=${openedPage.url()}`);
    const activeChat = await getActiveConversation(openedPage, resolvedChat.name, resolvedChat.id);
    console.error(`sendMessage: active conversation => id=${activeChat.id} name=${JSON.stringify(activeChat.name)} url=${JSON.stringify(activeChat.url)} headers=${JSON.stringify(activeChat.headers)}`);
    const composer = await getComposer(openedPage);
    console.error("sendMessage: found composer");

    await setComposerText(openedPage, composer, text);
    console.error(`sendMessage: populated composer => value=${JSON.stringify(await readComposerValue(openedPage))}`);

    const tryConfirm = async () => {
      for (let attempt = 1; attempt <= 4; attempt += 1) {
        await openedPage.waitForTimeout(1200);
        const postSendChat = await getActiveConversation(openedPage, activeChat.name || resolvedChat.name, activeChat.id || resolvedChat.id);
        const messages = await extractMessages(openedPage, activeChat.name || resolvedChat.name, activeChat.id || resolvedChat.id);
        const composerValue = await readComposerValue(openedPage);
        const expectedID = activeChat.id || resolvedChat.id;
        const stayedOnConversation = sameConversationID(postSendChat.id, expectedID);
        const confirmedByMessage = messages.some((message) => message.outgoing && normalizeText(message.text) === normalizeText(text));
        const confirmedByComposerClear = !composerValue && stayedOnConversation;
        const confirmed = confirmedByMessage || confirmedByComposerClear;
        console.error(
          `sendMessage: confirm check attempt=${attempt} confirmed=${confirmed} messageMatch=${confirmedByMessage} composerCleared=${confirmedByComposerClear} stayedOnConversation=${stayedOnConversation} expectedId=${JSON.stringify(expectedID)} actualId=${JSON.stringify(postSendChat.id)} active=${JSON.stringify(postSendChat.name)} url=${JSON.stringify(postSendChat.url)} headers=${JSON.stringify(postSendChat.headers)} messageCount=${messages.length} composer=${JSON.stringify(composerValue)}`
        );
        if (confirmed) {
          return { postSendChat, confirmed };
        }
      }
      const postSendChat = await getActiveConversation(openedPage, activeChat.name || resolvedChat.name, activeChat.id || resolvedChat.id);
      return { postSendChat, confirmed: false };
    };

    const sendButton = await firstVisibleLocator([
      openedPage.getByRole("button", { name: /send/i }),
      openedPage.locator("[aria-label*='send' i]"),
    ]);

    let result;
    if (sendButton) {
      console.error("sendMessage: clicking send button");
      await sendButton.click();
      result = await tryConfirm();
    }
    if (!result?.confirmed) {
      console.error("sendMessage: send button not confirmed, pressing Enter on page keyboard");
      await composer.click();
      await openedPage.keyboard.press("Enter");
      result = await tryConfirm();
    }

    if (!result?.confirmed) {
      const snapshot = await captureDebugSnapshot(openedPage);
      console.error(`sendMessage: confirmation failed snapshot=${snapshot.screenshotPath} html=${snapshot.htmlPath}`);
      throw new Error(
        `message send was not confirmed in chat "${activeChat.name || resolvedChat.name}" (url=${openedPage.url()} snapshot=${snapshot.screenshotPath})`
      );
    }

    console.error("sendMessage: confirmed");

    return {
      queued: true,
      chatID: result.postSendChat.id || activeChat.id || resolvedChat.id || chatID || makeChatID(resolvedChat.name),
      chatName: result.postSendChat.name || activeChat.name || resolvedChat.name,
      chatURL: (result.postSendChat.id && conversationURL(result.postSendChat.id)) || resolvedChat.url || "",
      text,
      confirmed: true,
    };
  }, { priority: 0 });
}

export async function getDiagnostics() {
  return withLock(async () => {
    const currentPage = await gotoSnapchat();
    const state = await detectState(currentPage);
    const snapshot = await captureDebugSnapshot(currentPage);

    const visibleTexts = await currentPage.evaluate(() => {
      const values = [];
      const walker = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
      while (walker.nextNode()) {
        const text = (walker.currentNode.textContent || "").replace(/\s+/g, " ").trim();
        if (!text || text.length < 2 || text.length > 120) {
          continue;
        }
        values.push(text);
      }
      return Array.from(new Set(values)).slice(0, 150);
    });

    const chatNames = state === "ready" ? (await extractChats(currentPage)).map((chat) => chat.name) : [];

    return {
      state,
      authenticated: state === "ready",
      url: currentPage.url(),
      title: await currentPage.title(),
      profileDir,
      headless,
      chatNames: chatNames.slice(0, 30),
      visibleTexts,
      snapshot,
      selectorHints: {
        loginTextPattern: loginTextPattern.source,
        shellTextPattern: shellTextPattern.source,
        composerPattern: composerPattern.source,
      },
    };
  });
}
