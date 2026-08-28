---
description: Guide for adding BinEval-style binary evaluations to agentic workflows — syntax, question decomposition methodology, result storage, and anti-patterns.
---

# BinEval Evaluations in Agentic Workflows

Each eval is a binary YES/NO question answered by an LLM judge reading the agent's output. Results go to an `evals.jsonl` artifact and a dedicated git branch for historical comparison.

---

## How Evals Work

Evals run on the agent output from `safe-outputs`, answering each declared question YES or NO. Results are redacted, uploaded as the `evals.jsonl` artifact, and committed to the `evals/<workflow-id>` branch.

---

## Basic Syntax

### Shorthand — plain list

> **Prerequisite:** `agent_output.json` is only included in the agent artifact when `safe-outputs` is also declared. Without it, the evals job runs with no agent context and every question will receive `UNKNOWN`.

```yaml
---
on:
  issues:
    types: [opened]
engine: copilot
safe-outputs:
  add-comment:
evals:
  - id: response_provided
    question: Does the agent output confirm that a response was written?
  - id: no_unrelated_files
    question: Does the agent output show that only the expected files were modified?
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
      question: Does the agent output show that only the expected files were modified?
  model: small         # model for all questions
  runs-on: ubuntu-latest
```

**Fields:**

- `questions:` — list of question objects (required in extended form, ≥ 1 entry).
- `model:` — LLM model for all questions. Use a model alias (`small`, `gpt-4o`) or a full model ID. Defaults to the engine's detection model (typically a small, cost-effective model).
- `runs-on:` — optional runner override for the evals job. Defaults to `ubuntu-latest` when omitted.

---

## Decomposing a Task into Binary Questions

BinEval questions must be answerable with a strict YES or NO by an LLM reading the agent's output alone. Follow this process:

### 1 — State the goal

One sentence describing a successful run:

> "The agent should update the CHANGELOG and bump the version number without touching unrelated files."

### 2 — Identify observable properties

Break the goal into properties a judge can verify from `agent_output.json`:

| Property | Observable signal |
|---|---|
| CHANGELOG updated | Agent output mentions or contains CHANGELOG edits |
| Version bumped | A version number appears changed in the diff or agent summary |
| No unrelated files changed | Agent output does not list changes outside CHANGELOG and version files |

### 3 — Write falsifiable YES/NO questions

One property per question, YES when the property holds, referencing observable evidence in the agent output — not intent or effort.

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

Prefer `model: small` (the default) for factual checks. Set `model` at the `evals:` level for reasoning-heavy questions:

```yaml
evals:
  questions:
    - id: changelog_updated
      question: Does the agent output confirm that CHANGELOG was updated?
    - id: design_sound
      question: Is the agent's proposed design consistent with established patterns described in the agent output?
  model: gpt-4o   # nuanced questions; override default small model
```

### Intent-derived scenarios

When a workflow has an `intent:`, load [intent.md](intent.md) during design. Derive positive and adversarial scenario fixtures from its required effects and inverse/no-op conditions. BinEval evaluates one fixture against one `agent_output.json`, so do not put mutually exclusive scenarios into one unconditional question list. Write a scenario-specific question about an observable result per fixture:

```yaml
evals:
  - id: actionable_case
    question: Does the agent output show that the novel, actionable case received the configured visible result?
  - id: duplicate_noop
    question: Does the agent output show that the already-tracked case produced no visible write action?
  - id: uncertainty_noop
    question: Does the agent output show that insufficient evidence produced no visible write action?
```

If a question is shared across fixtures, state how the provided scenario makes it applicable and return `UNKNOWN` when that scenario was not provided; do not treat missing scenario evidence as `NO`.

### Good question checklist

- ✅ Uses YES = success convention consistently.
- ✅ Avoids subjective terms ("good", "well-written") unless the question explicitly bounds them ("according to the coding style guide").

---

## Result Storage

### Artifact

Each run uploads `evals.jsonl` as the `evals` artifact. Each line is a JSON object:

```json
{"id":"compiles","question":"Does the generated code compile?","answer":"YES","model":"small","timestamp":"2026-07-15T10:00:00Z","runid":"12345678"}
```

### Git branch

Results are also committed to `evals/<sanitized-workflow-id>` by the `push_evals_state` job (requires `contents: write`), enabling comparison after artifact expiry.

Read results with:

```bash
gh aw audit <run-id> --artifacts evals    # downloads evals.jsonl from the run artifact
gh aw logs <workflow-name> --evals        # filter to runs that contain evals results
```

---

## Required Permissions

All of these are added automatically when `evals:` is declared — no manual configuration needed.

The compiler grants `contents: read` by default, and conditionally adds:

- `copilot-requests: write` — when the workflow uses Copilot API requests.
- `id-token: write` — when the workflow uses GitHub OIDC authentication or OTLP telemetry.

The `push_evals_state` job that persists results to a git branch always gets `contents: write`.

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
  contents: read
  issues: read
tools:
  github:
    toolsets: [issues]
safe-outputs:
  add-labels:
    allowed: [bug, enhancement, question, needs-triage]
evals:
  - id: label_requested
    question: Does the agent output show that at least one label was requested via a safe-output action?
  - id: label_in_allowed_set
    question: Does the agent output show that the requested label belongs to the allowed set (bug, enhancement, question, needs-triage)?
  - id: no_extra_labels
    question: Does the agent output show that no more than two labels were requested?
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
