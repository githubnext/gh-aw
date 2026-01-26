---
title: Security Audit Campaign Example
id: security-audit-2026
name: Security Audit 2026
project-url: https://github.com/orgs/example/projects/42

workflows:
  - security-scanner
  - dependency-updater
  - vulnerability-reporter

governance:
  max-new-items-per-run: 10
  max-discovery-items-per-run: 100
  max-discovery-pages-per-run: 5
  max-project-updates-per-run: 15
  max-comments-per-run: 5
  opt-out-labels:
    - no-campaign
    - no-bot

owners:
  - "@security-team"
executive-sponsors:
  - "@cto"
risk-level: high
---

# Security Audit 2026 Campaign

This campaign orchestrates a comprehensive security audit across all repositories, focusing on:

## Objective

Reduce security vulnerabilities to zero critical and less than 5 high-severity issues.

## Key Performance Indicators (KPIs)

### Primary KPI: Critical Vulnerabilities
- **Baseline**: 3 issues
- **Target**: 0 issues
- **Time Window**: 90 days
- **Unit**: issues

### Supporting KPI: High-Severity Vulnerabilities
- **Baseline**: 12 issues
- **Target**: 5 issues
- **Time Window**: 90 days
- **Unit**: issues

## Focus Areas

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

1. **Discover** security issues created by worker workflows via tracker-label
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
