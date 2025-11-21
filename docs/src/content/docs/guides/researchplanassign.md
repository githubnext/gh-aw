---
title: ResearchPlanAssign Strategy
description: Scaffold AI-powered code improvements with research agents, planning agents, and copilot execution while keeping developers in control
---

The ResearchPlanAssign strategy is a scaffolded approach to using AI agents for systematic code improvements. This strategy keeps developers in the driver's seat by providing clear decision points at each phase while leveraging AI agents to handle the heavy lifting of research, planning, and implementation.

## How ResearchPlanAssign Works

The strategy follows three distinct phases:

### Phase 1: Research

A research agent (typically scheduled daily or weekly) investigates the repository under a specific angle and generates a comprehensive report. The research agent:

- Uses advanced MCP tools for deep analysis (static analysis, logging data, semantic search)
- Examines the codebase from a specific perspective (e.g., "Are the docs in sync?", "What code is duplicated?", "What security issues exist?")
- Creates a detailed discussion or issue with findings, recommendations, and supporting data
- Maintains historical context using cache memory to track trends over time

### Phase 2: Plan

The developer reviews the research report to determine if worthwhile improvements were identified. If the findings merit action, the developer invokes a planner agent to:

- Convert the research report into specific, actionable issues
- Split complex work into smaller, focused tasks optimized for copilot agent success
- Format each issue with clear objectives, file paths, acceptance criteria, and implementation guidance
- Link all generated issues to the parent research discussion or report

### Phase 3: Assign

The developer reviews the generated issues and decides which ones to execute. For approved issues:

- Assign to `@copilot` for automated implementation
- Issues can be executed sequentially or in parallel depending on dependencies
- Each copilot agent creates a pull request with the implementation
- Developer reviews and merges completed work

## When to Use ResearchPlanAssign

Use this strategy when:

- Code improvements require systematic investigation before action
- Work needs to be broken down for optimal AI agent execution
- Developer oversight and approval is important at each phase
- Research findings may vary in priority or relevance
- You want to maintain control while leveraging AI capabilities

## Example Implementations

The following workflows demonstrate the ResearchPlanAssign pattern in practice:

### Static Analysis → Plan → Fix

