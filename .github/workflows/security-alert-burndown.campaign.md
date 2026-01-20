---
title: Security Alert Burndown
id: security-alert-burndown
name: Security Alert Burndown Campaign
version: v1
state: active
project-url: https://github.com/orgs/githubnext/projects/1234
tracker-label: "campaign:security-alert-burndown"

# Repositories this campaign can operate on
allowed-repos:
  - "githubnext/gh-aw"

# Worker workflows that will be discovered and dispatched
workflows:
  - code-scanning-fixer    # Creates full PRs for high-severity alerts (runs every 30m)
  - security-fix-pr        # Uses GitHub's autofix API (runs every 4h)

# Campaign memory storage
memory-paths:
  - memory/campaigns/security-alert-burndown/**
metrics-glob: memory/campaigns/security-alert-burndown/metrics/*.json
cursor-glob: memory/campaigns/security-alert-burndown/cursor.json

# Campaign goal and KPIs
objective: Reduce code security file write alert backlog to zero by automatically clustering and fixing alerts
kpis:
  - id: file-write-alerts-fixed
    name: "File Write Alerts Fixed"
    baseline: 0
    target: 100
    unit: alerts
    time-window-days: 90
    priority: primary
    direction: increase
  - id: alert-backlog-size
    name: "Alert Backlog Size"
    baseline: 100
    target: 0
    unit: alerts
    time-window-days: 90
    priority: supporting
    direction: decrease

# Governance
governance:
  max-new-items-per-run: 3        # Cluster up to 3 alerts per run
  max-discovery-items-per-run: 100
  max-discovery-pages-per-run: 5
  max-project-updates-per-run: 15
  max-comments-per-run: 5
  opt-out-labels:
    - no-campaign
    - no-bot

# Team
owners:
  - "@mnkiefer"
risk-level: medium
---

# Security Alert Burndown Campaign

This campaign orchestrates automated fixing of code security alerts, with a focus on file write issues. The campaign uses two complementary workflows to address the security alert backlog:

## Campaign Objectives

1. **Focus on File Write Issues**: Prioritize alerts related to file write operations
2. **Cluster Alerts**: Process up to 3 related alerts per workflow run to improve efficiency
3. **Automated Code Generation**: Use Claude for high-quality code generation and fixes
4. **Campaign Coordination**: Use Copilot for campaign management and orchestration

## Worker Workflows

### code-scanning-fixer (Every 30 minutes)
Automatically fixes high severity code scanning alerts by:
- Listing all open code scanning alerts with high severity
- Selecting an unfixed alert (checking cache to avoid duplicates)
- Analyzing the vulnerability and its context
- Generating a secure fix using Claude
- Creating a pull request with detailed security analysis
- Recording fixed alerts in cache memory

**Engine**: Copilot  
**Frequency**: Every 30 minutes  
**Output**: Full pull requests with code changes and security documentation

### security-fix-pr (Every 4 hours)
Creates autofixes using GitHub's native Code Scanning API by:
- Listing open code scanning alerts (or using a specific alert URL)
- Analyzing the security issue and its context
- Generating a code autofix
- Submitting the fix via GitHub's autofix API
- Tracking fixed alerts in cache memory

**Engine**: Copilot  
**Frequency**: Every 4 hours  
**Output**: GitHub Code Scanning autofixes

## Campaign Execution Flow

The campaign orchestrator will:

1. **Discover** security issues created by worker workflows via the tracker label
2. **Coordinate** by adding discovered items to the GitHub Project board
3. **Track Progress** using KPIs defined above
4. **Dispatch** worker workflows in sequence to maintain momentum
5. **Report** status updates on campaign progress

## Risk Assessment

**Risk Level**: Medium

**Rationale**:
- Automated code changes require review before merging
- Both workflows use cache memory to prevent duplicate fixes
- Changes are submitted as pull requests for human review
- Focus on high-severity issues ensures critical problems are addressed first
- Clustering helps manage workflow execution costs

## Project Board Configuration

The GitHub Project board should have these custom fields configured:

- **status** (single-select): `Todo`, `In Progress`, `Review required`, `Blocked`, `Done`
- **campaign_id** (text)
- **worker_workflow** (text)
- **repository** (text)
- **priority** (single-select): `High`, `Medium`, `Low`
- **size** (single-select): `Small`, `Medium`, `Large`
- **start_date** (date)
- **end_date** (date)

## Timeline

- **Start Date**: Upon campaign activation
- **Review Cadence**: Weekly status updates
- **Target**: Continuous alert reduction until backlog reaches zero

## Success Criteria

- File write alerts reduced to zero
- No duplicate fixes (verified via cache memory)
- All fixes submitted as reviewable pull requests
- 100% of alerts tracked in the project board
- Zero regression in code quality or security posture
