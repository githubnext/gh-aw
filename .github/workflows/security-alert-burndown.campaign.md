---
id: security-alert-burndown-2026
name: Security Alert Burndown 2026
description: Systematically fix code security alerts with focus on file write issues, clustering related alerts, and comprehensive documentation
version: v1
state: active

project-url: https://github.com/orgs/githubnext/projects/999
tracker-label: campaign:security-burndown-2026

# Worker workflows to execute
workflows:
  - security-alert-cluster-fixer

# Campaign scope
allowed-repos:
  - githubnext/gh-aw
allowed-orgs:
  - githubnext

# Campaign memory storage
memory-paths:
  - memory/campaigns/security-alert-burndown-2026/**
metrics-glob: memory/campaigns/security-alert-burndown-2026/metrics/*.json
cursor-glob: memory/campaigns/security-alert-burndown-2026/cursor.json

# Campaign goals and KPIs
objective: Systematically reduce code security alerts to zero, focusing on file write vulnerabilities first, with clustering of related alerts and comprehensive inline documentation

kpis:
  - name: File Write Vulnerability Alerts
    baseline: 10
    target: 0
    unit: alerts
    time-window-days: 60
    priority: primary
    direction: decrease
  - name: Total Open Security Alerts
    baseline: 25
    target: 5
    unit: alerts
    time-window-days: 90
    priority: supporting
    direction: decrease
  - name: Average Cluster Size
    baseline: 1
    target: 2.5
    unit: alerts per PR
    time-window-days: 60
    priority: supporting
    direction: increase

# Governance
governance:
  max-new-items-per-run: 5
  max-discovery-items-per-run: 50
  max-discovery-pages-per-run: 3
  max-project-updates-per-run: 10
  max-comments-per-run: 3
  opt-out-labels:
    - no-campaign
    - no-bot
    - wontfix

# Team
owners:
  - "@security-team"
executive-sponsors:
  - "@engineering-leads"
risk-level: medium

# Safe outputs configuration
allowed-safe-outputs:
  - create-issue
  - create-pull-request
  - add-comment
  - update-project
  - create-project-status-update
---

# Security Alert Burndown 2026 Campaign

This campaign orchestrates a systematic approach to reducing the code security alerts backlog across the repository, with a focus on efficiency and comprehensive documentation.

## Strategy

### 1. Focus on File Write Issues First
File write vulnerabilities (path injection, tainted paths, zip slip) are prioritized as they pose significant security risks and are often related, making them good candidates for clustering.

### 2. Cluster Related Alerts (Up to 3)
Instead of fixing alerts one-by-one, the campaign clusters related alerts that share:
- Same file or related files
- Same vulnerability type or pattern
- Similar root cause

This approach:
- **Reduces PR volume**: Fewer PRs to review
- **Provides context**: Related fixes reviewed together
- **Improves efficiency**: Common patterns addressed holistically
- **Enhances understanding**: Comprehensive view of security improvements

### 3. Add Inline Comments to Generated Code
All fixes include detailed inline comments explaining:
- What vulnerability was present
- How the fix addresses it
- What security best practices are applied

This ensures:
- **Maintainability**: Future developers understand security context
- **Knowledge transfer**: Security patterns are documented
- **Compliance**: Clear audit trail of security fixes

### 4. Engine Configuration
- **Claude**: Used for code generation in worker workflows (superior code quality and security reasoning)
- **Copilot**: Used for campaign orchestration (efficient workflow coordination)

## Worker Workflows

### security-alert-cluster-fixer
Identifies and fixes clustered code security alerts using Claude engine.

**Key Features:**
- Filters alerts by type (default: file write issues)
- Clusters up to 3 related alerts per PR
- Generates comprehensive fixes with inline comments
- Creates well-documented pull requests
- Tracks fixed alerts to avoid duplicates

**Configuration:**
- **Engine**: Claude (for superior code generation)
- **Alert Type Filter**: Focuses on file-write, path-injection, tainted-path vulnerabilities
- **Max Cluster Size**: 3 alerts per fix
- **Cache**: Prevents duplicate fixes
- **Safe Outputs**: create-pull-request (1), add-comment (3)

## Campaign Execution

The campaign orchestrator (using Copilot engine) will:

1. **Discover** security-related issues and PRs created by worker workflows via tracker-label
2. **Coordinate** by adding discovered items to the project board
3. **Track Progress** using KPIs:
   - File write vulnerability alerts: 10 → 0 (primary)
   - Total open security alerts: 25 → 5 (supporting)
   - Average cluster size: 1 → 2.5 (efficiency metric)
4. **Execute** worker workflows as needed to maintain campaign momentum
5. **Report** status updates with:
   - Most important findings
   - What was learned
   - KPI trends and velocity
   - Campaign summary and next steps

## Timeline

- **Start Date**: 2026-01
- **Target Completion**: 2026-03-31 (90 days)
- **Review Cadence**: Weekly status updates
- **Execution Frequency**: Worker runs every 4 hours (automatic via schedule)

## Success Criteria

- All file write vulnerability alerts resolved (current: 10, target: 0)
- Total security alerts reduced by 80% (current: 25, target: 5)
- Average cluster size increased from 1 to 2.5+ alerts per PR
- 100% of fixes include inline comments explaining security context
- Zero regression in security posture
- All identified issues tracked in project board

## Benefits

### Efficiency
- **Clustering**: Reduces number of PRs and review overhead
- **Automation**: Worker runs on schedule without manual intervention
- **Caching**: Prevents duplicate work

### Quality
- **Claude Engine**: Superior code generation for security fixes
- **Inline Comments**: Comprehensive documentation for maintainability
- **Comprehensive Fixes**: Addresses root causes, not just symptoms

### Visibility
- **Project Board**: Real-time tracking of campaign progress
- **KPIs**: Clear metrics for success
- **Status Updates**: Weekly reports to stakeholders

### Knowledge Transfer
- **Commented Code**: Security patterns documented inline
- **PR Descriptions**: Detailed explanations of vulnerabilities and fixes
- **Audit Trail**: Complete history of security improvements

## Risk Mitigation

- **Medium Risk Level**: Automated fixes require human review before merge
- **Copilot Reviewer**: PRs automatically assigned for review
- **Cache Memory**: Prevents duplicate fixes
- **Opt-out Labels**: Respects no-campaign, no-bot, wontfix labels
- **Governance Limits**: Bounded writes per run prevent runaway operations

## Project Board Structure

The GitHub Project board will track:
- **Epic Issue**: Overall campaign progress
- **Worker Issues**: Issues created by security scanning
- **Fix PRs**: Pull requests with clustered fixes
- **Status Field**: Todo / In Progress / Review Required / Blocked / Done
- **Priority Field**: Critical / High / Medium / Low
- **Campaign Fields**: campaign_id, worker_workflow, repository

## Next Steps

1. Campaign orchestrator discovers existing security alerts
2. Orchestrator executes security-alert-cluster-fixer workflow
3. Worker clusters related file write alerts (up to 3)
4. Worker generates fixes with Claude and creates PR with inline comments
5. PR is reviewed and merged
6. Orchestrator updates project board and creates status update
7. Process repeats until all alerts are resolved
