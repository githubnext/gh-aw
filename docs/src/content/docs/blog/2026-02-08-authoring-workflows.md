---
title: "Authoring New Workflows in Peli's Agent Factory"
description: "A practical guide to creating effective agentic workflows"
authors:
  - dsyme
  - pelikhan
  - mnkiefer
date: 2026-02-08
draft: true
prev:
  link: /gh-aw/blog/2026-02-05-how-workflows-work/
  label: How Workflows Work
next:
  link: /gh-aw/blog/2026-02-11-getting-started/
  label: Getting Started
---

[Previous Article](/gh-aw/blog/2026-02-05-how-workflows-work/)

---

<img src="/gh-aw/peli.png" alt="Peli de Halleux" width="200" style="float: right; margin: 0 0 20px 20px; border-radius: 8px;" />

Now that you understand [how workflows operate under the hood](/gh-aw/blog/2026-02-05-how-workflows-work/), this guide walks through creating your own. We'll share patterns, tips, and best practices from our collection of automated agentic workflows.

## The Authoring Process

Creating an effective agentic workflow breaks down into five straightforward stages:

```text
1. Define Purpose → 2. Choose Pattern → 3. Write Prompt → 4. Configure → 5. Test & Iterate
```

## Stage 1: Define Purpose

Before writing code, clarify what you're building:

**Problem**: Be specific. Instead of "improve code quality," define concrete goals like "detect duplicate code blocks over 50 lines" or "identify functions without test coverage."

**Audience**: Team members need conversational tone with explanations. Maintainers need technical detail and reproduction steps. External contributors need welcoming tone with context.

**Output**: Define the artifact - issue with findings, PR with fixes, discussion with metrics, or comment with analysis.

**Cadence**: Match frequency to need - continuous (quality checks), hourly (CI monitoring), daily (health checks), weekly (trend reports), or on-demand (reviews).

## Stage 2: Choose Your Pattern

Select from the [12 design patterns](03-design-patterns.md) and [9 operational patterns](04-operational-patterns.md):

### Quick Pattern Selection Guide

**For observability without changes:**
→ Read-Only Analyst + DailyOps

**For interactive assistance:**
→ ChatOps Responder + Role-gated

**For automated maintenance:**
→ Continuous Janitor + Human-in-the-Loop

**For quality enforcement:**
→ Quality Guardian + Network Restricted

**For complex improvements:**
→ Multi-Phase Improver + ResearchPlanAssign

**For ecosystem management:**
→ Meta-Agent Optimizer + Meta-Agent Orchestrator

## Stage 3: Write an Effective Prompt

The prompt is the heart of your workflow. Great prompts are clear, structured, contextual, personality-driven, and tool-aware.

### Clear and Specific

Instead of "Analyze the code," write "Analyze the codebase for functions longer than 100 lines. For each long function, identify: primary responsibility, potential split points, and suggested refactoring approach."

### Structured

Use headings, lists, and examples:

```markdown
## Task

Identify test coverage gaps in the repository.

## Process

1. **Analyze test files** in `test/` directory
2. **Map test coverage** to source files in `src/`
3. **Identify gaps** where source files lack corresponding tests
4. **Prioritize** based on file complexity and importance

## Output

Create an issue titled "Test Coverage Gaps - [Date]" with:

- Summary of overall coverage
- List of untested files (highest priority first)
- Suggested test cases for top 3 gaps
- Link to coverage report

## Example

    ### Untested Files
    
    1. **src/payment.js** (Critical - handles transactions)
       - Test cases needed:
         - Success path with valid payment
         - Failure path with invalid card
         - Edge case: $0.00 transaction
```

### Contextual

Provide relevant background:

```markdown
## Context

This repository uses Jest for unit tests and Playwright for integration tests.
Tests are co-located with source files as `*.test.js`.

Our coverage goal is 80% for critical paths (payment, auth, user management).

## Conventions

- One test file per source file
- Test names follow pattern: "should [expected behavior] when [condition]"
- Use mocks from `test/mocks/` directory
```

### Personality-Driven

Give your agent character: meticulous auditor (extreme attention to detail), helpful janitor (small incremental improvements), creative poet (capture team achievements), or critical reviewer (challenge assumptions, push for excellence).

### Tool-Aware

Reference available tools explicitly:

