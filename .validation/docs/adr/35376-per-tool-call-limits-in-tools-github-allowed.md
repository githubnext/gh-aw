# ADR-35376: Per-Tool Call Limits in `tools.github.allowed`

**Date**: 2026-05-28
**Status**: Draft
**Deciders**: PR author (pelikhan), reviewers of PR #35376

---

## Part 1 — Narrative (Human-Friendly)

### Context

`tools.github.allowed` already restricts *which* GitHub MCP tools a workflow can call, but it does not bound *how many times* each tool runs in a single workflow execution. This leaves single-read workflows (issue triage, label classification, cross-reference checks) exposed to an agent that re-invokes an allowed read tool dozens of times to expand context or chase references — a small allow-list is not the same as a small blast radius. The MCP gateway already enforces other allow-only policy fields (`repos`, `min-integrity`), so a per-tool call limit is the missing dimension on the same seam. The constraint is that the compiler must stay stateless about runtime tool-call counts: enforcement has to live downstream.

### Decision

We will extend `tools.github.allowed` to accept structured entries that carry a per-tool call cap, in addition to the existing string entries. Two syntaxes are accepted for an entry: a plain string `"tool"` (unchanged, no cap) and an object `{name: "tool", max-calls: N}`. Caps are collected into a `tool-call-limits: { "<tool>": <n> }` map under the existing `allow-only` guard policy, and enforcement is delegated to the MCP gateway/firewall layer — the compiler emits policy only. When a workflow declares call limits but no `allowed-repos`/`repos` or `min-integrity`, the generated `allow-only` policy is still emitted with `repos: "all"` because the gateway requires `repos` to be present on any allow-only block.

### Alternatives Considered

#### Alternative 1: New top-level field `tools.github.tool-call-limits` as a separate map

Keep `allowed` as a plain string array and add a sibling map `tool-call-limits: { issue_read: 1, list_labels: 3 }` at the same level as `allowed`. This was rejected because it splits a tool's definition across two YAML locations: a reviewer scanning `allowed` would not see the limit, and a tool name could drift between the two fields (in `allowed` but not in `tool-call-limits`, or vice versa) with no schema-level signal that one referenced the other. Co-locating the cap with the tool name on the `allowed` entry makes the schema self-describing and matches the principle that one decision goes in one place.

#### Alternative 2: Enforce call limits in the compiler or agent loop instead of the gateway

Emit a wrapper around each MCP tool invocation in the generated workflow that counts calls and refuses past the cap, or have the agent harness track its own call counts. This was rejected because the gateway already evaluates `allow-only` policy (`repos`, `min-integrity`) and is the trust boundary the agent process cannot bypass. Putting enforcement in the agent loop would require the agent to honor a self-imposed limit, which is exactly the failure mode (runaway re-invocation) the cap exists to prevent. Reusing the gateway path also keeps the compiler stateless w.r.t. runtime counts.

#### Alternative 3: Colon-delimited shorthand for limits in string entries

Accept a shorthand string like `"tool:N"` and interpret it as a per-tool limit. This was rejected to keep `allowed` string entries unambiguous tool names and avoid overloading free-form strings with parser-specific syntax.

### Consequences

#### Positive

- Single-read workflows can now declare an explicit per-tool ceiling and rely on the gateway to enforce it, closing the cross-reference-expansion gap that `allowed` alone could not address.
- Existing workflows with string-only `allowed` lists keep working unchanged; the new schema is strictly additive, so no migration is required.
- Co-locating the cap with the tool name (object form) means a reviewer reading the YAML sees both pieces of information at once, and schema validation catches drift.
- Enforcement lives at the same MCP gateway seam that already handles `repos` and `min-integrity`, so the firewall/gateway team owns one policy surface, not two.

#### Negative

- Only the object form can carry limits, so concise one-line string entries cannot express call caps.
- The `allowed` array schema changes from a plain string array to a `oneOf` of string and object, which is harder for YAML-aware editors (and humans) to autocomplete than a flat string array.
- Invalid limit values are silently dropped rather than rejected: `parseGitHubAllowedToolsAndLimits` skips `max-calls` values that fail `typeutil.ParseIntValue` or are `<= 0`. A typo (`max-calls: "one"`) becomes an unlimited tool, not a compile error.
- The emitted `tool-call-limits` field only takes effect against an MCP gateway version that understands it; older gateways will ignore the cap and the workflow will run uncapped. There is no compile-time check that the targeted gateway version supports the field.

#### Neutral

- When a workflow declares call limits but neither `allowed-repos`/`repos` nor `min-integrity`, the compiler now emits `allow-only.repos: "all"` automatically to satisfy the gateway's requirement that `allow-only` always carry a `repos` scope. This is a new implicit policy emission case but it matches the existing fallback already used for `min-integrity`-only configurations.
- The `GitHubAllowedTools` parser in `pkg/workflow/tools_parser.go` intentionally discards limit information at parse time; the limits travel exclusively through `getGitHubGuardPolicies`, so any downstream consumer of `config.Allowed` sees only tool names, as before.
- The new helper `parseGitHubAllowedToolsAndLimits` is reused by both `getGitHubAllowedTools` and `getGitHubGuardPolicies`, so the two paths cannot disagree about which entries are well-formed.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### `tools.github.allowed` Entry Syntax

1. Each entry in `tools.github.allowed` **MUST** be one of: a plain string tool name, or an object with a required `name` field and an optional `max-calls` field.
2. The schema for entries **MUST** be expressed as a `oneOf` of a `string` branch and an `object` branch with `additionalProperties: false` and `required: ["name"]`.
3. The object branch **MUST** declare `max-calls` as `type: integer, minimum: 1`.
4. Plain string entries **MUST** be treated as tool names with no call limit.
5. Object entries **MUST** be skipped (neither the tool name nor a limit emitted) when `name` is missing, not a string, or empty after trimming whitespace.
6. A `max-calls` value that is not a positive integer **MUST** result in the tool name being added to the allowed list without a call limit; the value **MUST NOT** raise a compile error.

### Parser Behavior

1. The function `parseGitHubAllowedToolsAndLimits` **MUST** be the single source of truth for parsing `tools.github.allowed` entries; both `getGitHubAllowedTools` and `getGitHubGuardPolicies` **MUST** call it rather than reimplement entry parsing.
2. The `GitHubToolConfig.Allowed` field produced by `parseGitHubTool` **MUST** contain only tool names; per-tool call limits **MUST NOT** be stored on the parsed config struct.
3. Tool names returned by the parser **MUST** preserve the order they appear in the source YAML.

### Guard Policy Emission

1. When at least one `allowed` entry carries a positive `max-calls`, `getGitHubGuardPolicies` **MUST** emit an `allow-only` block containing a `tool-call-limits` map keyed by tool name with integer values.
2. The `allow-only` block emitted under condition (1) **MUST** also carry a `repos` field; when neither `allowed-repos` nor `repos` is configured on the tool, `repos` **MUST** default to `"all"`.
3. The `tool-call-limits` map **MUST** only include tools whose limit successfully parsed as a positive integer; tools without limits **MUST NOT** appear as keys with value `0`.
4. When no `allowed` entry carries a call limit and neither `allowed-repos`/`repos` nor `min-integrity` is configured, `getGitHubGuardPolicies` **MUST NOT** emit an `allow-only` block solely because `allowed` is non-empty.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/26555753359) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
