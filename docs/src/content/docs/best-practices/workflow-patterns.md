---
title: Workflow Patterns & Best Practices
description: Proven patterns for workflow triggers, tools, security, and documentation based on production usage and research findings
tableOfContents:
  minHeadingLevel: 2
  maxHeadingLevel: 3
---

This guide captures proven patterns and best practices for creating effective agentic workflows, based on production usage patterns and [research findings](/gh-aw/research/agent-persona-exploration-2026-01/) from the Agent Persona Exploration study.

## Trigger Selection Patterns

Choose the right trigger based on when your workflow should run:

### Pull Request Triggers (50% of production workflows)

Use `pull_request` for code review and validation tasks:

```yaml
on:
  pull_request:
    types: [opened, synchronize, reopened]
```

**Best for**:
- Code review automation (security analysis, migration review)
- Pre-merge validation (accessibility audits, breaking change detection)
- Quality gates that must pass before merging

**Examples from research**:
- Database migration review (BE-1)
- API security analysis (BE-2)
- Accessibility audit (FE-2)

**Key advantages**:
- Provides immediate feedback during development
- Prevents issues from reaching main branch
- Integrates with GitHub's PR review workflow

### Schedule Triggers (33% of production workflows)

Use `schedule` for periodic analysis and reporting:

```yaml
on:
  schedule:
    - cron: '0 9 * * 1-5'  # Weekdays at 9 AM
```

**Best for**:
- Periodic monitoring and analysis
- Regular status reports and digests
- Batch processing of accumulated data

**Examples from research**:
- Flaky test detection (QA-1)
- Feature digest (PM-1)

**Key advantages**:
- Predictable execution schedule
- Reduces noise by batching updates
- Good for trends and patterns over time

### Workflow Run Triggers (17% of production workflows)

Use `workflow_run` to react to other workflow completions:

```yaml
on:
  workflow_run:
    workflows: ["Deploy Production"]
    types: [completed]
```

**Best for**:
- Post-deployment analysis
- Incident response automation
- Dependency chain workflows

**Examples from research**:
- Deployment incident analysis (DO-1)

**Key advantages**:
- Automatically triggered by other workflows
- Access to workflow run context
- Good for failure analysis and monitoring

## Tool Selection Patterns

### GitHub Tools (Universal - 100% of workflows)

All workflows interact with GitHub APIs for repository data:

```yaml
tools:
  github:
    mode: remote
    toolsets: [default]
```

**Common use cases**:
- Reading issues and pull requests
- Creating issues and comments (via safe-outputs)
- Querying repository data
- Analyzing commits and code changes

**Best practices**:
- Use `mode: remote` for GitHub-hosted MCP server (recommended)
- Use `mode: local` only if you need custom MCP server configuration
- Start with `toolsets: [default]` which includes common operations
- Add specific toolsets only when needed (`repos`, `issues`, `pull_requests`)

### Playwright for Browser Automation

Use Playwright when your workflow needs browser automation:

```yaml
tools:
  playwright:
    version: "v1.41.0"
    allowed_domains: ["example.com"]
```

**Best for**:
- Accessibility testing (WCAG compliance)
- Visual regression testing
- Web scraping with JavaScript execution
- End-to-end testing

**Example from research**: Accessibility audit (FE-2)

**Security considerations**:
- Always restrict with `allowed_domains`
- Use specific version for reproducibility
- Run in sandboxed environment (automatic)

### AI Analysis (Built-in)

Leverage the AI engine for pattern recognition and insight extraction:

**Best for**:
- Pattern classification (flaky test patterns)
- Business value extraction (feature analysis)
- Root cause analysis (deployment failures)
- Natural language summarization

**Examples from research**:
- Statistical analysis in flaky test detection (QA-1)
- Business value extraction in feature digest (PM-1)
- Root cause analysis in deployment incidents (DO-1)

**Best practices**:
- Provide clear context and examples
- Structure prompts for specific outputs
- Use memory for accumulated knowledge

