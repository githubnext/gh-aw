# Shared Alerts - Workflow Health Manager
**Last Updated**: 2026-01-04T02:59:53Z

## Critical Findings Requiring Cross-Orchestrator Attention

### 1. Metrics Collection Gap (P0)
**Status**: No execution metrics available
**Impact**: All meta-orchestrators affected
- Workflow Health Manager: Cannot analyze failure patterns
- Campaign Manager: Cannot track campaign workflow performance
- Agent Performance Analyzer: Cannot correlate agent quality with workflow success

**Action**: Metrics Collector workflow needs recompilation and verification

### 2. Outdated Workflows (P0)
**Count**: 10 workflows (7.8% of total)
**Impact**: Runtime behavior may not match source code
**Critical Item**: `metrics-collector.md` itself is outdated

**Coordination Note**: Campaign Manager should check if any campaign workflows are in the outdated list.

## Workflow Health Summary

- **Total Workflows**: 128 executable workflows
- **Compilation Coverage**: 100% (all have lock files)
- **Outdated**: 10 workflows need recompilation
- **Engine Distribution**: 69 Copilot, 25 Claude, 7 Codex
- **Scheduled Workflows**: 19 (17 daily, 1 weekly, 1 hourly)

## Systemic Patterns

### Positive
- Strong categorization and naming conventions
- Diverse engine usage (no single point of failure)
- Extensive GitHub MCP adoption (94 workflows)

### Concerns
- No metrics data available yet for health monitoring
- Safe outputs declaration appears missing in frontmatter (needs verification)

## Recommendations for Other Orchestrators

### Campaign Manager
- Check if campaign workflows are in outdated list
- Consider campaign performance analysis once metrics available
- Monitor campaign orchestrator workflows for health issues

### Agent Performance Analyzer
- Coordinate on metrics schema once metrics collector is fixed
- Consider workflow execution context when analyzing agent quality
- Track which agents are used by which workflows

## Next Coordination Point

After metrics collection is established (estimated 7 days), all meta-orchestrators should re-run with full execution data for comprehensive cross-system analysis.

