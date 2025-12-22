# CI Cleaner Architecture Issue

## Problem

The hourly CI cleaner workflow cannot execute because the agentic execution environment lacks required build tools (make, go, node).

## Root Cause

The `hourly-ci-cleaner.md` workflow has a hybrid architecture where:

1. Standard GitHub Actions steps install build tools
2. Agentic workflow assumes tools are pre-installed
3. Agent runs in separate container without access to installed tools

## Environment Analysis

**Available in agentic environment:**
- ✅ `gh` CLI (v2.83.2)
- ✅ `git`
- ✅ `curl`
- ❌ `make` - Not found
- ❌ `go` - Not found
- ❌ `node` - Not found
- ❌ `npm` - Not found

## Recommended Solutions

### Option 1: Inline Build Commands
Replace `make` commands with direct tool invocations in bash scripts.

### Option 2: Pre-build Binary
Build `gh-aw` binary in setup steps and make it available to agent.

### Option 3: Custom Container
Use a container image with pre-installed build tools for the agentic step.

### Option 4: Redesign Workflow
Split into separate workflows - one for detection, one for fixing (running on standard runner).

## Impact

The hourly CI cleanup workflow is currently non-functional in its current design.

## Date Identified

2025-12-22
