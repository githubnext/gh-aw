---
private: true
emoji: "👁️"
description: Visual regression checker that captures and compares screenshots on every pull request using Playwright
on:
  pull_request:
    types: [opened, synchronize]
    paths:
      - 'docs/package.json'
      - 'docs/package-lock.json'
      - 'docs/src/**/*.css'
      - 'docs/src/**/*.tsx'
      - 'docs/src/**/*.astro'
      - 'docs/astro.config.mjs'
permissions:
  contents: read
  pull-requests: read
engine: copilot
imports:
  - shared/otlp.md
tools:
  cli-proxy: true
  playwright:
    mode: cli
  bash:
    - "npm *"
    - "npx *"
    - "node *"
    - "curl http://localhost:*"
network:
  allowed:
    - defaults
    - playwright
    - local
    - node
safe-outputs:
  add-comment:
    max: 1
timeout-minutes: 15
steps:
  - name: Checkout repository
    uses: actions/checkout@v7.0.0
    with:
      persist-credentials: false

  - name: Setup Node.js
    uses: actions/setup-node@v7.0.0
    with:
      node-version: '24'
      cache: 'npm'
      cache-dependency-path: 'docs/package-lock.json'

  - name: Install dependencies
    working-directory: ./docs
    run: npm ci

  - name: Build documentation
    working-directory: ./docs
    run: npm run build

  - name: Start docs server
    run: |
      nohup make dev-docs > /tmp/gh-aw/agent/preview.log 2>&1 &
      PID=$!
      echo "$PID" > /tmp/gh-aw/agent/server.pid
      echo "Server PID: $PID"

  - name: Wait for server readiness
    run: |
      MAX_WAIT=90
      WAITED=0
      until (echo > /dev/tcp/127.0.0.1/4321) > /dev/null 2>&1; do
        if [ -f /tmp/gh-aw/agent/server.pid ] && ! kill -0 "$(cat /tmp/gh-aw/agent/server.pid)" 2>/dev/null; then
          echo "Docs server process exited before opening port 4321" >&2
          cat /tmp/gh-aw/agent/preview.log >&2
          exit 1
        fi
        WAITED=$((WAITED + 3))
        if [ $WAITED -ge $MAX_WAIT ]; then
          echo "Docs server port 4321 did not open in ${MAX_WAIT}s" >&2
          cat /tmp/gh-aw/agent/preview.log >&2
          exit 1
        fi
        echo "Waiting for docs port... ($WAITED/${MAX_WAIT}s)"
        sleep 3
      done
      WAITED=0
      until curl -sf http://localhost:4321/gh-aw/ > /dev/null 2>&1; do
        WAITED=$((WAITED + 3))
        if [ $WAITED -ge $MAX_WAIT ]; then
          echo "Dev server did not become ready in ${MAX_WAIT}s" >&2
          cat /tmp/gh-aw/agent/preview.log >&2
          exit 1
        fi
        echo "Waiting for dev server response... ($WAITED/${MAX_WAIT}s)"
        sleep 3
      done
      echo "Dev server is ready"

---

# Visual Regression Checker

You are a visual quality agent. The workflow started the docs server and verified readiness. It is running at `http://localhost:4321/gh-aw/`. For this pull request, use playwright-cli commands in bash to capture screenshots of key pages and report any visual differences.

## Steps

1. **Capture screenshots** — Use `playwright-cli` to resize the viewport and take full-page screenshots of the key pages:
   - **Mobile**: `playwright-cli browser_resize --width 375 --height 812 && playwright-cli browser_navigate --url "http://localhost:4321/gh-aw/" && playwright-cli browser_take_screenshot --filename /tmp/gh-aw/agent/screenshot-mobile.png --full-page true`
   - **Tablet**: resize to 768 × 1024, navigate, screenshot
   - **Desktop**: resize to 1440 × 900, navigate, screenshot
2. **Accessibility snapshot** — For each page, run `playwright-cli browser_snapshot` and note any violations.
3. **Report** — Post a summary comment with:
   - A table listing each page, viewport, and screenshot status (unchanged / changed / error)
   - Any accessibility issues found

Post the summary as a pull request comment using the `add_comment` safe-output tool.
If there are no differences and no accessibility issues, call `noop` with a brief message.
