---
id: security-alert-burndown
name: Security Alert Burndown
description: Systematically burns down code security alerts with focus on file write issues
project-url: https://github.com/orgs/githubnext/projects/134
version: v1
state: planned
workflows:
  - code-scanning-fixer
  - security-fix-pr
allowed-repos:
  - githubnext/gh-aw
discovery-repos:
  - githubnext/gh-aw
tracker-label: campaign:security-alert-burndown
memory-paths:
  - memory/campaigns/security-alert-burndown/**
metrics-glob: memory/campaigns/security-alert-burndown/metrics/*.json
cursor-glob: memory/campaigns/security-alert-burndown/cursor.json
objective: Systematically reduce code security alerts to zero critical issues and fewer than 5 high-severity issues
kpis:
  - name: Critical Security Alerts
    baseline: 5
    target: 0
    unit: alerts
    time-window-days: 90
    priority: primary
  - name: High-Severity Alerts
    baseline: 15
    target: 5
    unit: alerts
    time-window-days: 90
    priority: supporting
governance:
  max-new-items-per-run: 3
  max-discovery-items-per-run: 50
  max-discovery-pages-per-run: 3
  max-project-updates-per-run: 10
  max-comments-per-run: 3
  opt-out-labels:
    - no-campaign
    - no-bot
    - skip-security-fix
owners:
  - "@mnkiefer"
risk-level: high
allowed-safe-outputs:
  - create-pull-request
  - autofix-code-scanning-alert
  - add-comment
  - update-project
tags:
  - security
  - automated-fixes
  - code-scanning
workers:
  - id: code-scanning-fixer
    name: Code Scanning Fixer
    description: Automatically fixes high severity code scanning alerts by creating pull requests with remediation
    capabilities:
      - fix-code-scanning-alerts
      - create-pull-requests
      - cache-tracking
    payload-schema:
      repository:
        type: string
        description: Target repository in owner/repo format
        required: true
        example: githubnext/gh-aw
      alert_number:
        type: number
        description: Code scanning alert number to fix
        required: false
        example: 123
      severity:
        type: string
        description: Alert severity filter (high, critical)
        required: false
        example: high
    output-labeling:
      labels:
        - security
        - automated-fix
      key-in-title: true
      key-format: "[code-scanning-fix]"
      metadata-fields:
        - Campaign Id
        - Alert Number
        - Severity
    idempotency-strategy: pr-title-based
    priority: 10
  - id: security-fix-pr
    name: Security Fix PR
    description: Creates autofixes for code security issues via GitHub Code Scanning API
    capabilities:
      - autofix-code-scanning-alerts
      - targeted-alert-fixing
    payload-schema:
      repository:
        type: string
        description: Target repository in owner/repo format
        required: true
        example: githubnext/gh-aw
      security_url:
        type: string
        description: Optional security alert URL for targeted fixing
        required: false
        example: https://github.com/owner/repo/security/code-scanning/123
      alert_number:
        type: number
        description: Code scanning alert number to fix
        required: false
        example: 123
    output-labeling:
      labels:
        - security
        - autofix
      key-in-title: false
      metadata-fields:
        - Campaign Id
        - Alert Number
    idempotency-strategy: branch-based
    priority: 5
bootstrap:
  mode: seeder-worker
  seeder-worker:
    workflow-id: code-scanning-fixer
    payload:
      repository: githubnext/gh-aw
      severity: high
    max-items: 10
---

# Security Alert Burndown Campaign

This campaign systematically burns down code security alerts with the following strategy:

## Focus Areas

- **Prioritizes file write security issues** (highest risk)
- **Clusters related alerts** (up to 3) for efficient remediation
- **Uses Claude for code generation** with detailed security comments
- **All fixes go through PR review process** with automated testing

## Worker Workflows

### code-scanning-fixer

Automatically fixes high severity code scanning alerts by creating pull requests with remediation.

**Schedule:** Every 30 minutes
**Engine:** Copilot
**Capabilities:**
- Lists high severity alerts
- Analyzes vulnerability context
- Generates security fixes with explanatory comments
- Creates PRs with automated fixes
- Caches previously fixed alerts to avoid duplicates

### security-fix-pr

Identifies and automatically fixes code security issues by creating autofixes via GitHub Code Scanning.

**Schedule:** Every 4 hours (can be manually triggered)
**Engine:** Copilot
**Capabilities:**
- Can target specific security alert URLs via workflow_dispatch
- Generates autofixes for code scanning alerts
- Submits fixes directly to GitHub Code Scanning
- Tracks previously fixed alerts in cache-memory

## Campaign Execution

The campaign orchestrator will:

1. **Discover** security issues and PRs created by worker workflows via tracker label
2. **Coordinate** by adding discovered items to the project board
3. **Track Progress** using KPIs and project board status fields
4. **Dispatch** worker workflows to maintain campaign momentum
5. **Report** on progress and remaining alert backlog

## Timeline

### Phase 1 (Weeks 1-2): High severity file write issues
- Focus: File write vulnerabilities, path traversal, insecure file handling
- Goal: Eliminate all critical file write issues
- Approach: Fix 1-3 alerts per workflow run

### Phase 2 (Weeks 3-4): Clustered alert remediation
- Focus: Group related alerts by vulnerability type
- Goal: Reduce high-severity alert count by 50%
- Approach: Fix clusters of up to 3 related alerts

### Phase 3 (Week 5+): Remaining alerts cleanup
- Focus: All remaining high and medium severity alerts
- Goal: Achieve target KPIs (0 critical, <5 high-severity)
- Approach: Systematic processing of remaining backlog

## Risk Management

**Risk Level:** High (requires 2 approvals + sponsor)

**Mitigation:**
- All fixes create PRs for human review
- Automated testing runs on all PRs
- Claude engine provides detailed security comments explaining fixes
- Worker workflows have timeout limits (20 minutes)
- Cache prevents duplicate fixes
- Opt-out labels allow excluding specific repos/issues

## Success Criteria

- All critical security alerts resolved (target: 0)
- High-severity alerts reduced to target threshold (target: <5)
- Zero regression in security posture
- 100% of identified issues tracked in project board
- All fixes include clear security comments and justification
