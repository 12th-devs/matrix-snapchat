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

## Duplicate Portal Cleanup 2026-09-07

- Root cause: `delete-all-portals` (mautrix built-in) wiped bridge DB rows but left ~38 portal rooms in Beeper; later activity recreated rooms per chat, producing duplicate pairs (Sabrina ×2, Clarabel ×2 observed).
- Cleanup: 37 orphan portal rooms removed via Beeper room-yeet `POST /_matrix/client/unstable/com.beeper.yeet/rooms/{id}/delete` with the bridge as_token (room_yeeting flag confirmed in /versions). PFS room `!UYASx81RyuZ8VpkrY8Hw`, bridge bot room `!uOzqojkhgfrUUB9GDsW1`, and live tracked portals were kept.
- Rebuild: `!snapchat resync-all` scanned 67 chats, preserved 3, created 64, newly bridged 354 messages, errors 0. Final state: 68 portal rows with mxids, 70 bot rooms (68 portals + PFS + bot room), 0 untracked rooms, 0 duplicate portal keys.
- Remaining same-name pairs are real distinct conversations, verified against stored participants: Isaac ×2 are two different people (mikelswim / isaac.ramirez86). `f3bc85e9` "Phill 🥶🐒" and `870db223` "Umi Dalton" are group_dm chats (2 and 6 participants) that share a display name with the corresponding DM; an earlier unresolvable-participant theory was disproven and no connector filter was implemented (user chose leave-as-is).
- Ghost MXID format is `sh-snapchat_<snapchat-user-uuid>`; portal rooms carry no `m.bridge` state, so room→chat mapping is done via ghost members.

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

## Outgoing Voice Notes 2026-09-07

- Beeper Android records voice notes as OGG/Opus (declared application/ogg); first send to Lorelei failed at the connector MIME gate (25 KB file downloaded fine, then rejected audio/mp4-only).
- Fix: ffmpeg 8.0.1 installed in WSL (apt). snapapi SendMedia transcodes audio/ogg, application/ogg, audio/opus, audio/webm to audio-only MP4/AAC (64k, faststart) before classification/encryption; SNAPCHAT_FFMPEG_PATH overrides the binary. Pure audio/mp4 passes through untouched; audio/mpeg still rejected.
- Connector gate extracted to outboundMediaMIMEAllowed (voice MIMEs allowed only for m.audio); capabilities bumped to 2026_09_07_2 with ogg/webm fully supported and re-pushed to all rooms. Regression tests pin the gate, capabilities, and a real ffmpeg OGG->MP4 round trip.
- LIVE DELIVERY NOT YET VERIFIED: waiting on a user retry after the bridge-only restart (pid 32877). Megolm outbound sessions reset on restart (rows=2).

## System/Status Room Events 2026-09-07
- Snapchat renders system lines (streak start, using Snapchat for Web, screenshots, missed calls, saves) client-side from metadata; the web bundle translation strings confirm the exact texts. No STATUS_* envelopes appeared in any available bridge log, so wire shapes for streak/web lines are still unpinned.
- Implemented now: known STATUS_* + SHARE content types map to friendly text (statusEventText in internal/snapapi/messages.go) and render as gray m.notice room events via isSystemEventContentType in pkg/connector/api.go; STICKER rides the media path. Generic STATUS keeps its prior [Snapchat status] notice. Unhandled content types log a preview (previewEnvelopeBytes) so the next live streak/web occurrence can be pinned.
- Bridge restarted pid 34356. Beeper m.notice rendering (gray system line, no New Message push) not yet visually verified; live verification pending for both status lines and a future streak/web event sample.

## Sidebar System Lines Fix 2026-09-07 (screenshot evidence)
- Lorelei's screenshot arrived NOT as a message envelope but as a sidebar preview change; sidebarUpdateText synthesized literal "New message" with empty content type, which passed the notice gate as chatty m.text (line 3363 of bridge-2026-09-07T13-37-17.917.log).
- Fix: synthesized sidebar messages now carry ContentType STATUS so the m.notice gate applies; sidebar previews containing screenshot / screen record / missed call map to friendly system text via systemEventPreviewText; "new message" added to the isGeneratedSnapchatNotice body list.
- Bridge restarted pid 35632. Still open: streak start / web-session lines have no wire sample yet; unhandled content types log preview bytes for pinning.

## Incoming Snap Media Pipeline 2026-09-07 (pipeline complete, page-sync blocker pinned)

- Ground truth established: decrypted snap contents layout is 11 (snap item) → 5 → 1 → 1 → 4 (encryption) with 4.1 = base64 32-byte AES key, 4.2 = base64 16-byte IV (raw duplicates at 19.1/19.2), and 11.17.5 = capture timestamp ms. Snaps carry no media id inside contents and use a different analytics identifier shape in the web app, so numeric-ID matching cannot find them.
- Implemented end to end (all tested, all deployed):
  - snapapi: `snapMediaEncryptionKeys` parses the decrypted contents and attaches key/IV to `Message.Media` (`messageFromProto`); `envelopeContentsForDecode` threads the message's ServerCreatedAt ms into every EEL request; `EELDecryptRequest` gained `MediaIDs` + `TimestampMs` (sidecar + inspector passthrough included).
  - Connector: page-side protobuf walker + `snapTimestampMs`; lookup strategies now (a) exact ID match, (b) media-ID content match, (c) closest-timestamp match (±120s, misattribution-safe: a wrong neighbor fails decryption and stays retryable), (d) app-state harvest + manager `getConversation` (both probed: appState.messaging has NO message arrays — feed item holds only metadata).
  - Hydration policy: `decideMediaHydration(existing, confirmed, isSnap, snapKeysReady)` — snaps render like ordinary media once keys are recovered; without keys they keep the gray 📷/🎥 placeholder and are never auto-opened. A failed render stays retryable.
