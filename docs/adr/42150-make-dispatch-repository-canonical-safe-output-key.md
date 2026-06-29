# ADR-42150: Make `dispatch-repository` the Canonical Safe-Output Key

**Date**: 2026-06-29
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The `dispatch_repository` safe-output type allowed agents to trigger `repository_dispatch` events in external repositories. When first introduced, both the underscore form (`dispatch_repository`) and the dashed form (`dispatch-repository`) were accepted at runtime as aliases of each other. This created a mismatch between the runtime behavior and the rest of the safe-output naming convention, where all other types use hyphen-case (e.g., `dispatch-workflow`, `call-workflow`, `create-check-run`). The drift between runtime behavior, the JSON schema, and the public documentation made the contract ambiguous for both users and tooling. A codemod-based migration path allows backward compatibility to be preserved while the canonical key is established.

### Decision

We will make `dispatch-repository` (hyphen) the canonical key for this safe-output type, demote `dispatch_repository` (underscore) to a deprecated backward-compatible alias in the JSON schema and runtime parser, and ship a `safe-output-dispatch-repository-key` codemod that automatically renames existing frontmatter. All documentation, warning messages, and schema definitions are updated to reference only the dashed form. The underscore alias is preserved in the schema as a `$ref` to avoid hard-breaking existing workflows while the codemod propagates.

### Alternatives Considered

#### Alternative 1: Keep `dispatch_repository` (underscore) as canonical

The underscore key could remain primary and the dashed key could become the alias, matching Go struct field conventions. This was rejected because it contradicts the established hyphen-case naming pattern shared by every other safe-output type, and would make the public API feel inconsistent with its siblings.

#### Alternative 2: Hard removal — drop the underscore key with no alias

The underscore key could be removed entirely in a single release with no backward-compatible alias, requiring an immediate breaking migration. This was rejected because it would silently break all existing workflow files using `dispatch_repository` without any automated migration path, violating the project's convention of providing codemods for deprecations.

### Consequences

#### Positive
- Schema, runtime, documentation, and compiler warnings are now consistent: all references use `dispatch-repository`.
- Aligns with every other safe-output type (`dispatch-workflow`, `call-workflow`, `create-check-run`, etc.), reducing cognitive overhead for new users.
- The automated codemod (`safe-output-dispatch-repository-key`) lets existing users upgrade without manual edits.

#### Negative
- The JSON schema retains a `dispatch_repository` `$ref` alias entry, adding a small amount of schema complexity that must be maintained until the alias can be removed.
- Any downstream tooling or documentation outside this repository that hard-codes the underscore key will require an update.

#### Neutral
- The runtime parser lookup order is inverted: `dispatch-repository` is now checked first, then `dispatch_repository` as fallback — behavior is identical for users until the alias is eventually removed.
- Compiler warning text changes from `dispatch_repository` to `dispatch-repository`; any scripts that grep for the old warning string will need updating.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
