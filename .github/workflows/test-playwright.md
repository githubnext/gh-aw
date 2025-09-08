---
on:
  workflow_dispatch:
  reaction: eyes

engine: 
  id: claude

safe-outputs:
  create-issue:

network:
  allowed:
    - "github.com"
    - "*.github.com"
    - "*.githubusercontent.com"

tools:
  playwright:
    allowed: ["browser_navigate", "browser_take_screenshot", "browser_snapshot", "browser_evaluate"]
---

# Playwright Accessibility Testing for GitHub Repository

**Objective**: Take a screenshot of the GitHub repository page for `githubnext/gh-aw` and analyze it for accessibility color contrast issues.

## Instructions

1. **Navigate to the Repository**: Use Playwright to navigate to https://github.com/githubnext/gh-aw

2. **Take a Screenshot**: Capture a full-page screenshot of the repository's main page

3. **Accessibility Analysis**: Examine the screenshot for potential color contrast accessibility issues:
   - Check if text has sufficient contrast against background colors
   - Look for elements that might be difficult to read for users with visual impairments
   - Identify any UI elements that don't meet WCAG color contrast guidelines (4.5:1 for normal text, 3:1 for large text)

4. **Create Report**: Create an issue with your findings that includes:
   - The screenshot you captured
   - A detailed analysis of any color contrast issues found
   - Specific recommendations for improving accessibility
   - Note any areas that appear to meet accessibility standards

## Report Template

Use this template for your issue:

**Title**: "Accessibility Analysis: Color Contrast Issues in GitHub Repository Page"

**Body**: Include your analysis findings, recommendations, and the screenshot.

### AI Attribution

Include this footer in your issue description:

```markdown
> AI-generated content by [${{ github.workflow }}](https://github.com/${{ github.repository }}/actions/runs/${{ github.run_id }}) may contain mistakes.
```