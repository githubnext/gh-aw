---
description: Guide for adding BinEval-style binary evaluations to agentic workflows — syntax, question decomposition methodology, result storage, and anti-patterns.
---

# BinEval Evaluations in Agentic Workflows

Evals let you verify automatically whether an agentic run met its goals. Each evaluation is a binary YES/NO question answered by an LLM judge that reads the agent's output. Results are stored as `evals.jsonl` artifacts and persisted to a dedicated git branch for historical comparison.

---

## How Evals Work

Per run:

1. **Setup** — the evals job downloads the agent artifact (`agent_output.json`, `prompt.txt`) and writes a BinEval prompt containing all declared questions.
2. **Execute** — an LLM judge runs in a network-restricted sandbox (same engine as the agent job) and answers each question with YES or NO.
3. **Parse** — raw engine output is parsed into per-question records and written to `evals.jsonl`.
4. **Redact** — any credential patterns are removed from the results before upload.
5. **Upload** — `evals.jsonl` is uploaded as the `evals` workflow artifact and optionally persisted to the `evals/<workflow-id>` git branch.

The evals job runs **after** the agent job and **in parallel with** `safe_outputs`, so it does not block the write path.

---

## Basic Syntax

### Shorthand — plain list

```yaml
---
on:
  issues:
    types: [opened]
engine: copilot
evals:
  - id: scoped_change
    question: Is the implementation limited to the scope described in the issue?
  - id: no_regressions
    question: Does the change avoid modifying files unrelated to the task?
---

Implement the requested change described in ${{ github.event.issue.body }}.
```

Each entry requires:

- `id` — unique identifier for the question (used as the key in `evals.jsonl`). Must be a non-empty string; no duplicates allowed.
- `question` — the binary question the LLM judge will answer YES or NO.

### Extended form — with model and runs-on overrides

```yaml
evals:
  questions:
    - id: compiles
      question: Does the generated code compile without errors?
    - id: tests_pass
      question: Do all existing tests still pass according to the agent output?
    - id: scoped_change
      question: Is the implementation limited to the scope described in the issue?
      model: gpt-4o   # per-question model override
  model: small         # default model for all questions
  runs-on: ubuntu-latest
```

**Fields:**

- `questions:` — list of question objects (required in extended form, ≥ 1 entry).
- `model:` — default LLM model for all questions. Use a model alias (`small`, `gpt-4o`) or a full model ID. Defaults to the engine's detection model (typically a small, cost-effective model).
- `runs-on:` — optional runner override for the evals job. Inherits the workflow default when omitted.

Each question object may include its own `model:` field to override the top-level default for that question only.

---

## Decomposing a Task into Binary Questions

BinEval questions must be answerable with a strict YES or NO by an LLM reading the agent's output alone. Follow this process:

### 1 — State the goal

Write one sentence describing what a successful run looks like.

> "The agent should update the CHANGELOG and bump the version number without touching unrelated files."

### 2 — Identify observable properties

Break the goal into properties that a judge can verify from `agent_output.json` and `prompt.txt`:

| Property | Observable signal |
|---|---|
| CHANGELOG updated | Agent output mentions or contains CHANGELOG edits |
| Version bumped | A version number appears changed in the diff or agent summary |
| No unrelated files changed | Agent output does not list changes outside CHANGELOG and version files |

### 3 — Write falsifiable YES/NO questions

Each question should:

- Be answerable YES when the property holds, NO otherwise.
- Reference observable evidence in the agent output — not intent or effort.
- Cover exactly one property (no compound questions with "and" or "or").

```yaml
evals:
  - id: changelog_updated
    question: Does the agent output confirm that CHANGELOG was updated?
  - id: version_bumped
    question: Does the agent output confirm that the version number was incremented?
  - id: no_unrelated_files
    question: Does the agent output show that only CHANGELOG and version files were modified?
```

### 4 — Assign question cost

Prefer `model: small` (the default) for factual YES/NO checks. Reserve a larger model for questions that require nuanced reasoning:

```yaml
evals:
  questions:
    - id: changelog_updated
      question: Does the agent output confirm that CHANGELOG was updated?
    - id: design_sound
      question: Is the agent's proposed design consistent with established patterns in the codebase?
      model: gpt-4o   # nuanced, benefits from a larger model
  model: small
```

### Good question checklist

- ✅ Answerable from the agent output alone — no external calls needed.
- ✅ Exactly one binary claim per question.
- ✅ Uses YES = success convention consistently.
- ✅ Avoids subjective terms ("good", "well-written") unless the question explicitly bounds them ("according to the coding style guide").
- ✅ Each question has a unique `id`.

---

## Result Storage

### Artifact

Each run uploads `evals.jsonl` as the `evals` artifact (30-day retention). Each line is a JSON object:

```json
{"run_id":"12345678","workflow_id":"my-workflow","id":"compiles","question":"Does the generated code compile?","answer":"YES","timestamp":"2026-07-15T10:00:00Z"}
```

### Git branch

Results are also committed to `evals/<sanitized-workflow-id>` by the `push_evals_state` job (requires `contents: write`). This enables historical comparison across runs even after artifact expiry.

Read results with:

```bash
gh aw audit <run-id>          # includes evals section when present
gh aw logs <workflow-name>    # aggregates evals.jsonl across recent runs
```

---

## Required Permissions

The evals job itself reads artifacts and runs the engine — no extra permissions beyond `contents: read`. The `push_evals_state` job that persists results to a git branch needs:

```yaml
permissions:
  contents: write
```

This is added automatically when `evals:` is declared.

---

## Minimal Working Example

```markdown
---
description: Triage new issues and apply labels
on:
  issues:
    types: [opened]
engine: copilot
permissions:
  issues: write
tools:
  github:
    toolsets: [issues]
safe-outputs:
  add-label:
    allowed-labels: [bug, enhancement, question, needs-triage]
evals:
  - id: label_applied
    question: Did the agent apply at least one label to the issue?
  - id: correct_type
    question: Is the applied label appropriate for the issue content described in the prompt?
  - id: no_extra_labels
    question: Did the agent avoid applying more than two labels?
---

Read ${{ github.event.issue.title }} and ${{ github.event.issue.body }}.
Apply the most appropriate label(s) from the allowed set.
```

Compile and deploy:

```bash
gh aw compile issue-triage
```

---

## Anti-Patterns

- ❌ **Compound questions** — "Did the agent update CHANGELOG and bump the version?" splits into two questions. A single NO is ambiguous.
- ❌ **Unobservable questions** — "Did the agent try its best?" cannot be answered from output text.
- ❌ **Duplicate IDs** — `id` must be unique within a workflow; the compiler rejects duplicates.
- ❌ **Empty questions** — both `id` and `question` must be non-empty strings.
- ❌ **Using a frontier model for all questions** — factual checks are cheap on small models; save larger models for reasoning-heavy questions.
- ❌ **Removing `evals:` mid-experiment** — breaks historical trend comparisons stored in the `evals/<id>` branch.
- ❌ **Questions that require tool calls** — the evals engine runs in a network-restricted sandbox with only `bash`. Questions must be answerable from the downloaded agent artifact.
