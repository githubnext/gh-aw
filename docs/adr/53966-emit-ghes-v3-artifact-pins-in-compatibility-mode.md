# ADR-53966: Emit Pinned v3 Artifact Actions in GHES Compatibility Mode

**Date**: 2026-08-19
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

GHES instances running versions prior to `@actions/artifact` v2.0.0 (GHES 3.21.x and earlier) cannot use `actions/upload-artifact@v4+` or `actions/download-artifact@v4+` because those versions depend on the newer artifact backend. When gh-aw compiled workflows for GHES targets, the existing GHES compatibility mode continued to emit the latest non-v3 pins (v7 for upload, v8 for download), which still required the v4 artifact backend. This caused compiled workflows to fail at runtime on older GHES instances with a `GHESNotSupportedError` before any agent execution could begin. GHES compatibility mode therefore had no practical effect for the use case it was designed to address.

### Decision

We will make GHES compatibility mode actively emit SHA-pinned v3 artifact action references: `upload-artifact@v3.2.2` (SHA `c6a366c9...`) and `download-artifact@v3.1.0` (SHA `a9bc5e6e...`). The `ghesArtifactPins` lookup table in `pkg/actionpins/resolve.go` is checked before dynamic resolution whenever `PinContext.GHES` is true, bypassing the normal latest-pin logic. The `GHES` flag propagates from the CLI flag or `aw.json` through the compiler, `WorkflowData`, and `PinContext` so every code path that resolves artifact action pins respects the override. This path explicitly supports GHES 3.21.x and earlier; later GHES releases that support the v4 artifact backend should eventually disable compatibility mode.

### Alternatives Considered

#### Alternative 1: Dynamic resolution with a GHES-aware version constraint

The resolver could be configured to cap `upload-artifact` and `download-artifact` to the latest available v3.x tag via the GitHub API rather than hardcoding a specific SHA. This would pick up v3 patch releases automatically.

Not chosen because dynamic resolution adds a network call at compile time and removes the SHA-pinning guarantee that is central to gh-aw's security model. Hardcoded, audited SHAs are the pattern used elsewhere in the codebase and align with the tool's supply-chain integrity goals.

#### Alternative 2: Leave pin selection to the workflow author

Users could manually specify `uses: actions/upload-artifact@v3.2.2` in their workflow source files rather than having the compiler override the version in GHES mode.

Not chosen because it defeats the purpose of the `ghes` compatibility flag: the flag exists precisely so authors do not need to maintain separate workflow files per deployment target. Compiler-managed pinning is the established convention in gh-aw.

### Consequences

#### Positive
- GHES 3.21.x and earlier instances can now successfully run compiled workflows; artifact upload/download steps no longer fail with `GHESNotSupportedError`.
- The existing `aw.json` `ghes: true` and `gh aw compile --ghes` surface area is preserved; no changes to the public API or configuration schema are required.
- SHA-pinned references maintain the same supply-chain integrity guarantee as other pinned actions in the compiled output.

#### Negative
- The v3 artifact actions are deprecated by GitHub; GHES users running in compatibility mode are consuming end-of-life action versions.
- The hardcoded SHAs in `ghesArtifactPins` require a manual code change if the GHES-compatible target versions need to be updated in the future.

#### Neutral
- The `GHES` field is added to both `WorkflowData` and `PinContext`, widening the surface area of those structs slightly.
- `configureGHESCompatibility()` is extracted into its own method and called from both `ParseWorkflowFile` and `CompileWorkflowData` to ensure consistency between the parse and compile paths.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
