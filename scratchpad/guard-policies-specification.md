# Guard Policies Integration Proposal

## Executive Summary

This document proposes an extensible guard policies framework for the MCP Gateway, starting with GitHub-specific policies. Guard policies enable fine-grained access control at the MCP gateway level, restricting which repositories and operations AI agents can access through MCP servers.

## Problem Statement

The user requested support for guard policies in the MCP gateway configuration, with the following requirements:

1. Support GitHub-specific guard policies with flat frontmatter syntax:
   - `allowed-repos` (scope): Repository access patterns
   - `min-integrity` (minintegrity): Minimum min-integrity level required

2. Design an extensible system that can support future MCP servers (Jira, WorkIQ) with different policy schemas

3. Expose these parameters through workflow frontmatter fields

## Proposed Solution

### 1. Type Hierarchy

```
GitHubToolConfig (GitHub-specific)
  ├── Repos: GitHubReposScope (string or []any)
  └── MinIntegrity: GitHubIntegrityLevel (enum)

MCPServerConfig (general)
  └── GuardPolicies: map[string]any (extensible for all servers)
```

### 2. GitHub Guard Policy Schema

Based on the provided JSON schema, the implementation supports:

**Repos Scope:**
- `"all"` - All repositories accessible by the token
- `"public"` - Public repositories only
- Array of patterns:
  - `"owner/repo"` - Exact repository match
  - `"owner/*"` - All repositories under owner
  - `"owner/prefix*"` - Repositories with name prefix under owner

**Integrity Levels:**

Integrity levels are based on the combination of the `author_association` field associated with GitHub objects and whether an object is reachable from the main branch:

- `"merged"` - Objects reachable from the main branch (highest integrity, regardless of authorship)
- `"approved"` - Objects with `author_association` of `OWNER`, `MEMBER`, or `COLLABORATOR`
- `"unapproved"` - Objects with `author_association` of `CONTRIBUTOR` or `FIRST_TIME_CONTRIBUTOR`
- `"none"` - Objects with `author_association` of `FIRST_TIMER` or `NONE` (lowest integrity)

### 3. Frontmatter Syntax

**Minimal Example:**
```yaml
tools:
  github:
    mode: remote
    toolsets: [default]
    allowed-repos: "all"
    min-integrity: unapproved
```

**With Repository Patterns:**
```yaml
tools:
  github:
    mode: remote
    toolsets: [default]
    allowed-repos:
      - "myorg/*"
      - "partner/shared-repo"
      - "docs/api-*"
    min-integrity: approved
```

**Public Repositories Only:**
```yaml
tools:
  github:
    allowed-repos: "public"
    min-integrity: none
```

> **Note**: The field was originally named `repos` and renamed to `allowed-repos` in PR #22331. The old name is retained as a deprecated alias; run `gh aw fix` to migrate automatically.

### 4. MCP Gateway Configuration Flow

1. **Frontmatter Parsing** (`tools_parser.go`):
   - Extracts `allowed-repos` and `min-integrity` directly from GitHub tool config
   - Stores them as fields on `GitHubToolConfig`
   - Validates structure and types

2. **Validation** (`tools_validation.go`):
   - Validates allowed-repos format (all/public or valid patterns)
   - Validates min-integrity level (none/unapproved/approved/merged)
   - Validates repository pattern syntax (lowercase, valid characters, wildcard placement)
   - Called during workflow compilation

3. **Compilation**:
   - Guard policy fields (allowed-repos, min-integrity) included in compiled GitHub tool configuration
   - Passed through to MCP Gateway configuration

4. **Runtime (MCP Gateway)**:
   - Gateway receives guard policies in server configuration
   - Enforces policies on all tool invocations
   - Blocks unauthorized repository access

### 5. Safe Outputs Integration

When GitHub guard policies are configured, the compiler automatically derives a linked guard-policy for the safe-outputs MCP server. This ensures that safe output operations work correctly with guard policies by creating a write-sink configuration.

**Normative Requirements for `deriveSafeOutputsGuardPolicyFromGitHub()`:**

