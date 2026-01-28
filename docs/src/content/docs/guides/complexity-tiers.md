---
title: Workflow Complexity Tiers
description: Understanding complexity tiers for GitHub Agentic Workflows and how agents calibrate responses to match your needs
sidebar:
  order: 250
---

GitHub Agentic Workflows use a three-tier complexity system to calibrate documentation depth and feature sophistication based on your request. This ensures you get focused, efficient responses for simple workflows while receiving comprehensive guidance for complex scenarios.

## Why Complexity Tiers Matter

Without complexity calibration, agents might:
- **Over-engineer** simple workflows with unnecessary features
- **Overwhelm** beginners with excessive detail
- **Under-deliver** on complex workflows that need comprehensive guidance

The tier system solves this by matching response sophistication to request complexity.

## The Three Tiers

### Basic Tier: Simple, Focused Workflows

**When to use:**
- Getting started with GitHub Agentic Workflows
- Creating straightforward automation
- Single-purpose workflows
- Quick prototypes

**Characteristics:**
- Single standard trigger (push, pull_request, issues)
- Minimal or no tool usage
- Linear workflow logic
- Standard output formats

**What you get:**
- **Concise documentation** (1-2 paragraphs)
- **Single working example** 
- **Minimal comments** (only for non-obvious logic)
- **No feature suggestions** (unless requested)
- **Fast iteration** with focused responses

**Example Requests:**
```
"Run tests on pull requests"
"Label issues based on title keywords"  
"Post a comment when a PR is opened"
"Run linter on push to main branch"
```

**Example Response:**

```yaml
---
name: Test Runner
on:
  pull_request:
permissions:
  contents: read
engine: copilot
---

# Run Tests on Pull Requests

Run the test suite whenever a pull request is opened or updated.

Your task: Run `npm test` and report results.
```

Short, focused, gets the job done.

### Intermediate Tier: Standard Workflows

**When to use:**
- Building production workflows
- Integrating multiple GitHub features
- Conditional logic and branching
- Tool combinations

**Characteristics:**
- Multiple triggers or conditions
- Tool combinations (bash + GitHub API, jq)
- Moderate configuration needs
- Safe-outputs or safe-inputs usage

**What you get:**
- **Moderate documentation** (3-5 paragraphs)
- **2-3 examples** showing variations
- **Explanatory comments** for logic and integrations
- **Related feature mentions**
- **Balanced code-to-text ratio**

**Example Requests:**
```
"Triage issues using labels and assign to team members based on expertise"
"Run different test suites based on changed files"
"Create PRs with dynamic content from external APIs"
"Scheduled workflow that analyzes repository health metrics"
```

**Example Response:**

```yaml
---
name: Issue Triage Bot
on:
  issues:
    types: [opened, edited]
  schedule: every 6h
permissions:
  contents: read
  issues: write
engine: copilot
tools:
  github:
    toolsets: [issues]
safe-outputs:
  add-labels:
    max: 5
---

# Intelligent Issue Triage

Automatically categorize and route issues based on content analysis.

## Your Task

1. **Analyze issue content**: Extract key topics and technical areas
2. **Apply labels**: Add appropriate labels (bug, feature, documentation, etc.)
3. **Route to team**: Suggest assignees based on expertise areas
4. **Track patterns**: Note recurring themes for later analysis

## Guidelines

- Be conservative with labels (2-4 per issue maximum)
- Look for keywords: "error", "crash", "feature request", "docs"
- Consider issue history and similar past issues
- Suggest team members only when expertise is clear

Use safe-outputs to apply labels. Keep reasoning brief but clear.
```

More depth, multiple concerns, still practical.

### Advanced Tier: Complex Workflows

**When to use:**
- Enterprise-grade automation
- Multi-repo orchestration
- State management and persistence
- Performance-critical workflows
- Security-sensitive operations

**Characteristics:**
- Complex multi-trigger workflows
- Custom toolchains and MCP servers
- Multi-stage processes
- Repository memory or campaigns
- Performance optimization needs
- Strict mode and security considerations

**What you get:**
- **Comprehensive documentation** (6+ paragraphs)
- **Multiple detailed examples** with edge cases
- **Extensive architectural comments**
- **Proactive optimization suggestions**
- **Security best practices**
- **Performance tuning guidance**
- **Troubleshooting advice**

**Example Requests:**
```
"Multi-repo campaign for security vulnerability detection and remediation"
"Hierarchical agent system for project management with task delegation"
"Performance-optimized workflow with caching and parallel execution"
"Complex state machine with repo-memory persistence"
```

**Example Response:**

