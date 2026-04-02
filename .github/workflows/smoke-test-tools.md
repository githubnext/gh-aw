---
description: Smoke test to validate common development tools are available in the agent container
on: 
  workflow_dispatch:
  schedule: every 12h
  pull_request:
    types: [labeled]
    names: ["smoke"]
  status-comment: true
permissions:
  contents: read
  issues: read
  pull-requests: read
name: Agent Container Smoke Test
engine: copilot
strict: true
runtimes:
  node:
    version: "20"
  python:
    version: "3.11"
  go:
    version: "1.24"
  java:
    version: "21"
  dotnet:
    version: "8.0"
network:
  allowed:
    - defaults
    - github
    - node
tools:
  bash:
    - "*"
safe-outputs:
    allowed-domains: [default-safe-outputs]
    add-comment:
      hide-older-comments: true
      max: 2
    messages:
      footer: "> 🔧 *Tool validation by [{workflow_name}]({run_url})*{effective_tokens_suffix}{history_link}"
      run-started: "🔧 Starting tool validation... [{workflow_name}]({run_url}) is checking the agent container tools..."
      run-success: "✅ All tools validated successfully! [{workflow_name}]({run_url}) confirms agent container is ready."
      run-failure: "❌ Tool validation failed! [{workflow_name}]({run_url}) detected missing tools: {status}"
timeout-minutes: 5
---

# Smoke Test: Agent Container Tools

**Purpose:** Quick validation that common development tools are accessible in the agent container environment.

**IMPORTANT:** Keep all outputs concise. Report each tool test with ✅ or ❌ status.

## Required Tool Tests

Run each command and verify it produces valid output:

1. **Shell Tools:**
   - `bash --version` - Verify Bash shell is available
   - `sh --version` or `sh -c 'echo ok'` - Verify sh shell works

2. **Version Control:**
   - `git --version` - Verify Git is available

3. **JSON/YAML Processing:**
   - `jq --version` - Verify jq is available for JSON processing
   - `yq --version` - Verify yq is available for YAML processing

4. **HTTP Tools:**
   - `curl --version` - Verify curl is available for HTTP requests

5. **GitHub CLI:**
   - `gh --version` - Verify GitHub CLI is available

6. **Programming Runtimes:**
   - `node --version` - Verify Node.js runtime is available
   - `python3 --version` - Verify Python 3 runtime is available
   - `go version` - Verify Go runtime is available
   - `java --version` - Verify Java runtime is available
   - `dotnet --version` - Verify .NET runtime is available (C#)

## Output Requirements

After running all tests, add a **concise comment** to the pull request (if triggered by PR) with:

- Each tool name with ✅ (available) or ❌ (missing) status
- Total count: "X/12 tools available"
- Overall status: PASS (all tools found) or FAIL (any missing)

Example output format:
```
## Agent Container Tool Check

| Tool | Status | Version |
|------|--------|---------|
| bash | ✅ | 5.2.x |
| sh   | ✅ | available |
| git  | ✅ | 2.x.x |
| jq   | ✅ | 1.x |
| yq   | ✅ | 4.x |
| curl | ✅ | 8.x |
| gh   | ✅ | 2.x |
| node | ✅ | 20.x |
| python3 | ✅ | 3.x |
| go   | ✅ | 1.24.x |
| java | ✅ | 21.x |
| dotnet | ✅ | 8.x |

**Result:** 12/12 tools available ✅
```

## Error Handling

If any tool is missing:
1. Report which tool(s) are unavailable
2. Mark overall status as FAIL
3. Include the error message from the failed version check

**Important**: If no action is needed after completing your analysis, you **MUST** call the `noop` safe-output tool with a brief explanation. Failing to call any safe-output tool is the most common cause of safe-output workflow failures.

```json
{"noop": {"message": "No action needed: [brief explanation of what was analyzed and why]"}}
```
