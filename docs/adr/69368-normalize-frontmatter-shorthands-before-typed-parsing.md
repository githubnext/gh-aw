# ADR-69368: Normalize Schema-Valid Frontmatter Shorthands Before Typed Parsing

**Date**: 2026-10-20
**Status**: Draft (PR #69368 is not yet accepted)
**Deciders**: Not recorded; pending maintainer review of PR #69368. This draft does not establish approval.
**Author**: Design Decision Gate (generated from PR #69368 evidence)

---

### Context

The compiler has two readers of workflow frontmatter: the JSON schema, which accepts several shorthand spellings, and the typed `FrontmatterConfig` struct, which is populated by `encoding/json` unmarshaling in `ParseFrontmatterConfig`. The two have drifted. Schema-valid forms such as `on: workflow_dispatch` (string instead of map) and `safe-outputs: { noop: { max: 1 } }` (integer instead of the `*string` `Max` field) fail typed unmarshaling. `tryParseFrontmatterConfig` in `pkg/workflow/compiler_orchestrator_tools.go` logged that failure only to the debug logger and returned `nil`, so every field read exclusively from the typed config — notably top-level `excluded-env` — was silently dropped while compilation reported success (issue #69130). The security impact is concrete: variables a workflow explicitly excludes still reached the agent container.

### Decision

We will normalize schema-valid shorthands into a cloned frontmatter map immediately before JSON marshaling in `ParseFrontmatterConfig`, and we will surface any remaining typed-parse failure as a counted compiler warning. `normalizeFrontmatterConfig` in `pkg/workflow/frontmatter_parsing.go` rewrites the known shorthand forms (string `on`, `tools.github: false`, `sandbox.agent: false`, numeric safe-output `max` via the existing `preprocessIntFieldAsString` helper) and copies maps with `maps.Clone` so the raw frontmatter consumed by other compiler paths is never mutated. Inline JSON schemas under `safe-outputs.data` are explicitly skipped because they are user data, not handler configuration. The primary driver is correctness under silent failure: schema acceptance must not diverge from typed behaviour without the user being told.

### Alternatives Considered

#### Alternative 1: Widen the typed structs with custom `UnmarshalJSON`

Give `FrontmatterConfig.On` and `BaseSafeOutputConfig.Max` custom unmarshalers that accept both shapes, following the existing `TemplatableInt32` precedent. This keeps the adaptation next to the type it concerns and needs no pre-pass. It was rejected for this change because each new shorthand would require another bespoke unmarshaler on a different type, scattering the schema/struct reconciliation across the package, and because `tools.github: false` and `sandbox.agent: false` map onto nested types where a boolean-to-struct unmarshaler is more intrusive than a map rewrite. This was a close call.

#### Alternative 2: Fail compilation on typed-parse failure

Issue #69130 proposed that a typed-config parse failure on schema-valid frontmatter should fail the build outright. That is the strongest guarantee against silent field loss. It was rejected as the immediate step because any undiscovered schema/struct drift would turn previously compiling workflows into hard errors, and the blast radius across existing repositories is unknown. The warning plus warning-count increment preserves the signal without the breaking change.

#### Alternative 3: Normalize in place on the original frontmatter map

Mutating the incoming map avoids the clone allocations. Rejected because other compiler paths read the same raw frontmatter and would observe rewritten values; `TestParseFrontmatterConfigShorthands` asserts explicitly that parsing does not mutate its input.

### Consequences

#### Positive
- `excluded-env` and every other typed-config-only field now survive the documented shorthand spellings, closing the silent security-relevant gap in #69130.
- Remaining typed-parse failures are visible on stderr and counted via `IncrementWarningCount`, so future schema/struct drift is noticed rather than absorbed.
- Raw frontmatter stays untouched, so no other compiler path changes behaviour as a side effect of this normalization.

#### Negative
- The shorthand list in `normalizeFrontmatterConfig` is a hand-maintained mirror of the schema. Adding a schema shorthand without updating this function reintroduces the same class of bug, now one warning louder but still not a hard failure.
- `maps.Clone` per parse adds allocations proportional to the frontmatter size on a hot compiler path, in exchange for immutability.
- A warning does not block compilation, so a workflow with a genuinely unparseable typed config still produces a lock file with fields missing.

#### Neutral
- The warning message intentionally omits the raw error value and points users at `DEBUG=workflow:compiler_orchestrator_tools`, keeping potentially sensitive frontmatter content out of default output.
- Normalization is confined to `ParseFrontmatterConfig`; callers see no API change.
- A future move to Alternative 2 (hard failure) remains open and is now cheaper, since the failure site is already instrumented.

---

*ADR drafted by the Design Decision Gate. Review and finalize before changing status from Draft to Accepted.*
