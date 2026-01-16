---
id: security-alert-burndown
name: "Security Alert Burndown Campaign"
description: "Systematically fixes code security alerts with focus on file write vulnerabilities"
version: v1
state: active

project-url: "https://github.com/orgs/githubnext/projects/999"
tracker-label: "campaign:security-alert-burndown"

objective: "Systematically reduce the code security alerts backlog to zero, prioritizing file write vulnerabilities (path injection, unsafe file operations) and clustering related alerts for efficient fixes"

kpis:
  - name: "Critical/High alerts fixed"
    priority: primary
    unit: count
    baseline: 0
    target: 100
    time-window-days: 90
    direction: increase
    source: code_security
  - name: "File write alerts fixed"
    priority: supporting
    unit: count
    baseline: 0
    target: 50
    time-window-days: 90
    direction: increase
    source: code_security

# Worker workflows that fix alerts
workflows:
  - security-alert-fixer

# Campaign state in repo-memory
memory-paths:
  - memory/campaigns/security-alert-burndown/**
metrics-glob: memory/campaigns/security-alert-burndown/metrics/*.json
cursor-glob: memory/campaigns/security-alert-burndown/cursor.json

# Governance and pacing
governance:
  max-new-items-per-run: 10
  max-discovery-items-per-run: 100
  max-discovery-pages-per-run: 5
  opt-out-labels: [no-campaign, no-bot, security-reviewed]
  max-project-updates-per-run: 15
  max-comments-per-run: 5

owners:
  - "security-team"

tags: [security, code-quality, vulnerability-management]

allowed-safe-outputs:
  - create-issue
  - create-pull-request
  - add-comment
  - update-project

# Use Copilot for orchestration
engine: copilot
---

# Security Alert Burndown Campaign

This campaign systematically addresses code security alerts in the repository, with a focus on file write vulnerabilities and efficient alert clustering.

## Campaign Strategy

### Priority Tiers

1. **Tier 1: File Write Vulnerabilities** (Highest Priority)
   - Path injection (CWE-22)
   - Path traversal (CWE-23)
   - Unsafe file operations (CWE-73, CWE-434)
   - Zip slip vulnerabilities
   - Directory traversal issues

2. **Tier 2: Critical/High Severity Alerts**
   - SQL injection
   - Command injection
   - Cross-site scripting (XSS)
   - Authentication bypasses

3. **Tier 3: Medium Severity Alerts**
   - Information disclosure
   - Denial of service
   - Input validation issues

### Alert Clustering

To maximize efficiency, the campaign clusters related alerts (up to 3 per fix):
- Alerts in the same file
- Alerts with the same rule ID/CWE
- Alerts requiring similar remediation approach

### Code Quality Standards

All fixes must include:
- **Inline comments** explaining the security issue and fix
- **Documentation** of security best practices applied
- **Test considerations** for validating the fix
- **Minimal changes** focused on security remediation

## Worker Workflow

The `security-alert-fixer` workflow:
- Uses **Claude engine** for high-quality code generation with comments
- Discovers open code security alerts via GitHub MCP
- Prioritizes file write vulnerabilities first
- Clusters up to 3 related alerts per PR
- Generates fixes with comprehensive inline documentation
- Creates pull requests with detailed security analysis

## Campaign Lifecycle

### Discovery Phase
- Scan code scanning alerts using GitHub API
- Filter by severity and vulnerability type
- Identify file write issues for priority handling
- Group related alerts for clustering

### Execution Phase
- Execute security-alert-fixer workflow
- Monitor PR creation and review status
- Track fixed alert numbers in repo-memory

### Reporting Phase
- Update project board with progress
- Track KPIs: total fixes, file write fixes, velocity
- Report on remaining backlog and estimated completion

## Success Criteria

- Zero critical/high severity file write vulnerabilities
- 90% reduction in overall security alert backlog
- All fixes include proper documentation and comments
- No security regressions from automated fixes

## Protection

Issues and PRs created by this campaign are labeled with `campaign:security-alert-burndown` to prevent interference from other automation workflows.
