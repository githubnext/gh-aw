# Formal Notes: github-mcp-access-control-compliance/README.md

**Last formalized**: 2026-08-14-15-39-20
**Notation**: TLA+ / Z3-style guard conjunction
**Issue**: created via safe-output (number assigned post-processing)

## Predicates

| ID | Predicate | Description |
|---|---|---|
| P1 | `P1_ToolAllowed` | `allowed-tools` filter; empty tool name denies against non-empty list |
| P2 | `P2_RepoMatch` | repo pattern match (owner/repo, owner/*, */repo, */*); omitted=allow-all, empty=deny |
| P3 | `P3_RoleAllow` | role OR-logic match |
| P4 | `P4_PrivateRepoAllow` | private-repos:false denies private repo access |
| P5 | `P5_NotBlocked` | blocked-users author check (evaluated before P6) |
| P6 | `P6_IntegrityMet` | integrity rank ordering none<unapproved<approved<merged; unrecognized config=fail-safe deny; unknown content rank=-1 |
| INV1 | `INV1_CombinedAllow` | ALLOW iff all six guards hold |
| INV2 | `INV2_ErrorCode` | deny code = first failing guard in order |
| SAFETY1 | `SAFETY_BlockedUserAlwaysDenied` | blocked user always denied with -32005 |
| SAFETY2 | `SAFETY_NoSpuriousAllow` | no allow when any guard fails |
| EQ1 | `EQ1_BlockedTerminates` | (new, this run) blocked-users terminates effective-integrity computation regardless of trusted-users/approval-labels |
| EQ2 | `EQ2_TrustedElevatesToApproved` | (new, this run) trusted-users raises base integrity to max(base, approved) |
| EQ3 | `EQ3_LabelElevatesToApproved` | (new, this run) approval-labels raises base integrity to max(base, approved), applied before min-integrity check |
| EQ4 | `EQ4_DefaultIsBase` | (new, this run) no blocked/trusted/label match ⇒ effective = base |
| MONO1 | `MONO_ElevationNeverLowers` | (new, this run) elevation via max() never lowers integrity below base |
| DECISION1 | `DECISION_AccessDecision` | (new, this run) §4.6.3 access decision rule reproducing worked examples table |

## Key Invariants

- Guard evaluation is strictly ordered: tool → repo → role → private-repo → blocked-user → integrity.
- The first failing guard determines the returned JSON-RPC error code (-32001..-32006).
- Fail-safe defaults: unrecognized `min-integrity` config denies all; unknown `ContentIntegrity` (rank -1) is always below threshold.
- Effective integrity computation precedence (§4.6.2): blocked-users > trusted-users > approval-labels > author_association (base).
- Elevation (trusted-users, approval-labels) uses max() and can never lower an item's integrity level.

## Edge Cases Identified

- Empty tool name against a non-empty `allowed-tools` list (deny).
- Empty `repos` array is a compile-time validation error but treated as no-match at runtime.
- Unknown `ContentIntegrity` value (rank -1, below any valid minimum).
- Unrecognized `MinIntegrity` config value (fail-safe deny-all).
- Blocked user fires before integrity check even when integrity would also fail (P5 precedes P6).
- (new) Empty (non-nil) `trusted-users`/`approval-labels` arrays are semantically equivalent to omitted fields — no elevation.
- (new) User matching for `blocked-users`/`trusted-users` is case-insensitive.
- (new) When `min-integrity` is unset, any non-blocked effective integrity is allowed.

## Notes for Future Runs

- The repository has a complete, verified executable conformance suite at
  `pkg/workflow/github_mcp_access_control_formal_test.go` (542 lines) that is fixture-driven
  against all 11 YAML files in `specs/github-mcp-access-control-compliance/`, covering guards P1-P6.
- **This run's gap closure**: the effective-integrity computation algorithm (§4.6.2/§4.6.3 of the
  normative spec) governing `trusted-users` and `approval-labels` elevation was NOT previously
  tested anywhere in the repo. Confirmed via grep that no `computeEffectiveIntegrity`-style
  function exists in `pkg/workflow` — only parsing/env-var passthrough in
  `compiler_github_mcp_steps.go` and `tools_parser.go`. Added a new self-contained formal test
  file `github_mcp_effective_integrity_formal_test.go` with stub types (`contentItem`,
  `integrityGuardConfig`) marked for replacement once a real runtime evaluator is implemented.
- Cross-reference: the normative spec lives at `scratchpad/github-mcp-access-control-specification.md`
  (outside `specs/`). A future run could formalize the remaining untested areas of that document:
  §5 (Repository Scoping), §6+ (not yet reviewed in this or prior runs), or wire the new stub
  types to the real gateway implementation once `trusted-users`/`approval-labels` evaluation logic
  is added to the runtime (currently only compile-time parsing exists).
