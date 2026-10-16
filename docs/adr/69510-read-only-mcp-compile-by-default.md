# ADR-69510: Read-Only MCP `compile` by Default, Writes Gated Behind `--allow-write-tools`

**Date**: 2026-10-16
**Status**: Draft
**Deciders**: pelikhan

---

### Context

`gh aw mcp-server` exposes workflow tools to MCP clients (for example, coding agents). Before this change, `compile` was grouped with the other workflow-writing tools (`add`, `update`, `fix`) and was disabled unless the operator started the server with `--allow-write-tools`. That stopped untrusted or low-privilege clients from even *validating* a workflow, which is the most common reason to call `compile`. At the same time, `--no-emit` was not truly side-effect free. It still updated `.github/aw/actions-lock.json` and `.gitattributes` during post-processing, wrote `.invalid.yml` diagnostic files, and downloaded uncached remote imports into `.github/aw/imports/`. To make validation safe for any MCP client, `--no-emit` has to guarantee that compilation does not touch the filesystem.

### Decision

We will make the MCP `compile` tool available by default in a read-only mode. When `--allow-write-tools` is not set, the server adds `--no-emit --no-check-update`, sets the tool's `ReadOnlyHint` annotation to `true`, skips the actor permission check for compile, and rejects options that write files or run external code (`fix`, `dry-run`, and the external scanners zizmor, poutine, actionlint, runner-guard, syft, grype, grant, and yamllint) with an actionable `InvalidParams` error. We will also make `--no-emit` strictly side-effect free across the compiler. Post-processing (action pins and `.gitattributes`) is skipped, `.invalid.yml` diagnostic writes are suppressed in one shared `writeInvalidWorkflowYAML` helper, and the shared `ImportCache` is set read-only. In read-only mode, cached remote imports resolve normally and uncached imports fail with a "not cached" error instead of being downloaded. Write-enabled compilation keeps its current behavior and still requires the actor permission check.

### Alternatives Considered

#### Alternative 1: Keep `compile` gated behind `--allow-write-tools`

We could have left the previous behavior unchanged. This is the simplest option and has the smallest attack surface. However, MCP clients that lack write trust would still be unable to validate workflows, which is the main use case for agents that author workflows. Operators would have to grant full write capability just to get validation feedback.

#### Alternative 2: Add a separate `validate` MCP tool, leaving `compile` write-only

A dedicated validation tool would separate read and write concerns at the tool level. It was not chosen because it would duplicate the compile argument surface and output schema, and clients would need to learn two tools with overlapping semantics. Making one tool switch behavior based on server capability, and advertising that through `ReadOnlyHint`, keeps the MCP interface smaller.

#### Alternative 3: Allow downloads of uncached remote imports in read-only mode

Validation could have fetched and cached missing remote imports so that more workflows validate successfully. This was rejected because writing to the import cache is a filesystem side effect and a network fetch from a source the client controls. Both conflict with the read-only guarantee. Instead, the error tells users to populate the cache with a trusted compile.

### Consequences

#### Positive
- Any MCP client can validate workflows without the operator granting write capability, which lowers the barrier for agent-driven authoring.
- `--no-emit` now matches its name: no lock files, diagnostic YAML, action pins, import cache entries, or `.gitattributes` are created or changed. An end-to-end test checks this by comparing file content, mode, and modification time before and after compilation.
- The `ReadOnlyHint` annotation reports the tool's actual capability to MCP clients.

#### Negative
- Read-only validation fails for workflows whose remote imports are not cached yet, so the result depends on earlier trusted compiles.
- `fix`, `dry-run`, and security scanners are unavailable without `--allow-write-tools`, so read-only clients get less validation coverage than a full compile.
- `compile` now behaves differently depending on server startup flags. This adds a branch to the permission logic (`compile` is handled separately from `add`/`update`/`fix`) and adds a parameter to `registerCompileTool` and `registerMCPTools`.

#### Neutral
- Skipping post-processing under `--no-emit` also applies to CLI users of `gh aw compile --no-emit`, not only the MCP server.
- Making the import cache read-only is tied to the compiler's `noEmit` flag through `getSharedImportCache`, so any future no-emit caller gets the same behavior.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
