#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$(mktemp -d)"
trap 'rm -rf "$TEST_DIR"' EXIT

mkdir -p "$TEST_DIR/mcp-config" "$TEST_DIR/home" "$TEST_DIR/workspace" "$TEST_DIR/bin"
export HOME="$TEST_DIR/home"
export GITHUB_WORKSPACE="$TEST_DIR/workspace"
export MCP_GATEWAY_OUTPUT="$TEST_DIR/gateway-output.json"
export MCP_GATEWAY_DOMAIN="localhost"
export MCP_GATEWAY_HOST_DOMAIN="localhost"
export MCP_GATEWAY_PORT="8080"

cat > "$MCP_GATEWAY_OUTPUT" <<'EOF'
{"mcpServers":{"safeinputs":{"url":"http://localhost:8080/mcp/safeinputs"},"safeoutputs":{"url":"http://localhost:8080/mcp/safeoutputs"},"github":{"url":"http://localhost:8080/mcp/github","gateway":true,"headers":{"Authorization":"custom-gateway-credential","X-Api-Key":"parsed\u002dheader\u002dcredential"}}}}
EOF

assert_private_log() {
  if grep -Eq 'custom-gateway-credential|parsed-header-credential|Authorization|X-Api-Key' "$TEST_DIR/output.log"; then
    echo "FAIL: Credential-bearing configuration reached the log" >&2
    exit 1
  fi
}

for engine in claude codex copilot gemini; do
  # Isolate the converters' fixed output paths from other users and processes.
  sed "s|/tmp/gh-aw|$TEST_DIR|g" "$SCRIPT_DIR/convert_gateway_config_${engine}.sh" > "$TEST_DIR/converter.sh"
  bash -x "$TEST_DIR/converter.sh" > "$TEST_DIR/output.log" 2>&1
  assert_private_log
  case "$engine" in
    claude) output="$TEST_DIR/mcp-config/mcp-servers.json" ;;
    codex) output="$TEST_DIR/mcp-config/config.toml" ;;
    copilot) output="$HOME/.copilot/mcp-config.json" ;;
    gemini) output="$GITHUB_WORKSPACE/.gemini/settings.json" ;;
  esac
  grep -q 'custom-gateway-credential' "$output"
done

# Exercise the unknown-engine fallback without starting a real gateway.
cp "$MCP_GATEWAY_OUTPUT" "$TEST_DIR/mcp-config/gateway-output.json"
sed -n '/^case "\$ENGINE_TYPE" in$/,/^esac$/p' "$SCRIPT_DIR/start_mcp_gateway.sh" |
  sed "s|/tmp/gh-aw|$TEST_DIR|g" > "$TEST_DIR/fallback.sh"
(
  ENGINE_TYPE="unknown"
  GH_AW_MCP_CLI_SERVERS='[]'
  require_home_for_copilot_config() { :; }
  source "$TEST_DIR/fallback.sh"
) > "$TEST_DIR/output.log" 2>&1
assert_private_log
grep -q 'custom-gateway-credential' "$HOME/.copilot/mcp-config.json"

# Stub network and timing calls so health verification stays local and portable.
cat > "$TEST_DIR/bin/curl" <<'EOF'
#!/usr/bin/env bash
printf '{"specVersion":"1","gatewayVersion":"test"}\n200\n'
EOF
cat > "$TEST_DIR/bin/date" <<'EOF'
#!/usr/bin/env bash
echo 1000
EOF
cat > "$TEST_DIR/bin/head" <<'EOF'
#!/usr/bin/env bash
if [ "${1:-}" = "-n" ] && [ "${2:-}" = "-1" ]; then
  sed '$d'
else
  command /usr/bin/head "$@"
fi
EOF
chmod +x "$TEST_DIR/bin/curl" "$TEST_DIR/bin/date" "$TEST_DIR/bin/head"
PATH="$TEST_DIR/bin:$PATH" bash -x "$SCRIPT_DIR/verify_mcp_gateway_health.sh" \
  "http://localhost:8080" "$MCP_GATEWAY_OUTPUT" "$TEST_DIR/logs" > "$TEST_DIR/output.log" 2>&1
assert_private_log
grep -q 'MCP server connectivity test passed' "$TEST_DIR/output.log"

for script in start_mcp_gateway.sh verify_mcp_gateway_health.sh; do
  # The startup script also handles secrets before its converter dispatch.
  [ "$(sed -n '2p' "$SCRIPT_DIR/$script")" = 'set +x' ]
done

echo "PASS: Gateway configuration stays private in converter, fallback, and health-check logs"
