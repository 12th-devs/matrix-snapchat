#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
config_path="${SNAPCHAT_BRIDGE_CONFIG:-${HOME}/.local/share/bbctl/bridges/sh-snapchat/config.yaml}"
bridge_binary="${SNAPCHAT_BRIDGE_BINARY:-${repo_dir}/bin/mautrix-snapchat-bridgev2-wsl-amd64}"
mode="${1:-full}"
runtime_dir="${SNAPCHAT_BRIDGE_RUNTIME_DIR:-${repo_dir}/data/runtime}"
runtime_env="${SNAPCHAT_BRIDGE_RUNTIME_ENV:-${runtime_dir}/local-stack.env}"
connector_pid_file="${SNAPCHAT_CONNECTOR_PID_FILE:-${runtime_dir}/connector.pid}"
bridge_pid_file="${SNAPCHAT_BRIDGE_PID_FILE:-${runtime_dir}/bridge.pid}"
connector_log="${SNAPCHAT_CONNECTOR_LOG:-${repo_dir}/logs/connector.log}"
connector_stdout="${SNAPCHAT_CONNECTOR_STDOUT:-${repo_dir}/logs/connector.out.log}"
connector_stderr="${SNAPCHAT_CONNECTOR_STDERR:-${repo_dir}/logs/connector.err.log}"
bridge_log="${SNAPCHAT_BRIDGE_LOG:-${repo_dir}/logs/bridge.log}"
startup_timeout_seconds="${SNAPCHAT_CONNECTOR_STARTUP_TIMEOUT_SECONDS:-180}"

usage() {
    cat >&2 <<EOF
Usage: $0 [full|connector-only|bridge-only|stop]

  full         Start a fresh local stack: Windows connector/Chrome, then WSL bridge.
  bridge-only Rebuild/restart only the WSL bridge using the existing connector session.
  stop         Stop the bridge and connector processes tracked by this runner.
EOF
}

require_paths() {
    if [[ ! -f "${config_path}" ]]; then
        echo "Beeper bridge config not found at ${config_path}" >&2
        exit 1
    fi
    if [[ ! -x "${bridge_binary}" ]]; then
        echo "WSL bridge binary not found at ${bridge_binary}" >&2
        exit 1
    fi
}

configure_bridge_runtime() {
    python3 - "${config_path}" <<'PY'
import re
import sys
from pathlib import Path

path = Path(sys.argv[1])
text = path.read_text()
replacements = {
    "poll_interval_seconds": "2",
    "message_fetch_limit": "20",
    "api_mode": "api_only",
    "dom_fallback_enabled": "false",
    "auto_fetch_messages": "true",
    "delete_outbound_on_ack": "false",
    "disable_device_change_key_rotation": "false",
}
for key, value in replacements.items():
    text = re.sub(rf"^(\s*{re.escape(key)}\s*:\s*).*$", rf"\g<1>{value}", text, flags=re.MULTILINE)
path.write_text(text)
PY
}

reset_outbound_megolm_sessions() {
    if [[ "${SNAPCHAT_BRIDGE_RESET_OUTBOUND_MEGOLM_ON_START:-true}" != "true" ]]; then
        return
    fi
    python3 - "${config_path}" "${repo_dir}" <<'PY'
import re
import sqlite3
import sys
from pathlib import Path
from urllib.parse import unquote, urlparse

config_path = Path(sys.argv[1])
repo_dir = Path(sys.argv[2])
text = config_path.read_text()
match = re.search(r"(?m)^\s*uri:\s*(.+?)\s*$", text)
if not match:
    sys.exit(0)

uri = match.group(1).strip().strip('"').strip("'")
if uri.startswith("file:"):
    parsed = urlparse(uri)
    db_value = unquote(parsed.path or parsed.netloc)
else:
    db_value = uri.split("?", 1)[0]
if not db_value or re.match(r"^[a-z][a-z0-9+.-]*://", db_value, re.I):
    sys.exit(0)

db_path = Path(db_value)
if not db_path.is_absolute():
    db_path = repo_dir / db_path
if not db_path.exists():
    sys.exit(0)

with sqlite3.connect(str(db_path)) as db:
    present = db.execute(
        "SELECT 1 FROM sqlite_master WHERE type='table' AND name='crypto_megolm_outbound_session'",
    ).fetchone()
    if not present:
        sys.exit(0)
    deleted = db.execute("DELETE FROM crypto_megolm_outbound_session").rowcount
    db.commit()
print(f"Reset outbound Megolm sessions in {db_path} rows={deleted}")
PY
}

