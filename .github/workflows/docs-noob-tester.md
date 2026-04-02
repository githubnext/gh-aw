---
name: Documentation Noob Tester
description: Tests documentation as a new user would, identifying confusing or broken steps in getting started guides
on:
  schedule: daily
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
timeout-minutes: 45
runtimes:
  node:
    version: "22"
tools:
  timeout: 120  # Playwright navigation on Astro dev server can take >60s; increase to 120s
  playwright:
  edit:
  bash:
    - "*"
safe-outputs:
  upload-asset:
network:
  allowed:
    - defaults
    - node

imports:
  - uses: shared/daily-audit-discussion.md
    with:
      title-prefix: "[docs-noob-tester] "
      expires: 1d
  - shared/docs-server-lifecycle.md
  - shared/reporting.md
  - shared/keep-it-short.md
features:
  copilot-requests: true
---

# Documentation Noob Testing

You are a brand new user trying to get started with GitHub Agentic Workflows for the first time. Your task is to navigate through the documentation site, follow the getting started guide, and identify any confusing, broken, or unclear steps.

## Context

- Repository: ${{ github.repository }}
- Working directory: ${{ github.workspace }}
- Documentation directory: ${{ github.workspace }}/docs

## Your Mission

Act as a complete beginner who has never used GitHub Agentic Workflows before. Build and navigate the documentation site, follow tutorials step-by-step, and document any issues you encounter.

## Step 1: Build and Serve Documentation Site

Navigate to the docs folder and start the documentation site:

```bash
cd ${{ github.workspace }}/docs
npm install
```

Follow the shared **Documentation Server Lifecycle Management** instructions:
1. Start the preview server (section "Starting the Documentation Preview Server")
2. Wait for server readiness (section "Waiting for Server Readiness")

**Get the bridge IP for Playwright access** (run this after the server is ready):

```bash
SERVER_IP=$(ip -4 route get 1.1.1.1 2>/dev/null | awk '{print $7; exit}')
if [ -z "$SERVER_IP" ]; then SERVER_IP=$(hostname -I | awk '{print $1}'); fi
echo "Playwright server URL: http://${SERVER_IP}:4321/gh-aw/"
```

Use `http://${SERVER_IP}:4321/gh-aw/` (NOT `localhost:4321`) for all Playwright navigation below.

## Step 2: Navigate Documentation as a Noob

**IMPORTANT: Using Playwright in gh-aw Workflows**

Playwright is provided through an MCP server interface. Use the bridge IP obtained in Step 1 for all navigation:

- ✅ **Correct**: `browser_run_code` with `page.goto(url, { waitUntil: 'domcontentloaded', timeout: 30000 })`
- ✅ **Correct**: `browser_navigate` to `http://${SERVER_IP}:4321/gh-aw/` (use the bridge IP, NOT localhost)
- ❌ **Incorrect**: Using `http://localhost:4321/...` — Playwright runs with `--network host` so its localhost is the Docker host, not the agent container

**⚠️ Playwright Connectivity — If Playwright times out or fails:**
If `browser_navigate` or `browser_run_code` returns `net::ERR_CONNECTION_TIMED_OUT` or a timeout error, **do not attempt to debug the network or install alternative browsers** (chromium, puppeteer, etc.). This is a known network isolation constraint. Instead:
1. Skip the Playwright navigation step immediately
2. Use the following command to fetch and analyze page content via curl:
   ```bash
   curl -s http://localhost:4321/gh-aw/ | python3 -c "
   import sys, re
   html = sys.stdin.read()
   text = re.sub(r'<[^>]+>', '', html)
   print(text[:5000])
   "
   ```
3. Note in the report that visual screenshots were unavailable

**⚠️ CRITICAL: Navigation Timeout Prevention**

The Astro development server loads many JavaScript modules per page. Always use `waitUntil: 'domcontentloaded'`:

