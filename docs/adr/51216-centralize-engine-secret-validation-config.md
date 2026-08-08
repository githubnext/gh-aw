# ADR-51216: Centralize Engine Secret Validation via a Shared Config Helper

**Date**: 2026-08-07
**Status**: Draft
**Deciders**: Unknown (automated draft — review before accepting)

---

### Context

Each workflow engine in `pkg/workflow` implements a `GetSecretValidationStep` method. By PR #51216, at least seven implementations (Claude, Codex, Copilot, Gemini, Pi, behavior-defined, and the universal LLM consumer engine) each repeated the same guard-then-delegate pattern: check an optional skip predicate, check for an empty secret list, then call `BuildDefaultSecretValidationStep`. Only the skip condition, engine display name, documentation URL, and secret list source differed per engine. Duplicating this three-part structure in six-plus locations raises the risk that new auth modes (WIF, BYOK, provider-token fallbacks) are applied inconsistently when one engine is updated but others are not.

### Decision

We will introduce `EngineSecretValidationConfig` (a config struct holding `SecretNames`, `EngineName`, `DocsURL`, and an optional `Skip` predicate) and `BuildEngineSecretValidationStep` (a shared helper in `engine_helpers.go`) that applies the skip predicate, guards on an empty secret list, and delegates to `BuildDefaultSecretValidationStep`. All engine `GetSecretValidationStep` implementations will be migrated to call `BuildEngineSecretValidationStep` with an engine-specific config, keeping engine-specific skip logic encapsulated as a `func(*WorkflowData) bool` closure in each engine file.

### Alternatives Considered

#### Alternative 1: Status quo — leave per-engine wrappers unchanged

Keep each engine's explicit if-guard plus `BuildDefaultSecretValidationStep` call as-is. This requires no new types or shared code and is fully transparent at each call site. It was rejected because any future change to the skip-or-delegate pattern (for example, adding a unified logging hook or a new auth mode) must be applied manually across seven-plus engine files, increasing the risk of behavioral drift.

#### Alternative 2: Engine interface with a default validation implementation

Define a new `SecretValidator` interface (or embed a default implementation via struct embedding) on the engine type, moving the validation logic into a shared base. This is a more complete object-oriented approach and would also consolidate other shared engine behaviors. It was rejected as over-engineering for this change: the variation across engines is limited to a single config object, so a lightweight config-and-helper pattern achieves the same consolidation at lower structural cost and with no interface breakage for existing engine implementations.

### Consequences

#### Positive
- Eliminates six-plus instances of the duplicated skip-guard-then-delegate wrapper; the pattern now lives in one function that is unit-tested independently.
- Adding a new engine or a new auth-skip condition (WIF, BYOK, etc.) requires only populating a `Skip` field on the config struct rather than replicating the three-step pattern.
- Engine-specific skip predicates remain in each engine file, preserving locality of domain knowledge.

#### Negative
- `engine_helpers.go` gains a new exported type and function, widening the surface of the shared-helpers module that is already a common dependency.
- Callers must follow one level of indirection (the `Skip` function pointer) to understand when validation is suppressed; the condition is no longer a plain if-statement at the call site.

#### Neutral
- Unit tests for `BuildEngineSecretValidationStep` (skip policy, empty-secret-list guard, rendered-step assertion) are added to `secret_validation_test.go`, independent of per-engine tests.
- The underlying `BuildDefaultSecretValidationStep` function is unchanged; only its callers are updated.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
