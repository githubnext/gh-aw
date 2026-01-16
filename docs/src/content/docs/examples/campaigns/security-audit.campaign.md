---
title: Security Audit Campaign Example
id: security-audit-2026
name: Security Audit 2026
description: Quarterly security audit campaign focusing on vulnerabilities and dependencies
version: v1
state: planned
project-url: https://github.com/orgs/example/projects/42
tracker-label: campaign:security-audit-2026

# Worker workflows that will be discovered and dispatched
workflows:
  - security-scanner
  - dependency-updater
  - vulnerability-reporter

# Campaign memory storage
memory-paths:
  - memory/campaigns/security-audit-2026/**
metrics-glob: memory/campaigns/security-audit-2026/metrics/*.json
cursor-glob: memory/campaigns/security-audit-2026/cursor.json

# Campaign goals and KPIs
objective: Reduce security vulnerabilities to zero critical and less than 5 high-severity issues
kpis:
  - name: Critical Vulnerabilities
    baseline: 3
    target: 0
    unit: issues
    time_window_days: 90
    priority: primary
  - name: High-Severity Vulnerabilities
    baseline: 12
    target: 5
    unit: issues
    time_window_days: 90
    priority: supporting

# Governance
governance:
  max-new-items-per-run: 10
  max-discovery-items-per-run: 100
  max-discovery-pages-per-run: 5
  max-project-updates-per-run: 15
  max-comments-per-run: 5
  opt-out-labels:
    - no-campaign
    - no-bot

# Team
owners:
  - "@security-team"
executive-sponsors:
  - "@cto"
risk-level: high
---

# Security Audit 2026 Campaign

This campaign orchestrates a comprehensive security audit across all repositories, focusing on:

1. **Vulnerability Scanning**: Identify and track security vulnerabilities
2. **Dependency Updates**: Update outdated dependencies with known vulnerabilities
3. **Compliance Reporting**: Generate security compliance reports for stakeholders

## Worker Workflows

### security-scanner
Scans repositories for security vulnerabilities using multiple tools:
- CodeQL for static analysis
- Dependabot for dependency vulnerabilities
- Container scanning for Docker images

### dependency-updater
Automatically creates PRs to update dependencies with security fixes:
- Prioritizes critical and high-severity updates
- Groups related updates to reduce PR volume
- Adds security justification to PR descriptions

### vulnerability-reporter
Generates weekly security status reports:
- Summary of open vulnerabilities by severity
- Progress toward campaign goals
- Recommendations for security improvements

## Campaign Execution

The campaign orchestrator will:

1. **Discover** security issues created by worker workflows via tracker-id
2. **Coordinate** by adding discovered items to the project board
3. **Track Progress** using KPIs and project board status fields
4. **Dispatch** worker workflows as needed to maintain campaign momentum
5. **Report** weekly status updates to stakeholders

## Timeline

- **Start Date**: 2026-Q1
- **Target Completion**: 2026-03-31 (90 days)
- **Review Cadence**: Weekly status updates

## Success Criteria

- All critical vulnerabilities resolved (current: 3, target: 0)
- High-severity vulnerabilities reduced by 58% (current: 12, target: 5)
- Zero regression in security posture
- 100% of identified issues tracked in project board
