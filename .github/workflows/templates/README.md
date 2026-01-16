# Workflow Template Library

A curated collection of 7 reusable workflow templates based on high-performing automation patterns identified in production use.

## Purpose

These templates provide starting points for common automation patterns, reducing time-to-first-workflow from 25 minutes to under 5 minutes. Each template includes:

- **Working placeholder workflow** - Ready to customize and deploy
- **Configuration checklist** - Step-by-step setup guide
- **Common variations** - Alternative implementations
- **Usage examples** - Real-world applications
- **Links to source scenarios** - Based on proven patterns from research

## Available Templates

### 1. PR Code Review Automation

**File**: [`pr-code-review.md`](./pr-code-review.md)

**Use When**: You want automated code review on pull requests

**What It Does**: Analyzes code changes, identifies issues across multiple categories (code quality, best practices, security, testing), and posts detailed review comments.

**Key Features**:
- Line-specific review comments
- Multiple review categories (quality, security, testing, docs)
- Cache memory for learning patterns
- Configurable review depth

**Trigger Options**:
- Pull request events (opened, synchronize, reopened)
- Slash command for on-demand reviews

**Based On**: BE-1, FE-2, FE-3, QA-1 scenarios

---

### 2. Scheduled Monitoring with Alerting

**File**: [`scheduled-monitoring.md`](./scheduled-monitoring.md)

**Use When**: You need to monitor APIs, data sources, or system health on a schedule

**What It Does**: Periodically checks configured targets, detects issues or anomalies, and creates GitHub issues for alerting.

**Key Features**:
- Historical baseline comparison
- Configurable alert thresholds
- Multiple severity levels (critical, warning, info)
- Historical trend tracking with repo-memory

**Trigger Options**:
- Scheduled (hourly, daily, weekly, monthly)
- Manual workflow dispatch

**Based On**: BE-2, DO-1, DO-2, QA-2 scenarios

---

### 3. Visual Testing Automation

**File**: [`visual-testing.md`](./visual-testing.md)

**Use When**: You need visual regression testing for UI components or applications

**What It Does**: Runs Playwright tests, captures screenshots across multiple devices, compares against baselines, and provides visual feedback on PRs.

**Key Features**:
- Multi-device testing (mobile, tablet, desktop)
- Screenshot comparison with diff visualization
- Accessibility checks
- Artifact uploads for review

**Trigger Options**:
- Pull request events
- Scheduled for nightly regression tests

**Based On**: FE-1 scenario (5.0 rating - Visual regression testing)

---

### 4. On-Demand Report Generation

**File**: [`on-demand-report.md`](./on-demand-report.md)

**Use When**: You need to aggregate data and generate formatted reports

**What It Does**: Collects data from GitHub APIs or external sources, analyzes trends, formats into comprehensive reports, and publishes to GitHub Discussions.

**Key Features**:
- Multiple report types (daily, weekly, monthly, custom)
- Data visualization with charts
- Flexible data sources
- Published to Discussions for team visibility

**Trigger Options**:
- Manual workflow dispatch with inputs
- Can be adapted for scheduled reports

**Based On**: PM-1, PM-2 scenarios

---

### 5. Multi-Phase Analysis Pipelines

**File**: [`multi-phase-analysis.md`](./multi-phase-analysis.md)

**Use When**: You need comprehensive analysis with multiple stages (collection → analysis → tracking → reporting)

**What It Does**: Executes a multi-phase pipeline: data collection → analysis → historical comparison → trend detection → reporting → alerting.

**Key Features**:
- 6-phase structured pipeline
- Historical data tracking with repo-memory
- Trend detection with configurable thresholds
- Automated alerting on significant changes
- Comprehensive reporting with visualizations

**Trigger Options**:
- Scheduled (daily or weekly recommended)
- Manual workflow dispatch

**Based On**: QA-2 scenario (5.0 rating - Flaky test tracking)

---

### 6. Rate-Limited Automation

**File**: [`rate-limited-automation.md`](./rate-limited-automation.md)

**Use When**: You need automation that doesn't overwhelm the team with too many items at once

**What It Does**: Detects items requiring action, prioritizes them intelligently, creates a limited number of issues/PRs per run, and tracks processed items to avoid duplicates.

**Key Features**:
- Intelligent prioritization (severity, impact, recency)
- Configurable rate limits (max items per run)
- Deduplication using repo-memory
- Processes highest-priority items first

**Trigger Options**:
- Scheduled (daily recommended)
- Manual workflow dispatch

**Based On**: DO-2 scenario, Security scanner (5.0 rating)

---

### 7. API Integration with Persistence

**File**: [`api-persistence.md`](./api-persistence.md)

**Use When**: You need to monitor external APIs and track changes over time

**What It Does**: Queries external API, compares responses against historical baselines stored in repo-memory, detects anomalies, and creates alerts for significant changes.

**Key Features**:
- Baseline comparison with repo-memory
- Response structure validation
- Performance monitoring
- Data quality checks
- Automatic baseline updates

**Trigger Options**:
- Scheduled (hourly or daily)
- Manual workflow dispatch

**Based On**: BE-2 scenario (5.0 rating - API performance monitoring)

---

## Quick Start Guide

### Step 1: Choose a Template

Select the template that matches your automation needs from the list above.

### Step 2: Copy and Customize

