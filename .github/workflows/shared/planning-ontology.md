# Planning Ontology & Artifact Schemas

> **Version:** 2.0 · **Owner:** Procedures · **Status:** Active
>
> **Decision Ratifications:**
> - `copilot-plan-workflow-ux.md` → **Ratified Option A + Option 3**: Explicit commands (`/squad plan accept`) under the `/squad` namespace; planning logic lives in `shared/` components imported by `squad.md`. This file IS the shared component.
> - `copilot-sdlc-workflows.md` → **Ratified Option B (phased)**: Ship optional composable SDLC workflows. The planning ontology is the first deliverable; `shared/implement.md` and `shared/review.md` follow as separate issues.

---

## 1. Ontology Table

| Concept | Purpose | GitHub Representation | `squad_artifact` | Schema |
|---------|---------|----------------------|------------------|--------|
| **Intent** | Define what we're building and why | Root issue body | (issue body itself) | — |
| **Research** | Gather evidence and context | Issue comment | `research` | 1 |
| **Triage** | Classify findings → work / decision / excluded | Issue comment | `triage` | 1 |
| **Program Plan** | Strategic decomposition into initiatives/epics | Issue comment | `program` | 1 |
| **Implementation Plan** | PR-sized tasks with deps and sizing | Issue comment | `implementation` | 1 |
| **Validation** | Verify postconditions before activation | Issue comment | `validation` | 1 |
| **Scope Acceptance** | Confirm program plan is approved | Issue comment | `scope-accepted` | 1 |
| **Impl Acceptance** | Confirm implementation plan is approved | Issue comment | `impl-accepted` | 1 |
| **Activation** | Create sub-issues and begin execution | Issue comment | `activated` | 1 |
| **Lifecycle State** | Running summary updated on each transition | Issue comment | `lifecycle-state` | 1 |

### Preconditions & Postconditions

| Artifact | Preconditions | Postconditions |
|----------|---------------|----------------|
| Research | Intent exists (issue body non-empty) | Evidence gathered; sources cited |
| Triage | Research comment exists | Each finding classified: work / decision / excluded |
| Program Plan | Triage comment exists | Initiatives and epics defined with user stories |
| Implementation Plan | Program plan exists | PR-sized tasks with dependencies, sizing, rollout order |
| Validation | Implementation plan exists | All postconditions checked; pass/fail reported |
| Scope Acceptance | Program plan exists; user approves | Program scope locked |
| Impl Acceptance | Implementation plan exists; validation passes | Impl plan locked |
| Activation | Impl acceptance exists | Sub-issues created; labels applied; lifecycle complete |

---

## 2. State Transition Table

```
idle → researching
  triggered_by: /squad research
  requires: intent (issue body)
  produces: squad_artifact=research

researching → triaging
  triggered_by: /squad triage
  requires: squad_artifact=research
  produces: squad_artifact=triage

triaging → program_planning
  triggered_by: /squad plan program
  requires: squad_artifact=triage
  produces: squad_artifact=program

program_planning → implementation_planning
  triggered_by: /squad plan implementation
  requires: squad_artifact=program
  produces: squad_artifact=implementation

implementation_planning → validating
  triggered_by: /squad plan validate
  requires: squad_artifact=implementation
  produces: squad_artifact=validation

validating → scope_accepted
  triggered_by: /squad plan accept scope
  requires: squad_artifact=program
  produces: squad_artifact=scope-accepted

scope_accepted → impl_accepted
  triggered_by: /squad plan accept implementation
  requires: squad_artifact=validation with human-readable RESULT: PASS
  produces: squad_artifact=impl-accepted

impl_accepted → activated
  triggered_by: /squad plan activate
  requires: squad_artifact=impl-accepted
  produces: squad_artifact=activated
```

**Idempotency rule:** Re-running any command updates the existing artifact comment (edit, not duplicate). The lifecycle state comment is always updated on every transition. Idempotency applies to research, triage, program plan, implementation plan, and validation artifacts. Acceptance and activation artifacts are immutable once written — re-running these commands is a no-op if matching structured data already exists.

