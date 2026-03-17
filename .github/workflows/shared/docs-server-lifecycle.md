---
# Documentation Server Lifecycle Management
# 
# This shared workflow provides instructions for starting, waiting for readiness,
# and cleaning up the Astro Starlight documentation dev server.
#
# Prerequisites:
# - npm install must have been run in docs/ directory
# - Bash permissions: npm *, npx *, curl *, kill *, echo *, sleep *
# - Working directory should be in repository root
---

## Starting the Documentation Preview Server

Navigate to the docs directory and start the development server in the background, binding to all network interfaces on a fixed port:

```bash
cd docs
npm run dev -- --host 0.0.0.0 --port 4321 > /tmp/preview.log 2>&1 &
echo $! > /tmp/server.pid
```

This will:
- Start the Astro development server on port 4321, bound to all interfaces (`0.0.0.0`)
- Redirect output to `/tmp/preview.log`
- Save the process ID to `/tmp/server.pid` for later cleanup

**Why `npm run dev` instead of `npm run preview`:**
The `npm run preview` command serves the pre-built static output. However, Astro's Starlight documentation site uses hybrid routing which requires the development server (`astro dev`) to correctly serve all pages at the `/gh-aw/` base URL. Using `npm run preview` returns 404 for `/gh-aw/` paths.

**Why `--host 0.0.0.0 --port 4321` is required:**
The agent runs inside a Docker container. Playwright also runs in its own container with `--network host`, meaning its `localhost` is the Docker host — not the agent container. Binding to `0.0.0.0` makes the server accessible on the agent container's bridge IP (e.g. `172.30.x.x`). The `--port 4321` flag prevents port conflicts if a previous server instance is still shutting down.

## Waiting for Server Readiness

Poll the server with curl until the `/gh-aw/` path returns HTTP 200:

```bash
for i in {1..45}; do
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:4321/gh-aw/)
  [ "$STATUS" = "200" ] && echo "Server ready at http://localhost:4321/gh-aw/!" && break
  echo "Waiting for server... ($i/45) (status: $STATUS)" && sleep 3
done
```

This will:
- Attempt to connect up to 45 times (135 seconds total) to allow for Astro dev server startup
- Check that `/gh-aw/` specifically returns HTTP 200 (not just that the port is open)
- Wait 3 seconds between attempts
- Exit successfully when the docs site is fully accessible

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
- No `npm run build` step is required before starting the dev server
