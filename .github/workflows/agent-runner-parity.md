---
description: Agent-Runner Environment Parity Test
on:
  schedule: every 6h
  workflow_dispatch:
  pull_request:
    types: [labeled]
    names: ["test-parity"]
  reaction: "rocket"
permissions:
  contents: read
  issues: read
  pull-requests: read
name: Agent-Runner Parity
engine: copilot
network:
  allowed:
    - defaults
    - github
tools:
  bash:
    - "*"
safe-outputs:
  add-comment:
    max: 1
  messages:
    run-started: "🚀 Testing agent-runner environment parity..."
    run-success: "✅ Environment parity test completed successfully"
    run-failure: "❌ Environment parity test failed: {status}"
timeout-minutes: 10
strict: true
---

# Agent-Runner Environment Parity Test

This workflow validates that the agent container environment has parity with the GitHub Actions runner environment for essential tools, libraries, and environment variables.

## Test Categories

### 1. Utilities Accessibility (10+ utilities)

Verify the following utilities are accessible via `which`:
- `jq` - JSON processor
- `curl` - HTTP client
- `git` - Version control
- `wget` - File downloader
- `tar` - Archive utility
- `gzip` - Compression utility
- `unzip` - Archive extractor
- `sed` - Stream editor
- `awk` - Pattern processor
- `grep` - Text search
- `find` - File finder
- `xargs` - Argument builder

**Command**: `which <utility>`

### 2. Runtime Availability (4 runtimes)

Verify the following runtimes are available and can execute:
- `node --version` - Node.js runtime
- `python3 --version` - Python interpreter
- `go version` - Go compiler
- `ruby --version` - Ruby interpreter

### 3. Environment Variables (5+ variables)

Check that essential environment variables are set:
- `JAVA_HOME` - Java installation directory
- `ANDROID_HOME` - Android SDK directory
- `GOROOT` - Go installation root
- `PATH` - Executable search path
- `HOME` - User home directory

**Command**: `printenv <VAR>` or `echo $<VAR>`

### 4. Shared Library Linking

Use `ldd` to verify shared libraries can be loaded for key binaries:
- `/usr/bin/python3`
- `/usr/bin/node`
- `/usr/bin/git`
- `/usr/bin/curl`

**Command**: `ldd <binary>` (check for "not found" errors)

## Testing Instructions

For each test category:

1. Run the verification commands using bash
2. Record which items **PASS** ✅ and which **FAIL** ❌
3. For failures, include the error message or missing item

## Reporting

Create a summary report with:
- Total tests run
- Pass/Fail counts by category
- List of failed items (if any)
- Overall status (PASS if all tests pass, FAIL otherwise)

**Keep the report concise** - only list failures in detail.

## Example Output Format

```
🔍 Agent-Runner Environment Parity Test Results

Utilities: 12/12 ✅
Runtimes: 4/4 ✅
Environment Variables: 5/5 ✅
Shared Libraries: 4/4 ✅

Overall: PASS ✅
```

Or if there are failures:

```
🔍 Agent-Runner Environment Parity Test Results

Utilities: 11/12 ❌
  - Missing: unzip

Runtimes: 4/4 ✅
Environment Variables: 4/5 ❌
  - Missing: ANDROID_HOME
  
Shared Libraries: 4/4 ✅

Overall: FAIL ❌
```

## Post a summary comment with the results using safe-outputs add-comment
