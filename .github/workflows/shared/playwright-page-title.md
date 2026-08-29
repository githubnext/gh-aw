---
import-schema:
  mode:
    type: choice
    options: [cli, mcp]
    default: cli
    description: "Playwright integration mode: `cli` (playwright-cli on the runner) or `mcp` (deprecated Docker MCP server)"
  server:
    type: choice
    options: [agent, steps]
    default: agent
    description: "Where the page under test is served from: `agent` (agent starts the HTTP server inside the sandbox) or `steps` (a pre-agent step starts it outside AWF)"
  port:
    type: number
    default: 8129
    description: "TCP port of the static HTTP server serving the page under test"
  expected-title:
    type: string
    default: "gh-aw playwright smoke"
    description: "Exact `<title>` the browser must read back from the page"

tools:
  playwright:
    mode: ${{ github.aw.import-inputs.mode }}

network:
  allowed:
    - defaults
    - node
    - playwright

pre-agent-steps:
  - name: Start static page server for Playwright title check
    env:
      PW_SERVER: "${{ github.aw.import-inputs.server }}"
      PW_PORT: "${{ github.aw.import-inputs.port }}"
      PW_TITLE: "${{ github.aw.import-inputs.expected-title }}"
    run: |
      if [ "$PW_SERVER" != "steps" ]; then
        echo "Server placement is '$PW_SERVER'; the agent starts the page server itself."
        exit 0
      fi
      mkdir -p /tmp/gh-aw/agent/pw-site
      printf '<html><head><title>%s</title></head><body>playwright smoke page</body></html>' "$PW_TITLE" > /tmp/gh-aw/agent/pw-site/index.html
      nohup python3 -m http.server "$PW_PORT" --bind 0.0.0.0 --directory /tmp/gh-aw/agent/pw-site > /tmp/gh-aw/agent/pw-site-server.log 2>&1 &
      PID=$!
      echo "$PID" > /tmp/gh-aw/agent/pw-site-server.pid
      echo "Page server PID: $PID on port $PW_PORT"
  - name: Wait for static page server readiness
    env:
      PW_SERVER: "${{ github.aw.import-inputs.server }}"
      PW_PORT: "${{ github.aw.import-inputs.port }}"
    # runner-guard:ignore RGS-012 -- loopback-only readiness probe for the static page server started in this job; no external network or secrets are involved.
    run: |
      if [ "$PW_SERVER" != "steps" ]; then
        echo "Server placement is '$PW_SERVER'; nothing to wait for."
        exit 0
      fi
      for i in $(seq 1 30); do
        if curl -sf "http://127.0.0.1:${PW_PORT}/" > /dev/null 2>&1; then
          echo "Page server ready on port ${PW_PORT}"
          exit 0
        fi
        sleep 1
      done
      echo "Page server failed to start on port ${PW_PORT}" >&2
      cat /tmp/gh-aw/agent/pw-site-server.log >&2 || true
      exit 1
---

<!--
Shared Playwright matrix probe. Reads a page <title> with a real browser so that a
single combination of (playwright mode x engine x sandbox runtime x server placement)
is exercised end to end. Import it with `with:` values, for example:

  imports:
    - uses: shared/playwright-page-title.md
      with:
        mode: cli
        server: agent
-->

## Playwright Page Title Check

Configuration for this run: **mode `${{ github.aw.import-inputs.mode }}`**, **server `${{ github.aw.import-inputs.server }}`**, **port `${{ github.aw.import-inputs.port }}`**, expected title `${{ github.aw.import-inputs.expected-title }}`.

Read the page title **with the browser only**. Never use `curl`, `wget`, `web-fetch`, or any other HTTP client to obtain the title — the whole point of this check is the browser stack.

### 1. Page under test

Follow **only** the section matching the `server` value above.

- `server: steps` — the page is already served **outside the AWF sandbox** by a pre-agent step. Do not start a server.
- `server: agent` — start the server yourself **inside the sandbox** before browsing:

  ```bash
  mkdir -p /tmp/gh-aw/agent/pw-site
  printf '<html><head><title>%s</title></head><body>playwright smoke page</body></html>' '${{ github.aw.import-inputs.expected-title }}' > /tmp/gh-aw/agent/pw-site/index.html
  nohup python3 -m http.server ${{ github.aw.import-inputs.port }} --bind 0.0.0.0 --directory /tmp/gh-aw/agent/pw-site > /tmp/gh-aw/agent/pw-site-server.log 2>&1 &
  PID=$!
  echo "Page server PID: $PID"
  ```

Browse `http://localhost:${{ github.aw.import-inputs.port }}/` first. If navigation fails, retry once with `http://host.docker.internal:${{ github.aw.import-inputs.port }}/` and report which host worked — the reachable host differs per sandbox runtime, and that result is the signal this check exists to collect.

### 2. Read the title

- `mode: cli` — from bash:

  ```bash
  playwright-cli browser_navigate --url "http://localhost:${{ github.aw.import-inputs.port }}/"
  playwright-cli browser_evaluate --expression "document.title"
  ```

- `mode: mcp` — call the `browser_navigate` MCP tool with the same URL, then `browser_evaluate` with expression `document.title`.

### 3. Report

Add one line to your report (issue body, comment, or noop message):

`Playwright title check (mode=${{ github.aw.import-inputs.mode }}, server=${{ github.aw.import-inputs.server }}): ✅|❌ title="<title read>" url=<url that worked>`

Mark it ✅ only when the browser returned exactly `${{ github.aw.import-inputs.expected-title }}`; otherwise mark ❌ and include the browser error. Do not fail the whole run because of this check — report the outcome and continue.
