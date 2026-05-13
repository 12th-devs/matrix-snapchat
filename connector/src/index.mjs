import express from "express";
import {
  createSession,
  getDiagnostics,
  getChats,
  getMessages,
  getAPIAuth,
  getStatus,
  sendMedia,
  sendMessage,
} from "./snapchat.mjs";

const app = express();
const port = Number(process.env.PORT || 3101);
const sharedSecret = process.env.SNAPCHAT_SHARED_SECRET || "";
const acceptedSecrets = new Set([sharedSecret].filter(Boolean));
const safeNoOpen = process.env.SNAPCHAT_SAFE_NO_OPEN !== "0";

app.use(express.json({ limit: process.env.SNAPCHAT_JSON_LIMIT || "50mb" }));
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

function asyncRoute(handler) {
  return async (req, res) => {
    try {
      await handler(req, res);
    } catch (error) {
      res.status(500).json({ error: String(error) });
    }
  };
}

app.get("/healthz", asyncRoute(async (_req, res) => {
  // Keep Docker healthchecks cheap so they don't contend with chat polling or sends.
  res.json({ ok: true });
}));

app.post("/session/start", asyncRoute(async (_req, res) => {
  const status = await createSession();
  res.status(202).json(status);
}));

app.get("/session/status", asyncRoute(async (_req, res) => {
  res.json(await getStatus());
}));

app.get("/session/api-auth", asyncRoute(async (_req, res) => {
  res.json(await getAPIAuth());
}));

app.get("/debug/diagnostics", asyncRoute(async (_req, res) => {
  res.json(await getDiagnostics());
}));

app.get("/chats", asyncRoute(async (_req, res) => {
  res.json(await getChats());
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

  res.json(await getMessages(chatId, chatName, chatUrl));
}));

app.post("/messages", asyncRoute(async (req, res) => {
  if (safeNoOpen) {
    res.status(423).json({ error: "disabled by SNAPCHAT_SAFE_NO_OPEN to avoid opening Snapchat chats/snaps" });
    return;
  }
  const { chatId, chatName, chatUrl, text } = req.body || {};
  if ((!chatId && !chatName && !chatUrl) || !text) {
    res.status(400).json({ error: "chatId, chatUrl, or chatName and text are required" });
    return;
  }

  const result = await sendMessage(chatId || "", chatName || "", chatUrl || "", text);
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

  const result = await sendMedia(
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

process.on("unhandledRejection", (error) => {
  console.error("unhandled rejection", error);
});

app.listen(port, () => {
  console.log(`snapchat connector listening on http://127.0.0.1:${port} safeNoOpen=${safeNoOpen}`);
});
