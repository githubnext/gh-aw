---
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
tools:
  playwright:
    version: "v1.52.0"
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
    uses: actions/checkout@v6
    with:
      persist-credentials: false

  - name: Setup Node.js
    uses: actions/setup-node@v6
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

  - name: Start Astro dev server
    working-directory: ./docs
    run: npm run dev &

  - name: Wait for dev server
    run: |
      for i in $(seq 1 30); do
        if curl -sf http://localhost:4321/gh-aw/ > /dev/null 2>&1; then
          echo "Dev server is ready"
          exit 0
        fi
        echo "Waiting for dev server... attempt $i/30"
        sleep 1
      done
      echo "Dev server did not become ready in time" >&2
      exit 1
---

# Visual Regression Checker

You are a visual quality agent. The documentation site has been checked out, built, and the dev server is already running at `http://localhost:4321/gh-aw/`. For this pull request, use Playwright to capture screenshots of key pages and report any visual differences.

## Steps

1. **Capture screenshots** — Use Playwright to take full-page screenshots of the key pages at the following viewports:
   - **Mobile**: 375 × 812
   - **Tablet**: 768 × 1024
   - **Desktop**: 1440 × 900
2. **Accessibility snapshot** — For each page, run an accessibility snapshot and note any violations.
3. **Report** — Post a summary comment with:
   - A table listing each page, viewport, and screenshot status (unchanged / changed / error)
   - Any accessibility issues found

Post the summary as a pull request comment using the `add_comment` safe-output tool.
If there are no differences and no accessibility issues, call `noop` with a brief message.
