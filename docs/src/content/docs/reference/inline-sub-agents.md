---
title: Inline Sub-Agents
description: Define sub-agents directly inside a workflow markdown file using a level-2 heading delimiter.
sidebar:
  order: 645
---

An inline sub-agent is a named agent definition embedded directly in a workflow markdown file. Instead of creating a separate file in `.github/agents/`, you define the agent's frontmatter and instructions in a dedicated section of the same workflow file.

Inline sub-agents are enabled by default. `features.inline-agents` is deprecated/no-op, and `inline-sub-agents: false` is rejected at compile time.

## Syntax

Start a sub-agent block with a level-2 heading in the following form:

```markdown
## agent: `name`
```

The block continues until the next `##` heading or end of file. There is no explicit closing marker.

### Name constraints

- Must start with a lowercase letter (`a–z`)
- May contain only `a–z`, `0–9`, `_`, and `-`
- Examples: `file-summarizer`, `code_reviewer`, `pr-analyst`

### Structure

Each sub-agent block contains:

1. **YAML frontmatter** (optional) — wrapped in `---` delimiters
2. **Instructions** — natural language prompt for the agent

```markdown
## agent: `file-summarizer`
---
model: claude-haiku-4.5
description: Summarizes the content of a file in a few concise sentences
---
You are a file summarization assistant. When given a file path, read the file
and return a brief summary (2–4 sentences) describing its purpose and key
contents. Be concise and factual.
```

## Frontmatter fields

| Field | Required | Description |
|---|---|---|
| `model` | No | AI model to use (e.g. `claude-haiku-4.5`). Defaults to the parent workflow's model. |
| `description` | No | Short description of the sub-agent's purpose. |

## Runtime behavior

At runtime, `actions/setup` extracts each inline sub-agent block and writes it to:

```text
.agents/agents/<name>.agent.md
```

The Copilot CLI discovers agent files in `.agents/agents/` natively. To use a sub-agent, instruct the parent workflow's prompt to invoke it by name:

```aw wrap
## Test Requirements

15. **Sub-Agent Testing**: Use the `file-summarizer` sub-agent to summarize the
    file `.github/workflows/smoke-copilot.md`. Verify the sub-agent returns a
    brief summary (2–4 sentences). Mark this test as ❌ if the sub-agent is
    unavailable or returns an error.
```

The Copilot CLI finds `.agents/agents/file-summarizer.agent.md` and invokes it automatically.

## Complete example

The following excerpt shows a full workflow that defines and uses an inline sub-agent.

```aw wrap
---
on:
  workflow_dispatch:
engine: copilot
---

# File Summary Task

Use the `file-summarizer` sub-agent to summarize `README.md` and add a comment
to the current pull request with the result.

## agent: `file-summarizer`
---
model: claude-haiku-4.5
description: Summarizes the content of a file in a few concise sentences
---
You are a file summarization assistant. When given a file path, read the file
and return a brief summary (2–4 sentences) describing its purpose and key
contents. Be concise and factual.
```

The sub-agent block at the bottom is extracted before the workflow runs and has no effect on the parent workflow's instructions.

## Multiple sub-agents

A single workflow file may contain more than one sub-agent block. Each block starts with its own `## agent: \`name\`` heading and ends at the next `##` heading or EOF.

```aw wrap
## agent: `summarizer`
---
model: claude-haiku-4.5
description: Summarizes files concisely
---
Summarize the given file in 2–4 sentences.

## agent: `reviewer`
---
model: claude-sonnet-4.5
description: Reviews code for quality issues
---
Review the given code for bugs, style issues, and potential improvements.
```

## Related Documentation

- [Importing Copilot Agent Files](/gh-aw/reference/copilot-custom-agents/) — Importing agents from `.github/agents/`
- [Markdown](/gh-aw/reference/markdown/) — Workflow markdown body reference
- [Workflow Structure](/gh-aw/reference/workflow-structure/) — Overall workflow file organization
- [Frontmatter](/gh-aw/reference/frontmatter/) — YAML configuration options
