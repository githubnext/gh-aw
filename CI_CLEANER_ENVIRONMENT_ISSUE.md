# CI Cleaner Environment Issue

## Problem

The hourly CI cleaner workflow (run #279) encountered an environment setup issue where the agentic execution container does not have access to required development tools:

- **Missing**: `make`, `go`, `node` (or not in PATH)
- **CI Run ID that failed**: 20435224725
- **Current workflow run**: 20435733509

## Root Cause

The workflow `.github/workflows/hourly-ci-cleaner.md` has setup steps (lines 46-65) that install:
1. Make (`sudo apt-get install -y make`)
2. Go (via `actions/setup-go@v6`)
3. Node.js (via `actions/setup-node@v6`)
4. npm dependencies
5. dev dependencies (`make deps-dev`)

However, these setup steps run in the GitHub Actions runner environment, while the agentic execution happens in an isolated container that doesn't inherit the installed tools.

## Environment Details

- **OS**: Ubuntu 22.04.5 LTS
- **User**: awfuser (uid=1001, gid=1001)
- **Working Directory**: /home/runner/work/gh-aw/gh-aw
- **Available in PATH**: node, npm (but not go or make)

## Solution Options

### Option 1: Move tools installation into agent execution
The agent should install required tools at the start of execution:
```bash
sudo apt-get update -qq
sudo apt-get install -y make golang-go
```

However, this requires sudo access in the agentic container.

### Option 2: Use containerized agent with pre-installed tools
Create a custom Docker image with all required development tools and use it for the agentic execution step.

### Option 3: Run commands directly without make
Since the Makefile is just a wrapper, the agent could run the underlying commands directly:
- `go fmt ./...` instead of `make fmt-go`
- `golangci-lint run ./...` instead of `make golint`
- `go test -v -timeout=3m ./...` instead of `make test`

This requires the agent to have Go and golangci-lint available.

## Recommended Fix

**Option 2** is recommended: Configure the agentic workflow to use a container image that includes:
- make
- Go (matching version in go.mod)
- Node.js (version 24)
- npm/npx
- golangci-lint

This ensures the agent has all tools available without requiring installation at runtime.

## Immediate Action

Since the agent cannot proceed with CI cleanup due to missing tools, this PR documents the issue for manual investigation and resolution.

The CI failure (run #20435224725) should be investigated manually to determine what needs to be fixed.
