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

      const webpackRequire = findWebpackRequire();
      if (!webpackRequire) {
        return { ok: false, error: "webpack modules not found", url: location.href };
      }
      const keyStores = webpackRequire(38369);
      const storage = webpackRequire(36549);
      const appState = webpackRequire(97003)?.M?.getState?.();
      const wasmModule = webpackRequire(54897)?.gZ?.(appState);
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

      const webpackRequire = findWebpackRequire();
      const appState = webpackRequire?.(97003)?.M?.getState?.();
      const keyManager = webpackRequire?.(54897)?.gZ?.(appState)?.e2ee_E2EEKeyManager;
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
    getBrowserE2EESummary,
    deriveBrowserE2EESharedSecret,
  };
}
