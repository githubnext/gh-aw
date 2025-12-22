---
name: "dev"
on:
  workflow_dispatch: # do not remove this trigger
  push:
    branches:
      - copilot/*
      - pelikhan/*
safe-outputs:
  staged: true
  upload-asset:
  create-issue:
    title-prefix: "[docs] "
engine: 
  id: copilot
permissions: read-all
tools:
  playwright:
  bash:
    - "cd *"
    - "npm *"
    - "node *"
    - "curl *"
    - "ps *"
    - "kill *"
    - "sleep *"
    - "mkdir *"
    - "cp *"
    - "mv *"
steps:
  - name: Checkout repository
    uses: actions/checkout@v5

  - name: Setup Node.js
    uses: actions/setup-node@v6
    with:
      node-version: '24'
      cache: 'npm'
      cache-dependency-path: 'docs/package-lock.json'

  - name: Install dependencies
    working-directory: ./docs
    run: npm ci

  - name: Build documentation
    working-directory: ./docs
    env:
      GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: npm run build
---

# Documentation Build and Accessibility Analysis

This workflow compiles the documentation, launches the development server, takes a screenshot, and performs accessibility analysis.

Please follow these steps:

## Step 2: Build and Launch Documentation Server
0. Go to the `docs` directory
1. Start the documentation development server using `npm run dev`
2. Wait for the server to fully start (it should be accessible on `http://localhost:4321/gh-aw/`)
3. Verify the server is running by making a curl request to test accessibility

## Step 3: Take Screenshot with Playwright
1. Use Playwright to navigate to `http://localhost:4321/gh-aw/`
2. Wait for the page to fully load
3. Take a full-page screenshot of the documentation homepage
4. Save the screenshot to a file (e.g., `/tmp/gh-aw/docs-screenshot.png`)

## Step 4: Upload Screenshot
1. Use the `upload asset` tool from safe-outputs to upload the screenshot file
2. The tool will return a URL for the uploaded screenshot that can be included in the issue

## Step 5: Accessibility Analysis
1. Analyze the screenshot **image** for accessibility issues. DO NOT ANALYZE THE HTML. Focusing on:
   - Color contrast ratios (WCAG 2.1 AA requirements: 4.5:1 for normal text, 3:1 for large text)
   - Text readability against background colors
   - Navigation elements visibility
   - Button and link contrast
   - Code block readability
   - Overall visual hierarchy and accessibility

## Step 6: Create Issue with Results
1. Use the `safe-outputs create-issue` functionality to create a GitHub issue
2. Include in the issue:
   - The uploaded screenshot URL from step 4
   - Detailed accessibility analysis results
   - Any recommendations for improvements

Focus on providing a comprehensive accessibility analysis that would be useful for improving the documentation's usability for all users.
