#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
config_path="${SNAPCHAT_BRIDGE_CONFIG:-${HOME}/.local/share/bbctl/bridges/sh-snapchat/config.yaml}"
bridge_binary="${SNAPCHAT_BRIDGE_BINARY:-${repo_dir}/bin/mautrix-snapchat-bridgev2-wsl-amd64}"

if [[ ! -f "${config_path}" ]]; then
    echo "Beeper bridge config not found at ${config_path}" >&2
    exit 1
fi
if [[ ! -x "${bridge_binary}" ]]; then
    echo "WSL bridge binary not found at ${bridge_binary}" >&2
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
    text = re.sub(rf"^(\s*{re.escape(key)}\s*:\s*).*$", rf"\g<1>{value}", text, flags=re.MULTILINE)
path.write_text(text)
PY

runtime_secret="$(python3 -c 'import secrets; print(secrets.token_urlsafe(48))')"
windows_host="${SNAPCHAT_CONNECTOR_HOST:-$(ip route show default | awk '{ print $3; exit }')}"
if [[ -z "${windows_host}" ]]; then
    echo "Could not determine the Windows host gateway from WSL" >&2
    exit 1
fi
export SNAPCHAT_BRIDGE_CONFIG="${config_path}"
export SNAPCHAT_SHARED_SECRET="${runtime_secret}"
export SNAPCHAT_CONNECTOR_SHARED_SECRET="${runtime_secret}"
export SNAPCHAT_CONNECTOR_BASE_URL="${SNAPCHAT_CONNECTOR_BASE_URL:-http://${windows_host}:3101}"

# Text-only baseline. Experimental read/open and media paths stay disabled until
# they pass isolated live tests.
export SNAPCHAT_READ_RECEIPTS_ENABLED=false
export SNAPCHAT_SNAP_MEDIA_ENABLED=false
export SNAPCHAT_SNAP_MEDIA_ON_READ=false
export SNAPCHAT_SEND_MEDIA_ENABLED=false

connector_pid=""
cleanup() {
    if [[ -n "${connector_pid}" ]] && kill -0 "${connector_pid}" 2>/dev/null; then
        kill "${connector_pid}" 2>/dev/null || true
        wait "${connector_pid}" 2>/dev/null || true
    fi
}
trap cleanup EXIT INT TERM

"${repo_dir}/scripts/run_connector_windows_browser_from_wsl.sh" &
connector_pid=$!

for _ in $(seq 1 30); do
    if "${repo_dir}/scripts/check_connector_local_wsl.sh" >/dev/null 2>&1; then
        break
    fi
    if ! kill -0 "${connector_pid}" 2>/dev/null; then
        echo "Snapchat connector exited during startup" >&2
        wait "${connector_pid}"
    fi
    sleep 1
done

"${repo_dir}/scripts/check_connector_local_wsl.sh"
"${repo_dir}/scripts/ensure_beeper_snapchat_registration_wsl.sh"
cd "${repo_dir}"
"${bridge_binary}" -c "${config_path}"
