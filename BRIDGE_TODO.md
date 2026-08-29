# Snapchat Bridge: Operational TODO and Plan

Last audited: 2026-08-28

## Goal and Definition of Done

The bridge is "fully operational" when the local WSL/Docker deployment can stay connected to Snapchat and Beeper, survive restarts without duplicate or misattributed messages, and pass the end-to-end acceptance checklist below. This target is the reliable core bridge: login, portals, text, ordinary chat media, safe snap placeholders, identity, and optional API-only receipts. Unsupported Snapchat features must be advertised honestly instead of failing silently.

All build and stabilization work remains local to WSL. Do not upload, register, restart, or queue anything on the Pi until the user explicitly approves deployment.

## Current Snapshot

- [x] A real `bridgev2` entrypoint and connector implementation exist.
- [x] API-backed conversation sync, text send, media send, message conversion, persistent routing state, dedupe, portal resync, profiles/avatars, disappearing metadata, and API read-receipt code exist.
- [x] Compose keeps browser-backed DOM chat opening disabled with `SNAPCHAT_SAFE_NO_OPEN=1`.
- [x] Local WSL runtime is proven without Docker: Windows Chrome/Node is launched and supervised from WSL, while the bridge runs as a native WSL binary.
- [x] Current Node syntax checks pass for `connector/src/snapchat.mjs` and `connector/src/routes.mjs`.
- [x] Focused WSL Go tests pass for `./pkg/connector`, and a Linux amd64 bridge binary builds successfully.
- [x] A fresh private Beeper bridgev2 config authenticates successfully; websocket, whoami, appservice login, encryption setup, bridge icon, and management room creation all pass without `M_UNKNOWN_TOKEN`.
- [x] Safety-critical API/media flags are explicit in the repository config, and the WSL stack runner enforces a text-only baseline at runtime.
- [x] The August Snapchat Web webpack runtime is supported for identity discovery; the authenticated self UUID is resolved from the live `96821.M` store instead of the obsolete self-profile endpoint.
- [x] Safe browser discovery now reconciles all 74 unique visible Snapchat chat IDs into Matrix portals. The one-time baseline is atomically gated; repeated live polls fetch the same 74 chats with zero additional portal resyncs.
- [x] Hybrid startup no longer races legacy stored-portal, avatar, and message-backfill jobs against the browser baseline; a clean live restart completed without `SQLITE_BUSY` or missing-database errors.
- [x] `snapcap-native` has been validated against the logged-in browser session far enough to initialize auth and enumerate 20 protocol-level conversations with the recovered self UUID.
- [x] Outbound text is permitted on explicit Beeper sends while automatic DOM message reads remain locked by `SNAPCHAT_SAFE_NO_OPEN=1`; a no-content request reaches normal validation (`400`) rather than the former safety lock (`423`).
- [x] The bridge is registered as `sh-snapchat (bridgev2, self-hosted)` in `bbctl`; after the Beeper metadata/icon repair it reports `RUNNING - remote: CONNECTED (Snapchat Web / browser-session)`.
- [x] Beeper Desktop shows the Snapchat rooms in the Inbox, including `christian`, `Adeline Bainbrige`, `Phill`, `Sabrina Bolvin`, `Natalia`, `dawson`, and others.
- [x] API polling is now fast and API-only in the local WSL runner: `poll_interval_seconds: 2`, `message_fetch_limit: 20`, `api_mode: api_only`, `dom_fallback_enabled: false`, and `auto_fetch_messages: true` are enforced in the live `bbctl` config on startup.
- [x] Received and app-sent message rows are now directionally attributed: remote messages bridge through Snapchat ghost MXIDs, while messages sent from Snapchat Web bridge as `@12th-gen:beeper.com` instead of a `browser-session` ghost.
- [x] The August Snapchat Web auth store moved to `96821.M`; the connector now checks that module as well as the older `97003.M` path for API auth and EEL helper state.
- [ ] Basic two-way text is partially acceptance-tested: Beeper -> Snapchat sends, Snapchat Web/app-sent messages are detected, and inbound remote events are detected, but incoming encrypted text bodies still render as `[Snapchat message unavailable]`.
- [ ] Current per-envelope EEL probing reaches the live WASM/key-manager but exhausts candidate decrypt call shapes. The old `undefined wasm` failure is fixed; the remaining blocker is plaintext extraction from Fidelius/EEL messages.
- [ ] `snapcap-native` is the preferred next receive/decrypt path, but it cannot be imported blindly: a disposable warm-start probe seeded from the logged-in browser cookies crashed during chat-bundle DOM bootstrap (`setAttribute` on null). Bridge integration needs a small, controlled snapcap-style sidecar adapter.
- [ ] The README and `docs/BRIDGEV2_PROGRESS.md` are stale and incorrectly describe implemented bridgev2 features as missing.

