cd /tmp/gh-aw/safe-inputs

# Verify required files exist
echo "Verifying safe-inputs setup..."
if [ ! -f mcp-server.cjs ]; then
  echo "ERROR: mcp-server.cjs not found in /tmp/gh-aw/safe-inputs"
  ls -la /tmp/gh-aw/safe-inputs/
  exit 1
fi
if [ ! -f tools.json ]; then
  echo "ERROR: tools.json not found in /tmp/gh-aw/safe-inputs"
  ls -la /tmp/gh-aw/safe-inputs/
  exit 1
fi
echo "Configuration files verified"

# Log environment configuration
echo "Server configuration:"
echo "  Port: $GH_AW_SAFE_INPUTS_PORT"
echo "  API Key: ${GH_AW_SAFE_INPUTS_API_KEY:0:8}..."
echo "  Working directory: $(pwd)"

# Ensure logs directory exists
mkdir -p /tmp/gh-aw/safe-inputs/logs

# Start the HTTP server in the background
echo "Starting safe-inputs MCP HTTP server..."
node mcp-server.cjs > /tmp/gh-aw/safe-inputs/logs/server.log 2>&1 &
SERVER_PID=$!
echo "Started safe-inputs MCP server with PID $SERVER_PID"

# Wait for server to be ready (max 10 seconds)
echo "Waiting for server to become ready..."
for i in {1..10}; do
  # Check if process is still running
  if ! kill -0 $SERVER_PID 2>/dev/null; then
    echo "ERROR: Server process $SERVER_PID has died"
    echo "Server log contents:"
    cat /tmp/gh-aw/safe-inputs/logs/server.log
    exit 1
  fi
  
  # Check if server is responding
  CURL_OUTPUT=$(mktemp)
  CURL_HTTP_CODE=$(curl -s -w "%{http_code}" -H "Authorization: Bearer $GH_AW_SAFE_INPUTS_API_KEY" http://localhost:$GH_AW_SAFE_INPUTS_PORT/ -o "$CURL_OUTPUT" 2>&1)
  CURL_EXIT_CODE=$?
  
  if [ $CURL_EXIT_CODE -eq 0 ] && [ "$CURL_HTTP_CODE" = "200" ]; then
    echo "Safe Inputs MCP server is ready (attempt $i/10)"
    rm -f "$CURL_OUTPUT"
    break
  else
    # Log detailed failure information
    echo "Server check failed (attempt $i/10):"
    echo "  - curl exit code: $CURL_EXIT_CODE"
    echo "  - HTTP status code: $CURL_HTTP_CODE"
    if [ -f "$CURL_OUTPUT" ] && [ -s "$CURL_OUTPUT" ]; then
      echo "  - Response content:"
      head -20 "$CURL_OUTPUT"
    else
      echo "  - Response content: (empty)"
    fi
    rm -f "$CURL_OUTPUT"
    # Show server log for additional debugging context
    echo "  - Server log:"
    cat /tmp/gh-aw/safe-inputs/logs/server.log 2>/dev/null || echo "(log not available)"
  fi
  
  if [ $i -eq 10 ]; then
    echo "ERROR: Safe Inputs MCP server failed to start after 10 seconds"
    echo "Process status: $(ps aux | grep '[m]cp-server.cjs' || echo 'not running')"
    echo "Server log contents:"
    cat /tmp/gh-aw/safe-inputs/logs/server.log
    echo "Checking port availability:"
    netstat -tuln | grep $GH_AW_SAFE_INPUTS_PORT || echo "Port $GH_AW_SAFE_INPUTS_PORT not listening"
    exit 1
  fi
  
  echo "Waiting for server... (attempt $i/10)"
  sleep 1
done

# Output the configuration for the MCP client
echo "port=$GH_AW_SAFE_INPUTS_PORT" >> $GITHUB_OUTPUT
echo "api_key=$GH_AW_SAFE_INPUTS_API_KEY" >> $GITHUB_OUTPUT