**Concurrency:** Concurrent acceptance commands are serialized by the workflow's concurrency group. Duplicate artifacts are harmless — subsequent phases select the newest matching structured data.

**Revision:** `/squad plan revise <feedback>` may be issued from any planning state (program_planning, implementation_planning, or validating). It revises the most recent plan artifact and resets validation if present.

---

## 3. Artifact Schemas

Each artifact is a human-readable Markdown comment plus gh-aw safe-output `data`. The minimum envelope is:

```json
{
  "squad_artifact": "{artifact_kind}",
  "schema_version": "1",
  "origin_issue": 123,
  "phases": []
}
```

gh-aw requires every declared schema property. Use `phases: []` for non-phase artifacts and the accumulated phase numbers for phase-state artifacts. gh-aw appends the validated envelope as a `Structured data:` fenced JSON block. HTML comments are unsupported for Squad state because gh-aw removes them from compiled prompts and sanitized bodies.

### 3.1 Intent (Root Issue Body)

The issue body IS the intent. No special format required, but structured intents improve output:

```markdown
## Goal
<What we're building and why>

## Success Criteria
- <Measurable outcome 1>
- <Measurable outcome 2>

## Constraints
- <Technical or organizational constraints>

## Context
<Links, prior art, relevant decisions>
```

### 3.2 Research Findings

```markdown
## Research Findings

### Summary
<1-3 sentence overview of what was discovered>

### Sources
| # | Source | Type | Key Insight |
|---|--------|------|-------------|
| 1 | <link/file/doc> | <codebase/docs/external> | <insight> |

### Findings
#### Finding 1: <title>
<Evidence and analysis>

#### Finding 2: <title>
<Evidence and analysis>

### Open Questions
- <Unresolved question needing human input>

### Recommendations
- <Actionable recommendation derived from evidence>
```

### 3.3 Triage Disposition

```markdown
## Triage Disposition

### Work Items (→ planning)
| # | Item | Source | Rationale |
|---|------|--------|-----------|
| 1 | <work description> | Finding N | <why this is work> |

### Decisions Needed (→ decision gate)
| # | Decision | Context | Options |
|---|----------|---------|---------|
| 1 | <decision question> | Finding N | <A, B, C> |

### Excluded (→ out of scope)
| # | Item | Reason |
|---|------|--------|
| 1 | <excluded item> | <why excluded> |

### Triage Summary
- **Work items:** N
- **Decisions pending:** N
- **Excluded:** N
```

### 3.4 Program Plan

```markdown
## Program Plan

### Initiatives
#### Initiative 1: <name>
- **Goal:** <what this achieves>
- **Epics:**
  - Epic 1.1: <title> — <scope summary>
  - Epic 1.2: <title> — <scope summary>

### Milestone Map
| Milestone | Epics | Target |
|-----------|-------|--------|
| M1: <name> | 1.1, 1.2 | <relative ordering> |

### User Stories
| Epic | Story | Acceptance Criteria |
|------|-------|--------------------|
| 1.1 | As a <who>, I want <what>, so that <why> | <criteria> |

### Dependencies
- Epic 1.2 depends on Epic 1.1
- <other dependency>

### Scope Boundary
- **In scope:** <explicit inclusions>
- **Out of scope:** <explicit exclusions>
```

### 3.5 Implementation Plan

```markdown
## Implementation Plan

### Tasks
| # | Title | Epic | Size | Depends On | Agent |
|---|-------|------|------|------------|-------|
| 1 | <PR-sized task title> | 1.1 | S | — | <squad member> |
| 2 | <PR-sized task title> | 1.1 | M | 1 | <squad member> |

**Sizing key:** XS (<1h) · S (1-3h) · M (3-8h) · L (1-2d) · XL (2-5d)

### Rollout Order
1. **Phase 1 (foundation):** Tasks 1, 2
2. **Phase 2 (features):** Tasks 3, 4, 5
3. **Phase 3 (polish):** Tasks 6, 7

### Risk Register
| Risk | Mitigation | Impact |
|------|-----------|--------|
| <risk> | <mitigation strategy> | <H/M/L> |

### GitHub Mapping
- Issues: one per task row
- Dependencies: GitHub sub-issue relationships
- Milestones: one per rollout phase
- Labels: per `size_representation` policy (default: body; when `label`: `size:{t-shirt}`) + `squad` + agent label
```

