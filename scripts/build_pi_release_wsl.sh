#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
release_root="${repo_dir}/build/pi-release"
stage_dir="${release_root}/snapchat-bridge-pi"
archive_path="${release_root}/snapchat-bridge-pi-arm64.tar.gz"
binary_path="${stage_dir}/bin/mautrix-snapchat-bridgev2-linux-arm64"

rm -rf "${stage_dir}"
mkdir -p "${stage_dir}/bin" "${stage_dir}/connector" "${stage_dir}/scripts" "${stage_dir}/.codex-build"

cd "${repo_dir}"
GOWORK=off GOOS=linux GOARCH=arm64 CGO_ENABLED=1 CC="${CC:-aarch64-linux-gnu-gcc}" \
    go build -o "${binary_path}" ./cmd/mautrix-snapchat-bridgev2
chmod +x "${binary_path}"

cp -a connector/src "${stage_dir}/connector/"
cp connector/package.json "${stage_dir}/connector/package.json"
if [[ -f connector/package-lock.json ]]; then
    cp connector/package-lock.json "${stage_dir}/connector/package-lock.json"
fi

cp scripts/run_bridge_stack_pi.sh "${stage_dir}/scripts/run_bridge_stack_pi.sh"
cp scripts/ensure_beeper_snapchat_registration_wsl.sh "${stage_dir}/scripts/ensure_beeper_snapchat_registration_wsl.sh"
chmod +x "${stage_dir}/scripts/run_bridge_stack_pi.sh" "${stage_dir}/scripts/ensure_beeper_snapchat_registration_wsl.sh"

if [[ -d .codex-build/snapcap-native/dist && -f .codex-build/snapcap-native/package.json ]]; then
    mkdir -p "${stage_dir}/.codex-build/snapcap-native"
    cp .codex-build/snapcap-native/package.json "${stage_dir}/.codex-build/snapcap-native/package.json"
    cp -a .codex-build/snapcap-native/dist "${stage_dir}/.codex-build/snapcap-native/"
    if [[ -d .codex-build/snapcap-native/vendor ]]; then
        cp -a .codex-build/snapcap-native/vendor "${stage_dir}/.codex-build/snapcap-native/"
    fi
fi

cat > "${stage_dir}/run.sh" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
repo_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
exec "${repo_dir}/scripts/run_bridge_stack_pi.sh" "$@"
SH
chmod +x "${stage_dir}/run.sh"

cat > "${stage_dir}/README-PI.md" <<'MD'
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
MD

tar -C "${release_root}" -czf "${archive_path}" snapchat-bridge-pi
echo "${archive_path}"
