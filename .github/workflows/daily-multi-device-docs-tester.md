---
name: Multi-Device Docs Tester
description: Tests documentation site functionality and responsive design across multiple device form factors
on:
  schedule: daily
  workflow_dispatch:
    inputs:
      devices:
        description: 'Device types to test (comma-separated: mobile,tablet,desktop)'
        required: false
        default: 'mobile,tablet,desktop'
permissions:
  contents: read
  issues: read
  pull-requests: read
tracker-id: daily-multi-device-docs-tester
engine:
  id: claude
  max-turns: 80  # 10 devices × ~5 turns each + setup/report overhead
strict: true
timeout-minutes: 30
runtimes:
  node:
    version: "24"
tools:
  timeout: 120  # Playwright navigation on Astro dev server can take >60s; increase to 120s
  playwright:
    version: "v1.56.1"
  bash:
    - "npm install*"
    - "npm run dev*"
    - "npx astro*"
    - "npx playwright*"
    - "curl*"
    - "kill*"
    - "lsof*"
    - "ls*"      # List files for directory navigation
    - "pwd*"     # Print working directory
    - "cd*"      # Change directory
safe-outputs:
  upload-artifact:
    max-uploads: 3
    retention-days: 30
    skip-archive: true
  create-issue:
    expires: 2d
    labels: [cookie]

network:
  allowed:
    - node

imports:
  - shared/docs-server-lifecycle.md
  - shared/reporting.md
  - shared/observability-otlp.md
---