### 3.6 Validation Result

```markdown
## Plan Validation

### Result: ✅ PASS | ❌ FAIL

### Checks
| Check | Status | Detail |
|-------|--------|--------|
| All tasks map to an epic | ✅ | — |
| No circular dependencies | ✅ | — |
| Sizing within bounds (no >XL) | ✅ | — |
| Agent assignments valid | ✅ | — |
| Scope boundary respected | ✅ | — |
| <check> | ❌ | <diagnostic> |

### Diagnostics (if FAIL)
- <Specific issue and suggested fix>
```

### 3.7 Acceptance Records

**Scope Acceptance:**
```markdown
## Scope Accepted

- **Program plan version:** <comment link>
- **Accepted by:** <user>
- **Date:** <ISO date>
- **Notes:** <optional remarks>
```

**Implementation Acceptance:**
```markdown
## Implementation Accepted

- **Implementation plan version:** <comment link>
- **Validation result:** ✅ PASS (<comment link>)
- **Accepted by:** <user>
- **Date:** <ISO date>
- **Notes:** <optional remarks>
```

**Activation Record:**
```markdown
## Execution Activated

- **Issues created:** N
- **Milestone(s):** <milestone links>
- **Assigned agents:** <list>
- **Created issues:**
  | # | Title | Issue | Size | Agent |
  |---|-------|-------|------|-------|
  | 1 | <title> | #NNN | S | <agent> |
```

---

## 4. Structured Artifact Registry

| `squad_artifact` | Artifact | `phases` Data | Cardinality | Update Behavior |
|------------------|----------|---------------|-------------|-----------------|
| `research` | Research findings | `[]` | 1 current per issue | Replace body |
| `plan` | Fast-path plan | `[]` | 1 current per issue | Replace body |
| `plan-accepted` | Fast-path full acceptance | `[]` | 1 total | Immutable |
| `phases-accepted` | Fast-path accepted phase state | Accumulated accepted phases | 1 current per issue | Replace accumulated state |
| `triage` | Triage disposition | `[]` | 1 current per issue | Replace body |
| `program` | Program plan | `[]` | 1 current per issue | Replace body |
| `implementation` | Implementation plan | `[]` | 1 current per issue | Replace body |
| `validation` | Validation result | `[]`; body has `RESULT: PASS` or `FAIL` | 1 current per issue | Replace body |
| `scope-accepted` | Scope acceptance | `[]` | 1 total | Immutable |
| `impl-accepted` | Full implementation acceptance | `[]` | 1 total | Immutable |
| `impl-phases-accepted` | Accepted implementation phase state | Accumulated accepted phases | 1 current per issue | Replace accumulated state |
| `phases-activated` | Activated phase state | Accumulated activated phases | 1 current per issue | Replace accumulated state |
| `activated` | Terminal activation record | All phases when phased; otherwise `[]` | 1 total | Immutable |
| `lifecycle-state` | Lifecycle summary | `[]` | 1 current per issue | Updated every transition |

Every entry also requires `schema_version: "1"` and the triggering `origin_issue`.

**Version bump rule:** When this data contract changes incompatibly, increment `schema_version`. Consumers may continue recognizing earlier structured schemas during a migration window.

---

## 5. Lifecycle Summary Format

This comment is created on first transition and updated on every subsequent transition:

