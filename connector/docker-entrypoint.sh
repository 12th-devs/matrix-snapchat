#!/usr/bin/env bash
set -euo pipefail

export DISPLAY="${DISPLAY:-:99}"
export SNAPCHAT_PROFILE_DIR="${SNAPCHAT_PROFILE_DIR:-/data/snapchat-profile}"

display_number="${DISPLAY#:}"
display_socket="/tmp/.X11-unix/X${display_number}"
display_lock="/tmp/.X${display_number}-lock"

rm -f \
  "${SNAPCHAT_PROFILE_DIR}/SingletonCookie" \
  "${SNAPCHAT_PROFILE_DIR}/SingletonLock" \
  "${SNAPCHAT_PROFILE_DIR}/SingletonSocket"

rm -f "${display_lock}" "${display_socket}"

Xvfb "${DISPLAY}" -screen 0 1440x960x24 -ac +extension GLX +render -noreset >/tmp/xvfb.log 2>&1 &
XVFB_PID=$!

for _ in $(seq 1 50); do
  if ! kill -0 "${XVFB_PID}" 2>/dev/null; then
    echo "Xvfb exited early; startup log follows:" >&2
    cat /tmp/xvfb.log >&2 || true
    exit 1
  fi

  if [ -S "${display_socket}" ] && xdpyinfo -display "${DISPLAY}" >/tmp/xdpyinfo.log 2>&1; then
    break
  fi

  sleep 0.2
done

if ! xdpyinfo -display "${DISPLAY}" >/tmp/xdpyinfo.log 2>&1; then
  echo "Timed out waiting for live Xvfb display ${DISPLAY}" >&2
  cat /tmp/xvfb.log >&2 || true
  cat /tmp/xdpyinfo.log >&2 || true
  exit 1
fi

fluxbox >/tmp/fluxbox.log 2>&1 &
FLUXBOX_PID=$!

(
  while true; do
    x0vncserver \
      -display "${DISPLAY}" \
      -rfbport 5900 \
      -SecurityTypes None \
      -fg \
      -AlwaysShared=1 \
      -AcceptSetDesktopSize=0 \
      >/tmp/x0vnc.log 2>&1 || true
    echo "x0vncserver exited; retrying in 2s" >&2
    cat /tmp/x0vnc.log >&2 || true
    sleep 2
  done
) &
X0VNC_PID=$!

for _ in $(seq 1 50); do
  if python3 - <<'PY'
import socket, sys
s = socket.socket()
s.settimeout(0.5)
try:
    s.connect(("127.0.0.1", 5900))
    sys.exit(0)
except OSError:
    sys.exit(1)
finally:
    s.close()
PY
  then
    break
  fi

  sleep 0.2
done

websockify --web=/usr/share/novnc/ 0.0.0.0:6080 localhost:5900 >/tmp/novnc.log 2>&1 &
NOVNC_PID=$!

cleanup() {
  kill "${NOVNC_PID}" "${X0VNC_PID}" "${FLUXBOX_PID}" "${XVFB_PID}" 2>/dev/null || true
}
trap cleanup EXIT INT TERM

node src/index.mjs
