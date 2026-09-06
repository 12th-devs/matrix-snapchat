#!/usr/bin/env python3
"""WSL operator tool; requires PyYAML. No credential creation or local DB writes.
Snapshots span independent SQLite transactions and live Matrix reads, not one
global transaction. Run from any directory; relative DB paths use the repo root.
"""
import argparse
import collections
import contextlib
import json
import os
from pathlib import Path
import re
import sqlite3
import subprocess
import sys
import tempfile
import time
import urllib.error
import urllib.parse as url
import urllib.request as http
import uuid
ROOT = Path(__file__).resolve().parents[1]
RUNTIME = ROOT / "data/runtime"
LOG = ROOT / "logs/bridge.log"
METRICS = ("chats scanned", "healthy portals preserved", "portals created", "dead portals replaced",
           "PFS links repaired", "memberships repaired", "messages fetched", "messages already mapped",
           "messages newly bridged", "messages backfilled after room replacement", "media candidates",
           "media hydrated", "media skipped as delivered", "errors")

def database(value):
    parsed = url.urlsplit(str(value))
    if parsed.scheme not in ("", "file") or parsed.netloc or not parsed.path or parsed.path == ":memory:":
        raise ValueError("SQLite file required")
    path = (ROOT / Path(url.unquote(parsed.path)).expanduser()).resolve()
    db = sqlite3.connect(path.as_uri() + "?mode=ro", uri=True, timeout=10)
    db.row_factory = sqlite3.Row
    db.execute("PRAGMA query_only=ON")
    db.execute("BEGIN")
    return contextlib.closing(db)

def rows(db, query):
    return [dict(row) for row in db.execute(query)]

class NoRedirect(http.HTTPRedirectHandler):
    def redirect_request(self, *args, **kwargs):
        return None

def matrix(config, path, user=None, body=None):
    server = config["homeserver"]["address"]
    token = config["appservice"]["as_token"]
    if user:
        domain = user.split(":", 1)[1]
        dp = config["double_puppet"]
        secret = dp["secrets"][domain]
        if not secret.startswith("as_token:") or not secret[9:]:
            raise ValueError("Configured DP as_token required")
        token = secret[9:]
        server = dp.get("servers", {}).get(domain, server)
    else:
        user = "@" + config["appservice"]["bot"]["username"] + ":" + config["homeserver"]["domain"]
    parsed = url.urlsplit(server)
    if parsed.scheme != "https" or parsed.username or parsed.password or parsed.query or parsed.fragment:
        raise ValueError("Clean HTTPS Matrix address required")
    request = http.Request(server.rstrip("/") + "/_matrix/client/v3/" + path + "?" + url.urlencode({"user_id": user}),
                           data=None if body is None else json.dumps(body).encode(),
                           headers={"Authorization": "Bearer " + token, "Content-Type": "application/json"},
                           method="GET" if body is None else "PUT")
    with http.build_opener(http.ProxyHandler({}), NoRedirect()).open(request, timeout=30) as response:
        return json.load(response)

def snapshot(config):
    result = {"version": 1, "started_at": time.time()}
    with database(config["database"]["uri"]) as db:
        result["portals"] = rows(db, "SELECT rowid,bridge_id,id,receiver,mxid FROM portal ORDER BY bridge_id,id,receiver")
        result["logins"] = rows(db, "SELECT bridge_id,user_mxid,id,id AS receiver,remote_name,space_room FROM user_login ORDER BY bridge_id,id")
        result["messages"] = rows(db, "SELECT rowid,bridge_id,id,part_id,mxid,room_id,room_receiver,metadata FROM message ORDER BY rowid")
    for message in result["messages"]:
        metadata = json.loads(message["metadata"] or "{}")
        message["metadata"] = {key: metadata[key] for key in ("media_delivered", "attachment_count", "presentation_version")
                               if type(metadata.get(key)) in (bool, int)}
    with database(config["network"]["state_db_path"]) as db:
        result["state_messages"] = rows(db, "SELECT portal_key,remote_id,kind,has_media,hydrated_at FROM message_state ORDER BY portal_key,remote_id")
        result["state_logins"] = rows(db, "SELECT user_id,remote_id,remote_name,last_seen_state,updated_at FROM login_state ORDER BY user_id")
    counts = collections.Counter((m["bridge_id"], m["room_id"], m["room_receiver"]) for m in result["messages"])
    result["spaces"] = {}
    for room in sorted({login["space_room"] for login in result["logins"] if login["space_room"]}):
        state = matrix(config, "rooms/" + url.quote(room, safe="") + "/state")
        children = sorted({e["state_key"] for e in state if e["type"] == "m.space.child" and e.get("content", {}).get("via")})
        result["spaces"][room] = {"children": children, "count": len(children)}
    for portal in result["portals"]:
        portal["message_count"] = counts[(portal["bridge_id"], portal["id"], portal["receiver"])]
        portal["state_http"] = None
        if portal["mxid"]:
            try:
                matrix(config, "rooms/" + url.quote(portal["mxid"], safe="") + "/state")
                portal["state_http"] = 200
            except urllib.error.HTTPError as error:
                if error.code not in (403, 404):
                    raise
                portal["state_http"] = error.code
    flags = ("poll_interval_seconds", "message_fetch_limit", "auto_fetch_messages", "snap_media_enabled", "snap_media_on_read", "send_media_enabled")
    result["runtime"] = {"configured_network": {k: config["network"][k] for k in flags if type(config["network"].get(k)) in (bool, int)}}
    env = RUNTIME / "local-stack.env"
    result["runtime"]["env_flags"] = dict(re.findall(
        r'^export (SNAPCHAT_(?:READ_RECEIPTS_ENABLED|SNAP_MEDIA_ENABLED|SNAP_MEDIA_ON_READ|SEND_MEDIA_ENABLED))="?(true|false)"?$',
        env.read_text() if env.exists() else "", re.M))
    for name in ("bridge", "connector"):
        path = RUNTIME / (name + ".pid")
        pid = path.read_text().strip() if path.exists() else ""
        result["runtime"][name + "_tracked_pid"] = int(pid) if pid.isdecimal() else None
    result["counts"] = {key: len(result[key]) for key in ("portals", "messages", "state_messages", "logins", "spaces")}
    result["finished_at"] = time.time()
    return result

