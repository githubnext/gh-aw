# ADR-67746: Normalize Unknown Codex Trace Records as Canonical `session.info`

**Date**: 2026-10-11
**Status**: Draft
**Deciders**: Unknown (generated from PR #67746 by the Design Decision Gate)

---

### Context

The Codex engine emits a native JSONL trace whose event vocabulary changes between upstream releases (observed across 0.159.x and the latest stable 0.162.1). The gh-aw session normalizer (`actions/setup/js/codex_session.cjs`) previously forwarded any item it did not recognize as an engine-specific `codex.item_snapshot` event, and the mixed-stream collector in `actions/setup/js/parse_codex_log.cjs` decided whether a line belonged to the Codex stream by re-entering `normalizeCodexSession`. That combination meant unknown native records either produced engine-private event shapes that downstream consumers of the canonical agent-session schema could not interpret, or were dropped from recognized mixed streams and thereby fragmented the session's lifecycle, accounting, and provenance state. The project needs trace parsing to remain forward-compatible with new Codex releases without requiring a parser change for every new native item type.

### Decision

We will represent unrecognized Codex native records as canonical `session.info` observations carrying `sourceType` (the native record's `type`) and `content` (the opaque original record), instead of emitting the engine-specific `codex.item_snapshot` event. We will also decide mixed-stream membership structurally — a parsed JSON record with a non-empty string `type` belongs to the Codex record stream — rather than by asking the Codex normalizer whether it recognizes the record. The primary driver is forward compatibility: unknown-but-typed native events stay inside one session and remain inspectable without inventing new per-engine event shapes. Lifecycle types already handled elsewhere (`thread.started`, `item.started`, `item.updated`, `item.completed`) are explicitly excluded so they are not duplicated as info observations.

### Alternatives Considered

#### Alternative 1: Keep emitting `codex.item_snapshot` for unknown items

Retaining the engine-specific snapshot event would have preserved existing output byte-for-byte and required no changes to the mixed-stream collector. It was rejected because every downstream consumer of the canonical session schema would need Codex-specific handling, and each new upstream item type would keep widening that engine-private surface. Historical logs containing custom snapshots remain readable, which lowered the cost of switching.

#### Alternative 2: Drop unrecognized records entirely

Silently discarding records the normalizer does not understand is the simplest option and guarantees a clean canonical stream. It was rejected because the dropped records are exactly the evidence needed to diagnose new Codex releases, and the previous behaviour of letting recognition drive stream membership already caused unknown lines to be reclassified as text, splitting session state. Preserving the opaque payload costs little and keeps provenance intact.

*(A third option — pinning the parser to a known Codex version and failing on unknown events — was not pursued because version constants and generated lockfiles are coordinator-owned and the parser must tolerate drift.)*

### Consequences

#### Positive
- Unknown native Codex items remain visible in the canonical session stream, so new upstream releases (verified against stable 0.162.1) do not silently lose observations.
- Downstream consumers only need to understand `session.info`; no engine-private event type is required for forward compatibility.
- Mixed Codex streams keep conversation, tool correlation, accounting, error, and retry state in a single session instead of fragmenting on unrecognized lines.

#### Negative
- `session.info` becomes a heterogeneous bucket: consumers must branch on `sourceType` to interpret `content`, and the payload shape is unvalidated.
- The hard-coded exclusion list of lifecycle types (`thread.started`, `item.started`, `item.updated`, `item.completed`) is a maintenance point that must be kept in sync with the normalizer.
- Structural stream detection (any JSON record with a string `type`) is broader than recognition-based detection and could absorb foreign typed JSON lines that happen to be interleaved.

#### Neutral
- Historical traces containing `codex.item_snapshot` remain parseable; this is an emission change, not a schema removal.
- Standalone unsupported JSON still does not select the Codex parser, so parser selection behaviour is unchanged.
- Codex version constants (`DefaultCodexVersion`) and generated lockfiles stay coordinator-owned; the 0.162.1 work in this PR adds compatibility regressions only and does not switch the installer pin.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
