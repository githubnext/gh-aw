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
timeout-minutes: 30
runtimes:
  node:
    version: "22"
tools:
  mount-as-clis: true
  timeout: 120  # Playwright navigation on Astro dev server can take >60s; increase to 120s
  playwright:
  edit:
  bash:
    - "*"
safe-outputs:
  upload-asset:
    max: 10
    allowed-exts: [.png, .jpg, .jpeg, .svg]
network:
  allowed:
    - defaults
    - node

imports:
  - uses: shared/daily-audit-base.md
    with:
      title-prefix: "[docs-noob-tester] "
      expires: 1d
  - shared/docs-server-lifecycle.md
  - shared/keep-it-short.md
pre-agent-steps:
  - name: Install docs dependencies
    run: |
      cd "${{ github.workspace }}/docs"
      npm install
  - name: Start documentation server
    run: |
      cd "${{ github.workspace }}/docs"
      nohup npm run dev -- --host 0.0.0.0 --port 4321 > /tmp/preview.log 2>&1 &
      PID=$!
      echo $PID > /tmp/server.pid
      echo "Server PID: $PID"
  - name: Wait for server readiness
    run: |
      for i in {1..45}; do  # 45 attempts × 3s = 135s max wait
        STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:4321/gh-aw/)
        [ "$STATUS" = "200" ] && echo "Server ready at http://localhost:4321/gh-aw/!" && break
        echo "Waiting for server... ($i/45) (status: $STATUS)" && sleep 3
      done
  - name: Detect bridge IP and write server URL
    run: |
      SERVER_IP=$(ip -4 route get 1.1.1.1 2>/dev/null | awk '{print $7; exit}')
      if [ -z "$SERVER_IP" ]; then SERVER_IP=$(hostname -I | awk '{print $1}'); fi
      mkdir -p /tmp/gh-aw/agent
      echo "http://${SERVER_IP}:4321/gh-aw/" > /tmp/gh-aw/agent/server-url.txt
      echo "Playwright server URL: http://${SERVER_IP}:4321/gh-aw/"
features:
  mcp-cli: true
  copilot-requests: true
---

# Documentation Noob Testing

You are a brand new user trying to get started with GitHub Agentic Workflows for the first time. Your task is to navigate through the documentation site, follow the getting started guide, and identify any confusing, broken, or unclear steps.

## Context

- Repository: ${{ github.repository }}
- Working directory: ${{ github.workspace }}
- Documentation directory: ${{ github.workspace }}/docs

## Your Mission

Act as a complete beginner who has never used GitHub Agentic Workflows before. Navigate the documentation site, follow tutorials step-by-step, and document any issues you encounter.

> The documentation server is already running. Read the Playwright server URL from `/tmp/gh-aw/agent/server-url.txt`:
> ```bash
> cat /tmp/gh-aw/agent/server-url.txt
> ```

## Step 1: Navigate Documentation as a Noob

**IMPORTANT: Using Playwright in gh-aw Workflows**

- ✅ **Correct**: Read the URL from the file then navigate: `SERVER_URL=$(cat /tmp/gh-aw/agent/server-url.txt)` and use `browser_run_code` with `page.goto(SERVER_URL, { waitUntil: 'domcontentloaded', timeout: 30000 })`
- ❌ **Incorrect**: Using `http://localhost:4321/...` — use the bridge IP from `/tmp/gh-aw/agent/server-url.txt`

**⚠️ CRITICAL: Navigation Timeout Prevention** — Always use `waitUntil: 'domcontentloaded'` to prevent timeout on the Astro development server. If Playwright connectivity fails, see the shared **Documentation Server Lifecycle Management** fallback instructions.

Using Playwright, visit exactly these 3 pages and stop (use the Playwright server URL read from `/tmp/gh-aw/agent/server-url.txt`):

1. **Visit the home page** (e.g. `http://<BRIDGE_IP>:4321/gh-aw/`)
   - Take a screenshot
   - Note: Is it immediately clear what this tool does?
   - Note: Can you quickly find the "Get Started" or "Quick Start" link?

2. **Follow the Quick Start Guide** (e.g. `http://<BRIDGE_IP>:4321/gh-aw/setup/quick-start/`)
   - Take screenshots of each major section
   - Try to understand each step from a beginner's perspective
   - Questions to consider:
     - Are prerequisites clearly listed?
     - Are installation instructions clear and complete?
     - Are there any assumed knowledge gaps?
     - Do code examples work as shown?
     - Are error messages explained?

3. **Check the CLI Commands page** (e.g. `http://<BRIDGE_IP>:4321/gh-aw/setup/cli/`)
   - Take a screenshot
   - Note: Are the most important commands highlighted?
   - Note: Are examples provided for common use cases?

After visiting all 3 pages, immediately proceed to the report.

## Step 2: Identify Pain Points

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

## Step 3: Take Screenshots

For each confusing or broken area:
- Take a screenshot showing the issue
- Save it to a descriptive filename (e.g., "confusing-quick-start-step-3.png") in `/tmp/gh-aw/screenshots/`
- Note the page URL and specific section
- Upload the screenshot by calling the `upload_asset` safe-output tool with the absolute file path `path: "/tmp/gh-aw/screenshots/<filename>.png"`.
  Record the returned asset URL.

## Step 4: Create Discussion Report

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
For each uploaded screenshot, include its asset URL. Format:
```
📎 **[filename.png]** — asset URL: https://github.com/.../blob/.../filename.png?raw=true
```

Label the discussion with: `documentation`, `user-experience`, `automated-testing`

## Step 5: Cleanup

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

{{#import shared/noop-reminder.md}}
