# GitHub MCP Access Control Compliance Fixtures

This directory contains fixture stubs for the Section 11 compliance tests of the
[GitHub MCP Access Control Specification](../../scratchpad/github-mcp-access-control-specification.md).

Each fixture describes a test scenario with an input tool configuration and the expected
access-control decision. Fixtures are consumed by the compliance test runner to verify
that implementations satisfy the normative requirements in §§4–10 of the specification.

## Formal Model

Let:

- `r ∈ AccessRequest`
- `c ∈ ToolConfig`
- `Decision(r, c) ∈ {allow, deny(code)}`

The runtime decision is a conjunction of six guard predicates evaluated in the documented
order (spec §4.5.3):

```
ALLOW(r, c) ≜
  P1_ToolAllowed(r, c) ∧
  P2_RepoMatch(r, c)   ∧
  P3_RoleAllow(r, c)   ∧
  P4_PrivateRepoAllow(r, c) ∧
  P5_NotBlocked(r, c)  ∧
  P6_IntegrityMet(r, c)
```

**Evaluation order** (§4.5.3):
1. **Tool selection** — `allowed-tools` filter determines which tools are available
2. **Repository access control** — repo scope, role, and visibility are evaluated
3. **Integrity management** — blocked-user author check, then min-integrity threshold

Where:

- `P1_ToolAllowed`: if `allowed-tools` is configured, the requested tool name must be present; an empty or absent tool name against a non-empty list also denies
- `P2_RepoMatch`: if `repos` is configured, repository matches at least one pattern (`owner/repo`, `owner/*`, `*/repo`, `*/*`); omitted `repos` allows all accessible repositories; empty array is a compile-time validation error (§4.4.1)
- `P3_RoleAllow`: if `roles` is configured, user role matches one configured role (OR-logic)
- `P4_PrivateRepoAllow`: private repository access is denied when `private-repos: false`
- `P5_NotBlocked`: blocked users are denied within integrity management (author check)
- `P6_IntegrityMet`: integrity ordering is enforced as `none < unapproved < approved < merged`

The denial code is selected by the first failing guard in the evaluation order above.

## Behavioral Coverage Map

| Predicate / Invariant | Test Function | Description |
|---|---|---|
| `P1_ToolAllowed` (exact allow) | `TestFormal_ToolNameFilter` | `allowed-tools` allows named tool, denies others; empty tool name denies against non-empty list |
| `P2_RepoMatch` (exact) | `TestFormal_ExactMatchAllow` | Exact `owner/repo` pattern allows matching repo, denies others |
| `P2_RepoMatch` (wildcard) | `TestFormal_WildcardMatch` | `owner/*` and `*/repo` wildcard patterns; `*/*` full wildcard |
| `P2_RepoMatch` (omitted) | `TestFormal_OmittedReposAllowAll` | Omitted `repos` allows all accessible repositories; empty array is invalid config |
| `P3_RoleAllow` | `TestFormal_RoleFilter` | Role OR-logic: matching role allows, insufficient role denies |
| `P4_PrivateRepoAllow` | `TestFormal_PrivateRepoControl` | `private-repos: false` blocks private repos; public repos unaffected |
| `P5_NotBlocked` | `TestFormal_BlockedUserDeny` | Blocked user denied within integrity management |
| `P6_IntegrityMet` | `TestFormal_IntegrityLevelOrder` | Integrity ordinal order enforced; content below threshold denied |
| `P6_IntegrityMet` (unknown content) | `TestFormal_UnknownContentIntegrityDenied` | Unknown `ContentIntegrity` value (rank -1) is below any valid minimum threshold → denied |
| `P6_IntegrityMet` (invalid config) | `TestFormal_InvalidMinIntegrityConfigDenied` | Unrecognized `MinIntegrity` config is fail-safe: denies all requests |
| `INV1_CombinedAllow` | `TestFormal_CombinedFiltersAllAllow` | All conditions must be satisfied for allow |
| `INV2_ErrorCode` | `TestFormal_ErrorCodeFirstFailingGuard` | Deny error code matches first failing guard; table covers each guard as first failure |
| `SAFETY_BlockedUserAlwaysDenied` | `TestFormal_BlockedUserSafetyProperty` | Safety: blocked user always produces `-32005` when all earlier guards pass |
| `SAFETY_NoSpuriousAllow` | `TestFormal_NoSpuriousAllowInvariant` | Safety: no allow decision when any guard fails |

