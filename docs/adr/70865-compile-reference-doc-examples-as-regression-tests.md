# ADR-70865: Expose sandbox.mcp.trusted-bots in the Schema and Compile Reference Doc Examples as Regression Tests

**Date**: 2026-10-21
**Status**: Draft
**Deciders**: Not recorded; acceptance remains pending maintainer review.

---

### Context

The [MCP gateway reference](../src/content/docs/reference/mcp-gateway.md) and the workflow validation contract had diverged. The compiler already extracts `sandbox.mcp.trusted-bots` and emits `trustedBots` into the gateway configuration, but the closed `sandbox.mcp` object in the [canonical workflow schema](../../pkg/parser/schemas/main_workflow_schema.json) did not expose that property. The OIDC example also needed root-level `mcp-servers`, `id-token: write`, and an explicit network allowlist. These examples need to exercise the same validation and compilation path as workflow files, rather than remain unchecked YAML fragments.

Exposing the trusted-bot field also requires a bounded input contract. The generated configuration passes through a shell heredoc; JSON string encoding alone does not prevent shell expansion. Accepting arbitrary nonempty strings would therefore be insufficient. This ADR records the proposed contract and test strategy for [PR #70865](https://github.com/github/gh-aw/pull/70865), not maintainer approval or evidence of successful execution.

### Decision

Expose `sandbox.mcp.trusted-bots` as an optional, nonempty array of validated literal GitHub-style identities, such as `github-actions[bot]`. Require ASCII letters, digits, hyphens, or underscores, optionally followed by `[bot]`, in both schema validation and the renderer. Reject malformed identities, whitespace-only entries, shell substitutions, and template expressions rather than trimming or evaluating them. Remove the unreachable camelCase extraction alias: frontmatter accepts only `trusted-bots`, while generated gateway JSON uses `trustedBots`. These additional identities are merged by the gateway with its built-in list; they neither remove nor override built-in identities.

Keep the two frontmatter examples as complete, titled `aw` blocks in the reference page. The [reference regression test](../../pkg/workflow/mcp_gateway_reference_test.go) extracts those exact blocks, compiles them with the real compiler, and checks generated lock-file substrings for trusted-bot configuration and OIDC configuration, permission, and environment forwarding. [Schema regression tests](../../pkg/parser/schema_trusted_bots_test.go) cover the input contract separately. This keeps published examples, schema validation, and compiler output connected without introducing a general documentation-test framework.

### Alternatives Considered

#### Alternative 1: Fix only the documentation text and schema, without tests

This has the lowest immediate maintenance cost, but leaves no direct regression guard connecting the published examples to compiler behavior.

#### Alternative 2: Keep duplicate copies of the examples as fixture files under `pkg/`

Separate fixtures avoid parsing Markdown, but duplicate the examples and can pass while the published version breaks. Extracting the published blocks avoids that duplication.

#### Alternative 3: Remove `trusted-bots` from the docs instead of allowing it in the schema

Removal avoids expanding the schema but hides an existing compiler capability. A constrained schema entry makes the supported input explicit instead.

#### Alternative 4: Accept arbitrary nonempty identity strings

This matches the initial draft's broad string contract but permits values unsafe for the generated shell context. Strict literal identity validation is preferable to exposing arbitrary strings or treating JSON escaping as shell protection.

### Consequences

#### Positive
- Valid literal bot identities have an explicit schema contract; invalid shapes and unsafe identity values are intended to fail before configuration generation.
- Compiling the published blocks provides a focused regression signal for schema or compiler changes affecting these examples.
- Complete workflow structure makes the examples less ambiguous. The OIDC server URL and audience remain placeholders requiring a real server and its trust configuration.

#### Negative
- The test depends on the exact fence string (`` ```aw wrap title=".github/workflows/<name>.md" ``) and the docs file path. Editing the docs layout or fence attributes can break the test even when behavior is unchanged.
- Unit tests now read docs content, which mixes documentation and Go test concerns. Other contributors may not expect this coupling.
- Strict validation intentionally rejects arbitrary strings previously accepted by extraction code, including values that are not literal bot identities.

#### Neutral
- The test asserts on substrings of the generated lock file, so changes to lock file formatting may require updating the expected strings.
- Coverage is limited to these two examples. Compilation does not verify live OIDC authentication, remote-server authorization, gateway identity merging at runtime, or other snippets.
- Test definitions and automated completion notices are not evidence that checks passed or changes were published. Validation results must be reported separately for the final revision.
- This decision does not add `tools.github.refusal-labels` or change concurrency behavior.

---

*Draft retained pending maintainer acceptance; no decider or approval is inferred from automation.*
