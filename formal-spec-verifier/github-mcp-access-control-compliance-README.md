# Formal Notes: github-mcp-access-control-compliance/README.md

**Last formalized**: 2026-09-07-15-32-28
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
| R1 | `P_ExactMatch` | (new 2026-09-07) §5.1.1 exact owner/repo match |
| R2 | `P_OwnerWildcard` | (new 2026-09-07) §5.1.2 owner/* match |
| R3 | `P_NameWildcard` | (new 2026-09-07) §5.1.3 */repo match |
| R4 | `P_FullWildcard` | (new 2026-09-07) §5.1.4 */* matches all, equiv. to omitted repos |
| R5 | `P_AnyPatternMatch` | (new 2026-09-07) §5.2 OR-logic across multiple configured patterns |
| R6 | `P_ExtractOwnerRepo` | (new 2026-09-07) §5.3.1 owner+repo param extraction |
| R7 | `P_ExtractCombined` | (new 2026-09-07) §5.3.2 combined repository param extraction |
| R8 | `P_WideQueryUnfiltered` | (new 2026-09-07) §5.3.3 repo-wide query tools have no single-repo param |
| R9 | `P_CrossRepoPR` | (new 2026-09-07) §5.4.1 PR-across-forks requires both head+base match |
| R10 | `P_CrossRepoTransfer` | (new 2026-09-07) §5.4.2 issue transfer requires both source+target match |

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
- **This run's gap closure (2026-09-07)**: §5 Repository Scoping (pattern matching §5.1,
  multi-pattern OR-logic §5.2, parameter extraction §5.3, cross-repo operations §5.4) had
  no executable formal predicates anywhere in the repo. Added a new self-contained formal
  test file `github_mcp_repo_scoping_formal_test.go` with stub matcher/extractor functions
  (`matchRepoPattern`, `matchAnyPattern`, `extractRepoFromParams`, `crossRepoPRAllowed`,
  `crossRepoTransferAllowed`) marked `// stub — replace with real implementation`, since no
  independently testable exported unit for repo-pattern matching exists yet in `pkg/workflow`
  (repo-scope handling is inline in compiler/gateway wiring).
- Cross-reference: the normative spec lives at `scratchpad/github-mcp-access-control-specification.md`
  (outside `specs/`). Remaining unformalized areas for future runs: §6 Role-Based Filtering
  (permission verification/caching, §6.1-6.3 — only P3_RoleAllow's OR-logic is currently
  tested, not the caching/perf model), §7 Private Repository Controls (visibility caching,
  §7.2), §9 Security Model (threat model §9.1, defense-in-depth §9.2, lockdown override
  §9.5), and §10 Integration with MCP Gateway (middleware architecture, schema extension).
