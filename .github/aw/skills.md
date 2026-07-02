---
description: Guide for leveraging skills (SKILL.md files) in agentic workflows — hint, fusion, and inline fusion strategies
---

# Skills in Agentic Workflows

Use skills — domain-specific knowledge files (`SKILL.md`) under `skills/` or `.github/skills/` — in workflows.

---

## Detecting Skills

```bash
find "${GITHUB_WORKSPACE}" -name "SKILL.md" -maxdepth 6
```

List available skills before choosing a strategy.

---

## Frontmatter `skills:` (SHA-Pinned Installs)

Declare external skills to install at activation time with the top-level `skills:` array. At activation, `gh aw` installs each via `gh skill install` and wires it into the engine — no manual vendoring.

```yaml
skills:
  - owner/repo@<40-char-sha>              # repository-wide install
  - owner/repo/skill/path@<40-char-sha>   # path-scoped install
  - skill: owner/repo@<40-char-sha>       # object form: per-skill auth
    github-token: ${{ secrets.SKILLS_TOKEN }}
```

- Static references must be pinned to a full 40-character lowercase commit SHA; `${{ ... }}` expressions are allowed in the ref position and resolved at runtime.
- Object entries set per-skill auth via `github-token` or `github-app`.

This is distinct from the prompt-side strategies below (hint / fusion / inline), which shape skill *content* into the prompt rather than installing skill packages.

---

## Strategy 0 — Agent Finder (Discovery First)

**Use when**: the relevant skill is not obvious, the repository may not contain the right skill yet, or you want to discover installable skills before loading local ones.

Query **GitHub Agent Finder** through its REST API (ARD search shape: `query.text`; add `query.filter` to narrow by resource type):

```bash
curl -s https://agentfinder.github.com/api/v1/search \
  -H 'Content-Type: application/json' \
  -d '{"query":{"text":"<the user task, in plain language>"},"pageSize":10}'
```

To search specifically for skills, add a type filter:

```bash
curl -s https://agentfinder.github.com/api/v1/search \
  -H 'Content-Type: application/json' \
  -d '{"query":{"text":"<the user task, in plain language>","filter":{"type":["application/ai-skill"]}},"pageSize":10}'
```

After discovery:

- Prefer repository-local skills when they satisfy the task.
- If you use a discovered skill, extract only the specific guidance you need.
- Do not load or paste entire skills when a fragment is enough.
- Do not install or enable returned resources automatically; that requires explicit user choice.

Agent Finder helps locate candidates; **skill fusion** keeps the final prompt small.

---

## Inline Skills (Fusion at Authoring Time)

**Use when**: keeping the main prompt compact while shipping task-specific skill guidance with the workflow.

Inline skills embed a complete skill or fragment under `## skill: \`name\``. Extraction runs in the setup/interpolation step (not at compile time): gh-aw writes each block to engine-specific skill locations and removes it from the main prompt body. Fuse a full skill for a self-contained capability, or partial sections when only targeted guidance is needed.

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

Use a unique inline skill name per workflow file. Name must start with a lowercase letter, then lowercase letters, digits, `_`, or `-`. Avoid collisions with file-based skills under `.github/skills/<name>/SKILL.md` — inline extraction writes to the same paths.

---

## Strategy 1 — Hint (Generalist)

**Use when**: the task strategy is unknown at authoring time, or the agent must adapt to whatever skills are available.

The prompt tells the agent skills exist and to discover/apply the relevant ones itself.

**Pattern**:

```markdown
If the repository contains `SKILL.md` files under `skills/`, check which ones are
relevant to this task. For each relevant skill, read its content and apply the
guidance it provides.
```

---

## Strategy 2 — Fusion (Ultra-Cognitive)

**Use when**: you know exactly which skill (or part of it) is needed and want minimal context overhead.

Inline **only the specific sections** of the skill the agent needs. Do not paste the entire SKILL.md.

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

Prefer fusion when the task domain and required skill sections are known; it never loads entire skills.

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
