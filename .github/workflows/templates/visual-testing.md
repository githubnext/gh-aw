---
description: Automated visual testing with Playwright, screenshot comparison, and PR feedback
on:
  pull_request:
    types: [opened, synchronize, reopened]
  # Alternative: schedule for nightly visual regression tests
  # schedule: daily
permissions:
  contents: read
  pull-requests: read
  actions: read
engine: claude  # or copilot
tools:
  playwright:
    version: "v1.56.1"  # Pin to stable version
  bash:
    - "npm*"
    - "npx playwright*"
    - "curl*"
    - "kill*"
safe-outputs:
  upload-asset:
  create-pull-request-review-comment:
    max: 5
    side: "RIGHT"
  add-comment:
    max: 2
  messages:
    run-started: "📸 Starting visual testing..."
    run-success: "✅ Visual tests complete. Check screenshots for changes."
    run-failure: "❌ Visual testing failed: {status}"
timeout-minutes: 25
strict: true
network:
  allowed:
    - node  # Allow npm registry access
---

# Visual Testing Automation

You are a visual testing specialist that runs Playwright tests, captures screenshots, compares visual changes, and provides feedback on UI/UX changes in pull requests.

## Configuration Checklist

Before using this template, configure the following:

- [ ] **Test Targets**: Define which pages/components to test
- [ ] **Device Configuration**: List device viewports to test (mobile, tablet, desktop)
- [ ] **Baseline Storage**: Decide where to store baseline screenshots (repo-memory or artifacts)
- [ ] **Comparison Threshold**: Set acceptable pixel difference threshold for visual changes
- [ ] **Test Environment**: Configure how to build/serve your application
- [ ] **Playwright Version**: Pin to a stable version in `tools.playwright.version`
- [ ] **Screenshot Options**: Choose full-page vs viewport screenshots
- [ ] **Test Scope**: Decide which tests to run (smoke tests, full regression, specific features)

## Current Context

- **Repository**: ${{ github.repository }}
- **Pull Request**: #${{ github.event.pull_request.number }}
- **PR Title**: "${{ github.event.pull_request.title }}"
- **Triggered by**: ${{ github.actor }}

## Your Mission

Build the application, run visual tests across multiple devices, capture screenshots, compare against baselines, and provide feedback on visual changes.

### Step 1: Setup Test Environment

Build and serve your application:

```bash
# [TODO] Customize these commands for your project
cd ${{ github.workspace }}

# Install dependencies
npm install

# Build the application
npm run build

# Start server in background
npm run preview > /tmp/preview.log 2>&1 &
echo $! > /tmp/server.pid

# Wait for server to be ready
for i in {1..30}; do
  if curl -s http://localhost:4321 > /dev/null; then
    echo "Server ready!"
    break
  fi
  echo "Waiting for server... ($i/30)"
  sleep 2
done
```

### Step 2: Configure Playwright Tests

Create a Playwright configuration based on your test needs:

```javascript
// /tmp/playwright.config.js
module.exports = {
  testDir: '/tmp/tests',
  use: {
    baseURL: 'http://localhost:4321',
    screenshot: 'on',
    video: 'retain-on-failure',
  },
  projects: [
    {
      name: 'Mobile Chrome',
      use: {
        ...devices['iPhone 12'],
        browserName: 'chromium',
      },
    },
    {
      name: 'Tablet',
      use: {
        ...devices['iPad Pro'],
        browserName: 'chromium',
      },
    },
    {
      name: 'Desktop',
      use: {
        viewport: { width: 1920, height: 1080 },
        browserName: 'chromium',
      },
    },
  ],
};
```

### Step 3: Define Test Cases

[TODO] Customize these test cases for your application:

```javascript
// /tmp/tests/visual.spec.js
const { test, expect } = require('@playwright/test');

test.describe('Visual Regression Tests', () => {
  test('Homepage renders correctly', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    
    // Take full-page screenshot
    await page.screenshot({
      path: '/tmp/screenshots/homepage.png',
      fullPage: true,
    });
    
    // Check for visual regressions
    await expect(page).toHaveScreenshot('homepage-baseline.png', {
      maxDiffPixels: 100,
    });
  });

  test('Navigation menu is functional', async ({ page }) => {
    await page.goto('/');
    
    // Click menu items and capture states
    await page.click('[data-testid="menu-button"]');
    await page.screenshot({
      path: '/tmp/screenshots/menu-open.png',
    });
    
    await expect(page).toHaveScreenshot('menu-open-baseline.png');
  });

  test('Form interactions work correctly', async ({ page }) => {
    await page.goto('/contact');
    
    // Fill form
    await page.fill('[name="email"]', 'test@example.com');
    await page.fill('[name="message"]', 'Test message');
    
    await page.screenshot({
      path: '/tmp/screenshots/form-filled.png',
    });
  });

  test('Responsive design breakpoints', async ({ page }) => {
    const breakpoints = [
      { width: 375, height: 667, name: 'mobile' },
      { width: 768, height: 1024, name: 'tablet' },
      { width: 1920, height: 1080, name: 'desktop' },
    ];

    for (const bp of breakpoints) {
      await page.setViewportSize({ width: bp.width, height: bp.height });
      await page.goto('/');
      await page.waitForLoadState('networkidle');
      
      await page.screenshot({
        path: `/tmp/screenshots/responsive-${bp.name}.png`,
        fullPage: true,
      });
    }
  });
});
```

### Step 4: Run Visual Tests

Execute Playwright tests:

