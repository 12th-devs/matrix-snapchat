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

runtime_secret="$(python3 - "${config_path}" <<'PY'
import re
import sys
from pathlib import Path

text = Path(sys.argv[1]).read_text()
network = re.search(r"(?ms)^network:\n(?P<body>(?:^[ \t]+.*\n?)*)", text)
body = network.group("body") if network else text
match = re.search(r"(?m)^[ \t]+shared_secret:[ \t]*['\"]?([^'\"\n#]+)", body)
print((match.group(1).strip() if match else ""))
PY
)"
if [[ -z "${runtime_secret}" ]]; then
    runtime_secret="$(python3 -c 'import secrets; print(secrets.token_urlsafe(48))')"
    python3 - "${config_path}" "${runtime_secret}" <<'PY'
import re
import sys
from pathlib import Path

path = Path(sys.argv[1])
secret = sys.argv[2]
text = path.read_text()
pattern = r"(?m)^(\s*shared_secret:\s*).*$"
if re.search(pattern, text):
    text = re.sub(pattern, rf"\g<1>\"{secret}\"", text, count=1)
else:
    text = re.sub(r"(?m)^network:\s*$", f"network:\n    shared_secret: \"{secret}\"", text, count=1)
path.write_text(text)
PY
fi
export SNAPCHAT_BRIDGE_CONFIG="${config_path}"
export SNAPCHAT_SHARED_SECRET="${runtime_secret}"
export SNAPCHAT_CONNECTOR_SHARED_SECRET="${runtime_secret}"
export SNAPCHAT_CONNECTOR_BASE_URL="${SNAPCHAT_CONNECTOR_BASE_URL:-http://127.0.0.1:3101}"
export PORT="${PORT:-3101}"
export SNAPCHAT_PROFILE_DIR="${SNAPCHAT_PROFILE_DIR:-${repo_dir}/data/snapchat-profile}"
export SNAPCHAT_TRACE_DIR="${SNAPCHAT_TRACE_DIR:-${repo_dir}/data/debug}"
export SNAPCHAT_HEADLESS="${SNAPCHAT_HEADLESS:-true}"
export DISPLAY="${DISPLAY:-:99}"
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
display_pid=""
window_manager_pid=""
vnc_pid=""
novnc_pid=""
cleanup() {
    if [[ -n "${connector_pid}" ]] && kill -0 "${connector_pid}" 2>/dev/null; then
        kill "${connector_pid}" 2>/dev/null || true
        wait "${connector_pid}" 2>/dev/null || true
    fi
    for pid in "${novnc_pid}" "${vnc_pid}" "${window_manager_pid}" "${display_pid}"; do
        if [[ -n "${pid}" ]] && kill -0 "${pid}" 2>/dev/null; then
            kill "${pid}" 2>/dev/null || true
            wait "${pid}" 2>/dev/null || true
        fi
    done
}
trap cleanup EXIT INT TERM

