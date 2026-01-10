#!/usr/bin/env bash
# Stop MCP Gateway
# This script stops the MCP gateway process using the PID from the start step output

set -e

# Get PID from environment variable (passed from step output)
GATEWAY_PID="$1"

if [ -z "$GATEWAY_PID" ]; then
  echo "Gateway PID not provided"
  echo "Gateway may not have been started or PID was not captured"
  exit 0
fi

echo "Stopping MCP gateway (PID: $GATEWAY_PID)..."

# Check if process is still running
if ps -p "$GATEWAY_PID" > /dev/null 2>&1; then
  echo "Gateway process is running, sending termination signal..."
  kill "$GATEWAY_PID" 2>/dev/null || true
  
  # Wait up to 5 seconds for graceful shutdown
  for i in {1..5}; do
    if ! ps -p "$GATEWAY_PID" > /dev/null 2>&1; then
      echo "Gateway stopped successfully"
      exit 0
    fi
    sleep 1
  done
  
  # Force kill if still running
  if ps -p "$GATEWAY_PID" > /dev/null 2>&1; then
    echo "Gateway did not stop gracefully, forcing termination..."
    kill -9 "$GATEWAY_PID" 2>/dev/null || true
    sleep 1
    
    if ps -p "$GATEWAY_PID" > /dev/null 2>&1; then
      echo "Warning: Failed to stop gateway process"
      exit 1
    fi
  fi
  
  echo "Gateway stopped successfully"
else
  echo "Gateway process (PID: $GATEWAY_PID) is not running"
fi

exit 0