- **MUST derive a `write-sink` guard policy** for the safe-outputs MCP server whenever a GitHub guard policy (`allowed-repos` or `min-integrity`) is present in the workflow frontmatter. The derived policy MUST be applied before the workflow is executed.
- **MUST map `allowed-repos: "all"` or `allowed-repos: "public"` to `accept: ["*"]`**, allowing all safe output operations. Implementations MUST NOT restrict the write-sink scope when the GitHub guard policy already permits all repositories.
- **MUST transform each repository pattern** in an `allowed-repos` array to a `private:`-prefixed accept entry. Owner-wildcard patterns (`owner/*`) MUST be transformed to `private:owner` (the trailing `/*` is stripped). Prefix-wildcard patterns (`owner/prefix*`) MUST be transformed to `private:owner/prefix*` (the prefix is preserved). Exact repository patterns (`owner/repo`) MUST be transformed to `private:owner/repo`.
- **MUST NOT include duplicate accept entries** in the derived `write-sink` policy. If multiple input patterns resolve to the same `private:` value, the implementation MUST deduplicate before emitting the accept list.
- **SHOULD log a debug-level message** when a guard policy is derived, identifying the source GitHub `allowed-repos` value and the resulting accept list. This assists operators in diagnosing unexpected policy behavior.
- **MUST return `nil`** (no derived policy) when no GitHub guard policy fields are present on the tool configuration. The absence of a guard policy MUST NOT be treated as an implicit `accept: ["*"]` — the decision to omit the policy is intentional and MUST be preserved.

**Derivation Rules:**

- **`allowed-repos: "all"` or `allowed-repos: "public"`**: Creates `accept: ["*"]` to allow all safe output operations
- **`allowed-repos: [patterns]`**: Each pattern is transformed and added to the accept list:
  - `"owner/*"` → `"private:owner"` (owner wildcard → strip wildcard)
  - `"owner/prefix*"` → `"private:owner/prefix*"` (prefix wildcard → keep as-is)
  - `"owner/repo"` → `"private:owner/repo"` (specific repo → keep as-is)

**Example - Public Repositories:**

```yaml
tools:
  github:
    allowed-repos: "public"
    min-integrity: approved
```

Generates safeoutputs guard-policy:
```json
{
  "write-sink": {
    "accept": ["*"]
  }
}
```

**Example - Specific Repositories:**

```yaml
tools:
  github:
    allowed-repos:
      - "github/*"
      - "microsoft/copilot"
    min-integrity: approved
```

Generates safeoutputs guard-policy:
```json
{
  "write-sink": {
    "accept": [
      "private:github",
      "private:microsoft/copilot"
    ]
  }
}
```

**Implementation:**
- Function: `deriveSafeOutputsGuardPolicyFromGitHub()` in `pkg/workflow/mcp_github_config.go`
- Called during MCP renderer setup for safeoutputs server
- Tests: `pkg/workflow/safeoutputs_guard_policy_test.go`

### 6. Extensibility for Future Servers

The design supports future MCP servers (Jira, WorkIQ) through:

1. **Server-Specific Policy Fields:**
   ```go
   type JiraToolConfig struct {
       // ... other fields ...
       // Guard policy fields (flat syntax under jira:)
       Projects   []string `yaml:"projects,omitempty"`
       IssueTypes []string `yaml:"issue-types,omitempty"`
   }
   ```

2. **General MCPServerConfig Field:**
   ```go
   type MCPServerConfig struct {
       // ...
       GuardPolicies map[string]any `yaml:"guard-policies,omitempty"`
   }
   ```

3. **Frontmatter Configuration:**
   ```yaml
   tools:
     jira:
       mode: remote
       projects: ["PROJ-*", "SHARED"]
       issue-types: ["Bug", "Story"]
   ```

## Implementation Details

### Files Modified

1. **pkg/workflow/tools_types.go**
   - Added `GitHubIntegrityLevel` enum type
   - Added `GitHubReposScope` type alias
   - Extended `GitHubToolConfig` with flat `Repos` and `MinIntegrity` fields
   - Extended `MCPServerConfig` with `GuardPolicies` field

