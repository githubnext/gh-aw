---
title: Playwright
description: Configure Playwright browser automation for testing web applications, accessibility analysis, and visual testing in your agentic workflows
sidebar:
  order: 720
---

Configure Playwright for browser automation and testing in your agentic workflows. Playwright enables headless browser control for accessibility testing, visual regression detection, end-to-end testing, and web scraping.

## Configuration Options

### Version

Pin to a specific version or use the latest:

```yaml wrap
tools:
  playwright:
    version: "1.56.1"  # Pin to specific version (default)
  playwright:
    version: "latest"  # Use latest available version
```

**Default**: `1.56.1` (when `version` is not specified)

## Network Access Configuration

Domain access for Playwright is controlled by the top-level [`network:`](/gh-aw/reference/network/) field. By default, Playwright can only access `localhost` and `127.0.0.1`.

### Using Ecosystem Identifiers

```yaml wrap
network:
  allowed:
    - defaults
    - playwright     # Enables browser downloads
    - github         # For testing GitHub pages
    - node           # For testing Node.js apps
```

### Custom Domains

Add specific domains for the sites you want to test:

```yaml wrap
network:
  allowed:
    - defaults
    - playwright
    - "example.com"              # Matches example.com and subdomains
    - "*.staging.example.com"    # Wildcard for staging environments
```

**Automatic subdomain matching**: When you allow `example.com`, all subdomains like `api.example.com`, `www.example.com`, and `staging.example.com` are automatically allowed.

## GitHub Actions Compatibility

Playwright runs in a Docker container on GitHub Actions runners. gh-aw automatically applies `--security-opt seccomp=unconfined` and `--ipc=host` (required for Chromium) starting with version 0.41.0. No manual configuration is needed.

## Browser Support

Playwright includes three browser engines: **Chromium** (Chrome/Edge, most commonly used), **Firefox**, and **WebKit** (Safari). All three are available in the Playwright Docker container.

## Common Use Cases

### Accessibility Testing

```aw wrap
---
on:
  schedule:
    - cron: "0 9 * * *"  # Daily at 9 AM

tools:
  playwright:

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

Run automated accessibility checks using axe-core and report:
- Missing alt text on images
- Insufficient color contrast
- Missing ARIA labels
- Keyboard navigation issues

Create an issue for each category of problems found.
```

### Visual Regression Testing

Pin to a specific version and use `steps:` to start the application before the agent runs. This is the recommended pattern when testing a local dev server.

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
  - name: Checkout repository
    uses: actions/checkout@v6
    with:
      persist-credentials: false
  - name: Install dependencies
    working-directory: ./docs
    run: npm ci
  - name: Build documentation
    working-directory: ./docs
    run: npm run build
  - name: Start dev server
    working-directory: ./docs
    run: npm run dev &
  - name: Wait for dev server
    run: |
      for i in $(seq 1 30); do
        if curl -sf http://localhost:4321/ > /dev/null 2>&1; then
          echo "Dev server is ready"; exit 0
        fi
        sleep 1
      done
      exit 1

tools:
  playwright:
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

The documentation site dev server is running at http://localhost:4321/.

Check for visual regressions on these pages: home, getting-started, and reference.

Test on multiple viewports:
- Mobile: 375×812
- Tablet: 768×1024
- Desktop: 1440×900

Take screenshots at each viewport and compare against baseline. Report any visual differences as a pull request comment, including screenshots. If there are no regressions, call noop.
```

**Key patterns for dev server visual regression:**

- **Path filter** — restricts the trigger to runs affecting frontend assets, avoiding noise on unrelated changes.
- **`steps:`** — run before the agent. Use them to install dependencies, build, start the server, and poll until it is ready. The agent only starts after all steps succeed.
- **Version pin** — pin Playwright to a specific version (`v1.52.0`) to prevent baseline drift from browser engine upgrades mid-test.
- **`local` network identifier** — allows the Playwright container to reach `localhost`/`127.0.0.1` where the dev server runs. Required when testing local servers.
- **`bash` allowlist** — restricts the `bash` tool to `npm *` and `curl http://localhost:*` only, keeping the tool surface minimal.

### End-to-End Testing

```aw wrap
---
on:
  workflow_dispatch:

tools:
  playwright:
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

Start the development server locally and run end-to-end tests with Playwright.

1. Start the dev server on localhost:3000
2. Test the complete user journey
3. Report any failures with screenshots
```

## Related Documentation

- [Tools Reference](/gh-aw/reference/tools/) - All tool configurations
- [Network Permissions](/gh-aw/reference/network/) - Network access control
- [Network Configuration Guide](/gh-aw/guides/network-configuration/) - Common network patterns
- [Safe Outputs Reference](/gh-aw/reference/safe-outputs/) - Configure output creation
- [Frontmatter](/gh-aw/reference/frontmatter/) - All frontmatter configuration options