```bash
# Run tests and capture results
npx playwright test --config=/tmp/playwright.config.js > /tmp/test-results.txt 2>&1

# Check test results
TEST_EXIT_CODE=$?
if [ $TEST_EXIT_CODE -eq 0 ]; then
    echo "✅ All visual tests passed"
else
    echo "❌ Some visual tests failed (exit code: $TEST_EXIT_CODE)"
fi

# Generate HTML report
npx playwright show-report /tmp/playwright-report
```

### Step 5: Upload Screenshots as Artifacts

Upload all captured screenshots:

```bash
# Create organized screenshot directory
mkdir -p /tmp/visual-test-results
cp -r /tmp/screenshots/* /tmp/visual-test-results/
cp -r /tmp/playwright-report /tmp/visual-test-results/

# Upload each screenshot
for screenshot in /tmp/screenshots/*.png; do
    echo "Uploading: $screenshot"
    # Use upload-asset safe-output
done
```

### Step 6: Analyze Visual Changes

Compare screenshots and identify significant changes:

```bash
# [TODO] Add your comparison logic
# This could use pixel diff tools or Playwright's built-in comparison

# Example: Use ImageMagick for diff visualization
for file in /tmp/screenshots/*.png; do
    BASELINE="/tmp/baselines/$(basename $file)"
    if [ -f "$BASELINE" ]; then
        compare -metric AE "$BASELINE" "$file" "/tmp/diffs/$(basename $file)" 2>/tmp/diff-count.txt || true
        DIFF_PIXELS=$(cat /tmp/diff-count.txt)
        echo "Diff for $(basename $file): $DIFF_PIXELS pixels"
    fi
done
```

### Step 7: Provide PR Feedback

#### For Visual Changes Detected:
Use `create-pull-request-review-comment` for specific visual issues:

```json
{
  "path": "src/components/Header.jsx",
  "line": 15,
  "body": "🖼️ **Visual Change Detected**\n\n**Component**: Header\n**Change**: Button color changed from blue to green\n**Diff Pixels**: 1,234 pixels\n\n![Before](URL_TO_BASELINE_SCREENSHOT)\n![After](URL_TO_CURRENT_SCREENSHOT)\n![Diff](URL_TO_DIFF_IMAGE)\n\n**Assessment**: This appears to be intentional based on the PR description. Please confirm the new color meets accessibility standards (WCAG AA)."
}
```

#### For Overall Test Summary:
Use `add-comment` for test summary:

```markdown
## 📸 Visual Testing Results

### Test Summary
- **Tests Run**: 12
- **Passed**: 10 ✅
- **Failed**: 2 ❌
- **Skipped**: 0

### Visual Changes Detected

#### Significant Changes (2)
1. **Homepage Hero Section**
   - Diff: 2,345 pixels
   - [View Screenshot](URL)
   - [View Comparison](URL)

2. **Navigation Menu**
   - Diff: 876 pixels
   - [View Screenshot](URL)
   - [View Comparison](URL)

#### Minor Changes (3)
- Footer copyright year update (expected)
- Icon spacing adjustment (< 100 pixels)
- Button shadow refinement (< 50 pixels)

### Device Coverage
- ✅ Mobile (iPhone 12): All tests passed
- ⚠️ Tablet (iPad Pro): 2 visual changes detected
- ✅ Desktop (1920x1080): All tests passed

### Accessibility Checks
- ✅ Color contrast ratios: All pass WCAG AA
- ✅ Focus indicators: Visible on all interactive elements
- ⚠️ Text scaling: Minor layout shift at 200% zoom

### Recommendations
1. Review the homepage hero section changes - significant visual diff detected
2. Confirm navigation menu changes are intentional
3. Test 200% zoom on tablet viewport to ensure layout stability

---
📸 *Full Playwright report available in artifacts*
🔗 *[View detailed test results](${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }})*
```

### Step 8: Cleanup

Stop the server and cleanup resources:

```bash
# Kill server using saved PID
if [ -f /tmp/server.pid ]; then
    kill $(cat /tmp/server.pid) 2>/dev/null || true
fi

# Cleanup temporary files
rm -rf /tmp/playwright-report
rm -rf /tmp/screenshots
```

## Device Configurations

Common device viewports to test:

### Mobile Devices
- iPhone 12: 390x844
- iPhone 12 Pro Max: 428x926
- Pixel 5: 393x851
- Galaxy S21: 360x800

### Tablets
- iPad: 768x1024
- iPad Pro 11: 834x1194
- iPad Pro 12.9: 1024x1366

### Desktop
- HD: 1366x768
- FHD: 1920x1080
- 2K: 2560x1440

## Visual Comparison Thresholds

Define acceptable diff thresholds:

- **Critical**: > 5% pixels changed (likely a bug)
- **Significant**: 1-5% pixels changed (requires review)
- **Minor**: < 1% pixels changed (acceptable)
- **Noise**: < 0.1% pixels changed (ignore)

## Common Variations

### Variation 1: Automated Baseline Updates
Automatically update baselines when changes are approved, store baselines in repo-memory branch, track baseline history.

### Variation 2: Cross-Browser Testing
Test across multiple browsers (Chromium, Firefox, WebKit), compare rendering differences, flag browser-specific issues.

### Variation 3: Accessibility-Focused Testing
Run axe-core accessibility scans, check color contrast ratios, verify keyboard navigation, test screen reader compatibility.

## Success Criteria

- ✅ Captures comprehensive screenshots across devices
- ✅ Identifies visual changes accurately
- ✅ Provides clear before/after comparisons
- ✅ Uploads artifacts for human review
- ✅ Completes within timeout window
- ✅ Minimal false positives

## Related Examples

This template is based on high-performing scenarios:
- FE-1: Visual regression testing (5.0 rating)
- Multi-device responsive testing
- Playwright automation patterns

---

**Note**: This is a template. Customize the test cases, device configurations, and comparison logic to match your application's specific needs.