```markdown
## Tools Available

- **Serena MCP**: Use for semantic code analysis
- **GitHub API**: Query issues, PRs, workflow runs
- **jq**: Process JSON data from API responses
- **Python + matplotlib**: Generate visualizations

## Example Usage

Use Serena to find duplicate code:
```bash
serena find-duplicates --min-lines 30 --similarity 0.8
```

## Stage 4: Configure Effectively

### Frontmatter Best Practices

**Start minimal**: Begin with read-only permissions, add write permissions only when safe outputs require them.

**Use imports**: Reference common needs like `shared/reporting.md`, `shared/python-dataviz.md`, and `shared/jqschema.md`.

**Add guardrails**: Configure safe outputs with `max_items`, `close_older`, `expire`, and `if_no_changes: skip`.

**Allowlist tools**: Explicitly list needed GitHub toolsets (`[repos, issues]`) and bash commands (`[git, jq, python3]`).

**Restrict network**: Specify allowed domains (`api.github.com`, `api.tavily.com`). Avoid wildcards in production.

## Stage 5: Test and Iterate

### Local Testing

Before deploying, test locally:

```bash
# Validate syntax
gh aw compile workflow.md

# Check for issues
gh aw validate workflow.md

# Test compilation
gh aw compile --output test.lock.yml workflow.md
```

### Staged Rollout

**Phase 1**: Start with manual workflow dispatch. Run several times, review outputs, iterate on prompt.

**Phase 2**: Add limited schedule (once weekly) with `stop-after: "+1mo"`. Monitor for a month, adjust based on feedback.

**Phase 3**: Remove time limit, adjust schedule to production cadence (e.g., weekdays).

### Iteration Checklist

After each run, review:

- [ ] Did the agent understand the task correctly?
- [ ] Were the outputs useful and actionable?
- [ ] Did it respect all guardrails (max_items, etc.)?
- [ ] Were there any security concerns?
- [ ] Did it run within expected time/cost?
- [ ] What could be improved in the next version?

## Common Patterns from the Factory

### Pattern: The Weekly Analyst

```markdown
---
description: Weekly repository health report
imports:
  - shared/reporting.md
  - shared/python-dataviz.md
on:
  schedule: "0 9 * * 0"  # Sunday morning
permissions:
  contents: read
safe_outputs:
  create_discussion:
    title: "Weekly Health Report - {date}"
    category: "Reports"
---

## Weekly Repository Health Analysis

Create a comprehensive health report:

1. **Issue Metrics**: Open, closed, average resolution time
2. **PR Metrics**: Open, merged, average review time
3. **CI Health**: Success rate, failure patterns
4. **Security**: Open vulnerabilities, compliance status
5. **Trends**: Week-over-week comparisons

Include:
- Summary dashboard
- Key insights and recommendations
- Visualizations (charts, graphs)
- Links to detailed data
```

### Pattern: The ChatOps Responder

```markdown
---
description: On-demand code review
on:
  issue_comment:
    types: [created]
permissions:
  contents: read
  pull-requests: write
safe_outputs:
  add_comment:
    prefix: "## Review Complete\n\n"
---

## Grumpy Code Reviewer

When a user comments `/grumpy` on a PR:

1. Verify user has collaborator access
2. Download PR diff
3. Perform critical code review:
   - Architecture issues
   - Performance concerns
   - Security vulnerabilities
   - Style violations
4. Post review as comment

Use a critical, direct tone. Be thorough but constructive.
Never approve - always find room for improvement.
```

### Pattern: The Multi-Phase Improver

```markdown
---
description: Systematic test improvement
imports:
  - shared/reporting.md
on:
  schedule: "0 10 * * 1-5"  # Weekdays
permissions:
  contents: read
  issues: write
  pull-requests: write
repo-memory:
  - id: test-improver
    create-orphan: true
safe_outputs:
  create_discussion:
    category: "Plans"
  create_issue:
    labels: ["testing", "improvement"]
  create_pull_request:
    labels: ["testing", "automated"]
---

## Daily Test Improver

Systematically improve test coverage over multiple days:

### Phase 1: Research (Days 1-2)
- Analyze current test coverage
- Identify coverage gaps
- Prioritize files by importance
- Create discussion with findings

### Phase 2: Plan (Days 3-4)
- Design test strategies for top gaps
- Create issues for implementation
- Break down into manageable tasks
- Get human approval on plan

### Phase 3: Implement (Days 5+)
- Implement tests from approved issues
- Create PRs with new tests
- One PR per day for easy review
- Track progress in repo-memory

