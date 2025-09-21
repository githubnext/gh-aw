---
name: "dev"
on:
  workflow_dispatch: # do not remove this trigger
  push:
    branches:
      - copilot/*
      - pelikhan/*
safe-outputs:
  create-issue:
    title-prefix: "[docs] "
    labels: [documentation, accessibility, automation]
  staged: true
engine: 
  id: claude
  max-turns: 10
permissions: read-all
tools:
  playwright:
    allowed_domains: ["localhost", "127.0.0.1"]
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
---

# Documentation Build and Accessibility Analysis

This workflow compiles the documentation, launches the development server, takes a screenshot, and performs accessibility analysis.

Please follow these steps:

## Step 1: Prepare Documentation Build
1. Navigate to the `docs` directory in the repository
2. Temporarily modify the configuration files to disable the changelog functionality that requires GitHub API access:
   - Edit `src/content.config.ts` to remove the `changelogs` collection
   - Edit `astro.config.mjs` to remove starlight-changelogs imports and configuration
3. Install documentation dependencies with `npm install`

## Step 2: Build and Launch Documentation Server
1. Start the documentation development server using `npm run dev`
2. Wait for the server to fully start (it should be accessible on `http://localhost:4321/gh-aw/`)
3. Verify the server is running by making a curl request to test accessibility

## Step 3: Take Screenshot with Playwright
1. Use Playwright to navigate to `http://localhost:4321/gh-aw/`
2. Wait for the page to fully load
3. Take a full-page screenshot of the documentation homepage
4. Save the screenshot to a temporary file

## Step 4: Accessibility Analysis
1. Analyze the screenshot for accessibility issues, focusing on:
   - Color contrast ratios (WCAG 2.1 AA requirements: 4.5:1 for normal text, 3:1 for large text)
   - Text readability against background colors
   - Navigation elements visibility
   - Button and link contrast
   - Code block readability
   - Overall visual hierarchy and accessibility

## Step 5: Create Issue with Results
1. Use the `safe-outputs create-issue` functionality to create a GitHub issue
2. Include in the issue:
   - Summary of the documentation build process
   - Screenshot of the documentation homepage
   - Detailed accessibility analysis results
   - Any recommendations for improvements
   - Note that this was generated automatically by the dev workflow

## Step 6: Cleanup
1. Stop the development server
2. Restore the original configuration files
3. Clean up any temporary files

Focus on providing a comprehensive accessibility analysis that would be useful for improving the documentation's usability for all users.