windows_gateway() {
    ip route show default | awk '{ print $3; exit }'
}

write_runtime_env() {
    local runtime_secret windows_host
    runtime_secret="$(python3 -c 'import secrets; print(secrets.token_urlsafe(48))')"
    windows_host="${SNAPCHAT_CONNECTOR_HOST:-$(windows_gateway)}"
    if [[ -z "${windows_host}" ]]; then
        echo "Could not determine the Windows host gateway from WSL" >&2
        exit 1
    fi

    mkdir -p "${runtime_dir}" "${repo_dir}/logs"
    umask 077
    cat >"${runtime_env}" <<EOF
export SNAPCHAT_BRIDGE_CONFIG="${config_path}"
export SNAPCHAT_SHARED_SECRET="${runtime_secret}"
export SNAPCHAT_CONNECTOR_SHARED_SECRET="${runtime_secret}"
export SNAPCHAT_CONNECTOR_BASE_URL="${SNAPCHAT_CONNECTOR_BASE_URL:-http://${windows_host}:3101}"
export SNAPCHAT_READ_RECEIPTS_ENABLED="${SNAPCHAT_READ_RECEIPTS_ENABLED:-true}"
export SNAPCHAT_SNAP_MEDIA_ENABLED="${SNAPCHAT_SNAP_MEDIA_ENABLED:-true}"
export SNAPCHAT_SNAP_MEDIA_ON_READ=false
export SNAPCHAT_SEND_MEDIA_ENABLED="${SNAPCHAT_SEND_MEDIA_ENABLED:-true}"
EOF
    chmod 600 "${runtime_env}"
}

write_runtime_env_values() {
    local runtime_secret="$1" connector_base_url="$2"
    mkdir -p "${runtime_dir}" "${repo_dir}/logs"
    umask 077
    cat >"${runtime_env}" <<EOF
export SNAPCHAT_BRIDGE_CONFIG="${SNAPCHAT_BRIDGE_CONFIG:-${config_path}}"
export SNAPCHAT_SHARED_SECRET="${runtime_secret}"
export SNAPCHAT_CONNECTOR_SHARED_SECRET="${runtime_secret}"
export SNAPCHAT_CONNECTOR_BASE_URL="${connector_base_url}"
export SNAPCHAT_READ_RECEIPTS_ENABLED="${SNAPCHAT_READ_RECEIPTS_ENABLED:-false}"
export SNAPCHAT_SNAP_MEDIA_ENABLED=true
export SNAPCHAT_SNAP_MEDIA_ON_READ=false
export SNAPCHAT_SEND_MEDIA_ENABLED="${SNAPCHAT_SEND_MEDIA_ENABLED:-true}"
EOF
    chmod 600 "${runtime_env}"
}

# Caller-overridable runtime variables. Session state (shared secrets,
# connector base URL) stays env-file-authoritative because it must match the
# running connector; feature flags follow the precedence
# explicit caller environment > persisted env file > script default.
overridable_runtime_variables() {
    printf '%s\n' \
        SNAPCHAT_BRIDGE_CONFIG \
        SNAPCHAT_READ_RECEIPTS_ENABLED \
        SNAPCHAT_SNAP_MEDIA_ENABLED \
        SNAPCHAT_SNAP_MEDIA_ON_READ \
        SNAPCHAT_SEND_MEDIA_ENABLED
}

# caller_runtime_overrides emits NAME=value pairs for overridable variables
# that the caller explicitly placed in the environment.
caller_runtime_overrides() {
    local variable
    while IFS= read -r variable; do
        if [[ -n "${!variable+x}" ]]; then
            printf '%s=%s\n' "${variable}" "${!variable}"
        fi
    done < <(overridable_runtime_variables)
}

load_runtime_env() {
    if [[ ! -f "${runtime_env}" ]]; then
        echo "Runtime env file not found at ${runtime_env}" >&2
        echo "Run '$0 full' first to create a connector session, or restart the full stack." >&2
        exit 1
    fi
    local caller_overrides
    caller_overrides="$(caller_runtime_overrides)"
    # shellcheck disable=SC1090
    source "${runtime_env}"
    if [[ -n "${caller_overrides}" ]]; then
        # Restore explicit caller overrides clobbered by sourcing the file.
        while IFS= read -r entry; do
            export "${entry?}"
        done <<<"${caller_overrides}"
    fi
    if [[ -z "${SNAPCHAT_SHARED_SECRET:-}" || -z "${SNAPCHAT_CONNECTOR_BASE_URL:-}" ]]; then
        echo "Runtime env file ${runtime_env} is incomplete" >&2
        exit 1
    fi
    export SNAPCHAT_BRIDGE_CONFIG SNAPCHAT_SHARED_SECRET SNAPCHAT_CONNECTOR_SHARED_SECRET SNAPCHAT_CONNECTOR_BASE_URL
    export SNAPCHAT_READ_RECEIPTS_ENABLED SNAPCHAT_SNAP_MEDIA_ENABLED SNAPCHAT_SNAP_MEDIA_ON_READ SNAPCHAT_SEND_MEDIA_ENABLED
}

