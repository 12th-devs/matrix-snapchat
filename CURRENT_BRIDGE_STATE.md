# Current Bridge State

## Canonical Architecture

Persistent Windows Chrome profile -> Windows Snapchat connector on port 3101 -> WSL Go bridge -> Beeper / Matrix.

The supported local WSL runner is `scripts/run_bridge_stack_local_wsl.sh`. `scripts/run_connector_windows_browser_from_wsl.sh` remains the foreground diagnostic launcher for connector failures. The canonical full-stack runner starts a detached Windows connector process itself because WSL foreground relays were proven to die when the original runner exits. Do not invent another parallel startup flow unless this one is proven fundamentally broken.

## Commands

Full stack restart, used when the connector is dead, Chrome/session needs relaunching, or the runtime shared secret must be regenerated:

```bash
cd /mnt/c/Users/colej/Projects/Codex
bash scripts/run_bridge_stack_local_wsl.sh full
```

Bridge-only restart, used when only Go bridge code changed and the connector/Chrome session is healthy:

```bash
cd /mnt/c/Users/colej/Projects/Codex
bash scripts/run_bridge_stack_local_wsl.sh bridge-only
```

Stop tracked local processes:

```bash
cd /mnt/c/Users/colej/Projects/Codex
bash scripts/run_bridge_stack_local_wsl.sh stop
```

When launching from Windows PowerShell, specify Ubuntu explicitly:

```powershell
wsl.exe -d Ubuntu -e bash -lc 'cd /mnt/c/Users/colej/Projects/Codex && bash scripts/run_bridge_stack_local_wsl.sh full'
```

The default WSL distro on this machine has been observed as `docker-desktop`, where `wsl.exe -e bash` fails because `bash` is not present.

## Important Paths

- Windows repo: `C:\Users\colej\Projects\Codex`
- WSL repo: `/mnt/c/Users/colej/Projects/Codex`
- Beeper bridge config: `${HOME}/.local/share/bbctl/bridges/sh-snapchat/config.yaml`
- WSL bridge binary: `bin/mautrix-snapchat-bridgev2-wsl-amd64`
- Windows Chrome profile: `data/snapchat-profile`
- Connector debug traces: `data/debug`
- Runtime env file: `data/runtime/local-stack.env`
- Connector Windows PID file: `data/runtime/connector.pid`
- Bridge PID file: `data/runtime/bridge.pid`
- Connector log: `logs/connector.log`
- Connector stdout/stderr: `logs/connector.out.log`, `logs/connector.err.log`
- Bridge log: `logs/bridge.log`

## Runtime Secret Lifecycle

`run_bridge_stack_local_wsl.sh full` generates a new random runtime shared secret, writes it to `data/runtime/local-stack.env`, starts the connector with that secret, and starts the Go bridge with the same secret.

`run_bridge_stack_local_wsl.sh bridge-only` reads `data/runtime/local-stack.env` and reuses the existing connector secret. Bridge-only restart must not use process-memory scraping, PowerShell P/Invoke, or other secret-recovery hacks.

## Connector and Chrome Lifetime

The connector launcher starts Windows `node.exe`, which launches Windows Chrome through Playwright using the persistent `data/snapchat-profile` profile. A WSL `/init` relay process is not proof that the connector is alive. Verify the real Windows `node.exe`, port `3101`, and `/healthz`.

The full-stack runner writes connector stdout/stderr to `logs/connector.out.log` and `logs/connector.err.log`, records a short launch summary in `logs/connector.log`, and fails loudly if the connector exits early or health never succeeds. The connector is launched through a detached Windows PowerShell wrapper and tracked with `data/runtime/connector.pid` so a bridge-only restart does not kill the connector/Chrome session. It does not treat an empty console log as useful evidence.

## Current Working Evidence

