# Formal Notes: security-architecture-spec-summary.md

**Last formalized**: 2026-07-17-15-55-42
**Notation**: TLA+ / F* / Z3
**Issue**: created (run 29593849784)

## Predicates

| ID | Predicate | Description |
|---|---|---|
| P1 | `PM10a_PreActivationSeparation` | pre_activation job precedes activation when roles configured |
| P2 | `PM10b_ActivatedOutputGate` | pre_activation outputs `activated`; activation carries if: gate |
| P3 | `PM10c_PreActivationReadOnly` | pre_activation permissions contain no write-level scopes |
| P4 | `PM10d_RequiredRolesDefault` | extractRoles returns admin,maintainer,write when no roles: field |
| P5 | `AppG1_ActionPinning` | All uses: steps carry 40-char SHA pin (Appendix G checklist) |
| P6 | `AppG2_ForkProtection` | pull_request_target triggers include fork validation step |
| P7 | `StrictMode_BlockWritePermissions` | validateDangerousPermissions errors on write scope in strict mode |

## Key Invariants

- Pre-activation RBAC job must precede activation in compiled job order
- pre_activation job must not hold any write permissions
- extractRoles default is ["admin","maintainer","write"] when field absent
- Action pins must be full 40-char SHA-1 hashes
- pull_request_target workflows require fork validation step

## Edge Cases Identified

- Workflow with roles: [] vs absent roles: — compiler behavior for empty slice vs nil
- pre_activation job absent when no roles (negative invariant test)
- Local action paths (./...) are exempt from SHA-pin requirement
- StrictMode only enforced at compile time, not parse time

## Notes for Future Runs

- Prior runs covered SG-01..SG-07 (security_architecture_sg_formal_test.go) and P1-P10 (security_architecture_formal_test.go)
- This run covers PM-10a..PM-10d (pre-activation RBAC) and AppG1-AppG2 (lock-file checklist)
- Remaining gaps: T-IS-001..T-IS-008 input sanitization matrix, T-NI-001..T-NI-009 network isolation matrix
- AppG7 (runtime validation) and AppG8 (concurrency controls) have no dedicated formal test coverage yet
- Cross-spec dependency: PM-11 (trusted-users enforcement) covered in sg_formal_test.go
