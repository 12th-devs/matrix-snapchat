# Snapchat bridge Pi bundle

Run from the extracted bundle:

```bash
./run.sh
```

Optional overrides:

```bash
SNAPCHAT_BRIDGE_CONFIG=/path/to/config.yaml ./run.sh
SNAPCHAT_BROWSER_EXECUTABLE_PATH=/usr/bin/chromium ./run.sh
```

The runner starts the local connector, checks it, preserves Beeper appservice
tokens, tunes the Snapchat network to API-only safe-no-open mode, and then
execs the bridge binary.
