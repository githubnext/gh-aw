# Workflow Health Dashboard - 2026-01-14

## Executive Summary

- **Total Workflows**: 123 executable workflows
- **Compilation Status**: 123 lock files (100% coverage) ✅
- **Critical Failing Workflows**: 3 workflows (P0/P1)
- **Warning Status**: 2 workflows with intermittent failures (P2)
- **Overall Health Score**: 75/100 ⚠️ (Needs Attention)

## Critical Issues 🚨

### 1. CI Doctor - **COMPLETELY BROKEN** (P0)
- **Status**: 0% success rate (last 20 runs all failed)
- **Last Success**: Before 2025-12-21
- **Error Pattern**: Exit code 7 (timeout after 120s)
- **Impact**: Unable to diagnose CI failures
- **Issue**: Needs immediate investigation
- **Run**: https://github.com/githubnext/gh-aw/actions/runs/20685353923

### 2. Daily News - **HIGH FAILURE RATE** (P1)
- **Status**: 50% success rate (10/20 successful)
- **Error**: Exit code 7 - timeout after 120 seconds
- **Pattern**: Consistent timeout failures starting 2026-01-09
- **Last Failure**: Run #101 (2026-01-13)
- **Impact**: Users not receiving daily repository news
- **Logs**: "Retrying up to 120 times with 1s delay (120s total timeout)"
- **Run**: https://github.com/githubnext/gh-aw/actions/runs/20951135921

### 3. Metrics Collector - **RECENT REGRESSION** (P1)
- **Status**: 50% success rate (5/10 successful)
- **Error**: MCP Gateway configuration validation error
- **Pattern**: Started failing 2026-01-09, was 100% before
- **Impact**: No workflow metrics being collected since Jan 9
- **Root Cause**: MCP Gateway config schema mismatch
  - Missing 'container' property for stdio server config
  - Invalid 'command' property in config
  - Schema validation failing with v0.0.47
- **Last Failure**: Run #24 (2026-01-13)
- **Run**: https://github.com/githubnext/gh-aw/actions/runs/20960733354

## Warnings ⚠️

### 4. Agent Performance Analyzer - **DEGRADED** (P2)
- **Status**: 20% success rate (2/10 successful)
- **Pattern**: Consistent failures from 2026-01-09 to 2026-01-13
- **Last Success**: 2026-01-08
- **Impact**: No agent quality metrics for 5 days
- **Action Required**: Investigation needed

### 5. Daily Repo Chronicle - **INTERMITTENT** (P2)
- **Status**: 30% success rate (3/10 successful)
- **Pattern**: Intermittent failures
- **Last Success**: 2026-01-13
- **Impact**: Inconsistent repository chronicle updates
- **Action Required**: Monitor and investigate root cause

## Healthy Workflows ✅

- **Workflow Health Manager**: Running (this workflow)
- **Daily Repo Chronicle**: Mostly working (3 recent successes)
- **117+ other workflows**: Compilation status healthy

## Systemic Issues

### Issue 1: Timeout Pattern Across Multiple Workflows (P1)
- **Affected workflows**: CI Doctor, Daily News
- **Pattern**: Exit code 7 after 120s timeout
- **Root Cause**: Unknown - needs investigation
- **Recommendation**: 
  1. Review timeout configuration
  2. Check for network/API latency issues
  3. Consider increasing timeout or optimizing operations

### Issue 2: MCP Gateway Schema Breaking Change (P0)
- **Affected workflows**: Metrics Collector
- **Pattern**: Configuration validation failures with MCP Gateway v0.0.47
- **Root Cause**: Schema changed requiring 'container' property
- **Recommendation**:
  1. Update MCP server configurations to new schema
  2. Replace `command`/`args` with `container` property
  3. Review all workflows using MCP Gateway

### Issue 3: Meta-Orchestrator Cascade Failure (P1)
- **Agent Performance Analyzer**: Failing (20% success)
- **Metrics Collector**: Failing (50% success)
- **Workflow Health Manager**: Running (analyzing now)
- **Impact**: Loss of visibility into system health
- **Recommendation**: Prioritize fixing Metrics Collector to restore metrics

## Detailed Analysis

### Compilation Status
**Status**: Perfect ✅
- 123 `.md` workflow files
- 123 `.lock.yml` compiled workflows
- 0 missing lock files
- 0 outdated lock files

### Workflow Run Analysis (Last 7 Days)
| Workflow | Success Rate | Status | Priority |
|----------|--------------|--------|----------|
| CI Doctor | 0% (0/20) | 🚨 Critical | P0 |
| Agent Performance Analyzer | 20% (2/10) | 🚨 Critical | P2 |
| Daily Repo Chronicle | 30% (3/10) | ⚠️ Warning | P2 |
| Daily News | 50% (10/20) | 🚨 Critical | P1 |
| Metrics Collector | 50% (5/10) | 🚨 Critical | P1 |
| Workflow Health Manager | Running | ✅ Healthy | - |

### Error Pattern Summary
1. **Timeout errors (exit code 7)**: 2 workflows
2. **MCP Gateway schema errors**: 1 workflow
3. **Unknown failures**: 2 workflows

## Recommendations

### Immediate Actions (P0)
1. **Fix CI Doctor** - completely non-functional
   - Investigate exit code 7 timeout
   - Review workflow configuration
   - Consider temporary disable until fixed

### High Priority (P1)
1. **Fix Metrics Collector MCP Gateway config**
   - Update to new schema format
   - Add 'container' property
   - Remove invalid 'command' property
   - Test with MCP Gateway v0.0.47

2. **Fix Daily News timeout issues**
   - Investigate cause of 120s timeout
   - Optimize operations
   - Consider increasing timeout limit

3. **Restore Meta-Orchestrator Health**
   - Fix Metrics Collector (blocks other monitoring)
   - Fix Agent Performance Analyzer
   - Restore system visibility

### Medium Priority (P2)
1. **Investigate Agent Performance Analyzer failures**
2. **Stabilize Daily Repo Chronicle**
3. **Add alerting for cascade failures**

### Long-Term Improvements
1. Implement circuit breakers for failing workflows
2. Add health check endpoints
3. Create automated remediation for common failures
4. Improve timeout handling and retry logic

## Trends

- **Overall health**: 75/100 (↓ from 95/100 on 2026-01-08)
- **New failures this week**: 3 critical
- **Systemic issues identified**: 3
- **Workflows needing attention**: 5
- **Health degradation**: -20 points since last run

## Actions Taken This Run

- Analyzed 123 workflows for compilation status
- Queried recent run data for 5 workflows
- Downloaded and analyzed failure logs for 2 critical workflows
- Identified 3 systemic issues affecting multiple workflows
- Documented 5 failing/degraded workflows with error details

## Next Steps

1. Create GitHub issues for P0/P1 workflows
2. Alert team about meta-orchestrator cascade failure
3. Monitor for additional failures with similar patterns
4. Track resolution progress

---
**Last Updated**: 2026-01-14T02:56:33Z  
**Next Check**: 2026-01-15 (scheduled daily)  
**Workflows Analyzed**: 123/123 (100%)  
**Critical Issues**: 3  
**Warnings**: 2
