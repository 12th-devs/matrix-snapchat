#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
repo_windows="$(wslpath -w "${repo_dir}")"
config_path="${SNAPCHAT_BRIDGE_CONFIG:-${repo_dir}/docker/bridgev2-config.yaml}"
node_exe="${SNAPCHAT_WINDOWS_NODE:-/mnt/c/Program Files/nodejs/node.exe}"
chrome_linux_path="${SNAPCHAT_WINDOWS_CHROME:-/mnt/c/Program Files/Google/Chrome/Application/chrome.exe}"

if [[ ! -x "${node_exe}" ]]; then
    echo "Windows Node runtime not found at ${node_exe}" >&2
    exit 1
fi
if [[ ! -x "${chrome_linux_path}" ]]; then
    echo "Windows Chrome not found at ${chrome_linux_path}" >&2
    exit 1
fi
if [[ ! -f "${config_path}" ]]; then
    echo "Bridge config not found at ${config_path}" >&2
    exit 1
fi

export PORT="${PORT:-3101}"
export SNAPCHAT_BROWSER_EXECUTABLE_PATH="$(wslpath -w "${chrome_linux_path}")"
export SNAPCHAT_PROFILE_DIR="${repo_windows}\\data\\snapchat-profile"
export SNAPCHAT_TRACE_DIR="${repo_windows}\\data\\debug"
export SNAPCHAT_HEADLESS="${SNAPCHAT_HEADLESS:-true}"
export SNAPCHAT_SAFE_NO_OPEN="${SNAPCHAT_SAFE_NO_OPEN:-1}"

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

export WSLENV="${WSLENV:+${WSLENV}:}PORT/w:SNAPCHAT_BROWSER_EXECUTABLE_PATH/w:SNAPCHAT_PROFILE_DIR/w:SNAPCHAT_TRACE_DIR/w:SNAPCHAT_HEADLESS/w:SNAPCHAT_SAFE_NO_OPEN/w:SNAPCHAT_SHARED_SECRET/w"

rm -f \
    "${repo_dir}/data/snapchat-profile/SingletonCookie" \
    "${repo_dir}/data/snapchat-profile/SingletonLock" \
    "${repo_dir}/data/snapchat-profile/SingletonSocket"

cd "${repo_dir}/connector"
exec "${node_exe}" src/index.mjs
