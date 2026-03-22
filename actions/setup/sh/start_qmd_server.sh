#!/usr/bin/env bash
# Start qmd MCP HTTP Server
# This script starts the qmd MCP server with HTTP transport and waits for it to become ready.
#
# qmd uses node-llama-cpp to run embedding models. On first use the llama.cpp binary must be
# downloaded, which can take several minutes. The health probe loop below accounts for this
# by waiting up to 10 minutes before giving up.
#
# Required environment variables:
#   GH_AW_QMD_PORT     - Port to listen on (e.g. 3002)
#   QMD_CACHE_DIR      - Path to the pre-built qmd index (e.g. /tmp/gh-aw/qmd-index)
#
# Optional environment variables:
#   NODE_LLAMA_CPP_GPU - Set to "false" to disable GPU probing (default: "false")

set -e

echo "Starting qmd MCP HTTP server..."
echo "  Port:      ${GH_AW_QMD_PORT}"
echo "  Cache dir: ${QMD_CACHE_DIR}"
echo "  GPU:       ${NODE_LLAMA_CPP_GPU:-auto}"

# Ensure logs directory exists
mkdir -p /tmp/gh-aw/mcp-logs/qmd

# Create initial log file for artifact upload
{
  echo "qmd MCP HTTP Server Log"
  echo "Start time: $(date)"
  echo "==========================================="
  echo ""
} > /tmp/gh-aw/mcp-logs/qmd/server.log

# Start the qmd MCP server with HTTP transport in the background.
# QMD_CACHE_DIR tells qmd where the pre-built vector index lives.
# NODE_LLAMA_CPP_GPU controls GPU probing; "false" disables it on CPU runners.
QMD_CACHE_DIR="${QMD_CACHE_DIR}" \
NODE_LLAMA_CPP_GPU="${NODE_LLAMA_CPP_GPU:-false}" \
  npx --package=@tobilu/qmd serve-mcp --http --port "${GH_AW_QMD_PORT}" \
    >> /tmp/gh-aw/mcp-logs/qmd/server.log 2>&1 &

SERVER_PID=$!
echo "Started qmd MCP server with PID ${SERVER_PID}"

# Wait for the server to become ready.
# A long timeout (10 minutes = 600 seconds) is used because node-llama-cpp may download
# llama.cpp binaries and embedding model weights on the first run.
TIMEOUT_SECONDS=600
RETRY_DELAY=1
MAX_ATTEMPTS=$((TIMEOUT_SECONDS / RETRY_DELAY))
ATTEMPT=0
HEALTH_START=$(date +%s%3N)

echo "Waiting for qmd MCP server to become ready (timeout: ${TIMEOUT_SECONDS}s)..."

while [ "${ATTEMPT}" -lt "${MAX_ATTEMPTS}" ]; do
  ATTEMPT=$((ATTEMPT + 1))

  # Abort if the server process has already exited
  if ! kill -0 "${SERVER_PID}" 2>/dev/null; then
    echo "ERROR: qmd MCP server process (PID ${SERVER_PID}) has exited unexpectedly"
    echo "=== Server log ==="
    cat /tmp/gh-aw/mcp-logs/qmd/server.log
    exit 1
  fi

  # Poll the health endpoint
  if curl -s -f --max-time 2 --connect-timeout 1 \
      "http://localhost:${GH_AW_QMD_PORT}/health" > /dev/null 2>&1; then
    ELAPSED_MS=$(( $(date +%s%3N) - HEALTH_START ))
    echo "qmd MCP server is ready after ${ELAPSED_MS}ms (attempt ${ATTEMPT}/${MAX_ATTEMPTS})"
    echo ""
    echo "::group::qmd server startup log"
    cat /tmp/gh-aw/mcp-logs/qmd/server.log
    echo "::endgroup::"
    break
  fi

  # Log progress every 30 seconds
  if [ $(( ATTEMPT % 30 )) -eq 0 ]; then
    ELAPSED_SEC=$(( ($(date +%s%3N) - HEALTH_START) / 1000 ))
    echo "Still waiting... (attempt ${ATTEMPT}/${MAX_ATTEMPTS}, ${ELAPSED_SEC}s elapsed)"
    # Show last few log lines for diagnostics
    tail -5 /tmp/gh-aw/mcp-logs/qmd/server.log 2>/dev/null || true
  fi

  if [ "${ATTEMPT}" -eq "${MAX_ATTEMPTS}" ]; then
    echo "ERROR: qmd MCP server failed to respond within ${TIMEOUT_SECONDS}s"
    echo "Last HTTP check on http://localhost:${GH_AW_QMD_PORT}/health failed."
    echo ""
    echo "=== Server log (full) ==="
    cat /tmp/gh-aw/mcp-logs/qmd/server.log
    echo ""
    echo "Checking port availability:"
    ss -tuln 2>/dev/null | grep "${GH_AW_QMD_PORT}" || \
      netstat -tuln 2>/dev/null | grep "${GH_AW_QMD_PORT}" || \
      echo "Port ${GH_AW_QMD_PORT} not listed (ss/netstat not available)"
    exit 1
  fi

  sleep "${RETRY_DELAY}"
done

# Write the port to GITHUB_OUTPUT so downstream steps can reference it
{
  echo "port=${GH_AW_QMD_PORT}"
} >> "${GITHUB_OUTPUT}"

echo "qmd MCP server started successfully on port ${GH_AW_QMD_PORT}"
