# ADR-28900: Add `wiki` Field to Checkout Configuration

**Date**: 2026-04-28
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

The `checkout` frontmatter object in the agentic workflow system allows users to declare repository checkouts declaratively. GitHub wiki repositories are stored in a separate git repository accessible at `{owner}/{repo}.wiki`, a URL convention that differs from the standard repository URL. Users who need to check out wiki content alongside source code were previously required to know and manually specify this naming convention in the `repository` field, with no first-class support, no validation, and no correct deduplication semantics between wiki and non-wiki checkouts of the same `(repository, path)` pair.

### Decision

We will add a `wiki: bool` field (defaulting to `false`) to `CheckoutConfig`. When `wiki: true`, the system automatically computes the effective repository as `{repository}.wiki` (or `${{ github.repository }}.wiki` when no explicit repository is set). The `wiki` flag is incorporated into the checkout deduplication key (`checkoutKey`), so a wiki checkout and a regular checkout of the same `(repository, path)` remain as distinct steps. A guard in `wikiRepository()` prevents double-suffixing if the user provides a repository already ending in `.wiki`.

### Alternatives Considered

#### Alternative 1: Require users to manually specify `{owner}/{repo}.wiki`

Users could continue to write `repository: owner/repo.wiki` explicitly. No code change is needed. However, this approach offers poor discoverability (users must know the GitHub wiki URL convention), produces no validation error for malformed wiki URLs, and — most critically — cannot distinguish intent: a checkout of `owner/repo.wiki` is structurally indistinguishable from a regular checkout, so deduplication logic cannot correctly treat a wiki checkout and a non-wiki checkout of the same base repository as separate steps. The experience is also error-prone and inconsistent with other declarative fields.

#### Alternative 2: Introduce a distinct `wiki-checkout:` top-level key

A separate top-level key (e.g., `wiki-checkout:`) would make wiki checkout a fully independent concept with its own schema, parser, and step generator. While this provides maximum isolation, it duplicates the entire checkout schema, complicates the step generator with a parallel code path, and breaks the principle that all checkout variants share a unified configuration model. The existing extension pattern (adding a field to `CheckoutConfig`) is simpler and consistent with how other checkout options such as `sparse-checkout`, `lfs`, and `fetch` were added.

### Consequences

#### Positive
- Users express wiki checkout intent declaratively with a single boolean, without needing to know GitHub's wiki URL naming convention.
- The system automatically handles the `.wiki` suffix in all contexts (default checkout override and additional checkout steps), including an idempotency guard against double-suffixing.
- Deduplication is semantically correct: wiki and non-wiki checkouts of the same `(repository, path)` are kept as separate steps.
- The JSON schema gains a `wiki` property, so editors and validators can surface the option with documentation.

#### Negative
- The `checkoutKey` struct now carries a third dimension (`wiki bool`), changing the merge semantics for existing callers who rely on `(repository, path)` being the full deduplication key.
- `GetDefaultCheckoutOverride()` requires a two-lookup fallback (first `wiki=false`, then `wiki=true`) to handle the case where a user sets `wiki: true` at the root level, adding branching to a previously simple function.
- The `wikiRepository()` helper introduces a new function that callers in step generation must invoke correctly; forgetting to call it in future step-generation paths would silently produce incorrect repository strings.

#### Neutral
- All new behavior is gated on `wiki: true`; workflows that do not set `wiki` are unaffected at runtime.
- Tests cover parsing, deduplication, and step generation for both the default and additional checkout paths.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Checkout Configuration Schema

1. The `checkout` configuration object **MUST** support a `wiki` field of type `boolean`.
2. The `wiki` field **MUST** default to `false` when absent.
3. Implementations **MUST** reject a non-boolean value for `wiki` with a descriptive error (e.g., `"checkout.wiki must be a boolean"`).

### Repository Resolution

1. When `wiki` is `true` and no explicit `repository` is set, implementations **MUST** resolve the effective repository to `${{ github.repository }}.wiki`.
2. When `wiki` is `true` and an explicit `repository` is set, implementations **MUST** resolve the effective repository to `{repository}.wiki`.
3. Implementations **MUST NOT** append `.wiki` to a repository string that already ends with `.wiki` (double-suffix guard).
4. When `wiki` is `false` (or absent), implementations **MUST NOT** modify the repository string in any way.

### Deduplication

1. Implementations **MUST** include the `wiki` boolean as a component of the checkout deduplication key alongside `repository` and `path`.
2. A checkout with `wiki: true` and a checkout with `wiki: false` for the same `(repository, path)` **MUST NOT** be merged into a single step.
3. Two checkouts with the same `(repository, path, wiki)` triple **MUST** be merged using the existing merge semantics (e.g., deepest `fetch-depth` wins).

### Default Checkout Override

1. When resolving the default checkout override, implementations **MUST** prefer a `wiki: false` entry over a `wiki: true` entry if both exist for the empty `(repository="", path="")` key.
2. If no `wiki: false` default checkout override exists, implementations **MUST** fall back to the `wiki: true` variant, if present.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25047584674) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