## Security Patterns

Based on 100% security compliance in research findings:

### Minimal Permissions (Required)

Always start with minimal required permissions:

```yaml
permissions:
  contents: read
  issues: read
  pull-requests: read
```

**Pattern**: Read-only by default
- Never request write permissions in frontmatter
- Use safe-outputs for write operations instead
- Request only the permissions you actually use

### Safe-Outputs Pattern (100% adoption)

All write operations must use safe-outputs:

```yaml
safe-outputs:
  create-issue:
    title-prefix: "[automated] "
    labels: [automation, bot]
    close-older-issues: true
```

**Key benefits**:
- Automatic input sanitization
- Prevents injection attacks
- Auditable write operations
- Controlled write scope

**Common safe-output types**:
- `create-issue`: Create GitHub issues
- `add-comment`: Add comments to issues/PRs
- `update-labels`: Modify issue/PR labels
- Custom safe-outputs for specific needs

### Network Isolation

Restrict network access to required domains:

```yaml
network:
  allowed:
    - "api.example.com"
    - "docs.example.com"
```

**Best practices**:
- Only allow domains you need
- Use HTTPS endpoints only
- Document why each domain is needed
- Regular review and cleanup

## Documentation Patterns

Based on consistent 5-7 file documentation packages from research:

### Core Documentation Files

Create comprehensive documentation for production workflows:

1. **INDEX.md** (~14 KB): Package overview and navigation
   - Quick overview of the workflow
   - Links to all documentation
   - Quick start for impatient users

2. **README.md** (~10-15 KB): Complete setup guide
   - Detailed installation instructions
   - Configuration options explained
   - Prerequisites and dependencies
   - Troubleshooting section

3. **QUICKREF.md** (~5-6 KB): One-page cheat sheet
   - Common commands and patterns
   - Quick reference for daily use
   - No detailed explanations

4. **EXAMPLE.md** (~11-14 KB): Real-world usage samples
   - Actual workflow output examples
   - Before/after comparisons
   - Common scenarios demonstrated

5. **SETUP.md or CONFIG-TEMPLATE.md** (~10-16 KB): Deployment guide
   - Step-by-step deployment checklist
   - Configuration templates
   - Environment-specific setup

### Quality Indicators

Include these elements for production-ready documentation:

- **Progressive disclosure**: Use `<details>` tags for optional content
- **Visual hierarchy**: Emojis (✅ ❌ 🚀 🔒 💡) for quick scanning
- **Business value**: ROI calculations and time savings
- **Before/after**: Show workflow impact with comparisons
- **Troubleshooting**: Common issues and solutions
- **Best practices**: Pro tips based on real usage

### Lightweight Alternative

For simple workflows, minimal documentation is acceptable:

- **Single README.md**: Combined setup and reference
- **Inline comments**: Explain complex logic in workflow
- **External links**: Reference existing documentation

**When to use lightweight**:
- Simple, single-purpose workflows
- Internal team workflows
- Experimental or prototype workflows

## Workflow Quality Benchmarks

Based on research findings (4.97/5.0 average quality):

### Production-Ready Indicators

A production-ready workflow should have:

- ✅ **Clear purpose**: Single, well-defined responsibility
- ✅ **Minimal permissions**: Read-only with safe-outputs for writes
- ✅ **Appropriate triggers**: Matches use case requirements
- ✅ **Proper tools**: Only what's needed, properly configured
- ✅ **Complete documentation**: At least README with setup and examples
- ✅ **Error handling**: Graceful failures with actionable messages
- ✅ **Business value**: Clear ROI or benefit statement

### Quality Score Framework

Use this framework to evaluate workflow quality:

| Score | Quality Level | Characteristics |
|-------|--------------|-----------------|
| 5.0 | Exceptional | Production-ready, comprehensive docs, best practices throughout |
| 4.5-4.9 | Excellent | Production-ready, good docs, minor improvements possible |
| 4.0-4.4 | Good | Functional, basic docs, some refinement needed |
| 3.5-3.9 | Adequate | Works but needs improvement, limited docs |
| < 3.5 | Needs Work | Significant issues, incomplete, or poor quality |

