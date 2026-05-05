#!/usr/bin/env bash
set -euo pipefail

cd "${HOME}/Codex"
mkdir -p "${HOME}/Codex/bin" "${HOME}/Codex/data/logs"

/usr/local/go/bin/go build -tags goolm -o "${HOME}/Codex/bin/mautrix-snapchat-bridgev2" ./cmd/mautrix-snapchat-bridgev2
exec "${HOME}/Codex/bin/mautrix-snapchat-bridgev2" "$@"

