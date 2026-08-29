#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
config_path="${SNAPCHAT_BRIDGE_CONFIG:-${HOME}/.local/share/bbctl/bridges/sh-snapchat/config.yaml}"
bridge_name="${SNAPCHAT_BEEPER_BRIDGE_NAME:-sh-snapchat}"
bbctl_bin="${SNAPCHAT_BBCTL:-${HOME}/.local/bin/bbctl}"
icon_file="${SNAPCHAT_ICON_FILE:-${repo_dir}/assets/snapchat.svg}"

if [[ ! -f "${config_path}" ]]; then
    echo "Beeper bridge config not found at ${config_path}" >&2
    exit 1
fi
if [[ ! -x "${bbctl_bin}" ]]; then
    echo "bbctl not found at ${bbctl_bin}" >&2
    exit 1
fi
if [[ ! -f "${icon_file}" ]]; then
    echo "Snapchat icon file not found at ${icon_file}" >&2
    exit 1
fi

read_config_value() {
    local section="$1" key="$2"
    awk -v section="${section}" -v key="${key}" '
        $0 ~ "^[^[:space:]#][^:]*:" { in_section = ($1 == section ":") }
        in_section && $1 == key ":" {
            sub(/^[[:space:]]*[^:]+:[[:space:]]*/, "")
            gsub(/^["'\'']|["'\'']$/, "")
            print
            exit
        }
    ' "${config_path}"
}

