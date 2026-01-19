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
  max-turns: 30  # Prevent runaway token usage
strict: true
timeout-minutes: 30
tools:
  playwright:
    version: "v1.56.1"
  bash:
    - "npm install*"
    - "npm run build*"
    - "npm run preview*"
    - "npx playwright*"
    - "curl*"
    - "kill*"
    - "lsof*"
    - "ls*"      # List files for directory navigation
    - "pwd*"     # Print working directory
    - "cd*"      # Change directory
safe-outputs:
  upload-asset:
  create-issue:

network:
  allowed:
    - node

imports:
  - shared/docs-server-lifecycle.md
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

## Your Mission

Build the documentation site locally, serve it, and perform comprehensive multi-device testing. Test layout responsiveness, accessibility, interactive elements, and visual rendering across all device types. Use a single Playwright browser instance for efficiency.

## Step 1: Build and Serve

Navigate to the docs folder and build the site:

```bash
cd ${{ github.workspace }}/docs
npm install
npm run build
```

Follow the shared **Documentation Server Lifecycle Management** instructions:
1. Start the preview server (section "Starting the Documentation Preview Server")
2. Wait for server readiness (section "Waiting for Server Readiness")

## Step 2: Device Configuration

Test these device types based on input `${{ inputs.devices }}`:

**Mobile:** iPhone 12 (390x844), iPhone 12 Pro Max (428x926), Pixel 5 (393x851), Galaxy S21 (360x800)
**Tablet:** iPad (768x1024), iPad Pro 11 (834x1194), iPad Pro 12.9 (1024x1366)
**Desktop:** HD (1366x768), FHD (1920x1080), 4K (2560x1440)

## Step 3: Run Playwright Tests

For each device, use Playwright to:
- Set viewport size and navigate to http://localhost:4321
- Take screenshots and run accessibility audits
- Test interactions (navigation, search, buttons)
- Check for layout issues (overflow, truncation, broken layouts)

## Step 4: Analyze Results

Organize findings by severity:
- 🔴 **Critical**: Blocks functionality or major accessibility issues
- 🟡 **Warning**: Minor issues or potential problems
- 🟢 **Passed**: Everything working as expected

## Step 5: Report Results

If issues are detected, create a GitHub issue titled "🔍 Multi-Device Docs Testing Report - [Date]" with:
- Test summary (triggered by, workflow run, devices tested)
- Results overview (passed/warning/critical counts)
- Critical issues and warnings with device names
- Screenshots showing issues
- Accessibility findings and recommendations

Label with: `documentation`, `testing`, `automated`

## Step 6: Cleanup

Follow the shared **Documentation Server Lifecycle Management** instructions for cleanup (section "Stopping the Documentation Server").

## Summary

Provide: total devices tested, test results (passed/failed/warnings), key findings, and link to issue (if created).