2. **pkg/workflow/schemas/mcp-gateway-config.schema.json**
   - Added `guard-policies` field to `stdioServerConfig`
   - Added `guard-policies` field to `httpServerConfig`
   - Set `additionalProperties: true` for server-specific schemas

3. **pkg/workflow/tools_parser.go**
   - Extended `parseGitHubTool()` to extract `allowed-repos` and `min-integrity` directly

4. **pkg/workflow/tools_validation.go**
   - Updated `validateGitHubGuardPolicy()` function (validates flat fields)
   - Added `validateReposScope()` function
   - Added `validateRepoPattern()` function
   - Added `isValidOwnerOrRepo()` helper function

5. **pkg/workflow/compiler_orchestrator_workflow.go**
   - Added call to `validateGitHubGuardPolicy()`

6. **pkg/workflow/compiler_string_api.go**
   - Added call to `validateGitHubGuardPolicy()`

### Validation Rules

**Repository Patterns:**
- Must be lowercase
- Format: `owner/repo`, `owner/*`, or `owner/prefix*`
- Owner and repo parts must contain only: lowercase letters, numbers, hyphens, underscores
- Wildcards only allowed at end of repo name
- Empty arrays not allowed

**Integrity Levels:**
- Must be one of: `none`, `unapproved`, `approved`, `merged`
- Case-sensitive

**Required Fields:**
- `min-integrity` is required when using GitHub guard policies
- `allowed-repos` defaults to `"all"` if not specified

## Error Messages

The implementation provides clear, actionable error messages:

```
invalid guard policy: repository pattern 'Owner/Repo' must be lowercase

invalid guard policy: repository pattern 'owner/re*po' has wildcard in the middle.
Wildcards only allowed at the end (e.g., 'prefix*')

invalid guard policy: 'github.min-integrity' must be one of: 'none', 'unapproved', 'approved', 'merged'.
Got: 'admin'
```

## Usage Examples

### Example 1: Restrict to Organization

```yaml
tools:
  github:
    mode: remote
    toolsets: [default]
    allowed-repos:
      - "myorg/*"
    min-integrity: unapproved
```

### Example 2: Multiple Organizations

```yaml
tools:
  github:
    mode: remote
    toolsets: [default]
    allowed-repos:
      - "frontend-org/*"
      - "backend-org/*"
      - "shared/infrastructure"
    min-integrity: approved
```

### Example 3: Public Repositories Only

```yaml
tools:
  github:
    mode: remote
    toolsets: [repos, issues]
    allowed-repos: "public"
    min-integrity: none
```

### Example 4: Prefix Matching

```yaml
tools:
  github:
    mode: remote
    toolsets: [default]
    allowed-repos:
      - "myorg/api-*"     # Matches api-gateway, api-service, etc.
      - "myorg/web-*"     # Matches web-frontend, web-backend, etc.
    min-integrity: approved
```

## Testing Strategy

1. **Unit Tests** (Complete):
   - `TestValidateGitHubGuardPolicy`: 14 cases covering valid/invalid repos values, invalid min-integrity, missing fields
   - `TestValidateReposScopeWithStringSlice`: 4 cases covering `[]string` and `[]any` input types
   - Tests live in `pkg/workflow/tools_validation_test.go`

2. **Integration Tests** (Complete):
   - `TestGuardPolicyYAMLCompilationIntegration`: 5 round-trip tests in `pkg/workflow/guard_policy_compilation_integration_test.go`
     - `allowed-repos: all` → `accept: ["*"]` write-sink in compiled YAML
     - `allowed-repos: public` → `accept: ["*"]` write-sink in compiled YAML
     - Single specific repo → `"private:owner/repo"` in compiled YAML
     - Owner-wildcard repo (`owner/*`) → `"private:owner"` (stripped wildcard) in compiled YAML
     - Multiple repos → multiple `"private:..."` accept entries in compiled YAML
   - These tests verify that guard policies appear in the compiled lock YAML at the correct structure

## Next Steps

1. **Write Tests**:
   - Unit tests for parsing functions
   - Unit tests for validation functions
   - Integration tests for end-to-end workflow compilation

