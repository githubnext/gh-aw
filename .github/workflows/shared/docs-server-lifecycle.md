---
# Documentation Server Lifecycle Management
# 
# This shared workflow provides instructions for starting, waiting for readiness,
# and cleaning up the Astro Starlight documentation preview server.
#
# Prerequisites:
# - Documentation must be built first (npm run build in docs/ directory)
# - Bash permissions: npm *, curl *, kill *, echo *, sleep *
# - Working directory should be in repository root
---

## Starting the Documentation Preview Server

**Context**: The documentation has been pre-built using `npm run build`. Use the preview server to serve the static build.

Navigate to the docs directory and start the preview server in the background, binding to all network interfaces on a fixed port:

```bash
cd docs
npm run preview -- --host 0.0.0.0 --port 4321 > /tmp/preview.log 2>&1 &
echo $! > /tmp/server.pid
```

This will:
- Start the preview server on port 4321, bound to all interfaces (`0.0.0.0`)
- Redirect output to `/tmp/preview.log`
- Save the process ID to `/tmp/server.pid` for later cleanup

**Why `--host 0.0.0.0 --port 4321` is required:**
The agent runs inside a Docker container. Playwright also runs in its own container with `--network host`, meaning its `localhost` is the Docker host — not the agent container. Binding to `0.0.0.0` makes the server accessible on the agent container's bridge IP (e.g. `172.30.x.x`). The `--port 4321` flag prevents port conflicts if a previous server instance is still shutting down.

## Waiting for Server Readiness

Poll the server with curl to ensure it's ready before use:

```bash
for i in {1..30}; do
  curl -s http://localhost:4321 > /dev/null && echo "Server ready!" && break
  echo "Waiting for server... ($i/30)" && sleep 2
done
```

This will:
- Attempt to connect up to 30 times (60 seconds total)
- Wait 2 seconds between attempts
- Exit successfully when server responds

## Playwright Browser Access

**Important**: Playwright runs in a container with `--network host`, so its `localhost` is the Docker host's localhost — not the agent container. To access the docs server from Playwright browser tools, use the agent container's bridge network IP instead of `localhost`.

Get the container's bridge IP (this uses route lookup — `1.1.1.1` is never actually contacted, it only determines which interface handles outbound traffic):

```bash
SERVER_IP=$(ip -4 route get 1.1.1.1 2>/dev/null | awk '{print $7; exit}')
# Fallback if route lookup fails
if [ -z "$SERVER_IP" ]; then
  SERVER_IP=$(hostname -I | awk '{print $1}')
fi
echo "Playwright server URL: http://${SERVER_IP}:4321/gh-aw/"
```

Then use `http://${SERVER_IP}:4321/gh-aw/` (not `http://localhost:4321/gh-aw/`) when navigating with Playwright tools.

The `curl` readiness check and bash commands still use `localhost:4321` since they run inside the agent container where the server is local.

## Verifying Server Accessibility (Optional)

Optionally verify the server is serving content:

```bash
curl -s http://localhost:4321/gh-aw/ | head -20
```

## Stopping the Documentation Server

After you're done using the server, clean up the process:

```bash
kill $(cat /tmp/server.pid) 2>/dev/null || true
rm -f /tmp/server.pid /tmp/preview.log
```

This will:
- Kill the server process using the saved PID
- Remove temporary files
- Ignore errors if the process already stopped

## Usage Notes

- The server runs on `http://localhost:4321` (agent container's localhost)
- Documentation is accessible at `http://localhost:4321/gh-aw/` for curl/bash
- For Playwright browser tools, use the container bridge IP (see "Playwright Browser Access" section above)
- Always clean up the server when done to avoid orphan processes
- If the server fails to start, check `/tmp/preview.log` for errors