- Foreground connector launch with `scripts/run_connector_windows_browser_from_wsl.sh` stayed alive.
- Windows `node.exe` listened on port `3101`.
- Windows Chrome launched with `--user-data-dir=C:\Users\colej\Projects\Codex\data\snapchat-profile`.
- `/healthz` returned success.
- `/session/status` returned `state=ready`, `authenticated=true`.
- `/session/api-auth` returned authenticated cookies and an SSO token after the timeout fix.
- Existing `logs/bridge.log` shows prior Beeper websocket pings and Matrix message sends on 2026-08-30.
- Beeper→Snapchat delete/unsend is live-verified (2026-08-30) for newly sent/mapped outgoing text messages: deleting such a message in Beeper reached `UpdateAction_Erase` with the correct message ID and returned `success=true`; the message disappeared in Snapchat.
- The nested `conversationDestinationResult.createdMessageId` field in `CreateContentMessageResponse` is required to capture the real Snapchat server message ID on outgoing sends. The top-level `result.createdMessageId` alone was insufficient, which caused the bridge to persist the client resolution ID and later erase attempts to fail with `failure_type=NOT_FOUND`.
- Legacy caveat: older Beeper-sent messages persisted before the server-ID fix may carry client-resolution IDs (huge ~10^18 values). Deleting those can still fail with `failure_type=NOT_FOUND`, and redactions of messages the bridge has no mapping for are reported as `redaction target message not found`. Only newly sent messages verify the fix.

## Current Broken or Risky Areas

- Before the timeout fix, `/session/api-auth` could time out after about 50 seconds even though messenger warmup could legitimately run longer. The timeout handler then closed Chrome while warmup was still using it.
- Launching Windows `node.exe` through a foreground WSL relay ties connector lifetime to the runner. Bridge-only restart could kill the old bridge, causing the old full runner to exit and take connector/Chrome down with it. The canonical runner now starts and tracks a detached Windows PID instead.
- Captured messenger headers and self identity may still be absent on a cold warmup. API auth can still return cookies and SSO token, but SyncConversations is the real proof.
- Do not continue Snapchat delete/unsend, replies, edits, disappearing-message metadata, usernames/chat secondary text, or snap/media decryption until runtime startup is stable.

## Safety Rules

- Do not wipe `data/snapchat-profile` or Snapchat cookies/session.
- Do not recreate Beeper registration unless there is concrete proof it is broken.
- Preserve `safeNoOpen=true`.
- Keep snap/media auto-open disabled.
- Use WSL/Ubuntu for local bridge work. Do not upload to or queue work on the Pi without explicit approval.
- Verify concrete process, port, and endpoint state. Do not infer from process names alone.
- `pgrep -f` can match itself. Prefer exact tracked PID files plus endpoint checks.
- PowerShell can interpolate Bash `$?` before Bash receives it; quote commands carefully.

## Resync Recovery Verified 2026-09-05

- Matrix management command: `!snapchat resync-all [chat-id]`. Coverage is native discovery union existing portals for the current login, deduplicated by chat ID; stored metadata is fallback only. Messages use the existing native query path with limit 20.
- Target `86ed4c7f-229d-50e9-bcd9-7b5adad048ea` kept portal rowid 16 and receiver `browser-session`. Dead `!xepqzRcmGfBzoBXEy67B:beeper.local` was replaced through normal room creation with `!ihl4S0nA5R5TpxoaWna5:beeper.local`.
- Existing PFS `!UYASx81RyuZ8VpkrY8Hw:beeper.local` retained; new child linked, old child cleared. New room live members and Megolm encryption state returned HTTP 200.
- Scoped first resync: coverage 66 native / 75 union, scanned 1, replaced 1, fetched 20, bridged 20, hydrated 1, errors 0. Message 4114 downloaded/decrypted an 89793-byte JPEG, sent encrypted, persisted MediaDelivered and HydratedAt.
- Immediate second scoped resync: scanned 1, preserved 1, bridged 0, hydrated 0, errors 0; 4114 skipped before download. Its unchanged event `$BTDiqZ9HOlLepJyb4ieTZGA5pK_UpZprpOoINxXA0Zs:beeper.local` is retrievable in the new room.
- Incoming media remains enabled, media-on-read and outgoing media disabled. Manual resync locks this login; polling skips while it runs. Historical delivery proof checks the current room; post-send persistence does not require an immediate event read.
- Chat Accounts icon remains a Beeper canonical-type/icon-registry blocker: no supported identity-preserving metadata update found. Do not re-register or rotate credentials to test this.

