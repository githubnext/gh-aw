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
| `role-deny.yaml` | Role filter denies access when user role is insufficient | T-GH-019, T-GH-020 |
| `private-repo-block.yaml` | `private-repos: false` blocks access to private repository | T-GH-024, T-GH-025 |
| `integrity-level-block.yaml` | `min-integrity: approved` blocks content below the threshold | T-GH-051, T-GH-052 |

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