read_bot_value() {
    local key="$1"
    awk -v key="${key}" '
        /^[^[:space:]#][^:]*:/ { in_appservice = ($1 == "appservice:"); in_bot = 0 }
        in_appservice && /^[[:space:]]{4}bot:/ { in_bot = 1; next }
        in_bot && /^[[:space:]]{4}[^[:space:]#][^:]*:/ { in_bot = 0 }
        in_bot && $1 == key ":" {
            sub(/^[[:space:]]*[^:]+:[[:space:]]*/, "")
            gsub(/^["'\'']|["'\'']$/, "")
            print
            exit
        }
    ' "${config_path}"
}

as_token="$(read_config_value appservice as_token)"
homeserver="$(read_config_value homeserver address)"
homeserver_domain="$(read_config_value homeserver domain)"
bot_username="$(read_bot_value username)"
icon_mxc="$(read_config_value network network_icon_mxc)"
if [[ -z "${icon_mxc}" ]]; then
    icon_mxc="$(read_bot_value avatar)"
fi

if [[ -z "${as_token}" || -z "${homeserver}" || -z "${homeserver_domain}" || -z "${bot_username}" ]]; then
    echo "Config is missing appservice token, homeserver, domain, or bot username" >&2
    exit 1
fi

if [[ "${icon_mxc}" != mxc://* ]]; then
    bot_mxid="@${bot_username}:${homeserver_domain}"
    upload_url="$(python3 - "${homeserver}" "${bot_mxid}" <<'PY'
import sys
from urllib.parse import quote, urlencode

base = sys.argv[1].rstrip("/")
user_id = sys.argv[2]
print(f"{base}/_matrix/media/v3/upload?{urlencode({'filename': 'snapchat.svg', 'user_id': user_id}, quote_via=quote)}")
PY
)"
    upload_resp="$(curl -fsS --max-time 30 \
        -H "Authorization: Bearer ${as_token}" \
        -H "Content-Type: image/svg+xml" \
        --data-binary "@${icon_file}" \
        "${upload_url}")"
    icon_mxc="$(python3 -c 'import json,sys; print(json.load(sys.stdin).get("content_uri", ""))' <<<"${upload_resp}")"
    if [[ "${icon_mxc}" != mxc://* ]]; then
        echo "Icon upload did not return an MXC URI: ${upload_resp}" >&2
        exit 1
    fi
fi

tmp_path="$(mktemp "${config_path}.snapchat-meta.XXXXXX")"
python3 - "${config_path}" "${tmp_path}" "${icon_mxc}" <<'PY'
import os
import stat
import sys

src, dst, icon = sys.argv[1:4]
with open(src, "r", encoding="utf-8") as fh:
    lines = fh.read().splitlines()

def section_bounds(name):
    start = None
    for i, line in enumerate(lines):
        if line == f"{name}:":
            start = i
            break
    if start is None:
        return None, None
    end = len(lines)
    for i in range(start + 1, len(lines)):
        line = lines[i]
        if line and not line.startswith((" ", "\t")) and line.rstrip().endswith(":"):
            end = i
            break
    return start, end

def set_section_key(section, key, value):
    start, end = section_bounds(section)
    if start is None:
        lines.extend(["", f"{section}:", f"    {key}: {value}"])
        return True
    prefix = f"    {key}:"
    for i in range(start + 1, end):
        if lines[i].startswith(prefix):
            wanted = f"    {key}: {value}"
            if lines[i] == wanted:
                return False
            lines[i] = wanted
            return True
    lines.insert(start + 1, f"    {key}: {value}")
    return True

def set_appservice_bot_key(key, value):
    start, end = section_bounds("appservice")
    if start is None:
        lines.extend(["", "appservice:", "    bot:", f"        {key}: {value}"])
        return True
    bot = None
    for i in range(start + 1, end):
        if lines[i].startswith("    bot:"):
            bot = i
            break
    if bot is None:
        lines.insert(start + 1, "    bot:")
        bot = start + 1
        end += 1
    bot_end = end
    for i in range(bot + 1, end):
        if lines[i].startswith("    ") and not lines[i].startswith("        ") and lines[i].strip() and not lines[i].lstrip().startswith("#"):
            bot_end = i
            break
    prefix = f"        {key}:"
    wanted = f"        {key}: {value}"
    for i in range(bot + 1, bot_end):
        if lines[i].startswith(prefix):
            if lines[i] == wanted:
                return False
            lines[i] = wanted
            return True
    lines.insert(bot_end, wanted)
    return True

changed = False
changed |= set_section_key("network", "network_icon_mxc", icon)
changed |= set_appservice_bot_key("displayname", "Snapchat bridge bot")
changed |= set_appservice_bot_key("avatar", icon)

if changed:
    with open(dst, "w", encoding="utf-8") as fh:
        fh.write("\n".join(lines) + "\n")
    try:
        mode = stat.S_IMODE(os.stat(src).st_mode)
        os.chmod(dst, mode)
    except OSError:
        os.chmod(dst, 0o600)
else:
    open(dst, "w").close()
PY

if [[ -s "${tmp_path}" ]]; then
    backup_path="${config_path}.bak.$(date +%Y%m%d%H%M%S)"
    cp -p "${config_path}" "${backup_path}"
    mv "${tmp_path}" "${config_path}"
    echo "Updated Snapchat Beeper metadata in ${config_path}"
    echo "Backed up previous config to ${backup_path}"
else
    rm -f "${tmp_path}"
fi

username="$("${bbctl_bin}" whoami 2>/dev/null | sed -nE 's/^User ID: @([^:]+):beeper\.com$/\1/p' | head -1)"
if [[ -z "${username}" ]]; then
    echo "Could not determine Beeper username from bbctl whoami; skipping bridge-state post" >&2
    exit 0
fi

state_resp="$(curl -sS --max-time 30 -w '\n%{http_code}' \
    -H "Authorization: Bearer ${as_token}" \
    -H "Content-Type: application/json" \
    -d '{"stateEvent":"STARTING","reason":"SELF_HOST_REGISTERED","info":{"network":"snapchat","source":"mautrix-snapchat local WSL"},"isSelfHosted":true,"bridgeType":"snapchat"}' \
    "https://api.beeper.com/bridgebox/${username}/bridge/${bridge_name}/bridge_state")"
state_body="$(sed '$d' <<<"${state_resp}")"
state_code="$(tail -n1 <<<"${state_resp}")"
case "${state_code}" in
    2*)
        echo "Posted Beeper bridge state for ${bridge_name} as Snapchat (${icon_mxc})"
        ;;
    *)
        echo "Warning: failed to post Beeper bridge state (HTTP ${state_code}): ${state_body}" >&2
        ;;
esac

sync_direct_account_data() {
    local bbctl_config="${HOME}/.config/bbctl/config.json"
    if [[ ! -f "${bbctl_config}" ]]; then
        echo "bbctl config not found; skipping Matrix m.direct sync" >&2
        return 0
    fi

    local bridge_db="${repo_dir}/sh-snapchat.db"
    if [[ ! -f "${bridge_db}" ]]; then
        echo "Bridge database not found at ${bridge_db}; skipping Matrix m.direct sync"
        return 0
    fi

    python3 - "${bbctl_config}" "${bridge_db}" "${homeserver}" "${bot_username}" "${homeserver_domain}" <<'PY'
import json
import sqlite3
import sys
from urllib import error, parse, request

bbctl_config, bridge_db, homeserver, bot_username, homeserver_domain = sys.argv[1:6]
with open(bbctl_config, "r", encoding="utf-8") as fh:
    cfg = json.load(fh)
prod = cfg.get("environments", {}).get("prod", {})
username = (prod.get("username") or "").strip()
access_token = (prod.get("access_token") or "").strip()
if not username or not access_token:
    print("bbctl prod username/token unavailable; skipping Matrix m.direct sync", file=sys.stderr)
    raise SystemExit(0)

conn = sqlite3.connect(f"file:{bridge_db}?mode=ro", uri=True, timeout=5)
rows = conn.execute("""
    SELECT mxid, other_user_id
    FROM portal
    WHERE mxid IS NOT NULL
      AND mxid != ''
      AND room_type = 'dm'
      AND other_user_id IS NOT NULL
      AND other_user_id != ''
""").fetchall()
if not rows:
    print("No Snapchat DM portals found for Matrix m.direct sync")
    raise SystemExit(0)

direct_updates = {}
for room_id, other_user_id in rows:
    ghost_mxid = f"@{bot_username.removesuffix('bot')}_{other_user_id}:{homeserver_domain}"
    direct_updates.setdefault(ghost_mxid, [])
    if room_id not in direct_updates[ghost_mxid]:
        direct_updates[ghost_mxid].append(room_id)

base = homeserver.rstrip("/")
user_id = f"@{username}:beeper.com"
url = f"{base}/_matrix/client/v3/user/{parse.quote(user_id, safe='')}/account_data/m.direct"
headers = {"Authorization": "Bearer " + access_token}
try:
    with request.urlopen(request.Request(url, headers=headers), timeout=15) as resp:
        existing = json.load(resp)
except error.HTTPError as exc:
    if exc.code == 404:
        existing = {}
    else:
        raise
if not isinstance(existing, dict):
    existing = {}

for ghost_mxid, room_ids in direct_updates.items():
    merged = list(existing.get(ghost_mxid) or [])
    for room_id in room_ids:
        if room_id not in merged:
            merged.append(room_id)
    existing[ghost_mxid] = merged

payload = json.dumps(existing, separators=(",", ":")).encode()
put = request.Request(
    url,
    data=payload,
    headers={**headers, "Content-Type": "application/json"},
    method="PUT",
)
with request.urlopen(put, timeout=15) as resp:
    resp.read()
print(f"Synced Matrix m.direct for {len(direct_updates)} Snapchat DM identities")
PY
}

sync_direct_account_data
