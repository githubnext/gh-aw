---
id: security-alert-burndown
name: Security Alert Burndown Campaign
version: v1
state: active
description: Systematically burns down code security alerts backlog by clustering related issues and creating comprehensive fixes

# Project integration
project-url: https://github.com/orgs/githubnext/projects/TBD
tracker-label: campaign:security-alert-burndown

# Worker workflows that will be discovered and dispatched
workflows:
  - security-alert-burndown-worker

# Campaign memory storage
memory-paths:
  - memory/campaigns/security-alert-burndown/**
metrics-glob: memory/campaigns/security-alert-burndown/metrics/*.json
cursor-glob: memory/campaigns/security-alert-burndown/cursor.json

# Campaign goals and KPIs
objective: Eliminate security alert backlog by systematically fixing vulnerabilities, prioritizing file write issues and clustering related alerts for efficient remediation

kpis:
  - name: File Write Vulnerabilities
    baseline: 0
    target: 0
    unit: alerts
    time-window-days: 90
    priority: primary
  - name: High Severity Alerts
    baseline: 10
    target: 5
    unit: alerts
    time-window-days: 90
    priority: supporting
  - name: Average Alerts Per Fix
    baseline: 1.0
    target: 2.5
    unit: alerts
    time-window-days: 90
    priority: supporting

# Governance policies
governance:
  max-new-items-per-run: 10
  max-discovery-items-per-run: 200
  max-discovery-pages-per-run: 10
  max-project-updates-per-run: 20
  max-comments-per-run: 10
  opt-out-labels:
    - no-campaign
    - no-bot
    - no-automation

# Team and risk
owners:
  - "@githubnext"
executive-sponsors:
  - "@githubnext"
risk-level: high

# Engine configuration
engine: copilot  # Campaign orchestrator uses Copilot
---

# Security Alert Burndown Campaign

## Overview

This campaign systematically burns down the code security alerts backlog by:
1. **Prioritizing file write issues** - Focus on CWE-22 (path traversal), CWE-23 (relative path traversal), CWE-73 (external file control), and CWE-434 (unrestricted upload)
2. **Clustering related alerts** - Group up to 3 related alerts for efficient remediation
3. **Comprehensive fixes with comments** - Generate well-documented code that explains security mitigations
4. **Using Claude for codegen** - Worker workflows use Claude Sonnet 4 for superior code generation and security analysis
5. **Using Copilot for orchestration** - Campaign manager uses Copilot for coordination and project management

## Problem Statement

Security alert backlogs grow over time, becoming overwhelming and difficult to manage. Manual triage and fixes are time-consuming, and fixing alerts one-by-one is inefficient when related issues exist.

## Solution

This campaign uses an intelligent approach to burn down the backlog:

### 1. Priority-Based Processing
- **Phase 1**: File write vulnerabilities (highest risk)
- **Phase 2**: Critical and high severity alerts
- **Phase 3**: Medium severity alerts

### 2. Alert Clustering
The worker workflow intelligently clusters up to 3 related alerts:
- Same file or module
- Same vulnerability type (CWE)
- Common attack vector or remediation approach

This approach:
- Reduces PR count (fewer reviews needed)
- Applies consistent security patterns
- Improves fix quality through holistic analysis
- Accelerates burndown rate

### 3. Well-Commented Code
All generated fixes include:
- Explanation of the security vulnerability
- Rationale for the secure implementation
- Edge cases and assumptions
- Security best practices applied

This ensures:
- Knowledge transfer to the team
- Easier code review
- Long-term maintainability
- Security pattern documentation

### 4. AI Engine Selection
- **Worker workflows**: Claude Sonnet 4 for superior code generation, security analysis, and detailed commenting
- **Campaign orchestrator**: Copilot for coordination, project updates, and tracking

## Worker Workflow

### security-alert-burndown-worker

**Engine**: Claude Sonnet 4  
**Purpose**: Fix security alerts by clustering related issues and creating comprehensive PRs

**Capabilities**:
- Lists open code scanning alerts filtered by type
- Clusters up to 3 related alerts using intelligent grouping
- Analyzes vulnerabilities holistically
- Generates secure code with detailed comments
- Creates single PR addressing all clustered alerts

**Configuration**:
- Default alert type: `file-write` (can be changed to `path-traversal` or `all`)
- Max alerts per cluster: 3
- Timeout: 30 minutes per run
- Safe output: 1 PR per run

**Dispatch**:
```yaml
tracker-id: security-alert-burndown-{run-id}
alert_type: file-write  # or path-traversal, all
max_alerts: 3
```

## Campaign Execution Strategy

### Discovery Phase
The campaign orchestrator discovers security alerts that need fixes:
1. Lists all open code scanning alerts via GitHub API
2. Categorizes by type (file-write, path-traversal, etc.)
3. Prioritizes by severity and age
4. Creates tracking items in project board

### Coordination Phase
The orchestrator dispatches worker workflows:
1. Selects high-priority alerts for next batch
2. Dispatches security-alert-burndown-worker with appropriate filters
3. Tracks PR creation and review status
4. Updates project board with progress

### Tracking Phase
Progress is tracked via:
- GitHub Project board with custom fields
- Metrics snapshots in repo-memory
- KPI tracking (alerts fixed, PRs merged, clustering efficiency)
- Weekly status updates

## Timeline

- **Start Date**: Immediate upon activation
- **Cadence**: Daily runs (can be adjusted)
- **Review**: Weekly progress review
- **Target**: 90-day burndown for high/critical alerts

## Success Metrics

1. **Burndown Rate**: Number of alerts fixed per week
2. **Clustering Efficiency**: Average alerts per PR (target: 2.5)
3. **Fix Quality**: PR merge rate, security regression rate
4. **Time to Fix**: Average time from alert creation to PR merge
5. **Coverage**: Percentage of file-write issues addressed

## Risk Mitigation

**Risk Level**: High (security-sensitive code changes)

**Mitigations**:
- All fixes go through PR review (no auto-merge)
- Comprehensive testing recommendations in PR descriptions
- Detailed code comments for review clarity
- Gradual rollout (file-write issues first)
- Project board tracking for visibility
- Weekly status updates to stakeholders

## Opt-Out

Repositories or alerts can opt out using labels:
- `no-campaign`: Exclude from campaign processing
- `no-bot`: Exclude from automated fixes
- `no-automation`: Exclude from all automation

## Future Enhancements

1. **Expanded clustering**: Support for larger clusters (4-5 alerts) for experienced reviewers
2. **Security tests**: Auto-generate security tests alongside fixes
3. **Fix templates**: Reusable security patterns for common vulnerabilities
4. **Metrics dashboard**: Real-time visualization of campaign progress
5. **Multi-repo support**: Expand beyond single repository
