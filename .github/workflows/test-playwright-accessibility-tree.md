---
on:
  workflow_dispatch:

permissions: read-all

engine:
  id: claude

tools:
  playwright:
    docker_image_version: "latest"
    allowed_domains: ["github.com", "*.github.com"]

safe-outputs:
  create-issue:
    title-prefix: "[Accessibility Tree] "
    labels: [accessibility, automation, playwright, wcag, a11y]
    max: 3
---

# Accessibility Tree Analysis and WCAG Compliance Scan

Analyze the accessibility tree of the GitHub repository page at github.com/githubnext/gh-aw to identify structural accessibility issues and WCAG compliance problems. Create issues for any significant accessibility problems found.

## Task Steps

1. **Navigate to Repository Page**:
   - Open https://github.com/githubnext/gh-aw using Playwright browser automation
   - Wait for the page to fully load and stabilize

2. **Extract Accessibility Tree Information**:
   - Get the complete accessibility tree structure of the page
   - Extract ARIA roles, labels, and properties for all interactive elements
   - Identify landmarks, headings hierarchy, and navigational structure
   - Capture form controls and their labeling associations

3. **Analyze for WCAG 2.1 AA Compliance Issues**:
   - **Missing Alt Text**: Images without proper alternative text descriptions
   - **Heading Structure**: Improper heading hierarchy (h1, h2, h3 sequence)
   - **Form Labels**: Form controls without associated labels or ARIA descriptions
   - **ARIA Landmarks**: Missing or improperly used ARIA landmark roles
   - **Keyboard Navigation**: Elements that are not keyboard accessible
   - **Focus Management**: Missing or inadequate focus indicators
   - **Color Dependence**: Information conveyed only through color
   - **Link Context**: Links with insufficient descriptive text
   - **Button Labels**: Buttons without clear accessible names

4. **Document Specific Issues**:
   - For each accessibility problem found, record:
     - Element location and selector
     - Current accessibility tree information
     - WCAG guideline violation
     - Severity level (Critical, High, Medium, Low)
     - Recommended fix

5. **Create Issues for Problems**:
   - Group related accessibility issues into comprehensive reports
   - Prioritize critical and high-severity issues
   - Include specific WCAG guidelines references
   - Provide actionable remediation steps
   - Use the create-issue safe output for significant problems

6. **Summary Report**:
   - If no critical issues found, report successful accessibility audit completion
   - If issues found, summarize the types and count of problems discovered

Use the create-issue safe output for any accessibility tree problems that violate WCAG guidelines and should be addressed to improve the repository page's accessibility.