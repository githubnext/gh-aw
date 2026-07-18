# Formal Notes: security-architecture-spec-validation.md

**Last formalized**: 2026-07-18-15-41-29
**Notation**: TLA+ / Z3 / F*
**Issue**: (pending assignment)

## Predicates

| ID | Predicate | Description |
|---|---|---|
| P1 | `JobArchitectureTripartite` | activation, agent, safe_outputs jobs must all be present |
| P2 | `InputSanitizationPipeline` | sanitizeRunStepExpressions must process steps with ${{ }} before any other transform |
| P3 | `PermissionDefaultReadOnly` | agent job must have only read permissions (PM-01/PM-02) |
| P4 | `ForkProtectionRepoIDGuard` | PR workflows must contain head.repo.id == repository_id comparison |
| P5 | `RBACPreActivationGates` | pre_activation must gate activation via membership check |
| P6 | `ThreatDetectionOrdering` | detection job must sit between agent and safe_outputs |
| P7 | `ActionPinningSHA40` | all non-local uses: must end with @<40-hex-sha> |
| P8 | `TimestampValidationStep` | activation job must reference check_workflow_timestamp |
| P9 | `ConcurrencyDynamicGroup` | concurrency.group must contain ${{ }} expression |
| P10 | `PermissionWriteOnlySafeOutput` | activation/agent jobs must not carry write permissions |
| P11 | `ConcurrencyQueueConflict` | queue:max + cancel-in-progress:true must be rejected |
| P12 | `SanitizationRunStepExpr` | table-driven edge cases for run_step_sanitizer |
| P13 | `EmptyGroupExpressionRejected` | empty/whitespace concurrency group must be rejected |
| P14 | `MalformedGroupExprRejected` | unbalanced ${{ }} in concurrency group must be rejected |
| P15 | `WritePolicyRejectedInStrict` | strict mode must not emit write perms in activation/agent |

## Key Invariants

- Multi-layer job architecture with strict ordering: pre_activation → activation → agent → detection → safe_outputs
- Fork protection via repository-ID comparison guard in job `if:` conditions
- Write permissions isolated exclusively to safe_outputs job
- All action references pinned to 40-character SHA digests in strict mode
- Concurrency groups must carry at least one dynamic GitHub Actions expression
- Input sanitization extracts ${{ }} from run: steps to env: before other transforms

## Edge Cases Identified

- Workflows without roles should NOT emit pre_activation job
- Workflows with queue:max and cancel-in-progress:true must fail at compile time
- Empty or whitespace-only concurrency group expressions must be rejected
- run: steps with no ${{ }} must pass through the sanitizer unchanged
- Detection job may be absent in non-complete-conformance configurations (test skips gracefully)

## Notes for Future Runs

- VAL-P6 uses t.Skip when detection is absent; future run could verify the non-threat-detection path explicitly
- The T-PM-003/T-PM-005/T-PM-007 gaps from the compliance matrix (strict mode, workflow_run repo validation, token validation) are not covered here — a future spec or follow-up issue should address them
- The T-TD-002 through T-TD-007 gap (detection capability assertions) could be formalized once the threat-detection engine interface is exposed via a testable Go API
