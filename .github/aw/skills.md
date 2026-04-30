---
description: Guide for leveraging skills (SKILL.md files) in agentic workflows — hint strategy and fusion strategy
---

# Skills in Agentic Workflows

Consult this file when you want a workflow to take advantage of skills — domain-specific knowledge files (`SKILL.md`) that live in the repository under `skills/` or `.github/skills/`.

---

## Detecting Skills

At runtime, find skill files with:

```bash
find "${GITHUB_WORKSPACE}" -name "SKILL.md" -maxdepth 6
```

List available skills and their locations before deciding which strategy to apply.

---

## Strategy 1 — Hint (Generalist)

**Use when**: The task strategy is not fully known at authoring time, or when the agent must adapt to whatever skills are available.

The workflow prompt hints that skills exist and asks the agent to discover and apply the relevant ones itself. The agent decides which skill files to read and how much of each to use.

**Pattern**:

```markdown
If the repository contains `SKILL.md` files under `skills/`, check which ones are
relevant to this task. For each relevant skill, read its content and apply the
guidance it provides.
```

**When to prefer this**:
- The task domain is broad or open-ended
- You don't know at authoring time which skills will exist
- You want the agent to self-select relevant skills from a growing library
- The overhead of reading an extra file is acceptable

**Tradeoff**: The agent may read and summarise more skill content than needed, which costs context tokens and can dilute focus.

---

## Strategy 2 — Fusion (Ultra-Cognitive)

**Use when**: You know exactly which skill (or which part of a skill) is needed, and you want to minimise context overhead.

Extract and inline **only the specific sections** of the skill content that the agent needs. Do not paste the entire SKILL.md; identify the minimal fragment, then remix it into the workflow prompt so the agent receives precise, surgical guidance without loading the full file.

**Pattern**:

```markdown
<!-- gh-skill-fusion: skills/github-mcp-server/SKILL.md#authentication -->

When calling GitHub MCP tools, use the pre-configured token already injected into the
environment. Never prompt the user for credentials.
```

**When to prefer this**:
- You know exactly which skill section applies
- Context budget is limited (long prompts or many tools already registered)
- The workflow runs frequently and latency matters
- You want deterministic, stable behaviour without dynamic skill discovery

**Tradeoff**: Requires manual authoring effort. If the upstream skill changes, the fused fragment must be updated by hand.

---

## Choosing Between the Two Strategies

| Factor | Hint | Fusion |
|---|---|---|
| **Task domain** | Broad / unknown | Narrow / well-defined |
| **Skill set** | Grows dynamically | Known and stable |
| **Context budget** | Generous | Tight |
| **Maintenance burden** | Low (agent self-selects) | Higher (manual sync with source) |
| **Determinism** | Lower (agent chooses) | Higher (exact fragment) |
| **Scale** | Poor (entire skills loaded) | Good (minimal content) |

Fusion scales better because entire skills are never loaded. Prefer fusion when you know the task domain and the specific skill sections required.

---

## Example: Hint Strategy

```markdown
---
on:
  issues:
    types: [opened]
engine: copilot
tools:
  github:
    toolsets: [issues]
permissions:
  issues: write
---

Triage the newly opened issue.

If there are relevant skills under `skills/`, read them and apply their guidance.
Focus on skills related to issue classification or project conventions.
```

---

## Example: Fusion Strategy

```markdown
---
on:
  pull_request:
    types: [opened, synchronize]
engine: copilot
tools:
  github:
    toolsets: [pull_requests]
permissions:
  pull-requests: write
---

Review the pull request for adherence to project conventions.

<!-- Fused from skills/developer/SKILL.md#code-organization -->
Prefer many smaller files grouped by functionality. Add new files for new features
rather than extending existing ones. Keep validators under 300 lines; split when
a single file covers more than one domain.
<!-- End fusion -->

Report findings as inline review comments.
```

---

## Anti-Patterns

- ❌ **Do not load entire skill files** when only one section is relevant — use fusion instead
- ❌ **Do not hint without bounds** — if using the hint strategy, constrain the agent with a `maxdepth` and a relevance filter to avoid reading every SKILL.md in a large repo
- ❌ **Do not paste skills verbatim** without adapting them to the workflow's context — fused fragments should read as natural prose, not as lifted documentation
- ❌ **Do not hard-code skill file paths** in hints — use `find` so the prompt still works when skills are reorganised
