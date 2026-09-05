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


// debugMediaResolve replays a resolveContentObjects request from inside the
// authenticated Snapchat Web page so the browser's own cookies, headers, and
// transport are used. Responses are summarized structurally only: field
// numbers, wire types, lengths, and HTTPS hostnames. No URL, token, or cookie
// values are returned.
async function debugMediaResolve(payload) {
  const descriptorHex = String(payload?.descriptorHex || "").replace(/[^0-9a-fA-F]/g, "");
  const token = String(payload?.token || "");
  const snapClientUserAgent = String(payload?.snapClientUserAgent || "");
  if (!descriptorHex || descriptorHex.length < 8 || descriptorHex.length % 2 !== 0) {
    return { ok: false, error: "descriptor hex required" };
  }
  return withLock(async () => {
    const { page: currentPage } = await ensureSession();
    if (!currentPage || currentPage.isClosed()) {
      return { ok: false, error: "no active page" };
    }
    return await currentPage.evaluate(async (probe) => {
      const endpoint = "https://web.snapchat.com/snapchat.content.v2.MediaDeliveryService/resolveContentObjects";
      const descriptor = new Uint8Array(probe.descriptorHex.match(/.{2}/g).map((h) => parseInt(h, 16)));

      function varint(value) {
        const out = [];
        let n = value;
        do {
          let byte = n & 127;
          n >>>= 7;
          if (n) byte |= 128;
          out.push(byte);
        } while (n);
        return out;
      }
      function bytesField(fieldNumber, payload) {
        const head = varint((fieldNumber << 3) | 2);
        return new Uint8Array([...head, ...varint(payload.length), ...payload]);
      }
      function grpcFrame(message) {
        const out = new Uint8Array(5 + message.length);
        new DataView(out.buffer).setUint32(1, message.length);
        out.set(message, 5);
        return out;
      }
      function parseFrames(buffer) {
        const view = new Uint8Array(buffer);
        const frames = [];
        let offset = 0;
        while (offset < view.length) {
          if (view.length - offset < 5) {
            frames.push({ truncated: true, remaining: view.length - offset });
            break;
          }
          const flags = view[offset];
          const size = new DataView(buffer, offset, 5).getUint32(1);
          if (size > view.length - offset - 5) {
            frames.push({ flags, size, truncated: true });
            break;
          }
          frames.push({ flags, size });
          offset += 5 + size;
        }
        return frames;
      }
      function safeWalk(bytes, depth) {
        // Shallow protobuf walk: field numbers, wire types, lengths, and
        // HTTPS URL hostnames only.
        const out = [];
        let offset = 0;
        while (offset < bytes.length && out.length < 24) {
          let tag = 0;
          let shift = 0;
          for (;;) {
            if (offset >= bytes.length) return out;
            const b = bytes[offset++];
            tag |= (b & 127) << shift;
            shift += 7;
            if (b < 128) break;
          }
          const fieldNumber = tag >>> 3;
          const wireType = tag & 7;
          if (fieldNumber === 0 || fieldNumber > 100) return out;
          if (wireType === 0) {
            let value = 0;
            shift = 0;
            for (;;) {
              if (offset >= bytes.length) return out;
              const b = bytes[offset++];
              value |= (b & 127) << shift;
              shift += 7;
              if (b < 128) break;
            }
            out.push(`f${fieldNumber}:varint=${value}`);
          } else if (wireType === 2) {
            let length = 0;
            shift = 0;
            for (;;) {
              if (offset >= bytes.length) return out;
              const b = bytes[offset++];
              length |= (b & 127) << shift;
              shift += 7;
              if (b < 128) break;
            }
            if (offset + length > bytes.length) {
              out.push(`f${fieldNumber}:bytes_len=${length}:truncated`);
              return out;
            }
            const value = bytes.subarray(offset, offset + length);
            offset += length;
            let entry = `f${fieldNumber}:bytes_len=${length}`;
            const text = new TextDecoder("utf-8", { fatal: false }).decode(value);
            if (/^https:\/\//.test(text)) {
              try {
                entry += ` https_url_host=${new URL(text).hostname}`;
              } catch {}
            } else if (depth < 4 && value.length > 1) {
              const nested = safeWalk(value, depth + 1);
              if (nested.length) {
                out.push(entry);
                out.push(...nested.map((line) => ` f${fieldNumber}.${line}`));
                continue;
              }
            }
            out.push(entry);
          } else {
            out.push(`f${fieldNumber}:wire=${wireType}`);
            return out;
          }
        }
        return out;
      }

      async function probeVariant(name, requestBytes, extraHeaders) {
        try {
          const response = await fetch(endpoint, {
            method: "POST",
            credentials: "include",
            headers: {
              "accept": "*/*",
              "content-type": "application/grpc-web+proto",
              "x-grpc-web": "1",
              "x-user-agent": "grpc-web-javascript/0.1",
              ...Object.fromEntries(Object.entries(extraHeaders || {}).filter(([, v]) => v)),
            },
            body: requestBytes,
          });
          const buffer = await response.arrayBuffer();
          const frames = parseFrames(buffer);
          const view = new Uint8Array(buffer);
          let payloadWalk = [];
          if (frames.length && frames[0].flags === 0 && !frames[0].truncated) {
            const payload = view.subarray(5, 5 + frames[0].size);
            payloadWalk = safeWalk(payload, 0);
          }
          return {
            name,
            status: response.status,
            contentType: response.headers.get("content-type") || "",
            contentLength: response.headers.get("content-length"),
            grpcStatus: response.headers.get("grpc-status") || "",
            grpcMessage: response.headers.get("grpc-message") || "",
            respBytes: buffer.byteLength,
            frames,
            payloadWalk,
          };
        } catch (error) {
          return { name, error: String(error).slice(0, 200) };
        }
      }

      // Confirmed request shape only:
      // requests[0].reference.v2ContentObject = raw descriptor bytes
      // (ResolveContentObjectsRequest field 1 -> request item field 3).
      const requestBytes = grpcFrame(bytesField(1, bytesField(3, descriptor)));

      // Resolve the exact auth pair the app's own MediaDeliveryService fetch
      // path uses: 96789 wires 37308.W(34010.s); 34010.s is default-authed-fetch
      // whose request interceptor Tl(85997.c) sets authorization from the app
      // auth state and setSnapchatWebUserAgent uses 6755.Ay(). Only
      // lengths/fingerprints/equality are reported, never token values.
      function findWebpackRequire() {
        for (const key of Object.keys(globalThis)) {
          if (!/^webpackChunk/.test(key)) {
            continue;
          }
          const chunk = globalThis[key];
          if (!Array.isArray(chunk) || typeof chunk.push !== "function") {
            continue;
          }
          let req;
          try {
            chunk.push([[`codex-media-resolve-${Date.now()}`], {}, (r) => {
              req = r;
            }]);
          } catch {
            continue;
          }
          if (req?.m) {
            return req;
          }
        }
        return undefined;
      }

      async function fingerprint(value) {
        if (!value) return "";
        const digest = await crypto.subtle.digest("SHA-256", new TextEncoder().encode(value));
        return Array.from(new Uint8Array(digest)).slice(0, 6).map((b) => b.toString(16).padStart(2, "0")).join("");
      }

      const req = findWebpackRequire();
      let tokenB = "";
      let tokenBError = "";
      let uaB = "";
      let uaBError = "";
      if (req) {
        try {
          tokenB = (await req(85997).c()) || "";
        } catch (error) {
          tokenBError = String(error).slice(0, 140);
        }
        try {
          uaB = req(6755).Ay() || "";
        } catch (error) {
          uaBError = String(error).slice(0, 140);
        }
      } else {
        tokenBError = "webpack modules not found";
      }

      const variants = [];
      if (probe.token) {
        variants.push(await probeVariant("authA-captured", requestBytes, {
          authorization: `Bearer ${probe.token}`,
          ...(probe.snapClientUserAgent ? { "x-snap-client-user-agent": probe.snapClientUserAgent } : {}),
        }));
      }
      if (tokenB && tokenB !== probe.token) {
        variants.push(await probeVariant("authB-appstate", requestBytes, {
          authorization: `Bearer ${tokenB}`,
          ...(uaB ? { "x-snap-client-user-agent": uaB } : {}),
        }));
      }

      return {
        ok: true,
        pageOrigin: location.origin,
        pagePath: location.pathname,
        auth: {
          capturedPresent: Boolean(probe.token),
          capturedLength: probe.token ? probe.token.length : 0,
          capturedFingerprint: await fingerprint(probe.token),
          appStatePresent: Boolean(tokenB),
          appStateLength: tokenB ? tokenB.length : 0,
          appStateFingerprint: await fingerprint(tokenB),
          tokensEqual: Boolean(probe.token) && Boolean(tokenB) && probe.token === tokenB,
          appStateError: tokenBError,
          uaCapturedLength: probe.snapClientUserAgent ? probe.snapClientUserAgent.length : 0,
          uaCapturedFingerprint: await fingerprint(probe.snapClientUserAgent),
          uaAppStateLength: uaB ? uaB.length : 0,
          uaAppStateFingerprint: await fingerprint(uaB),
          uaEqual: Boolean(probe.snapClientUserAgent) && Boolean(uaB) && probe.snapClientUserAgent === uaB,
          uaAppStateError: uaBError,
        },
        variants,
      };
    }, { descriptorHex, token, snapClientUserAgent });
  }, { priority: 0 });
}