if [[ "${SNAPCHAT_HEADLESS}" != "true" ]]; then
    if ! command -v Xvfb >/dev/null 2>&1; then
        echo "Xvfb is required when SNAPCHAT_HEADLESS=false. Install xvfb, or set SNAPCHAT_HEADLESS=true." >&2
        exit 1
    fi
    display_number="${DISPLAY#:}"
    display_socket="/tmp/.X11-unix/X${display_number}"
    display_lock="/tmp/.X${display_number}-lock"
    rm -f "${display_lock}" "${display_socket}"

    Xvfb "${DISPLAY}" -screen 0 1440x960x24 -ac +extension GLX +render -noreset >/tmp/snapchat-xvfb.log 2>&1 &
    display_pid=$!
    for _ in $(seq 1 50); do
        if ! kill -0 "${display_pid}" 2>/dev/null; then
            echo "Xvfb exited early; startup log follows:" >&2
            cat /tmp/snapchat-xvfb.log >&2 || true
            exit 1
        fi
        if [[ -S "${display_socket}" ]]; then
            break
        fi
        sleep 0.2
    done
    if [[ ! -S "${display_socket}" ]]; then
        echo "Timed out waiting for live Xvfb display ${DISPLAY}" >&2
        cat /tmp/snapchat-xvfb.log >&2 || true
        exit 1
    fi

    if command -v fluxbox >/dev/null 2>&1; then
        fluxbox >/tmp/snapchat-fluxbox.log 2>&1 &
        window_manager_pid=$!
    fi

    if command -v x0vncserver >/dev/null 2>&1; then
        x0vncserver -display "${DISPLAY}" -rfbport 5900 -SecurityTypes None -fg -AlwaysShared=1 -AcceptSetDesktopSize=0 >/tmp/snapchat-x0vnc.log 2>&1 &
        vnc_pid=$!
    elif command -v x11vnc >/dev/null 2>&1; then
        x11vnc -display "${DISPLAY}" -rfbport 5900 -forever -shared -nopw >/tmp/snapchat-x11vnc.log 2>&1 &
        vnc_pid=$!
    fi

    if [[ -n "${vnc_pid}" ]] && command -v websockify >/dev/null 2>&1; then
        novnc_web="/usr/share/novnc"
        if [[ -d /usr/share/novnc ]]; then
            websockify --web="${novnc_web}" 0.0.0.0:6080 localhost:5900 >/tmp/snapchat-novnc.log 2>&1 &
        else
            websockify 0.0.0.0:6080 localhost:5900 >/tmp/snapchat-novnc.log 2>&1 &
        fi
        novnc_pid=$!
        echo "snapchat_novnc=http://$(hostname -I | awk '{print $1}'):6080/vnc.html"
    else
        echo "VNC/noVNC not started; install tigervnc-scraping-server or x11vnc plus websockify/novnc for a visible sign-in window." >&2
    fi
fi

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
                print("connector_status=" + json.dumps({"ok": True}))
                sys.exit(0)
    except Exception:
        time.sleep(1)
print("Snapchat connector did not become healthy", file=sys.stderr)
sys.exit(1)
PY

if [[ "${SNAPCHAT_WAIT_FOR_AUTH_BEFORE_BRIDGE:-false}" == "true" ]]; then
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
deadline = time.time() + int(os.environ.get("SNAPCHAT_AUTH_WAIT_SECONDS", "900"))
last_log = 0

while time.time() < deadline:
    try:
        req = urllib.request.Request(base + "/session/api-auth", headers=headers)
        with urllib.request.urlopen(req, timeout=50) as resp:
            raw = resp.read().decode("utf-8", "replace")
            data = json.loads(raw)
            if data.get("authenticated"):
                print("snapchat_auth=" + json.dumps({
                    "ready": True,
                    "state": data.get("state"),
                    "selfUserID": data.get("selfUserID"),
                    "webVersion": data.get("webVersion"),
                    "cached": data.get("cached", False),
                }))
                sys.exit(0)
            if time.time() - last_log > 20:
                print("snapchat_auth_waiting=" + json.dumps({
                    "state": data.get("state"),
                    "authenticated": data.get("authenticated", False),
                    "url": data.get("url"),
                }), flush=True)
                last_log = time.time()
    except Exception as error:
        if time.time() - last_log > 20:
            print("snapchat_auth_waiting=" + json.dumps({"error": str(error)[:240]}), flush=True)
            last_log = time.time()
    time.sleep(5)

print("Timed out waiting for Snapchat authentication before bridge start", file=sys.stderr)
sys.exit(1)
PY
fi

if [[ -x "${repo_dir}/scripts/ensure_beeper_snapchat_registration_wsl.sh" ]]; then
    "${repo_dir}/scripts/ensure_beeper_snapchat_registration_wsl.sh" || true
fi

exec "${bridge_binary}" -c "${config_path}"
