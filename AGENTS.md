# Snapchat Bridge Agent Rules

Read `CURRENT_BRIDGE_STATE.md` first.

## Before Changing Anything

- Inspect `git status` and `git diff`.
- Inspect `scripts/run_bridge_stack_local_wsl.sh`.
- Inspect `scripts/run_connector_windows_browser_from_wsl.sh`.
- Inspect real WSL and Windows process state.
- Verify whether port `3101` is actually free or listening.
- Reproduce connector startup failures in the foreground when the failure is unknown.

## Permanent Rules

- Do not invent alternate launchers. Fix the canonical runner unless it is proven fundamentally broken.
- Do not wipe the persistent Chrome profile, cookies, or Snapchat session.
- Do not recreate Beeper registration unless it is proven broken.
- Preserve `safeNoOpen=true` and keep snap/media auto-open disabled.
- Do not use PowerShell P/Invoke, process-memory scraping, or secret-recovery hacks for disposable runtime state.
- Prefer concrete evidence over speculation: real process, real port, real endpoint, real log.
- A WSL `/init` relay is not proof that Windows `node.exe` is alive.
- `pgrep -f` can self-match.
- If the expected plan stops matching reality, stop and report before taking a large detour.
- Use Ubuntu explicitly from PowerShell: `wsl.exe -d Ubuntu -e bash -lc '...'`.
- Do not upload to or queue work on the Pi unless the user explicitly approves it.

## Paused Feature Work

Do not continue Snapchat delete/unsend reverse engineering, reply capability, message editing, disappearing-message metadata, usernames/chat secondary text, or snap/media decryption until the runtime is stable and documented.