```yaml
---
name: Security Scanner Campaign
description: Multi-repository security vulnerability detection and automated remediation
engine: copilot
strict: true
on:
  schedule: every 24h
  workflow_dispatch:
permissions:
  contents: read
  security-events: write
tools:
  github:
    toolsets: [repos, security]
  bash:
    - "jq *"
    - "npm *"
network:
  allowed:
    - defaults
    - github
repo-memory:
  - id: security-findings
    create-orphan: true
    max-files: 1000
    max-file-size: 10MB
timeout-minutes: 60
---

# Enterprise Security Scanner Campaign

This workflow implements a multi-repository security scanning campaign with:

- **Automated vulnerability detection** across all repositories
- **Priority-based remediation** with intelligent routing
- **Persistent state management** using repo-memory
- **Performance optimization** through caching and parallelization
- **Compliance reporting** with detailed audit trails

## Architecture Overview

The campaign operates in three stages:

1. **Discovery Phase**: Scan all repositories for security issues
2. **Analysis Phase**: Prioritize findings and identify auto-fix candidates  
3. **Remediation Phase**: Create PRs for fixable issues

State is preserved in the `memory/security-findings` branch between runs.

## Your Task

### Stage 1: Repository Discovery

[Detailed instructions for discovery logic]

### Stage 2: Vulnerability Analysis  

[Detailed instructions for analysis logic]

### Stage 3: Automated Remediation

[Detailed instructions for remediation logic]

## Performance Considerations

- Use GitHub's code scanning API for efficiency
- Cache npm package vulnerability database
- Process repositories in parallel (max 5 concurrent)
- Skip archived and disabled repositories

## Security Best Practices

- Run in strict mode to prevent accidental privilege escalation
- Validate all external inputs before processing
- Never log secrets or sensitive data
- Use minimal permissions (read-only where possible)

## Error Handling

- Retry failed scans up to 3 times with exponential backoff
- Log errors to repo-memory for manual review
- Continue processing other repositories on individual failures
- Alert on-call team if critical vulnerabilities detected

## Monitoring and Alerts

Track these metrics in repo-memory:
- Repositories scanned per run
- Vulnerabilities detected and remediated
- False positive rate
- Average remediation time

See the [Campaign Lifecycle Guide](/gh-aw/guides/campaigns/lifecycle/) for deployment and monitoring details.
```

Comprehensive, production-ready, covers all aspects.

## How Complexity Detection Works

Agents automatically analyze your request and assign a complexity score based on these indicators:

### Scoring System

| Indicator Type | Examples | Points |
|----------------|----------|--------|
| **Basic** | Single trigger, simple verbs ("run", "check"), no tools | 1 each |
| **Intermediate** | Multiple triggers, conditionals ("if", "when"), tool mentions, safe-outputs | 2 each |
| **Advanced** | Multi-stage keywords, state management, performance needs, custom MCP servers, campaigns | 3 each |

**Tier Assignment:**
- **Basic**: Total score 1-3
- **Intermediate**: Total score 4-7  
- **Advanced**: Total score 8+

### Examples

**"Run tests on pull requests"**
- Single trigger (pull_request): +1
- Simple verb (run): +1
- **Total: 2 → Basic Tier**

**"Triage issues using labels and assign to team members"**
- Single trigger (issues): +1
- Tool integration implied (GitHub API): +2
- Conditional logic (based on content): +2
- **Total: 5 → Intermediate Tier**

**"Multi-repo security scanning campaign with state persistence"**
- Multi-stage workflow: +3
- State management (repo-memory): +3
- Campaign keyword: +3
- **Total: 9 → Advanced Tier**

## Requesting a Specific Tier

You can override automatic detection by including tier hints in your request:

### Force Basic Tier

Add phrases like:
- "Keep it simple"
- "Minimal example"
- "Just the basics"
- "Quick and easy"

**Example:** "Run tests on pull requests (keep it simple)"

### Force Advanced Tier

Add phrases like:
- "Comprehensive guide"
- "Production-ready"
- "Enterprise-grade"
- "With all options"
- "Full documentation"

**Example:** "Run tests on pull requests (comprehensive, production-ready)"

## Quality Across All Tiers

**All complexity tiers maintain high standards:**

✅ **Correct, working code**  
✅ **Proper error handling**  
✅ **Security best practices**  
✅ **Clear documentation**  
✅ **Professional quality**

**The difference is depth, not quality.**

- **Basic tier**: Streamlined for efficiency
- **Intermediate tier**: Balanced detail and practicality  
- **Advanced tier**: Comprehensive coverage and guidance

## Tips for Best Results

1. **Be clear about your needs**: Include relevant details in your request
2. **Start simple**: Begin with basic tier, then request more detail if needed
3. **Use tier hints**: Add "keep it simple" or "comprehensive guide" to guide complexity
4. **Consider your audience**: Basic for beginners, advanced for production systems
5. **Iterate**: You can always ask for more detail or simplification

## Examples by Tier

### Basic Tier Examples

```
"Run linter on every push"
"Comment on new issues with welcome message"  
"Tag releases automatically"
"Run security scan on PRs"
```

### Intermediate Tier Examples

```
"Auto-triage issues based on labels and content"
"Run different CI jobs based on changed files"
"Create weekly summary of repository activity"
"Sync issues between repositories"
```

### Advanced Tier Examples

```
"Multi-repo vulnerability scanning with automated fixes"
"Hierarchical project management agent system"
"Performance-optimized test suite with intelligent caching"
"State machine for complex approval workflows with persistence"
```

## Related Resources

- [Getting Started Guide](/gh-aw/introduction/how-they-work/) - Learn the basics
- [Reference: Frontmatter](/gh-aw/reference/frontmatter/) - Complete configuration options
- [Reference: Tools](/gh-aw/reference/tools/) - Available tools and MCP servers
- [Developer Guide: Complexity Calibration](https://github.com/githubnext/gh-aw/blob/main/specs/complexity-calibration.md) - Technical specification

---

**Have feedback on complexity tiers?** [Open an issue](https://github.com/githubnext/gh-aw/issues/new) to help us improve the system.
