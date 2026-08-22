# ADR-54714: Share GitHub MCP Common Options via Embedded Struct

**Date**: 2026-08-22
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The GitHub MCP renderer supports two transport modes — Docker (local) and Remote (hosted). Each mode has its own configuration struct (`GitHubMCPDockerOptions` and `GitHubMCPRemoteOptions`). Both structs independently declared the same 8 fields: `ReadOnly`, `Lockdown`, `LockdownFromStep`, `GuardPoliciesFromStep`, `Toolsets`, `Features`, `AllowedTools`, and `GuardPolicies`. These fields control shared MCP behaviour (read-only access, lockdown enforcement, guard policies, toolset selection, feature flags, and allowed-tool filtering) regardless of transport. The duplication meant that any change to shared behaviour required coordinated edits in two places, with no compiler-level guarantee that the structs remained in sync, creating ongoing drift risk.

### Decision

We will introduce `GitHubMCPCommonOptions` as a new shared struct containing all 8 transport-agnostic fields, and update `GitHubMCPDockerOptions` and `GitHubMCPRemoteOptions` to embed it anonymously. A regression test (`TestGitHubMCPOptionsEmbedCommonOptions`) uses reflection to assert that both transport structs embed `GitHubMCPCommonOptions`, enforcing the constraint at compile/test time. All construction sites in `mcp_renderer_github.go` initialise shared fields via the embedded struct literal.

### Alternatives Considered

#### Alternative 1: Keep duplicated fields (status quo)

Each transport struct retains its own independent copy of the 8 shared fields. Behaviour is identical to the new approach at runtime. Rejected because: there is no mechanism to prevent the structs from diverging independently — the problem that motivated this PR — and future changes to shared fields must always be made twice with no compiler enforcement.

#### Alternative 2: Shared builder function instead of struct embedding

A helper function (e.g., `newCommonOptions(...)`) could accept the common arguments and populate each transport struct's fields individually. This avoids anonymous embedding and keeps field access flat. Rejected because: it does not create a named type boundary visible in struct literals, making it harder to see at a glance which fields are shared; and it provides no compile-time or test-time guarantee that both transport structs expose the same set of shared fields.

### Consequences

#### Positive
- Single definition for all shared GitHub MCP configuration fields; any new shared field is added once.
- Compile/test-time enforcement via `TestGitHubMCPOptionsEmbedCommonOptions` prevents future structs from silently omitting the embedded type.
- Reduced field count in transport-specific structs; transport-specific fields are clearly distinguished from shared ones.

#### Negative
- Call sites must use the explicit embedded struct key (`GitHubMCPCommonOptions: GitHubMCPCommonOptions{...}`) in keyed struct literals, which is more verbose than flat field assignment.
- Go's field promotion means shared fields appear on the transport struct's surface, which can obscure their origin for readers unfamiliar with the embedding relationship.

#### Neutral
- All existing test cases required mechanical updates to use the embedded struct literal syntax — no test logic changed, only struct initialisation syntax.
- The regression test uses `reflect.Type.FieldByName` + `Anonymous` check, which is a non-zero dependency on reflection in the test suite.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
