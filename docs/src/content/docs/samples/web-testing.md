---
title: Web Testing & Screenshots
description: Automated browser testing workflows using Playwright for visual testing, accessibility analysis, and comprehensive web application validation
sidebar:
  order: 5
---

Web testing workflows provide comprehensive browser automation capabilities for visual testing, accessibility analysis, and user experience validation using containerized Playwright.

These workflows enable automated testing scenarios including screenshot comparison, accessibility audits, functional testing, and cross-browser compatibility validation.

## 🌐 Core Web Testing Capabilities

### Browser Automation with Screenshots
Playwright provides full browser automation with screenshot capabilities for visual testing and documentation:

```yaml
---
on: workflow_dispatch
permissions:
  contents: read
engine: claude
tools:
  playwright:
    allowed_domains: ["localhost", "*.github.com", "github.com"]
safe-outputs:
  upload-assets:
  create-issue:
    title-prefix: "[Visual Testing] "
---

# Visual Testing Workflow

1. Navigate to the target application using Playwright
2. Take full-page screenshots for visual comparison
3. Upload screenshots as assets for documentation
4. Analyze visual changes and report findings
```

### Accessibility Testing with Visual Analysis
Automated accessibility testing combining browser inspection with visual analysis:

```yaml
tools:
  playwright:
    allowed_domains: ["github.com", "*.github.com"]
safe-outputs:
  create-issue:
    title-prefix: "[Accessibility] "
    labels: [accessibility, a11y-audit]
```

## 🔧 Playwright Configuration Options

### Domain Access Control
Configure which domains Playwright can access using ecosystem bundles or specific domains:

```yaml
tools:
  playwright:
    docker_image_version: "v1.41.0"    # Specific Playwright version
    allowed_domains: 
      - "defaults"                     # Basic infrastructure
      - "github"                      # GitHub ecosystem
      - "localhost"                   # Local development
      - "*.example.com"               # Custom domains
```

### Security-First Approach
- **Default**: Localhost-only access for enhanced security
- **Containerized**: Isolated Docker environment prevents host system access
- **Domain restrictions**: Explicit allow-list prevents unauthorized network access

## 📋 Common Testing Patterns

### Visual Regression Testing
Take screenshots at different breakpoints and compare changes:

```markdown
# Visual Testing Instructions

1. **Setup Test Environment**
   - Start the application server
   - Verify all services are running

2. **Capture Screenshots**
   - Navigate to key application pages
   - Take full-page screenshots at different viewport sizes
   - Capture specific UI components

3. **Upload and Analyze**
   - Use safe-outputs upload-assets to store screenshots
   - Compare with baseline images
   - Report visual differences and regressions
```

### Cross-Browser Compatibility
Test functionality across different browser engines:

```markdown
# Multi-Browser Testing

Test the same functionality across:
- Chromium (default)
- Firefox 
- WebKit (Safari)

For each browser:
1. Navigate to test pages
2. Verify core functionality
3. Take comparison screenshots
4. Document any browser-specific issues
```

### Performance and Accessibility Audits
Comprehensive analysis combining multiple testing approaches:

```markdown
# Comprehensive Web Audit

1. **Performance Analysis**
   - Measure page load times
   - Check Core Web Vitals
   - Identify performance bottlenecks

2. **Accessibility Testing**
   - WCAG 2.1 AA compliance checks
   - Color contrast analysis from screenshots
   - Keyboard navigation testing

3. **Visual Analysis**
   - Screenshot-based accessibility review
   - Text readability assessment
   - UI element visibility validation
```

## 🎯 Real-World Examples

### Documentation Site Testing
Based on the pattern used in this repository's development workflow:

- **Local Development Server**: Start docs server and verify accessibility
- **Screenshot Capture**: Full-page screenshots for visual documentation
- **Asset Upload**: Store screenshots as downloadable assets
- **Accessibility Analysis**: Visual contrast and readability assessment

### Repository Page Analysis
Following the accessibility contrast testing pattern:

- **GitHub Navigation**: Automated browsing of repository pages
- **Visual Elements**: Screenshots of key UI components
- **Contrast Analysis**: WCAG compliance checking from visual output
- **Issue Creation**: Automated reporting with visual evidence

## ⚡ Best Practices

### Efficient Screenshot Management
- Use descriptive filenames with timestamps
- Upload screenshots as assets for permanent storage
- Include screenshot URLs in issue reports for visual context

### Domain Security
- Start with localhost-only for local testing
- Add specific domains as needed for external testing
- Use ecosystem bundles (`github`, `node`, etc.) for trusted environments

### Test Organization
- Structure tests with clear step-by-step instructions
- Combine screenshot capture with analysis steps
- Use safe-outputs for secure result reporting

### Error Handling
- Verify page loads before taking screenshots
- Handle network timeouts gracefully
- Provide fallback instructions for manual verification

## 🔗 Integration with Safe Outputs

Playwright workflows integrate seamlessly with safe output processing:

```yaml
safe-outputs:
  upload-assets:           # Store screenshots and test artifacts
  create-issue:           # Report findings with visual evidence
    title-prefix: "[Testing] "
    labels: [automated-testing, playwright]
  add-comment:            # Update existing issues with new results
```

## 📚 Learn More

- **Playwright Tool Reference**: [Playwright Configuration](/gh-aw/reference/tools/#playwright-tool-playwright)
- **Safe Outputs Guide**: [Output Processing](/gh-aw/reference/frontmatter/#safe-outputs)
- **Security Guidelines**: [Network Access Control](/gh-aw/guides/security/)

> [!TIP]
> Start with simple screenshot workflows and gradually add more complex testing scenarios. The containerized Playwright environment provides a safe sandbox for comprehensive web testing.

> [!WARNING]
> GitHub Agentic Workflows is a research demonstrator, and not for production use.