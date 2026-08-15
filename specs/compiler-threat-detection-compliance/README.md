# Compiler Threat Detection Compliance Map

This directory maps the threat-detection rule catalog in the [Compiler Threat Detection Specification](../compiler-threat-detection-spec.md#8-compliance-testing) to its conformance test IDs. Each active `CTR-*` rule has one required `T-CTR-*` test ID.

| Rule ID | Test ID |
|---------|---------|
| CTR-001 | T-CTR-001 |
| CTR-002 | T-CTR-002 |
| CTR-003 | T-CTR-003 |
| CTR-004 | T-CTR-004 |
| CTR-005 | T-CTR-005 |
| CTR-006 | T-CTR-006 |
| CTR-007 | T-CTR-007 |
| CTR-008 | T-CTR-008 |
| CTR-009 | T-CTR-009 |
| CTR-010 | T-CTR-010 |
| CTR-011 | T-CTR-011 |
| CTR-012 | T-CTR-012 |
| CTR-013 | T-CTR-013 |
| CTR-014 | T-CTR-014 |
| CTR-015 | T-CTR-015 |
| CTR-016 | T-CTR-016 |
| CTR-017 | T-CTR-017 |
| CTR-018 | T-CTR-018 |
| CTR-019 | T-CTR-019 |
| CTR-020 | T-CTR-020 |
| CTR-021 | T-CTR-021 |
| CTR-022 | T-CTR-022 |
| CTR-023 | T-CTR-023 |

The test triggers, expected compiler actions, and stable diagnostics are defined in [Section 8.1](../compiler-threat-detection-spec.md#81-test-id-catalog). The implementation and concrete test-file mappings are defined in [Section 7.1](../compiler-threat-detection-spec.md#71-baseline-rule-mapping).

## Section 6.4 False-Positive Handling Norms

| Test ID | Norm |
|---------|------|
| T-CTR-024 | Suppression validation requires `rule` and a non-empty `reason`. |
| T-CTR-025 | Active suppressions retain `rule`, `reason`, and `expires` for audit. |
| T-CTR-026 | MUST-level suppressions have a 10-business-day resolution SLA. |
| T-CTR-027 | SLA breaches include `rule`, `reason`, `age_business_days`, `owner`, and `expires`. |
| T-CTR-028 | MUST-level suppressions older than 20 business days create a follow-up sync action. |
| T-CTR-029 | Expired suppressions are re-evaluated and treated as absent. |

## Section 6.6 Optimizer Failure Safeguards

| Test ID | Norm |
|---------|------|
| T-CTR-030 | API unavailability emits `OPTIMIZER_DEGRADED`. |
| T-CTR-031 | Degraded evaluation cannot mutate specifications or open a pull request. |
| T-CTR-032 | API retry uses bounded exponential back-off. |
| T-CTR-033 | Runner timeout emits `OPTIMIZER_TIMEOUT`. |
| T-CTR-034 | Runner timeout discards partial output. |
| T-CTR-035 | The workflow defines a timeout and same-day retry behavior. |
| T-CTR-036 | Rate limiting applies `RATE_LIMIT_RETRY_CONFIG`. |
| T-CTR-037 | Exhausted rate limiting emits `OPTIMIZER_RATE_LIMITED`. |
| T-CTR-038 | Rate-limited runs remain incomplete and retry in the next window. |

The Section 6 norm tests are implemented in `pkg/workflow/compiler_threat_optimizer_protocol_test.go`. Run them with:

```bash
go test -run "TestThreatSuppression|TestThreatOptimizer" ./pkg/workflow/
```