## Known Quirks

- The Windows connector process appears in WSL process lists as an `/init` relay. Check Windows processes and port `3101` to confirm it is real.
- Headless Windows Chrome may only appear after an endpoint forces browser startup.
- `check_connector_local_wsl.sh` intentionally exercises `/healthz`, `/session/status`, `/session/api-auth`, and `/chats`, so it can take time on a cold profile.

## Alpha Validation Blockers 2026-09-05

- Beeper Desktop API (supported OAuth, localhost:23373) decoded 4114 as IMAGE, image/jpeg, 89793 bytes, with its encrypted MXC attachment. User visually confirmed the image with literal `Media` caption. Transport was not the presentation failure: the generated fallback was being retained as a caption. New conversion uses the filename instead; real captions are preserved.
- Historical caption repair uses normal bridgev2 edits without download/upload, but current ratcheting policy prevents bridge-side historical GetEvent decryption. Preserve encryption policy and the original visible image/caption; this cosmetic limitation no longer aborts resync. No claim that 4114's caption was repaired.
- Live text regression reproduced in Phill (`58867445-6026-5827-8bfc-0aac38259257`): messages 740, 742, 743 around 18:29 local have `[Snapchat message unavailable]`; connector EEL helper logs `attempts_exhausted`. This is Snapchat plaintext recovery, not a Matrix/Megolm failure. Underlying cause remains unresolved; do not call text generally alpha-stable.
- Phill ordinary image 745 (18:35:37 local) has key/IV metadata but descriptor RPC returns an empty response. It remains retryable with HydratedAt NULL and no successful media proof. Two other sampled chats likewise yielded unavailable/invalid payloads. Only JPEG 4114 is visually verified; PNG conversion has tests, video is not live verified. Do not auto-open view-once snaps or investigate forbidden pagination as a shortcut.
- Manual resync now suppresses undecoded historical text instead of replaying unavailable placeholders. It does not repair EEL decryption; existing Matrix placeholders remain intact.
- Restart test reproduced a restored login using config-default localhost instead of the WSL gateway because the sidecar client can be created before connector Start. Client construction now applies existing runtime URL/secret overrides directly, with a regression test.
- Reusable operator tools: `python3 scripts/bridge_admin.py snapshot --output data/runtime/NAME.json`, `compare data/runtime/NAME.json`, `resync [chat-id]`, `report --since-byte N`, `chat NAME`, `media`. Snapshots are gitignored, read-only DB/live state evidence. `node scripts/check_beeper_media.mjs ROOM EVENT [--focus]` uses normal Desktop OAuth with credentials held in memory; focus may mark the chat read.
- Baseline `data/runtime/alpha-before.json`: 75 portals, 476 mappings, one login/PFS. Subsequent comparison preserved rows, healthy MXIDs, login identity and exact PFS children; live traffic increased mappings. Full 75-chat resync twice, complete alpha audit/cleanup and stable checkpoint commit are deferred because priority-1 media/text usability remains blocked.
- Targeted package tests and vet pass. Bridge-only restarts use `SNAPCHAT_BRIDGE_RESET_OUTBOUND_MEGOLM_ON_START=false` to retain existing outbound sessions, original connector/Chrome session and credentials. Additional builds were necessary after reproduced runtime regressions; no full connector restart or registration change was performed.
