---
title: Review Readiness Campaign
id: review-readiness-campaign
name: Review Readiness Campaign
version: v1
state: planned
project-url: https://github.com/orgs/githubnext/projects/1
tracker-label: campaign:review-readiness

# Worker workflows that will be discovered and dispatched
workflows:
  - grumpy-reviewer
  - pr-nitpick-reviewer
  - breaking-change-checker
  - code-scanning-fixer

# Campaign memory storage
memory-paths:
  - memory/campaigns/review-readiness/**
metrics-glob: memory/campaigns/review-readiness/metrics/*.json
cursor-glob: memory/campaigns/review-readiness/cursor.json

# Campaign goals and KPIs
objective: Ensure contributions meet quality standards before human review by automating pre-review quality checks
kpis:
  - name: Code Review Quality Issues Detected
    baseline: 0
    target: 100
    unit: issues
    time-window-days: 30
    priority: primary
    direction: increase
  - name: Security Vulnerabilities Fixed
    baseline: 0
    target: 50
    unit: fixes
    time-window-days: 30
    priority: supporting
    direction: increase
  - name: Breaking Changes Detected
    baseline: 0
    target: 10
    unit: changes
    time-window-days: 30
    priority: supporting
    direction: increase

# Governance
governance:
  max-new-items-per-run: 15
  max-discovery-items-per-run: 100
  max-discovery-pages-per-run: 5
  max-project-updates-per-run: 20
  max-comments-per-run: 10
  opt-out-labels:
    - no-campaign
    - no-bot
    - skip-review

# Team
owners:
  - "@githubnext/engineering"
executive-sponsors:
  - "@githubnext/leadership"
risk-level: low
---

# Review Readiness Campaign

This campaign automates pre-review quality checks to ensure contributions are ready for effective human review. By catching common issues early, we reduce review burden and improve overall code quality.

## Campaign Overview

The Review Readiness Campaign orchestrates four specialized workflows to provide comprehensive automated review before human reviewers are engaged:

1. **Critical Code Review**: Deep analysis of edge cases and potential bugs
2. **Style & Convention Checks**: Detailed review of coding standards and best practices
3. **Breaking Change Detection**: Early identification of API and CLI breaking changes
4. **Security Vulnerability Fixes**: Automated remediation of security issues

## Worker Workflows

### grumpy-reviewer
**Trigger**: Slash command `/grumpy` on pull request comments

Performs critical code review with a focus on:
- Edge cases and error handling
- Potential bugs and logic errors
- Code quality issues
- Performance concerns
- Best practice violations

The grumpy reviewer provides thorough, specific feedback with a grumpy but helpful personality.

### pr-nitpick-reviewer
**Trigger**: Slash command `/nit` on pull request comments

Detail-oriented review checking:
- Code style and formatting
- Naming conventions
- Documentation quality
- Test coverage
- Code organization
- Minor improvements and polish

### breaking-change-checker
**Trigger**: Daily schedule

Automated daily analysis to detect:
- CLI command changes (flags, arguments, behavior)
- API contract modifications
- Configuration format changes
- Dependency updates with breaking changes
- Backward compatibility issues

Reports breaking changes with impact assessment and migration guidance.

### code-scanning-fixer
**Trigger**: Every 30 minutes

Automated security remediation:
- Monitors CodeQL and security scanning alerts
- Prioritizes high and critical severity issues
- Creates automated fix PRs when possible
- Adds detailed explanations and test cases
- Links to relevant security advisories

## Campaign Execution

The campaign orchestrator will:

1. **Discover** review feedback, security alerts, and breaking changes created by worker workflows
2. **Coordinate** by tracking items via the tracker-label
3. **Track Progress** using KPIs to measure campaign effectiveness
4. **Dispatch** worker workflows as needed to maintain quality standards
5. **Report** regular status updates on review readiness metrics

## Timeline

- **Start Date**: 2026-01-20
- **Status**: Ongoing
- **Review Cadence**: Weekly progress reports

## Success Criteria

- Consistent automated review coverage on all pull requests
- Reduction in human review iterations due to early issue detection
- Zero critical security vulnerabilities in production
- Breaking changes identified before release
- Improved code quality metrics over time

## Usage

### For Contributors

When your PR is ready for review, invoke the review workflows:

```bash
# Get critical code review
/grumpy

# Get detailed style and convention feedback
/nit
```

### Automated Workflows

The campaign automatically runs:
- **Breaking change checks** daily at midnight UTC
- **Security fixes** every 30 minutes
- Both workflows operate independently and create issues/PRs as needed

## Benefits

- **Reduced Review Burden**: Catch common issues before human review
- **Faster Feedback**: Automated reviews provide immediate feedback
- **Improved Quality**: Consistent application of standards and best practices
- **Security First**: Proactive security vulnerability remediation
- **Breaking Change Awareness**: Early detection prevents production issues
