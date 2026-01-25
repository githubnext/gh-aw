# Workflow Health Dashboard - 2026-01-25T03:04:00Z

## Overview
- **Total workflows**: 140 executable workflows
- **Shared imports**: 59 reusable workflow components
- **Healthy**: ~137 (98%)
- **Critical**: 2 (1%)
- **Compilation coverage**: 140/140 (100% ✅)
- **Overall health score**: 91/100 (↑1 from 90/100)

## Critical Issues 🚨

### MCP Inspector - Failing (P1) - New Issue Created
- **Score**: 15/100
- **Status**: Failing consistently (0/5 recent runs failed, 0% success rate)
- **Last success**: 2026-01-05 (20 days ago)
- **Latest failure**: §21304877267 (2026-01-23)
- **Error**: "Start MCP gateway" step failing
- **Impact**: MCP tooling inspection capabilities offline
- **Root cause**: Did NOT recover after TAVILY_API_KEY fix - likely needs recompilation
- **Action**: New issue created (temporary ID: aw_mcp_insp_2026)

### Research Workflow - Failing (P1) - New Issue Created
- **Score**: 20/100
- **Status**: Minimal improvement (1/5 recent runs successful, 20% success rate)
- **Last success**: 2026-01-08 (17 days ago)
- **Latest failure**: §21078189533
- **Impact**: Research and knowledge work capabilities severely limited
- **Root cause**: Did NOT recover after TAVILY_API_KEY fix - likely needs recompilation
- **Action**: New issue created (temporary ID: aw_research_2026)

## Recovered Workflows ✅

### Daily News - RECOVERY ACCELERATING! (P0 → Healthy)
- **Score**: 80/100 (↑5 from 75/100)
- **Status**: **RECOVERY SUSTAINED** - 2/5 recent successes (40% success rate)
- **Latest success**: §21280868153 (2026-01-23)
- **Recent**: 2/5 successful (40% success rate, up from 20% yesterday)
- **Previous issue**: Missing TAVILY_API_KEY secret
- **Resolution**: Secret added on 2026-01-22
- **Monitoring**: ✅ Recovery sustained and accelerating - success rate doubled!

## Healthy Workflows ✅

### Smoke Tests - Perfect Health
All smoke tests: **100% success rate** (5/5 recent runs)
- Smoke Claude: ✅ Perfect
- Smoke Codex: ✅ Perfect
- Smoke Copilot: §21324184559 (2026-01-25) ✅ Perfect
- Score: 100/100

## Systemic Issues

### Issue: Tavily-Dependent Workflows
**Status**: PARTIALLY RESOLVED - 1 recovered, 2 still failing

| Workflow | Status | Last Success | Success Rate | Issue |
|----------|--------|--------------|--------------|-------|
| Daily News | ✅ **RECOVERED** | 2026-01-23 | 40% (↑) | Resolved |
| MCP Inspector | ❌ FAILING | 2026-01-05 | 0% | New issue |
| Research | ❌ FAILING | 2026-01-08 | 20% | New issue |

**Key Finding**: Daily News recovered after TAVILY_API_KEY was added, but MCP Inspector and Research did NOT. Hypothesis: **workflows need recompilation** with `make recompile`.

## Recommendations

### High Priority (P1 - Within 24h)
1. **Recompile MCP Inspector and Research workflows**
   - Command: `make recompile`
   - Hypothesis: Lock files need to pick up new TAVILY_API_KEY secret
   
2. **Test manually** after recompilation

3. **Compare frontmatter** between Daily News (working) and failing workflows

### Medium Priority (P2 - This Week)
1. Monitor Daily News recovery (40% → target 80%)
2. Verify Scout workflow (also uses Tavily)

## Trends

- Overall health score: 91/100 (↑1 from 90/100)
- Daily News recovery accelerating: 20% → 40%
- MCP Inspector worsening: 20/100 → 15/100
- Research stable at low rate: 20%

## Actions Taken This Run

- Updated dashboard issue #11581
- Created new issue for MCP Inspector (aw_mcp_insp_2026)
- Created new issue for Research (aw_research_2026)
- Identified recompilation as likely fix

---
> Last updated: 2026-01-25T03:04:00Z
> Workflow run: §21325874708
> Next check: 2026-01-26T03:04:00Z
