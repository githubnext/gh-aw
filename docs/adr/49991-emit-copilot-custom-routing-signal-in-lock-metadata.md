# ADR-49991: Emit an authoritative Copilot custom-routing signal in lock file metadata

**Date**: 2026-08-03
**Status**: Draft
**Deciders**: pelikhan (PR author, copilot-swe-agent)

---

### Context

The compiled lock file metadata (`# gh-aw-metadata:`) already records agent identity, schema version, strict mode, and engine versions. However, it did not capture whether the Copilot engine was running with default GitHub-hosted routing or a custom provider/base URL (BYOK mode, a non-GitHub `engine.model-provider`, or an explicit `engine.api-target`).

Downstream consumers — telemetry pipelines, dashboards, reporting tools — need to classify runs as "default Copilot" vs "custom-routed Copilot." The only existing approach was to infer this from the compiled step `env:` blocks, which has three concrete failure modes: (1) `engine.api-target` is embedded inside the AWF config JSON in the `run:` script and is invisible in `env:`, (2) `GITHUB_COPILOT_BASE_URL` is injected into the DIFC integrity-proxy step even for default configurations, producing false positives, and (3) any env-based inference is silently coupled to the compiled step structure and can drift when the compiler changes.

The compiler already derives the authoritative answer at compile time via internal predicates. Surfacing it once in the metadata eliminates the fragile inference layer entirely.

### Decision

We will add a boolean field `engine_base_url_customized` (with `omitempty`) to `LockMetadata` and `AgentMetadataInfo`, populate it in `generateWorkflowHeader` for Copilot agents by calling a new shared predicate `isCopilotCustomConfig`, and extract the existing inline BYOK predicate into a reusable `isCopilotBYOKMode` helper so that runtime execution and emitted metadata are driven by the same logic.

`isCopilotCustomConfig` returns `true` when either BYOK routing is active (`isCopilotBYOKMode`: non-GitHub `model-provider` + sandbox enabled, or `COPILOT_PROVIDER_BASE_URL` in `engine.env`) or `GetCopilotAPITarget` returns a non-empty string (covers `engine.api-target` and a literal `GITHUB_COPILOT_BASE_URL` override). The field is omitted (`false`) from the JSON output for default configurations because the `bool` zero-value is `false` and the tag is `omitempty`.

### Alternatives Considered

#### Alternative 1: Infer routing mode from compiled step env at read time

Downstream consumers scan each compiled step's `env:` block for `COPILOT_PROVIDER_BASE_URL`, `GITHUB_COPILOT_BASE_URL`, and related keys to classify the run. Why not chosen: `engine.api-target` is not present in any step env (it lives inside the AWF config JSON string in the `run:` block), `GITHUB_COPILOT_BASE_URL` appears in DIFC proxy steps even in default configurations (false positives), and the inference logic is tightly coupled to step IDs and key placement in compiler output — a silent regression risk on every compiler refactor.

#### Alternative 2: Bump the lock file schema version alongside the new field

Introduce `LockSchemaV5` to signal to consumers that the new field is present. Why not chosen: the field is purely additive and uses `omitempty`, so older consumers that do not recognise it ignore it without any schema version signal. A version bump would impose a schema-version negotiation requirement on all consumers for what is a backward-compatible, optional extension. The field's `omitempty` absence in default-config lock files provides enough information to distinguish "field not present" (old compiler) from "field present and false" (impossible — `omitempty` suppresses it) without a version bump.

### Consequences

#### Positive
- Downstream consumers get a single drift-free, authoritative boolean covering all custom-routing modes (BYOK via model-provider gateway, BYOK via `COPILOT_PROVIDER_BASE_URL`, explicit `engine.api-target`, literal `GITHUB_COPILOT_BASE_URL` override).
- Runtime BYOK behavior and emitted metadata are now derived from the same `isCopilotBYOKMode` predicate, eliminating a class of behavioral/observability drift bugs.
- The field is backward-compatible: existing lock file consumers ignore the new key; existing lock files compiled without this change simply omit the field.

#### Negative
- The single boolean does not distinguish *which* form of customization is in use; consumers that need to distinguish BYOK-via-provider-gateway from api-target overrides still need to parse step env or AWF config JSON.
- `engine_base_url_customized` uses `omitempty` on a `bool`, so a `false` value is indistinguishable from "field absent" in JSON — consumers cannot tell whether the compiler that produced the lock file supports the field. A future truthiness migration (if the field ever needs to be `false`-with-meaning) would require a schema bump.

#### Neutral
- Existing lock files checked into the repository (`.github/workflows/*.lock.yml`) for custom-configured Copilot workflows are updated in the same PR to add `engine_base_url_customized: true`, making the repository state consistent with the new compiler output.
- No schema version was bumped; the lock file format remains `v4`.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
