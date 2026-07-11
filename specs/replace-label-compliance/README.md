# Replace-Label Compliance Fixtures

This directory contains normative compliance fixtures for the
[`replace-label` safe-output type](../replace-label-spec.md) and the formal
predicate model used by the Go testify suite.

The fixtures are the ground truth for RL-001, RL-002, and RL-003 decision
behavior. The formal tests in `pkg/workflow/replace_label_formal_test.go` bind
these fixture scenarios to executable predicates and validate additional
schema, gating, set-computation, staged-mode, and error invariants.

## Fixture Files

| Filename | Scenario | Spec Coverage |
|---|---|---|
| `rl-001-glob-semantics.yaml` | Glob pattern matching for `allowed-add`, `allowed-remove`, and `blocked` follows gobwas/glob semantics | RL-001, T-RL-020–T-RL-023 |
| `rl-002-allowlist-enforcement.yaml` | Non-empty allowlists enforce matches while empty allowlists permit any non-blocked label | RL-002, T-RL-021b, T-RL-022b, T-RL-025 |
| `rl-003-blocklist-ordering.yaml` | Blocklist evaluation occurs before allowlist evaluation (security boundary) | RL-003, T-RL-023–T-RL-024 |

## Formal Model

- **Preconditions (`F*`)**: non-empty required labels, bounded label/repo lengths,
  count gate (`count < max`), and repository target restrictions.
- **Decision predicates (SMT-style)**:
  - `GlobSemantics(label, pattern[])`
  - `AllowlistPermits(label, allowed[])`
  - `BlocklistRejects(label, blocked[])`
  - `BlocklistBeforeAllowlist` (ordering safety property)
- **Postconditions**:
  - label set uses remove-then-add arithmetic with deduplication;
  - staged mode reports success with `staged=true` and no write side effects;
  - hard REST failures return `success=false` with non-nil error.

Evaluation order is modeled as: blocked check → allowlist check → gates
(required-labels/title-prefix) → staged/execute branch.

## Behavioral Coverage Map

| Predicate / Invariant | Test Function | Description |
|---|---|---|
| P1 — GlobSemantics | `TestFormalGlobSemantics` | Star and char-class glob matching from fixture rl-001 |
| P2 — AllowlistPermits | `TestFormalAllowlistEnforcement` | Empty vs non-empty allowlist, add and remove directions |
| P3 + P4 — BlocklistRejects + BlocklistBeforeAllowlist | `TestFormalBlocklistOrdering` | Blocklist takes priority over allowlist; symmetric for add/remove |
| P5 — SchemaRequiredFields | `TestFormalSchemaRequiredFields` | Missing/empty/too-long label_to_remove and label_to_add |
| P6 — RepoMaxLength | `TestFormalRepoMaxLength` | repo field ≤ 256 characters |
| P7 — CountGateExclusive | `TestFormalCountGate` | count < max allowed, count = max rejected, default max = 5 |
| P8 — LabelSetComputation | `TestFormalLabelSetComputation` | Correct set arithmetic; missing-remove proceeds |
| P9 — StagedNoWrite | `TestFormalStagedMode` | staged=true returns success+staged=true, no writes |
| P10 — SingleRESTCall | `TestFormalSingleRESTCall` | Verify exactly one PUT; no separate add/remove |
| P11 — BlocklistAppliesSymmetrically | `TestFormalBlocklistSymmetry` | blocked applies to both label_to_add and label_to_remove |
| P12 — RequiredLabelsGate | `TestFormalRequiredLabelsGate` | All required labels present → proceed; missing → skip |
| P13 — TitlePrefixGate | `TestFormalTitlePrefixGate` | Matching prefix → proceed; non-matching → skip |
| P14 — AddDeduplication | `TestFormalAddDeduplication` | label_to_add appears exactly once in output |
| P15 — HardErrorOnSetLabelsFail | `TestFormalHardErrorOnRESTFail` | REST failure yields success=false, non-nil error |
| Edge: Exact glob no-wildcard | `TestFormalGlobExactNoWildcard` | Exact pattern `bug` does not match `bug-fix` |
| Edge: Alias fields | `TestFormalItemNumberAliases` | issue_number/pr_number/pull_number resolve correctly |
| Edge: Cross-repo restriction | `TestFormalCrossRepoRestriction` | repo not in allowed-repos is rejected |

## Fixture Schema

Each fixture file is a YAML document with the following top-level keys:

```yaml
fixture_id: string          # Unique identifier referencing the RL requirement code
description: string         # Human-readable scenario description
spec_refs:                  # Normative requirements under test (RL codes and § references)
  - string
scenarios:
  - scenario_id: string     # Unique sub-scenario identifier
    description: string     # Sub-scenario description
    input:
      safe_output_config:   # replace-label safe-output configuration under test
        allowed-add: [...]
        allowed-remove: [...]
        blocked: [...]
      message:              # Simulated agent message
        label_to_add: string
        label_to_remove: string
    expected:
      decision: allow | deny   # Required outcome
      error_code: integer | null  # Expected error code on deny
      reason: string           # Expected denial reason substring (informative)
```

## Adding New Fixtures

1. Copy the most relevant existing fixture file.
2. Assign a new `fixture_id` matching the RL requirement code being tested.
3. Update `input.safe_output_config` and `input.message` to reflect the new scenario.
4. Set `expected` fields to match the required outcome.
5. Register the new fixture in the table above and reference it from §9 of
   `specs/replace-label-spec.md`.

## Related Test IDs

The following test IDs defined in the replace-label specification map to these fixtures:

| Test ID | Fixture | Description |
|---------|---------|-------------|
| T-RL-020 | `rl-001-glob-semantics.yaml` | Star glob matches label name substring |
| T-RL-021 | `rl-001-glob-semantics.yaml` | Exact pattern matches only exact name |
| T-RL-021b | `rl-002-allowlist-enforcement.yaml` | Non-empty allowed-add accepts a matching label |
| T-RL-022 | `rl-001-glob-semantics.yaml` | Character class pattern matches correctly |
| T-RL-022b | `rl-002-allowlist-enforcement.yaml` | Non-empty allowed-add rejects a non-matching label |
| T-RL-023 | `rl-001-glob-semantics.yaml`, `rl-003-blocklist-ordering.yaml` | Glob pattern rejects non-matching label; blocked label rejected even when allowed |
| T-RL-024 | `rl-003-blocklist-ordering.yaml` | Blocked label rejected even with wildcard allowed-add |
| T-RL-025 | `rl-002-allowlist-enforcement.yaml` | Empty allowed-remove permits any non-blocked label |

## Generated Test Suite

The formal compliance suite is implemented in:

- `pkg/workflow/replace_label_formal_test.go`

The suite runs fully in-process under Go test, reads the fixture YAML files in
this directory, and does not require a JavaScript runtime to validate the
formal predicates.
