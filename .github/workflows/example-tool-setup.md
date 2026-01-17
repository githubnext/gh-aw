---
on: workflow_dispatch
name: Example Tool Setup
description: Demonstrates Playwright, axe-core, and webpack-bundle-analyzer setup
timeout-minutes: 10
strict: false
sandbox: true
engine: copilot

permissions:
  contents: read

network:
  allowed:
    - playwright

tools:
  playwright:
    version: "v1.40.0"
    allowed_domains: ["example.com"]

steps:
  - name: Setup Node.js
    uses: actions/setup-node@v4
    with:
      node-version: '20'
  
  - name: Install testing tools
    run: |
      npm init -y
      npm install -D @playwright/test @axe-core/playwright webpack-bundle-analyzer
      npx playwright install chromium

  - name: Create Playwright config
    run: |
      cat > playwright.config.js << 'EOF'
      module.exports = {
        use: {
          headless: true,
          viewport: { width: 1280, height: 720 },
          screenshot: 'only-on-failure',
        },
        projects: [
          { name: 'chromium', use: { browserName: 'chromium' } },
        ],
      };
      EOF

  - name: Create accessibility test
    run: |
      cat > accessibility.spec.js << 'EOF'
      const { test } = require('@playwright/test');
      const { injectAxe, checkA11y } = require('@axe-core/playwright');

      test('accessibility check', async ({ page }) => {
        await page.goto('https://example.com');
        await injectAxe(page);
        await checkA11y(page);
      });
      EOF

safe-outputs:
  staged: true
  create-issue:
    max: 1
---

# Tool Setup Validation Workflow

This workflow validates the tool setup examples provided in the developer instructions.

## Tasks

1. **Setup Environment**: Install Node.js and initialize a test project
2. **Install Tools**: Install Playwright, axe-core, and webpack-bundle-analyzer
3. **Create Configurations**: Generate configuration files based on the examples
4. **Validate Setup**: Verify that tools are installed and configured correctly
5. **Run Basic Tests**: Execute a simple accessibility test to ensure everything works
6. **Report Results**: Create an issue summarizing the validation results

## Expected Outcome

All tool setup examples should work correctly out of the box with minimal configuration.
