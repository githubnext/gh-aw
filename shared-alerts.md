# Shared Alerts - Workflow Health Manager
**Last Updated**: 2026-01-14T02:56:33Z

## 🚨 CRITICAL ALERT: Meta-Orchestrator Cascade Failure

**Status**: System health monitoring degraded  
**Severity**: P0 - Immediate attention required

### Affected Systems
1. **Metrics Collector**: 50% success rate - MCP Gateway schema error
2. **Agent Performance Analyzer**: 20% success rate - unknown failures
3. **CI Doctor**: 0% success rate - completely broken

### Impact
- No workflow metrics collected since 2026-01-09
- No agent performance data for 5 days
- No CI failure diagnostics available
- Loss of system visibility and monitoring

### Root Causes Identified

#### 1. MCP Gateway Schema Breaking Change (P0)
**Workflow**: Metrics Collector  
**Error**: Configuration validation error with MCP Gateway v0.0.47
```
Error: doesn't validate with mcp-gateway-config.schema.json
  Missing properties: 'container'
  additionalProperties 'command' not allowed
```
**Fix Required**: Update MCP server config to new schema format

#### 2. Timeout Pattern (P1)
**Workflows**: CI Doctor, Daily News  
**Error**: Exit code 7 after 120s timeout  
**Pattern**: Started around 2026-01-09  
**Fix Required**: Investigation needed for root cause

## Critical Issues Summary

| Workflow | Status | Success Rate | Priority | Last Success |
|----------|--------|--------------|----------|--------------|
| CI Doctor | 🚨 Broken | 0% | P0 | Pre-Dec 21 |
| Daily News | 🚨 Degraded | 50% | P1 | Intermittent |
| Metrics Collector | 🚨 Degraded | 50% | P1 | 2026-01-08 |
| Agent Performance Analyzer | ⚠️ Degraded | 20% | P2 | 2026-01-08 |
| Daily Repo Chronicle | ⚠️ Intermittent | 30% | P2 | 2026-01-13 |

## Systemic Issues

### Issue 1: MCP Gateway Breaking Change
- **Impact**: All workflows using MCP Gateway may fail
- **Affected**: Currently 1 confirmed (Metrics Collector), potentially more
- **Action**: Audit all workflows for MCP Gateway usage
- **Timeline**: Schema change occurred around 2026-01-09

### Issue 2: Timeout Epidemic
- **Impact**: Multiple workflows timing out after 120s
- **Pattern**: Started 2026-01-09
- **Hypothesis**: Network latency, API rate limiting, or resource contention
- **Action**: System-wide timeout investigation needed

### Issue 3: Meta-Orchestrator Dependencies
- **Risk**: Cascade failure affecting monitoring infrastructure
- **Current State**: 2 of 3 meta-orchestrators degraded
- **Action**: Prioritize Metrics Collector fix to restore visibility

## Recommendations for Other Orchestrators

### Campaign Manager
- **Alert**: Monitor campaigns for timeout issues
- **Action**: Check if any campaigns use MCP Gateway
- **Risk**: Campaign workflows may be affected by same issues

### Agent Performance Analyzer
- **Alert**: Self-failing (20% success rate)
- **Action**: Needs immediate investigation
- **Impact**: No quality metrics for agents

### All Workflows
- **Alert**: Check for MCP Gateway usage and update configs
- **Alert**: Monitor for timeout patterns (exit code 7)
- **Alert**: Consider increasing timeout limits if needed

## Coordination Notes

### Issues Created
1. CI Doctor completely broken (P0)
2. Metrics Collector MCP Gateway config error (P1)
3. Daily News timeout failures (P1)

### Actions Required
1. **Immediate**: Fix Metrics Collector MCP config
2. **High**: Investigate CI Doctor timeout
3. **High**: Fix Daily News timeout
4. **Medium**: Investigate Agent Performance Analyzer
5. **Medium**: Stabilize Daily Repo Chronicle

### Monitoring Plan
- Continue daily health checks
- Track resolution progress
- Alert on new failures with similar patterns
- Re-evaluate health score after fixes

---
**Analysis Coverage**: 123/123 workflows (100%)  
**Critical Issues**: 3  
**Systemic Issues**: 3  
**Next Analysis**: 2026-01-15T03:00:00Z