## Phase 0 - Protect State and Establish a Reproducible Baseline

- [ ] Preserve the existing `data/snapchat-profile`, bridge databases, and current config before any reset or regeneration.
- [ ] Record the exact local baseline: Git commit, dirty files, Docker/WSL versions, Compose-rendered config, image IDs, and container state.
- [x] Select the user-preferred WSL-native runtime; Docker Desktop is not required for the local baseline.
- [ ] Remove or override the hard-coded Compose DNS servers if they reproduce the known container-network failure.
- [ ] Reconcile `docker/bridgev2-config.yaml` against the binary-generated config without overwriting appservice credentials or unrelated user changes.
- [x] Make the safety-critical network values explicit for the current WSL text baseline:
  - `base_url: http://connector:3101`
  - `api_mode: hybrid` (browser sidebar for safe/stable portal IDs; API only for an explicit outbound send)
  - `dom_fallback_enabled: false`
  - `auto_fetch_messages: true`
  - `read_receipts_enabled: false` until receipt validation passes
  - `snap_media_enabled: false` until ordinary-media and snap policies are separated
  - `snap_media_on_read: false`
  - `send_media_enabled: false` until isolated outgoing-media proof passes
- [ ] Render and validate Compose/config before launching containers.

Exit gate: Ubuntu can reach Docker, Compose renders cleanly, data is preserved, and the intended safe configuration is explicit and reviewable.

## Phase 1 - Restore Local Connector Health

- [ ] Rebuild both images from the current source; do not reuse stale containers as proof.
- [x] Start the connector and verify `/healthz` with the runtime shared-secret header.
- [x] Confirm the persisted Snapchat session reaches `ready`; the current authenticated session exposes 74 visible chats.
- [ ] If Chromium reports a profile lock, first prove no Chromium process owns it, then remove only `SingletonCookie`, `SingletonLock`, and `SingletonSocket`.
- [x] Verify API authentication and conversation discovery without opening chats through the DOM.
- [ ] Confirm `127.0.0.1:3101` and noVNC `127.0.0.1:6080` are local-only.

Exit gate: connector remains healthy for 30 minutes, authenticated API calls work, and no Snapchat conversation has been opened or marked read by automation.

## Phase 2 - Restore Beeper Appservice Authentication

- [ ] Keep the bridge stopped while its token is rejected to avoid a crash/retry loop.
- [x] Generate or retrieve a fresh Beeper third-party bridgev2 registration using the current `bbctl` flow.
- [x] Repair the existing generated config without deleting the appservice: preserve the current tokens/rooms, add Snapchat network icon metadata, and post the Beeper self-hosted bridge state.
- [ ] Keep secrets in ignored local runtime files and never place them in TODOs, logs, commits, or build artifacts.
- [x] Start the bridge and prove successful Matrix/Beeper authentication; `RUNNING` alone is not success.
- [x] Verify stable connector-to-bridge networking in the WSL-native runner and prove connector health/API auth before starting the bridge.

Exit gate: no `M_UNKNOWN_TOKEN`, no rejected `as_token`, no crash loop, and Beeper shows the Snapchat bridge/login as connected.

## Phase 3 - Prove Core Messaging End to End

Use a dedicated low-volume test conversation and record Matrix event IDs plus Snapchat message IDs for each case.

