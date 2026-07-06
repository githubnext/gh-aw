# Replace-Label Compliance Fixtures

This directory contains compliance fixtures for the normative requirements of the
[Replace-Label Specification](../replace-label-spec.md).

Each fixture describes test scenarios with input configurations and the expected
access-control decisions. Fixtures cover the glob-matching and blocklist-ordering
requirements defined in §4 of the specification.

## Fixture Files

| Filename | Scenario | Spec Coverage |
|---|---|---|
| `rl-001-glob-semantics.yaml` | Glob pattern matching for `allowed-add`, `allowed-remove`, and `blocked` follows gobwas/glob semantics | RL-001, T-RL-020–T-RL-023 |
| `rl-003-blocklist-ordering.yaml` | Blocklist evaluation occurs before allowlist evaluation (security boundary) | RL-003, T-RL-023–T-RL-024 |

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
| T-RL-022 | `rl-001-glob-semantics.yaml` | Character class pattern matches correctly |
| T-RL-023 | `rl-001-glob-semantics.yaml`, `rl-003-blocklist-ordering.yaml` | Glob pattern rejects non-matching label; blocked label rejected even when allowed |
| T-RL-024 | `rl-003-blocklist-ordering.yaml` | Blocked label rejected even with wildcard allowed-add |
