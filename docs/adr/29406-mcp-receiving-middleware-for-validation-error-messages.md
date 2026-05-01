# ADR-29406: MCP Receiving Middleware for Helpful Validation Error Messages

**Date**: 2026-05-01
**Status**: Draft
**Deciders**: pelikhan

---

## Part 1 — Narrative (Human-Friendly)

### Context

The MCP server enforces strict JSON schema validation on tool call arguments using `additionalProperties: false`. When a caller passes an unknown parameter (e.g., `workflow-name` instead of `workflows`), the underlying `jsonschema-go` library produces a raw, internal error message such as `validating "arguments": validating root: unexpected additional properties ["workflow-name"]`. This opaque message is surfaced directly to the MCP client with no guidance on the correct parameter name, degrading the developer experience for all 10 registered tools. The MCP Go SDK provides a middleware hook (`AddReceivingMiddleware`) that can intercept tool results at the server boundary before they are returned to the caller.

### Decision

We will implement a generic MCP receiving middleware (`argumentValidationMiddleware`) that intercepts `CallToolResult` errors containing "unexpected additional properties" and replaces them with a human-readable message including the unknown parameter name, a "Did you mean?" suggestion derived from a longest-common-prefix fuzzy matcher, and a pointer to the tool's `--help` output. A hardcoded per-tool parameter registry (`mcpToolParams()`) maintains the list of valid parameter names for each tool, linked by comment to the source-of-truth `*Args` struct in each `register*Tool` function.

### Alternatives Considered

#### Alternative 1: Patch the upstream JSON schema or MCP SDK

The `jsonschema-go` validation error format is controlled by the upstream library; customizing it would require forking or monkey-patching a third-party dependency. This was rejected because it introduces maintenance overhead for upstream changes and violates the project's policy of minimizing fork-based patches to dependencies.

#### Alternative 2: Per-tool error handling at each registration site

Each `register*Tool` function could wrap its handler to detect and reformat validation errors locally. While this avoids a central registry, it means duplicating the detection and formatting logic across ten tools and increases the risk of inconsistent messaging. The single middleware with a shared registry was preferred for uniformity and locality of change.

#### Alternative 3: Levenshtein edit-distance fuzzy matching

Full edit-distance similarity (e.g., via `golang.org/x/text` or a dedicated library) would handle transpositions and mid-string substitutions that longest-common-prefix cannot. It was considered but rejected for the typical case: MCP callers most often pass hyphen/underscore variants of a correct name (e.g., `workflow-name` for `workflows`), which normalization plus LCP handles reliably without an additional dependency.

### Consequences

#### Positive
- All ten MCP tools immediately get helpful error messages without changes to their individual implementations.
- Normalization (strip hyphens and underscores, lowercase) correctly resolves the most common error pattern — delimiter mismatch — before prefix comparison.
- The middleware is transparent: non-validation errors and successful results pass through unchanged, so no existing behavior is affected.

#### Negative
- `mcpToolParams()` is a manually maintained registry; if a parameter is added or renamed in a `*Args` struct without updating this file, the suggestion will become stale or absent.
- LCP-based similarity does not catch mid-string differences or transpositions (e.g., `workflwo` → `workflows`); callers who mistype in those ways receive no suggestion.

#### Neutral
- The middleware is registered once at server construction time, so its cost is one closure allocation; per-call overhead is limited to a single string comparison in the non-matching fast path.
- A maintenance comment block in `mcpToolParams()` links each entry to its source-of-truth registration function to reduce the risk of the registry drifting.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Middleware Registration

1. The MCP server **MUST** register `argumentValidationMiddleware` as a receiving middleware via `AddReceivingMiddleware` after all tools have been registered.
2. The middleware **MUST** be initialized with the result of `mcpToolParams()` at server construction time.
3. The middleware **MUST NOT** modify results for MCP methods other than `tools/call`.
4. The middleware **MUST NOT** modify `CallToolResult` values where `IsError` is `false`.

### Error Interception and Transformation

1. The middleware **MUST** detect tool-call errors whose text content contains the substring `unexpected additional properties`.
2. The middleware **MUST** extract all unknown parameter names from the error message using the `extractUnknownParams` parser.
3. For each unknown parameter, the middleware **MUST** emit an `Unknown parameter '<name>'.` prefix in the replacement message.
4. If a valid parameter whose normalized form shares a longest-common-prefix ratio ≥ 0.70 with the unknown parameter exists, the middleware **MUST** append `Did you mean '<suggestion>'?` to that parameter's line.
5. If the tool name is known, the middleware **MUST** append a `Run 'agenticworkflows <tool> --help' for usage.` line.
6. The replacement `CallToolResult` **MUST** preserve `IsError: true` and **MUST** contain exactly one `TextContent` element with the replacement message.

### Parameter Registry (`mcpToolParams`)

1. The `mcpToolParams()` registry **MUST** contain an entry for every tool registered in `createMCPServer`.
2. Each entry **MUST** list all valid JSON parameter names derived from the corresponding `*Args` struct's `json` tags.
3. Parameter lists **MUST** be kept in sorted order to produce deterministic suggestion output.
4. When a tool's `*Args` struct gains, removes, or renames a parameter, the corresponding `mcpToolParams()` entry **MUST** be updated in the same commit.

### Normalization

1. Parameter name normalization **MUST** lowercase the name and remove all hyphen (`-`) and underscore (`_`) characters before comparison.
2. An exact match after normalization **SHALL** be preferred over any prefix-score comparison and **MUST** return the matching valid parameter immediately.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25196075860) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
