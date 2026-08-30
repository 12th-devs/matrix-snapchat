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

## Known Quirks

- The Windows connector process appears in WSL process lists as an `/init` relay. Check Windows processes and port `3101` to confirm it is real.
- Headless Windows Chrome may only appear after an endpoint forces browser startup.
- `check_connector_local_wsl.sh` intentionally exercises `/healthz`, `/session/status`, `/session/api-auth`, and `/chats`, so it can take time on a cold profile.