- [ ] Snapchat -> Beeper plain text arrives once, with correct author and timestamp.
- [x] Beeper -> Snapchat plain text delivers once and does not echo back as a duplicate in the current live test path.
- [x] Messages sent from Snapchat Web are attributed to the logged-in user, never to a ghost such as `@sh-snapchat_browser-session`.
- [x] DM portals use stable Snapchat conversation IDs and remain mapped after restart; group membership still needs explicit validation.
- [x] A clean local restart completes one bounded 74-chat portal reconciliation and later polls queue zero duplicate portal resyncs; message-level restart dedupe still needs a real two-way test.
- [ ] Deleted/re-added bridge portals can be repaired/resynced without resetting the Snapchat session.
- [x] The bridge displays as Snapchat with a working icon/account identity; friend avatars still need broader validation without excessive profile requests.
- [ ] Disappearing-message metadata matches the remote conversation without claiming unsupported behavior.

### Phase 3A - Replace Per-Envelope EEL Guessing with a Snapcap-Style Plaintext Stream

- [ ] Build a local-only Node adapter that warm-starts from the already-authenticated connector session instead of requiring Snapchat username/password:
  - import cookies from `/session/api-auth`;
  - reuse the browser user-agent;
  - seed snapcap-style `cookie_jar` storage in memory or ignored local state;
  - never persist credentials in git-tracked files.
- [ ] Port only the required `snapcap-native` receive/decrypt pieces into the adapter:
  - chat bundle + WASM boot;
  - `setupBundleSession`;
  - `messagingDelegate` wrapping;
  - `fetchConversationWithMessages` inbox pump;
  - `deliverPlaintext` conversion.
- [ ] Expose a bridge-safe connector endpoint such as `POST /session/plaintext-pump` or `GET /session/plaintext-cache` that returns decrypted plaintext by `(conversationId, messageId)` without opening chats or snaps.
- [ ] Feed plaintext cache hits into `internal/snapapi.messageBody` before falling back to `[Snapchat message unavailable]`.
- [ ] Validate against `christian` first, then one low-volume inbound test chat:
  - fresh remote text;
  - fresh self text sent in Snapchat;
  - Beeper-sent echo dedupe;
  - restart persistence with no duplicate Matrix events.
- [ ] Once text passes, reuse the same plaintext bytes path for ordinary media metadata; true snaps remain placeholders and unopened.
- [ ] When a decrypted `contentType = 3` media header is available, adapt the TextsHQ/Beeper decode flow: extract `assetId`, AES key, and IV from the media protobuf, fetch the encrypted CDN object, AES-CBC decrypt it, detect JPEG/PNG/MP4/WAV/MP3 by magic bytes, and upload only ordinary chat media to Matrix.

Exit gate: all cases pass twice, including once after a full local container restart.

## Phase 4 - Separate Ordinary Media from Snaps

The present `snap_media_enabled` switch gates all incoming media. That is too broad for the intended policy and must be split before enabling media in production.

- [ ] Add separate configuration/capability paths for ordinary chat media and true snaps.
- [ ] Ordinary chat images/videos: download via API, decrypt if required, upload to Matrix, preserve MIME/type, and dedupe across restart.
- [ ] True snaps: emit only a safe `New Snap` placeholder; do not download, render, open, hydrate on read, or mark opened from Beeper.
- [ ] Remove or permanently disable the current read-receipt-triggered snap hydration path for the safe production profile.
- [ ] Add regression fixtures that prove ordinary attachments are not classified as snaps and true snaps never invoke media download.
- [ ] Run an isolated, manually controlled outgoing image/video proof before enabling `send_media_enabled`.
- [ ] Confirm size/type errors are surfaced to Beeper and never silently dropped.

Exit gate: ordinary media works in both directions, while a true snap remains unopened and appears only as `New Snap`.

## Phase 5 - Receipts, Typing, and Feature Honesty