## Fixture Files

| Filename | Scenario | Spec Coverage |
|---|---|---|
| `exact-match-allow.yaml` | Exact repository pattern allows matching repo | T-GH-011, T-GH-012 |
| `wildcard-deny.yaml` | Owner-wildcard pattern denies non-matching owner | T-GH-013, T-GH-014 |
| `empty-repos-block.yaml` | Empty `repos` array is rejected at compile time | T-GH-015, T-GH-016 |
| `role-deny.yaml` | Role filter denies access when user role is insufficient | T-GH-019, T-GH-020 |
| `tool-name-filter.yaml` | `allowed-tools` filter allows or denies by tool name | T-GH-031, T-GH-032, T-GH-033 |
| `blocked-user-deny.yaml` | `blocked-users` denies listed actors unconditionally | T-GH-071, T-GH-072 |
| `private-repo-block.yaml` | `private-repos: false` blocks access to private repository | T-GH-024, T-GH-025 |
| `integrity-level-block.yaml` | `min-integrity: approved` blocks content below the threshold | T-GH-051, T-GH-052 |
| `combined-filter-allow.yaml` | All access-control conditions must be jointly satisfied | T-GH-081, T-GH-082, T-GH-083 |

## Fixture Schema

Each fixture file is a YAML document with the following top-level keys:

```yaml
fixture_id: string          # Unique identifier matching the test IDs in §11.1
description: string         # Human-readable scenario description
spec_refs:                  # Normative requirements under test (§ references)
  - string
input:
  tool_config: object       # Compiled GitHub MCP tool configuration under test
  request: object           # Simulated access request (repository, user, content)
expected:
  decision: allow | deny    # Required access-control outcome
  error_code: integer | null  # Expected MCP JSON-RPC error code on deny (e.g., -32002)
  reason: string            # Expected denial reason substring (informative)
```

## Error Code Reference

When `expected.decision` is `deny`, the fixture records the MCP JSON-RPC error code that the
implementation MUST return. The codes are defined in `pkg/cli/gateway_logs_types.go` and the
normative specification (Appendix B):

| Code | Denial Reason |
|---|---|
| `-32001` | General access denied (tool not in `allowed-tools` filter) |
| `-32002` | Repository not in allowlist (`repos` filter) |
| `-32003` | Insufficient permissions (`roles` filter) |
| `-32004` | Repository is private and `private-repos: false` |
| `-32005` | Content from blocked user (`blocked-users` filter) |
| `-32006` | Content integrity below minimum threshold (`min-integrity` filter) |

A `null` error code in `expected.error_code` means the scenario produces an `allow` decision
and no error is returned.

## Adding New Fixtures

1. Copy the most relevant existing fixture file.
2. Change `fixture_id` to a new unique identifier.
3. Update `input.tool_config` and `input.request` to reflect the new scenario.
4. Update `expected` fields to match the required outcome.
5. Register the new fixture in the table above and link it from §11.4 of the specification.

## Running Compliance Tests

Compliance tests that consume these fixtures are located in (or will be added to):

```
pkg/workflow/tools_validation_test.go   — §11.1.1 configuration validation
pkg/workflow/tools_validation_test.go   — §11.1.8 blocked-user tests
```

To run all related tests:

```bash
go test -v -run "TestValidateGitHubGuardPolicy" ./pkg/workflow/
```

To run the formal conformance test suite (predicate-mapped tests):

```bash
go test -v -run "TestFormal_(ExactMatch|WildcardMatch|OmittedRepos|RoleFilter|PrivateRepo|BlockedUser|ToolName|IntegrityLevel|UnknownContent|InvalidMinIntegrity|CombinedFilters|ErrorCode|NoSpurious)" ./pkg/workflow/
```

To run the YAML fixture runner (drives every scenario from the fixture files above through the formal evaluator):

```bash
go test -v -run "TestFormal_FixtureRunner" ./pkg/workflow/
```

## Generated Test Suite

Formal conformance tests are implemented in:

`pkg/workflow/github_mcp_access_control_formal_test.go`

The test suite includes:
- **Predicate-mapped tests** (`TestFormal_*`) — each test maps to a specific guard predicate (P1–P6) or invariant documented in the Formal Model section above.
- **Fixture runner** (`TestFormal_FixtureRunner`) — loads every YAML fixture file from this directory and drives each scenario through the formal evaluator. This ensures the fixture files, error codes, and expected decisions remain consistent with the formal model.
