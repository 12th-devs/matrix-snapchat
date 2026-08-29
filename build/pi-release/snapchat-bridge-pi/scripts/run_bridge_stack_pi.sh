#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
config_path="${SNAPCHAT_BRIDGE_CONFIG:-${HOME}/.local/share/bbctl/bridges/sh-snapchat/config.yaml}"
bridge_binary="${SNAPCHAT_BRIDGE_BINARY:-${repo_dir}/bin/mautrix-snapchat-bridgev2-linux-arm64}"

if [[ ! -f "${config_path}" ]]; then
    echo "Beeper bridge config not found at ${config_path}" >&2
    echo "Set SNAPCHAT_BRIDGE_CONFIG=/path/to/config.yaml if it lives somewhere else." >&2
    exit 1
fi
if [[ ! -x "${bridge_binary}" ]]; then
    echo "Bridge binary not found or not executable at ${bridge_binary}" >&2
    exit 1
fi
if ! command -v node >/dev/null 2>&1; then
    echo "Node.js is required for the Snapchat connector. Install Node 22+ and rerun this script." >&2
    exit 1
fi
node_major="$(node -p 'Number(process.versions.node.split(".")[0])')"
if [[ "${node_major}" -lt 22 ]]; then
    echo "Node.js 22+ is required; found $(node -v)." >&2
    exit 1
fi

python3 - "${config_path}" <<'PY'
import re
import sys
from pathlib import Path

path = Path(sys.argv[1])
text = path.read_text()
replacements = {
    "poll_interval_seconds": "2",
    "message_fetch_limit": "20",
    "api_mode": "api_only",
    "dom_fallback_enabled": "false",
    "auto_fetch_messages": "true",
}
for key, value in replacements.items():
    pattern = rf"^(\s*{re.escape(key)}\s*:\s*).*$"
    if re.search(pattern, text, flags=re.MULTILINE):
        text = re.sub(pattern, rf"\g<1>{value}", text, flags=re.MULTILINE)
path.write_text(text)
PY

runtime_secret="$(python3 -c 'import secrets; print(secrets.token_urlsafe(48))')"
export SNAPCHAT_BRIDGE_CONFIG="${config_path}"
export SNAPCHAT_SHARED_SECRET="${runtime_secret}"
export SNAPCHAT_CONNECTOR_SHARED_SECRET="${runtime_secret}"
export SNAPCHAT_CONNECTOR_BASE_URL="${SNAPCHAT_CONNECTOR_BASE_URL:-http://127.0.0.1:3101}"
export PORT="${PORT:-3101}"
export SNAPCHAT_PROFILE_DIR="${SNAPCHAT_PROFILE_DIR:-${repo_dir}/data/snapchat-profile}"
export SNAPCHAT_TRACE_DIR="${SNAPCHAT_TRACE_DIR:-${repo_dir}/data/debug}"
export SNAPCHAT_HEADLESS="${SNAPCHAT_HEADLESS:-true}"
export SNAPCHAT_SAFE_NO_OPEN="${SNAPCHAT_SAFE_NO_OPEN:-1}"
export SNAPCHAT_READ_RECEIPTS_ENABLED="${SNAPCHAT_READ_RECEIPTS_ENABLED:-false}"
export SNAPCHAT_SNAP_MEDIA_ENABLED="${SNAPCHAT_SNAP_MEDIA_ENABLED:-false}"
export SNAPCHAT_SNAP_MEDIA_ON_READ="${SNAPCHAT_SNAP_MEDIA_ON_READ:-false}"
export SNAPCHAT_SEND_MEDIA_ENABLED="${SNAPCHAT_SEND_MEDIA_ENABLED:-false}"

if [[ -z "${SNAPCHAT_BROWSER_EXECUTABLE_PATH:-}" ]]; then
    for candidate in /usr/bin/chromium /usr/bin/chromium-browser /usr/bin/google-chrome /usr/bin/google-chrome-stable; do
        if [[ -x "${candidate}" ]]; then
            export SNAPCHAT_BROWSER_EXECUTABLE_PATH="${candidate}"
            break
        fi
    done
fi
if [[ -z "${SNAPCHAT_BROWSER_EXECUTABLE_PATH:-}" ]]; then
    echo "Chromium/Chrome was not found. Install chromium, or set SNAPCHAT_BROWSER_EXECUTABLE_PATH." >&2
    exit 1
fi

mkdir -p "${repo_dir}/data/logs" "${SNAPCHAT_PROFILE_DIR}" "${SNAPCHAT_TRACE_DIR}"
rm -f \
    "${SNAPCHAT_PROFILE_DIR}/SingletonCookie" \
    "${SNAPCHAT_PROFILE_DIR}/SingletonLock" \
    "${SNAPCHAT_PROFILE_DIR}/SingletonSocket"

if [[ ! -d "${repo_dir}/connector/node_modules" ]]; then
    (cd "${repo_dir}/connector" && npm install --omit=dev)
fi
if [[ -f "${repo_dir}/.codex-build/snapcap-native/package.json" && ! -d "${repo_dir}/.codex-build/snapcap-native/node_modules" ]]; then
    (cd "${repo_dir}/.codex-build/snapcap-native" && npm install --omit=dev)
fi

connector_pid=""
cleanup() {
    if [[ -n "${connector_pid}" ]] && kill -0 "${connector_pid}" 2>/dev/null; then
        kill "${connector_pid}" 2>/dev/null || true
        wait "${connector_pid}" 2>/dev/null || true
    fi
}
trap cleanup EXIT INT TERM

(cd "${repo_dir}/connector" && node src/index.mjs) &
connector_pid=$!

python3 - <<'PY'
import json
import os
import sys
import time
import urllib.error
import urllib.request

base = os.environ["SNAPCHAT_CONNECTOR_BASE_URL"].rstrip("/")
secret = os.environ["SNAPCHAT_SHARED_SECRET"]
headers = {"X-Bridge-Secret": secret}

for _ in range(45):
    try:
        req = urllib.request.Request(base + "/healthz", headers=headers)
        with urllib.request.urlopen(req, timeout=3) as resp:
            if resp.status == 200:
                req = urllib.request.Request(base + "/session/status", headers=headers)
                with urllib.request.urlopen(req, timeout=10) as status_resp:
                    print("connector_status=" + json.dumps(json.load(status_resp)))
                sys.exit(0)
    except Exception:
        time.sleep(1)
print("Snapchat connector did not become healthy", file=sys.stderr)
sys.exit(1)
PY

if [[ -x "${repo_dir}/scripts/ensure_beeper_snapchat_registration_wsl.sh" ]]; then
    "${repo_dir}/scripts/ensure_beeper_snapchat_registration_wsl.sh" || true
fi

exec "${bridge_binary}" -c "${config_path}"
