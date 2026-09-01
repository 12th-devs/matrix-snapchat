#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
config_path="${SNAPCHAT_BRIDGE_CONFIG:-${repo_dir}/docker/bridgev2-config.yaml}"
windows_host="${SNAPCHAT_CONNECTOR_HOST:-$(ip route show default | awk '{ print $3; exit }')}"

shared_secret="${SNAPCHAT_SHARED_SECRET:-}"
if [[ -z "${shared_secret}" ]]; then
    shared_secret="$({
        awk '
            /^network:/ { in_network = 1; next }
            in_network && /^[^[:space:]]/ { exit }
            in_network && /^[[:space:]]+shared_secret:/ {
                sub(/^[[:space:]]*shared_secret:[[:space:]]*/, "")
                gsub(/^['\"']|['\"']$/, "")
                print
                exit
            }
        ' "${config_path}"
    })"
fi
if [[ -z "${shared_secret}" ]]; then
    echo "network.shared_secret is missing from ${config_path}" >&2
    exit 1
fi

export SNAPCHAT_CHECK_BASE_URL="http://${windows_host}:3101"
export SNAPCHAT_CHECK_SECRET="${shared_secret}"

python3 - <<'PY'
import json
import os
import urllib.error
import urllib.request

base_url = os.environ["SNAPCHAT_CHECK_BASE_URL"]
secret = os.environ["SNAPCHAT_CHECK_SECRET"]

def request(path):
    req = urllib.request.Request(
        base_url + path,
        headers={"X-Bridge-Secret": secret},
    )
    try:
        with urllib.request.urlopen(req, timeout=180) as response:
            return json.loads(response.read().decode("utf-8"))
    except urllib.error.HTTPError as error:
        detail = error.read().decode("utf-8", errors="replace")
        raise RuntimeError(f"{path} returned HTTP {error.code}: {detail}") from error

health = request("/healthz")
status = request("/session/status")
api_auth = request("/session/api-auth")
chats = request("/chats") if status.get("authenticated") else []

print(f"connector_url={base_url}")
print(f"health_ok={bool(health.get('ok'))}")
print(f"session_state={status.get('state', 'unknown')}")
print(f"session_authenticated={bool(status.get('authenticated'))}")
print(f"visible_chat_count={status.get('visibleChatCount', 0)}")
print(f"api_state={api_auth.get('state', 'unknown')}")
print(f"api_authenticated={bool(api_auth.get('authenticated'))}")
print(f"api_sso_token_present={bool(api_auth.get('ssoToken'))}")
print(f"api_self_user_id_present={bool(api_auth.get('selfUserID'))}")
print(f"api_identity_source={api_auth.get('identitySource', '')}")
print(f"api_authorization_endpoint={api_auth.get('authorizationEndpoint', '')}")
captured_headers = api_auth.get("apiRequestHeaders", {}).get("headers", {})
print("api_captured_header_names=" + ",".join(sorted(captured_headers)))
print("api_cookie_names=" + ",".join(sorted(cookie.get("name", "") for cookie in api_auth.get("cookies", []))))
print(f"chat_count={len(chats)}")

missing = []
if not health.get("ok"):
    missing.append("healthz")
if not status.get("authenticated"):
    missing.append("authenticated_session")
if not api_auth.get("authenticated"):
    missing.append("api_authenticated")
if not api_auth.get("ssoToken"):
    missing.append("sso_token")
if not api_auth.get("selfUserID"):
    missing.append("self_user_id")
if not api_auth.get("mcsCofIdsBin"):
    missing.append("mcs_cof_ids_bin")
if not captured_headers.get("mcs-cof-ids-bin"):
    missing.append("captured_messenger_headers")
if status.get("authenticated") and len(chats) == 0:
    missing.append("visible_chats")
if missing:
    raise SystemExit("connector_not_ready=" + ",".join(missing))
PY