- Deployed but NOT yet rendering newest snaps. Blocker is precisely pinned: the connector page's conversation fetch (`uk`/`Gq`) lags the newest messages by tens of messages / tens of minutes (its newest snap contents timestamp ~15:00 while the conversation is at ~15:26+). The app's delta sync does not advance in the connector page. Force-sync entry points exist in module 56639 (`Mw`=enterConversation, `Kz`=syncServerConversation({conversationId,conversationType,minVersion},?,?,callbacks), `QL`=getConversation) but their argument shapes are unpinned — both timeout or return nothing. Conversation-open via `openChat` navigation works (UI enters the chat) but the manager's fetch page still lags.
- Next step (single, well-defined): pin `syncServerConversation`/`enterConversation` arguments (conversationType enum + options) so the manager fetches the newest page before each EEL lookup. Everything downstream (matching → keys → CDN download → decrypt → Matrix render) is implemented and verified working for messages that reach the page window (eight earlier 200s for in-window texts).
- Bridge binary rebuilt; bridge pid 47833. Do not commit yet.

## EEL Starvation Stabilization 2026-09-07

- Root cause chain: `/healthz` stayed OK while the serialized session task lock was held; experimental EEL lookup strategies (page fetches + unbounded key-manager probing + an enterConversation attempt that always hung) made each bridge-triggered EEL retry cost 80s+; task timeout then invoked `resetSession()` → session churn; bridge `request_timeout_seconds` was 20 so callers aborted while connector work continued.
- Connector hardening (connector/src/snapchat.mjs):
  - EEL lookup budget: one attempt gets `lookupDeadline = now + 26s` shared across fetchMessage (2 shapes × 6s), `syncServerConversation` (1 attempt, 10s), and uk/Gq page fetches (8s each, budget-checked); key-manager probing stops at the deadline; `failureClass: lookup_budget_exhausted` when the budget stops work. Bounded worst case ≈ 33s observed.
  - Removed a dead `finally` block that still referenced the deleted `enterConversation` state (would have thrown ReferenceError inside the page on every attempt).
  - Task wrapper: `resetOnTimeout` opt-out; eel-decrypt sets it, so a failed/overrunning decrypt rejects the caller, recovers the queue, and NEVER resets Chrome. `timeoutMs: 90000`.
  - Lock-free `getTaskStats()` (current task, age, queue depth, last completed, last error) exposed on `/healthz` as `tasks` — health checks no longer need the lock and starvation is visible while `ok:true`.
- Bridge hardening: `request_timeout_seconds` 20 → 60 in the Beeper config; per-message EEL in-flight suppression (`eelInflight` keyed `chatID|messageID` in internal/snapapi) so poll/resync/read-hydrate cannot enqueue duplicate decrypt tasks; waiters skip (message stays retryable); success still clears `failedEEL` backoff.
- Controlled test (single POST, 92639b1c/10487): completed in 33.6s with structured 422 (`attempts_exhausted`, `fetch_message_timeout` ×2, `sync_conversation_timeout`); immediately after, healthz instant with the next task already running, `/session/status` ready, /chats fine, bridge traffic flowing, no Chrome reset.
- NEW EEL state: messages inside the app's synced pages now DECRYPT (eight consecutive `/session/eel-decrypt -> 200` at ~22s from the bridge's own backoff retries). 10487 still fails only because it fell out of the fetched page window (pages now cover 10603-10647) and `fetchMessage`/`syncServerConversation` arg shapes are unpinned — that reverse-engineering is parked until runtime stability is confirmed.
- Bridge pid 41412. Do not commit yet.

## Snap vs Chat Media Matrix Representation 2026-09-07
- Goal: incoming Snaps must be immediately distinguishable in Matrix from each other and from saved chat media. Hydration (read-based media download) already works; this changes presentation only.
- snapapi: messageFromProto refines the generic "New Snap" placeholder with the envelope-declared media kind pre-download: image/GIF -> "📷 New Snap", video -> "🎥 New Snap" (snapPlaceholderForKind in internal/snapapi/messages.go).
- connector: hydrated media parts carry unsigned extra `net.colej.snapchat.type` = snap_image | snap_video | snap | chat_media (snapTypeMarker in pkg/connector/api.go), so tooling/clients can classify structurally.
- Visible captions: disappearing Snaps get body "📷 Snap" / "🎥 Snap" (snapMediaCaption) instead of the filename body chat media keeps; real caption text still wins. isGeneratedSnapchatNotice pins the four new bodies so they stay gray-notice placeholders and never become outgoing-user text.
- Chat media (EXTERNAL_MEDIA etc.) is unchanged visually: filename body, marker chat_media.
- Tests: TestMessageFromProtoSnapPlaceholderReflectsMediaKind (snapapi), TestConvertMessageDistinguishesSnapsFromChatMedia + TestSnapPlaceholderTextStaysNotice (pkg/connector); full connector/snapapi/store packages pass, vet clean.
- Deployment gotcha: run_bridge_stack_local_wsl.sh bridge-only does NOT rebuild bin/mautrix-snapchat-bridgev2-wsl-amd64; `go build -o bin/mautrix-snapchat-bridgev2-wsl-amd64 ./cmd/mautrix-snapchat-bridgev2` must run first. Bridge restarted pid 36631 with the new binary. Live visual verification pending (next incoming snap should show 📷/🎥 placeholder then captioned media).
