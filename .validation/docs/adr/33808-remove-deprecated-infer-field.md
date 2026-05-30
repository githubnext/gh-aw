# ADR-33808: Remove Deprecated `infer` Frontmatter Field

**Date**: 2026-05-21
**Status**: Draft
**Deciders**: Unknown

---

## Part 1 — Narrative (Human-Friendly)

### Context

The `gh-aw` workflow frontmatter previously supported a top-level boolean field `infer` that controlled whether the agent could be invoked by an external model. It was superseded by `disable-model-invocation`, which is the semantic inverse (`infer: false` ≡ `disable-model-invocation: true`). For multiple releases both fields were accepted, with `infer` flagged as deprecated via a schema-driven warning. Carrying two names for the same concept added documentation noise, complicated schema validation, and forced every reader of a workflow to mentally translate between the two spellings. With `disable-model-invocation` now the documented and tested form, continuing to accept `infer` no longer carries a benefit proportional to the cost of maintaining it.

### Decision

We will remove the `infer` field entirely from the schema, parser allow-list, compiler deprecation warning, and generated documentation, and we will ship a new codemod `infer-to-disable-model-invocation` so that `gh aw fix --write` automatically rewrites `infer: <bool>` into `disable-model-invocation: <!bool>`. The codemod inverts the boolean (since the two fields are semantic inverses), preserves indentation and surrounding frontmatter, and skips files that already declare `disable-model-invocation` to avoid clobbering an explicit choice.

### Alternatives Considered

#### Alternative 1: Keep `infer` Accepted with a Louder Warning

Retain `infer` in the schema and parser, but escalate the deprecation warning to an error in strict mode. This was rejected because the field has already been deprecated for long enough that users have had ample notice, and a warning-only path keeps two spellings of the same concept alive in the codebase and documentation indefinitely. It also leaves the schema, autocomplete data, and generated reference docs cluttered with a duplicate concept.

#### Alternative 2: Remove `infer` Without a Codemod

Drop the field from the schema and parser and ask users to perform the rename manually. This was rejected because the rename is mechanical but error-prone: users must remember to invert the boolean, not just rename the key, and a silent mistake (e.g. `infer: false` rewritten to `disable-model-invocation: false`) would change runtime behavior. A codemod encodes the inversion correctly and makes `gh aw fix --write` a single safe migration step.

### Consequences

#### Positive
- Eliminates the two-spellings-for-one-concept problem in the schema, autocomplete data, and generated reference docs.
- The new codemod makes migration mechanical and inversion-safe; users do not need to remember to flip the boolean by hand.
- Validation surface shrinks: `include_processor.go` no longer needs to allow `infer` in the frontmatter allow-list, and the deprecation warning test is no longer required.

#### Negative
- Workflows that still set `infer` will fail schema validation after upgrade; users who skip running `gh aw fix --write` will see a hard error instead of a warning.
- The codemod only handles boolean values for `infer`. Any non-boolean value (e.g. a string `"false"`) is left in place and will surface as a schema error rather than being auto-migrated.
- Workflows that set both `infer` and `disable-model-invocation` at the same time are intentionally skipped by the codemod; resolving the conflict requires manual intervention.

#### Neutral
- The total codemod count in `GetAllCodemods()` increases by one, which is reflected in the registry count test and ordering assertion.
- Generated artifacts (`autocomplete-data.json`, `frontmatter-full.md`) are regenerated from the updated schema; the diff is large but mechanical.
- `ROOT_SORT_ORDER` in `generate-autocomplete-data.js` loses its `infer` entry; this only affects the deterministic ordering of root keys in generated docs.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Schema and Parser

1. The `main_workflow_schema.json` schema **MUST NOT** declare an `infer` property at the top level.
2. The frontmatter allow-list used by `include_processor.go` **MUST NOT** include `infer`.
3. The compiler **MUST NOT** emit a deprecation warning specifically for the `infer` field; any prior test enforcing that warning **MUST** be removed.
4. The `disable-model-invocation` field description **MUST NOT** contain a back-reference to `infer`.

### Migration Codemod

1. A codemod with ID `infer-to-disable-model-invocation` **MUST** be registered in `GetAllCodemods()` so that it runs as part of `gh aw fix --write`.
2. The codemod **MUST** rewrite a top-level `infer: <bool>` line into `disable-model-invocation: <!bool>`, inverting the boolean value.
3. The codemod **MUST** preserve the original indentation of the rewritten line.
4. The codemod **MUST NOT** modify any file that does not declare a top-level `infer` field.
5. The codemod **MUST NOT** modify any file that already declares a top-level `disable-model-invocation` field; in that case it **MUST** return the content unchanged and **SHOULD** log that both fields were present.
6. The codemod **MUST NOT** modify the file when the value of `infer` is not a boolean; it **MUST** return the content unchanged and **SHOULD** log that the value was not a boolean.
7. The codemod **MUST** leave all content outside the rewritten line, including the markdown body below the frontmatter delimiter, byte-identical.

### Generated Documentation

1. `docs/src/content/docs/reference/frontmatter-full.md` and `docs/public/editor/autocomplete-data.json` **MUST** be regenerated from the updated schema and **MUST NOT** reference the `infer` field.
2. `ROOT_SORT_ORDER` in `docs/scripts/generate-autocomplete-data.js` **MUST NOT** contain `infer`.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/26241589570) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
