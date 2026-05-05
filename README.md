# mautrix-snapchat

Snapchat Web -> Matrix/Beeper bridge workspace, modeled after the browser-backed pattern used by `mautrix-gmessages`.

## Current status

This repo now gives you:

- a Go bridge service with config loading, health endpoints, polling, and a connector client
- a Playwright Snapchat Web sidecar with persistent login, chat scraping, message scraping, message sending, and diagnostics capture
- a WSL-first local run flow that matches Beeper's current self-hosting requirements
- a new compileable `bridgev2` binary scaffold for Beeper-oriented integration

What it does **not** give you yet is a production-complete `mautrix-go/bridgev2` implementation. That means you can start it, log into Snapchat Web, list chats, fetch messages, send test messages, and debug selectors locally, but you should treat Beeper hookup as the next integration layer rather than something already finished end-to-end.

That limitation is real, and I do not want to pretend otherwise.

## Files you will care about

- `cmd/mautrix-snapchat/main.go`: bridge entrypoint
- `cmd/mautrix-snapchat-bridgev2/main.go`: new `bridgev2` entrypoint
- `internal/bridge/bridge.go`: bridge runtime + debug HTTP endpoints
- `internal/bridgev2/`: `bridgev2` connector/login/API scaffolding
- `internal/connector/client.go`: Go client for the Playwright sidecar
- `connector/src/index.mjs`: sidecar HTTP server
- `connector/src/snapchat.mjs`: hardened Snapchat Web Playwright logic
- `config.example.yaml`: starter config

## How the hardened selector layer works

The Snapchat connector is intentionally not tied to one brittle selector path. It uses:

- multiple fallback locators for login detection, search input, chat rows, message bubbles, and composer
- visible-region filtering so it favors the left chat list and current conversation instead of random page text
- serialized browser operations so chat scraping and sending do not stomp each other
- persistent Chromium user data so Snapchat login survives restarts
- diagnostics capture at `/debug/diagnostics`, which saves both HTML and screenshot files for retuning

Diagnostics land in `data/debug/`.

## Step-by-step: start it locally in WSL

These instructions are based on Beeper's current self-hosting docs, last updated **November 19, 2025**, which still say:

- supported host OS: Linux or macOS
- Windows users should use WSL
- BridgeV2 third-party bridges should use `bbctl config --type bridgev2 sh-<bridgename>`

### 1. Install WSL2 Ubuntu

In Windows PowerShell:

```powershell
wsl --install -d Ubuntu
```

Reboot if Windows asks you to, then open Ubuntu.

### 2. Install system packages inside WSL

```bash
sudo apt update
sudo apt install -y golang-go nodejs npm ffmpeg git curl
```

### 3. Open this repo in WSL

```bash
cd /mnt/c/Users/colej/Projects/Codex
```

### 4. Install the Node sidecar dependencies

```bash
cd connector
npm install
npx playwright install chromium
cd ..
```

### 5. Create your runtime config

```bash
cp config.example.yaml config.yaml
```

Edit `config.yaml` and set at least:

- `connector.shared_secret`: change this from `change-me`
- `network.headless`: keep `false` for first login
- `network.browser_profile_dir`: leave as-is unless you want the browser profile elsewhere

### 6. Build the Go service

```bash
go build -o bin/mautrix-snapchat ./cmd/mautrix-snapchat
```

### 7. Start the Snapchat connector

In terminal 1:

```bash
cd /mnt/c/Users/colej/Projects/Codex/connector
export PORT=3101
export SNAPCHAT_SHARED_SECRET='replace-this-with-the-same-secret'
export SNAPCHAT_PROFILE_DIR='/mnt/c/Users/colej/Projects/Codex/data/snapchat-profile'
export SNAPCHAT_TRACE_DIR='/mnt/c/Users/colej/Projects/Codex/data/debug'
export SNAPCHAT_HEADLESS=false
npm run dev
```

### 8. Start the Go bridge service

In terminal 2:

```bash
cd /mnt/c/Users/colej/Projects/Codex
./bin/mautrix-snapchat -config ./config.yaml
```

### 9. Launch Snapchat Web login

In terminal 3:

```bash
curl -X POST http://127.0.0.1:3101/session/start \
  -H "X-Bridge-Secret: replace-this-with-the-same-secret"
```

Check status:

```bash
curl http://127.0.0.1:3101/session/status \
  -H "X-Bridge-Secret: replace-this-with-the-same-secret"
```

You want to see:

- `"state":"ready"`
- `"authenticated":true`

### 10. Log into Snapchat Web

Use the browser window Playwright opens.

- scan the QR code or complete the login flow
- wait until chats are visible
- re-run `/session/status`

### 11. Test chat discovery

```bash
curl http://127.0.0.1:3101/chats \
  -H "X-Bridge-Secret: replace-this-with-the-same-secret"
```