{{#runtime-import? .github/shared-instructions.md}}

# Multi-Device Documentation Testing

You are a documentation testing specialist. Your task is to comprehensively test the documentation site across multiple devices and form factors.

## Context

- Repository: ${{ github.repository }}
- Triggered by: @${{ github.actor }}
- Devices to test: ${{ inputs.devices }}
- Working directory: ${{ github.workspace }}

**IMPORTANT SETUP NOTES:**
1. You're already in the repository root
2. The docs folder is at: `${{ github.workspace }}/docs`
3. Use absolute paths or change directory explicitly
4. Keep token usage low by being efficient with your code and minimizing iterations
5. **Playwright is available via MCP tools only** - do NOT try to `require('playwright')` or install it via npm

## Your Mission

Start the documentation development server and perform comprehensive multi-device testing. Test layout responsiveness, accessibility, interactive elements, and visual rendering across all device types. Use a single Playwright browser instance for efficiency.

## Step 1: Install Dependencies and Start Server

Navigate to the docs folder and install dependencies:

```bash
cd ${{ github.workspace }}/docs
npm install
```

Follow the shared **Documentation Server Lifecycle Management** instructions:
1. Start the dev server (section "Starting the Documentation Preview Server")
2. Wait for server readiness (section "Waiting for Server Readiness")

## Step 2: Device Configuration

Test these device types based on input `${{ inputs.devices }}`:

**Mobile:** iPhone 12 (390x844), iPhone 12 Pro Max (428x926), Pixel 5 (393x851), Galaxy S21 (360x800)
**Tablet:** iPad (768x1024), iPad Pro 11 (834x1194), iPad Pro 12.9 (1024x1366)
**Desktop:** HD (1366x768), FHD (1920x1080), 4K (2560x1440)

## Step 3: Run Playwright Tests

**IMPORTANT: Using Playwright in gh-aw Workflows**

Playwright is provided through an MCP server interface, **NOT** as an npm package. You must use the MCP Playwright tools:

- ✅ **Correct**: Use `mcp__playwright__browser_run_code` with `page.goto(..., { waitUntil: 'domcontentloaded' })`
- ❌ **Incorrect**: Do NOT try to `require('playwright')` or create standalone Node.js scripts
- ❌ **Incorrect**: Do NOT install playwright via npm - it's already available through MCP

**⚠️ CRITICAL: Navigation Timeout Prevention**

The Astro development server uses Vite, which loads many JavaScript modules per page. Using the default `waitUntil: 'load'` or `waitForLoadState('networkidle')` will cause 60s timeouts because the browser waits for all modules to finish. **Always use `waitUntil: 'domcontentloaded'`** for navigation:

- ✅ **Correct**: `page.goto(url, { waitUntil: 'domcontentloaded', timeout: 30000 })`
- ❌ **Never use**: `page.waitForLoadState('networkidle')` — causes guaranteed timeouts
- ❌ **Never use**: `mcp__playwright__browser_navigate` for first load — it uses default 'load' wait which times out

**Example Usage:**

```bash
# First, get the container's bridge IP (needed for Playwright - see shared lifecycle instructions)
SERVER_IP=$(ip -4 route get 1.1.1.1 2>/dev/null | awk '{print $7; exit}')
if [ -z "$SERVER_IP" ]; then SERVER_IP=$(hostname -I | awk '{print $1}'); fi
echo "Playwright server URL: http://${SERVER_IP}:4321/gh-aw/"
```

```javascript
// Use browser_run_code to execute Playwright commands.
// IMPORTANT: Replace 172.30.0.20 below with the actual SERVER_IP from the bash command above.
// Do NOT use "localhost" — Playwright runs with --network host so its localhost differs.
// ALWAYS use waitUntil: 'domcontentloaded' to prevent timeout on the Vite dev server.
mcp__playwright__browser_run_code({
  code: `async (page) => {
    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto('http://172.30.0.20:4321/gh-aw/', { waitUntil: 'domcontentloaded', timeout: 30000 });  // substitute actual SERVER_IP
    return { url: page.url(), title: await page.title() };
  }`
})
```

For each device viewport, use Playwright MCP tools to:
- Set viewport size and navigate to `http://${SERVER_IP}:4321/gh-aw/` (substitute the bridge IP you obtained above, NOT localhost)
- Take screenshots and run accessibility audits
- Test interactions (navigation, search, buttons)
- Check for layout issues (overflow, truncation, broken layouts)

## Step 4: Analyze Results

Organize findings by severity:
- 🔴 **Critical**: Blocks functionality or major accessibility issues
- 🟡 **Warning**: Minor issues or potential problems
- 🟢 **Passed**: Everything working as expected

## Step 5: Report Results

### If NO Issues Found

**YOU MUST CALL** the `noop` tool to log completion:

```json
{
  "noop": {
    "message": "Multi-device documentation testing complete. All {device_count} devices tested successfully with no issues found."
  }
}
```

**DO NOT just write this message in your output text** - you MUST actually invoke the `noop` tool. The workflow will fail if you don't call it.

### If Issues ARE Found

Create a GitHub issue titled "🔍 Multi-Device Docs Testing Report - [Date]" with:

```markdown
### Test Summary
- Triggered by: @${{ github.actor }}
- Workflow run: [§${{ github.run_id }}](https://github.com/${{ github.repository }}/actions/runs/${{ github.run_id }})
- Devices tested: {count}
- Test date: [Date]

### Results Overview
- 🟢 Passed: {count}
- 🟡 Warnings: {count}
- 🔴 Critical: {count}

### Critical Issues
[List critical issues that block functionality or major accessibility problems - keep visible]

<details>
<summary>View All Warnings</summary>

[Minor issues and potential problems with device names and details]

</details>

<details>
<summary>View Detailed Test Results by Device</summary>

#### Mobile Devices
[Test results, screenshots, findings]

#### Tablet Devices
[Test results, screenshots, findings]

#### Desktop Devices
[Test results, screenshots, findings]

</details>

### Accessibility Findings
[Key accessibility issues - keep visible as these are important]

### Recommendations
[Actionable recommendations for fixing issues - keep visible]
```

Label with: `documentation`, `testing`, `automated`

## Step 6: Cleanup

Follow the shared **Documentation Server Lifecycle Management** instructions for cleanup (section "Stopping the Documentation Server").

## Summary

**Always provide a safe output:**
- **If issues found**: Create GitHub issue with test results, findings, and recommendations
- **If no issues found**: Call `noop` tool with completion message including total devices tested and pass status

The workflow requires explicit safe output (either issue creation or noop) to confirm completion.