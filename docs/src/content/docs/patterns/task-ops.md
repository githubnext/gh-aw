---
title: TaskOps Strategy
description: Scaffold AI-powered code improvements with research agents, planning agents, and copilot execution while keeping developers in control
---

The TaskOps strategy is a scaffolded approach to using AI agents for systematic code improvements. This strategy keeps developers in the driver's seat by providing clear decision points at each phase while leveraging AI agents to handle the heavy lifting of research, planning, and implementation.

## How TaskOps Works

The strategy follows three distinct phases:

### Phase 1: Research

A research agent (typically scheduled daily or weekly) investigates the repository under a specific angle and generates a comprehensive report. Using advanced Model Context Protocol (MCP) tools for deep analysis (static analysis, logging data, semantic search), it examines the codebase from a specific perspective and creates a detailed discussion or issue with findings, recommendations, and supporting data. Cache memory maintains historical context to track trends over time.

### Phase 2: Plan

The developer reviews the research report to determine if worthwhile improvements were identified. If the findings merit action, the developer invokes a planner agent to convert the research into specific, actionable issues. The planner splits complex work into smaller, focused tasks optimized for copilot agent success, formatting each issue with clear objectives, file paths, acceptance criteria, and implementation guidance.

### Phase 3: Assign

The developer reviews the generated issues and decides which ones to execute. Approved issues are [assigned to Copilot](https://docs.github.com/en/copilot/how-tos/use-copilot-agents/coding-agent/create-a-pr#assigning-an-issue-to-copilot) for automated implementation and can be executed sequentially or in parallel depending on dependencies. Copilot creates a pull request with the implementation for developer review and merging.

## When to Use TaskOps

Use this strategy when code improvements require systematic investigation before action, work needs to be broken down for optimal AI agent execution, or when research findings may vary in priority and require developer oversight at each phase.

## Example: Static Analysis → Plan → Fix

**Research Phase**: [`static-analysis-report.md`](https://github.com/github/gh-aw/blob/main/.github/workflows/static-analysis-report.md)

Runs daily to scan all agentic workflows with security tools (zizmor, poutine, actionlint), creating a comprehensive security discussion with clustered findings by tool and issue type, severity assessment, fix prompts, and historical trends.

**Plan Phase**: Developer reviews the security discussion and uses the `/plan` command to convert high-priority findings into issues.

**Assign Phase**: Developer [assigns generated issues to Copilot](https://docs.github.com/en/copilot/how-tos/use-copilot-agents/coding-agent/create-a-pr#assigning-an-issue-to-copilot) for automated fixes.

## Example: Duplicate Code Detection → Plan → Refactor

**Research Phase**: [`duplicate-code-detector.md`](https://github.com/github/gh-aw/blob/main/.github/workflows/duplicate-code-detector.md)

Runs daily using Serena MCP for semantic code analysis to identify exact, structural, and functional duplication. Creates one issue per distinct pattern (max 3 per run) that are [assigned to Copilot](https://docs.github.com/en/copilot/how-tos/use-copilot-agents/coding-agent/create-a-pr#assigning-an-issue-to-copilot) (via `assignees: copilot` in workflow config) since duplication fixes are typically straightforward.

**Plan Phase**: Since issues are already well-scoped, the plan phase is implicit in the research output.

**Assign Phase**: Issues are created and [assigned to Copilot](https://docs.github.com/en/copilot/how-tos/use-copilot-agents/coding-agent/create-a-pr#assigning-an-issue-to-copilot) (via `assignees: copilot`) for automated refactoring.

## Customization

Adapt the TaskOps strategy by customizing the research focus (static analysis, performance metrics, documentation quality, security, code duplication, test coverage), frequency (daily, weekly, on-demand), report format (discussions vs issues), planning approach (automatic vs manual), and assignment method (pre-assign via `assignees: copilot` in workflow config, [manual assignment through GitHub UI](https://docs.github.com/en/copilot/how-tos/use-copilot-agents/coding-agent/create-a-pr#assigning-an-issue-to-copilot), or mixed).

## Limitations

The three-phase approach takes longer than direct execution and requires developers to review research reports and generated issues. Research agents may flag issues that don't require action (false positives), and multiple phases require workflow coordination and clear handoffs. Research agents often need specialized MCPs (Serena, Tavily, etc.).

## Related Strategies

- **[Orchestration](/gh-aw/patterns/orchestration/)**: Coordinate multiple TaskOps cycles toward a shared goal
- **[Threat Detection](/gh-aw/reference/threat-detection/)**: Continuous monitoring without planning phase
- **[Custom Safe Outputs](/gh-aw/reference/custom-safe-outputs/)**: Create custom actions for plan phase
