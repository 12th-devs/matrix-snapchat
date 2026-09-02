import { chromium } from "playwright";
import fs from "node:fs/promises";
import path from "node:path";
import { createDebugTools } from "./debug-tools.mjs";
import {
  conversationURL,
  extractSelfUserIDFromMessagingRequest,
  inspectProtoUUIDCandidates,
  getConversationIDFromURL,
  inferAuthorFromEventText,
  inferEventAuthor,
  looksLikeConversationID,
  makeChatID,
  normalizeText,
  prettifyChatText,
  stableID,
} from "./utils.mjs";

const SNAPCHAT_URL = "https://www.snapchat.com/web";
const SNAPCHAT_VERSION_URL = "https://www.snapchat.com/web/version.json";
const SNAPCHAT_HOST_PATTERN = /(^|\.)snapchat\.com$/i;
const profileDir = process.env.SNAPCHAT_PROFILE_DIR || path.resolve("../data/snapchat-profile");
const headless = process.env.SNAPCHAT_HEADLESS === "true";
const browserExecutablePath = process.env.SNAPCHAT_BROWSER_EXECUTABLE_PATH?.trim() || undefined;
const traceDir = process.env.SNAPCHAT_TRACE_DIR || path.resolve("../data/debug");
const knownChatsPath = process.env.SNAPCHAT_KNOWN_CHATS_PATH || path.join(path.dirname(profileDir), "known-chats.json");
const DEFAULT_TIMEOUT_MS = Number(process.env.SNAPCHAT_DEFAULT_TIMEOUT_MS || 30000);
const NAVIGATION_TIMEOUT_MS = Number(process.env.SNAPCHAT_NAVIGATION_TIMEOUT_MS || 45000);
const MESSAGE_ATTEMPTS = Math.max(1, Number(process.env.SNAPCHAT_MESSAGE_ATTEMPTS || 1));
const CHAT_DEEP_REFRESH_MS = Math.max(60000, Number(process.env.SNAPCHAT_CHAT_DEEP_REFRESH_MS || 15 * 60 * 1000));
const CHAT_LIST_SETTLE_MS = Math.max(0, Number(process.env.SNAPCHAT_CHAT_LIST_SETTLE_MS || 250));
const CHAT_DEEP_SCAN_MIN_CACHE = Math.max(0, Number(process.env.SNAPCHAT_CHAT_DEEP_SCAN_MIN_CACHE || 25));
const CHAT_OPEN_NAVIGATION_TIMEOUT_MS = Math.max(3000, Number(process.env.SNAPCHAT_CHAT_OPEN_NAVIGATION_TIMEOUT_MS || 8000));
const CONVERSATION_OPEN_ATTEMPTS = Math.max(4, Number(process.env.SNAPCHAT_CONVERSATION_OPEN_ATTEMPTS || 10));
const API_AUTH_WARMUP_TIMEOUT_MS = Math.max(3000, Number(process.env.SNAPCHAT_API_AUTH_WARMUP_TIMEOUT_MS || 10000));
const API_AUTH_CACHE_MS = Math.max(10000, Number(process.env.SNAPCHAT_API_AUTH_CACHE_MS || 5 * 60 * 1000));

let context;
let page;
let inflight;
const pendingTasks = [];
const knownChatsByKey = new Map();
let knownChatsLoaded = false;
let knownChatsDirty = false;
let lastDeepChatSync = 0;
let lastSnapAPIRequestHeaders = {};
let capturedSelfUserID = "";
let capturedSSOToken = "";
let capturedAuthorizationEndpoint = "";
const capturedAuthorizationEndpoints = new Set();
let capturedAuthorizationRequestMeta = {};
let identityWarmupAttempted = false;
let accountIdentityWarmupAttempted = false;
let capturedIdentitySource = "";
const capturedAccountEndpoints = new Set();
let lastAPIAuthSnapshot = null;
let lastAPIAuthSnapshotAt = 0;

function identityUUIDFromJSON(value, pathParts = [], seen = new Set(), depth = 0) {
  if (!value || typeof value !== "object" || seen.has(value) || depth > 8) {
    return "";
  }
  seen.add(value);
  for (const [key, child] of Object.entries(value)) {
    const nextPath = [...pathParts, key];
    if (/^(?:user_?id|userid|ghost_?id|ghostid|account_?id|accountid)$/i.test(key)) {
      const candidate = String(child || "").trim();
      if (looksLikeConversationID(candidate)) {
        return { uuid: candidate.toLowerCase(), path: nextPath.join(".") };
      }
    }
    const nested = identityUUIDFromJSON(child, nextPath, seen, depth + 1);
    if (nested) {
      return nested;
    }
  }
  return "";
}

const loginTextPattern = /log in|login|sign up|continue on phone|scan|qr|use mobile app|use your phone/i;
const shellTextPattern = /chat|chats|search|conversations|send a chat|new chat/i;
const ignoreChatPattern = /^(search|camera|stories|spotlight|map|settings|profile|my ai|discover|for you|click to install the desktop app|try the camera.*)$/i;
const composerPattern = /send a chat|message|type a chat|write a chat|send message/i;
const chatDetailPattern = /(opened|received|sent|replied|reply|say hi!?|typing|new|call|\d+\s*[mhdwy])/i;
const ignoreMessagePattern = /^(notifications are off|turn on|call|reply|close chat|drag & drop to upload|search|not now|enable notifications|click to install the desktop app|to always have access to your chats!)$/i;
const dateDividerPattern = /^(today|yesterday|sun(?:day)?|mon(?:day)?|tue(?:s|sday|day)?|wed(?:nesday)?|thu(?:r|rs|rsday|rday)?|fri(?:day)?|sat(?:urday)?|jan(?:uary)?|feb(?:ruary)?|mar(?:ch)?|apr(?:il)?|may|jun(?:e)?|jul(?:y)?|aug(?:ust)?|sep(?:t|tember)?|oct(?:ober)?|nov(?:ember)?|dec(?:ember)?)(?:\s+\d{1,2})?(?:,?\s+\d{4})?$/i;
const unsupportedMediaLabel = "[Unsupported Snapchat snap/media]";
const unsupportedEventLabel = "[Unsupported Snapchat event]";
const snapchatStatusLabel = "[Snapchat status]";
const snapchatErrorPattern = /oops!? something went wrong|reload the page|error id:/i;
const ignoreMessageTokenPattern = /^(opened|received|sent|say hi!?|typing|my ai|🔥|·|\d+\s*[mhdwy]|\d+)$/i;
const snapPreviewPattern = /\b(new snap|snap|video|photo|picture|media|voice note|audio|sticker|sent you|tap to view|screenshot|screen recorded|opened|received|delivered)\b/i;

function runNextTask() {
  if (inflight || pendingTasks.length === 0) {
    return;
  }

  pendingTasks.sort((a, b) => a.priority - b.priority || a.id - b.id);
  const next = pendingTasks.shift();
  const taskTimeoutMs = Math.max(3000, Number(next.timeoutMs || DEFAULT_TIMEOUT_MS + 5000));
  let taskSettled = false;
  let timeoutFired = false;
  let watchdog;
  const taskPromise = Promise.resolve()
    .then(next.fn)
    .finally(() => {
      taskSettled = true;
      clearTimeout(watchdog);
    });
  const responsePromise = withTimeout(taskPromise, taskTimeoutMs, `connector task ${next.label || next.id}`)
    .then(next.resolve, async (error) => {
      timeoutFired = true;
      try {
        console.error(`connector task ${next.label || next.id} failed: ${String(error)}`);
      } catch {
      }
      try {
        await resetSession();
      } catch {
      }
      next.reject(error);
    });
  watchdog = setTimeout(async () => {
    if (taskSettled || !timeoutFired) {
      return;
    }
    try {
      console.error(`connector task ${next.label || next.id} still running after timeout; forcing queue recovery`);
      await resetSession();
    } catch {
    }
    if (!taskSettled && inflight === promise) {
      inflight = null;
      runNextTask();
    }
  }, taskTimeoutMs + 5000);
  const promise = taskPromise
    .catch(() => {
      // The per-request timeout above may already have returned an error to the
      // caller, but the underlying browser operation can keep running. Keep the
      // queue locked until it actually settles so a second persistent Chrome
      // launch can't race the first one against the same profile.
    })
    .finally(() => {
      inflight = null;
      runNextTask();
    });
  responsePromise.catch(() => {});
  inflight = promise;
}

let taskID = 0;
function withLock(fn, { priority = 1, timeoutMs, label = "" } = {}) {
  return new Promise((resolve, reject) => {
    pendingTasks.push({
      fn,
      priority,
      timeoutMs,
      label,
      resolve,
      reject,
      id: taskID++,
    });
    runNextTask();
  });
}

