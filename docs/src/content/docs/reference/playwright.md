---
title: Playwright
description: Configure Playwright browser automation for testing web applications, accessibility analysis, and visual testing in your agentic workflows
sidebar:
  order: 720
---

Configure Playwright for browser automation and testing in your agentic workflows. Playwright enables headless browser control for accessibility testing, visual regression detection, end-to-end testing, and web scraping.

```yaml wrap
tools:
  playwright:
  playwright:
    version: "1.56.1"  # Optional: specify version, defaults to 1.56.1
  playwright:
    version: "latest"  # Use the latest available version
```

## Configuration Options

### Version

Specify the Playwright version to use:

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

### Default Localhost Access

Without any `network:` configuration, Playwright defaults to:

```yaml wrap
network:
  allowed:
    - "localhost"
    - "127.0.0.1"
```

This is sufficient for testing local development servers.

## GitHub Actions Compatibility

Playwright runs in a Docker container on GitHub Actions runners. To ensure Chromium functions correctly, gh-aw automatically configures required security flags:

- `--security-opt seccomp=unconfined` - Allows Chromium's sandboxing mechanisms
- `--ipc=host` - Enables inter-process communication for browser processes

These flags are automatically applied starting with **gh-aw version 0.41.0 and later**. No manual configuration is needed.

## Browser Support

Playwright includes three browser engines:

- **Chromium** - Chrome/Edge engine (most commonly used)
- **Firefox** - Mozilla Firefox engine
- **WebKit** - Safari engine

All three browsers are available in the Playwright Docker container. Your workflow can use any or all of them based on your testing needs.

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
  issues: write
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

```aw wrap
---
on:
  pull_request:
    types: [opened, synchronize]

tools:
  playwright:

network:
  allowed:
    - defaults
    - playwright
    - github

permissions:
  pull-requests: write
  contents: read

safe-outputs:
  add-comment:
    max: 1
---

# Visual Regression Check

Compare screenshots of the documentation site before and after this PR.

Test on multiple viewports (mobile, tablet, desktop) and report any visual differences.
```

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
