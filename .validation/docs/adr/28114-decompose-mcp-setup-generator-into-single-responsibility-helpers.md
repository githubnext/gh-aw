# ADR-28114: Decompose MCP Setup Generator into Single-Responsibility Helpers

**Date**: 2026-04-23
**Status**: Draft
**Deciders**: pelikhan

---

## Part 1 — Narrative (Human-Friendly)

### Context

`pkg/workflow/mcp_setup_generator.go` contains a single orchestration function, `generateMCPSetup`, that had grown to approximately 852 lines. It handled MCP tool discovery, conditional safe-outputs config generation, agentic-workflows install step emission, safe-outputs setup, mcp-scripts setup, and MCP gateway step/env/export/container-command assembly — all inlined within one function body. The Architecture Guardian flagged this as a critical function-size violation. The function's size made individual concerns hard to navigate, test in isolation, or extend without risk of unintended side effects on adjacent logic.

### Decision

We will decompose `generateMCPSetup` into a thin coordinator that delegates each distinct concern to a focused helper function. `generateMCPSetup` itself becomes a sequencing orchestrator that calls `collectMCPTools`, `generateSafeOutputsConfigIfEnabled`, `generateAgenticWorkflowsInstallStep`, `generateSafeOutputsSetup`, `generateMCPScriptsSetup`, and `generateMCPGatewaySetup`. All helpers remain in `mcp_setup_generator.go`; no new packages are introduced. External behavior and output structure are preserved identically.

### Alternatives Considered

#### Alternative 1: Retain the Monolithic Function with Inline Comments

Add section comments and godoc cross-references to document the structure of `generateMCPSetup` without splitting it. This approach has zero structural diff risk. It was rejected because documentation describes structure but does not enforce it — the function remains a single unit that cannot be tested or reviewed by concern, and the comments will drift as the function evolves. The root problem is co-mingled responsibilities, which only decomposition fixes.

#### Alternative 2: Extract MCP Setup Concerns into Separate Files or a Sub-Package

Move each concern into a dedicated file (e.g., `mcp_tool_collector.go`, `mcp_gateway_setup.go`) or a new `pkg/workflow/mcpsetup` sub-package. This maximises navigability at the directory level. It was not chosen because the helpers are small and tightly coupled to `WorkflowData` and `Compiler` types that live in `pkg/workflow`. A sub-package would add import indirection without meaningful boundary enforcement — all code changes together in the same PR. File-level decomposition within `mcp_setup_generator.go` is sufficient to resolve the function-size violation while keeping locality of reference.

### Consequences

#### Positive
- `generateMCPSetup` is now a readable four-line coordinator; the intent of the setup sequence is visible at a glance.
- Each helper (`collectMCPTools`, `generateMCPGatewaySetup`, etc.) can be reviewed, tested, and extended independently.
- Higher-context error wrapping at orchestration boundaries (`"safe outputs setup preparation failed: %w"`) improves diagnostic clarity when errors surface.
- The deduplication set for forwarded env vars and the gateway command construction are now isolated, reducing the risk of accidental mutation when either concern changes.

#### Negative
- The refactor produces a large diff (470 additions, 628 deletions across a single file) with no functional change, which increases reviewer burden to verify behavioral equivalence.
- Adding function call boundaries introduces a small indirection cost: understanding the full setup sequence now requires reading multiple function signatures rather than one linear flow.

#### Neutral
- All helpers are package-private (`func`, not exported), so no public API surface changes.
- The `hasGhAwSharedImport` helper extracted here may be reused by other generators in `pkg/workflow/` as a shared import-detection utility.
- Existing tests continue to exercise the behavior through `generateMCPSetup`; unit tests targeting individual helpers can be added incrementally.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Function Decomposition

1. `generateMCPSetup` **MUST** act as a coordinator only: it **MUST NOT** contain inline logic for tool discovery, config generation, or YAML emission beyond delegating to named helpers.
2. Each concern in the MCP setup pipeline (tool collection, safe-outputs config, install step emission, gateway assembly) **MUST** be implemented in a dedicated helper function with a name that describes its single responsibility.
3. New logic added to the MCP setup pipeline **MUST** be placed in an appropriately scoped helper rather than inlined into `generateMCPSetup`.

### File Ownership

1. All MCP setup helpers **MUST** reside in `pkg/workflow/mcp_setup_generator.go` unless a helper is broadly reusable across the `pkg/workflow` package, in which case it **SHOULD** be moved to a shared file within the same package.
2. MCP setup helpers **MUST NOT** be placed in a separate sub-package unless the sub-package is justified by an independent ADR.

### Error Handling

1. Errors returned from helpers called within `generateMCPSetup` **MUST** be wrapped with a context phrase that identifies the failing phase (e.g., `"safe outputs setup preparation failed: %w"`).
2. Helper functions **SHOULD NOT** wrap errors internally when the caller can provide richer context.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24844753173) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
