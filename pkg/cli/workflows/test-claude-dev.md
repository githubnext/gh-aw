---
on:
  workflow_dispatch:
permissions:
  contents: read
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
    - "ps *"
    - "mkdir *"
    - "cp *"
    - "mv *"
safe-outputs:
  create-issue:
    title-prefix: "[test] "
---

# Test Claude Dev Workflow

This is a test workflow based on the dev.md workflow to verify Claude functionality with Playwright and accessibility analysis capabilities.

This workflow tests the integration of:
- Claude engine with extended max-turns
- Playwright browser automation tools
- Bash command execution with specific allowed commands
- Safe outputs for issue creation

Please follow these simplified steps:

## Step 1: Repository Analysis
1. Analyze the current repository structure
2. Identify the main directories and their purposes
3. Focus on the documentation structure if present

## Step 2: Simple Web Page Test
1. If possible, use Playwright to navigate to a simple web page or local file
2. Take a basic screenshot
3. Perform a simple accessibility check on the visual elements

## Step 3: Create Test Issue
1. Use the safe-outputs create-issue functionality to create a test issue
2. Include in the issue:
   - Repository structure analysis
   - Any findings from the web page test
   - Confirmation that the workflow executed successfully

This is a simplified version of the dev workflow for testing purposes, focusing on validating the core agentic workflow functionality rather than complex documentation builds.