```bash
# Copy template to your workflows directory
cp .github/workflows/templates/[template-name].md .github/workflows/my-workflow.md

# Edit the workflow
# - Replace [TODO] placeholders
# - Configure safe-outputs
# - Set appropriate triggers
# - Add required network domains
# - Customize the agent instructions
```

### Step 3: Complete Configuration Checklist

Each template includes a "Configuration Checklist" section at the top. Work through this checklist to ensure all required settings are configured.

### Step 4: Compile and Test

```bash
# Compile the workflow
gh aw compile my-workflow.md

# Test manually (if manual trigger enabled)
gh workflow run my-workflow.lock.yml
```

### Step 5: Monitor and Refine

After deployment, monitor the workflow's performance and adjust:
- Rate limits (max items in safe-outputs)
- Alert thresholds
- Schedule frequency
- Review criteria

---

## Template Selection Guide

Use this decision tree to choose the right template:

```
Need to review code changes?
├─ Yes → PR Code Review Automation
└─ No
   │
   Need to monitor external systems?
   ├─ Yes → API monitoring or health checks?
   │  ├─ API monitoring → API Integration with Persistence
   │  └─ Health checks → Scheduled Monitoring with Alerting
   │
   └─ No
      │
      Need visual/UI testing?
      ├─ Yes → Visual Testing Automation
      │
      └─ No
         │
         Need to generate reports?
         ├─ Yes → On-Demand Report Generation
         │
         └─ No
            │
            Need complex multi-stage analysis?
            ├─ Yes → Multi-Phase Analysis Pipelines
            │
            └─ No
               │
               Need to create many items without overwhelming team?
               └─ Yes → Rate-Limited Automation
```

---

## Common Configuration Patterns

### Pattern: Using Repo-Memory

Many templates use repo-memory for persistence:

```yaml
tools:
  repo-memory:
    branch-prefix: my-data
    description: "Persistent data storage"
    file-glob: ["*.json", "*.jsonl"]
    max-file-size: 102400  # 100KB
```

**Access in workflow**: `/tmp/gh-aw/repo-memory/default/`

### Pattern: Rate Limiting

Control output frequency to avoid overwhelming:

```yaml
safe-outputs:
  create-issue:
    max: 3  # Max 3 issues per run
  create-pull-request:
    max: 2  # Max 2 PRs per run
```

### Pattern: Scheduled Triggers

Set appropriate monitoring frequency:

```yaml
on:
  schedule: daily    # Options: hourly, daily, weekly, monthly
  workflow_dispatch: # Allow manual triggering
```

### Pattern: Network Access

Whitelist required domains:

```yaml
network:
  allowed:
    - "api.example.com"
    - "status.example.com"
```

---

## Best Practices

### 1. Start Simple

Begin with basic configuration and add complexity gradually:
- Use minimal safe-output limits initially
- Add features incrementally
- Test thoroughly before expanding scope

### 2. Monitor Performance

Track workflow performance:
- Review execution times
- Check token usage
- Monitor alert quality (false positive rate)
- Adjust thresholds based on results

### 3. Maintain Baselines

For templates using repo-memory:
- Review baselines periodically
- Update thresholds as system evolves
- Clean up old data (90-day retention recommended)

### 4. Document Customizations

Keep a changelog of your customizations:
- Why you changed specific thresholds
- Custom detection logic
- Team-specific conventions

### 5. Iterate Based on Feedback

Continuously improve workflows:
- Collect team feedback
- Adjust alert criteria
- Refine prioritization logic
- Update documentation

---

## Template Comparison

| Template | Complexity | Setup Time | Use Frequency | Requires repo-memory |
|----------|------------|------------|---------------|----------------------|
| PR Code Review | Medium | 10 min | Per PR | Optional |
| Scheduled Monitoring | Low-Medium | 15 min | Hourly/Daily | Recommended |
| Visual Testing | High | 20 min | Per PR | Optional |
| On-Demand Report | Medium | 15 min | On-demand | No |
| Multi-Phase Analysis | High | 25 min | Daily/Weekly | Required |
| Rate-Limited Automation | Medium-High | 20 min | Daily | Required |
| API Persistence | Medium | 15 min | Hourly/Daily | Required |

---

## Support and Resources

### Documentation
- [GitHub Agentic Workflows Docs](https://githubnext.github.io/gh-aw/)
- [Safe Outputs Guide](https://githubnext.github.io/gh-aw/guides/safe-outputs/)
- [Repo-Memory Guide](https://githubnext.github.io/gh-aw/features/repo-memory/)

### Getting Help
- File issues in [githubnext/gh-aw](https://github.com/githubnext/gh-aw/issues)
- Ask in `#continuous-ai` on [GitHub Next Discord](https://gh.io/next-discord)

### Contributing
Have a high-performing workflow pattern to share? Consider contributing it as a template:
1. Create a template following the existing format
2. Include configuration checklist and variations
3. Test thoroughly
4. Submit a pull request

---

## Research Background

These templates are based on analysis of high-performing workflows across multiple categories:

- **Backend Engineering (BE)**: Code review, API monitoring
- **Frontend Engineering (FE)**: Visual testing, component review
- **DevOps (DO)**: Infrastructure monitoring, rate-limited alerting
- **QA Engineering (QA)**: Test analysis, flaky test tracking
- **Product Management (PM)**: Report generation, metrics aggregation

Each template links to the specific scenarios that informed its design, ensuring patterns are based on real-world success.

---

**Last Updated**: 2024-01-16  
**Template Version**: 1.0  
**Maintenance**: Templates are maintained as part of the gh-aw repository
