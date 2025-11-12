---
name: Multi-Device Docs Tester
on:
  schedule:
    - cron: '0 9 * * *'  # Daily at 9 AM UTC
  workflow_dispatch:
    inputs:
      devices:
        description: 'Device types to test (comma-separated: mobile,tablet,desktop)'
        required: false
        default: 'mobile,tablet,desktop'
permissions:
  contents: read
  issues: write
engine: claude
timeout-minutes: 30
tools:
  playwright:
    version: "v1.56.1"
    allowed_domains:
      - "localhost"
      - "127.0.0.1"
  bash:
    - "npm install*"
    - "npm run build*"
    - "npm run preview*"
    - "npx playwright*"
    - "curl*"
    - "kill*"
    - "lsof*"
safe-outputs:
  upload-assets:
  create-issue:

network:
  allowed:
    - node
---

# Multi-Device Documentation Testing

You are a documentation testing specialist. Your task is to comprehensively test the documentation site across multiple devices and form factors.

## Context

- Repository: ${{ github.repository }}
- Triggered by: @${{ github.actor }}
- Devices to test: ${{ inputs.devices }}

## Your Mission

Build the documentation site locally, serve it, and perform comprehensive multi-device testing including:

1. **Layout & Responsive Design Testing**
   - Test across all specified device types (mobile, tablet, desktop)
   - Verify proper breakpoint behavior
   - Check for layout overflow, text truncation, or element overlapping
   - Validate navigation menu behavior on different screen sizes

2. **Accessibility Audits**
   - Run automated accessibility checks using Playwright's accessibility tree
   - Check for proper heading hierarchy
   - Verify ARIA labels and roles
   - Test keyboard navigation
   - Check color contrast ratios

3. **Interactive Element Testing**
   - Test navigation links and menu interactions
   - Verify search functionality (if present)
   - Test code copy buttons
   - Check form submissions (if any)
   - Validate anchor links and table of contents navigation

4. **Visual Validation**
   - Capture screenshots for each device type
   - Check for visual regressions or rendering issues
   - Verify images load correctly and have proper alt text
   - Check for broken styles or missing CSS

## Setup Instructions

### Step 1: Build and Serve the Documentation

```bash
cd docs
npm install
npm run build
```

After building, start the preview server in the background:

```bash
npm run preview &
SERVER_PID=$!
echo "Server PID: $SERVER_PID"
```

Wait for the server to be ready (usually runs on port 4321):

```bash
for i in {1..30}; do
  if curl -s http://localhost:4321 > /dev/null; then
    echo "Server is ready!"
    break
  fi
  echo "Waiting for server... ($i/30)"
  sleep 2
done
```

### Step 2: Run Multi-Device Tests

Parse the device input and run tests for each device type:

```javascript
const devices = '${{ inputs.devices }}'.split(',').map(d => d.trim());

// Device configurations
const deviceConfigs = {
  mobile: [
    { name: 'iPhone 12', width: 390, height: 844 },
    { name: 'iPhone 12 Pro Max', width: 428, height: 926 },
    { name: 'Pixel 5', width: 393, height: 851 },
    { name: 'Galaxy S21', width: 360, height: 800 }
  ],
  tablet: [
    { name: 'iPad', width: 768, height: 1024 },
    { name: 'iPad Pro 11', width: 834, height: 1194 },
    { name: 'iPad Pro 12.9', width: 1024, height: 1366 }
  ],
  desktop: [
    { name: 'Desktop HD', width: 1366, height: 768 },
    { name: 'Desktop FHD', width: 1920, height: 1080 },
    { name: 'Desktop 4K', width: 2560, height: 1440 }
  ]
};

const results = {
  passed: [],
  failed: [],
  warnings: [],
  screenshots: []
};

for (const deviceType of devices) {
  const configs = deviceConfigs[deviceType] || [];
  
  for (const config of configs) {
    console.log(`Testing ${config.name} (${config.width}x${config.height})...`);
    
    // Test this device configuration
    // (Playwright testing code will be executed by the agent)
  }
}
```

Use Playwright to:
- Set viewport size for each device
- Navigate to http://localhost:4321
- Take screenshots
- Run accessibility audits
- Test interactions (click navigation, test search, etc.)
- Detect layout issues

### Step 3: Analyze Results

Review all test results and compile findings:

- **Layout Issues**: Any overflow, truncation, or broken layouts
- **Accessibility Issues**: Missing alt text, poor contrast, heading hierarchy problems
- **Interaction Failures**: Broken links, non-functional buttons, search issues
- **Visual Problems**: Missing images, broken styles, rendering issues

Organize findings by severity:
- 🔴 **Critical**: Blocks functionality or major accessibility issues
- 🟡 **Warning**: Minor issues or potential problems
- 🟢 **Passed**: Everything working as expected

### Step 4: Create Issue if Problems Found

If any critical or warning-level issues are detected, create a GitHub issue with:

**Title**: "🔍 Multi-Device Docs Testing Report - [Date]"

**Body**:
```markdown
## Test Summary

- **Triggered by**: @${{ github.actor }}
- **Workflow Run**: ${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}
- **Devices Tested**: [list all devices]

## Results Overview

- ✅ Passed: [count]
- 🟡 Warnings: [count]  
- 🔴 Critical: [count]

## Critical Issues

[List critical issues with device names and descriptions]

## Warnings

[List warning-level issues]

## Screenshots

[Include key screenshots showing issues, organized by device]

## Accessibility Report

[Summarize accessibility findings]

## Recommendations

[Provide specific recommendations for fixing issues]

---
*Automated multi-device testing report*
```

Label the issue with: `documentation`, `testing`, `automated`

### Step 5: Cleanup

Stop the preview server:

```bash
kill $SERVER_PID
```

## Output

Provide a summary of:
- Total devices tested
- Number of tests passed/failed/warnings
- Key findings
- Link to created issue (if any)
- Workflow artifacts location for detailed logs and screenshots