**Research Phase**: [`static-analysis-report.md`](https://github.com/githubnext/gh-aw/blob/main/.github/workflows/static-analysis-report.md)

Runs daily to scan all agentic workflows with security tools (zizmor, poutine, actionlint):

```aw wrap
---
on:
  schedule:
    - cron: "0 9 * * *"  # Daily at 9 AM UTC
engine: claude
tools:
  github:
    toolsets: [default, actions]
  cache-memory: true
safe-outputs:
  create-discussion:
    category: "security"
imports:
  - shared/mcp/gh-aw.md
---

# Static Analysis Report

Scan all workflows with static analysis tools, cluster findings by type,
and provide actionable fix suggestions. Store results in cache memory
for trend analysis.
```

The research agent creates a comprehensive security discussion with:
- Clustered findings by tool and issue type
- Severity assessment and affected workflows
- Detailed fix prompt for the most common or severe issue
- Historical trends comparing with previous scans

**Plan Phase**: Developer reviews the security discussion and uses the `/plan` command to convert high-priority findings into issues:

```aw wrap
---
on:
  command:
    name: plan
safe-outputs:
  create-issue:
    title-prefix: "[task] "
    max: 5
---

# Planning Assistant

Break down the security findings into actionable sub-issues
optimized for copilot agent execution.
```

**Assign Phase**: Developer assigns generated issues to `@copilot` for automated fixes.

### Duplicate Code Detection → Plan → Refactor

**Research Phase**: [`duplicate-code-detector.md`](https://github.com/githubnext/gh-aw/blob/main/.github/workflows/duplicate-code-detector.md)

Runs daily to identify code duplication patterns:

```aw wrap
---
on:
  schedule:
    - cron: "0 21 * * *"  # Daily at 9 PM UTC
engine: codex
imports:
  - shared/mcp/serena.md
safe-outputs:
  create-issue:
    title-prefix: "[duplicate-code] "
    assignees: copilot
    max: 3
---

# Duplicate Code Detection

Analyze code using Serena's semantic analysis to identify
duplicated patterns. Create separate issues for each distinct
duplication pattern found.
```

The research agent:
- Uses Serena MCP for semantic code analysis
- Identifies exact, structural, and functional duplication
- Creates one issue per distinct pattern (max 3 per run)
- Assigns directly to `@copilot` since duplication fixes are typically straightforward

**Plan Phase**: Since issues are already well-scoped, the plan phase is implicit in the research output.

**Assign Phase**: Issues are pre-assigned to `@copilot` for automated refactoring.

### File Size Analysis → Plan → Refactor

**Research Phase**: [`daily-file-diet.md`](https://github.com/githubnext/gh-aw/blob/main/.github/workflows/daily-file-diet.md)

Monitors file sizes and identifies oversized files:

```aw wrap
---
on:
  schedule:
    - cron: "0 13 * * 1-5"  # Weekdays at 1 PM UTC
engine: codex
imports:
  - shared/mcp/serena.md
safe-outputs:
  create-issue:
    title-prefix: "[file-diet] "
    max: 1
---

# Daily File Diet Agent

Identify the largest Go source file and determine if it requires
refactoring. Create an issue with specific guidance for splitting
into smaller, focused files.
```

The research agent:
- Finds files exceeding healthy size thresholds (1000+ lines)
- Analyzes file structure to identify natural split boundaries
- Creates a detailed refactoring issue with suggested approach
- Includes file organization recommendations

**Plan Phase**: The research issue already contains a concrete refactoring plan.

**Assign Phase**: Developer reviews and assigns to `@copilot` or handles manually depending on complexity.

### Deep Research → Plan → Implementation

**Research Phase**: [`scout.md`](https://github.com/githubnext/gh-aw/blob/main/.github/workflows/scout.md)

Performs deep research investigations:

```aw wrap
---
on:
  command:
    name: scout
engine: claude
imports:
  - shared/mcp/tavily.md
  - shared/mcp/arxiv.md
  - shared/mcp/deepwiki.md
safe-outputs:
  add-comment:
    max: 1
---

# Scout Deep Research Agent

Conduct comprehensive research using multiple sources
and synthesize findings into actionable recommendations.
```

The research agent:
- Uses multiple research MCPs (Tavily, arXiv, DeepWiki)
- Gathers information from diverse sources
- Creates structured research summary with recommendations
- Posts findings as a comment on the triggering issue

**Plan Phase**: Developer uses `/plan` command on the research comment to convert recommendations into issues.

**Assign Phase**: Developer assigns resulting issues to appropriate agents or team members.

## Best Practices

### Research Agent Design

- **Schedule appropriately**: Daily for critical metrics, weekly for comprehensive analysis
- **Use cache memory**: Store historical data to identify trends and track improvements
- **Focus scope**: Each research agent should investigate one specific angle or concern
- **Be actionable**: Research reports should lead to concrete, implementable recommendations
- **Avoid noise**: Only create reports when findings exceed meaningful thresholds

### Planning Phase

- **Review carefully**: Not all research findings require immediate action
- **Prioritize**: Focus on high-impact issues first
- **Right-size tasks**: Break work into chunks suitable for AI agent execution
- **Clear objectives**: Each issue should have unambiguous success criteria
- **Link context**: Reference the parent research report for full context

### Assignment Phase

- **Sequential vs parallel**: Consider dependencies when assigning multiple issues
- **Agent capabilities**: Some tasks are better suited for human developers
- **Review PRs**: Always review AI-generated code before merging
- **Iterate**: Refine prompts and task descriptions based on agent performance

## Customization

Adapt the ResearchPlanAssign strategy to your needs:

- **Research focus**: Static analysis, performance metrics, documentation quality, security, code duplication, test coverage
- **Research frequency**: Daily, weekly, on-demand via commands
- **Report format**: Discussions for in-depth analysis, issues for immediate action
- **Planning approach**: Automatic (research creates issues directly), manual (developer uses `/plan` command)
- **Assignment method**: Pre-assign to `@copilot`, manual assignment, mixed approach

## Benefits

The ResearchPlanAssign strategy provides:

- **Developer control**: Clear decision points at research review and issue assignment
- **Systematic improvement**: Regular, focused analysis identifies issues proactively
- **Optimal task sizing**: Planning phase ensures tasks are properly scoped for AI agents
- **Historical context**: Cache memory tracks trends and measures improvement over time
- **Reduced overhead**: Automation handles research and execution while developers focus on decisions

## Limitations

Be aware of these considerations:

- **Latency**: Three-phase approach takes longer than direct execution
- **Review burden**: Developers must review research reports and generated issues
- **False positives**: Research agents may flag issues that don't require action
- **Coordination**: Multiple phases require workflow coordination and clear handoffs
- **Tool requirements**: Research agents often need specialized MCPs (Serena, Tavily, etc.)

## Related Strategies

- **[Campaigns](/gh-aw/guides/campaigns/)**: Coordinate multiple ResearchPlanAssign cycles toward a shared goal
- **[Threat Detection](/gh-aw/guides/threat-detection/)**: Continuous monitoring without planning phase
- **[Custom Safe Outputs](/gh-aw/guides/custom-safe-outputs/)**: Create custom actions for plan phase

:::note
The ResearchPlanAssign strategy works best when research findings vary in relevance and priority. For issues that always require immediate action, consider using direct execution workflows instead.
:::
