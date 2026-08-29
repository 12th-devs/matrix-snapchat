export function createDebugTools({ withLock, ensureSession }) {
  if (typeof withLock !== "function" || typeof ensureSession !== "function") {
    throw new Error("debug tools require withLock and ensureSession");
  }

async function getBrowserStorageSummary() {
  return withLock(async () => {
    const { page: currentPage } = await ensureSession();
    if (!currentPage || currentPage.isClosed()) {
      return { ok: false, error: "no active page" };
    }
    return await currentPage.evaluate(async () => {
      function summarizeValue(value, depth = 0, seen = new Set()) {
        if (value == null) {
          return { type: String(value) };
        }
        if (typeof value !== "object") {
          const text = String(value);
          return {
            type: typeof value,
            length: typeof value === "string" ? value.length : undefined,
            preview: typeof value === "string" ? text.slice(0, 80) : undefined,
          };
        }
        if (seen.has(value) || depth > 2) {
          return { type: "object", circularOrMaxDepth: true };
        }
        seen.add(value);
        if (ArrayBuffer.isView(value)) {
          return { type: value.constructor?.name || "TypedArray", byteLength: value.byteLength };
        }
        if (value instanceof ArrayBuffer) {
          return { type: "ArrayBuffer", byteLength: value.byteLength };
        }
        if (value instanceof Blob) {
          return { type: "Blob", size: value.size, mimeType: value.type };
        }
        if (value instanceof CryptoKey) {
          return {
            type: "CryptoKey",
            algorithm: value.algorithm?.name || "",
            extractable: value.extractable,
            usages: value.usages || [],
          };
        }
        const keys = Object.keys(value).slice(0, 30);
        const interesting = {};
        for (const key of keys) {
          if (!/(fidelius|eel|encrypt|crypto|key|private|public|rwk|cek|secret)/i.test(key)) {
            continue;
          }
          try {
            interesting[key] = summarizeValue(value[key], depth + 1, seen);
          } catch (error) {
            interesting[key] = { error: String(error).slice(0, 120) };
          }
        }
        return {
          type: Array.isArray(value) ? "array" : (value.constructor?.name || "object"),
          keys,
          interesting,
        };
      }

      async function readIndexedDB() {
        if (!indexedDB.databases) {
          return { supported: false };
        }
        const databases = await indexedDB.databases();
        const output = [];
        for (const dbInfo of databases) {
          const name = dbInfo.name;
          if (!name) {
            continue;
          }
          const dbResult = { name, version: dbInfo.version, stores: [] };
          try {
            const db = await new Promise((resolve, reject) => {
              const request = indexedDB.open(name);
              request.onsuccess = () => resolve(request.result);
              request.onerror = () => reject(request.error || new Error("open failed"));
            });
            for (const storeName of Array.from(db.objectStoreNames)) {
              const storeResult = { name: storeName, samples: [] };
              try {
                const tx = db.transaction(storeName, "readonly");
                const store = tx.objectStore(storeName);
                const keys = await new Promise((resolve, reject) => {
                  const request = store.getAllKeys(undefined, 8);
                  request.onsuccess = () => resolve(request.result || []);
                  request.onerror = () => reject(request.error || new Error("getAllKeys failed"));
                });
                for (const key of keys) {
                  const value = await new Promise((resolve, reject) => {
                    const request = store.get(key);
                    request.onsuccess = () => resolve(request.result);
                    request.onerror = () => reject(request.error || new Error("get failed"));
                  });
                  storeResult.samples.push({
                    key: summarizeValue(key),
                    value: summarizeValue(value),
                  });
                }
              } catch (error) {
                storeResult.error = String(error).slice(0, 160);
              }
              dbResult.stores.push(storeResult);
            }
            db.close();
          } catch (error) {
            dbResult.error = String(error).slice(0, 160);
          }
          output.push(dbResult);
        }
        return { supported: true, databases: output };
      }

      const localStorageSummary = [];
      for (let index = 0; index < localStorage.length; index++) {
        const key = localStorage.key(index);
        const value = localStorage.getItem(key);
        localStorageSummary.push({
          key,
          length: value ? value.length : 0,
          interesting: /(fidelius|eel|encrypt|crypto|key|private|public|rwk|cek|secret)/i.test(`${key} ${value || ""}`),
          preview: value ? value.slice(0, 120) : "",
        });
      }

      return {
        url: location.href,
        localStorage: localStorageSummary,
        indexedDB: await readIndexedDB(),
      };
    });
  }, { priority: 0 });
}


async function searchBrowserBundle(query) {
  const needle = String(query || "").trim();
  if (!needle || needle.length < 3) {
    return { ok: false, error: "query must be at least 3 characters" };
  }
  return withLock(async () => {
    const { page: currentPage } = await ensureSession();
    if (!currentPage || currentPage.isClosed()) {
      return { ok: false, error: "no active page" };
    }
    return await currentPage.evaluate((needleArg) => {
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
            chunk.push([[`codex-bundle-search-${Date.now()}`], {}, (req) => {
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

      const webpackRequire = findWebpackRequire();
      const pattern = new RegExp(needleArg.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"), "i");
      const results = [];
      if (!webpackRequire?.m) {
        return { ok: false, error: "webpack modules not found", url: location.href };
      }
      for (const [moduleID, factory] of Object.entries(webpackRequire.m)) {
        let source = "";
        try {
          source = Function.prototype.toString.call(factory);
        } catch {
          continue;
        }
        const match = source.search(pattern);
        if (match < 0) {
          continue;
        }
        const start = Math.max(0, match - 500);
        const end = Math.min(source.length, match + 1200);
        results.push({
          moduleID,
          sourceLength: source.length,
          matchOffset: match,
          snippet: source.slice(start, end),
        });
        if (results.length >= 20) {
          break;
        }
      }
      return { ok: true, query: needleArg, url: location.href, moduleCount: Object.keys(webpackRequire.m).length, results };
    }, needle);
  }, { priority: 0 });
}


async function getBrowserBundleModule(moduleID) {
  const id = String(moduleID || "").trim();
  if (!id) {
    return { ok: false, error: "module id is required" };
  }
  return withLock(async () => {
    const { page: currentPage } = await ensureSession();
    if (!currentPage || currentPage.isClosed()) {
      return { ok: false, error: "no active page" };
    }
    return await currentPage.evaluate((moduleIDArg) => {
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
            chunk.push([[`codex-bundle-module-${Date.now()}`], {}, (req) => {
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

      const webpackRequire = findWebpackRequire();
      if (!webpackRequire?.m) {
        return { ok: false, error: "webpack modules not found", url: location.href };
      }
      const factory = webpackRequire.m[moduleIDArg];
      if (!factory) {
        return { ok: false, error: "module not found", url: location.href };
      }
      let source = "";
      try {
        source = Function.prototype.toString.call(factory);
      } catch (error) {
        return { ok: false, error: String(error).slice(0, 160), url: location.href };
      }
      let exportKeys = [];
      let exportValue;
      try {
        const moduleExports = webpackRequire(moduleIDArg);
        exportKeys = Object.keys(moduleExports || {});
        if (typeof moduleExports === "string" || typeof moduleExports === "number" || typeof moduleExports === "boolean") {
          exportValue = moduleExports;
        }
      } catch {
        exportKeys = [];
      }
      return { ok: true, moduleID: moduleIDArg, url: location.href, publicPath: webpackRequire.p || "", source, exportKeys, exportValue };
    }, id);
  }, { priority: 0 });
}

async function getBrowserConversationMessages(chatID, limit = 20) {
  const id = String(chatID || "").trim();
  if (!id) {
    return { ok: false, error: "chat id is required" };
  }
  return withLock(async () => {
    const { context, page: sessionPage } = await ensureSession();
    const pages = typeof context?.pages === "function" ? context.pages() : [sessionPage].filter(Boolean);
    const currentPage = pages
      .filter((candidate) => candidate && !candidate.isClosed())
      .sort((left, right) => {
        const score = (candidate) => {
          const url = candidate.url();
          if (/^https:\/\/www\.snapchat\.com\/web\b/i.test(url)) {
            return 3;
          }
          if (/^https:\/\/[^/]*snapchat\.com\b/i.test(url)) {
            return 2;
          }
          if (url && url !== "about:blank" && !url.startsWith("chrome://newtab")) {
            return 1;
          }
          return 0;
        };
        return score(right) - score(left);
      })[0] || sessionPage;
    if (!currentPage || currentPage.isClosed()) {
      return { ok: false, error: "no active page" };
    }
    if (!/^https:\/\/[^/]*snapchat\.com\b/i.test(currentPage.url())) {
      try {
        await currentPage.goto("https://www.snapchat.com/web", { waitUntil: "domcontentloaded", timeout: 30000 });
      } catch {
        // If the profile already has a valid Snapchat page racing in another tab, the frame loop below will report it.
      }
    }
    function summarizeFrameResult(item) {
      return {
        ok: item.ok,
        frameURL: item.frameURL,
        error: item.error,
        appStateModule: item.appStateModule,
        initAttempt: item.initAttempt,
        hasMessagingClient: item.hasMessagingClient,
        hasUserId: item.hasUserId,
        wasmKeys: item.wasmKeys,
        hasWasmWorkerProxy: item.hasWasmWorkerProxy,
        hasFeedItem: item.hasFeedItem,
        hasConversationRef: item.hasConversationRef,
        userSummary: item.userSummary,
        authSummary: item.authSummary,
        accountSummary: item.accountSummary,
        selectedUserSummary: item.selectedUserSummary,
        selectedUserError: item.selectedUserError,
        feedKeysSample: item.feedKeysSample,
        messagingStateKeys: item.messagingStateKeys,
        actionTypes: item.actionTypes,
        messageCount: item.messages?.length || 0,
      };
    }
    async function probeFrame(targetFrame) {
      return await targetFrame.evaluate(async ({ chatIDArg, limitArg }) => {
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
            chunk.push([[`codex-conversation-debug-${Date.now()}`], {}, (req) => {
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

      function safeRequire(webpackRequire, moduleId) {
        try {
          return webpackRequire?.(moduleId);
        } catch {
          return undefined;
        }
      }

      function resolveAppState(webpackRequire) {
        for (const moduleId of [96821, 97003]) {
          const appState = safeRequire(webpackRequire, moduleId)?.M?.getState?.();
          if (appState) {
            return { appState, moduleId };
          }
        }
        return {};
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

      async function resolveMessagingState(webpackRequire) {
        let { appState, moduleId } = resolveAppState(webpackRequire);
        let initAttempt = "";
        const fetchUserData = appState?.auth?.fetchUserData;
        if (!appState?.auth?.me && typeof fetchUserData === "function") {
          try {
            await Promise.race([
              fetchUserData("codex_message_debug"),
              new Promise((_, reject) => setTimeout(() => reject(new Error("fetch_user_data_timeout")), 5000)),
            ]);
            initAttempt = "user_ok";
          } catch (error) {
            initAttempt = String(error?.message || error || "fetch_user_data_failed").slice(0, 200);
          }
          ({ appState, moduleId } = resolveAppState(webpackRequire));
        }
        const initWasm = appState?.wasm?.initialize;
        if (!appState?.wasm?.workerProxy && typeof initWasm === "function") {
          try {
            await Promise.race([
              initWasm(),
              new Promise((_, reject) => setTimeout(() => reject(new Error("wasm_init_timeout")), 30000)),
            ]);
            initAttempt = initAttempt ? `${initAttempt};wasm_ok` : "wasm_ok";
          } catch (error) {
            const message = String(error?.message || error || "wasm_init_failed").slice(0, 200);
            initAttempt = initAttempt ? `${initAttempt};${message}` : message;
          }
          for (let i = 0; i < 100; i++) {
            ({ appState, moduleId } = resolveAppState(webpackRequire));
            if (appState?.wasm?.workerProxy) {
              break;
            }
            await new Promise((resolve) => setTimeout(resolve, 100));
          }
        }
        const initClient = appState?.messaging?.initClient || appState?.messaging?.initializeClient;
        if (!appState?.messaging?.client && typeof initClient === "function") {
          try {
            await Promise.race([
              initClient(),
              new Promise((_, reject) => setTimeout(() => reject(new Error("messaging_init_timeout")), 30000)),
            ]);
            initAttempt = initAttempt ? `${initAttempt};messaging_ok` : "messaging_ok";
          } catch (error) {
            const message = String(error?.message || error || "init_failed").slice(0, 200);
            initAttempt = initAttempt ? `${initAttempt};${message}` : message;
          }
          for (let i = 0; i < 100; i++) {
            ({ appState, moduleId } = resolveAppState(webpackRequire));
            if (appState?.messaging?.client) {
              break;
            }
            await new Promise((resolve) => setTimeout(resolve, 100));
          }
        }
        return { appState, moduleId, initAttempt };
      }

      function textFromWebContent(value, depth = 0, seen = new Set()) {
        if (!value || depth > 8) {
          return "";
        }
        if (typeof value === "string") {
          return value.trim();
        }
        if (Array.isArray(value)) {
          return value.map((item) => textFromWebContent(item, depth + 1, seen)).filter(Boolean).join("\n\n").trim();
        }
        if (typeof value !== "object" || seen.has(value)) {
          return "";
        }
        seen.add(value);
        const directText = value.text;
        if (typeof directText === "string" && directText.trim()) {
          return directText.trim();
        }
        if (directText && typeof directText === "object") {
          const text = textFromWebContent(directText, depth + 1, seen);
          if (text) {
            return text;
          }
        }
        for (const key of [
          "messageContent",
          "content",
          "chatMessage",
          "textContent",
          "slideupText",
          "statusMessage",
          "component",
          "components",
          "quotedMessage",
        ]) {
          const text = textFromWebContent(value[key], depth + 1, seen);
          if (text) {
            return text;
          }
        }
        return "";
      }

      function bytesSummary(value) {
        if (!value) {
          return undefined;
        }
        const bytes = ArrayBuffer.isView(value)
          ? new Uint8Array(value.buffer, value.byteOffset, value.byteLength)
          : value instanceof ArrayBuffer
            ? new Uint8Array(value)
            : Array.isArray(value)
              ? Uint8Array.from(value)
              : undefined;
        if (!bytes) {
          return undefined;
        }
        let utf8 = "";
        try {
          utf8 = new TextDecoder().decode(bytes).replace(/\s+/g, " ").trim().slice(0, 240);
        } catch {
        }
        return {
          type: value.constructor?.name || "bytes",
          byteLength: bytes.byteLength,
          utf8,
          firstBytesHex: Array.from(bytes.slice(0, 16)).map((byte) => byte.toString(16).padStart(2, "0")).join(""),
        };
      }

      function summarizeValue(value, depth = 0, seen = new Set()) {
        if (value == null) {
          return { type: String(value) };
        }
        if (ArrayBuffer.isView(value) || value instanceof ArrayBuffer || Array.isArray(value) && value.every((item) => typeof item === "number")) {
          return bytesSummary(value);
        }
        if (typeof value !== "object" || seen.has(value) || depth > 2) {
          const simple = typeof value === "string" || typeof value === "number" || typeof value === "boolean" || typeof value === "bigint";
          return { type: typeof value, value: simple ? String(value).slice(0, 240) : undefined };
        }
        seen.add(value);
        const keys = Object.keys(value).slice(0, 50);
        const fields = {};
        for (const key of keys) {
          if (!/(id|text|content|type|sender|author|timestamp|metadata|message|snap|media|decrypt|failure)/i.test(key)) {
            continue;
          }
          try {
            fields[key] = summarizeValue(value[key], depth + 1, seen);
          } catch (error) {
            fields[key] = { error: String(error).slice(0, 120) };
          }
        }
        return { type: Array.isArray(value) ? "array" : value.constructor?.name || "object", keys, fields };
      }

      function summarizeStoreSection(value) {
        if (!value || typeof value !== "object") {
          return { type: String(value) };
        }
        const summary = {
          type: Array.isArray(value) ? "array" : value.constructor?.name || "object",
          keys: Object.keys(value).slice(0, 120),
        };
        for (const key of ["id", "userId", "user_id", "uuid", "username", "displayName", "display_name", "currentUserId", "currentUser"]) {
          const item = value[key];
          if (item === undefined || item === null) {
            continue;
          }
          if (typeof item === "string" || typeof item === "number" || typeof item === "boolean") {
            summary[key] = String(item).slice(0, 160);
          } else if (typeof item === "object") {
            summary[key] = {
              type: item.constructor?.name || "object",
              keys: Object.keys(item).slice(0, 40),
              id: String(item.id || item.userId || item.user_id || item.uuid || "").slice(0, 160),
              username: String(item.username || item.mutable_username || item.displayName || item.display_name || "").slice(0, 160),
            };
          }
        }
        return summary;
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

      function conversationMessagesFromState(appState, chatID) {
        const conversations = appState?.messaging?.conversations || {};
        const conversation = conversations?.[chatID]
          || conversations?.[String(chatID)]
          || Object.values(conversations || {}).find((item) => {
            const candidate = item?.conversationId?.str
              || item?.conversationId
              || item?.id
              || item?.descriptor?.conversationId?.str;
            return String(candidate || "") === String(chatID);
          });
        const candidates = [
          conversation?.messages,
          conversation?.messageList,
          conversation?.conversationMessages,
          conversation?.items,
          conversation?.entries,
          conversation?.messagesById && Object.values(conversation.messagesById),
        ];
        for (const value of candidates) {
          if (Array.isArray(value) && value.length > 0) {
            return { conversation, messages: value };
          }
        }
        return { conversation, messages: [] };
      }

      function summarizeMessages(messages, attempt, limitArg) {
        return messages.slice(-Number(limitArg || 20)).map((message) => ({
          attempt,
          identifiers: messageIdentifiers(message),
          keys: Object.keys(message || {}).slice(0, 80),
          text: textFromWebContent(message?.messageContent) || textFromWebContent(message?.content) || textFromWebContent(message),
          contentType: String(message?.messageContent?.contentType ?? message?.contentType ?? ""),
          senderId: String(message?.senderId || message?.metadata?.senderId || message?.sender?.id || ""),
          isSender: Boolean(message?.isSender),
          messageContent: summarizeValue(message?.messageContent),
          content: summarizeValue(message?.content),
          metadata: summarizeValue(message?.metadata),
          analytics: summarizeValue(message?.messageAnalytics),
        }));
      }

      const webpackRequire = findWebpackRequire();
      const { appState, moduleId, initAttempt } = await resolveMessagingState(webpackRequire);
      const messaging = safeRequire(webpackRequire, 56639);
      const authSelectors = safeRequire(webpackRequire, 31995);
      let selectedUser;
      let selectedUserError;
      try {
        selectedUser = authSelectors?.Zw?.(appState);
      } catch (error) {
        selectedUserError = String(error?.message || error || "selector_failed").slice(0, 200);
      }
      const feedItem = appState?.messaging?.feed?.[chatIDArg];
      const messagingClient = appState?.messaging?.client;
      const conversationRef = feedItem?.conversationId || { id: uuidBytes(chatIDArg), str: chatIDArg };
      const output = {
        ok: Boolean(conversationRef?.id),
        url: location.href,
        appStateModule: moduleId,
        initAttempt,
        hasMessagingClient: Boolean(messagingClient),
        hasUserId: Boolean(appState?.user?.userId || appState?.auth?.userId || appState?.account?.userId),
        wasmKeys: Object.keys(appState?.wasm || {}).slice(0, 80),
        hasWasmWorkerProxy: Boolean(appState?.wasm?.workerProxy),
        hasFeedItem: Boolean(feedItem),
        hasConversationRef: Boolean(conversationRef?.id),
        userSummary: summarizeStoreSection(appState?.user),
        authSummary: summarizeStoreSection(appState?.auth),
        accountSummary: summarizeStoreSection(appState?.account),
        selectedUserSummary: summarizeStoreSection(selectedUser),
        selectedUserError,
        messagingStateKeys: Object.keys(appState?.messaging || {}).slice(0, 120),
        feedItemKeys: feedItem ? Object.keys(feedItem).slice(0, 80) : [],
        messagingExportKeys: Object.keys(messaging || {}).slice(0, 80),
        actionTypes: {},
        messages: [],
        attempts: [],
      };
      if (!output.ok) {
        output.feedKeysSample = Object.keys(appState?.messaging?.feed || {}).slice(0, 20);
        return output;
      }

      const stateBefore = conversationMessagesFromState(appState, chatIDArg);
      if (stateBefore.messages.length > 0) {
        output.attempts.push({
          name: "state-before",
          ok: true,
          conversationKeys: Object.keys(stateBefore.conversation || {}).slice(0, 80),
          messageCount: stateBefore.messages.length,
        });
        output.messages.push(...summarizeMessages(stateBefore.messages, "state-before", limitArg));
      }

      const stateActions = [
        ["ensureConversationAvailable", appState?.messaging?.ensureConversationAvailable],
        ["fetchConversation", appState?.messaging?.fetchConversation],
        ["hydrateConversationWithMessages", appState?.messaging?.hydrateConversationWithMessages],
        ["paginateMessages", appState?.messaging?.paginateMessages],
        ["syncServerConversation", appState?.messaging?.syncServerConversation],
      ];
      output.actionTypes = Object.fromEntries(stateActions.map(([name, fn]) => [name, typeof fn]));
      if (!messagingClient) {
        output.attempts.push({
          name: "state-actions-skipped",
          ok: false,
          error: "messaging_client_unavailable",
        });
      }
      for (const [name, fn] of messagingClient ? stateActions : []) {
        if (typeof fn !== "function") {
          continue;
        }
        for (const args of [
          [chatIDArg],
          [conversationRef],
          [conversationRef, Number(limitArg || 20)],
          [{ conversationId: conversationRef, id: chatIDArg }],
        ]) {
          try {
            const result = await Promise.race([
              fn.apply(appState.messaging, args),
              new Promise((_, reject) => setTimeout(() => reject(new Error("state_action_timeout")), 15000)),
            ]);
            const refreshed = resolveAppState(webpackRequire).appState || appState;
            const fromState = conversationMessagesFromState(refreshed, chatIDArg);
            const resultMessages = Array.isArray(result?.messages) ? result.messages : [];
            const messages = resultMessages.length > 0 ? resultMessages : fromState.messages;
            output.attempts.push({
              name,
              ok: true,
              args: args.map((arg) => typeof arg === "string" ? "string" : Object.keys(arg || {}).slice(0, 8).join(",")),
              resultKeys: Object.keys(result || {}).slice(0, 40),
              stateConversationKeys: Object.keys(fromState.conversation || {}).slice(0, 80),
              messageCount: messages.length,
            });
            if (messages.length > 0) {
              output.messages.push(...summarizeMessages(messages, name, limitArg));
              output.ok = true;
              return output;
            }
          } catch (error) {
            output.attempts.push({
              name,
              ok: false,
              args: args.map((arg) => typeof arg === "string" ? "string" : Object.keys(arg || {}).slice(0, 8).join(",")),
              error: String(error).slice(0, 300),
            });
          }
        }
      }

      output.ok = Boolean(messagingClient && conversationRef?.id && messaging?.uk);
      if (!output.ok) {
        output.feedKeysSample = Object.keys(appState?.messaging?.feed || {}).slice(0, 20);
        return output;
      }

      const attempts = [
        ["uk", () => messaging.uk(messagingClient, conversationRef)],
      ];
      if (typeof messaging.Gq === "function") {
        attempts.push(["Gq", () => messaging.Gq(messagingClient, conversationRef, undefined)]);
      }
      for (const [name, fn] of attempts) {
        try {
          const result = await Promise.race([
            fn(),
            new Promise((_, reject) => setTimeout(() => reject(new Error("conversation_debug_timeout")), 25000)),
          ]);
          const messages = Array.isArray(result?.messages) ? result.messages : [];
          output.attempts.push({
            name,
            ok: true,
            resultKeys: Object.keys(result || {}).slice(0, 40),
            messageCount: messages.length,
          });
          for (const message of messages.slice(-Number(limitArg || 20))) {
            output.messages.push(...summarizeMessages([message], name, limitArg));
          }
        } catch (error) {
          output.attempts.push({ name, ok: false, error: String(error).slice(0, 300) });
        }
      }
        return output;
      }, { chatIDArg: id, limitArg: Number(limit) || 20 });
    }

    const seenFrames = new Set();
    const results = [];
    for (const targetFrame of [currentPage.mainFrame(), ...currentPage.frames()]) {
      if (!targetFrame || seenFrames.has(targetFrame)) {
        continue;
      }
      seenFrames.add(targetFrame);
      try {
        const result = await probeFrame(targetFrame);
        result.frameURL = targetFrame.url();
        results.push(result);
        if (result.ok || result.messages?.length) {
          result.frameResults = results.map(summarizeFrameResult);
          return result;
        }
      } catch (error) {
        results.push({ ok: false, frameURL: targetFrame.url(), error: String(error).slice(0, 300) });
      }
    }
    return {
      ok: false,
      error: "conversation messages unavailable in all frames",
        frameResults: results.map(summarizeFrameResult),
      };
  }, { priority: 0 });
}


async function getBrowserE2EESummary() {
  return withLock(async () => {
    const { page: currentPage } = await ensureSession();
    if (!currentPage || currentPage.isClosed()) {
      return { ok: false, error: "no active page" };
    }
    return await currentPage.evaluate(async () => {
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
            chunk.push([[`codex-e2ee-summary-${Date.now()}`], {}, (req) => {
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

      function toBase64(value) {
        if (!value) {
          return "";
        }
        const bytes = ArrayBuffer.isView(value)
          ? new Uint8Array(value.buffer, value.byteOffset, value.byteLength)
          : Array.isArray(value)
            ? Uint8Array.from(value)
            : value instanceof ArrayBuffer
              ? new Uint8Array(value)
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

      async function tryGet(label, fn) {
        try {
          const value = await fn();
          return { label, ok: true, value };
        } catch (error) {
          return { label, ok: false, error: String(error).slice(0, 200) };
        }
      }

      function summarizeIdentity(identity) {
        if (!identity) {
          return undefined;
        }
        return {
          version: identity.version,
          identityKeyId: toBase64(identity.identityKeyId?.data || identity.identityKeyId),
          cleartextPublicKey: toBase64(identity.cleartextPublicKey),
          cleartextPrivateKey: toBase64(identity.cleartextPrivateKey),
        };
      }

      function safeRequire(webpackRequire, moduleId) {
        try {
          return webpackRequire?.(moduleId);
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

      async function resolveWasmModule(webpackRequire, appState) {
        const wasmExports = safeRequire(webpackRequire, 54897);
        try {
          const existing = wasmExports?.gZ?.(appState);
          if (existing) {
            return existing;
          }
        } catch {
        }
        try {
          const wasmFactory = safeRequire(webpackRequire, 51867);
          const params = wasmExports?.U7?.(appState);
          return await wasmFactory?.W?.(params);
        } catch {
          return undefined;
        }
      }

      const webpackRequire = findWebpackRequire();
      if (!webpackRequire) {
        return { ok: false, error: "webpack modules not found", url: location.href };
      }
      const keyStores = safeRequire(webpackRequire, 38369);
      const storage = safeRequire(webpackRequire, 36549);
      const appState = resolveAppState(webpackRequire);
      const wasmModule = await resolveWasmModule(webpackRequire, appState);
      const keyManager = wasmModule?.e2ee_E2EEKeyManager;
      const keyManagerPrototype = keyManager ? Object.getPrototypeOf(keyManager) : undefined;
      const output = { ok: true, url: location.href, exportKeys: Object.keys(keyStores || {}), checks: [] };
      output.keyManager = {
        present: Boolean(keyManager),
        argCount: keyManager?.argCount,
        ownKeys: keyManager ? Object.keys(keyManager).slice(0, 80) : [],
        protoKeys: keyManagerPrototype ? Object.getOwnPropertyNames(keyManagerPrototype).slice(0, 120) : [],
        methods: keyManager ? Object.fromEntries(Object.keys(keyManager)
          .filter((key) => typeof keyManager[key] === "function")
          .map((key) => {
            let noArgError = "";
            let expectedArgError = "";
            try {
              if (key !== "generateKeyInitializationRequest") {
                keyManager[key]();
              }
            } catch (error) {
              noArgError = String(error).slice(0, 220);
            }
            const expectedMatch = noArgError.match(/expected (\d+) args/);
            if (expectedMatch) {
              try {
                keyManager[key](...Array(Number(expectedMatch[1])).fill(undefined));
              } catch (error) {
                expectedArgError = String(error).slice(0, 260);
              }
            }
            return [key, { length: keyManager[key].length, noArgError, expectedArgError, source: String(keyManager[key]).slice(0, 160) }];
          })) : {},
      };
      const sharedLocal = await tryGet("identity.shared.local", async () => keyStores.XX(storage.Lg()).sharedItem().get());
      const rootSession = await tryGet("root.shared.session", async () => keyStores.Qs().sharedItem().get());
      const tempLocal = await tryGet("temp.shared.local", async () => keyStores.Ep(storage.Lg()).sharedItem().get());
      const tempSession = await tryGet("temp.shared.session", async () => keyStores.Ep(storage.W9()).sharedItem().get());
      const tempIDB = await tryGet("temp.shared.idb", async () => keyStores.Ep(storage.gd()).sharedItem().get());
      for (const check of [sharedLocal, rootSession, tempLocal, tempSession, tempIDB]) {
        if (!check.ok) {
          output.checks.push(check);
          continue;
        }
        const value = check.value;
        if (Array.isArray(value)) {
          output.checks.push({
            label: check.label,
            ok: true,
            count: value.length,
            items: value.slice(0, 5).map((item) => ({
              data: toBase64(item.data),
              dataLength: item.data?.byteLength || item.data?.length || 0,
              lastUpdatedTimestamp: item.lastUpdatedTimestamp?.toISOString?.() || "",
            })),
          });
        } else {
          output.checks.push({
            label: check.label,
            ok: true,
            rwk: toBase64(value?.rwk?.data || value?.rwk),
            keyIdentifier: toBase64(value?.keyIdentifier?.data || value?.keyIdentifier),
            identity: summarizeIdentity(value?.identity || value),
          });
        }
      }
      return output;
    });
  }, { priority: 0 });
}


async function deriveBrowserE2EESharedSecret(senderPublicKeyBase64, senderVersion) {
  const pub = String(senderPublicKeyBase64 || "").trim();
  if (!pub) {
    return { ok: false, error: "sender public key is required" };
  }
  return withLock(async () => {
    const { page: currentPage } = await ensureSession();
    if (!currentPage || currentPage.isClosed()) {
      return { ok: false, error: "no active page" };
    }
    return await currentPage.evaluate(async ({ pub, version }) => {
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
            chunk.push([[`codex-e2ee-secret-${Date.now()}`], {}, (req) => {
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

      function fromBase64(value) {
        const binary = atob(value);
        const bytes = new Uint8Array(binary.length);
        for (let index = 0; index < binary.length; index += 1) {
          bytes[index] = binary.charCodeAt(index);
        }
        return bytes;
      }

      function summarize(value, depth = 0, seen = new Set()) {
        if (value == null) {
          return { type: String(value) };
        }
        if (ArrayBuffer.isView(value)) {
          return {
            type: value.constructor?.name || "TypedArray",
            length: value.byteLength,
            base64: btoa(String.fromCharCode(...new Uint8Array(value.buffer, value.byteOffset, value.byteLength))),
          };
        }
        if (value instanceof ArrayBuffer) {
          const bytes = new Uint8Array(value);
          return { type: "ArrayBuffer", length: bytes.byteLength, base64: btoa(String.fromCharCode(...bytes)) };
        }
        if (typeof value !== "object" || seen.has(value) || depth > 2) {
          return { type: typeof value, value: typeof value === "string" || typeof value === "number" || typeof value === "boolean" ? value : undefined };
        }
        seen.add(value);
        const output = { type: value.constructor?.name || "object", keys: Object.keys(value).slice(0, 40), fields: {} };
        for (const key of output.keys) {
          try {
            output.fields[key] = summarize(value[key], depth + 1, seen);
          } catch (error) {
            output.fields[key] = { error: String(error).slice(0, 120) };
          }
        }
        return output;
      }

      function safeRequire(webpackRequire, moduleId) {
        try {
          return webpackRequire?.(moduleId);
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

      async function resolveWasmModule(webpackRequire, appState) {
        const wasmExports = safeRequire(webpackRequire, 54897);
        try {
          const existing = wasmExports?.gZ?.(appState);
          if (existing) {
            return existing;
          }
        } catch {
        }
        try {
          const wasmFactory = safeRequire(webpackRequire, 51867);
          const params = wasmExports?.U7?.(appState);
          return await wasmFactory?.W?.(params);
        } catch {
          return undefined;
        }
      }

      const webpackRequire = findWebpackRequire();
      const appState = resolveAppState(webpackRequire);
      const keyManager = (await resolveWasmModule(webpackRequire, appState))?.e2ee_E2EEKeyManager;
      if (!keyManager?.createSharedSecretKeys) {
        return { ok: false, error: "E2EE key manager unavailable" };
      }
      const senderPublicKey = fromBase64(pub);
      const attempts = [];
      for (const args of [
        [senderPublicKey, version],
        [version, senderPublicKey],
        [senderPublicKey, Number(version)],
        [senderPublicKey.buffer, version],
      ]) {
        try {
          const result = keyManager.createSharedSecretKeys(...args);
          attempts.push({ ok: true, args: args.map((arg) => ArrayBuffer.isView(arg) ? `${arg.constructor.name}:${arg.byteLength}` : typeof arg), result: summarize(result) });
        } catch (error) {
          attempts.push({ ok: false, args: args.map((arg) => ArrayBuffer.isView(arg) ? `${arg.constructor.name}:${arg.byteLength}` : typeof arg), error: String(error).slice(0, 260) });
        }
      }
      return { ok: true, attempts };
    }, { pub, version: Number(senderVersion) || 0 });
  }, { priority: 0 });
}


  return {
    getBrowserStorageSummary,
    searchBrowserBundle,
    getBrowserBundleModule,
    getBrowserConversationMessages,
    getBrowserE2EESummary,
    deriveBrowserE2EESharedSecret,
  };
}
