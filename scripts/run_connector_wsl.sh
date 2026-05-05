#!/usr/bin/env bash
set -euo pipefail

cd "${HOME}/Codex/connector"
export PORT="${PORT:-3101}"
export SNAPCHAT_SHARED_SECRET="${SNAPCHAT_SHARED_SECRET:-1225}"
export SNAPCHAT_PROFILE_DIR="${SNAPCHAT_PROFILE_DIR:-${HOME}/Codex/data/snapchat-profile}"
export SNAPCHAT_TRACE_DIR="${SNAPCHAT_TRACE_DIR:-${HOME}/Codex/data/debug}"
export SNAPCHAT_HEADLESS="${SNAPCHAT_HEADLESS:-false}"

mkdir -p "${HOME}/Codex/data/debug" "${HOME}/Codex/data/logs"
exec npm start
