# ADR-70865: Expose sandbox.mcp.trusted-bots in the Schema and Compile Reference Doc Examples as Regression Tests

**Date**: 2026-10-21
**Status**: Draft
**Deciders**: pelikhan, Copilot (PR #70865) [TODO: verify]

---

### Context

The MCP gateway reference page (`docs/src/content/docs/reference/mcp-gateway.md`) documented two frontmatter examples that did not work as written. The `sandbox.mcp.trusted-bots` option was already implemented by the compiler (it emits `trustedBots` into the gateway configuration), but the canonical workflow schema (`pkg/parser/schemas/main_workflow_schema.json`) sets `additionalProperties: false` on `sandbox.mcp`, so any workflow using the documented option was rejected at validation time. Separately, the OIDC example placed `mcp-servers` under `tools:` instead of at the frontmatter root and omitted the `id-token: write` permission and network allowlist it needs. These mismatches were reported by users in discussion #69855. The docs had been plain `yaml` snippets that nothing checked against the compiler, which is how they drifted.

### Decision

We will add `trusted-bots` to the `sandbox.mcp` schema as a nonempty array of nonempty strings. The schema description states that entries are merged with the gateway's built-in trusted identities and cannot remove or override them. We will also turn both reference examples into complete `aw` workflow blocks and add a Go regression test (`pkg/workflow/mcp_gateway_reference_test.go`). The test pulls those exact fenced blocks out of the documentation page, compiles them with the real compiler, and checks the generated lock file for the expected gateway configuration. The main driver is keeping the schema, compiler, and published docs in agreement, so documented examples cannot silently break again.

### Alternatives Considered

#### Alternative 1: Fix only the documentation text and schema, without tests

Correcting the YAML snippets and adding the schema property would fix the reported bugs at the lowest cost. It was not chosen because nothing would stop the same drift from coming back. The bugs happened precisely because the docs were never checked against the compiler.

#### Alternative 2: Keep duplicate copies of the examples as fixture files under `pkg/`

Tests could compile hand-written fixture workflows that mirror the docs. This avoids parsing Markdown in a Go test, but there would be two copies of each example, and the published docs could still drift from the fixtures. Extracting the actual blocks from the page makes the docs the single source of truth.

#### Alternative 3: Remove `trusted-bots` from the docs instead of allowing it in the schema

The compiler already supports the field, and users asked for it, so removing the documentation would hide a working feature. Adding the schema entry is the smaller and more accurate change.

### Consequences

#### Positive
- Workflows using `sandbox.mcp.trusted-bots` now pass schema validation. Invalid shapes (empty list, empty string, scalar, mixed types, null) are rejected, as tested in `pkg/parser/schema_trusted_bots_test.go`.
- The documented examples are now proven to compile and to produce the expected `trustedBots` and OIDC gateway output. Future docs or compiler changes that break them will fail CI.
- The examples are complete workflows that users can copy and use directly.

#### Negative
- The test depends on the exact fence string (`` ```aw wrap title=".github/workflows/<name>.md" ``) and the docs file path. Editing the docs layout or fence attributes can break the test even when behavior is unchanged.
- Unit tests now read docs content, which mixes documentation and Go test concerns. Other contributors may not expect this coupling.
- Only two examples on the page are covered. Other snippets stay unverified unless the same pattern is extended to them.

#### Neutral
- The test asserts on substrings of the generated lock file, so changes to lock file formatting may require updating the expected strings.
- The PR intentionally does not add `tools.github.refusal-labels` or change concurrency behavior, which were also mentioned in the source discussion.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