```markdown
## Planning Lifecycle

| Phase | Status | Artifact | Updated |
|-------|--------|----------|---------|
| Intent | ✅ Done | (issue body) | <date> |
| Research | ✅ Done | <comment link> | <date> |
| Triage | ✅ Done | <comment link> | <date> |
| Program Plan | ⬚ Pending | — | — |
| Implementation Plan | ⬚ Pending | — | — |
| Validation | ⬚ Pending | — | — |
| Scope Accepted | ⬚ Pending | — | — |
| Impl Accepted | ⬚ Pending | — | — |
| Activated | ⬚ Pending | — | — |

**Current state:** Triaged
**Last command:** `/squad triage` by @user at <timestamp>
**Next action:** `/squad plan program` — create a program plan from triage dispositions
**Also available:** `/squad triage revise <feedback>` — adjust triage before planning
```

Status icons: `✅ Done` · `⏳ In Progress` · `⬚ Pending` · `❌ Failed` · `⏭ Skipped`

---

## 6. Backward Compatibility

### Fast-Path Mapping

The original `/squad plan` and `/squad plan accept` commands remain fully supported as **fast paths** that combine multiple lifecycle phases:

| Legacy Command | Equivalent Phases | Artifact Data Produced |
|---------------|-------------------|------------------------|
| `/squad plan` | program + implementation (combined) | `squad_artifact=plan` |
| `/squad plan accept` | scope + impl + activate (combined) | See note below |
| `/squad plan revise` | revise (same as granular) | Updates `squad_artifact=plan` |
| `/squad research` | research (unchanged) | `squad_artifact=research` |

> **`/squad plan accept` behavior:** When only a fast-path `plan` artifact exists, accept produces `plan-accepted` or `phases-accepted`. When granular artifacts exist, accept runs the granular accept/activate sequence, producing `scope-accepted`, `impl-accepted`, and `activated` artifacts instead.

### Coexistence Rules

1. Fast-path artifact kinds (`plan`, `plan-accepted`, `phases-accepted`) and granular artifact kinds (`program`, `implementation`, etc.) are **independent namespaces** — they do not interfere.
2. An issue may use EITHER the fast path OR the granular path, not both simultaneously.
3. If a granular artifact exists and a user runs `/squad plan` (fast path), the system warns that granular planning is in progress and asks for confirmation.
4. The lifecycle state comment tracks whichever path is active.

Legacy Squad HTML markers are not scanned. gh-aw removes HTML comments during prompt compilation and safe-output sanitization; durable state uses the structured data registry above.

---

## 7. Command Surface

| Command | Mode | Phase | `squad_artifact` Output |
|---------|------|-------|-------------------------|
| `/squad research` | Research | researching | `research` |
| `/squad triage` | Triage | triaging | `triage` |
| `/squad plan program` | Program Planning | program_planning | `program` |
| `/squad plan implementation` | Impl Planning | implementation_planning | `implementation` |
| `/squad plan validate` | Validation | validating | `validation` |
| `/squad plan accept scope` | Scope Acceptance | scope_accepted | `scope-accepted` |
| `/squad plan accept implementation` | Impl Acceptance | impl_accepted | `impl-accepted` or `impl-phases-accepted` |
| `/squad plan activate` | Activation | activated | `activated` or `phases-activated` |
| `/squad plan revise <feedback>` | Revision | (current phase) | Updates latest plan artifact |
| `/squad plan` | Fast path (plan) | — | `plan` |
| `/squad plan accept` | Fast path (accept) | — | `plan-accepted` or `phases-accepted` |

### Execution Model Support

All commands work identically under both execution models:
- **Persistent Squad** (long-running cast): Agents referenced by name in assignments.
- **Ephemeral Squad** (single-shot): Agent assignments become labels; cast is performed at activation time if none exists.

---

## Design Principles

1. **Research produces evidence, not backlog** — findings inform triage, not issue creation.
2. **Triage is a human-in-the-loop gate** — nothing becomes work without classification.
3. **Program ≠ Implementation** — strategic scope and tactical tasks are separate products.
4. **GitHub-native hierarchy** — sub-issues, milestones, and dependencies over custom labels.
5. **Relative sizing** — XS/S/M/L/XL signals effort, not hours.
6. **Postcondition validation** — acceptance checks structure before creating real artifacts.
7. **Idempotent commands** — re-running updates rather than duplicates.
8. **Path independence** — fast path and granular path coexist without conflict.
