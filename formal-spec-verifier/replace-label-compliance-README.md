# Formal Notes: specs/replace-label-compliance/README.md

**Last formalized**: 2026-08-29-15-32-55
**Notation**: TLA+-style state predicates / Z3-style guard conjunction
**Issue**: (created via safeoutputs create_issue; number resolved post-run)

## Predicates

| ID | Predicate | Description |
|---|---|---|
| E1 | `RepoFormatValid` (RL-014) | Resolved repo string MUST satisfy `owner/repo` shape; malformed strings rejected |
| E2 | `ResolveRepo` (RL-013) | Repository resolution priority: message repo > target-repo config > triggering repo |
| E3 | `ErrorCategoryClassification` (RL-044/RL-045) | §7.1 taxonomy partitions into soft-skip (never fails run) vs hard-error (core.error()) categories |
| E4 | `GateFailureShape` (RL-024/RL-025/RL-026) | required-labels/title-prefix gate failures MUST yield `{success:false, skipped:true}`, distinct from hard errors |
| E5 | `HardError404Rejected` (RL-046a) | setLabels HTTP 404 classifies SETLABELS_FAILED, non-retryable |
| E6 | `SoftSkipNeverFailsRun` (RL-044) | All soft-skip categories are the exact complement of hard-error categories |
| E7 | `HardErrorSurfacedAsCoreError` (RL-045/RL-046) | Hard error path produces a non-nil, descriptive error |
| E8 (edge) | `TransientVsHardServerError` (RL-046b) | 5xx / transport-error (status 0) are transient+retryable, contrasting with non-retryable 404 |
| E9 (edge) | `MalformedRepoRejectedEvenWhenAllowlisted` (RL-014/RL-015) | Shape validation (RL-014) precedes and is independent of allowlist membership (RL-015) |
| E10 (edge) | `EmptyOwnerOrRepoSegmentRejected` (RL-014) | `/repo` and `owner/` (empty segments) are rejected |

## Key Invariants

- Repository resolution (RL-013) is strictly priority-ordered: message `repo` field wins, then `target-repo` config, then the triggering workflow's repo — no other precedence is valid.
- The §7.1 error-category table is a strict bipartition: every category is either soft-skip (SCHEMA_INVALID, MAX_EXCEEDED, TARGET_UNRESOLVABLE, REPO_NOT_ALLOWED, LABEL_BLOCKED, LABEL_NOT_ALLOWED, GATE_REQUIRED_LABELS, GATE_TITLE_PREFIX) or hard-error (SETLABELS_FAILED, RATE_LIMIT_EXHAUSTED) — never both, never neither.
- Gate check failures (required-labels, title-prefix) are represented by `skipped: true`, which is a distinct outcome shape from a hard REST failure (`skipped: false`); conflating the two would violate RL-026.
- HTTP 404 vs 5xx/transport-error share the SETLABELS_FAILED category but diverge on retry eligibility — 404 is permanent (target/repo gone), 5xx/transport is transient (retry via RATE_LIMIT_RETRY_CONFIG per RL-046b).

## Edge Cases Identified

- Malformed repo strings (no slash, extra slash, empty owner/repo segment, embedded whitespace) must fail RL-014 shape validation regardless of `allowed-repos` contents — shape validation is a precondition to the allowlist check, not a substitute for it.
- Empty owner segment (`/repo`) and empty repo segment (`owner/`) are both invalid and must be rejected distinctly from a well-formed but disallowed repo.
- Transport-level failures (status 0, representing timeouts/connection errors) must be treated identically to 5xx for retry eligibility purposes, per RL-046b's "or fails with a transport error" clause.

## Notes for Future Runs

- This run intentionally targeted the remaining formal-coverage gaps in `specs/replace-label-spec.md` not covered by the three existing test files (`replace_label_formal_test.go`: P1-P15 + edges; `replace_label_transitions_formal_test.go`: transitions + post-setLabels; `replace_label_security_formal_test.go`: sanitization, rate-limit retry bound, REST failure, label pre-existence, server-side enforcement, token scope). The new file `replace_label_errors_formal_test.go` covers §5.3.1 repository resolution (RL-013/RL-014/RL-015), §5.5 gate-check result shape (RL-026), and §7.1 error taxonomy classification (RL-044/RL-045/RL-046a/RL-046b).
- Coverage is now effectively complete across RL-001 through RL-062 (staged mode, cross-repo T-RL-070–072 spot checks remain only lightly covered via `formalRepoAllowed`/new `formalResolveRepo` — a future run could add fixture-style tests for T-RL-070/071/072 specifically if audit finds gaps).
- Environment note: this sandbox's Go toolchain (1.24.13) could not run `go test` because go.mod pins `go 1.26.7` and the toolchain download was blocked by the network policy (proxy.golang.org returned 403 Forbidden). Validation was limited to `gofmt` syntax/format checking; a CI run with network access to the Go module proxy should execute `go test ./pkg/workflow/... -run TestFormalErr` to confirm runtime correctness.
- Next specs due for rotation: `specs/security-architecture-spec-summary.md`, `specs/security-architecture-spec-validation.md`, `specs/security-architecture-spec.md`, etc. (processed list resets after 14 days per rotation.json cache).
