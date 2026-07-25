---
name: agentic-workflows
description: Route gh-aw workflow design/create/debug/upgrade requests to the right prompts.
---

# Agentic Workflows Router

Use this skill when a user asks to design, create, update, debug, or upgrade GitHub Agentic Workflows in this repository.

This skill is a dispatcher: identify the task type, load the matching workflow prompt/skill file, and follow it directly. Keep responses concise and ask a clarifying question if the correct prompt is unclear.

Repository overlay (optional):
- If `.github/aw/instructions.md` exists, load it with `@.github/aw/instructions.md` after loading the matched prompt/skill.
- Precedence: repository overlay instructions override upstream defaults when they conflict.

Read only the files you need:
Load these files from `github/gh-aw` (they are not available locally).
- `.github/aw/action-container-substitutions.md`
- `.github/aw/agentic-chat.md`
- `.github/aw/agentic-workflows-mcp.md`
- `.github/aw/asciicharts.md`
- `.github/aw/campaign.md`
- `.github/aw/charts-trending.md`
- `.github/aw/charts.md`
- `.github/aw/cli-commands.md`
- `.github/aw/configure-agentic-engine.md`
- `.github/aw/context.md`
- `.github/aw/create-agentic-workflow-trigger-details.md`
- `.github/aw/create-agentic-workflow.md`
- `.github/aw/create-shared-agentic-workflow.md`
- `.github/aw/debug-agentic-workflow.md`
- `.github/aw/dependabot.md`
- `.github/aw/deployment-status.md`
- `.github/aw/designer.md`
- `.github/aw/evals.md`
- `.github/aw/experiments.md`
- `.github/aw/github-agentic-workflows.md`
- `.github/aw/github-mcp-server.md`
- `.github/aw/instructions.md`
- `.github/aw/llms.md`
- `.github/aw/loop.md`
- `.github/aw/lsp.md`
- `.github/aw/mcp-clis.md`
- `.github/aw/memory-stateful-patterns.md`
- `.github/aw/memory.md`
- `.github/aw/messages.md`
- `.github/aw/multi-agent-research.md`
- `.github/aw/network.md`
- `.github/aw/optimize-agentic-workflow.md`
- `.github/aw/patterns.md`
- `.github/aw/pr-reviewer.md`
- `.github/aw/report.md`
- `.github/aw/reuse.md`
- `.github/aw/safe-outputs-automation.md`
- `.github/aw/safe-outputs-content.md`
- `.github/aw/safe-outputs-management.md`
- `.github/aw/safe-outputs-runtime.md`
- `.github/aw/safe-outputs.md`
- `.github/aw/serena-tool.md`
- `.github/aw/shared-safe-jobs.md`
- `.github/aw/skills.md`
- `.github/aw/subagents.md`
- `.github/aw/syntax-agentic.md`
- `.github/aw/syntax-core.md`
- `.github/aw/syntax-tools-imports.md`
- `.github/aw/syntax.md`
- `.github/aw/test-coverage.md`
- `.github/aw/test-expression.md`
- `.github/aw/token-optimization.md`
- `.github/aw/triggers.md`
- `.github/aw/update-agentic-workflow.md`
- `.github/aw/upgrade-agentic-workflows.md`
- `.github/aw/visual-regression.md`
- `.github/aw/workflow-constraints.md`
- `.github/aw/workflow-editing.md`
- `.github/aw/workflow-patterns.md`

## Task Selection

When the task type is not immediately clear from the user's message, present this menu and ask the user to pick:

1. **Create / design** a new workflow
2. **Update** an existing workflow
3. **Debug** or audit a workflow run
4. **Optimize** a workflow for token consumption
5. **Upgrade** workflows and fix deprecations

Once the task is known, load and follow the matching prompt:

| Task | Prompt |
|---|---|
| 1. Create / design | `.github/aw/create-agentic-workflow.md` |
| 2. Update | `.github/aw/update-agentic-workflow.md` |
| 3. Debug / audit | `.github/aw/debug-agentic-workflow.md` |
| 4. Optimize | `.github/aw/optimize-agentic-workflow.md` |
| 5. Upgrade | `.github/aw/upgrade-agentic-workflows.md` |
| Configure engine | `.github/aw/configure-agentic-engine.md` |
| Shared components / MCP wrappers | `.github/aw/create-shared-agentic-workflow.md` |
| Report-generating workflow | `.github/aw/report.md` |
| Dependabot manifest PR | `.github/aw/dependabot.md` |
| Coverage analysis | `.github/aw/test-coverage.md` |
| Charts | `.github/aw/asciicharts.md` |
| CLI → MCP mapping | `.github/aw/cli-commands.md` |
| Patterns / architecture | `.github/aw/patterns.md` |
| Multi-agent research | `.github/aw/multi-agent-research.md` |

When the task involves OTEL, OTLP, traces, observability backends, or telemetry-driven analysis, also read and follow `skills/otel-queries/SKILL.md` after loading the matching prompt.