- [x] Add a conservative bridge-side read-receipt guard: only ordinary stored chat/text rows may call Snapchat `UpdateAction_Read`; exact receipts for unknown rows and all snap/media rows are skipped.
- [x] Keep read receipts disabled by default in local and template configs until the isolated live test passes.
- [x] Add bridgev2 typing-event plumbing behind `typing_enabled: false`; do not advertise typing notifications by default.
- [x] Keep outbound Matrix message edits rejected in capabilities until a real Snapchat edit API call is identified and proven.
- [x] Harden inbound Snapchat edit handling: if a previously stored text placeholder later decrypts into a real body, queue a Matrix edit instead of a duplicate message.
- [ ] Validate read receipts in an isolated chat with `read_receipts_enabled: true`; prove the API path does not open the DOM chat and is correctly debounced.
- [ ] Finish the snapcap-native typing adapter:
  - warm-start from `/session/api-auth` cookies/user-agent;
  - boot the chat bundle/session without the current React DOM bootstrap crash;
  - expose a narrow sidecar endpoint that calls snapcap-native `setTyping(conversationId, isTyping)`;
  - debounce/pulse at Snapchat's expected cadence and stop cleanly when Matrix typing stops.
- [ ] Investigate Snapchat-native message edit/delete semantics. If no safe API exists, keep Matrix -> Snapchat edits/deletes rejected and only support inbound remote edits/deletions where Snapchat emits them.
- [ ] Audit reactions, replies, edits, deletes, voice notes, stickers, calls, and story events. Implement only verified mappings and reject/label the rest clearly.
- [ ] Ensure bridge capability declarations match what passed live validation.

Exit gate: enabled interaction features are API-only and proven; everything else is explicitly unsupported in UI/capabilities.

## Phase 6 - Reliability, Tests, and Documentation

- [ ] Pass `go test ./...` in WSL, plus focused connector/API/store tests.
- [ ] Pass `node --check connector/src/snapchat.mjs` and `node --check connector/src/routes.mjs`.
- [ ] Add mocked integration tests for authentication failure, token expiry, retry/backoff, restart dedupe, portal repair, and media/snap separation.
- [ ] Expand EEL decryption fixtures using sanitized real envelope shapes.
- [ ] Add a repository-relative, path-safe WSL smoke script that checks config, connector health, authenticated API sync, bridge health, and recent fatal errors.
- [ ] Add bounded retry/backoff and useful health states for Snapchat auth loss, API rate limits, Matrix auth loss, and connector unavailability.
- [ ] Run a 24-hour local soak test and record CPU, memory, poll rate, API failures, reconnects, duplicates, and missed messages.
- [ ] Update README and bridgev2 progress docs to match the implementation and the tested support matrix.
- [ ] Remove secrets and machine-specific artifacts from diffs; commit only reviewed source/config-template/documentation changes.

Exit gate: all automated checks pass, the smoke test is repeatable, and the 24-hour soak has no unexplained disconnects, duplicates, or unintended snap opens.

## Phase 7 - Controlled Pi Deployment (Blocked Until Explicit Approval)

- [ ] Obtain explicit user approval before any Pi connection, upload, registration edit, service restart, or queued action.
- [ ] Build the exact reviewed ARM64 artifact and record its checksum.
- [ ] Back up the Pi service file, config, registration, databases, and Snapchat profile while preserving other bridges/accounts.
- [ ] Deploy once, migrate config deliberately, and restart only the Snapchat bridge service.
- [ ] Re-run the full acceptance checklist against the real Beeper account.
- [ ] Keep a tested rollback artifact and restore procedure ready.

Exit gate: the deployed service stays healthy through restart and the user confirms real Snapchat/Beeper behavior, not merely process status.

## Final Acceptance Checklist

- [x] Connector authenticated and healthy.
- [x] Matrix/Beeper appservice authenticated and stable for the current live WSL run.
- [x] Snapchat rooms are visible in Beeper Desktop under the registered Snapchat bridge account.
- [ ] Inbound and outbound text pass.
- [ ] Correct self/ghost attribution passes.
- [ ] Restart dedupe and portal persistence pass.
- [ ] Ordinary inbound/outbound media pass.
- [ ] True snap remains unopened and renders only as `New Snap`.
- [ ] Profiles, names, and friend avatars pass; bridge account icon now passes.
- [ ] Enabled receipts/typing pass without DOM chat opening, or are disabled and advertised unsupported.
- [ ] No fatal auth errors, crash loops, duplicate events, or unintended read/open side effects during the soak test.
- [x] User-visible Beeper room visibility is confirmed separately from container/service health.
