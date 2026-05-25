---
description: Guide for defining inline sub-agents in workflow markdown files — syntax, engine placement, frontmatter fields, and best practices.
---

# Inline Sub-Agents

Inline sub-agents let you define specialised agents directly inside a workflow markdown file. At runtime the sub-agent sections are extracted from the prompt (after `{{#runtime-import}}` macros are resolved) and written to the engine-specific agents directory so the engine CLI can discover and invoke them.

---

## Enabling the Feature

Inline sub-agent extraction is always on — no config needed. The deprecated `features.inline-agents` field and the unsupported `inline-sub-agents: false` field both fail compilation; remove them.

---

## Syntax

Define a sub-agent with a level-2 Markdown heading of the form `## agent: \`name\``:

```markdown
## agent: `file-summarizer`
---
description: Summarizes the content of a file in a few concise sentences
model: small
---
You are a file summarization assistant. When given a file path, read the
file and return a brief summary (2–4 sentences) describing its purpose
and key contents. Be concise and factual.
```

### Name rules

- Must be enclosed in backticks: `` `name` ``
- Lowercase only: `[a-z][a-z0-9_-]*`
- Examples: `` `planner` ``, `` `file-summarizer` ``, `` `code-reviewer` ``

### Block boundary

The block ends at the next `##` heading (any level-2 heading) or at EOF — no explicit end marker is needed. Place sub-agent blocks **at the bottom** of the file, after all main workflow content.

### Frontmatter fields

Only two fields are supported inside a sub-agent frontmatter block:

| Field | Required | Default | Notes |
|---|---|---|---|
| `description` | No | — | Human-readable summary of the sub-agent's role |
| `model` | No | `"inherited"` | Model override; `"inherited"` uses the parent workflow's model. Prefer model aliases (e.g. `small`, `large`) over specific model IDs for portability. |

**Prefer model aliases over model IDs.** Built-in aliases resolve to the best available model for each provider, so they continue to work as models are updated. Commonly used aliases for sub-agents:

| Alias | Resolves to | When to use |
|---|---|---|
| `small` | `mini` → haiku, gpt-5-mini, gpt-5-nano, gemini-flash | Cheap, fast tasks: extraction, classification, formatting |
| `large` | sonnet, gpt-5-pro, gpt-5, gemini-pro | Complex reasoning or synthesis tasks |
| `inherited` | Parent workflow model | Default — use when the sub-agent needs the same capability as the parent |

All other fields (`engine`, `tools`, `network`, etc.) are stripped at runtime with a warning. Sub-agents inherit the parent's engine, tool access, and network configuration.

---

## Engine-Specific Placement

Sub-agent files are written to the directory and with the extension each engine natively expects:

| Engine | Directory | Extension |
|---|---|---|
| Copilot (default) | `.agents/agents/` | `.agent.md` |
| Claude | `.claude/agents/` | `.md` |
| Codex | `.codex/agents/` | `.md` |
| Gemini | `.gemini/agents/` | `.md` |

The engine is detected at compile time from the `engine:` field and injected as `GH_AW_ENGINE_ID` into the interpolation step's environment.

---

## MCP Access in Sub-Agents

Sub-agents **do not have their own MCP servers** — they run within the parent's agent environment without independent tool config. To give them file system and shell access, enable on the parent workflow:

- **`cli-proxy: true`** — GitHub CLI proxy for authenticated `gh` calls. Recommended for any sub-agent that reads/writes repo content.
- **`tools.github.mode: gh-proxy`** — routes GitHub API calls through the gh proxy sidecar; required for private repos or the GitHub MCP toolset.

```yaml
---
engine: copilot
tools:
  github:
    mode: gh-proxy
  cli-proxy: true
---
```

---

## When to Use Sub-Agents

Sub-agents are most useful in two scenarios:

### 1 — Parallel specialised tasks with smaller models

Break a large workflow into parallel units of work, each handled by a small, cheap model, and then use the parent (large) model to reason over the aggregated results:

```markdown
# Investigate: Repository Health

## Step 1 — gather data

Use the `dependency-scanner` agent to list all outdated packages.
Use the `test-coverage` agent to summarise uncovered code paths.
Use the `secret-scanner` agent to check for leaked credentials.

## Step 2 — synthesise

Combine the three reports above into a prioritised action plan.
The top item must have a linked PR draft or issue.

## agent: `dependency-scanner`
---
description: Lists outdated npm/pip/go packages
model: small
---
Run the appropriate package-manager audit command and return a
machine-readable list of outdated packages with their current and
latest versions.

## agent: `test-coverage`
---
description: Summarises low-coverage code paths
model: small
---
Read the most recent test coverage report and list the top 5 files or
functions with coverage below 60 %. Include the file path and line range.

## agent: `secret-scanner`
---
description: Checks for potential credential leaks
model: small
---
Scan staged changes and recently modified files for patterns that
resemble API keys, tokens, or passwords. Report any findings with the
file name and approximate line number.
```

The parent model (e.g. Claude Sonnet or Copilot) orchestrates, while the sub-agents do the heavy lifting with a `small` model at lower cost.

### 2 — Reusable specialised helpers

Extract a repetitive sub-task (file summarisation, commit-message generation, code explanation) into a named sub-agent that the main prompt can call by name, keeping the main prompt concise.

---

## Full Example

```markdown
---
engine: copilot
tools:
  github:
    mode: gh-proxy
  cli-proxy: true
  bash:
    - "cat *"
    - "ls *"
---

# PR Review Assistant

1. Use the `diff-explainer` agent to produce a plain-English summary of the
   diff for PR #${{ github.event.pull_request.number }}.
2. Post the summary as a PR comment.

## agent: `diff-explainer`
---
description: Produces a plain-English summary of a pull request diff
model: small
---
You receive a unified diff. Describe each changed file in one sentence,
focusing on *what changed* and *why it matters*. Ignore formatting-only
changes. Return a bulleted list, one bullet per file.
```

---

## Limitations

- Sub-agents do not support `engine:`, `tools:`, `network:`, or `mcp-servers:` fields — those are stripped at runtime.
- Sub-agents cannot define their own safe-output jobs.
- Sub-agent blocks must appear in the main workflow file body; they are not resolved inside imported shared files.