// debugMediaSignedDownload follows Snapchat Web's ordinary DownloadMedia path:
// 24977.hB(contentObject, metricsContext) -> signed URL -> browser fetch blob,
// with the ARROYO_MESSAGING 404 fallback through 24977.NA(contentObject).
// It returns only structural URL diagnostics and byte/container sniffing.
async function debugMediaSignedDownload(payload) {
  const descriptorHex = String(payload?.descriptorHex || "").replace(/[^0-9a-fA-F]/g, "");
  const metricsContext = String(payload?.metricsContext || "chat_media");
  if (!descriptorHex || descriptorHex.length < 8 || descriptorHex.length % 2 !== 0) {
    return { ok: false, error: "descriptor hex required" };
  }
  return withLock(async () => {
    const { page: currentPage } = await ensureSession();
    if (!currentPage || currentPage.isClosed()) {
      return { ok: false, error: "no active page" };
    }
    return await currentPage.evaluate(async ({ descriptorHex, metricsContext }) => {
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
            chunk.push([[`codex-media-download-${Date.now()}`], {}, (req) => {
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

      function contentObjectFromHex(hex) {
        return new Uint8Array(hex.match(/.{2}/g).map((pair) => Number.parseInt(pair, 16)));
      }

      function summarizeUrl(value) {
        const url = new URL(value);
        return {
          host: url.host,
          pathShape: url.pathname.replace(/[A-Za-z0-9_-]{12,}/g, ":id"),
          queryParamNames: Array.from(url.searchParams.keys()).sort(),
          uc: url.searchParams.get("uc") || "",
        };
      }

      function sniff(bytes, contentType) {
        const head = Array.from(bytes.slice(0, 16));
        const ascii = String.fromCharCode(...bytes.slice(0, 16)).replace(/[^\x20-\x7e]/g, ".");
        let container = "unknown";
        if (head[0] === 0xff && head[1] === 0xd8 && head[2] === 0xff) {
          container = "jpeg";
        } else if (head[0] === 0x89 && head[1] === 0x50 && head[2] === 0x4e && head[3] === 0x47) {
          container = "png";
        } else if (ascii.startsWith("GIF8")) {
          container = "gif";
        } else if (ascii.slice(4, 8) === "ftyp") {
          container = "mp4";
        } else if (head[0] === 0x52 && head[1] === 0x49 && head[2] === 0x46 && head[3] === 0x46) {
          container = "riff";
        }
        const clear = ["jpeg", "png", "gif", "mp4", "riff"].includes(container)
          || /^image\/|^video\/|^audio\//i.test(contentType || "");
        return { container, appearsClear: clear, headHex: head.map((b) => b.toString(16).padStart(2, "0")).join("") };
      }

      const webpackRequire = findWebpackRequire();
      if (!webpackRequire?.m) {
        return { ok: false, error: "webpack modules not found", url: location.href };
      }
      const resolver = webpackRequire(24977);
      if (typeof resolver?.hB !== "function" || typeof resolver?.NA !== "function") {
        return { ok: false, error: "media resolver exports missing", exportKeys: Object.keys(resolver || {}) };
      }

      const contentObject = contentObjectFromHex(descriptorHex);
      let signedUrl = "";
      let fallbackUsed = false;
      let first = {};
      try {
        signedUrl = await resolver.hB(contentObject, metricsContext);
        first = summarizeUrl(signedUrl);
      } catch (error) {
        return { ok: false, stage: "hB", errorName: error?.name || "", errorMessage: String(error?.message || error).slice(0, 180) };
      }

      const downloader = webpackRequire(76993);
      const requestHelper = webpackRequire(17196);
      async function fetchBytes(url) {
        if (typeof requestHelper?.u9 === "function") {
          const response = await requestHelper.u9(new Request(url), 60000, `media_download_for_${metricsContext}_codex_probe`);
          const buffer = await response.arrayBuffer();
          return {
            via: "17196.u9",
            bytes: new Uint8Array(buffer),
            status: response.status,
            ok: response.ok,
            contentType: response.headers.get("content-type") || "",
          };
        }
        if (typeof downloader?.e === "function") {
          return {
            via: "76993.e",
            bytes: new Uint8Array(await downloader.e(url, metricsContext)),
            status: 200,
            contentType: "",
          };
        }
        const response = await fetch(url, { credentials: "include" });
        const buffer = await response.arrayBuffer();
        return {
          via: "fetch",
          bytes: new Uint8Array(buffer),
          status: response.status,
          ok: response.ok,
          contentType: response.headers.get("content-type") || "",
        };
      }

      let downloaded;
      try {
        downloaded = await fetchBytes(signedUrl);
      } catch (error) {
        return {
          ok: false,
          stage: "fetch",
          contentObjectBytes: contentObject.byteLength,
          firstUrl: first,
          fallbackUsed,
          errorName: error?.name || "",
          errorMessage: String(error?.message || error).slice(0, 180),
        };
      }
      if ((downloaded.status === 404 || /status_404/.test(downloaded.errorMessage || "")) && first.uc === "4") {
        signedUrl = `${await resolver.NA(contentObject)}&forceRefresh=True`;
        fallbackUsed = true;
        try {
          downloaded = await fetchBytes(signedUrl);
        } catch (error) {
          return {
            ok: false,
            stage: "fallback_fetch",
            contentObjectBytes: contentObject.byteLength,
            firstUrl: first,
            finalUrl: summarizeUrl(signedUrl),
            fallbackUsed,
            errorName: error?.name || "",
            errorMessage: String(error?.message || error).slice(0, 180),
          };
        }
      }

      const finalUrl = summarizeUrl(signedUrl);
      const contentType = downloaded.contentType || "";
      const bytes = downloaded.bytes || new Uint8Array(0);
      return {
        ok: downloaded.ok ?? downloaded.status === 200,
        pageOrigin: location.origin,
        via: downloaded.via,
        contentObjectBytes: contentObject.byteLength,
        firstUrl: first,
        finalUrl,
        fallbackUsed,
        httpStatus: downloaded.status,
        contentType,
        byteCount: bytes.byteLength,
        sniff: sniff(bytes, contentType),
      };
    }, { descriptorHex, metricsContext });
  }, { priority: 0 });
}

  return {
    getBrowserStorageSummary,
    searchBrowserBundle,
    getBrowserBundleModule,
    getBrowserConversationMessages,
    getBrowserE2EESummary,
    deriveBrowserE2EESharedSecret,
    debugMediaResolve,
    debugMediaSignedDownload,
  };
}
