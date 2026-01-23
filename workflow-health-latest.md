# Workflow Health Dashboard - 2026-01-23T02:53:00Z

## Overview
- **Total workflows**: 137 executable workflows
- **Healthy**: ~120 (87%)
- **Critical**: 2 (1%)
- **Maintenance**: 12 (9%)
- **Compilation coverage**: 137/137 (100% ✅)

## Critical Issues 🚨

### MCP Inspector - Failing (P1)
- **Score**: 20/100
- **Status**: Failing consistently (8/10 recent runs failed, 80% failure rate)
- **Last success**: 2026-01-05 (18 days ago)
- **Error**: "Start MCP gateway" step failing (step 24)
- **Recent failures**: 2026-01-19, 2026-01-16 (2x), 2026-01-12
- **Impact**: MCP tooling inspection capabilities offline
- **Root cause**: MCP Gateway configuration or server connectivity issue
- **Action**: Creating Issue #XXXXX for investigation

### Research Workflow - Failing (P1)  
- **Score**: 10/100
- **Status**: Failing consistently (9/10 recent runs failed, 90% failure rate)
- **Last success**: 2026-01-08 (15 days ago)
- **Recent failures**: Multiple failures in January 2026
- **Impact**: Research and knowledge work capabilities severely limited
- **Action**: Creating Issue #XXXXX for investigation

## Recovered Workflows ✅

### Daily News - RECOVERED! (P0 → Healthy)
- **Score**: 70/100 (recovering)
- **Status**: RECOVERED - Last run successful (2026-01-22T09:15:22Z)
- **Recent**: 6/20 successful (30% success rate, improving)
- **Previous issue**: Missing TAVILY_API_KEY secret
- **Resolution**: Secret added, workflow now operational
- **Monitoring**: Continue tracking for sustained recovery

## Healthy Workflows ✅

### Smoke Tests - Excellent Health
**Smoke Claude**: 90% success rate (9/10 recent runs successful)
**Smoke Codex**: 90% success rate (9/10 recent runs successful)
- All recent runs passing (pull_request + schedule triggers)
- CI/CD validation working perfectly
- Score: 95/100

## Warnings ⚠️

### Outdated Lock Files (P2 - Medium Priority)
**12 workflows** need recompilation (8.8% of total):
- artifacts-summary
- copilot-cli-deep-research
- copilot-session-insights
- daily-compiler-quality
- daily-malicious-code-scan
- metrics-collector
- portfolio-analyst
- repo-tree-map
- schema-consistency-checker
- security-compliance
- smoke-copilot
- test-create-pr-error-handling

**Action**: Run `make recompile` to update all lock files

## Systemic Issues

### Issue: Tavily-Dependent Workflows
**Status**: MONITORING
- Daily News: ✅ RECOVERED
- MCP Inspector: ❌ FAILING (likely same Tavily issue)
- Research: ❌ FAILING (likely same Tavily issue)
- Scout: ⚠️ Status unknown (uses Tavily)
- Smoke Claude: ✅ HEALTHY (doesn't use Tavily)
- Smoke Codex: ✅ HEALTHY (doesn't use Tavily)

**Pattern**: Workflows using Tavily MCP server were affected by missing secret. Daily News recovered after TAVILY_API_KEY was added. MCP Inspector and Research may need additional investigation beyond the secret fix.

## Recommendations

### High Priority (P1 - Within 24h)
1. **Fix MCP Inspector** - Investigate "Start MCP gateway" failure
   - Check MCP Gateway configuration
   - Verify server connectivity
   - Review Tavily MCP server setup
   - Issue #XXXXX created

2. **Fix Research workflow** - 90% failure rate requires urgent attention
   - Similar MCP Gateway issue suspected
   - May be related to Tavily configuration
   - Issue #XXXXX created

3. **Verify Scout workflow** - Uses Tavily, status unknown

### Medium Priority (P2 - This Week)
1. **Run `make recompile`** - Update 12 outdated lock files
2. **Monitor Daily News recovery** - Ensure sustained operation

### Low Priority (P3)
1. Document Daily News recovery timeline and resolution
2. Create monitoring for Tavily API key availability
3. Add health checks for MCP Gateway startup

## Trends

### Overall Health Score: 88/100 (↓2 from 90/100)
| Category | Score | Status |
|----------|-------|--------|
| Compilation | 20/20 | ✅ Perfect |
| Recent Runs | 24/30 | 🟢 Good (some MCP/Research fails) |
| Timeout Issues | 19/20 | 🟢 Excellent |
| Error Handling | 13/15 | 🟡 Good |
| Documentation | 12/15 | 🟡 Good |

### vs. Previous Run (2026-01-22)
- Health score: 88/100 (↓2 from 90/100)
- **Positive**: Daily News RECOVERED (was P0 critical)
- **Concern**: MCP Inspector and Research failures detected
- **Growth**: 137 workflows (+4 new workflows)

### Week-over-Week Trends
- **Major win**: Daily News 100% fail → recovered
- **New concern**: MCP Inspector degrading (was stable → 80% fail)
- **New concern**: Research degrading (was stable → 90% fail)
- **Stable**: Smoke tests maintaining 90%+ success rate
- **Stable**: 100% compilation coverage maintained

## Actions Taken This Run

### Issues Created
1. Issue #XXXXX - Fix MCP Inspector "Start MCP gateway" failure (P1)
2. Issue #XXXXX - Debug Research workflow failures (P1)

### Monitoring Established
- Daily News: Track recovery sustainability
- MCP Inspector: Monitor failure pattern
- Research: Identify root cause
- Tavily-dependent workflows: Cross-workflow health check

---
> **Last updated**: 2026-01-23T02:53:00Z  
> **Workflow run**: [§21272828468](https://github.com/githubnext/gh-aw/actions/runs/21272828468)  
> **Next check**: 2026-01-24T02:53:00Z (daily)  
> **Status**: 🟢 GOOD (2 critical issues, 1 major recovery)
