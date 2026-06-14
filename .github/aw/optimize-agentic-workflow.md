---
description: Analyze and reduce token consumption in agentic workflows — audit-based measurement, DataOps, gh-proxy, sub-agents, and prompt optimization.
disable-model-invocation: true
---

# Agentic Workflow Token Optimizer

Help users reduce the AI token usage and cost of GitHub Agentic Workflows in this repository.

## Load These References First

- [github-agentic-workflows.md](github-agentic-workflows.md)
- [token-optimization.md](token-optimization.md)
- [workflow-editing.md](workflow-editing.md)
- [syntax.md](syntax.md)

Load these only when relevant:

- [experiments.md](experiments.md)
- [safe-outputs.md](safe-outputs.md)

## Available Commands

```bash
gh aw audit <run-id> --json
gh aw audit <base-run-id> <optimized-run-id>
gh aw logs <workflow-name> --json
gh aw compile <workflow-name>
gh aw status
```

## Start the Conversation

Ask for one of these inputs:

- a workflow run URL (or run ID) to analyze
- a workflow name to review the source
- the guardrail that was exceeded (max-ai-credits, max-daily-ai-credits, max-tool-denials, max-turns / timeout)

## Fast Path: Run URL Provided

If the user gives a GitHub Actions run URL:

1. Extract the run ID
2. Run `gh aw audit <run-id> --json`
3. Inspect `agent_usage.aic`, `agent_usage.input_tokens`, `agent_usage.output_tokens`, `agent_usage.cache_read_tokens`
4. Identify the most expensive phases before asking additional questions

## Guardrail-Specific Entry Points

### `max-ai-credits` exceeded

The workflow was stopped because it consumed more AI Credits than the configured per-run budget.

Priority checks:
1. Which tool calls dominated token usage? (`token-usage.jsonl`)
2. Is the prompt front-loading large payloads that could be fetched on demand?
3. Are there repetitive extraction steps that sub-agents could handle cheaply?
4. Does the frontier model handle tasks that a small model could do?

### `max-daily-ai-credits` exceeded

The workflow is being blocked because its 24-hour AI Credits budget is exhausted.

Priority checks:
1. What is the run cadence? (scheduled too frequently?)
2. Does the workflow use cheap triage before escalating to the frontier model?
3. Is batching or caching applicable to reduce run frequency?
4. Are there noop early-exits for events that do not require agent action?

### `max-tool-denials` exceeded

The Copilot SDK hit the tool-denial threshold, indicating the prompt attempted actions outside the allowed tool policy.

Priority checks:
1. What tool was repeatedly denied? (last denied reason in the failure issue)
2. Is the tool missing from the workflow's permissions/firewall config?
3. Can the prompt be revised to avoid the denied operation entirely?
4. Would a DataOps pre-step satisfy the data need without a tool call?

### Timeout / `max-turns` exceeded

The agent ran out of time or turns before completing the task.

Priority checks:
1. Is the task decomposable into smaller, faster sub-tasks?
2. Are there long-running tool calls that could be replaced with DataOps pre-steps?
3. Is the prompt asking the agent to do too much in one run?
4. Can `max-turns` or `timeout-minutes` be raised, or should the task be split?

## Optimization Analysis Plan

After measuring token usage, produce a prioritized plan:

1. **Measure** — run `gh aw audit <run-id>` and summarize AI Credits and per-call token breakdown
2. **Identify top cost drivers** — list the three most expensive phases/tool calls
3. **Apply quick wins first** — DataOps pre-steps, `gh-proxy`, `cli-proxy`, prompt trimming
4. **Sub-agent delegation** — identify repetitive per-item loops suitable for small-model workers
5. **Prompt caching** — verify stable context appears before dynamic content
6. **Experiment** — add an `experiments:` entry with `metric: "aic"` to measure the change
7. **Validate quality** — confirm the optimized run produces equivalent safe outputs

Present the plan clearly before making any edits. Confirm with the user before applying changes.

## Editing Workflow

1. Edit `.github/workflows/<workflow-name>.md`
2. Recompile: `gh aw compile <workflow-name>`
3. Commit both the source and the generated `.lock.yml`
4. Report the estimated savings and link to the PR or commit
