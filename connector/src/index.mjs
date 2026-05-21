import { createApp } from "./routes.mjs";
import * as bridge from "./snapchat.mjs";

const port = Number(process.env.PORT || 3101);
const safeNoOpen = process.env.SNAPCHAT_SAFE_NO_OPEN !== "0";

const app = createApp({
  bridge,
  jsonLimit: process.env.SNAPCHAT_JSON_LIMIT || "50mb",
  safeNoOpen,
  sharedSecret: process.env.SNAPCHAT_SHARED_SECRET || "",
});

process.on("unhandledRejection", (error) => {
  console.error("unhandled rejection", error);
});

app.listen(port, () => {
  console.log(`snapchat connector listening on http://127.0.0.1:${port} safeNoOpen=${safeNoOpen}`);
});
