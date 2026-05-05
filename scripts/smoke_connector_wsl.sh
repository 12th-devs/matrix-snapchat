#!/usr/bin/env bash
set -euo pipefail

cd "${HOME}/Codex/connector"

export PORT="${PORT:-3101}"
export SNAPCHAT_SHARED_SECRET="${SNAPCHAT_SHARED_SECRET:-1225}"
export SNAPCHAT_PROFILE_DIR="${SNAPCHAT_PROFILE_DIR:-${HOME}/Codex/data/snapchat-profile}"
export SNAPCHAT_TRACE_DIR="${SNAPCHAT_TRACE_DIR:-${HOME}/Codex/data/debug}"
export SNAPCHAT_HEADLESS="${SNAPCHAT_HEADLESS:-false}"

node src/index.mjs > /tmp/codex-snapchat-smoke.log 2>&1 &
pid=$!
cleanup() {
  kill "${pid}" >/dev/null 2>&1 || true
  wait "${pid}" >/dev/null 2>&1 || true
}
trap cleanup EXIT

sleep 5

python3 - <<'PY'
import json
import sys
import urllib.parse
import urllib.request

secret = "1225"
base = "http://127.0.0.1:3101"

def fetch(path):
    req = urllib.request.Request(base + path, headers={"X-Bridge-Secret": secret})
    with urllib.request.urlopen(req, timeout=30) as resp:
        return json.loads(resp.read().decode())

status = fetch("/session/status")
print("STATUS", json.dumps(status))

chats = fetch("/chats")
print("CHATS", len(chats))
if chats:
    print("FIRST_CHAT", json.dumps(chats[0]))
    name = chats[0]["name"]
    quoted = urllib.parse.quote(name)
    messages = fetch(f"/messages?chatName={quoted}")
    print("MESSAGES", len(messages))
    if messages:
        print("FIRST_MESSAGE", json.dumps(messages[0]))
PY

echo "--- LOG ---"
cat /tmp/codex-snapchat-smoke.log