sync_runtime_env_from_running_bridge() {
	if ! tracked_pid_alive "${bridge_pid_file}"; then
		return 1
	fi
	local pid proc_env runtime_secret connector_base_url value caller_overrides
	caller_overrides="$(caller_runtime_overrides)"
	pid="$(cat "${bridge_pid_file}" 2>/dev/null || true)"
	proc_env="$(tr '\0' '\n' <"/proc/${pid}/environ" 2>/dev/null || true)"
	runtime_secret="$(printf '%s\n' "${proc_env}" | sed -n 's/^SNAPCHAT_CONNECTOR_SHARED_SECRET=//p' | head -n 1)"
	if [[ -z "${runtime_secret}" ]]; then
		runtime_secret="$(printf '%s\n' "${proc_env}" | sed -n 's/^SNAPCHAT_SHARED_SECRET=//p' | head -n 1)"
	fi
	connector_base_url="$(printf '%s\n' "${proc_env}" | sed -n 's/^SNAPCHAT_CONNECTOR_BASE_URL=//p' | head -n 1)"
	if [[ -z "${runtime_secret}" || -z "${connector_base_url}" ]]; then
		return 1
	fi
	for value in SNAPCHAT_BRIDGE_CONFIG SNAPCHAT_READ_RECEIPTS_ENABLED SNAPCHAT_SNAP_MEDIA_ENABLED SNAPCHAT_SNAP_MEDIA_ON_READ SNAPCHAT_SEND_MEDIA_ENABLED; do
		if printf '%s\n' "${caller_overrides}" | grep -aqs "^${value}="; then
			continue # explicit caller override wins over the old process env
		fi
		if proc_value="$(printf '%s\n' "${proc_env}" | sed -n "s/^${value}=//p" | head -n 1)" && [[ -n "${proc_value}" ]]; then
			export "${value}=${proc_value}"
		fi
	done
	echo "Runtime env is stale; rewriting ${runtime_env} from running bridge pid ${pid}"
	write_runtime_env_values "${runtime_secret}" "${connector_base_url}"
}

tracked_pid_alive() {
    local pid_file="$1"
    [[ -f "${pid_file}" ]] || return 1
    local pid
    pid="$(cat "${pid_file}" 2>/dev/null || true)"
    [[ "${pid}" =~ ^[0-9]+$ ]] || return 1
    kill -0 "${pid}" 2>/dev/null
}

windows_pid_alive() {
    local pid_file="$1"
    [[ -f "${pid_file}" ]] || return 1
    local pid
    pid="$(cat "${pid_file}" 2>/dev/null || true)"
    [[ "${pid}" =~ ^[0-9]+$ ]] || return 1
    powershell.exe -NoProfile -Command "\$p = Get-Process -Id ${pid} -ErrorAction SilentlyContinue; if (\$p) { exit 0 } else { exit 1 }" >/dev/null 2>&1
}

stop_pid_file() {
    local pid_file="$1" label="$2"
    if ! [[ -f "${pid_file}" ]]; then
        return
    fi
    local pid
    pid="$(cat "${pid_file}" 2>/dev/null || true)"
    if [[ "${pid}" =~ ^[0-9]+$ ]] && kill -0 "${pid}" 2>/dev/null; then
        echo "Stopping ${label} pid ${pid}"
        kill "${pid}" 2>/dev/null || true
        for _ in $(seq 1 10); do
            if ! kill -0 "${pid}" 2>/dev/null; then
                break
            fi
            sleep 1
        done
        if kill -0 "${pid}" 2>/dev/null; then
            echo "${label} pid ${pid} did not stop after SIGTERM; sending SIGKILL" >&2
            kill -9 "${pid}" 2>/dev/null || true
        fi
    fi
    rm -f "${pid_file}"
}

