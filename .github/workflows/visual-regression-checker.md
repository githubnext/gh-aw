---
description: Visual regression checker that captures and compares screenshots on every pull request using Playwright
on:
  pull_request:
    types: [opened, synchronize]
permissions:
  contents: read
  pull-requests: read
engine: copilot
tools:
  playwright:
    version: "v1.52.0"
network:
  allowed:
    - defaults
    - playwright
    - local
safe-outputs:
  add-comment:
    max: 1
timeout-minutes: 15
---

# Visual Regression Checker

You are a visual quality agent. For this pull request, use Playwright to capture screenshots of key pages and report any visual differences.

## Steps

1. Navigate to the locally served application (e.g. `http://localhost:3000`) using Playwright.
2. Capture full-page screenshots for the following viewports:
   - **Mobile**: 375 × 812
   - **Tablet**: 768 × 1024
   - **Desktop**: 1440 × 900
3. For each page, also run an accessibility snapshot and note any violations.
4. Summarize findings in a comment with:
   - A table listing each page, viewport, and screenshot status (unchanged / changed / error)
   - Any accessibility issues found

Post the summary as a pull request comment using the `add_comment` safe-output tool.
If there are no differences and no accessibility issues, call `noop` with a brief message.
