# GitHub MCP Access Control Compliance Fixtures

This directory contains fixture stubs for the Section 11 compliance tests of the
[GitHub MCP Access Control Specification](../../scratchpad/github-mcp-access-control-specification.md).

Each fixture describes a test scenario with an input tool configuration and the expected
access-control decision. Fixtures are consumed by the compliance test runner to verify
that implementations satisfy the normative requirements in §§4–10 of the specification.

## Fixture Files

| Filename | Scenario | Spec Coverage |
|---|---|---|
| `exact-match-allow.yaml` | Exact repository pattern allows matching repo | T-GH-011, T-GH-012 |
| `wildcard-deny.yaml` | Owner-wildcard pattern denies non-matching owner | T-GH-013, T-GH-014 |
| `empty-repos-block.yaml` | Absent or empty `repos` list denies all repository access | T-GH-015, T-GH-016 |
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
  error_code: integer | null  # Expected MCP JSON-RPC error code on deny (e.g., -32001)
  reason: string            # Expected denial reason substring (informative)
```

## Error Code Reference

When `expected.decision` is `deny`, the fixture records the MCP JSON-RPC error code that the
implementation MUST return. The codes used in these fixtures are:

| Code | Denial Reason |
|---|---|
| `-32001` | Repository not on the allowlist (`repos` filter) |
| `-32002` | User role is insufficient (`role` filter) |
| `-32003` | Repository is private and `private-repos: false` |
| `-32004` | User is blocked (`blocked-users` filter) |
| `-32005` | Tool name not permitted (`allowed-tools` filter) |
| `-32006` | Content integrity level below threshold (`min-integrity` filter) |

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
