---
id: security-alert-burndown
name: Security Alert Burndown
version: v1
state: planned
project-url: "https://github.com/orgs/githubnext/projects/TBD"
tracker-label: campaign:security-alert-burndown

# Worker workflows that execute the campaign
workflows:
  - code-scanning-fixer
  - security-fix-pr

# Campaign memory storage
memory-paths:
  - memory/campaigns/security-alert-burndown/**
metrics-glob: memory/campaigns/security-alert-burndown/metrics/*.json
cursor-glob: memory/campaigns/security-alert-burndown/cursor.json

# Campaign goals and KPIs
objective: Reduce critical and high severity code scanning alerts to zero, prioritizing file write vulnerabilities (CWE-73, CWE-22, path traversal, arbitrary file write)
kpis:
  - name: Total Security Alerts
    baseline: 0
    target: 0
    unit: alerts
    time-window-days: 60
    priority: primary
    direction: decrease
  - name: Critical Severity Alerts
    baseline: 0
    target: 0
    unit: alerts
    time-window-days: 60
    priority: supporting
    direction: decrease
  - name: File Write Vulnerability Alerts
    baseline: 0
    target: 0
    unit: alerts
    time-window-days: 60
    priority: supporting
    direction: decrease

# Governance
governance:
  max-new-items-per-run: 3
  max-discovery-items-per-run: 50
  max-discovery-pages-per-run: 5
  max-project-updates-per-run: 10
  max-comments-per-run: 5
  opt-out-labels:
    - no-campaign
    - no-bot
    - wontfix

# Team
owners:
  - "@security-team"
risk-level: medium
---

# Security Alert Burndown Campaign

This campaign orchestrates a systematic burndown of code security alerts, with a focus on file write vulnerabilities and alert clustering for efficient remediation.

## 🎯 Campaign Mission

Systematically reduce the backlog of code security alerts to zero by:
1. **Prioritizing file write vulnerabilities** (CWE-73, CWE-22, path traversal, arbitrary file write)
2. **Clustering related alerts** (up to 3 per PR) for efficient remediation
3. **Generating well-documented fixes** with inline comments explaining security best practices
4. **Maintaining comprehensive audit trail** in campaign memory

## 🔧 Worker Workflows

### code-scanning-fixer
- **Schedule**: Every 30 minutes
- **Purpose**: Quickly addresses urgent high-severity alerts
- **Behavior**: 
  - Processes one alert at a time
  - Creates PR with fix
  - Records fixed alerts in cache memory to prevent duplicates
  - Coordinates with security-fix-pr via shared cache

### security-fix-pr
- **Schedule**: Every 4 hours
- **Purpose**: Batch processing with GitHub autofix API
- **Behavior**:
  - Can fix up to 5 alerts per run
  - Uses GitHub Code Scanning autofix API
  - Checks cache memory to avoid duplicate work
  - Coordinates with code-scanning-fixer via shared cache

## 🎨 Key Features

### Alert Clustering
Group up to 3 related alerts in a single PR when they share:
- **Same file or module**: Co-located vulnerabilities
- **Same vulnerability type**: Similar CWE or rule ID
- **Similar remediation**: Common fix pattern

This reduces PR volume and review overhead while maintaining fix quality.

### Prioritization Strategy
1. **File write issues first**: CWE-73, CWE-22, path traversal, arbitrary file write
2. **Critical severity**: Immediate attention to highest risk
3. **High severity**: Secondary priority
4. **Related alerts**: Cluster when efficient

### Quality Standards
All generated code fixes include:
- **Inline comments**: Explain security fixes and why they're necessary
- **Function/method documentation**: Update docs to reflect security changes
- **Security best practices**: Reference CWE, OWASP, or other standards
- **Testing guidance**: Note what testing should be performed

## 📊 Campaign Execution

The campaign orchestrator will:

1. **Discover** security alerts created by worker workflows via tracker label
2. **Coordinate** alert fixes by managing workflow execution
3. **Track Progress** using KPIs and campaign memory
4. **Prevent Conflicts** via shared cache memory between workflows
5. **Report** status updates to project board

## 🔄 Workflow Coordination

Both worker workflows share cache memory to prevent conflicts:
- **Cache File**: `/tmp/gh-aw/cache-memory/fixed-alerts.jsonl`
- **Format**: JSON Lines with alert number, timestamp, PR number
- **Purpose**: Ensure no duplicate fixes between workflows

Example cache entry:
```jsonl
{"alert_number": 123, "fixed_at": "2024-01-15T10:30:00Z", "pr_number": 456}
```

## 📈 Success Criteria

- ✅ All critical and high severity alerts resolved
- ✅ File write vulnerabilities (CWE-73, CWE-22) fully addressed
- ✅ All fixes include descriptive comments and documentation
- ✅ No duplicate fixes between workflows
- ✅ Comprehensive audit trail in campaign memory
- ✅ PRs clustered efficiently (up to 3 related alerts per PR)

## ⏱️ Timeline

- **Start Date**: TBD (after compilation and approval)
- **Target Completion**: Ongoing until backlog cleared (~60 days estimated)
- **Review Cadence**: Weekly progress review

## 🔍 Campaign Memory Structure

```
memory/campaigns/security-alert-burndown/
├── cursor.json                    # Campaign state and progress
├── baseline.json                  # Initial alert count and breakdown
├── metrics/
│   └── YYYY-MM-DD.json           # Daily metrics snapshots
└── fixes/
    └── alert-{number}.json       # Detailed fix records
```

## 🚀 Getting Started

1. Compile the campaign: `gh aw compile security-alert-burndown`
2. Review the compiled workflow and project board configuration
3. Approve and merge the campaign PR
4. Worker workflows will begin processing alerts automatically
5. Monitor progress via project board and campaign memory

## 📝 Notes

- Both workflows coordinate via shared cache to prevent duplicate fixes
- code-scanning-fixer runs every 30 minutes for rapid response
- security-fix-pr runs every 4 hours for batch processing
- Campaign memory tracks all fixes for compliance and audit purposes
- Alert clustering improves efficiency while maintaining quality
