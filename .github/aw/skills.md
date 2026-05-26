---
description: Guide for leveraging skills (SKILL.md files) in agentic workflows — hint, fusion, and inline fusion strategies
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

## Inline Skills (Fusion at Authoring Time)

**Use when**: You want to keep the main prompt compact while still shipping task-specific skill guidance with the workflow.

Inline skills let a workflow embed a complete skill or a partial skill fragment under `## skill: \`name\``.
Extraction happens in the setup/interpolation runtime step of workflow execution, not at `.md` to `.lock.yml` compile time.
gh-aw writes each block into engine-specific skill locations and removes those blocks from the main prompt body.
This keeps the main prompt slim and flexible while still making the fused guidance available as skills.

Use this to fuse:

- A full skill when the workflow needs a self-contained capability.
- Partial skill sections when only targeted guidance is needed.

**Pattern**:

```markdown
on:
  workflow_dispatch:
engine: copilot
---

Triage the issue and propose next steps.

## skill: `issue-triage`
---
description: Classify issues and suggest next actions.
---
Classify by bug / feature / question, identify missing information, and suggest
the smallest actionable next step.
```

Use a unique inline skill name per workflow file. The name can be arbitrary, but it must start with a lowercase letter and then use only lowercase letters, digits, `_`, or `-`.
These constraints keep extracted skill paths predictable and engine-compatible.
Avoid naming collisions with repository file-based skills (for example `.github/skills/<name>/SKILL.md`), because inline extraction writes to the same engine skill paths.

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