def compare(before, after):
    if before["version"] != after["version"]:
        raise ValueError("Snapshot version mismatch")
    result = {"counts_before": before["counts"], "counts_after": after["counts"]}
    for name, keys in (("portals", ("bridge_id", "id", "receiver")),
                       ("messages", ("bridge_id", "room_receiver", "id", "part_id")),
                       ("logins", ("bridge_id", "id"))):
        old, new = ({tuple(row[k] for k in keys): row for row in data[name]} for data in (before, after))
        result[name] = {"removed": sorted(old.keys() - new.keys()), "added": sorted(new.keys() - old.keys()),
                        "changed": sorted(k for k in old.keys() & new.keys() if old[k] != new[k])}
        if name == "portals":
            result["rows_preserved"] = all(k in new and row["rowid"] == new[k]["rowid"] for k, row in old.items())
            result["healthy_mxids_preserved"] = all(k in new and row["mxid"] == new[k]["mxid"] and new[k]["state_http"] == 200
                                                   for k, row in old.items() if row["state_http"] == 200)
            result["mapping_counts_not_decreased"] = all(k in new and row["message_count"] <= new[k]["message_count"] for k, row in old.items())
    result["spaces"] = {room: {"removed": sorted(set(before["spaces"].get(room, {}).get("children", [])) - set(after["spaces"].get(room, {}).get("children", []))),
                                "added": sorted(set(after["spaces"].get(room, {}).get("children", [])) - set(before["spaces"].get(room, {}).get("children", [])))}
                        for room in sorted(before["spaces"].keys() | after["spaces"].keys())}
    result["space_ids_preserved"] = before["spaces"].keys() <= after["spaces"].keys()
    result["portal_count_preserved"] = before["counts"]["portals"] == after["counts"]["portals"]
    result["ok"] = all(result[k] for k in ("rows_preserved", "healthy_mxids_preserved", "mapping_counts_not_decreased", "space_ids_preserved")) and not result["messages"]["removed"] and not result["logins"]["removed"]
    return result

