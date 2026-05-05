#!/usr/bin/env bash
set -euo pipefail

export DISPLAY="${DISPLAY:-:99}"

Xvfb "${DISPLAY}" -screen 0 1440x960x24 -ac +extension GLX +render -noreset &
XVFB_PID=$!

fluxbox >/tmp/fluxbox.log 2>&1 &
FLUXBOX_PID=$!

x11vnc -display "${DISPLAY}" -forever -shared -nopw -listen 0.0.0.0 -xkb >/tmp/x11vnc.log 2>&1 &
X11VNC_PID=$!

websockify --web=/usr/share/novnc/ 0.0.0.0:6080 localhost:5900 >/tmp/novnc.log 2>&1 &
NOVNC_PID=$!

cleanup() {
  kill "${NOVNC_PID}" "${X11VNC_PID}" "${FLUXBOX_PID}" "${XVFB_PID}" 2>/dev/null || true
}
trap cleanup EXIT INT TERM

node src/index.mjs