You should get chat names back as JSON.

### 12. Test reading a conversation

```bash
curl "http://127.0.0.1:3101/messages?chatName=Exact%20Friend%20Name" \
  -H "X-Bridge-Secret: replace-this-with-the-same-secret"
```

### 13. Test sending a message

```bash
curl -X POST http://127.0.0.1:3101/messages \
  -H "Content-Type: application/json" \
  -H "X-Bridge-Secret: replace-this-with-the-same-secret" \
  -d '{"chatName":"Exact Friend Name","text":"hello from mautrix-snapchat"}'
```

### 14. Use diagnostics if Snapchat's UI changed

```bash
curl http://127.0.0.1:3101/debug/diagnostics \
  -H "X-Bridge-Secret: replace-this-with-the-same-secret"
```

That response tells you:

- current state
- visible chat names
- visible page text
- saved HTML path
- saved screenshot path

If the selectors miss, open the saved files in `data/debug/` and tune `connector/src/snapchat.mjs`.

## Step-by-step: prepare for Beeper

This is the honest state as of **March 18, 2026**:

- Beeper's latest self-hosting docs still support third-party `bridgev2` bridges via `bbctl`
- this repo is **not yet a complete `bridgev2` bridge**, so `bbctl` cannot attach it to your Beeper account as-is

When the `bridgev2` layer is implemented, the Beeper steps are:

### 1. Install `bbctl` in WSL

Follow Beeper's current download instructions from:

- [Self-Hosting Bridges](https://developers.beeper.com/bridges/self-hosting/)
- [bridge-manager releases](https://github.com/beeper/bridge-manager/releases)

### 2. Log in to Beeper

```bash
bbctl login
```

### 3. Generate a third-party bridgev2 config

```bash
bbctl config --type bridgev2 sh-snapchat
```

### 4. Merge in this repo's `network` section

Take the generated config and merge:

- `connector`
- `network`
- any bridge-specific runtime settings

### 5. Run the bridge with that config

```bash
./bin/mautrix-snapchat -config ~/.local/share/bbctl/bridges/sh-snapchat/config.yaml
```

At that point Beeper would expect:

- a valid Matrix appservice / bridgev2 implementation
- remote network login via the bridge
- incoming and outgoing event mapping

This repo currently has the remote-network side in place, but not the full Matrix bridge side.

## Bridgev2 progress

There is now a separate `bridgev2` path in the repo. It builds in WSL with:

```bash
cd ~/Codex
go build -tags goolm -o ./bin/mautrix-snapchat-bridgev2 ./cmd/mautrix-snapchat-bridgev2
```

Generate an example `bridgev2` config:

```bash
./bin/mautrix-snapchat-bridgev2 -e -c /tmp/mautrix-snapchat-bridgev2.yaml
```

That generated config already contains the Snapchat Web sidecar settings under `network:`.
It also now contains:

- `state_db_path` for persistent bridgev2 chat routing state
- `login_wait_seconds` for browser-session login timeout

For a more detailed explanation of the architecture and current state, see:

- `docs/REFERENCE_ANALYSIS.md`
- `docs/BRIDGEV2_PROGRESS.md`

## Debug endpoints

The sidecar exposes:

- `POST /session/start`
- `GET /session/status`
- `GET /chats`
- `GET /messages?chatName=<name>`
- `POST /messages`
- `GET /debug/diagnostics`

The Go service mirrors:

- `GET /healthz`
- `POST /connector/start`
- `GET /connector/chats`
- `GET /connector/messages?chatName=<name>`
- `POST /connector/send`
- `GET /connector/debug`

## Common failure cases

### Status stays `login_required`

- keep `SNAPCHAT_HEADLESS=false`
- delete the browser profile only if the session is corrupt:

```bash
rm -rf /mnt/c/Users/colej/Projects/Codex/data/snapchat-profile
```

### Chats do not show up

- call `/debug/diagnostics`
- inspect the saved screenshot
- inspect the saved HTML
- update the fallback selectors in `connector/src/snapchat.mjs`

### Sending fails but chat scraping works

- the composer selector likely moved
- inspect the diagnostics HTML for `textarea`, `contenteditable`, or `aria-label` changes

### The Go service cannot reach the connector

- make sure `connector.base_url` matches the sidecar port
- make sure the shared secret matches on both sides

## Sources

- [Build a Beeper Bridge](https://blog.beeper.com/2025/10/28/build-a-beeper-bridge/)
- [Beeper self-hosting docs](https://developers.beeper.com/bridges/self-hosting/)
- [bridge-manager](https://github.com/beeper/bridge-manager)
- [mautrix-go](https://github.com/mautrix/go)
- [mautrix-whatsapp](https://github.com/mautrix/whatsapp)
- [mautrix-gmessages](https://github.com/mautrix/gmessages)
