import { createRequire } from "node:module";
import { MemoryDataStore, SnapcapClient } from "../.codex-build/snapcap-native/dist/index.js";

const requireFromSnapcap = createRequire(new URL("../.codex-build/snapcap-native/package.json", import.meta.url));
const { CookieJar } = requireFromSnapcap("tough-cookie");

const connectorURL = process.env.SNAPCHAT_CONNECTOR_URL || "http://127.0.0.1:3101";
const secret = process.env.SNAPCHAT_SHARED_SECRET || "";
const waitMs = Number(process.env.SNAPCAP_PROBE_WAIT_MS || "20000");
const conversations = (process.env.SNAPCAP_PROBE_CONVERSATIONS || "")
  .split(",")
  .map((value) => value.trim())
  .filter(Boolean);

if (!secret) {
  throw new Error("SNAPCHAT_SHARED_SECRET is required");
}

const headers = { "X-Bridge-Secret": secret };
const authResp = await fetch(`${connectorURL}/session/api-auth`, { headers });
if (!authResp.ok) {
  throw new Error(`api-auth failed: ${authResp.status}`);
}
const auth = await authResp.json();
if (!auth.authenticated || !auth.cookieString) {
  throw new Error("connector session is not authenticated");
}

const store = new MemoryDataStore();
const jar = new CookieJar();
for (const cookie of auth.cookies || []) {
  if (!cookie?.name || !cookie?.value) continue;
  const parts = [`${cookie.name}=${cookie.value}`];
  parts.push(`Domain=${cookie.domain || ".snapchat.com"}`);
  parts.push(`Path=${cookie.path || "/"}`);
  if (cookie.secure !== false) parts.push("Secure");
  if (cookie.httpOnly) parts.push("HttpOnly");
  await jar.setCookie(parts.join("; "), "https://www.snapchat.com/web");
}
await store.set("cookie_jar", new TextEncoder().encode(JSON.stringify(jar.serializeSync())));

const client = new SnapcapClient({
  dataStore: store,
  browser: {
    userAgent: auth.browserUserAgent || auth.snapClientUserAgent,
    viewport: { width: 1280, height: 900 },
  },
});

const seen = [];
client.messaging.on("message", (message) => {
  const text = new TextDecoder().decode(message.content);
  const raw = message.raw || {};
  seen.push({
    conversationId: raw.conversationId,
    isSender: message.isSender,
    contentType: message.contentType,
    byteLength: message.content.byteLength,
    textPreview: text.slice(0, 120),
  });
  console.log(JSON.stringify(seen.at(-1)));
});

await client.authenticate();
const convs = await client.messaging.listConversations(auth.selfUserID);
console.error(JSON.stringify({
  authenticated: true,
  selfUserID: auth.selfUserID,
  conversationCount: convs.length,
  targetConversations: conversations,
}));

// Touch target conversations through the public API so the bundle session
// prioritizes them when the subscription brings up its inbox pump.
for (const conv of convs.filter((item) => conversations.includes(item.id)).slice(0, 5)) {
  console.error(JSON.stringify({ target: conv.id, title: conv.title || conv.name || "" }));
}

await new Promise((resolve) => setTimeout(resolve, waitMs));
console.error(JSON.stringify({ plaintextSeen: seen.length }));
