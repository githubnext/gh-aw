---
on:
  workflow_dispatch:

permissions: read-all

engine:
  id: claude

network:
  allowed:
    - defaults
    - playwright

tools:
  playwright:
    docker_image_version: "latest"

safe-outputs:
  create-issue:
    title-prefix: "[Accessibility Test] "
    labels: [accessibility, automation, playwright, color-contrast]
    max: 1
---

# Accessibility Color Contrast Screenshot Analysis

Take a screenshot of the GitHub repository page at github.com/githubnext/gh-aw and analyze it for accessibility color contrast issues. If any issues are found, create an issue to document them.

## Task Steps

1. Navigate to https://github.com/githubnext/gh-aw using Playwright browser automation
2. Take a full-page screenshot of the repository page
3. Analyze the screenshot for potential accessibility color contrast issues, including:
   - Text-to-background contrast ratios that may not meet WCAG guidelines
   - Color-only information that may be hard for colorblind users to distinguish  
   - Low contrast UI elements like buttons, links, or form controls
   - Any visual elements that rely solely on color to convey information

4. If accessibility issues are found:
   - Document the specific issues with their locations on the page
   - Create a detailed issue with recommendations for improvement
   - Include the screenshot as evidence

5. If no significant issues are found, simply report the successful completion of the accessibility audit

Use the create-issue safe output if any accessibility color contrast problems are discovered that should be addressed.