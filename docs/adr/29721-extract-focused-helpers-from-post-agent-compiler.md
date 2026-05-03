# ADR-29721: Extract Focused Helpers from Post-Agent Workflow Compiler Function

**Date**: 2026-05-02
**Status**: Draft
**Deciders**: pelikhan (copilot-swe-agent)

---

## Part 1 — Narrative (Human-Friendly)

### Context

The `generatePostAgentCollectionAndUpload` function in `pkg/workflow/compiler_yaml_main_job.go` had grown to 203 lines handling over 10 unrelated concerns: artifact path collection (MCP logs, DIFC proxy, OTLP spans, safe-outputs, patches, firewall audit paths), `GITHUB_STEP_SUMMARY` log-parsing step emission, engine cleanup, access log upload, safe-outputs staging, token invalidation, dev-mode restore, and step-order validation. This violated the Single Responsibility Principle and made it impossible to unit-test individual concerns (e.g., path collection logic) without exercising the entire post-agent pipeline.

### Decision

We will decompose `generatePostAgentCollectionAndUpload` into three clearly scoped units: a pure `collectArtifactPaths` helper that returns all paths for the unified artifact upload without touching the YAML builder, a `generateSummarySteps` helper that emits all `GITHUB_STEP_SUMMARY` log-parsing steps, and a reduced `generatePostAgentCollectionAndUpload` orchestrator (~97 lines) that delegates to both helpers and retains only the remaining YAML writes. The primary driver is testability and maintainability: `collectArtifactPaths` is now a pure function (no side effects on the YAML builder) that can be unit-tested in isolation.

### Alternatives Considered

#### Alternative 1: Keep the monolithic function

Leave `generatePostAgentCollectionAndUpload` as a single function and rely on comments to delineate logical sections. This was rejected because the function had already become difficult to reason about incrementally — the interleaving of path-collection assignments and YAML step emissions made it hard to follow the data flow, and the comment-only boundary offered no enforcement. Future additions would have continued growing the function.

#### Alternative 2: Extract into more than three units (fine-grained decomposition)

Further split into per-concern functions (e.g., separate helpers for firewall paths, safe-outputs paths, MCP paths). This was not chosen because the natural seams in the existing code are two: data collection (no side effects) vs. step emission (YAML side effects). Adding more helpers beyond those two seams would have created unnecessary call-graph complexity without proportionate testability benefit for this refactor.

### Consequences

#### Positive
- `collectArtifactPaths` is a pure function with no YAML builder side effects, enabling direct unit testing of path-collection logic.
- The orchestrator function is reduced from 203 to ~97 lines, making the post-agent pipeline easier to read and modify.
- `generateSummarySteps` groups all `GITHUB_STEP_SUMMARY`-related step emissions in one place, making it easier to add or audit summary steps.

#### Negative
- Introduces two additional exported-equivalent methods on `Compiler`, increasing the surface that future contributors must be aware of when modifying post-agent behavior.
- The refactor slightly changes the call order relative to the original function (engine cleanup now precedes access log extraction rather than being interleaved with path collection), which requires careful review to confirm no behavioral change.

#### Neutral
- All compiled lock files recompile identically — the refactor is purely structural with no behavioral change.
- New contributors must understand that `generatePostAgentCollectionAndUpload` is now an orchestrator and that the two helpers must be called together to reproduce the original behavior.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Compiler Function Structure

1. Implementations **MUST** keep `collectArtifactPaths` free of side effects on the YAML builder; it **MUST** only read `WorkflowData`, `CodingAgentEngine`, and the initial paths slice, and **MUST** return the augmented paths slice.
2. Implementations **MUST NOT** add `GITHUB_STEP_SUMMARY` step-emission calls to `collectArtifactPaths`; all such calls **MUST** reside in `generateSummarySteps` or `generatePostAgentCollectionAndUpload`.
3. Implementations **MUST NOT** add artifact path accumulation (`paths = append(paths, ...)`) directly inside `generatePostAgentCollectionAndUpload`; new paths **MUST** be added inside `collectArtifactPaths`.
4. Implementations **SHOULD** call `collectArtifactPaths` and `generateSummarySteps` exclusively via `generatePostAgentCollectionAndUpload` to preserve the correct step ordering in the emitted workflow YAML.

### Post-Agent Step Ordering

1. The orchestrator **MUST** invoke engine output cleanup before calling `collectArtifactPaths` so that engine-declared output paths are still present on disk when collected.
2. The orchestrator **MUST** invoke `generateSummarySteps` after `collectArtifactPaths` so that all paths are resolved before any step-summary steps reference them.
3. The orchestrator **MUST** emit the unified artifact upload step after all path-collection and step-summary emission, following the ordering established in this refactor.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25249422020) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
