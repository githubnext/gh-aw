# ADR-27707: Unify GitHub Access Mode Under `tools.github.mode`

**Date**: 2026-04-21
**Status**: Draft
**Deciders**: pelikhan

---

## Part 1 — Narrative (Human-Friendly)

### Context

The gh-aw workflow compiler historically controlled GitHub access behavior through a `features.cli-proxy` boolean flag. When `true`, the agent used a pre-authenticated `gh` CLI for GitHub reads instead of the GitHub MCP server. Separately, `tools.github.mode` controlled only MCP transport type (`local` vs `remote`). This split created semantic ambiguity: one concept (how the agent talks to GitHub) was expressed in two different configuration places with incompatible value spaces. As the platform matured, the MCP server gained a `remote` hosted variant at `api.githubcopilot.com`, and a cleaner abstraction was needed that could express all three access modes — CLI, local MCP, and remote MCP — in one unified location.

### Decision

We will consolidate GitHub agent-access semantics into `tools.github.mode` with three values: `gh-proxy` (pre-authenticated `gh` CLI guidance, replacing `features.cli-proxy: true`), `local` (Docker-based MCP server, previously the implicit default), and `remote` (hosted MCP server at api.githubcopilot.com). We will additionally introduce `tools.github.type` to carry the MCP transport type (`local|remote`) independently of the new CLI/MCP semantic distinction, allowing them to evolve separately. Legacy `features.cli-proxy: true` configurations remain functional through a backward-compatibility fallback; a codemod migrates them automatically to `tools.github.mode: gh-proxy`.

### Alternatives Considered

#### Alternative 1: Keep `features.cli-proxy` as the primary flag

The boolean flag could have been extended with a third `remote-mcp` option or companion flags. This was rejected because the `features.*` namespace is intended for unstable/experimental toggles, and CLI-vs-MCP is now a stable, first-class configuration concern. Mixing stable mode semantics into the feature-flag namespace would increase confusion and make schema documentation harder to maintain.

#### Alternative 2: Overload `tools.github.mode` with both access mode and transport type in a flat value set

`tools.github.mode` could have accepted all four combinations as individual string values (e.g., `gh-proxy`, `local-mcp`, `remote-mcp`). This was rejected because `local` and `remote` already had documented meanings as MCP transport values; renaming them would be a breaking change. Separating concerns into `mode` (access paradigm) and `type` (transport) is more expressive and forwards-compatible.

### Consequences

#### Positive
- Single, canonical location for GitHub access configuration, reducing cognitive load for workflow authors.
- `features.cli-proxy` removal cleans up the feature-flag namespace; the codemod ensures a smooth migration path.
- `tools.github.type` provides an independent dimension for future MCP transport evolution without re-breaking `mode` semantics.

#### Negative
- `tools.github.mode` now carries two overlapping semantic layers: the new `gh-proxy` value has a distinct meaning from the legacy `local`/`remote` values, which now describe only transport. This requires careful documentation and can confuse readers who encounter a `mode: local` value and expect it to mean "not CLI mode."
- The backward-compatibility fallback means the compiler must maintain both code paths until all existing workflows have been migrated, increasing maintenance surface.

#### Neutral
- All existing lock files are regenerated as a side effect of the schema change (frontmatter hash churn across `.lock.yml` files).
- The new `tools.github.type` field is plumbed through the schema and compiler but has no user-visible prompt behavior change; it is effectively a no-op for current workflows that don't set it explicitly.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### GitHub Access Mode (`tools.github.mode`)

1. Implementations **MUST** treat `tools.github.mode: gh-proxy` as equivalent to the legacy `features.cli-proxy: true` behavior: the agent **MUST** receive pre-authenticated `gh` CLI prompt guidance and **MUST NOT** register a GitHub MCP server for reads.
2. Implementations **MUST** treat `tools.github.mode: local` and `tools.github.mode: remote` as MCP transport selectors; these values **MUST NOT** activate CLI-proxy prompt behavior.
3. When `tools.github.mode` is absent, implementations **MUST** fall back to the value of `features.cli-proxy` for backward compatibility.
4. Implementations **MUST NOT** silently ignore unrecognized `tools.github.mode` values; they **SHOULD** log a warning and fall back to legacy behavior.

### GitHub MCP Transport Type (`tools.github.type`)

1. Implementations **MUST** accept `tools.github.type: local` and `tools.github.type: remote` to specify MCP transport independently of `tools.github.mode`.
2. When `tools.github.type` is present, implementations **MUST** prefer it over the legacy `tools.github.mode` transport interpretation for determining MCP transport.
3. Implementations **MAY** accept `tools.github.mode: local` and `tools.github.mode: remote` as a fallback transport selector when `tools.github.type` is absent, for backward compatibility.

### Migration (Codemod)

1. The codemod **MUST** transform `features.cli-proxy: true` into `tools.github.mode: gh-proxy` in workflow frontmatter.
2. The codemod **MUST NOT** alter any other frontmatter keys or values.
3. The codemod **SHOULD** be idempotent: re-running it on an already-migrated file **MUST NOT** produce additional changes.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24751474414) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
