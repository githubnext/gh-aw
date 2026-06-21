---
description: Decision tree for selecting the right `.github/aw/*.md` guidance file to load for workflow design, updates, debugging, and optimization.
disable-model-invocation: true
---

# Agentic Workflow Decision Tree

Use this tree to decide which guidance files to load first.

## 1) What are you trying to do?

- **Create a new workflow**
  - Load: [create-agentic-workflow.md](create-agentic-workflow.md)
  - Then load: [workflow-patterns.md](workflow-patterns.md), [triggers.md](triggers.md), [safe-outputs.md](safe-outputs.md)
- **Update an existing workflow**
  - Load: [update-agentic-workflow.md](update-agentic-workflow.md)
  - Then load: [workflow-editing.md](workflow-editing.md), [reuse.md](reuse.md)
- **Debug or audit a failing workflow/run**
  - Load: [debug-agentic-workflow.md](debug-agentic-workflow.md)
  - Then load: [github-agentic-workflows.md](github-agentic-workflows.md), [report.md](report.md)
- **Upgrade and fix deprecated syntax/features**
  - Load: [upgrade-agentic-workflows.md](upgrade-agentic-workflows.md)
  - Then load: [syntax.md](syntax.md), [syntax-tools-imports.md](syntax-tools-imports.md)
- **Optimize cost/tokens/performance**
  - Load: [optimize-agentic-workflow.md](optimize-agentic-workflow.md)
  - Then load: [token-optimization.md](token-optimization.md), [experiments.md](experiments.md)

## 2) What subsystem is most relevant?

- **Safe outputs / write automation**
  - Load: [safe-outputs.md](safe-outputs.md)
  - If needed: [safe-outputs-runtime.md](safe-outputs-runtime.md), [safe-outputs-automation.md](safe-outputs-automation.md), [safe-outputs-management.md](safe-outputs-management.md)
- **MCP/GitHub API tooling**
  - Load: [github-mcp-server.md](github-mcp-server.md), [agentic-workflows-mcp.md](agentic-workflows-mcp.md), [mcp-clis.md](mcp-clis.md)
- **Syntax/frontmatter/imports**
  - Load: [syntax.md](syntax.md), [syntax-core.md](syntax-core.md), [syntax-tools-imports.md](syntax-tools-imports.md)
- **Patterns/architecture**
  - Load: [patterns.md](patterns.md), [workflow-patterns.md](workflow-patterns.md)
- **Memory/persistence/reporting**
  - Load: [memory.md](memory.md), [report.md](report.md), [campaign.md](campaign.md)
- **Charts/visualization**
  - Load: [charts.md](charts.md), [charts-trending.md](charts-trending.md), [asciicharts.md](asciicharts.md)

## 3) Extra overlays (load only when needed)

- **Sub-agents:** [subagents.md](subagents.md)
- **Skills routing:** [skills.md](skills.md)
- **Context expressions / `{{#if}}`:** [context.md](context.md)
- **Network/firewall rules:** [network.md](network.md)
- **CLI ↔ MCP command mapping:** [cli-commands.md](cli-commands.md)
- **Specialized modes:** [pr-reviewer.md](pr-reviewer.md), [agentic-chat.md](agentic-chat.md), [dependabot.md](dependabot.md), [visual-regression.md](visual-regression.md), [test-coverage.md](test-coverage.md), [test-expression.md](test-expression.md)

## 4) Minimum default load set

If uncertain, start with:

1. [github-agentic-workflows.md](github-agentic-workflows.md)
2. [workflow-editing.md](workflow-editing.md)
3. [syntax.md](syntax.md)
4. [safe-outputs.md](safe-outputs.md)
5. One task-specific file from section 1
