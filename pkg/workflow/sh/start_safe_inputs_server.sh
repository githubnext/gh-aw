# Ensure logs directory exists
mkdir -p /tmp/gh-aw/safe-inputs/logs

# Start the HTTP server in the background
echo "Starting safe-inputs MCP HTTP server..."
node mcp-server.cjs > /tmp/gh-aw/safe-inputs/logs/server.log 2>&1 &
SERVER_PID=$!
echo "Started safe-inputs MCP server with PID $SERVER_PID"

# Give server a moment to initialize or write any startup errors
sleep 0.5

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
  if curl -s -f -H "Authorization: Bearer $GH_AW_SAFE_INPUTS_API_KEY" http://localhost:$GH_AW_SAFE_INPUTS_PORT/ > /dev/null 2>&1; then
    echo "Safe Inputs MCP server is ready (attempt $i/10)"
    break
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
