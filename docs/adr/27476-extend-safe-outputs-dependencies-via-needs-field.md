# ADR-27476: Extend `safe_outputs` Job Dependencies via `safe-outputs.needs` Field

**Date**: 2026-04-21
**Status**: Draft
**Deciders**: pelikhan

---

## Part 1 — Narrative (Human-Friendly)

### Context

The consolidated `safe_outputs` job in gh-aw workflows had a hardcoded dependency set (`agent`, `activation`, optional `detection`, optional `unlock`). Some workflows need credentials supplied by a user-defined job (e.g., a `secrets_fetcher` that retrieves app IDs and private keys from a vault) before the `safe_outputs` job can run. Because the `safe_outputs` dependency list was not extensible, `needs.<custom-job>.outputs.*` expressions inside `safe-outputs` handler configs (notably `github-app`) referenced jobs that were not declared as upstream dependencies, causing `actionlint` lint failures and runtime breakage.

### Decision

We will add a `safe-outputs.needs` array field to the frontmatter schema that lets workflow authors declare explicit custom upstream jobs for the consolidated `safe_outputs` job. At compile time, the compiler merges these user-declared dependencies with the built-in set and deduplicates them. A validation pass rejects references to unknown jobs and built-in/control job names, providing actionable error messages.

### Alternatives Considered

#### Alternative 1: Rely Solely on Environment-Scoped Secrets

Workflows could store credentials in GitHub environment secrets, accessible to the `safe_outputs` job without any custom upstream job. This avoids the dependency wiring problem entirely. It was not chosen because it requires credentials to be pre-registered in the GitHub environment UI and cannot dynamically fetch or rotate secrets at workflow runtime — a real constraint for vault-backed or short-lived credentials.

#### Alternative 2: Auto-Detect Dependencies from Expression References

The compiler could parse expression strings inside `safe-outputs` config values (e.g., `${{ needs.secrets_fetcher.outputs.app_id }}`) and automatically infer the required `needs` entries. This approach was not chosen because parsing arbitrary GitHub Actions expression syntax inside YAML values is fragile, error-prone, and creates a tight coupling between the expression evaluator and the compiler. An explicit declaration (`safe-outputs.needs`) is simpler to validate, easier to understand, and immune to expression parsing edge cases.

### Consequences

#### Positive
- Unlocks the credential-fetcher pattern: workflows can now supply dynamic, vault-retrieved credentials to `safe_outputs` handlers such as `github-app`.
- Eliminates `actionlint` failures caused by undeclared `needs.*` references in the compiled GitHub Actions workflow.
- The deduplication logic prevents duplicate dependency entries regardless of how `needs` is populated (built-in vs. user-declared vs. imported).

#### Negative
- Authors must know which job names are reserved (built-in control jobs) and cannot accidentally use them as custom dependencies; this adds cognitive overhead and requires consulting documentation or error messages.
- Import merge behavior for `safe-outputs.needs` must be maintained in sync with other merge fields as the import system evolves.

#### Neutral
- A new validation function (`validateSafeOutputsNeeds`) is added to the compiler pipeline, following the same structure as the existing `validateSafeJobNeeds` validator.
- The schema change (`main_workflow_schema.json`) affects all schema-validation tooling and IDE integrations that consume the schema.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Dependency Declaration

1. Workflow authors **MAY** declare additional upstream job dependencies for the consolidated `safe_outputs` job using the `safe-outputs.needs` array field in workflow frontmatter.
2. Each entry in `safe-outputs.needs` **MUST** be a string matching the pattern `^[a-zA-Z_][a-zA-Z0-9_-]*$`.
3. The `safe-outputs.needs` array **MUST** have `uniqueItems: true` enforced at the schema level; duplicate entries **SHOULD** be deduplicated silently at compile time.

### Compiler Behavior

1. The compiler **MUST** merge `safe-outputs.needs` entries with the built-in dependency set (`agent`, `activation`, and optionally `detection` and `unlock`) before writing the compiled `safe_outputs` job's `needs` list.
2. The compiler **MUST** deduplicate the final `needs` list so that no job name appears more than once.
3. The compiler **MUST** validate `safe-outputs.needs` entries before generating the compiled workflow and **MUST** reject invalid entries with an actionable error message.

### Validation Rules

1. A `safe-outputs.needs` entry **MUST NOT** reference a built-in or control job name (`agent`, `activation`, `pre_activation`, `pre-activation`, `conclusion`, `safe_outputs`, `safe-outputs`, `detection`, `unlock`, `push_repo_memory`, `update_cache_memory`).
2. A `safe-outputs.needs` entry **MUST NOT** reference a job name that is not declared in the workflow's top-level `jobs:` map.
3. When a violation of rules 1 or 2 is detected, the compiler **MUST** emit a compile-time error and **MUST NOT** produce a compiled workflow artifact.

### Import Merge Behavior

1. When a workflow imports another workflow, the importer's `safe-outputs.needs` **MUST** be merged with the imported workflow's `safe-outputs.needs` as a deduplicated union.
2. Import merge **MUST NOT** introduce duplicate entries into the merged `needs` list.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24701927396) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
