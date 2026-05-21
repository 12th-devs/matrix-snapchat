#!/usr/bin/env bash
set -euo pipefail
cd /home/cole/mautrix-snapchat
mkdir -p bin
curl -fsSL http://192.168.12.22:8765/bin/mautrix-snapchat-bridgev2 -o bin/mautrix-snapchat-bridgev2
chmod +x bin/mautrix-snapchat-bridgev2
python3 - <<'PY'
from pathlib import Path
compose = Path('docker-compose.yml')
text = compose.read_text()
mount = '      - ./bin/mautrix-snapchat-bridgev2:/usr/local/bin/mautrix-snapchat-bridgev2:ro\n'
if mount not in text:
    text = text.replace('      - ./docker:/config\n', '      - ./docker:/config\n' + mount)
compose.write_text(text)
cfg = Path('docker/bridgev2-config.yaml')
text = cfg.read_text()
for old, new in {
    '    poll_interval_seconds: 2': '    poll_interval_seconds: 10',
    '    poll_interval_seconds: 8': '    poll_interval_seconds: 10',
    '    message_fetch_limit: 10': '    message_fetch_limit: 5',
    '    message_fetch_limit: 40': '    message_fetch_limit: 5',
    '    private_chat_portal_meta: true': '    private_chat_portal_meta: false',
}.items():
    text = text.replace(old, new)
cfg.write_text(text)
PY
docker compose up -d connector bridge
sleep 8
docker compose ps connector bridge
curl -s -H 'X-Bridge-Secret: change-me' http://127.0.0.1:3101/session/status || true
printf '\n--- bridge errors ---\n'
docker compose logs --tail=120 bridge | grep -E 'ERR|Portal event channel|RemoteEventChatInfoChange|Failed to handle Matrix message|panic' || true
printf '\n--- config gate ---\n'
grep -E 'poll_interval_seconds|message_fetch_limit|private_chat_portal_meta|dom_fallback_enabled|read_receipts_enabled' docker/bridgev2-config.yaml