stop_windows_pid_file() {
    local pid_file="$1" label="$2"
    if ! [[ -f "${pid_file}" ]]; then
        return
    fi
    local pid
    pid="$(cat "${pid_file}" 2>/dev/null || true)"
    if [[ "${pid}" =~ ^[0-9]+$ ]]; then
        echo "Stopping ${label} Windows pid ${pid}"
        powershell.exe -NoProfile -Command "Get-CimInstance Win32_Process | Where-Object { \$_.ParentProcessId -eq ${pid} } | ForEach-Object { Stop-Process -Id \$_.ProcessId -Force -ErrorAction SilentlyContinue }" >/dev/null 2>&1 || true
        powershell.exe -NoProfile -Command "Stop-Process -Id ${pid} -Force -ErrorAction SilentlyContinue" >/dev/null 2>&1 || true
    fi
    rm -f "${pid_file}"
}

stop_stack() {
    stop_pid_file "${bridge_pid_file}" "bridge"
    stop_windows_pid_file "${connector_pid_file}" "connector"
}

connector_health() {
    "${repo_dir}/scripts/check_connector_local_wsl.sh"
}

start_connector() {
    if windows_pid_alive "${connector_pid_file}"; then
        echo "Connector already tracked as running with Windows pid $(cat "${connector_pid_file}")"
        return
    fi

    : >"${connector_stdout}"
    : >"${connector_stderr}"
    : >"${connector_log}"
    echo "Starting detached Windows connector/Chrome"
    echo "Connector stdout: ${connector_stdout}"
    echo "Connector stderr: ${connector_stderr}"

    local ps_script="${runtime_dir}/start-connector.ps1"
    local repo_windows node_windows connector_stdout_windows connector_stderr_windows connector_pid_windows
    repo_windows="$(wslpath -w "${repo_dir}")"
    node_windows="$(wslpath -w "/mnt/c/Program Files/nodejs/node.exe")"
    connector_stdout_windows="$(wslpath -w "${connector_stdout}")"
    connector_stderr_windows="$(wslpath -w "${connector_stderr}")"
    connector_pid_windows="$(wslpath -w "${connector_pid_file}")"
    cat >"${ps_script}" <<EOF
\$ErrorActionPreference = "Stop"
\$env:PORT = "3101"
\$env:SNAPCHAT_BROWSER_EXECUTABLE_PATH = "C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe"
\$env:SNAPCHAT_PROFILE_DIR = "${repo_windows}\\data\\snapchat-profile"
\$env:SNAPCHAT_TRACE_DIR = "${repo_windows}\\data\\debug"
\$env:SNAPCHAT_HEADLESS = "${SNAPCHAT_HEADLESS:-false}"
\$env:SNAPCHAT_SAFE_NO_OPEN = "1"
\$env:SNAPCHAT_SHARED_SECRET = "${SNAPCHAT_SHARED_SECRET}"
\$env:SNAPCHAT_REALTIME_ENABLED = "${SNAPCHAT_REALTIME_ENABLED:-false}"
\$process = Start-Process -FilePath "${node_windows}" -ArgumentList "src/index.mjs" -WorkingDirectory "${repo_windows}\\connector" -RedirectStandardOutput "${connector_stdout_windows}" -RedirectStandardError "${connector_stderr_windows}" -WindowStyle Hidden -PassThru
Set-Content -LiteralPath "${connector_pid_windows}" -Value ([string]\$process.Id) -Encoding ascii
EOF
    cmd.exe /c start "" /min powershell.exe -NoProfile -ExecutionPolicy Bypass -File "$(wslpath -w "${ps_script}")" >/dev/null 2>&1 </dev/null &
    for _ in $(seq 1 50); do
        if [[ -s "${connector_pid_file}" ]]; then
            break
        fi
        sleep 0.1
    done
    local connector_pid
    connector_pid="$(tr -dc '0-9' <"${connector_pid_file}" 2>/dev/null || true)"
    if [[ -z "${connector_pid}" ]]; then
        echo "Failed to start Windows connector. PowerShell launcher did not return a pid." >&2
        exit 1
    fi
    echo "${connector_pid}" >"${connector_pid_file}"
    {
        echo "Started Windows connector pid ${connector_pid}"
        echo "stdout: ${connector_stdout}"
        echo "stderr: ${connector_stderr}"
    } >"${connector_log}"
}

wait_for_connector() {
    local deadline=$((SECONDS + startup_timeout_seconds))
    local last_error=""
    while (( SECONDS < deadline )); do
        if ! windows_pid_alive "${connector_pid_file}"; then
            echo "Snapchat connector exited during startup. Connector stdout/stderr:" >&2
            tail -80 "${connector_stdout}" >&2 || true
            tail -80 "${connector_stderr}" >&2 || true
            exit 1
        fi
        if output="$(connector_health 2>&1)"; then
            echo "${output}"
            return
        fi
        last_error="${output}"
        sleep 2
    done

    echo "Timed out after ${startup_timeout_seconds}s waiting for Snapchat connector health at ${SNAPCHAT_CONNECTOR_BASE_URL}" >&2
    echo "Last health-check error:" >&2
    printf '%s\n' "${last_error}" >&2
    echo "Last connector log lines:" >&2
    tail -120 "${connector_stdout}" >&2 || true
    tail -120 "${connector_stderr}" >&2 || true
    exit 1
}

