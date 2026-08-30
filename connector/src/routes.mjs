import express from "express";

function asyncRoute(handler) {
  return async (req, res) => {
    try {
      await handler(req, res);
    } catch (error) {
      res.status(500).json({ error: String(error) });
    }
  };
}

export function createApp({
  bridge,
  jsonLimit = "50mb",
  sharedSecret = "",
  safeNoOpen = true,
} = {}) {
  if (!bridge) {
    throw new Error("bridge implementation is required");
  }

  const app = express();
  const acceptedSecrets = new Set([sharedSecret].filter(Boolean));

  app.use(express.json({ limit: jsonLimit }));
  app.use((req, res, next) => {
    const startedAt = Date.now();
    res.on("finish", () => {
      console.log(`${req.method} ${req.originalUrl} -> ${res.statusCode} (${Date.now() - startedAt}ms)`);
    });
    next();
  });
  app.use((req, res, next) => {
    if (!sharedSecret) {
      next();
      return;
    }

    if (!acceptedSecrets.has(req.header("X-Bridge-Secret") || "")) {
      res.status(401).json({ error: "invalid bridge secret" });
      return;
    }

    next();
  });

  app.get("/healthz", asyncRoute(async (_req, res) => {
    // Keep Docker healthchecks cheap so they don't contend with chat polling or sends.
    res.json({ ok: true });
  }));

  app.post("/session/start", asyncRoute(async (_req, res) => {
    const status = await bridge.createSession();
    res.status(202).json(status);
  }));

  app.get("/session/status", asyncRoute(async (_req, res) => {
    res.json(await bridge.getStatus());
  }));

  app.get("/session/api-auth", asyncRoute(async (_req, res) => {
    res.json(await bridge.getAPIAuth());
  }));

  app.get("/debug/diagnostics", asyncRoute(async (_req, res) => {
    res.json(await bridge.getDiagnostics());
  }));

  app.get("/debug/storage", asyncRoute(async (_req, res) => {
    res.json(await bridge.getBrowserStorageSummary());
  }));

  app.get("/debug/e2ee", asyncRoute(async (_req, res) => {
    res.json(await bridge.getBrowserE2EESummary());
  }));

  app.get("/debug/bundle-search", asyncRoute(async (req, res) => {
    res.json(await bridge.searchBrowserBundle(String(req.query.q || "")));
  }));

  app.get("/debug/bundle/:moduleID", asyncRoute(async (req, res) => {
    res.json(await bridge.getBrowserBundleModule(String(req.params.moduleID || "")));
  }));

  app.get("/debug/conversation-messages", asyncRoute(async (req, res) => {
    res.json(await bridge.getBrowserConversationMessages(
      String(req.query.chatId || ""),
      Number(req.query.limit || 20),
    ));
  }));

  app.post("/session/eel-decrypt", asyncRoute(async (req, res) => {
    const result = await bridge.decryptEELMessage(req.body || {});
    res.status(result.ok ? 200 : 422).json(result);
  }));

  app.get("/chats", asyncRoute(async (_req, res) => {
    res.json(await bridge.getChats());
  }));

  app.get("/messages", asyncRoute(async (req, res) => {
    if (safeNoOpen) {
      res.status(423).json({ error: "disabled by SNAPCHAT_SAFE_NO_OPEN to avoid opening Snapchat chats/snaps" });
      return;
    }
    const chatName = String(req.query.chatName || "");
    const chatId = String(req.query.chatId || "");
    const chatUrl = String(req.query.chatUrl || "");
    if (!chatName && !chatId && !chatUrl) {
      res.status(400).json({ error: "chatId, chatUrl, or chatName is required" });
      return;
    }

    res.json(await bridge.getMessages(chatId, chatName, chatUrl));
  }));

  app.post("/messages", asyncRoute(async (req, res) => {
    const { chatId, chatName, chatUrl, text } = req.body || {};
    if ((!chatId && !chatName && !chatUrl) || !text) {
      res.status(400).json({ error: "chatId, chatUrl, or chatName and text are required" });
      return;
    }

    const result = await bridge.sendMessage(chatId || "", chatName || "", chatUrl || "", text);
    res.status(202).json(result);
  }));

  app.post("/media", asyncRoute(async (req, res) => {
    if (safeNoOpen) {
      res.status(423).json({ error: "disabled by SNAPCHAT_SAFE_NO_OPEN to avoid opening Snapchat chats/snaps" });
      return;
    }
    const { chatId, chatName, chatUrl, fileName, mimeType, data, caption } = req.body || {};
    if ((!chatId && !chatName && !chatUrl) || !data) {
      res.status(400).json({ error: "chatId, chatUrl, or chatName and media data are required" });
      return;
    }

    const result = await bridge.sendMedia(
      chatId || "",
      chatName || "",
      chatUrl || "",
      fileName || "snap-media.bin",
      mimeType || "application/octet-stream",
      data,
      caption || "",
    );
    res.status(202).json(result);
  }));

  app.post("/typing", asyncRoute(async (req, res) => {
    const { chatId, typing, durationMs } = req.body || {};
    if (!chatId || typeof typing !== "boolean") {
      res.status(400).json({ ok: false, error: "chatId and typing are required" });
      return;
    }
    const result = await bridge.setTyping(String(chatId), Boolean(typing), Number(durationMs || 1500));
    res.status(result.ok ? 202 : 501).json(result);
  }));

  return app;
}
