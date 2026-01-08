---
title: "Comparison with GitHub Security Campaigns"
description: "How GitHub's agentic campaigns and security campaigns compare"
---

**GitHub Security Campaigns** (Enterprise Cloud only) focus on remediating security alerts through alert assignment and Copilot Autofix.

**Agentic campaigns** are flexible automation initiatives for any repeatable work, coordinated by AI agents via GitHub Actions workflows.

## Key Differences

| | Security Campaigns | Agentic Campaigns |
|---|---|---|
| **Availability** | Enterprise Cloud only | Any GitHub account with Actions |
| **Use Case** | Security alert remediation | Any repeatable initiative |
| **Configuration** | Web UI | YAML in `.campaign.md` |
| **Automation** | Alert assignment, Copilot Autofix | AI agents, custom workflows |
| **Tracking** | Built-in dashboard | GitHub Projects |
| **API** | REST API | Actions + Projects v2 |

## When to Use Each

**Use Security Campaigns** for security alert remediation on Enterprise Cloud with Copilot Autofix integration.

**Use Agentic Campaigns** for custom automation needs across repositories with version-controlled specs and AI-driven coordination.

## Integration Example

Automate Security Campaign creation using agentic campaigns (requires Enterprise Cloud):

```yaml
---
name: "Security Campaign Automation"
engine: copilot
tools:
  github:
    toolsets: [default, code_security]
on:
  schedule:
    - cron: "0 9 * * 1"
---

# Task
Analyze code scanning alerts from last 7 days.
For CWE categories with 5+ alerts, create a Security Campaign via REST API.
Assign alerts to appropriate teams.
```

## Feature Comparison

| Feature | Security Campaigns | Agentic Campaigns |
|---------|-------------------|-------------------|
| Alert grouping | ✅ Native | ➖ Custom logic |
| Copilot Autofix | ✅ Integrated | ➖ Separate |
| Custom objectives | ➖ Security only | ✅ Any goal |
| Multi-repository | ➖ Single org | ✅ Any scope |
| Version control | ➖ No | ✅ YAML in repo |
| Scheduling | ➖ Manual | ✅ Cron + events |
| Historical metrics | ➖ Live only | ✅ Git branches |

## Learn More

- [Security Campaigns docs](https://docs.github.com/en/enterprise-cloud@latest/code-security/securing-your-organization/fixing-security-alerts-at-scale/about-security-campaigns)
- [Agentic Campaigns overview](/gh-aw/guides/campaigns/)
- [Getting started guide](/gh-aw/guides/campaigns/getting-started/)