start_bridge() {
    echo "Starting WSL bridge: ${bridge_binary}"
    echo "Bridge log: ${bridge_log}"
    rm -f "${bridge_pid_file}"

    local sh_script="${runtime_dir}/start-bridge.sh"
    cat >"${sh_script}" <<EOF
#!/usr/bin/env bash
set -euo pipefail
cd "${repo_dir}"
source "${runtime_env}"
echo "\$\$" >"${bridge_pid_file}"
echo "==== bridge start \$(date -Is) ====" >>"${bridge_log}"
exec "${bridge_binary}" -c "\${SNAPCHAT_BRIDGE_CONFIG}" >>"${bridge_log}" 2>&1
EOF
    chmod +x "${sh_script}"

    local ps_script="${runtime_dir}/start-bridge.ps1"
    cat >"${ps_script}" <<EOF
\$ErrorActionPreference = "Stop"
Start-Process -FilePath "wsl.exe" -ArgumentList @("-d", "Ubuntu", "-e", "bash", "${sh_script}") -WindowStyle Hidden
EOF
    powershell.exe -NoProfile -ExecutionPolicy Bypass -File "$(wslpath -w "${ps_script}")" >/dev/null 2>&1

    for _ in $(seq 1 50); do
        if [[ -s "${bridge_pid_file}" ]]; then
            break
        fi
        sleep 0.1
    done
    local bridge_pid
    bridge_pid="$(tr -dc '0-9' <"${bridge_pid_file}" 2>/dev/null || true)"
    if [[ -z "${bridge_pid}" ]]; then
        echo "Failed to start WSL bridge. Detached WSL launcher did not return a pid." >&2
        exit 1
    fi
    echo "${bridge_pid}" >"${bridge_pid_file}"
    echo "Bridge started in background with pid ${bridge_pid}"
}

case "${mode}" in
    full)
        require_paths
        configure_bridge_runtime
        stop_stack
        write_runtime_env
        load_runtime_env
        start_connector
        wait_for_connector
        "${repo_dir}/scripts/ensure_beeper_snapchat_registration_wsl.sh"
        cd "${repo_dir}"
        reset_outbound_megolm_sessions
        start_bridge
        ;;
    connector-only)
        require_paths
        load_runtime_env
        stop_windows_pid_file "${connector_pid_file}" "connector"
        start_connector
        wait_for_connector
        ;;
    bridge-only)
        require_paths
        configure_bridge_runtime
        load_runtime_env
        SNAPCHAT_SNAP_MEDIA_ENABLED="${SNAPCHAT_SNAP_MEDIA_ENABLED:-true}"
        SNAPCHAT_SNAP_MEDIA_ON_READ="${SNAPCHAT_SNAP_MEDIA_ON_READ:-false}"
        SNAPCHAT_SEND_MEDIA_ENABLED="${SNAPCHAT_SEND_MEDIA_ENABLED:-true}"
        export SNAPCHAT_SNAP_MEDIA_ENABLED SNAPCHAT_SNAP_MEDIA_ON_READ SNAPCHAT_SEND_MEDIA_ENABLED
        write_runtime_env_values "${SNAPCHAT_SHARED_SECRET}" "${SNAPCHAT_CONNECTOR_BASE_URL}"
        load_runtime_env
        if ! connector_health; then
            if sync_runtime_env_from_running_bridge; then
                load_runtime_env
                connector_health || {
                    echo "Connector is not healthy after syncing runtime env from the running bridge; use '$0 full' to restart connector/Chrome and regenerate the runtime secret." >&2
                    exit 1
                }
            else
                echo "Connector is not healthy and no running bridge secret could be recovered; use '$0 full' to restart connector/Chrome and regenerate the runtime secret." >&2
                exit 1
            fi
        fi
        stop_pid_file "${bridge_pid_file}" "bridge"
        cd "${repo_dir}"
        reset_outbound_megolm_sessions
        start_bridge
        ;;
    stop)
        stop_stack
        ;;
    -h|--help|help)
        usage
        ;;
    *)
        usage
        exit 2
        ;;
esac
