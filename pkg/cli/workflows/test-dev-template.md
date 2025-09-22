---
name: "test-dev-template"
on:
  workflow_dispatch: # do not remove this trigger
permissions: read-all
engine: 
  id: claude
  max-turns: 10
tools:
  playwright:
  bash:
    - "cd *"
    - "npm *"
    - "node *"
    - "curl *"
    - "mkdir *"
    - "cp *"
    - "mv *"
steps:
  - name: Checkout repository
    uses: actions/checkout@v5

  - name: Setup Node.js
    uses: actions/setup-node@v4
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

# Test Dev Template Workflow

This is a test workflow based on the current dev.md template. It demonstrates the key features of the dev template including:

- Node.js setup and documentation building steps
- Playwright browser automation capabilities  
- Extended bash tool permissions
- Documentation compilation and testing

## Test Instructions

Please follow these steps to test the dev template functionality:

### Step 1: Verify Repository Structure
1. Examine the repository structure, focusing on the `docs` directory
2. Check if the documentation build configuration exists

### Step 2: Documentation Build Test
1. Navigate to the `docs` directory
2. Verify that `package.json` and `package-lock.json` exist
3. Check the available npm scripts for building documentation

### Step 3: Basic Analysis
1. Provide a summary of the documentation build setup
2. Identify any potential issues with the current configuration
3. Suggest any improvements for the development workflow

This test helps validate that the dev template structure works correctly for documentation-focused workflows.