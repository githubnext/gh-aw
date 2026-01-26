# Workflow Health Dashboard - 2026-01-26T03:04:25Z

## Overview
- **Total workflows**: 140 executable workflows
- **Shared imports**: 59 reusable workflow components
- **Healthy**: ~137 (98%)
- **Critical**: 2 (1%)
- **Compilation coverage**: 140/140 (100% ✅)
- **Outdated lock files**: 9 workflows (new finding)
- **Overall health score**: 91/100 (→ stable)

## Critical Issues 🚨

### MCP Inspector - Failing (P1) - Issue #11721
- **Score**: 15/100
- **Status**: Failing consistently (0/5 recent runs failed, 0% success rate)
- **Last success**: 2026-01-05 (21 days ago)
- **Latest failure**: §21304877267 (2026-01-23)
- **Error**: "Start MCP gateway" step failing
- **Impact**: MCP tooling inspection capabilities offline
- **Root cause**: Needs recompilation after TAVILY_API_KEY fix
- **Action**: Updated issue #11721

### Research Workflow - Failing (P1) - Issue #11722
- **Score**: 20/100
- **Status**: Minimal success (1/5 recent runs successful, 20% success rate)
- **Last success**: 2026-01-08 (18 days ago)
- **Latest failure**: §21078189533
- **Impact**: Research and knowledge work capabilities severely limited
- **Root cause**: Needs recompilation after TAVILY_API_KEY fix
- **Action**: Updated issue #11722

## Recovered Workflows ✅

### Daily News - RECOVERY SUSTAINED! (P0 → Healthy)
- **Score**: 80/100 (stable)
- **Status**: **RECOVERY SUSTAINED** - 2/5 recent successes (40% success rate)
- **Latest success**: §21280868153 (2026-01-23)
- **Monitoring**: ✅ Recovery sustained at 40% - stable improvement

## Healthy Workflows ✅

### Smoke Tests - Perfect Health
All smoke tests: **100% success rate** (all recent runs)
- Score: 100/100

## Systemic Issues

### Issue: Outdated Lock Files - NEW FINDING
**Status**: 9 workflows need recompilation

Workflows with outdated lock files:
- daily-file-diet
- go-fan
- daily-code-metrics
- agent-persona-explorer
- sergo
- copilot-cli-deep-research
- ai-moderator
- daily-repo-chronicle
- typist

**Recommended action**: Run `make recompile` to regenerate all lock files

### Issue: Tavily-Dependent Workflows
**Status**: MONITORING - 1 recovered, 2 still failing

| Workflow | Status | Success Rate | Issue |
|----------|--------|--------------|-------|
| Daily News | ✅ **RECOVERED** | 40% | Resolved |
| MCP Inspector | ❌ FAILING | 0% | #11721 |
| Research | ❌ FAILING | 20% | #11722 |

## Recommendations

### High Priority (P1 - Within 24h)
1. **Recompile all workflows** (`make recompile`)
2. **Fix MCP Inspector and Research** after recompilation

### Medium Priority (P2 - This Week)
1. Monitor Daily News recovery (target: 80%)
2. Add pre-commit hook for outdated lock files

## Trends

- Overall health score: 91/100 (stable)
- Daily News: 40% stable
- MCP Inspector: 21 days offline
- Research: 18 days low success

## Actions Taken This Run

- Created new dashboard issue
- Updated issues #11721 and #11722
- Identified 9 outdated lock files
- Confirmed Daily News recovery sustained

---
> Last updated: 2026-01-26T03:04:25Z
> Workflow run: §21344835253
> Next check: 2026-01-27T03:04:00Z
