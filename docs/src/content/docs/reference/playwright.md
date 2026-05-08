---
title: Playwright
description: Configure Playwright browser automation for testing web applications, accessibility analysis, and visual testing in your agentic workflows
sidebar:
  order: 720
---

Configure Playwright for browser automation and testing in your agentic workflows. Playwright enables headless browser control for accessibility testing, visual regression detection, end-to-end testing, and web scraping.

## Modes

Playwright supports two integration modes. **CLI mode is recommended** for all new workflows.

### CLI Mode (Recommended)

```yaml wrap
tools:
  playwright:
    mode: cli
```

CLI mode installs `@playwright/cli` as a global npm package on the runner before the agent starts. The agent uses `playwright-cli <command>` in bash to automate the browser — no Docker container is required.

**Benefits of CLI mode:**
- **Token-efficient** — no large MCP tool schemas loaded into the model context
- **No Docker overhead** — playwright runs directly on the runner
- **Use `localhost` directly** — no bridge IP detection needed when accessing local dev servers
- **Same commands** — all `browser_*` commands are available (e.g. `playwright-cli browser_navigate --url ...`)

```bash wrap
# Navigate to a page
playwright-cli browser_navigate --url "https://example.com"

# Take a screenshot
playwright-cli browser_take_screenshot --filename /tmp/screenshot.png --full-page true

# Get the accessibility tree
playwright-cli browser_snapshot

# Evaluate JavaScript
playwright-cli browser_evaluate --expression "document.title"

# Run arbitrary Playwright code
playwright-cli browser_run_code --code "async (page) => { await page.goto('https://example.com'); return await page.title(); }"
```

### MCP Mode (Deprecated)

:::caution[Deprecated]
MCP mode is deprecated. Migrate to CLI mode (`mode: cli`) for better token efficiency and simpler local server access. MCP mode will emit a deprecation warning during compilation.
:::

```yaml wrap
tools:
  playwright:
    mode: mcp  # deprecated — use mode: cli instead
```

MCP mode starts a Docker-based `mcr.microsoft.com/playwright/mcp` container and exposes browser tools via the Model Context Protocol. Because the browser runs in Docker with `--network host`, agents cannot use `localhost` to reach local dev servers and must detect the container bridge IP.

**Why migrate away from MCP mode:**
- Large MCP tool schemas consume significant model context tokens
- Requires Docker container startup time
- Local server access requires bridge IP detection instead of `localhost`
- CLI mode provides the same browser automation capabilities with less overhead

## Configuration Options

### Version

The `version` field controls different things depending on the mode:

**CLI mode** (`mode: cli`, recommended) — pins the `@playwright/cli` npm package version:

```yaml wrap
tools:
  playwright:
    mode: cli
    version: "0.1.13"  # @playwright/cli npm package version (default)
```

**Default** (CLI mode): `0.1.13`

**MCP mode** (deprecated) — pins the Playwright browser Docker image version:

```yaml wrap
tools:
  playwright:
    mode: mcp  # deprecated
    version: "v1.56.1"  # Browser Docker image version
```

When `version` is not specified, the compiler uses the built-in default for the active mode.

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

In CLI mode, `playwright-cli` runs directly on the GitHub Actions runner — no Docker container is involved. The runner's Node.js environment is used, and `localhost` connects directly to any server running on the runner.

In MCP mode (deprecated), Playwright runs in a Docker container with `--security-opt seccomp=unconfined` and `--ipc=host` (required for Chromium). Because the container uses `--network host`, its `localhost` resolves to the Docker host rather than the agent container, requiring bridge IP detection to reach local servers.

## Browser Support

Playwright includes three browser engines: **Chromium** (Chrome/Edge, most commonly used), **Firefox**, and **WebKit** (Safari). All three are available in both CLI mode and MCP mode.

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

Navigate to the site and capture the accessibility snapshot:
```bash
playwright-cli browser_navigate --url "https://docs.example.com"
playwright-cli browser_snapshot
```

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

The documentation site dev server is running at http://localhost:4321/.

Check for visual regressions on these pages: home, getting-started, and reference.

Test on multiple viewports:
- Mobile: 375×812
- Tablet: 768×1024
- Desktop: 1440×900

For each viewport, resize the browser and take a screenshot:
```bash
playwright-cli browser_resize --width 375 --height 812
playwright-cli browser_navigate --url "http://localhost:4321/"
playwright-cli browser_take_screenshot --filename /tmp/mobile-screenshot.png --full-page true
```

Take screenshots at each viewport and compare against baseline. Report any visual differences as a pull request comment, including screenshots. If there are no regressions, call noop.
```

**Key patterns for dev server visual regression:**

- **Path filter** — restricts the trigger to runs affecting frontend assets, avoiding noise on unrelated changes.
- **`steps:`** — run before the agent. Use them to install dependencies, build, start the server, and poll until it is ready. The agent only starts after all steps succeed.
- **Version pin** — pin Playwright to a specific version (`v1.52.0`) to prevent baseline drift from browser engine upgrades mid-test.
- **`mode: cli`** — playwright-cli runs on the runner; use `localhost` to reach the dev server directly.
- **`bash` allowlist** — restricts the `bash` tool to `npm *` and `curl http://localhost:*` only, keeping the tool surface minimal.

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

Start the development server locally and run end-to-end tests with Playwright.

1. Start the dev server on localhost:3000
2. Navigate with playwright-cli: `playwright-cli browser_navigate --url "http://localhost:3000"`
3. Test the complete user journey
4. Report any failures with screenshots
```

## Related Documentation

- [Tools Reference](/gh-aw/reference/tools/) - All tool configurations
- [Network Permissions](/gh-aw/reference/network/) - Network access control
- [Network Configuration Guide](/gh-aw/guides/network-configuration/) - Common network patterns
- [Safe Outputs Reference](/gh-aw/reference/safe-outputs/) - Configure output creation
- [Frontmatter](/gh-aw/reference/frontmatter/) - All frontmatter configuration options
