#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
config_path="${SNAPCHAT_BRIDGE_CONFIG:-${repo_dir}/docker/bridgev2-config.yaml}"
node_root="${SNAPCHAT_NODE_ROOT:-${HOME}/.local/node-v22.22.1-linux-x64}"

if [[ ! -x "${node_root}/bin/node" ]]; then
    echo "WSL Node runtime not found at ${node_root}" >&2
    exit 1
fi
if [[ ! -f "${config_path}" ]]; then
    echo "Bridge config not found at ${config_path}" >&2
    exit 1
fi

export PATH="${node_root}/bin:${PATH}"
export PORT="${PORT:-3101}"
export PLAYWRIGHT_HOST_PLATFORM_OVERRIDE="${PLAYWRIGHT_HOST_PLATFORM_OVERRIDE:-ubuntu24.04-x64}"
export SNAPCHAT_PROFILE_DIR="${SNAPCHAT_PROFILE_DIR:-${repo_dir}/data/snapchat-profile}"
export SNAPCHAT_TRACE_DIR="${SNAPCHAT_TRACE_DIR:-${repo_dir}/data/debug}"
export SNAPCHAT_HEADLESS="${SNAPCHAT_HEADLESS:-true}"
export SNAPCHAT_SAFE_NO_OPEN="${SNAPCHAT_SAFE_NO_OPEN:-1}"

if [[ -z "${SNAPCHAT_BROWSER_EXECUTABLE_PATH:-}" ]]; then
    windows_chrome="/mnt/c/Program Files/Google/Chrome/Application/chrome.exe"
    if [[ -x "${windows_chrome}" ]]; then
        export SNAPCHAT_BROWSER_EXECUTABLE_PATH="${windows_chrome}"
    fi
fi

if [[ -z "${SNAPCHAT_SHARED_SECRET:-}" ]]; then
    SNAPCHAT_SHARED_SECRET="$({
        awk '
            /^network:/ { in_network = 1; next }
            in_network && /^[^[:space:]]/ { exit }
            in_network && /^[[:space:]]+shared_secret:/ {
                sub(/^[[:space:]]*shared_secret:[[:space:]]*/, "")
                gsub(/^['\"']|['\"']$/, "")
                print
                exit
            }
        ' "${config_path}"
    })"
    export SNAPCHAT_SHARED_SECRET
fi
if [[ -z "${SNAPCHAT_SHARED_SECRET}" ]]; then
    echo "network.shared_secret is missing from ${config_path}" >&2
    exit 1
fi

mkdir -p "${SNAPCHAT_PROFILE_DIR}" "${SNAPCHAT_TRACE_DIR}" "${repo_dir}/data/logs"

if pgrep -f "chrome.*snapchat-profile|connector/src/index.mjs" >/dev/null; then
    echo "A connector or browser process is already using the Snapchat profile" >&2
    exit 1
fi

rm -f \
    "${SNAPCHAT_PROFILE_DIR}/SingletonCookie" \
    "${SNAPCHAT_PROFILE_DIR}/SingletonLock" \
    "${SNAPCHAT_PROFILE_DIR}/SingletonSocket"

cd "${repo_dir}/connector"
exec node src/index.mjs
