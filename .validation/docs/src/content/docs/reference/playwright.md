---
title: Playwright
description: Configure Playwright browser automation for testing web applications, accessibility analysis, and visual testing in your agentic workflows
sidebar:
  order: 720
---

Playwright enables headless browser control for accessibility testing, visual regression detection, end-to-end testing, and web scraping.

## Modes

Playwright supports two integration modes. **CLI mode is recommended** for all new workflows — it is token-efficient (no large MCP schemas in context), avoids Docker overhead, and lets the agent reach local dev servers via `localhost` directly.

### CLI Mode (Recommended)

```yaml wrap
tools:
  playwright:
    mode: cli
```

CLI mode installs `@playwright/cli` as a global npm package on the runner. The agent invokes `playwright-cli <command>` from bash:

```bash wrap
playwright-cli browser_navigate --url "https://example.com"
playwright-cli browser_take_screenshot --filename /tmp/screenshot.png --full-page true
playwright-cli browser_snapshot
playwright-cli browser_evaluate --expression "document.title"
playwright-cli browser_run_code --code "async (page) => { await page.goto('https://example.com'); return await page.title(); }"
```

### MCP Mode (Deprecated)

MCP mode is deprecated and emits a compile-time warning. Migrate to `mode: cli` for the reasons listed above. MCP mode runs Playwright in a Docker container with `--network host`, so `localhost` resolves to the Docker host and bridge IP detection is required to reach local servers.

```yaml wrap
tools:
  playwright:
    mode: mcp  # deprecated — use mode: cli instead
```

## Configuration

### Version

The `version` field has different meaning per mode:

| Mode | Pins | Default |
|------|------|---------|
| `cli` (recommended) | `@playwright/cli` npm package | `0.1.13` |
| `mcp` (deprecated) | Playwright browser Docker image | built-in |

```yaml wrap
tools:
  playwright:
    mode: cli
    version: "0.1.13"
```

### Network Access

Domain access is controlled by the top-level [`network:`](/gh-aw/reference/network/) field. By default, Playwright can only reach `localhost` and `127.0.0.1`. Use ecosystem identifiers and explicit domains together:

```yaml wrap
network:
  allowed:
    - defaults
    - playwright                 # enables browser downloads
    - "example.com"              # matches example.com and subdomains
    - "*.staging.example.com"    # wildcard pattern
```

Allowing `example.com` automatically allows its subdomains.

### Browser Support

Chromium (Chrome/Edge), Firefox, and WebKit (Safari) are all available in both modes.

## Common Use Cases

### Accessibility Testing

```aw wrap
---
on:
  schedule: daily

tools:
  playwright:
    mode: cli

network:
  allowed:
    - defaults
    - playwright
    - "docs.example.com"

permissions:
  contents: read

safe-outputs:
  create-issue:
    title-prefix: "[a11y] "
    labels: [accessibility, automated]
    max: 3
---

# Accessibility Audit

Use Playwright to check docs.example.com for WCAG 2.1 Level AA compliance.

```bash
playwright-cli browser_navigate --url "https://docs.example.com"
playwright-cli browser_snapshot
```

Run automated accessibility checks using axe-core and report missing alt text,
insufficient color contrast, missing ARIA labels, and keyboard navigation issues.
Create an issue for each category found.
```

### Visual Regression Testing

Use `steps:` to start the dev server before the agent runs, and pin Playwright to prevent baseline drift from browser-engine upgrades:

```aw wrap
---
on:
  pull_request:
    types: [opened, synchronize]
    paths:
      - 'docs/src/**/*.css'
      - 'docs/src/**/*.tsx'
      - 'docs/src/**/*.astro'
      - 'docs/astro.config.mjs'

steps:
  - uses: actions/checkout@v6
    with:
      persist-credentials: false
  - working-directory: ./docs
    run: npm ci && npm run build && npm run dev &
  - run: |
      # wait for dev server (max 30s)
      for i in $(seq 1 30); do
        curl -sf http://localhost:4321/ >/dev/null && exit 0
        sleep 1
      done
      exit 1

tools:
  playwright:
    mode: cli
    version: "v1.52.0"
  bash:
    - "npm *"
    - "curl http://localhost:*"

network:
  allowed:
    - defaults
    - playwright
    - local
    - node

permissions:
  contents: read

safe-outputs:
  add-comment:
    max: 1
  noop:
---

# Visual Regression Check

The dev server is running at http://localhost:4321/. Check for visual regressions
on the home, getting-started, and reference pages across three viewports:

- Mobile: 375×812
- Tablet: 768×1024
- Desktop: 1440×900

For each viewport, resize and screenshot:

```bash
playwright-cli browser_resize --width 375 --height 812
playwright-cli browser_navigate --url "http://localhost:4321/"
playwright-cli browser_take_screenshot --filename /tmp/mobile-screenshot.png --full-page true
```

Compare against baseline and report differences as a PR comment with screenshots.
If there are no regressions, call noop.
```

### End-to-End Testing

```aw wrap
---
on:
  workflow_dispatch:

tools:
  playwright:
    mode: cli
  bash: [":*"]

network:
  allowed:
    - defaults
    - playwright
    - "localhost"

permissions:
  contents: read
---

# E2E Testing

Start the dev server on localhost:3000, then drive a full user journey with
`playwright-cli browser_navigate --url "http://localhost:3000"`. Report any
failures with screenshots.
```

## Related Documentation

- [Tools Reference](/gh-aw/reference/tools/) — All tool configurations
- [Network Permissions](/gh-aw/reference/network/) — Network access control
- [Network Configuration Guide](/gh-aw/guides/network-configuration/) — Common patterns
- [Safe Outputs Reference](/gh-aw/reference/safe-outputs/) — Configure output creation
- [Frontmatter](/gh-aw/reference/frontmatter/) — All frontmatter options
