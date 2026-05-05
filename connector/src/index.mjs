import express from "express";
import {
  createSession,
  getDiagnostics,
  getChats,
  getMessages,
  getStatus,
  sendMessage,
} from "./snapchat.mjs";

const app = express();
const port = Number(process.env.PORT || 3101);
const sharedSecret = process.env.SNAPCHAT_SHARED_SECRET || "";
const acceptedSecrets = new Set([sharedSecret].filter(Boolean));

app.use(express.json());
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
  res.json({ ok: true, status: await getStatus() });
}));

app.post("/session/start", asyncRoute(async (_req, res) => {
  const status = await createSession();
  res.status(202).json(status);
}));

app.get("/session/status", asyncRoute(async (_req, res) => {
  res.json(await getStatus());
}));

app.get("/debug/diagnostics", asyncRoute(async (_req, res) => {
  res.json(await getDiagnostics());
}));

app.get("/chats", asyncRoute(async (_req, res) => {
  res.json(await getChats());
}));

app.get("/messages", asyncRoute(async (req, res) => {
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
  const { chatId, chatName, chatUrl, text } = req.body || {};
  if ((!chatId && !chatName && !chatUrl) || !text) {
    res.status(400).json({ error: "chatId, chatUrl, or chatName and text are required" });
    return;
  }

  const result = await sendMessage(chatId || "", chatName || "", chatUrl || "", text);
  res.status(202).json(result);
}));

process.on("unhandledRejection", (error) => {
  console.error("unhandled rejection", error);
});

app.listen(port, () => {
  console.log(`snapchat connector listening on http://127.0.0.1:${port}`);
});
