# AWF Config Sources Compliance Fixtures

This directory contains conformance test IDs and fixture stubs for the `DriftRecord` entity
defined in [§6.5 of the AWF Config Canonical Sources Specification](../awf-config-sources-spec.md#65-driftrecord-entity-schema).

All automation and agents that produce or consume drift reports **MUST** use the `DriftRecord` schema
defined in §6.5 of the specification for structured drift output.

---

## DriftRecord Conformance Tests

The following test IDs cover the `DriftRecord` schema and its usage requirements from §6.5.

| Test ID | Requirement | Description |
|---------|-------------|-------------|
| T-DR-001 | §6.5.1 — required fields | `DriftRecord` MUST include `property_path`, `drift_category`, `suggested_action`, and `detected_at`; records missing any required field are invalid and MUST be rejected. |
| T-DR-002 | §6.5.1 — `drift_category` enum | `drift_category` MUST be one of `missing_in_ghaw`, `missing_in_schema`, or `spec_mismatch`; any other value is invalid. |
| T-DR-003 | §6.5.1 — `detected_at` format | `detected_at` MUST be a valid ISO 8601 UTC timestamp; non-conforming values MUST be rejected. |
| T-DR-004 | §6.5.1 — `suggested_action` non-empty | `suggested_action` MUST NOT be empty (`minLength: 1`); an empty string MUST be rejected. |
| T-DR-005 | §6.5.1 — no additional properties | `DriftRecord` objects MUST NOT include properties beyond the four required fields; additional properties MUST be rejected. |
| T-DR-006 | §6.5.3 — corrective PR trigger | When any `DriftRecord` in the output list has `drift_category` of `missing_in_ghaw` or `spec_mismatch`, the detecting automation MUST open a corrective PR (CR-05). |
| T-DR-007 | §6.5.3 — SLA escalation trigger | When CR-06 SLA window is exceeded and `DriftRecord` items with actionable categories are present, an escalation issue MUST be opened or updated. |
| T-DR-008 | §6.5.3 — corrective PR embeds records | The corrective PR description MUST embed the full `DriftRecord` list as JSON. |
| T-DR-009 | §6.5.3 — empty list is valid | An empty `DriftRecord` list (no drift detected) is a valid output and MUST NOT trigger corrective PR or escalation actions. |
| T-DR-010 | §6.2 Step 5 integration | The drift detection procedure Step 5 MUST produce a list of zero or more `DriftRecord` objects; the output format MUST be a JSON array conforming to the §6.5.1 schema. |

---

## Spec Reference

- **Specification**: `specs/awf-config-sources-spec.md`
- **Defining section**: §6.5 — DriftRecord Entity Schema
- **Related sections**: §6.2 (Drift Detection Procedure), §5 (Conformance Requirements CR-05, CR-06)

---

## Running Conformance Tests

Conformance tests that validate `DriftRecord` schema compliance will be located in (or added to):

```
pkg/workflow/awf_config_drift_test.go   — DriftRecord schema validation (T-DR-001 through T-DR-005)
```

To run related tests (once implemented):

```bash
go test -v -run "TestDriftRecord" ./pkg/workflow/
```

---

## Adding New Conformance Tests

1. Assign a new `T-DR-xxx` identifier (increment from the last used ID).
2. Add a row to the table above with the test ID, requirement reference (§ number), and description.
3. Implement the test in the conformance test file listed above.
4. Cross-reference the new test ID from the relevant subsection of `specs/awf-config-sources-spec.md`.
