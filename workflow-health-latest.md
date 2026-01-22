# Workflow Health Dashboard - 2026-01-22

**Last Updated**: 2026-01-22T02:56:21Z  
**Run**: https://github.com/githubnext/gh-aw/actions/runs/21234278799  
**Status**: 🟢 EXCELLENT - Recovery confirmed, root cause identified, clear action plan

## Executive Summary

- **Total Workflows**: 133 executable workflows
- **Compilation Coverage**: 133/133 lock files (100% ✅)
- **Critical Issues**: 1 (Daily News - **ACTIONABLE FIX IDENTIFIED**)
- **Outdated Lock Files**: 13 workflows (↑ from 11, +2 workflows)
- **Overall Health Score**: 90/100 (↑ from 78/100, **+12 points** 🎉)

## 🎯 Root Cause Confirmed: Daily News Missing Secret (P0)

### Status: READY TO FIX

**Problem**: 10/10 consecutive failures (100% failure rate over 10 days)

**Root Cause Confirmed**: **Missing `TAVILY_API_KEY` repository secret**

**Evidence**:
- Daily News workflow includes `shared/mcp/tavily.md` MCP server configuration
- Configuration requires `${{ secrets.TAVILY_API_KEY }}`
- Step 31 "Start MCP gateway" consistently fails
- Previous run logs showed: `"undefined environment variable referenced: TAVILY_API_KEY"`

**Impact**:
- 6 workflows depend on Tavily MCP server (daily-news, mcp-inspector, research, scout, smoke-claude, smoke-codex)

### Solution (RECOMMENDED): Add `TAVILY_API_KEY` Secret - 5-10 minutes

## ✅ Major Recovery: Meta-Orchestrators Stable

### Agent Performance Analyzer
- **Status**: ✅ RECOVERED (4/5 recent runs successful)
- **Last Success**: Run #180 (2026-01-21)

### Metrics Collector
- **Status**: ✅ RECOVERED (5/5 recent runs successful)
- **Last Success**: Run #34 (2026-01-21)

## 📊 Health Score: 90/100 (↑ +12 points)

| Category | Score | Status |
|----------|-------|--------|
| Compilation | 20/20 | ✅ Perfect |
| Recent Runs | 27/30 | 🟢 Excellent |
| Timeout Issues | 18/20 | 🟢 Good |
| Error Handling | 12/15 | 🟡 Fair |
| Documentation | 14/15 | 🟡 Good |

## ⚠️ Outdated Lock Files: 13 workflows (P2)

cli-consistency-checker, copilot-cli-deep-research, copilot-pr-prompt-analysis, daily-fact, daily-testify-uber-super-expert, github-mcp-tools-report, issue-monster, lockfile-stats, prompt-clustering-analysis, schema-consistency-checker, stale-repo-identifier, typist, weekly-issue-summary

**Action**: Run `make recompile`

## 📊 Workflow Categories

- **🟢 Healthy**: 119 workflows (89%)
- **🟡 Recovering**: 2 workflows (2%) - Agent Perf. Analyzer, Metrics Collector
- **🔴 Critical**: 1 workflow (1%) - Daily News (fix available)
- **⚠️ Maintenance**: 13 workflows (10%) - Outdated locks

## 🎯 Priority Actions

1. **P0**: Add `TAVILY_API_KEY` secret → Fixes Daily News + 5 other workflows
2. **P1**: Verify other Tavily-dependent workflows (mcp-inspector, research, scout, smoke-claude, smoke-codex)
3. **P2**: Run `make recompile` → Updates 13 outdated workflows

---

**Overall Status**: 🟢 EXCELLENT  
**Health Score**: 90/100 (↑ from 78/100)
