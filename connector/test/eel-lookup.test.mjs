// Focused tests for the targeted EEL message lookup. The lookup helpers are
// defined inside a Playwright page.evaluate closure in snapchat.mjs (they must
// exist in the browser context), so these tests extract the real function
// sources from the module and evaluate them here. This keeps a single source
// of truth: the tested code is literally the shipped code, and any edit that
// breaks or loosens the strict matching fails these tests.

import test from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";

const sourcePath = new URL("../src/snapchat.mjs", import.meta.url);
const source = readFileSync(sourcePath, "utf8");

function extractFunction(name) {
  const marker = `function ${name}(`;
  const start = source.indexOf(marker);
  assert.ok(start >= 0, `function ${name} not found in snapchat.mjs`);
  const bodyStart = source.indexOf("{", start);
  assert.ok(bodyStart > start, `function ${name} has no opening brace`);
  let depth = 0;
  for (let index = bodyStart; index < source.length; index += 1) {
    const char = source[index];
    if (char === "{") {
      depth += 1;
    } else if (char === "}") {
      depth -= 1;
      if (depth === 0) {
        return source.slice(start, index + 1);
      }
    }
  }
  assert.fail(`function ${name} is unbalanced`);
}

const helperNames = [
  "textFromWebContent",
  "resultBytes",
  "serverHexFromMessageId",
  "analyticsMatchesMessageId",
  "messageIdentifiers",
  "messageMatchesTarget",
  "decryptedContentFromFetchedMessages",
];

const build = new Function(
  `${helperNames.map((name) => extractFunction(name)).join("\n\n")}\nreturn { ${helperNames.join(", ")} };`,
);
const helpers = build();

const bytes267 = new Uint8Array([0x26, 0x07, 0x00, 0x01]);
const bytes268 = new Uint8Array([0x26, 0x08, 0x00, 0x02]);
const bytesUnrelated = new Uint8Array([0x09, 0x09, 0x09]);

test("strict lookup: no single-message fallback remains in snapchat.mjs", () => {
  assert.equal(source.includes("messages.length === 1"), false, "the misattributing single-message fallback must be removed");
});

test("strict lookup: exact target found returns only the target's bytes", () => {
  const messages = [
    { descriptor: { messageId: "267" }, content: bytes267 },
    { descriptor: { messageId: "268" }, content: bytes268 },
  ];
  const result = helpers.decryptedContentFromFetchedMessages({ messages }, "268");
  assert.equal(result.exactMatch, true);
  assert.equal(result.matchedWebMessageId, "268");
  assert.equal(result.messageCount, 2);
  assert.equal(result.requestedMessageId, "268");
  assert.equal(result.contentSource, "web_message_bytes");
  assert.deepEqual(Array.from(result.content), Array.from(bytes268));
  assert.notDeepEqual(Array.from(result.content), Array.from(bytes267));
});

test("strict lookup: target absent with one unrelated message is NOT attributed", () => {
  const messages = [{ descriptor: { messageId: "267" }, content: bytes267 }];
  const result = helpers.decryptedContentFromFetchedMessages({ messages }, "268");
  assert.equal(result.exactMatch, false);
  assert.equal(result.content, undefined);
  assert.equal(result.matchedWebMessageId, "");
  assert.equal(result.messageCount, 1);
  assert.equal(result.missReason, "no_exact_match");
});

test("strict lookup: target absent with multiple messages is NOT attributed", () => {
  const messages = [
    { messageId: "266", content: bytesUnrelated },
    { descriptor: { messageId: "267" }, content: bytes267 },
  ];
  const result = helpers.decryptedContentFromFetchedMessages({ messages }, "268");
  assert.equal(result.exactMatch, false);
  assert.equal(result.content, undefined);
  assert.equal(result.messageCount, 2);
  assert.equal(result.missReason, "no_exact_match");
});

test("strict lookup: matches descriptor.messageId", () => {
  const messages = [
    { messageId: "999", content: bytesUnrelated },
    { descriptor: { messageId: "268" }, content: bytes268 },
  ];
  const result = helpers.decryptedContentFromFetchedMessages({ messages }, "268");
  assert.equal(result.exactMatch, true);
  assert.equal(result.matchedWebMessageId, "268");
  assert.deepEqual(Array.from(result.content), Array.from(bytes268));
});

test("strict lookup: matches message.messageId", () => {
  const messages = [{ messageId: "268", content: bytes268 }];
  const result = helpers.decryptedContentFromFetchedMessages({ messages }, "268");
  assert.equal(result.exactMatch, true);
  assert.equal(result.matchedWebMessageId, "268");
  assert.deepEqual(Array.from(result.content), Array.from(bytes268));
});

test("strict lookup: exact match with text-only content returns the text", () => {
  const messages = [{ descriptor: { messageId: "268" }, messageContent: { text: "ZEBRA_CAPTION_9274" } }];
  const result = helpers.decryptedContentFromFetchedMessages({ messages }, "268");
  assert.equal(result.exactMatch, true);
  assert.equal(result.matchedWebMessageId, "268");
  assert.equal(result.content, "ZEBRA_CAPTION_9274");
  assert.equal(result.contentSource, "web_message_text");
});

test("strict lookup: matched message without content is reported as a miss", () => {
  const messages = [{ descriptor: { messageId: "268" } }];
  const result = helpers.decryptedContentFromFetchedMessages({ messages }, "268");
  assert.equal(result.exactMatch, false);
  assert.equal(result.content, undefined);
  assert.equal(result.missReason, "matched_message_has_no_content");
});

test("strict lookup: empty page never attributes content", () => {
  const result = helpers.decryptedContentFromFetchedMessages({ messages: [] }, "268");
  assert.equal(result.exactMatch, false);
  assert.equal(result.content, undefined);
  assert.equal(result.messageCount, 0);
});

test("messageMatchesTarget: analytics id embedding the server hex matches", () => {
  assert.equal(helpers.analyticsMatchesMessageId("ABC-010C-DEF", "268"), true);
  assert.equal(helpers.analyticsMatchesMessageId("ABC-010D-DEF", "268"), false);
  assert.equal(helpers.messageMatchesTarget({ messageAnalytics: { analyticsMessageId: "ABC-010C-DEF" } }, "268"), true);
});
