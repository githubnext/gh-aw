# Formal Notes: github-mcp-access-control-compliance/README.md

**Last formalized**: 2026-07-31-16-13-09
**Notation**: TLA+ / Z3-style guard conjunction
**Issue**: (created via safe-output; number assigned post-processing)

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

## Key Invariants

- Guard evaluation is strictly ordered: tool → repo → role → private-repo → blocked-user → integrity.
- The first failing guard determines the returned JSON-RPC error code (-32001..-32006).
- Fail-safe defaults: unrecognized `min-integrity` config denies all; unknown `ContentIntegrity` (rank -1) is always below threshold.

## Edge Cases Identified

- Empty tool name against a non-empty `allowed-tools` list (deny).
- Empty `repos` array is a compile-time validation error but treated as no-match at runtime.
- Unknown `ContentIntegrity` value (rank -1, below any valid minimum).
- Unrecognized `MinIntegrity` config value (fail-safe deny-all).
- Blocked user fires before integrity check even when integrity would also fail (P5 precedes P6).

## Notes for Future Runs

- The repository already has a complete, verified executable conformance suite at
  `pkg/workflow/github_mcp_access_control_formal_test.go` (495 lines) that is fixture-driven
  against all 10 YAML files in `specs/github-mcp-access-control-compliance/`. No gaps found
  between the README's formal model and the existing test implementation.
- Cross-reference: the normative spec lives at `scratchpad/github-mcp-access-control-specification.md`
  (outside `specs/`) — a future run could formalize that document directly for deeper §4-§10 coverage
  beyond what the README fixture summary captures.
