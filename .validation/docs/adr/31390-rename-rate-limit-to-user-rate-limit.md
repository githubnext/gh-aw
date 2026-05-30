# ADR-31390: Rename `rate-limit` to `user-rate-limit` with backward-compatible aliases

**Date**: 2026-05-10
**Status**: Draft
**Deciders**: pelikhan (PR author), to be confirmed

---

## Part 1 — Narrative (Human-Friendly)

### Context

The workflow frontmatter has supported a top-level `rate-limit` block (with a nested `max` key) for limiting how often a given user can trigger a workflow. The names were ambiguous: `rate-limit` did not convey that the limit is per *user* (as opposed to repository-, organization-, or token-scoped), and `max` did not convey that the cap is per *time window*. The feature is still flagged experimental, which lowers the migration cost of renaming. Documentation, JSON schema, autocomplete data, compiler types, role checks, and example workflows all reference the old names and must be migrated in lockstep.

### Decision

We will rename the top-level frontmatter key from `rate-limit` to `user-rate-limit`, and rename its nested integer field from `max` to `max-runs-per-window`. The compiler **MUST** continue to accept the legacy `rate-limit` key (and the legacy `max` / `max-runs` nested keys) so existing workflows do not break, but the canonical schema, examples, and warnings reference the new names. A one-shot codemod (`rate-limit-to-user-rate-limit`) rewrites both renames in a single pass for users who run `gh aw fix`.

### Alternatives Considered

#### Alternative 1: Hard rename without backward compatibility

Replace the keys outright and require all consumers to update their workflows on the next CLI bump. Rejected because, even though the feature is experimental, breaking active workflows on upgrade introduces friction and would force every downstream user to perform manual edits before the next compile succeeds. The codemod approach gives the same end state without the breakage.

#### Alternative 2: Keep the old names and improve documentation only

Leave `rate-limit` and `max` in place and clarify their per-user, per-window semantics in the reference docs. Rejected because documentation cannot fix ambiguous field names that appear directly in user-authored YAML; readers seeing `rate-limit:` in a workflow still cannot tell at a glance whether it is per-user, per-repo, or global, and `max:` reads as an absolute cap rather than a windowed one.

#### Alternative 3: Different naming (e.g., `trigger-rate-limit`, `per-user-rate-limit`)

Considered alternative prefixes that emphasize "trigger" or "per-user". Rejected in favor of `user-rate-limit` for brevity and consistency with the existing dashed-key style. `max-runs-per-window` was preferred over shorter alternatives like `max-per-window` because the unit (runs) is informative for someone scanning the file.

### Consequences

#### Positive
- Frontmatter keys are self-describing — readers can infer scope (per-user) and semantics (per-window) without consulting docs.
- A codemod automates the rename for existing workflows, reducing migration toil.
- The experimental warning now matches the canonical name, reinforcing the new vocabulary in CLI output.

#### Negative
- The JSON schema and the compiler now carry two parallel definitions (`user-rate-limit` and a legacy `rate-limit`) and three accepted nested key spellings (`max-runs-per-window`, `max-runs`, `max`), increasing maintenance surface.
- The role-check extraction code has additional branches for resolving the legacy keys, which slightly complicates the parsing path.
- Three name variants for the same field can confuse readers of the codebase until the legacy schema is eventually removed.

#### Neutral
- Reference docs, autocomplete data, embedded example workflows, and shared maintainer-facing workflows all needed parallel edits — any future renames in this area should expect a similarly wide blast radius.
- A future ADR will be needed to drop the legacy `rate-limit` / `max` keys once the feature exits experimental status.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Canonical naming

1. New documentation, examples, autocomplete data, and embedded workflows **MUST** use `user-rate-limit` as the top-level frontmatter key.
2. New documentation, examples, autocomplete data, and embedded workflows **MUST** use `max-runs-per-window` as the nested per-window cap key.
3. The JSON schema **MUST** declare `user-rate-limit` as the canonical object definition, with `max-runs-per-window` listed as the required nested property.
4. The experimental-feature warning emitted by the compiler **MUST** reference `user-rate-limit`, not `rate-limit`.

### Backward compatibility

1. The compiler **MUST** continue to accept the legacy `rate-limit` top-level key and treat it as equivalent to `user-rate-limit` when only one of the two is present.
2. The compiler **MUST** accept the legacy nested keys `max` and `max-runs` and treat each as equivalent to `max-runs-per-window` when the canonical key is absent.
3. The compiler **MUST NOT** silently merge configuration when both `rate-limit` and `user-rate-limit` are present in the same frontmatter; the codemod **MUST** skip such ambiguous documents without modification.
4. The legacy `rate-limit` schema definition **MUST** be marked as a legacy alias in its `description` field so schema readers can identify it as deprecated.

### Codemod behavior

1. The `rate-limit-to-user-rate-limit` codemod **MUST** rename only the top-level `rate-limit` key (not any other key whose name happens to contain `rate-limit`).
2. The codemod **MUST** rewrite the nested `max-runs` or `max` key to `max-runs-per-window` only within the bounds of the `user-rate-limit` block it just renamed (or an existing `user-rate-limit` block); it **MUST NOT** rewrite `max-runs` keys that appear in other frontmatter blocks.
3. The codemod **MUST** be a no-op when the document already uses `user-rate-limit` exclusively, and **MUST** be a no-op when both legacy and canonical keys are present.
4. The codemod **SHOULD** log each rename it performs with the source line number to aid debugging.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25642880879) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
