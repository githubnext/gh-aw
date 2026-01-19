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

Navigate to the docs directory and start the preview server in the background:

```bash
cd docs
npm run preview > /tmp/preview.log 2>&1 &
echo $! > /tmp/server.pid
```

This will:
- Start the preview server on port 4321
- Redirect output to `/tmp/preview.log`
- Save the process ID to `/tmp/server.pid` for later cleanup

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

- The server runs on `http://localhost:4321`
- Documentation is accessible at `http://localhost:4321/gh-aw/`
- Always clean up the server when done to avoid orphan processes
- If the server fails to start, check `/tmp/preview.log` for errors