function withTimeout(promise, timeoutMs, label) {
  let timer;
  return Promise.race([
    Promise.resolve(promise).finally(() => clearTimeout(timer)),
    new Promise((_, reject) => {
      timer = setTimeout(() => reject(new Error(`${label} timed out after ${timeoutMs}ms`)), timeoutMs);
    }),
  ]);
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
    executablePath: browserExecutablePath,
    viewport: { width: 1440, height: 960 },
    args: [
      "--disable-blink-features=AutomationControlled",
      "--disable-dev-shm-usage",
      "--disable-session-crashed-bubble",
      "--disable-restore-session-state",
      "--no-first-run",
    ],
  });

  context.on("request", async (request) => {
    try {
      const requestURL = request.url();
      const headers = await request.allHeaders();
      const authorization = String(headers.authorization || "").trim();
      const authorizationMatch = authorization.match(/^Bearer\s+([^\s]+)$/i);
      if (authorizationMatch?.[1] && authorizationMatch[1].length <= 4096) {
        capturedSSOToken = authorizationMatch[1];
        try {
          const parsed = new URL(requestURL);
          capturedAuthorizationEndpoint = `${parsed.hostname}${parsed.pathname}`
            .replace(/[0-9a-f]{8}-[0-9a-f-]{27,}/gi, ":uuid")
            .slice(0, 240);
          capturedAuthorizationEndpoints.add(capturedAuthorizationEndpoint);
          while (capturedAuthorizationEndpoints.size > 40) {
            capturedAuthorizationEndpoints.delete(capturedAuthorizationEndpoints.values().next().value);
          }
        } catch {
          capturedAuthorizationEndpoint = "unparseable";
        }
        const authorizationPostData = request.postDataBuffer?.();
        capturedAuthorizationRequestMeta = {
          method: request.method(),
          contentType: String(headers["content-type"] || "").slice(0, 120),
          postDataLength: authorizationPostData?.length || 0,
          uuidCandidates: inspectProtoUUIDCandidates(authorizationPostData?.subarray(5) || Buffer.alloc(0))
            .map(({ path, kind }) => ({ path, kind })),
        };
      }
      const postData = request.postDataBuffer?.();
      const selfUserID = extractSelfUserIDFromMessagingRequest(requestURL, postData);
      if (selfUserID) {
        capturedSelfUserID = selfUserID;
      }
      if (!/messagingcoreservice|snapchat\.notification|fidelius/i.test(requestURL)) {
        return;
      }
      const safeHeaders = {};
      for (const [name, value] of Object.entries(headers)) {
        if (/authorization|cookie/i.test(name)) {
          continue;
        }
        safeHeaders[name] = value;
      }
      lastSnapAPIRequestHeaders = {
        url: requestURL,
        userAgent: headers["user-agent"] || "",
        snapClientUserAgent: headers["x-snap-client-user-agent"] || "",
        secChUa: headers["sec-ch-ua"] || "",
        secChUaPlatform: headers["sec-ch-ua-platform"] || "",
        origin: headers.origin || "",
        referer: headers.referer || "",
        capturedSelfUserID,
        headers: safeHeaders,
      };
    } catch {
      // Best-effort diagnostics only.
    }
  });

  context.on("response", async (response) => {
    try {
      const responseURL = new URL(response.url());
      if (!/(^|\.)accounts\.snapchat\.com$/i.test(responseURL.hostname)) {
        return;
      }
      capturedAccountEndpoints.add(`${response.request().method()} ${responseURL.pathname}`.slice(0, 240));
      while (capturedAccountEndpoints.size > 40) {
        capturedAccountEndpoints.delete(capturedAccountEndpoints.values().next().value);
      }
      if (!/json/i.test(String(response.headers()["content-type"] || ""))) {
        return;
      }
      const identity = identityUUIDFromJSON(await response.json());
      if (identity?.uuid) {
        capturedSelfUserID = identity.uuid;
        capturedIdentitySource = `accounts-response:${responseURL.pathname}:${identity.path}`;
      }
    } catch {
      // Best-effort identity capture only.
    }
  });

  context.on("page", (newPage) => {
    page = newPage;
    page.setDefaultTimeout(DEFAULT_TIMEOUT_MS);
    page.setDefaultNavigationTimeout(NAVIGATION_TIMEOUT_MS);
  });

  page = pickBestPage(context.pages()) || (await context.newPage());
  page.setDefaultTimeout(DEFAULT_TIMEOUT_MS);
  page.setDefaultNavigationTimeout(NAVIGATION_TIMEOUT_MS);
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
  for (let attempt = 0; attempt < CONVERSATION_OPEN_ATTEMPTS; attempt += 1) {
    await currentPage.waitForTimeout(500);
    if (attempt > 0 && attempt % 10 === 0) {
      await dismissInterferingOverlays(currentPage);
    }
    const state = await getConversationState(currentPage, chatName);
    const urlChanged = state.url !== previousURL && isConversationPath(state.url);
    const openedCorrectID = Boolean(chatID && state.url.includes(`/${chatID}`));
    const openedByURL = openedCorrectID || (urlChanged && !chatID);
    if ((state.hasTargetHeader || openedByURL) && state.hasComposer) {
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

async function tryOpenFromSidebar(currentPage, chatName, previousURL = "", chatID = "") {
  if (!chatName) {
    return false;
  }

  const target = await findSidebarChatTarget(currentPage, chatName);
  if (!target) {
    return false;
  }

  await currentPage.locator("[data-codex-chat-target='1']").first().click({ force: true });
  await waitForConversationOpen(currentPage, chatName, previousURL, chatID);
  return true;
}

async function returnToChatList(currentPage) {
  try {
    await currentPage.goto(SNAPCHAT_URL, { waitUntil: "domcontentloaded", timeout: CHAT_OPEN_NAVIGATION_TIMEOUT_MS });
  } catch (error) {
    if (!isSnapchatURL(currentPage.url())) {
      throw error;
    }
  }

  await currentPage.waitForTimeout(500);
  await dismissInterferingOverlays(currentPage);
  await clearSearchBox(currentPage);
}

async function openChat(chatName = "", chatID = "", chatURL = "", { searchFallback = true } = {}) {
  const currentPage = await gotoSnapchat();
  if (!(await isLoggedIn(currentPage))) {
    throw new Error("snapchat session is not logged in");
  }

  await dismissInterferingOverlays(currentPage);
  await clearSearchBox(currentPage);
  let previousURL = currentPage.url();
  let target;
  let lastOpenError;

  try {
    if (await tryOpenFromSidebar(currentPage, chatName, previousURL, chatID)) {
      return currentPage;
    }
  } catch (error) {
    lastOpenError = error;
    console.error(`openChat: sidebar open failed chatName=${JSON.stringify(chatName)} chatId=${chatID} error=${String(error)}`);
  }

  if (chatURL && isSnapchatURL(chatURL)) {
    try {
      await currentPage.goto(chatURL, { waitUntil: "domcontentloaded", timeout: CHAT_OPEN_NAVIGATION_TIMEOUT_MS });
      const activeID = getConversationIDFromURL(chatURL) || chatID;
      await waitForConversationOpen(currentPage, chatName, previousURL, activeID);
      return currentPage;
    } catch (error) {
      lastOpenError = error;
      console.error(`openChat: direct URL open failed chatName=${JSON.stringify(chatName)} chatId=${chatID} chatUrl=${JSON.stringify(chatURL)} error=${String(error)}`);
      await returnToChatList(currentPage);
      previousURL = currentPage.url();
      if (await tryOpenFromSidebar(currentPage, chatName, previousURL, chatID || getConversationIDFromURL(chatURL))) {
        return currentPage;
      }
    }
  }

  if (looksLikeConversationID(chatID)) {
    try {
      await currentPage.goto(conversationURL(chatID), { waitUntil: "domcontentloaded", timeout: CHAT_OPEN_NAVIGATION_TIMEOUT_MS });
      await waitForConversationOpen(currentPage, chatName, previousURL, chatID);
      return currentPage;
    } catch (error) {
      lastOpenError = error;
      console.error(`openChat: conversation ID open failed chatName=${JSON.stringify(chatName)} chatId=${chatID} error=${String(error)}`);
      await returnToChatList(currentPage);
      previousURL = currentPage.url();
    }
  }

  if (!searchFallback) {
    throw lastOpenError || new Error(`could not open Snapchat chat "${chatName || chatID}" without search fallback`);
  }

  target = await findSidebarChatTarget(currentPage, chatName);
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

function chatCacheKey(chat) {
  return chat?.id || normalizeText(chat?.name).toLowerCase();
}

function normalizeChatIdentity(value) {
  return normalizeText(value).toLowerCase();
}

function chatIdentityKey(chat) {
  return normalizeChatIdentity(chat?.name) || chatCacheKey(chat);
}

function mergeChatIdentity(existing = {}, incoming = {}) {
  return {
    ...existing,
    ...incoming,
    id: incoming.id || existing.id || "",
    url: incoming.url || existing.url || (incoming.id ? conversationURL(incoming.id) : ""),
    name: incoming.name || existing.name || "",
    preview: incoming.preview || existing.preview || "",
    lastMessage: incoming.lastMessage || existing.lastMessage || "",
    unread: Boolean(incoming.unread || existing.unread),
  };
}

async function ensureKnownChatsLoaded() {
  if (knownChatsLoaded) {
    return;
  }
  knownChatsLoaded = true;
  try {
    const raw = JSON.parse(await fs.readFile(knownChatsPath, "utf8"));
    const chats = Array.isArray(raw?.chats) ? raw.chats : Array.isArray(raw) ? raw : [];
    for (const chat of chats) {
      const key = chatCacheKey(chat);
      if (key) {
        knownChatsByKey.set(key, chat);
      }
    }
    if (knownChatsByKey.size >= CHAT_DEEP_SCAN_MIN_CACHE) {
      lastDeepChatSync = Date.now();
    }
    console.error(`knownChats: loaded ${knownChatsByKey.size} chats from ${knownChatsPath}`);
  } catch (error) {
    if (error?.code !== "ENOENT") {
      console.error(`knownChats: failed to load cache: ${String(error)}`);
    }
  }
}

async function saveKnownChats() {
  if (!knownChatsDirty) {
    return;
  }
  knownChatsDirty = false;
  try {
    await fs.mkdir(path.dirname(knownChatsPath), { recursive: true });
    const chats = Array.from(knownChatsByKey.values()).slice(0, 200);
    await fs.writeFile(knownChatsPath, JSON.stringify({ chats }, null, 2), "utf8");
  } catch (error) {
    knownChatsDirty = true;
    console.error(`knownChats: failed to save cache: ${String(error)}`);
  }
}

function mergeKnownChats(chats) {
  for (const chat of chats) {
    const key = chatCacheKey(chat);
    if (!key) {
      continue;
    }
    const previous = knownChatsByKey.get(key) || {};
    knownChatsByKey.set(key, {
      ...previous,
      ...chat,
      preview: chat.preview || previous.preview || "",
      lastMessage: chat.lastMessage || previous.lastMessage || "",
      unread: Boolean(chat.unread || previous.unread),
    });
    knownChatsDirty = true;
  }
}

function orderKnownChats(primaryChats) {
  const ordered = [];
  const seen = new Set();
  const byIdentity = new Map();

  const addChat = (chat, preferIncoming = false) => {
    const identity = chatIdentityKey(chat);
    if (!identity || seen.has(identity)) {
      return;
    }
    if (preferIncoming || !byIdentity.has(identity)) {
      byIdentity.set(identity, mergeChatIdentity(byIdentity.get(identity), chat));
    }
    ordered.push(byIdentity.get(identity));
    seen.add(identity);
  };

  for (const chat of primaryChats) {
    addChat(chat, true);
  }
  for (const chat of knownChatsByKey.values()) {
    addChat(chat, false);
  }
  return ordered.slice(0, 100);
}

function findKnownChat(chatID, chatName) {
  if (chatID && knownChatsByKey.has(chatID)) {
    return knownChatsByKey.get(chatID);
  }
  const normalizedName = normalizeText(chatName).toLowerCase();
  for (const chat of knownChatsByKey.values()) {
    if ((chat.id && chat.id === chatID) || normalizeText(chat.name).toLowerCase() === normalizedName) {
      return chat;
    }
  }
  return null;
}

async function readSidebarChatRows(currentPage) {
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
      const preview = [status, time].map((value) => value.replace(/\s+/g, " ").trim()).filter(Boolean).join(" - ");
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

  return rows;
}

async function scrollSidebarChatList(currentPage, direction) {
  const before = await currentPage.evaluate((scrollDirection) => {
    const widthLimit = Math.max(window.innerWidth * 0.45, 420);
    const rows = Array.from(document.querySelectorAll("[role='button'][aria-labelledby*='title-'], [role='link'][aria-labelledby*='title-']"))
      .filter((row) => {
        const rect = row.getBoundingClientRect();
        return rect.width && rect.height && rect.left < widthLimit;
      });
    const firstRow = rows[0];
    const signature = rows
      .map((row) => row.querySelector("span[id^='title-']")?.id || row.textContent || "")
      .join("|");

    const candidates = Array.from(document.querySelectorAll("div, main, section, aside, nav"))
      .map((element) => {
        const rect = element.getBoundingClientRect();
        const overflow = window.getComputedStyle(element).overflowY;
        const rowCount = rows.filter((row) => element.contains(row)).length;
        const scrollable = element.scrollHeight - element.clientHeight;
        return { element, rect, overflow, rowCount, scrollable };
      })
      .filter(({ rect, rowCount, scrollable }) =>
        rect.width &&
        rect.height &&
        rect.left < widthLimit &&
        rowCount > 0 &&
        scrollable > 20
      )
      .sort((a, b) => (b.rowCount - a.rowCount) || (b.scrollable - a.scrollable));
    const styled = candidates.find(({ overflow }) => /auto|scroll|overlay/i.test(overflow));
    const scroller = styled?.element || candidates[0]?.element || document.scrollingElement || document.documentElement;
    const rect = (firstRow || scroller).getBoundingClientRect();
    const scrollTop = scroller.scrollTop;
    const amount = scrollDirection === "top" ? -scroller.scrollHeight : Math.max(700, scroller.clientHeight * 0.9);
    try {
      if (scrollDirection === "top") {
        scroller.scrollTop = 0;
      } else {
        scroller.scrollTop += amount;
      }
      scroller.dispatchEvent(new Event("scroll", { bubbles: true }));
    } catch {}

    return {
      signature,
      scrollTop,
      scrollHeight: scroller.scrollHeight,
      clientHeight: scroller.clientHeight,
      x: Math.max(30, Math.min(widthLimit - 20, rect.left + Math.min(80, rect.width / 2))),
      y: Math.max(80, Math.min(window.innerHeight - 80, rect.top + Math.min(60, rect.height / 2))),
    };
  }, direction);

  await currentPage.mouse.move(before.x, before.y);
  await currentPage.mouse.wheel(0, direction === "top" ? -5000 : Math.max(700, before.clientHeight * 0.9));
  await currentPage.waitForTimeout(500);

  return currentPage.evaluate((previous) => {
    const widthLimit = Math.max(window.innerWidth * 0.45, 420);
    const rows = Array.from(document.querySelectorAll("[role='button'][aria-labelledby*='title-'], [role='link'][aria-labelledby*='title-']"))
      .filter((row) => {
        const rect = row.getBoundingClientRect();
        return rect.width && rect.height && rect.left < widthLimit;
      });
    const signature = rows
      .map((row) => row.querySelector("span[id^='title-']")?.id || row.textContent || "")
      .join("|");

    const candidates = Array.from(document.querySelectorAll("div, main, section, aside, nav"))
      .map((element) => {
        const rect = element.getBoundingClientRect();
        const overflow = window.getComputedStyle(element).overflowY;
        const rowCount = rows.filter((row) => element.contains(row)).length;
        const scrollable = element.scrollHeight - element.clientHeight;
        return { element, rect, overflow, rowCount, scrollable };
      })
      .filter(({ rect, rowCount, scrollable }) =>
        rect.width &&
        rect.height &&
        rect.left < widthLimit &&
        rowCount > 0 &&
        scrollable > 20
      )
      .sort((a, b) => (b.rowCount - a.rowCount) || (b.scrollable - a.scrollable));
    const styled = candidates.find(({ overflow }) => /auto|scroll|overlay/i.test(overflow));
    const scroller = styled?.element || candidates[0]?.element || document.scrollingElement || document.documentElement;
    const scrollMoved = Math.abs(scroller.scrollTop - previous.scrollTop) > 4;
    const contentChanged = signature !== previous.signature;

    return {
      moved: scrollMoved || contentChanged,
      atBottom: scroller.scrollTop + scroller.clientHeight >= scroller.scrollHeight - 8,
      scrollTop: scroller.scrollTop,
      scrollHeight: scroller.scrollHeight,
      clientHeight: scroller.clientHeight,
    };
  }, before);
}

async function extractChats(currentPage, { deep = false } = {}) {
  if (!deep) {
    return parseChatRows(await readSidebarChatRows(currentPage));
  }

  const byKey = new Map();
  const mergeRows = (rows) => {
    for (const chat of parseChatRows(rows)) {
      const key = chatCacheKey(chat);
      if (key && !byKey.has(key)) {
        byKey.set(key, chat);
      }
    }
  };

  await scrollSidebarChatList(currentPage, "top");
  await currentPage.waitForTimeout(250);

  let idlePasses = 0;
  for (let pass = 0; pass < 36; pass += 1) {
    const before = byKey.size;
    mergeRows(await readSidebarChatRows(currentPage));
    idlePasses = byKey.size === before ? idlePasses + 1 : 0;

    const scroll = await scrollSidebarChatList(currentPage, "down");
    await currentPage.waitForTimeout(250);
    if (scroll.atBottom || (!scroll.moved && idlePasses >= 2)) {
      break;
    }
  }

  return Array.from(byKey.values()).slice(0, 100);
}

async function extractMessages(currentPage, activeChatName, activeChatID = "") {
  const state = await getConversationState(currentPage, activeChatName);
  if (!state.hasComposer) {
    return [];
  }

  const messages = await currentPage.evaluate(({ activeName, activeID, unsupportedLabel, unsupportedEvent, statusLabel, dateDividerSource, snapPreviewSource }) => {
    const normalize = (value) => String(value || "").replace(/\s+/g, " ").trim();
    const escapeRegExp = (value) => String(value || "").replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
    const dateDivider = new RegExp(dateDividerSource, "i");
    const snapPreview = new RegExp(snapPreviewSource, "i");
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

    const timeOnly = /^(?:\d{1,2}:\d{2}|\d{1,2})(?:\s?[ap]\.?m\.?)$/i;
    const combineTimestamp = (explicit, current) => {
      const timeHint = normalize(explicit);
      const dateHint = normalize(current);
      if (timeHint && dateHint && timeOnly.test(timeHint) && !timeOnly.test(dateHint)) {
        return `${dateHint} ${timeHint}`;
      }
      return timeHint || dateHint;
    };

    for (const item of conversationRoot.querySelectorAll(":scope > li")) {
      const rect = item.getBoundingClientRect();
      if (!rect.width || !rect.height) {
        continue;
      }

      const timeElement = item.querySelector("time");
      const explicitTimestamp = normalize(
        timeElement?.getAttribute("datetime") ||
        timeElement?.getAttribute("title") ||
        timeElement?.textContent ||
        ""
      );
      if (timeElement && !item.querySelector(".KB4Aq, [dir='auto'], .ogn1z, .ijyxU")) {
        currentTimestamp = explicitTimestamp;
        continue;
      }

      const headerText = normalize(item.querySelector("header")?.textContent || "");
      const outgoing = /^me$/i.test(headerText);
      const author = outgoing ? "You" : (headerText || activeName || "");
      const fullText = normalize(item.textContent || "");
      const activePrefix = activeName ? new RegExp(`^\\s*${escapeRegExp(activeName)}\\s*`, "i") : null;
      const statusText = normalize(
        fullText
          .replace(/^me\b/i, "")
          .replace(activePrefix || /^$/, "")
          .replace(/Close Chat/gi, "")
      );
      const textNodes = Array.from(item.querySelectorAll("[dir='auto'], .ogn1z, .ijyxU, .mZgqh"))
        .map((node) => normalize(node.textContent))
        .filter(Boolean);

      let text = textNodes.length > 0 ? textNodes[textNodes.length - 1] : "";
      const hasMedia = item.querySelectorAll("img, video, canvas, input[type='file']").length > 0;
      const hasBubble = Boolean(item.querySelector(".KB4Aq"));
      const isStatusOnly = Boolean(item.querySelector(".mZgqh")) && !item.querySelector("[dir='auto'], .ogn1z");
      const isSnapStatus = snapPreview.test(statusText) && !/send a chat|search|reply|call|typing/i.test(statusText);

      if (!text && !hasMedia && !hasBubble && !isSnapStatus) {
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
      } else if (isStatusOnly && isSnapStatus) {
        text = `${statusLabel}: ${statusText}`;
      } else if (isStatusOnly) {
        continue;
      } else if (!text && (hasMedia || isSnapStatus)) {
        text = isSnapStatus ? `${unsupportedLabel}: ${statusText}` : unsupportedLabel;
      } else if (!text) {
        continue;
      }

      const lower = text.toLowerCase();
      if (!text || lower === activeLower || dateDivider.test(text)) {
        continue;
      }

      const isSyntheticStatus = text.startsWith(statusLabel) || text.startsWith(unsupportedLabel);
      const key = isSyntheticStatus
        ? `${outgoing ? "out" : "in"}:${explicitTimestamp || currentTimestamp}:${text}`
        : `${outgoing ? "out" : "in"}:${Math.round(rect.top / 6)}:${text}`;
      if (seen.has(key)) {
        continue;
      }
      seen.add(key);
      items.push({ author, text, top: rect.top, outgoing, timestamp: combineTimestamp(explicitTimestamp, currentTimestamp) });
    }

    return items.sort((a, b) => a.top - b.top).slice(-80);
  }, {
    activeName: activeChatName,
    activeID: activeChatID,
    unsupportedLabel: unsupportedMediaLabel,
    unsupportedEvent: unsupportedEventLabel,
    statusLabel: snapchatStatusLabel,
    dateDividerSource: dateDividerPattern.source,
    snapPreviewSource: snapPreviewPattern.source,
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
  await ensureKnownChatsLoaded();
  const knownChat = findKnownChat(chatID, chatName);
  if (chatURL && isSnapchatURL(chatURL)) {
    const urlID = getConversationIDFromURL(chatURL);
    return {
      id: urlID || chatID || "",
      name: chatName || knownChat?.name || "",
      url: chatURL,
      preview: knownChat?.preview || "",
      lastMessage: knownChat?.lastMessage || "",
      unread: Boolean(knownChat?.unread),
    };
  }

  if (looksLikeConversationID(chatID)) {
    return {
      id: chatID,
      name: chatName || knownChat?.name || "",
      url: knownChat?.url || conversationURL(chatID),
      preview: knownChat?.preview || "",
      lastMessage: knownChat?.lastMessage || "",
      unread: Boolean(knownChat?.unread),
    };
  }

  const chats = await extractChats(currentPage);
  mergeKnownChats(chats);
  await saveKnownChats();
  if (chatName) {
    const match = chats.find((chat) => chat.name === chatName) || knownChat;
    return {
      id: match?.id || "",
      name: chatName,
      url: match?.url || (match?.id ? conversationURL(match.id) : ""),
      preview: match?.preview || "",
      lastMessage: match?.lastMessage || "",
      unread: Boolean(match?.unread),
    };
  }
  if (!chatID) {
    throw new Error("chatId or chatName is required");
  }

  const match = chats.find((chat) => chat.id === chatID) || knownChat;
  if (!match) {
    throw new Error(`could not resolve chatId "${chatID}"`);
  }
  return {
    id: match.id,
    name: match.name,
    url: match.url || conversationURL(match.id),
    preview: match.preview || "",
    lastMessage: match.lastMessage || "",
    unread: Boolean(match.unread),
  };
}

function extractTimestampHint(value) {
  const text = normalizeText(value);
  const match = text.match(/\b(today|yesterday|sun(?:day)?|mon(?:day)?|tue(?:s|sday|day)?|wed(?:nesday)?|thu(?:r|rs|rsday|rday)?|fri(?:day)?|sat(?:urday)?|\d+\s*[mhdwy]|(?:jan|feb|mar|apr|may|jun|jul|aug|sep|sept|oct|nov|dec)[a-z]*\s+\d{1,2})\b/i);
  return match ? match[1] : "";
}

function makeSidebarStatusMessage(chat, activeChat = {}) {
  const preview = normalizeText(chat?.lastMessage || chat?.preview || "");
  if (!preview || !snapPreviewPattern.test(preview)) {
    return null;
  }

  const isMedia = /\b(new snap|snap|video|photo|picture|media|voice note|audio|sticker|tap to view|received|opened|delivered|sent)\b/i.test(preview);
  const label = isMedia ? unsupportedMediaLabel : snapchatStatusLabel;
  const outgoing = /^(sent|delivered|opened)/i.test(preview) && !/sent you/i.test(preview);
  return {
    id: stableID(["sidebar-status", chat?.id || activeChat?.id || chat?.name || "", preview]),
    author: outgoing ? "You" : (activeChat?.name || chat?.name || "Snapchat"),
    text: `${label}: ${preview}`,
    timestamp: extractTimestampHint(preview),
    outgoing,
  };
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

async function findSendButton(currentPage) {
  const found = await currentPage.evaluate(() => {
    document.querySelectorAll("[data-codex-send-button]").forEach((element) => {
      element.removeAttribute("data-codex-send-button");
    });

    const rightPaneLeft = Math.max(window.innerWidth * 0.28, 300);
    const candidates = Array.from(document.querySelectorAll("button, [role='button'], [aria-label*='send' i]"))
      .map((element) => {
        const rect = element.getBoundingClientRect();
        const label = [
          element.getAttribute("aria-label") || "",
          element.getAttribute("title") || "",
          element.textContent || "",
        ].join(" ");
        const disabled = element.disabled || element.getAttribute("aria-disabled") === "true";
        return { element, rect, label, disabled };
      })
      .filter(({ rect, label, disabled }) =>
        !disabled &&
        rect.width &&
        rect.height &&
        rect.left >= rightPaneLeft &&
        rect.bottom >= window.innerHeight * 0.55 &&
        /send/i.test(label)
      )
      .sort((a, b) => b.rect.bottom - a.rect.bottom || a.rect.left - b.rect.left);

    const match = candidates[0];
    if (!match) {
      return false;
    }
    match.element.setAttribute("data-codex-send-button", "1");
    return true;
  });

  return found ? currentPage.locator("[data-codex-send-button='1']").first() : null;
}

async function findUploadInput(currentPage) {
  const input = currentPage.locator("input[type='file']").last();
  try {
    if (await input.count()) {
      return input;
    }
  } catch {
  }

  const uploadButton = await firstVisibleLocator([
    currentPage.getByRole("button", { name: /upload|attach|camera roll|photo|picture|media/i }),
    currentPage.locator("[aria-label*='upload' i], [aria-label*='attach' i], [aria-label*='camera' i], [title*='upload' i], [title*='attach' i]"),
  ]);
  if (uploadButton) {
    await uploadButton.click({ force: true });
    await currentPage.waitForTimeout(500);
  }

  const postClickInput = currentPage.locator("input[type='file']").last();
  if (await postClickInput.count()) {
    return postClickInput;
  }
  throw new Error("could not find Snapchat media upload input");
}

function sanitizeUploadFileName(value) {
  const base = normalizeText(value) || "snap-media.bin";
  return base.replace(/[^a-zA-Z0-9._-]+/g, "-").replace(/^-+|-+$/g, "") || "snap-media.bin";
}

async function attachMediaFile(currentPage, filePath) {
  const input = await findUploadInput(currentPage);
  await input.setInputFiles(filePath);
  await currentPage.waitForTimeout(2000);
}

export async function createSession() {
  return withLock(async () => {
    await ensureKnownChatsLoaded();
    const currentPage = await gotoSnapchat();
    const state = await detectState(currentPage);
    return {
      state,
      authenticated: state === "ready",
      url: currentPage.url(),
      title: await currentPage.title(),
    };
  }, { priority: 0, timeoutMs: DEFAULT_TIMEOUT_MS + 5000, label: "session/start" });
}

export async function getStatus() {
  return withLock(async () => {
    await ensureKnownChatsLoaded();
    const currentPage = await gotoSnapchat();
    const state = await detectState(currentPage);

    return {
      state,
      authenticated: state === "ready",
      activeChatName: await currentPage.title(),
      url: currentPage.url(),
      visibleChatCount: state === "ready" ? knownChatsByKey.size : 0,
    };
  }, { priority: 0, timeoutMs: DEFAULT_TIMEOUT_MS + 5000, label: "session/status" });
}

async function readSnapchatAPIIdentity(currentPage) {
  if (!currentPage || currentPage.isClosed()) {
    return { selfUserID: "", username: "", displayName: "", source: "", debug: {} };
  }

  try {
    const frames = typeof currentPage.frames === "function" ? currentPage.frames() : [currentPage.mainFrame()];
    const frameDebug = [];
    for (const frame of frames) {
      try {
        const identity = await frame.evaluate(async () => {
      const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

      function bytesToUUID(bytes) {
        if (!bytes || bytes.length !== 16) {
          return "";
        }
        const hex = Array.from(bytes, (byte) => Number(byte).toString(16).padStart(2, "0")).join("");
        return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`.toLowerCase();
      }

      function uuidFromValue(value) {
        if (!value) {
          return "";
        }
        if (typeof value === "string") {
          return uuidPattern.test(value) ? value.toLowerCase() : "";
        }
        if (typeof value !== "object") {
          return "";
        }
        if (typeof value.str === "string" && uuidPattern.test(value.str)) {
          return value.str.toLowerCase();
        }
        if (value.id instanceof Uint8Array) {
          return bytesToUUID(value.id);
        }
        if (Array.isArray(value.id)) {
          return bytesToUUID(value.id);
        }
        if (ArrayBuffer.isView(value.id)) {
          return bytesToUUID(new Uint8Array(value.id.buffer, value.id.byteOffset, value.id.byteLength));
        }
        if (value.encodedId instanceof Uint8Array) {
          return bytesToUUID(value.encodedId);
        }
        if (Array.isArray(value.encodedId)) {
          return bytesToUUID(value.encodedId);
        }
        return "";
      }

      function identityFromState(state, source) {
        const auth = state?.auth;
        if (!auth || typeof auth !== "object") {
          return null;
        }
        const me = auth.currentUserOverride || auth.me || {};
        const selfUserID =
          uuidFromValue(auth.userId) ||
          uuidFromValue(me.userId) ||
          uuidFromValue(me.id) ||
          uuidFromValue(me);
        if (!selfUserID) {
          return null;
        }
        return {
          selfUserID,
          username: me.mutableUsername || me.username || auth.username || "",
          displayName: me.displayName || me.display_name || auth.displayName || "",
          source,
        };
      }

      function identityFromObject(root, source) {
        const seen = new Set();
        let visited = 0;
        function scan(value, path = "", depth = 0) {
          if (!value || typeof value !== "object" || seen.has(value) || depth > 7 || visited++ > 10000) {
            return null;
          }
          seen.add(value);
          for (const [key, child] of Object.entries(value)) {
            const childPath = path ? `${path}.${key}` : key;
            if (/^(?:user_?id|userid|self_?id|account_?id)$/i.test(key)) {
              const selfUserID = uuidFromValue(child);
              if (selfUserID) {
                return {
                  selfUserID,
                  username: String(value.username || value.mutableUsername || ""),
                  displayName: String(value.displayName || value.display_name || ""),
                  source: `${source}:${childPath}`,
                };
              }
            }
            const nested = scan(child, childPath, depth + 1);
            if (nested) {
              return nested;
            }
          }
          return null;
        }
        return scan(root);
      }

      function findWebpackRequire() {
        for (const key of Object.keys(globalThis)) {
          if (!/^webpackChunk/.test(key)) {
            continue;
          }
          const chunk = globalThis[key];
          if (!Array.isArray(chunk) || typeof chunk.push !== "function") {
            continue;
          }
          let webpackRequire;
          try {
            chunk.push([[`codex-api-auth-${Date.now()}`], {}, (req) => {
              webpackRequire = req;
            }]);
          } catch {
            continue;
          }
          if (webpackRequire) {
            return webpackRequire;
          }
        }
        return undefined;
      }

      function safeRequire(webpackRequire, moduleId) {
        if (!webpackRequire) {
          return undefined;
        }
        try {
          return webpackRequire(moduleId);
        } catch {
          return undefined;
        }
      }

      function scanExports(exports, source, debug, seen = new Set(), depth = 0) {
        if (!exports || (typeof exports !== "object" && typeof exports !== "function") || seen.has(exports) || depth > 3) {
          return null;
        }
        seen.add(exports);
        if (typeof exports.getState === "function") {
          const candidate = source;
          if (debug.storeCandidates.length < 20) {
            debug.storeCandidates.push(candidate);
          }
          try {
            const identity = identityFromState(exports.getState(), source);
            if (identity) {
              return identity;
            }
          } catch {
          }
        }
        if (typeof exports !== "object") {
          return null;
        }
        for (const [key, value] of Object.entries(exports)) {
          if (!value || (typeof value !== "object" && typeof value !== "function")) {
            continue;
          }
          const identity = scanExports(value, `${source}.${key}`, debug, seen, depth + 1);
          if (identity) {
            return identity;
          }
        }
        return null;
      }

      const debug = {
        webpackChunkKeys: Object.keys(globalThis).filter((key) => /^webpackChunk/.test(key)).slice(0, 10),
        requireKeys: [],
        moduleCount: 0,
        moduleDefinitionCount: 0,
        definitionCandidates: [],
        storeCandidates: [],
        localStorageKeys: Object.keys(localStorage).slice(0, 50),
        sessionStorageKeys: Object.keys(sessionStorage).slice(0, 50),
        indexedDB: [],
      };
      for (const storage of [localStorage, sessionStorage]) {
        for (let index = 0; index < storage.length; index++) {
          const key = storage.key(index);
          const raw = storage.getItem(key);
          if (!raw) {
            continue;
          }
          try {
            const identity = identityFromObject(JSON.parse(raw), `web-storage:${key}`);
            if (identity) {
              identity.debug = debug;
              return identity;
            }
          } catch {
          }
        }
      }
      try {
        if (typeof indexedDB.databases === "function") {
          const databases = await indexedDB.databases();
          for (const databaseInfo of databases.slice(0, 20)) {
            if (!databaseInfo.name) {
              continue;
            }
            const database = await new Promise((resolve, reject) => {
              const request = indexedDB.open(databaseInfo.name);
              request.onsuccess = () => resolve(request.result);
              request.onerror = () => reject(request.error || new Error("open failed"));
            });
            debug.indexedDB.push({
              name: databaseInfo.name,
              stores: Array.from(database.objectStoreNames).slice(0, 50),
            });
            for (const storeName of Array.from(database.objectStoreNames)) {
              const transaction = database.transaction(storeName, "readonly");
              const store = transaction.objectStore(storeName);
              const keys = await new Promise((resolve, reject) => {
                const request = store.getAllKeys(undefined, 100);
                request.onsuccess = () => resolve(request.result || []);
                request.onerror = () => reject(request.error || new Error("get keys failed"));
              });
              const databaseDebug = debug.indexedDB[debug.indexedDB.length - 1];
              databaseDebug.keyNames = keys
                .filter((key) => typeof key === "string")
                .map((key) => key.slice(0, 120));
              for (const key of keys) {
                const value = await new Promise((resolve, reject) => {
                  const request = store.get(key);
                  request.onsuccess = () => resolve(request.result);
                  request.onerror = () => reject(request.error || new Error("get failed"));
                });
                const identity = identityFromObject(value, `indexeddb:${databaseInfo.name}/${storeName}`);
                if (identity) {
                  database.close();
                  identity.debug = debug;
                  return identity;
                }
              }
            }
            database.close();
          }
        }
      } catch {
      }
      const webpackRequire = findWebpackRequire();
      if (webpackRequire) {
        debug.requireKeys = Object.keys(webpackRequire).slice(0, 30);
      }
      if (webpackRequire?.c) {
        debug.moduleCount = Object.keys(webpackRequire.c).length;
        for (const [moduleID, module] of Object.entries(webpackRequire.c)) {
          const identity = scanExports(module?.exports, `webpack-cache:${moduleID}`, debug);
          if (identity) {
            identity.debug = debug;
            return identity;
          }
        }
      }
      if (webpackRequire?.m) {
        const definitions = Object.entries(webpackRequire.m);
        debug.moduleDefinitionCount = definitions.length;
        for (const [moduleID, factory] of definitions) {
          let source = "";
          try {
            source = Function.prototype.toString.call(factory);
          } catch {
          }
          if (!/auth\.userId|auth:\{|currentUserOverride|getState\(\)\.auth/.test(source)) {
            continue;
          }
          if (debug.definitionCandidates.length < 20) {
            debug.definitionCandidates.push(moduleID);
          }
          try {
            const exports = webpackRequire(moduleID);
            const identity = scanExports(exports, `webpack-module:${moduleID}`, debug);
            if (identity) {
              identity.debug = debug;
              return identity;
            }
          } catch {
          }
        }
      }

          return { selfUserID: "", username: "", displayName: "", source: "", debug };
        });
        identity.debug = { ...identity.debug, frameURL: frame.url() };
        frameDebug.push(identity.debug);
        if (identity.selfUserID) {
          return identity;
        }
      } catch (error) {
        frameDebug.push({
          frameURL: frame.url(),
          error: String(error).slice(0, 160),
        });
      }
    }
    return { selfUserID: "", username: "", displayName: "", source: "", debug: { frames: frameDebug } };
  } catch {
    return { selfUserID: "", username: "", displayName: "", source: "", debug: { error: "page-evaluate-failed" } };
  }
}

function readIdentityFromSSOToken(token) {
  const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
  try {
    const parts = String(token || "").split(".");
    if (parts.length < 2) {
      return "";
    }
    const payload = JSON.parse(Buffer.from(parts[1], "base64url").toString("utf8"));
    for (const key of ["sub", "user_id", "userId", "uid", "snapchat_user_id"]) {
      const value = String(payload?.[key] || "").trim();
      if (uuidPattern.test(value)) {
        return value.toLowerCase();
      }
    }
  } catch {
  }
  return "";
}

async function readSnapchatAPIUserAgents(currentPage) {
  const fallbackBrowserUserAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36";
  let browserUserAgent = fallbackBrowserUserAgent;
  try {
    if (currentPage && !currentPage.isClosed()) {
      browserUserAgent = await currentPage.evaluate(() => navigator.userAgent || "");
    }
  } catch {
    browserUserAgent = fallbackBrowserUserAgent;
  }
  if (!browserUserAgent) {
    browserUserAgent = fallbackBrowserUserAgent;
  }

  let webVersion = "";
  try {
    const response = await fetch(SNAPCHAT_VERSION_URL, { cache: "no-store" });
    if (response.ok) {
      const version = await response.json();
      webVersion = String(version.version || "");
    }
  } catch {
    webVersion = "";
  }
  if (!webVersion) {
    webVersion = "13.79.0";
  }

  const chromeVersionMatch = browserUserAgent.match(/(?:Chrome|Chromium|HeadlessChrome)\/([0-9.]+)/i);
  const chromeVersion = chromeVersionMatch?.[1] || "120.0.0.0";
  const captured = lastSnapAPIRequestHeaders || {};
  const capturedHeaders = captured.headers || {};
  return {
    browserUserAgent: captured.userAgent || browserUserAgent,
    snapClientUserAgent: captured.snapClientUserAgent || `SnapchatWeb/${webVersion} PROD (linux 0.0.0; chrome ${chromeVersion})`,
    secChUa: captured.secChUa || "",
    secChUaPlatform: captured.secChUaPlatform || "",
    grpcWebUserAgent: capturedHeaders["x-user-agent"] || "grpc-web-javascript/0.1",
    mcsCofIdsBin: capturedHeaders["mcs-cof-ids-bin"] || "",
    webVersion,
  };
}

export async function getAPIAuth() {
  return withLock(async () => {
    if (
      lastAPIAuthSnapshot?.authenticated
      && Date.now() - lastAPIAuthSnapshotAt < API_AUTH_CACHE_MS
    ) {
      return {
        ...lastAPIAuthSnapshot,
        cached: true,
        cacheAgeMs: Date.now() - lastAPIAuthSnapshotAt,
      };
    }
    await ensureKnownChatsLoaded();
    let { context: currentContext, page: currentPage } = await ensureSession();
    if (!lastSnapAPIRequestHeaders?.headers?.["mcs-cof-ids-bin"]) {
      try {
        currentPage = await withTimeout(gotoSnapchat({ reload: true }), API_AUTH_WARMUP_TIMEOUT_MS, "Snapchat API auth warmup");
        currentContext = context || currentContext;
        await withTimeout(currentPage.waitForTimeout(3000), API_AUTH_WARMUP_TIMEOUT_MS, "Snapchat API auth settle");
      } catch (error) {
        console.error(`getAPIAuth: warmup skipped: ${String(error)}`);
        // Auth can still succeed from cookies; this only warms API header capture.
      }
    }
    if (!capturedSelfUserID && !identityWarmupAttempted) {
      identityWarmupAttempted = true;
      const safeChat = Array.from(knownChatsByKey.values())
        .find((chat) => /^Delivered\b/i.test(String(chat.preview || chat.lastMessage || "")) && chat.url);
      if (safeChat) {
        try {
          await withTimeout(currentPage.goto(safeChat.url, { waitUntil: "domcontentloaded" }), API_AUTH_WARMUP_TIMEOUT_MS, "Snapchat identity safe-chat warmup");
          await currentPage.waitForTimeout(4000);
          await withTimeout(currentPage.goto(SNAPCHAT_URL, { waitUntil: "domcontentloaded" }), API_AUTH_WARMUP_TIMEOUT_MS, "Snapchat identity return warmup");
          await currentPage.waitForTimeout(2000);
        } catch (error) {
          console.error(`getAPIAuth: identity warmup skipped: ${String(error)}`);
          // A delivered outgoing chat is used only to provoke the current
          // messaging RPC. Auth can continue through the other probes.
        }
      }
    }
    if (!capturedSelfUserID && !accountIdentityWarmupAttempted) {
      accountIdentityWarmupAttempted = true;
      let accountPage;
      try {
        accountPage = await currentContext.newPage();
        await withTimeout(accountPage.goto("https://accounts.snapchat.com/v2/welcome", { waitUntil: "domcontentloaded" }), API_AUTH_WARMUP_TIMEOUT_MS, "Snapchat account identity warmup");
        await accountPage.waitForTimeout(5000);
      } catch (error) {
        console.error(`getAPIAuth: account identity warmup skipped: ${String(error)}`);
      } finally {
        try {
          await accountPage?.close();
        } catch {
        }
      }
    }
    const cookies = await currentContext.cookies([
      "https://www.snapchat.com",
      "https://web.snapchat.com",
      "https://accounts.snapchat.com",
    ]);
    const wanted = new Set([
      "__Host-sc-a-nonce",
      "__Host-sc-a-session",
      "__Host-sc-a-auth-session",
      "__Host-X-Snap-Client-Cookie",
      "sc-a-nonce",
    ]);
    const selected = cookies
      .filter((cookie) => wanted.has(cookie.name))
      .sort((a, b) => a.name.localeCompare(b.name));
    const byName = new Map(selected.map((cookie) => [cookie.name, cookie.value]));
    const cookieString = selected
      .map((cookie) => `${cookie.name}=${cookie.value}`)
      .join("; ");
    const hasNonce = Boolean(byName.get("__Host-sc-a-nonce") || byName.get("sc-a-nonce"));
    const hasSession = Boolean(byName.get("__Host-sc-a-session") || byName.get("__Host-sc-a-auth-session"));
    const hasClient = Boolean(byName.get("__Host-X-Snap-Client-Cookie"));
    const identity = await readSnapchatAPIIdentity(currentPage);
    const userAgents = await readSnapchatAPIUserAgents(currentPage);
    let ssoToken = capturedSSOToken;
    try {
      const response = await currentContext.request.post(
        "https://accounts.snapchat.com/accounts/sso?client_id=web-calling-corp--prod",
        {
          maxRedirects: 0,
          headers: {
            "user-agent": userAgents.browserUserAgent,
            "x-snap-client-user-agent": userAgents.snapClientUserAgent,
          },
        },
      );
      if (response.status() === 200) {
        const candidate = (await response.text()).trim();
        if (candidate && candidate.length <= 4096 && !candidate.startsWith("<")) {
          ssoToken = candidate;
        }
      }
    } catch {
      // The Go client can still attempt the legacy cookie exchange when this
      // browser-context exchange is unavailable.
    }

    const snapshot = {
      state: hasNonce && hasSession && hasClient ? "ready" : "missing_cookies",
      authenticated: hasNonce && hasSession && hasClient,
      cookieString,
      ssoToken,
      selfUserID: identity.selfUserID || capturedSelfUserID || readIdentityFromSSOToken(ssoToken),
      username: identity.username,
      displayName: identity.displayName,
      identitySource: identity.source || capturedIdentitySource,
      identityDebug: identity.debug,
      browserUserAgent: userAgents.browserUserAgent,
      snapClientUserAgent: userAgents.snapClientUserAgent,
      secChUa: userAgents.secChUa,
      secChUaPlatform: userAgents.secChUaPlatform,
      grpcWebUserAgent: userAgents.grpcWebUserAgent,
      mcsCofIdsBin: userAgents.mcsCofIdsBin,
      webVersion: userAgents.webVersion,
      apiRequestHeaders: lastSnapAPIRequestHeaders,
      authorizationEndpoint: capturedAuthorizationEndpoint,
      authorizationEndpoints: Array.from(capturedAuthorizationEndpoints),
      authorizationRequestMeta: capturedAuthorizationRequestMeta,
      accountEndpoints: Array.from(capturedAccountEndpoints),
      cookies: selected.map((cookie) => ({
        name: cookie.name,
        domain: cookie.domain,
        expires: cookie.expires,
      })),
      url: page && !page.isClosed() ? page.url() : "",
    };
    if (snapshot.authenticated) {
      lastAPIAuthSnapshot = snapshot;
      lastAPIAuthSnapshotAt = Date.now();
    }
    return snapshot;
  }, { priority: 0, timeoutMs: DEFAULT_TIMEOUT_MS + API_AUTH_WARMUP_TIMEOUT_MS + 10000, label: "session/api-auth" });
}

export async function decryptEELMessage(input = {}) {
  const messageId = String(input.messageId || "");
  const conversationId = String(input.conversationId || "");
  const contentBase64 = String(input.contentBase64 || "");
  if (!contentBase64) {
    return { ok: false, error: "missing_content", retryable: false };
  }
  return withLock(async () => {
    const { page: currentPage } = await ensureSession();
    if (!currentPage || currentPage.isClosed()) {
      return { ok: false, error: "no_active_page", retryable: true };
    }
    const result = await currentPage.evaluate(async (payload) => {
      function fromBase64(value) {
        if (!value) {
          return new Uint8Array();
        }
        const binary = atob(value);
        const bytes = new Uint8Array(binary.length);
        for (let index = 0; index < binary.length; index += 1) {
          bytes[index] = binary.charCodeAt(index);
        }
        return bytes;
      }

      function toBase64(value) {
        if (typeof value === "string") {
          value = new TextEncoder().encode(value);
        }
        const bytes = ArrayBuffer.isView(value)
          ? new Uint8Array(value.buffer, value.byteOffset, value.byteLength)
          : value instanceof ArrayBuffer
            ? new Uint8Array(value)
            : Array.isArray(value)
              ? Uint8Array.from(value)
              : undefined;
        if (!bytes) {
          return "";
        }
        let binary = "";
        for (const byte of bytes) {
          binary += String.fromCharCode(byte);
        }
        return btoa(binary);
      }

      async function decryptAESGCM(keyBytes, nonceBytes, dataBytes) {
        if (![16, 24, 32].includes(keyBytes.byteLength) || nonceBytes.byteLength === 0) {
          return undefined;
        }
        const key = await crypto.subtle.importKey("raw", keyBytes, { name: "AES-GCM" }, false, ["decrypt"]);
        return new Uint8Array(await crypto.subtle.decrypt({ name: "AES-GCM", iv: nonceBytes }, key, dataBytes));
      }

      function findWebpackRequire() {
        for (const key of Object.keys(globalThis)) {
          if (!/^webpackChunk/.test(key)) {
            continue;
          }
          const chunk = globalThis[key];
          if (!Array.isArray(chunk) || typeof chunk.push !== "function") {
            continue;
          }
          let webpackRequire;
          try {
            chunk.push([[`codex-eel-decrypt-${Date.now()}`], {}, (req) => {
              webpackRequire = req;
            }]);
          } catch {
            continue;
          }
          if (webpackRequire?.m) {
            return webpackRequire;
          }
        }
        return undefined;
      }

      function textFromWebContent(value, depth = 0) {
        if (!value || depth > 8) {
          return "";
        }
        if (typeof value === "string") {
          return value.trim();
        }
        if (Array.isArray(value)) {
          return value.map((item) => textFromWebContent(item, depth + 1)).filter(Boolean).join("\n\n").trim();
        }
        if (typeof value !== "object") {
          return "";
        }
        const directText = value.text;
        if (typeof directText === "string" && directText.trim()) {
          return directText.trim();
        }
        if (directText && typeof directText === "object") {
          const text = textFromWebContent(directText, depth + 1);
          if (text) {
            return text;
          }
        }
        for (const key of ["messageContent", "content", "chatMessage", "textContent", "slideupText", "statusMessage", "component"]) {
          const text = textFromWebContent(value[key], depth + 1);
          if (text) {
            return text;
          }
        }
        return "";
      }

      function safeRequire(webpackRequire, moduleId) {
        if (!webpackRequire || !moduleId) {
          return undefined;
        }
        try {
          return webpackRequire(moduleId);
        } catch {
          return undefined;
        }
      }

      function resolveAppState(webpackRequire) {
        for (const moduleId of [96821, 97003]) {
          const appState = safeRequire(webpackRequire, moduleId)?.M?.getState?.();
          if (appState) {
            return appState;
          }
        }
        return undefined;
      }

      function uuidBytes(uuid) {
        const hex = String(uuid || "").replace(/-/g, "");
        if (!/^[0-9a-f]{32}$/i.test(hex)) {
          return undefined;
        }
        const bytes = new Uint8Array(16);
        for (let i = 0; i < 16; i++) {
          bytes[i] = Number.parseInt(hex.slice(i * 2, i * 2 + 2), 16);
        }
        return bytes;
      }

      async function resolveMessagingClient(webpackRequire) {
        let appState = resolveAppState(webpackRequire);
        if (appState?.messaging?.client) {
          return { appState, messagingClient: appState.messaging.client };
        }
        const initWasm = appState?.wasm?.initialize;
        if (!appState?.wasm?.workerProxy && typeof initWasm === "function") {
          try {
            await Promise.race([
              initWasm(),
              new Promise((_, reject) => setTimeout(() => reject(new Error("wasm_init_timeout")), 5000)),
            ]);
          } catch {
            // Re-read state below; the page sometimes records fatal WASM errors without throwing useful details.
          }
          for (let i = 0; i < 100; i++) {
            appState = resolveAppState(webpackRequire);
            if (appState?.wasm?.workerProxy) {
              break;
            }
            await new Promise((resolve) => setTimeout(resolve, 100));
          }
        }
        const initClient = appState?.messaging?.initClient || appState?.messaging?.initializeClient;
        if (typeof initClient === "function") {
          try {
            await Promise.race([
              initClient(),
              new Promise((_, reject) => setTimeout(() => reject(new Error("messaging_init_timeout")), 5000)),
            ]);
          } catch {
            // Re-read state below; initClient swallows some internal errors while still setting partial state.
          }
          for (let i = 0; i < 100; i++) {
            appState = resolveAppState(webpackRequire);
            if (appState?.messaging?.client) {
              break;
            }
            await new Promise((resolve) => setTimeout(resolve, 100));
          }
        }
        return { appState, messagingClient: appState?.messaging?.client };
      }

      async function resolveE2EEWasmModule(webpackRequire, appState) {
        const wasmExports = safeRequire(webpackRequire, 54897);
        try {
          const existing = wasmExports?.gZ?.(appState);
          if (existing?.e2ee_E2EEKeyManager) {
            return { wasmModule: existing, method: "state_ref" };
          }
        } catch {
        }
        try {
          const wasmFactory = safeRequire(webpackRequire, 51867);
          const params = wasmExports?.U7?.(appState);
          const created = await wasmFactory?.W?.(params);
          if (created?.e2ee_E2EEKeyManager) {
            return { wasmModule: created, method: "factory" };
          }
        } catch (error) {
          return {
            error: String(error?.message || error || "factory_error").replace(/\s+/g, " ").slice(0, 160),
          };
        }
        return {};
      }

      function resultBytes(value) {
        if (!value) {
          return undefined;
        }
        if (ArrayBuffer.isView(value) || value instanceof ArrayBuffer || Array.isArray(value)) {
          return value;
        }
        for (const key of ["content", "contents", "plaintext", "plainText", "decrypted", "cek", "key", "data"]) {
          const nested = value[key];
          if (nested && (ArrayBuffer.isView(nested) || nested instanceof ArrayBuffer || Array.isArray(nested))) {
            return nested;
          }
        }
        return undefined;
      }

      function serverHexFromMessageId(messageId) {
        const numeric = Number(messageId || 0);
        if (!Number.isFinite(numeric) || numeric <= 0) {
          return "";
        }
        return numeric.toString(16).toUpperCase().padStart(4, "0");
      }

      function analyticsMatchesMessageId(analyticsMessageId, targetMessageId) {
        const analytics = String(analyticsMessageId || "").toUpperCase();
        const target = String(targetMessageId || "");
        if (!analytics || !target) {
          return false;
        }
        if (analytics === target) {
          return true;
        }
        const targetHex = serverHexFromMessageId(target);
        return Boolean(targetHex && analytics.includes(`-${targetHex}-`));
      }

      function messageIdentifiers(message) {
        return [
          message?.descriptor?.messageId,
          message?.messageId,
          message?.serverMessageId,
          message?.messageAnalytics?.serverMessageId,
          message?.messageAnalytics?.analyticsMessageId,
          message?.metadata?.serverMessageId,
          message?.metadata?.messageId,
        ].filter((value) => value !== undefined && value !== null).map((value) => String(value));
      }

      function messageMatchesTarget(message, targetMessageId) {
        const target = String(targetMessageId || "");
        if (!target) {
          return false;
        }
        for (const id of messageIdentifiers(message)) {
          if (id === target || analyticsMatchesMessageId(id, target)) {
            return true;
          }
        }
        return false;
      }

      function decryptedContentFromFetchedMessages(result, targetMessageId) {
        const messages = Array.isArray(result?.messages) ? result.messages : [];
        let match = messages.find((message) => messageMatchesTarget(message, targetMessageId));
        if (!match && messages.length === 1) {
          match = messages[0];
        }
        const content = resultBytes(match?.content)
          || resultBytes(match?.messageContent?.content)
          || resultBytes(match?.messageContent);
        if (!content) {
          const text = textFromWebContent(match?.messageContent) || textFromWebContent(match?.content);
          if (!text) {
            return undefined;
          }
          return {
            content: text,
            webMessageId: String(match?.descriptor?.messageId || match?.messageId || ""),
            analyticsMessageId: String(match?.messageAnalytics?.analyticsMessageId || ""),
            contentSource: "web_message_text",
          };
        }
        return {
          content,
          webMessageId: String(match?.descriptor?.messageId || match?.messageId || ""),
          analyticsMessageId: String(match?.messageAnalytics?.analyticsMessageId || ""),
          contentSource: "web_message_bytes",
        };
      }

      async function resolveDecryptedContentViaMessaging(webpackRequire, targetConversationId, targetMessageId) {
        if (!targetConversationId || !targetMessageId) {
          return undefined;
        }
        const messaging = safeRequire(webpackRequire, 56639);
        const { appState, messagingClient } = await resolveMessagingClient(webpackRequire);
        const feedItem = appState?.messaging?.feed?.[targetConversationId];
        const conversationRef = feedItem?.conversationId || { id: uuidBytes(targetConversationId), str: targetConversationId };
        if (!messagingClient || !conversationRef?.id || !messaging?.uk) {
          return undefined;
        }
        const attempts = [
          () => messaging.uk(messagingClient, conversationRef),
        ];
        if (messaging.Gq) {
          attempts.push(() => messaging.Gq(messagingClient, conversationRef, undefined));
        }
        for (const fetchPage of attempts) {
          const result = await Promise.race([
            fetchPage(),
            new Promise((_, reject) => setTimeout(() => reject(new Error("messaging_fetch_timeout")), 5000)),
          ]);
          const match = decryptedContentFromFetchedMessages(result, targetMessageId);
          if (match) {
            return match;
          }
        }
        return undefined;
      }

      const content = fromBase64(payload.contentBase64);
      const cek = fromBase64(payload.cekBase64);
      const cekIv = fromBase64(payload.cekIvBase64);
      const nonce = fromBase64(payload.nonceBase64);
      const senderPublicKey = fromBase64(payload.senderPublicKeyBase64);

      if (cek.byteLength > 0) {
        try {
          const decrypted = await decryptAESGCM(cek, cekIv, content);
          if (decrypted) {
            return { ok: true, decryptedContentBase64: toBase64(decrypted), method: "inline_cek" };
          }
        } catch {
          return { ok: false, error: "inline_cek_decrypt_failed", retryable: false };
        }
      }

      const webpackRequire = findWebpackRequire();
      try {
        const messagingContent = await resolveDecryptedContentViaMessaging(webpackRequire, payload.conversationId, payload.messageId);
        if (messagingContent?.content) {
          return {
            ok: true,
            decryptedContentBase64: toBase64(messagingContent.content),
            method: "messaging_fetch",
            webMessageId: messagingContent.webMessageId,
            analyticsMessageId: messagingContent.analyticsMessageId,
            contentSource: messagingContent.contentSource,
          };
        }
      } catch (error) {
        // Fall through to direct key-manager probing. The bridge will log only the failure class.
      }

      const appState = resolveAppState(webpackRequire);
      const e2ee = await resolveE2EEWasmModule(webpackRequire, appState);
      const keyManager = e2ee.wasmModule?.e2ee_E2EEKeyManager;
      if (e2ee.error) {
        return {
          ok: false,
          error: "eel_key_manager_unavailable",
          retryable: true,
          failureClass: e2ee.error,
        };
      }
      if (!keyManager) {
        return { ok: false, error: "eel_key_manager_unavailable", retryable: true };
      }

      const attempts = [];
      const methodNames = Object.keys(keyManager).filter((name) => /decrypt|unwrap|shared|secret|cek/i.test(name));
      for (const methodName of methodNames) {
        const method = keyManager[methodName];
        if (typeof method !== "function") {
          continue;
        }
        for (const args of [
          [content, cekIv, nonce, senderPublicKey, payload.senderVersion],
          [senderPublicKey, payload.senderVersion, cekIv, nonce, content],
          [{ content, cekIv, nonce, senderPublicKey, senderVersion: payload.senderVersion }],
          [senderPublicKey, payload.senderVersion],
        ]) {
          try {
            const rawResult = await method.apply(keyManager, args);
            const bytesLike = resultBytes(rawResult);
            if (!bytesLike) {
              attempts.push(`${methodName}:no_bytes`);
              continue;
            }
            const bytes = ArrayBuffer.isView(bytesLike)
              ? new Uint8Array(bytesLike.buffer, bytesLike.byteOffset, bytesLike.byteLength)
              : bytesLike instanceof ArrayBuffer
                ? new Uint8Array(bytesLike)
                : Uint8Array.from(bytesLike);
            if (bytes.byteLength === content.byteLength) {
              return { ok: true, decryptedContentBase64: toBase64(bytes), method: methodName };
            }
            if ([16, 24, 32].includes(bytes.byteLength)) {
              const decrypted = await decryptAESGCM(bytes, cekIv, content);
              if (decrypted) {
                return { ok: true, decryptedContentBase64: toBase64(decrypted), method: `${methodName}:derived_cek` };
              }
            }
            attempts.push(`${methodName}:bytes_${bytes.byteLength}`);
          } catch (error) {
            attempts.push(`${methodName}:${String(error).replace(/\s+/g, " ").slice(0, 80)}`);
          }
        }
      }
      return { ok: false, error: "eel_decrypt_unavailable", retryable: true, failureClass: attempts.length > 0 ? "attempts_exhausted" : "no_candidate_methods" };
    }, {
      messageId,
      conversationId,
      contentBase64,
      cekBase64: String(input.cekBase64 || ""),
      cekIvBase64: String(input.cekIvBase64 || ""),
      nonceBase64: String(input.nonceBase64 || ""),
      senderPublicKeyBase64: String(input.senderPublicKeyBase64 || ""),
      senderVersion: Number(input.senderVersion || 0),
    }).catch((error) => ({
      ok: false,
      error: "eel_browser_evaluate_failed",
      retryable: true,
      failureClass: String(error?.message || error || "unknown").replace(/\s+/g, " ").slice(0, 160),
    }));
    if (!result.ok) {
      console.error(`eel-decrypt: failed messageId=${messageId} conversationId=${conversationId} error=${result.error || "unknown"} failureClass=${result.failureClass || ""}`);
    } else {
      console.error(`eel-decrypt: ok messageId=${messageId} conversationId=${conversationId} method=${result.method || "unknown"}`);
    }
    return result;
  }, { priority: 0 });
}

const debugTools = createDebugTools({ withLock, ensureSession });
export const getBrowserStorageSummary = debugTools.getBrowserStorageSummary;
export const searchBrowserBundle = debugTools.searchBrowserBundle;
export const getBrowserBundleModule = debugTools.getBrowserBundleModule;
export const getBrowserConversationMessages = debugTools.getBrowserConversationMessages;
export const getBrowserE2EESummary = debugTools.getBrowserE2EESummary;
export const deriveBrowserE2EESharedSecret = debugTools.deriveBrowserE2EESharedSecret;

export async function getChats() {
  return withLock(async () => {
    await ensureKnownChatsLoaded();
    const currentPage = await gotoSnapchat();
    if (!(await isLoggedIn(currentPage))) {
      throw new Error("snapchat session is not logged in");
    }

    let currentChats = [];
    try {
      await clearSearchBox(currentPage);
      await currentPage.waitForTimeout(CHAT_LIST_SETTLE_MS);
      currentChats = await extractChats(currentPage);
      if (currentChats.length === 0) {
        await returnToChatList(currentPage);
        currentChats = await extractChats(currentPage);
      }
    } catch (error) {
      if (knownChatsByKey.size > 0) {
        console.error(`getChats: quick scan failed, returning ${knownChatsByKey.size} cached chats: ${String(error)}`);
        return orderKnownChats([]);
      }
      throw error;
    }
    mergeKnownChats(currentChats);

    const hasWarmCache = knownChatsByKey.size >= CHAT_DEEP_SCAN_MIN_CACHE;
    const shouldDeepScan = !hasWarmCache && (lastDeepChatSync === 0 || Date.now() - lastDeepChatSync > CHAT_DEEP_REFRESH_MS);
    if (shouldDeepScan) {
      try {
        const deepChats = await extractChats(currentPage, { deep: true });
        mergeKnownChats(deepChats);
        lastDeepChatSync = Date.now();
        currentChats = deepChats.length > 0 ? deepChats : currentChats;
        console.error(`getChats: deep scan found ${deepChats.length} chats, cache=${knownChatsByKey.size}`);
      } catch (error) {
        lastDeepChatSync = Date.now();
        console.error(`getChats: deep scan failed: ${String(error)}`);
      }
    }

    await saveKnownChats();
    return orderKnownChats(currentChats);
  }, { priority: 2 });
}

export async function getMessages(chatID, chatName, chatURL = "") {
  return withLock(async () => {
    let lastError;
    for (let attempt = 1; attempt <= MESSAGE_ATTEMPTS; attempt += 1) {
      try {
        const currentPage = await gotoSnapchat({ reload: attempt > 1 });
        const resolvedChat = await resolveChat(currentPage, chatID, chatName, chatURL);
        const openedPage = await openChat(resolvedChat.name, resolvedChat.id, resolvedChat.url, { searchFallback: false });
        const activeChat = await getActiveConversation(openedPage, resolvedChat.name, resolvedChat.id);
        const messages = await extractMessages(openedPage, activeChat.name || resolvedChat.name, activeChat.id || resolvedChat.id);
        const sidebarStatus = makeSidebarStatusMessage(resolvedChat, activeChat);
        if (sidebarStatus && !messages.some((message) => message.id === sidebarStatus.id || normalizeText(message.text) === normalizeText(sidebarStatus.text))) {
          return [...messages, sidebarStatus].slice(-80);
        }
        return messages;
      } catch (error) {
        lastError = error;
        console.error(`getMessages: attempt=${attempt} failed chatId=${chatID} chatName=${JSON.stringify(chatName)} chatUrl=${JSON.stringify(chatURL)} error=${String(error)}`);
      }
    }
    throw lastError;
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

    const sendButton = await findSendButton(openedPage);

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

export async function sendMedia(chatID, chatName, chatURL = "", fileName = "snap-media.bin", mimeType = "application/octet-stream", dataBase64, caption = "") {
  return withLock(async () => {
    console.error(`sendMedia: begin chatId=${chatID} chatName=${JSON.stringify(chatName)} chatUrl=${JSON.stringify(chatURL)} fileName=${JSON.stringify(fileName)} mimeType=${JSON.stringify(mimeType)}`);
    const data = Buffer.from(String(dataBase64 || ""), "base64");
    if (!data.length) {
      throw new Error("media payload was empty");
    }

    await fs.mkdir(traceDir, { recursive: true });
    const safeName = sanitizeUploadFileName(fileName);
    const tempPath = path.join(traceDir, `upload-${Date.now()}-${stableID([safeName, String(data.length)])}-${safeName}`);
    await fs.writeFile(tempPath, data);

    try {
      const currentPage = await gotoSnapchat();
      const resolvedChat = await resolveChat(currentPage, chatID, chatName, chatURL);
      console.error(`sendMedia: resolved chat => id=${resolvedChat.id} name=${JSON.stringify(resolvedChat.name)} url=${JSON.stringify(resolvedChat.url)}`);
      const openedPage = await openChat(resolvedChat.name, resolvedChat.id, resolvedChat.url);
      const activeChat = await getActiveConversation(openedPage, resolvedChat.name, resolvedChat.id);
      console.error(`sendMedia: active conversation => id=${activeChat.id} name=${JSON.stringify(activeChat.name)} url=${JSON.stringify(activeChat.url)} headers=${JSON.stringify(activeChat.headers)}`);

      await attachMediaFile(openedPage, tempPath);
      console.error("sendMedia: attached media file");

      const cleanCaption = normalizeText(caption);
      if (cleanCaption) {
        try {
          const composer = await getComposer(openedPage);
          await setComposerText(openedPage, composer, cleanCaption);
          console.error("sendMedia: populated caption");
        } catch (error) {
          console.error(`sendMedia: caption skipped: ${String(error)}`);
        }
      }

      const sendButton = await findSendButton(openedPage);
      if (sendButton) {
        console.error("sendMedia: clicking send button");
        await sendButton.click();
      } else {
        console.error("sendMedia: send button not found, pressing Enter");
        await openedPage.keyboard.press("Enter");
      }
      await openedPage.waitForTimeout(3500);
      const postSendChat = await getActiveConversation(openedPage, activeChat.name || resolvedChat.name, activeChat.id || resolvedChat.id);
      const expectedID = activeChat.id || resolvedChat.id;
      const confirmed = !expectedID || sameConversationID(postSendChat.id, expectedID);
      if (!confirmed) {
        const snapshot = await captureDebugSnapshot(openedPage);
        throw new Error(`media send did not stay in expected chat "${activeChat.name || resolvedChat.name}" (snapshot=${snapshot.screenshotPath})`);
      }
      console.error("sendMedia: queued");
      return {
        queued: true,
        confirmed: true,
        chatID: postSendChat.id || activeChat.id || resolvedChat.id || chatID || makeChatID(resolvedChat.name),
        chatName: postSendChat.name || activeChat.name || resolvedChat.name,
        chatURL: (postSendChat.id && conversationURL(postSendChat.id)) || resolvedChat.url || "",
        text: cleanCaption || safeName,
      };
    } finally {
      try {
        await fs.unlink(tempPath);
      } catch {
      }
    }
  }, { priority: 0 });
}

export async function setTyping(chatID, typing, durationMs = 1500) {
  return {
    ok: false,
    error: `Snapchat typing is not wired yet for chat ${chatID || "<missing>"}; next step is the snapcap-native setTyping adapter (typing=${Boolean(typing)}, durationMs=${Number(durationMs || 0)})`,
  };
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

    const scrollCandidates = await currentPage.evaluate(() => {
      const widthLimit = Math.max(window.innerWidth * 0.45, 420);
      const rows = Array.from(document.querySelectorAll("[role='button'][aria-labelledby*='title-'], [role='link'][aria-labelledby*='title-']"));
      return Array.from(document.querySelectorAll("div, main, section, aside, nav"))
        .map((element) => {
          const rect = element.getBoundingClientRect();
          const rowCount = rows.filter((row) => element.contains(row)).length;
          const text = (element.textContent || "").replace(/\s+/g, " ").trim();
          return {
            tag: element.tagName.toLowerCase(),
            role: element.getAttribute("role") || "",
            className: typeof element.className === "string" ? element.className.slice(0, 120) : "",
            left: Math.round(rect.left),
            top: Math.round(rect.top),
            width: Math.round(rect.width),
            height: Math.round(rect.height),
            scrollTop: Math.round(element.scrollTop),
            clientHeight: Math.round(element.clientHeight),
            scrollHeight: Math.round(element.scrollHeight),
            overflowY: window.getComputedStyle(element).overflowY,
            rowCount,
            text: text.slice(0, 160),
            inSidebar: rect.left < widthLimit,
          };
        })
        .filter((candidate) => candidate.inSidebar && (candidate.rowCount > 0 || candidate.scrollHeight > candidate.clientHeight + 20))
        .sort((a, b) => (b.rowCount - a.rowCount) || ((b.scrollHeight - b.clientHeight) - (a.scrollHeight - a.clientHeight)))
        .slice(0, 20);
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
      scrollCandidates,
      snapshot,
      selectorHints: {
        loginTextPattern: loginTextPattern.source,
        shellTextPattern: shellTextPattern.source,
        composerPattern: composerPattern.source,
      },
    };
  });
}
