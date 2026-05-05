# Reference Analysis: `lalomorales22/snapchat-bridge`

This project was reviewed as a reference for turning our browser-backed Snapchat bridge into something that can eventually plug into Beeper cleanly.

## What I inspected

- `main.go`
- `connector.go`
- `bridge.go`
- `login.go`
- `message.go`
- `database.go`
- `config.go`
- `crypto.go`
- `README.md`

## What the reference repo does better

### 1. It is shaped like a Beeper bridge

The strongest part of the reference repo is not the Snapchat API code. It is the way the bridge is structured around `mautrix-go/bridgev2`.

Useful patterns:

- a dedicated `bridgev2` connector
- per-login metadata
- portal/user/message identifiers
- explicit login flows
- persistent state for users, rooms, and message IDs
- a real Beeper-oriented entrypoint using `mxmain.BridgeMain`

This is the right long-term shape for our project too.

### 2. It treats persistence as first-class

The reference repo stores:

- login state
- room mappings
- message mappings

That matters for Beeper because a real `bridgev2` bridge needs stable remote IDs, dedupe, and restart-safe state. Our browser connector already keeps a Chromium profile, but that alone is not enough for Matrix-side bookkeeping.

### 3. It separates “network logic” from “bridge logic”

Even though the direct Snapchat client is speculative, the separation itself is good:

- network API client
- login lifecycle
- message conversion
- persistence
- bridge entrypoint

That separation maps well to our browser-backed design:

- Playwright sidecar becomes the network client boundary
- Go bridge owns Beeper-facing lifecycle and state

## What I am not copying blindly

### 1. The guessed private Snapchat API endpoints

The reference repo assumes endpoints like:

- `/loq/conversations`
- `/loq/conversation_post_messages`
- `/bq/login`

Those may have been reasonable research targets, but they are not verified in the repo itself. Copying them into our code would not make the bridge more real; it would just move us from one fragile layer to another.

### 2. The direct username/password login flow

The reference repo models Snapchat login as a username/password bridge login. For Beeper, that is attractive from a product perspective, but it is risky unless the private API login path is verified and maintained.

Our current browser-backed path is less elegant, but it is more honest and testable today:

- start browser
- authenticate through Snapchat Web
- persist the session profile
- reuse the live session

### 3. The “95% complete” framing

The reference repo is useful, but the README overstates how close it is. The missing 5% is the hard part: a verified, durable Snapchat transport. I am using its structure, not its confidence level.

## What I adopted into our project

### Persistent bridge state

I added a SQLite-backed store in `internal/store/store.go`, inspired by the reference repo’s `database.go`.

This now gives us durable tables for:

- login/session state
- portal/chat state
- message dedupe state

That is directly relevant to Beeper and `bridgev2`.

### Beeper-oriented architectural direction

The project direction is now explicitly:

1. Keep the browser sidecar as the Snapchat transport for now.
2. Move the Go service toward `bridgev2` structure.
3. Preserve remote IDs, portal keys, and message mappings in SQLite.
4. Replace the current ad-hoc bridge HTTP service with an actual Beeper-facing `mxmain` entrypoint in the next phase.

## Why this is the right hybrid approach

The reference repo answers “what should a Beeper bridge look like?”

Our current project answers “what can actually talk to Snapchat today in a verifiable way?”

The right path is to combine those:

- **reference repo for bridge architecture**
- **our project for Snapchat Web transport**

That is the most realistic path to a Beeper-usable Snapchat bridge without pretending a guessed private mobile API is production-ready.

## Next implementation step

The next real milestone is:

### Replace the current Go HTTP wrapper with a `bridgev2` entrypoint

That means:

- adopting `mautrix-go/bridgev2`
- creating a `SnapchatConnector`
- modeling Beeper login flow around browser-session initiation
- mapping sidecar chats/messages into remote events
- using the new SQLite store for portal/message identity

At that point, `bbctl config --type bridgev2 sh-snapchat` becomes the correct operating model.