```javascript
// ALWAYS use domcontentloaded - replace SERVER_IP with the actual IP from Step 1
mcp__playwright__browser_run_code({
  code: `async (page) => {
    await page.goto('http://SERVER_IP:4321/gh-aw/', { waitUntil: 'domcontentloaded', timeout: 30000 });
    return { url: page.url(), title: await page.title() };
  }`
})
```

Using Playwright, visit exactly these 3 pages and stop:

1. **Visit the home page** at `http://${SERVER_IP}:4321/gh-aw/`
   - Take a screenshot
   - Note: Is it immediately clear what this tool does?
   - Note: Can you quickly find the "Get Started" or "Quick Start" link?

2. **Follow the Quick Start Guide** at `http://${SERVER_IP}:4321/gh-aw/setup/quick-start/`
   - Take screenshots of each major section
   - Try to understand each step from a beginner's perspective
   - Questions to consider:
     - Are prerequisites clearly listed?
     - Are installation instructions clear and complete?
     - Are there any assumed knowledge gaps?
     - Do code examples work as shown?
     - Are error messages explained?

3. **Check the CLI Commands page** at `http://${SERVER_IP}:4321/gh-aw/setup/cli/`
   - Take a screenshot
   - Note: Are the most important commands highlighted?
   - Note: Are examples provided for common use cases?

After visiting all 3 pages, immediately proceed to the report.

## Step 3: Identify Pain Points

As you navigate, specifically look for:

### 🔴 Critical Issues (Block getting started)
- Missing prerequisites or dependencies
- Broken links or 404 pages
- Incomplete or incorrect code examples
- Missing critical information
- Confusing navigation structure
- Steps that don't work as described

### 🟡 Confusing Areas (Slow down learning)
- Unclear explanations
- Too much jargon without definitions
- Lack of examples or context
- Inconsistent terminology
- Assumptions about prior knowledge
- Layout or formatting issues that make content hard to read

### 🟢 Good Stuff (What works well)
- Clear, helpful examples
- Good explanations
- Useful screenshots or diagrams
- Logical flow

## Step 4: Take Screenshots

For each confusing or broken area:
- Take a screenshot showing the issue
- Name the screenshot descriptively (e.g., "confusing-quick-start-step-3.png")
- Note the page URL and specific section

## Step 5: Create Discussion Report

Create a GitHub discussion titled "📚 Documentation Noob Test Report - [Date]" with:

### Summary
- Date of test: [Today's date]
- Pages visited: [List URLs]
- Overall impression: [1-2 sentences as a new user]

### Critical Issues Found
[List any blocking issues with screenshots]

### Confusing Areas
[List confusing sections with explanations and screenshots]

### What Worked Well
[Positive feedback on clear sections]

### Recommendations
- Prioritized suggestions for improving the getting started experience
- Quick wins that would help new users immediately
- Longer-term documentation improvements

### Screenshots
[Embed all relevant screenshots showing issues or confusing areas]

Label the discussion with: `documentation`, `user-experience`, `automated-testing`

## Step 6: Cleanup

Follow the shared **Documentation Server Lifecycle Management** instructions for cleanup (section "Stopping the Documentation Server").

## Guidelines

- **Be genuinely naive**: Don't assume knowledge of Git, GitHub Actions, or AI workflows
- **Document everything**: Even minor confusion points matter
- **Be specific**: "This is confusing" is less helpful than "I don't understand what 'frontmatter' means"
- **Be constructive**: Focus on helping improve the docs, not just criticizing
- **Be thorough but efficient**: Cover key getting started paths without testing every single page
- **Take good screenshots**: Make sure they clearly show the issue

## Success Criteria

You've successfully completed this task if you:
- Navigated exactly 3 key documentation pages
- Identified specific pain points with examples
- Provided actionable recommendations
- Created a discussion with clear findings and screenshots

**Important**: If no action is needed after completing your analysis, you **MUST** call the `noop` safe-output tool with a brief explanation. Failing to call any safe-output tool is the most common cause of safe-output workflow failures.

```json
{"noop": {"message": "No action needed: [brief explanation of what was analyzed and why]"}}
```
