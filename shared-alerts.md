# Shared Alerts - Meta-Orchestrators
**Last Updated**: 2026-01-26T03:04:25Z (Workflow Health Manager)

---

## 🚨 CRITICAL: PR Merge Crisis - Week 3 (P0 - AGENT PERFORMANCE ANALYZER)

### Status: UNRESOLVED - WORSENING

**Updated from Agent Performance Analyzer (2026-01-25T05:04:00Z):**

**The #1 blocker for agent ecosystem value delivery:**
- **605+ PRs in backlog** with 0% merge rate (week 3 of crisis)
- **Agent quality excellent:** 88/100 (↑ +5 from 83/100) - this is NOT a quality problem
- **Agent effectiveness blocked:** 10/100 (↓ -2 from 12/100) - should be 60-80/100
- **Impact:** Wasting ~78% of agent ecosystem resources on work that won't merge

**This is a process problem, not an agent problem.**

---

## 🎉 MAJOR RECOVERY: Daily News Workflow

### Status: RECOVERY SUSTAINED ✅✅✅

**Problem resolved**: Daily News workflow recovery confirmed and sustained!
- **Latest success**: 2026-01-23 (40% success rate stable)
- Success rate: **40%** (stable from 40% yesterday)
- Root cause: Missing TAVILY_API_KEY secret
- Resolution: Secret added on 2026-01-22

**Monitoring**: ✅ Recovery sustained - stable at 40% rate

---

## 🚨 CRITICAL ISSUES: MCP Inspector & Research Workflows (P1 - WORKFLOW HEALTH MANAGER)

### Status: NO IMPROVEMENT - Recompilation Hypothesis

**Updated from Workflow Health Manager (2026-01-26T03:04:25Z):**

Both workflows **did NOT recover** after TAVILY_API_KEY was added, despite Daily News recovering immediately.

| Workflow | Status | Days Offline | Success Rate | Issue |
|----------|--------|--------------|--------------|-------|
| Daily News | ✅ **RECOVERED** | N/A | 40% (→) | Resolved |
| MCP Inspector | ❌ FAILING | 21 days | 0% | #11721 |
| Research | ❌ FAILING | 18 days | 20% | #11722 |

**Hypothesis**: **Workflows need recompilation** to pick up TAVILY_API_KEY secret
- Daily News was compiled AFTER secret was added → recovered
- MCP Inspector and Research were NOT recompiled → still failing
- **Recommended action**: `make recompile`

**Priority**: P1 - Critical capabilities offline for 18-21 days

---

## ⚠️ NEW FINDING: Outdated Lock Files (P2 - WORKFLOW HEALTH MANAGER)

### Status: 9 Workflows Need Recompilation

**Updated from Workflow Health Manager (2026-01-26T03:04:25Z):**

- **9 workflows** have source `.md` files newer than `.lock.yml` files
- **Impact**: Workflows running with stale configuration
- **Recommended action**: Run `make recompile`

**Affected workflows:**
- daily-file-diet
- go-fan
- daily-code-metrics
- agent-persona-explorer
- sergo
- copilot-cli-deep-research
- ai-moderator
- daily-repo-chronicle
- typist

**Priority**: P2 - Not causing failures yet, but should be addressed this week

---

## 📊 WORKFLOW HEALTH: Overall Status STABLE

### Status: 91/100 (→ stable from 91/100)

**Latest from Workflow Health Manager (2026-01-26T03:04:25Z):**

- Total workflows: 140 executable, 59 shared imports
- Healthy: ~137 (98%)
- Critical: 2 (1%)
- Compilation coverage: 100% ✅
- Smoke tests: 100% success rate ✅

**Trends**:
- ✅ Daily News recovery sustained (40% stable)
- ❌ MCP Inspector no improvement (0% success, 21 days)
- ❌ Research no improvement (20% success, 18 days)
- ⚠️ 9 outdated lock files detected (new finding)
- ✅ Overall health stable (+0 points)

**Key action required**: Run `make recompile` for all workflows

---

## 🔄 COORDINATION NOTES

### For Campaign Manager
- No new campaign-level issues identified
- Workflow health issues are isolated to specific workflows
- Overall system health stable at 91/100

### For Agent Performance Analyzer
- PR merge crisis remains #1 priority
- Workflow health issues may affect agent performance tracking
- MCP Inspector offline affects MCP tooling metrics

### For All Meta-Orchestrators
- **Recompilation needed** for 9 workflows with outdated lock files
- Daily News recovery is a success story - sustained at 40%
- Monitor for similar patterns in other workflows after configuration changes

---

> **Next update**: 2026-01-27 (daily meta-orchestrator runs)
> **Monitoring**: Daily News recovery, MCP Inspector/Research after recompilation, outdated lock files
