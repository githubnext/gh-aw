# ADR-41801: Use Raw Value Writing for MCP Server env/headers Sections in Lock Files

**Date**: 2026-06-26
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

Custom MCP server configurations in workflow lock files are emitted as JSON inside unquoted bash heredocs. When the workflow compiler processes template expressions like `${{ secrets.MY_TOKEN }}`, `ReplaceTemplateExpressionsWithEnvVars` converts them to shell-safe placeholders (e.g., `\${MY_TOKEN}` — one backslash). Since v0.81.2, the shared `writeJSONStringMapSection` helper serializes these values through `json.Marshal`, which re-escapes the backslash to produce `\\${MY_TOKEN}`. When bash processes the heredoc, `\\` becomes `\`, so the MCP gateway receives `\<secret-value>` — an invalid JSON escape — causing startup failures with `Configuration is not valid JSON` / `Bad escaped character in JSON`. The fix must preserve the single-backslash form that `ReplaceTemplateExpressionsWithEnvVars` already produced, without altering the upstream substitution logic or the heredoc template.

### Decision

We will introduce `writeJSONStringMapEntriesRaw` and `writeJSONStringMapSectionRaw` helpers that write map values verbatim inside quotes (bypassing `json.Marshal`) while still JSON-encoding keys via `mustMarshalJSONString`. The `renderSharedMCPConfig` function will use these new helpers for both the `env` and `headers` sections of custom MCP server lock file output, since both sections live inside unquoted heredocs and their values are already pre-escaped shell placeholders.

### Alternatives Considered

#### Alternative 1: Unescape Values Before Passing to json.Marshal

Strip the backslash from `\${VAR}` placeholders before passing them to the existing `writeJSONStringMapSection` (which calls `json.Marshal`). `json.Marshal` would then produce `${VAR}` with no backslash, and bash heredoc processing would expand `${VAR}` as a shell variable — breaking secret injection and requiring invasive changes to the heredoc quoting strategy.

#### Alternative 2: Post-Process Output to Revert Double Escaping

Apply a string-replacement pass after `writeJSONStringMapSection` to convert `\\${` back to `\${`. This is fragile: it operates on raw YAML/JSON text, is hard to scope safely, and could silently affect unrelated sections or break when value formats change in the future.

### Consequences

#### Positive
- Eliminates the double-escape regression that caused MCP Gateway startup failures for custom MCP servers with env or header secrets.
- The fix is precisely scoped to the affected `env` and `headers` sections; all other JSON map sections continue to use the `json.Marshal`-based helper.
- Regression tests at both the helper level (`TestWriteJSONStringMapSectionRaw`, `TestWriteJSONStringMapSectionRawDoesNotDoubleEscapeShellPlaceholders`) and end-to-end compilation level (`TestCustomMCPEnvSecretSingleEscape`) guard against reintroduction.

#### Negative
- `writeJSONStringMapSectionRaw` bypasses `json.Marshal` for values, placing full responsibility on the caller to ensure values are in their final, heredoc-safe form. Any future caller passing un-escaped or untrusted values through this helper will silently produce syntactically invalid JSON.
- The codebase now has two parallel JSON-map writing helpers with subtly different semantics, increasing cognitive overhead and the risk of choosing the wrong helper at a new call site.

#### Neutral
- Keys are still encoded via `mustMarshalJSONString`, so key-injection attacks are not introduced by this change.
- No changes are required to the upstream template expression substitution logic (`ReplaceTemplateExpressionsWithEnvVars` / `ReplaceSecretsWithEnvVars`) or the bash heredoc template structure.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