def report(since):
    with LOG.open("rb") as stream:
        end = os.fstat(stream.fileno()).st_size
        if since < 0 or since > end:
            raise ValueError("Invalid offset or truncated log")
        start = max(since, end - 4 * 1024 * 1024)
        stream.seek(max(0, start - 1))
        if start and stream.read(1) != b"\n":
            stream.readline(4 * 1024 * 1024)
        data = stream.read(max(0, end - stream.tell()))
    summaries, media, errors = [], collections.Counter(), collections.Counter()
    for line in data.splitlines():
        try:
            event = json.loads(line)
        except (ValueError, UnicodeError):
            continue
        if not isinstance(event, dict):
            continue
        if event.get("level") == "error" or event.get("message") == "media: incoming attachment unavailable":
            errors[event.get("message", "unknown")] += 1
        summary = event.get("resync_summary", "")
        if isinstance(summary, str) and summary:
            summaries.append({k: int(v) for k, v in re.findall(r"^([^\n:]+): (\d+)$", summary, re.M) if k in METRICS})
        for evidence in ("media: SKIP current-room-delivery", "media: skipping already hydrated attachment",
                         "media: uploaded Snapchat media", "media: downloaded incoming attachment",
                         "media: failed to upload Snapchat media", "media: incoming attachment unavailable"):
            if event.get("message") == evidence:
                media[evidence] += 1
    return {"start_byte": start, "end_byte": end, "truncated": start > since, "resync_summaries": summaries, "media_evidence_counts": media, "error_counts": errors}

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--config", default="~/.local/share/bbctl/bridges/sh-snapchat/config.yaml")
    commands = parser.add_subparsers(dest="command", required=True)
    commands.add_parser("snapshot").add_argument("--output", required=True, help="JSON path under data/runtime (atomic replace)")
    commands.add_parser("compare").add_argument("baseline", type=Path)
    commands.add_parser("resync").add_argument("chat_id", nargs="?", default="")
    commands.add_parser("report").add_argument("--since-byte", type=int, default=0)
    commands.add_parser("media", help="List a small sample of persisted media mappings across chats")
    commands.add_parser("chat", help="Inspect recent stored message status without displaying message text").add_argument("name")
    args = parser.parse_args()
    if args.command == "report":
        return report(args.since_byte)
    import yaml
    config = yaml.safe_load(Path(args.config).expanduser().read_text())
    if args.command == "chat":
        with database(config["database"]["uri"]) as db:
            portals = [dict(row) for row in db.execute("SELECT id,name,mxid,receiver FROM portal WHERE name LIKE ?", ('%' + args.name + '%',))]
        with database(config["network"]["state_db_path"]) as db:
            for portal in portals:
                portal["recent"] = [dict(row) for row in db.execute("SELECT remote_id,kind,has_media,hydrated_at,timestamp_raw,text='[Snapchat message unavailable]' AS unavailable,length(text) AS text_length FROM message_state WHERE portal_key=? ORDER BY last_seen_at DESC LIMIT 25", (portal["id"],))]
        return {"chats": portals}
    if args.command == "media":
        with database(config["database"]["uri"]) as db:
            candidates = rows(db, "SELECT m.id,m.mxid,p.id AS chat_id,p.mxid AS room,p.name FROM message m JOIN portal p ON p.bridge_id=m.bridge_id AND p.id=m.room_id AND p.receiver=m.room_receiver WHERE json_extract(m.metadata,'$.media_delivered')=1 ORDER BY m.rowid DESC")
        selected, counts = [], collections.Counter()
        for candidate in candidates:
            if counts[candidate["chat_id"]] < 2 and (candidate["chat_id"] in counts or len(counts) < 3):
                selected.append(candidate)
                counts[candidate["chat_id"]] += 1
            if len(selected) == 5:
                break
        with database(config["network"]["state_db_path"]) as db:
            pending = rows(db, "SELECT portal_key,remote_id,kind FROM message_state WHERE kind='media' AND has_media=1 AND hydrated_at IS NULL ORDER BY last_seen_at DESC LIMIT 30")
        return {"delivered_media_count": len(candidates), "sample": selected, "pending_sample": pending}
    if args.command == "resync":
        if args.chat_id and not re.fullmatch(r"[A-Za-z0-9_-]+", args.chat_id):
            raise ValueError("Invalid chat ID")
        with database(config["database"]["uri"]) as db:
            users = rows(db, 'SELECT DISTINCT u.mxid,u.management_room FROM "user" u JOIN user_login l ON l.bridge_id=u.bridge_id AND l.user_mxid=u.mxid')
        if len(users) != 1 or not users[0]["management_room"]:
            raise ValueError("Exactly one logged-in management user required")
        offset = LOG.stat().st_size
        path = "rooms/" + url.quote(users[0]["management_room"], safe="") + "/send/m.room.message/" + uuid.uuid4().hex
        sent = matrix(config, path, users[0]["mxid"], {"msgtype": "m.text", "body": (config["bridge"]["command_prefix"] + " resync-all " + args.chat_id).strip()})
        return {"event_id": sent["event_id"], "log_start_byte": offset, "status": "sent, not completion proof"}
    if args.command == "compare":
        return compare(json.loads(args.baseline.expanduser().read_text()), snapshot(config))
    output = (ROOT / Path(args.output).expanduser()).resolve()
    if not output.is_relative_to(RUNTIME.resolve()) or not output.parent.is_dir() or output.suffix != ".json":
        raise ValueError("Output must be a JSON file inside existing data/runtime directory")
    if subprocess.run(["git", "check-ignore", "-q", "--", str(output)], cwd=ROOT).returncode != 0:
        raise ValueError("Output must be gitignored")
    result = snapshot(config)
    temporary = None
    try:
        with tempfile.NamedTemporaryFile(mode="w", dir=output.parent, delete=False) as stream:
            temporary = Path(stream.name)
            json.dump(result, stream, indent=2)
            stream.flush()
            os.fsync(stream.fileno())
        os.replace(temporary, output)
    finally:
        if temporary and temporary.exists():
            temporary.unlink()
    return {"output": str(output), "counts": result["counts"]}

if __name__ == "__main__":
    try:
        result = main()
        print(json.dumps(result, indent=2))
        sys.exit(2 if result.get("ok") is False else 0)
    except Exception as error:
        print("bridge_admin failed (details suppressed): " + type(error).__name__, file=sys.stderr)
        sys.exit(1)
