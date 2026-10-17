# ADR-69510: Read-Only MCP `compile` by Default, Writes Gated Behind `--allow-write-tools`

**Date**: 2026-10-16
**Status**: Draft
**Deciders**: pelikhan

---

### Context

`gh aw mcp-server` exposes workflow tools to MCP clients (for example, coding agents). Before this change, `compile` was grouped with the other workflow-writing tools (`add`, `update`, `fix`) and was disabled unless the operator started the server with `--allow-write-tools`. That stopped untrusted or low-privilege clients from even *validating* a workflow. However, `--no-emit` alone was not side-effect free: post-processing updated action pins and `.gitattributes`, failures wrote `.invalid.yml` diagnostics, remote imports populated caches or temporary include files, and container validation could pull Docker images. Trusted CLI callers, including `validate`, maintenance jobs, and label discovery, rely on import-cache population during no-emit compilation. MCP therefore needs a stronger, separate read-only boundary.

### Decision

The MCP `compile` tool is available by default in a dedicated read-only mode. Without `--allow-write-tools`, the server adds an internal read-only flag alongside `--no-emit --no-check-update`, sets `ReadOnlyHint: true`, skips the compile write-actor check, and rejects fixes, dry-run compilation, and external scanners. No-emit suppresses output files, diagnostic YAML, and action-pin/attributes post-processing. The stronger read-only mode additionally prevents import-cache writes, temporary include downloads, and Docker pulls. All compiler include-expansion paths share the repository-root cache, including nested body-level includes; cache misses fail explicitly. Container validation inspects local images and reports missing images without pulling. No-emit validation relies on newly generated YAML rather than stale output files. Ordinary CLI no-emit compilation retains import-cache population, while write-enabled MCP compilation retains emission and actor authorization.

### Alternatives Considered

#### Alternative 1: Keep `compile` gated behind `--allow-write-tools`

We could have left the previous behavior unchanged. This is the simplest option and has the smallest attack surface. However, MCP clients that lack write trust would still be unable to validate workflows, which is the main use case for agents that author workflows. Operators would have to grant full write capability just to get validation feedback.

#### Alternative 2: Add a separate `validate` MCP tool, leaving `compile` write-only

A dedicated validation tool would separate read and write concerns at the tool level. It was not chosen because it would duplicate the compile argument surface and output schema, and clients would need to learn two tools with overlapping semantics. Making one tool switch behavior based on server capability, and advertising that through `ReadOnlyHint`, keeps the MCP interface smaller.

#### Alternative 3: Allow downloads of uncached remote imports in read-only mode

Validation could have fetched and cached missing remote imports so that more workflows validate successfully. This was rejected because writing to the import cache is a filesystem side effect and a network fetch from a source the client controls. Both conflict with the read-only guarantee. Instead, the error tells users to populate the cache with a trusted compile.

#### Alternative 4: Make every no-emit caller use a read-only import cache

This would break trusted CLI validation, maintenance's validate-before-emit sequence, and label discovery when an import is uncached or an upstream branch changes. A separate MCP read-only mode preserves those callers without granting untrusted MCP requests cache-writing capability.

### Consequences

#### Positive
- Any MCP client can validate workflows without the operator granting write capability, which lowers the barrier for agent-driven authoring.
- Read-only MCP compilation creates or changes no lock files, diagnostic YAML, action pins, import-cache entries, temporary includes, or `.gitattributes`, and never pulls Docker images. Repository snapshots, isolated temporary directories, and fake-Docker tests cover these effects.
- The `ReadOnlyHint` annotation reports the tool's actual capability to MCP clients.

#### Negative
- Read-only validation fails for workflows whose remote imports are not cached yet, so the result depends on earlier trusted compiles.
- `fix`, `dry-run`, and security scanners are unavailable without `--allow-write-tools`, so read-only clients get less validation coverage than a full compile.
- Missing local container images cannot be downloaded automatically during read-only validation.
- `compile` now behaves differently depending on server startup flags. This adds a branch to the permission logic (`compile` is handled separately from `add`/`update`/`fix`) and adds a parameter to `registerCompileTool` and `registerMCPTools`.

#### Neutral
- Skipping post-processing under `--no-emit` also applies to CLI users of `gh aw compile --no-emit`, not only the MCP server.
- Import-cache read-only enforcement is tied to the dedicated compiler read-only flag, not `noEmit`; trusted CLI no-emit callers can still populate imports.
- Read-only compilation can still resolve branch/tag refs over the network; it is not an offline mode.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