2. **Update Documentation**:
   - Add guard policies section to MCP gateway documentation
   - Add examples to GitHub MCP server documentation
   - Update frontmatter configuration reference

3. **Runtime Implementation** (Separate from this PR):
   - MCP Gateway enforcement of guard policies
   - Repository pattern matching logic
   - Integrity level verification
   - Access control logging

## Benefits

1. **Security**: Restrict AI agent access to specific repositories
2. **Compliance**: Enforce minimum min-integrity requirements
3. **Flexibility**: Support diverse repository patterns and wildcards
4. **Extensibility**: Supports adding policies for Jira, WorkIQ, etc.
5. **Clarity**: Clear error messages and validation
6. **Documentation**: Self-documenting through type system

## Open Questions

> **Status**: All four open questions below have been resolved with decision records.

1. **Should we support negative patterns (e.g., exclude certain repos)?**

   **Decision**: No, negative patterns (e.g., `!owner/repo`) are **not supported** in the initial implementation.
   *Rationale*: Negative patterns introduce ordering complexity and ambiguity when combined with wildcard rules (e.g., `"owner/*"` and `"!owner/private-repo"` create a subtraction model that is hard to reason about safely). The preferred approach is to use an explicit allowlist — specify only what is permitted rather than excluding items from a broader grant. If a workflow requires fine-grained exclusions, it SHOULD use a narrower `allowed-repos` pattern. Negative patterns may be revisited in a future version if a clear security use-case emerges.

2. **Should we support combining multiple policies (AND/OR logic)?**

   **Decision**: Policies within a single MCP server are evaluated as **AND** conjunctions. Multiple `allowed-repos` entries in an array are evaluated as **OR** (any match grants access).
   *Rationale*: AND semantics for the combination of `allowed-repos` + `min-integrity` is the only safe default — a request must satisfy both the repository scope constraint AND the integrity constraint to proceed. Within `allowed-repos`, OR semantics (any matching pattern) is the standard allowlist behavior and consistent with how `roles` and other list-valued fields work throughout the compiler. Explicit cross-policy AND/OR combinators are deferred as unnecessary complexity; the current model covers all known production use-cases.

3. **How should conflicts between lockdown and guard policies be resolved?**

   **Decision**: `lockdown: true` takes **absolute precedence** over guard policies. When `lockdown: true` is set, all tool invocations are blocked regardless of any `allowed-repos` or `min-integrity` configuration. Guard policies are not evaluated when lockdown is active.
   *Rationale*: Lockdown is an emergency/security stop; it MUST NOT be weakened by other configuration. Guard policies narrow access within an otherwise-open tool session; they do not grant access that lockdown has revoked. The compiler SHOULD warn operators at compilation time when both `lockdown: true` and guard-policy fields (`allowed-repos`, `min-integrity`) are present, as the combination is likely a misconfiguration. A future enhancement to `pkg/workflow/tools_validation.go` (`validateGitHubGuardPolicy`) SHOULD add this cross-field validation check.

4. **Should we add a "dry-run" mode to test policies before enforcement?**

   **Decision**: Dry-run enforcement mode is **deferred** to a future release. A compile-time validation (`gh aw compile --strict`) that reports which repositories would be permitted or denied under the configured guard policy SHOULD be implemented instead.
   *Rationale*: A runtime dry-run mode requires MCP Gateway support for pass-through logging of policy decisions, which is out of scope for the initial implementation. Compile-time policy analysis covers the majority of the validation need (catching misconfigured patterns before deployment) at lower implementation cost. Runtime dry-run may be added when MCP Gateway observability tooling matures.

## Conclusion

This implementation covers guard policies in the MCP gateway. The design is:

- **Type-safe**: Strongly-typed structs with validation
- **Extensible**: New servers and policy types can be added without structural changes
- **Consistent syntax**: Follows existing frontmatter conventions
- **Well-validated**: Validation with clear error messages
- **Forward-compatible**: Supports future enhancements

The implementation follows established patterns in the codebase and integrates with the existing compilation and validation infrastructure.