Check repo-memory/test-improver/ for current phase.
```

## Advanced Techniques

### Using Repo-Memory

Store state across runs:

```yaml
repo-memory:
  - id: daily-metrics
    max-file-size: 1MB
    max-files: 100
```

In your prompt:

```markdown
Check `/tmp/gh-aw/repo-memory/daily-metrics/` for historical data.
Compare today's metrics with last week's baseline.
Store today's results as `metrics-{date}.json`.
```

### Cache-Memory for ChatOps

Prevent duplicate work:

```markdown
Before analyzing, check cache-memory/{issue-number}.json.
If analysis exists and issue hasn't changed, skip analysis.
Store new analysis results for future reference.
```

### Conditional Execution

```markdown
## Pre-flight Checks

1. Check if CI is currently failing - if not, exit early
2. Check if diagnostic issue already exists - if so, update it
3. Check if this is a known transient failure - if so, just log it
```

## Common Mistakes to Avoid

**Vague prompts**: Use specific criteria like "flag functions with cyclomatic complexity > 15" instead of "tell me what's wrong."

**Missing constraints**: Always set `max_items`, `close_older`, and `expire` to prevent duplicate issues.

**Too many permissions**: Start with `contents: read`, add specific permissions as needed. Never use `write-all`.

**No testing**: Test manually first, then use limited schedule with expiration before production.

**Overly complex**: Create multiple focused workflows instead of one workflow doing 10 things.

## Resources and Examples

### Example Workflows to Study

Browse the factory for inspiration:

**Simple Examples:**

- [`poem-bot`](https://github.com/githubnext/gh-aw/blob/bb7946527af340043f1ebb31fc21bd491dd0f42d/.github/workflows/poem-bot.md) - ChatOps personality
- [`daily-repo-chronicle`](https://github.com/githubnext/gh-aw/blob/bb7946527af340043f1ebb31fc21bd491dd0f42d/.github/workflows/daily-repo-chronicle.md) - Daily summary
- [`issue-triage-agent`](https://github.com/githubnext/gh-aw/blob/bb7946527af340043f1ebb31fc21bd491dd0f42d/.github/workflows/issue-triage-agent.md) - Event-driven

**Intermediate Examples:**

- [`ci-doctor`](https://github.com/githubnext/gh-aw/blob/bb7946527af340043f1ebb31fc21bd491dd0f42d/.github/workflows/ci-doctor.md) - Diagnostic analysis
- [`glossary-maintainer`](https://github.com/githubnext/gh-aw/blob/bb7946527af340043f1ebb31fc21bd491dd0f42d/.github/workflows/glossary-maintainer.md) - Content sync
- [`terminal-stylist`](https://github.com/githubnext/gh-aw/blob/bb7946527af340043f1ebb31fc21bd491dd0f42d/.github/workflows/terminal-stylist.md) - Code analysis

**Advanced Examples:**

- [`daily-test-improver`](https://github.com/githubnext/agentics/blob/main/workflows/daily-test-improver.md) - Multi-phase
- [`agent-performance-analyzer`](https://github.com/githubnext/gh-aw/blob/bb7946527af340043f1ebb31fc21bd491dd0f42d/.github/workflows/agent-performance-analyzer.md) - Meta-agent
- [`workflow-health-manager`](https://github.com/githubnext/gh-aw/blob/bb7946527af340043f1ebb31fc21bd491dd0f42d/.github/workflows/workflow-health-manager.md) - Orchestration

### Documentation

- [Workflow Reference](https://githubnext.github.io/gh-aw/reference/workflows/)
- [Safe Outputs Guide](https://githubnext.github.io/gh-aw/reference/safe-outputs/)
- [Tools Reference](https://githubnext.github.io/gh-aw/reference/tools/)
- [Examples Gallery](https://githubnext.github.io/gh-aw/examples/)

### Community

- Share workflows in [Agentics Collection](https://github.com/githubnext/agentics)
- Get help in [GitHub Next Discord](https://gh.io/next-discord) #continuous-ai
- Browse [discussions](https://github.com/githubnext/gh-aw/discussions)

## Your Turn

You now have everything you need: start with one repetitive task, follow proven patterns, test iteratively (manual → limited → production), and share learnings with the community. The next article walks you through getting your first workflow running.

## What's Next?

_More articles in this series coming soon._

[Previous Article](/gh-aw/blog/2026-02-05-how-workflows-work/)