**Research baseline**: 4.97/5.0 average across diverse scenarios

## Anti-Patterns to Avoid

### Over-Engineering

**Problem**: Creating 5-7 documentation files for simple workflows

**Solution**: Match documentation to workflow complexity
- Simple workflows: Single README
- Medium workflows: README + EXAMPLE
- Complex workflows: Full documentation suite

### Over-Permissions

**Problem**: Requesting broad write permissions

**Solution**: Use read-only + safe-outputs pattern
- Never use `contents: write` unless absolutely necessary
- Prefer safe-outputs for GitHub operations
- Request minimal permissions only

### Trigger Mismatches

**Problem**: Using wrong trigger for the use case

**Solution**: Match trigger to workflow purpose
- Use `pull_request` for pre-merge checks
- Use `schedule` for periodic analysis
- Use `workflow_run` for post-deployment actions

### Undocumented Complexity

**Problem**: Complex workflow with no documentation

**Solution**: Document complexity proportionally
- Explain WHY, not just WHAT
- Provide examples of expected behavior
- Include troubleshooting for common issues

## Pattern Templates

### Code Review Workflow Pattern

```yaml
---
on:
  pull_request:
    types: [opened, synchronize]
permissions:
  contents: read
  pull-requests: read
safe-outputs:
  add-comment:
    prefix: "## 🤖 Automated Review\n\n"
---

# Code Review Workflow

Analyze pull requests for [specific concern] and provide feedback.
```

**Use for**: Security analysis, breaking change detection, quality gates

### Periodic Analysis Pattern

```yaml
---
on:
  schedule:
    - cron: '0 9 * * 1'  # Monday mornings
permissions:
  contents: read
  issues: write
safe-outputs:
  create-issue:
    title-prefix: "[weekly-report] "
    labels: [report, automation]
---

# Weekly Analysis Report

Generate periodic analysis of [metrics/patterns] with trends.
```

**Use for**: Status reports, trend analysis, periodic monitoring

### Incident Response Pattern

```yaml
---
on:
  workflow_run:
    workflows: ["Deploy Production"]
    types: [completed]
permissions:
  actions: read
  contents: read
safe-outputs:
  create-issue:
    title-prefix: "[incident] "
    labels: [incident, urgent]
memory:
  enabled: true
---

# Deployment Incident Analysis

Analyze failed deployments and build incident knowledge base.
```

**Use for**: Post-deployment analysis, failure investigation, MTTR tracking

## Continuous Improvement

### Measuring Success

Track these metrics to improve workflows:

- **Quality score**: Aim for 4.5+ average
- **Security compliance**: 100% minimal permissions + safe-outputs
- **Documentation coverage**: All production workflows documented
- **User adoption**: Team actively using workflows
- **Incident reduction**: Measurable decrease in issues

### Iteration Strategy

1. **Start simple**: Begin with minimal viable workflow
2. **Gather feedback**: Monitor usage and collect user input
3. **Measure impact**: Track time saved, issues prevented, value delivered
4. **Refine documentation**: Update based on common questions
5. **Add features**: Enhance based on actual needs, not assumptions

## Related Resources

- [Agent Persona Exploration Research](/gh-aw/research/agent-persona-exploration-2026-01/) - Full research findings
- [Security Best Practices](/gh-aw/guides/security/) - Comprehensive security guide
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Technical reference
- [Safe Outputs](/gh-aw/reference/safe-outputs/) - Safe-outputs documentation
- [Triggers](/gh-aw/reference/triggers/) - Trigger types and configuration

---

> **Note**: These patterns are based on research across 6 production scenarios achieving an average quality score of 4.97/5.0. They represent proven practices for creating effective, secure, and well-documented agentic workflows.
