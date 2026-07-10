# Intent Attribution Compliance Fixtures

This directory defines the minimum compliance scenarios for the
[Intent Attribution & Agent Governance Specification](../intent-attribution-agent-governance.md).

## Required scenarios

| Scenario | Expected result | Spec references |
|---|---|---|
| Explicit intent wins over linked issues | Attribution uses explicit metadata as the sole source | §RFC 2119 Norms → Attribution-Resolution Order |
| Ambiguous root issue set | Attribution is `status: "ambiguous"` with `source: "closing_issue"` | §RFC 2119 Norms → Ambiguous-Root Handling |
| Unlinked pull request fails closed | Governance resolves to the safest policy (`propose_only`, no writes, approval required) | §RFC 2119 Norms → Fail-Closed Behavior |

## Fixture guidance

Each future fixture in this directory SHOULD record:

- the input artifact shape (explicit metadata, linked issues, labels)
- the expected attribution source and status
- the expected compiled execution policy when attribution is missing or ambiguous

The minimum fixture set for conformance claims is:

1. `explicit-intent-wins`
2. `ambiguous-root-closing-issues`
3. `unlinked-pr-fail-closed`
