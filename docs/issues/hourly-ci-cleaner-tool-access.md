# Hourly CI Cleaner Workflow - Tool Access Issue

## Issue

The hourly-ci-cleaner workflow cannot execute because development tools (Go, make, Node.js) are not available in the agent execution environment.

## Root Cause

Agentic workflows run in two separate jobs:
1. **Activation job** - Installs tools via setup steps
2. **Agent job** - Runs in separate container without access to installed tools

The agent job only has access to tools specified in the `tools:` configuration (currently bash and edit), not tools installed in the activation job.

## Current Impact

- Workflow run #281 (and all subsequent runs) fail to perform CI cleanup
- Cannot execute: `make fmt`, `make lint`, `make test-unit`, `make recompile`
- Manual intervention required for all CI failures on main branch

## Evidence

Agent environment check shows tools not available:
```bash
$ which make && which go && which node && which npm
(all return: command not found)

$ id
uid=1001(awfuser) gid=1001(awfuser) groups=1001(awfuser),118(docker)
(no sudo access to install packages)
```

## Proposed Solution

Convert from agentic workflow to regular GitHub Actions workflow, as CI cleanup is a deterministic process that doesn't require AI decision-making.

### Why Regular GitHub Actions?

- ✅ Direct access to setup actions (setup-go, setup-node)
- ✅ Simpler execution model  
- ✅ Easier to debug and maintain
- ✅ Consistent with other CI workflows
- ❌ Less flexibility in decision-making (not needed for CI cleanup)

### Alternative Solutions

1. **Fix tool access in agentic workflows** - Investigate if tools from activation job can persist to agent job
2. **Use runtime configuration** - Check if `runtime:` field can specify required tools
3. **Hybrid approach** - Use GitHub Actions for commands, agentic for PR creation

## Status

- **Discovered**: 2025-12-22, Run #281
- **Impact**: High - Automated CI cleanup non-functional
- **Priority**: High - Requires immediate fix

## Temporary Workaround

Until fixed, manually run CI cleanup when main branch CI fails:
```bash
make fmt && make lint && make test-unit && make recompile
```

Then create PR with fixes.

## Files Affected

- `.github/workflows/hourly-ci-cleaner.md` - Workflow definition
- `.github/workflows/hourly-ci-cleaner.lock.yml` - Compiled workflow
- `.github/agents/ci-cleaner.agent.md` - Agent instructions
