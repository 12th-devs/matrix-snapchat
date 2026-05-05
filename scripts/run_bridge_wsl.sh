#!/usr/bin/env bash
set -euo pipefail

cd "${HOME}/Codex"
mkdir -p "${HOME}/Codex/data/logs"
exec ./bin/mautrix-snapchat -config ./config.yaml
