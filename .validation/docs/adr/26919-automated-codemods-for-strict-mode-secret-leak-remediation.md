# ADR-26919: Automated Codemods for Strict-Mode Secret Leak Remediation

**Date**: 2026-04-17
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

With the introduction of strict mode for `gh-aw` workflows (see ADR-0002), two common authoring patterns became non-conformant: (1) secrets interpolated directly into step `run:` commands as `${{ secrets.NAME }}`, and (2) secret-bearing entries in `engine.env` that are not required engine credential overrides. Both patterns cause recurring `aw-compat` CI failures because the validator rejects them under strict mode. Authors need a way to automatically migrate existing workflow files to the compliant form without manually editing every affected file. The `gh aw fix --write` command already provides a codemod registry for automated workflow migrations; extending it is the natural path to zero-friction remediation.

### Decision

We will add two new codemods to the `gh aw fix` codemod registry. The first codemod (`steps-run-secrets-to-env`) rewrites inline `${{ secrets.NAME }}` expressions in step `run:` fields to shell environment variable references (`$NAME`) and injects the corresponding `NAME: ${{ secrets.NAME }}` binding into the step's `env:` block, creating one if absent. The second codemod (`engine-env-secrets-to-engine-config`) removes secret-bearing `engine.env` entries that are not on the allowlist of required engine credential overrides, and prunes any `engine.env` block left empty by those removals. Both codemods use text-based (line-by-line) YAML transformation to preserve original formatting.

### Alternatives Considered

#### Alternative 1: Emit Only Errors, Require Manual Migration

The validator already reports which lines violate strict mode. Requiring authors to fix violations manually, without providing an automated path, was the status quo. It was rejected because the same patterns appear across many workflow files, making manual remediation costly and error-prone, and because the codemod framework exists specifically to absorb this class of migration burden.

#### Alternative 2: Full YAML Parse-and-Serialize for Transformation

Transforming by parsing the YAML into a structured representation and re-serializing would be more robust against edge cases (e.g., unusual indentation, multi-line scalars). It was rejected because full round-trip YAML serialization destroys formatting, comments, and non-standard quoting that authors rely on. Every other codemod in the registry uses the same line-based approach, so consistency and formatting preservation outweigh the robustness gain.

#### Alternative 3: Mask or Remove Secrets Instead of Injecting env: Bindings

For step `run:` secrets, an alternative is to replace the `${{ secrets.NAME }}` expression with a fixed mask string (e.g., `***`) or remove the expression entirely. This was rejected because it produces non-functional workflows: the `run:` command needs the credential at runtime. The `env:` injection strategy produces a fully functional, compliant workflow by moving the secret to a location the runner safely masks.

#### Alternative 4: Migrate engine.env Secrets to engine.config Instead of Removing

Rather than removing unsafe `engine.env` keys, they could be moved to a different engine configuration location. This was rejected because there is no structural equivalent of `engine.env` that is both strict-mode-safe and semantically correct for arbitrary user-supplied secrets. Allowed engine credential overrides are a defined, finite set; everything else in `engine.env` that carries a secret is an authoring mistake rather than a value that belongs somewhere else.

### Consequences

#### Positive
- Authors can run `gh aw fix --write` to automatically remediate both classes of strict-mode secret leak with no manual edits.
- The transformation produces fully functional workflows: step `run:` commands continue to work via shell environment variables; engine configuration is consistent with the allowlist.
- Formatting, comments, and indentation of the original YAML are preserved, minimizing diff noise.
- Both codemods are version-stamped (`IntroducedIn: 0.26.0`) so the registry can report when they were added.

#### Negative
- Line-based transformation can mishandle pathological YAML (e.g., multi-line block scalars with atypical indentation). Edge cases not covered by tests may produce incorrect output.
- The `engine.env` codemod silently removes keys without migrating their values elsewhere. If a removed key was intentional, the author must restore the credential through a different mechanism.
- The allowlist for required engine credential overrides is computed at codemod execution time via `getSecretRequirementsForEngine()`; if that function's output changes, previously safe keys may become removable (or vice versa).

#### Neutral
- Both codemods are appended to the existing ordered registry in `GetAllCodemods()`. The registry order matters for codemods that may interact; these two are appended after existing codemods and have no known ordering dependencies.
- Tests for the new codemods follow the established pattern of table-driven before/after string comparisons, consistent with the rest of the codemod test suite.
- The `removeEmptyEngineEnvBlock` cleanup pass runs unconditionally after key removal, regardless of whether any keys were actually removed; callers must check the `modified` flag from the prior step before invoking it.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### steps-run-secrets-to-env Codemod

1. Implementations **MUST** scan the `pre-steps`, `steps`, `post-steps`, and `pre-agent-steps` sections of a workflow frontmatter for step entries that contain `${{ secrets.NAME }}` expressions in their `run:` field.
2. For each matched secret expression in a `run:` field, implementations **MUST** replace the expression with the shell variable reference `$NAME` (where `NAME` is the secret identifier captured from the expression).
3. Implementations **MUST** add a `NAME: ${{ secrets.NAME }}` binding to the step's `env:` block for each secret replaced, merging into an existing `env:` block if one is present, or inserting a new `env:` block immediately before the `run:` field if none exists.
4. Implementations **MUST NOT** add a duplicate `env:` binding for a secret name that is already present as a key in the step's existing `env:` block.
5. Implementations **MUST** handle multi-line block scalar `run:` values (indicated by `|`, `|-`, `>`, or `>-` scalar indicators) by replacing secret expressions within each continuation line.
6. Implementations **MUST NOT** modify workflow files that contain no `pre-steps`, `steps`, `post-steps`, or `pre-agent-steps` keys in their frontmatter.
7. Implementations **SHOULD** deduplicate env bindings when the same secret name appears multiple times within a single `run:` field.

### engine-env-secrets-to-engine-config Codemod

1. Implementations **MUST** inspect the `engine.env` map of a workflow frontmatter and identify entries whose values contain one or more `${{ secrets.* }}` expressions.
2. Implementations **MUST** compute the allowlist of required engine credential override key names by calling `getSecretRequirementsForEngine()` with `includeSystemSecrets=false` and `includeOptional=false` for the engine identified in the frontmatter.
3. Implementations **MUST** remove any `engine.env` key that (a) contains a secret expression and (b) is not present in the allowlist computed in requirement 2.
4. Implementations **MUST NOT** remove `engine.env` keys that contain secret expressions if those keys are in the computed allowlist.
5. Implementations **MUST NOT** remove `engine.env` keys whose values do not contain any secret expression, regardless of allowlist membership.
6. Implementations **MUST** remove the `env:` block from `engine:` if, after removing unsafe keys, the block contains no remaining entries.
7. Implementations **MUST NOT** modify workflow files that have no `engine` key or no `engine.env` block in their frontmatter.

### Codemod Registry Integration

1. Both codemods **MUST** be registered in `GetAllCodemods()` with stable, unique `ID` values (`steps-run-secrets-to-env` and `engine-env-secrets-to-engine-config` respectively).
2. Each codemod **MUST** declare an `IntroducedIn` version string reflecting the `gh-aw` CLI version in which the codemod was first available.
3. Codemods **MUST** return `(content, false, nil)` when no transformation applies to a given file, leaving the content unchanged.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance. In particular: silently dropping secrets from `run:` without injecting the corresponding `env:` binding, and removing allowlisted engine credential override keys, are both non-conformant behaviors.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24581919524) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
