# ADR-29416: Disapproval Labels for MCP Gateway Integrity

**Date**: 2026-05-01
**Status**: Draft
**Deciders**: pelikhan

---

## Part 1 — Narrative (Human-Friendly)

### Context

The MCP gateway integrity system controls which content items an agentic workflow may act on, using a precedence chain of signals: blocked users, trusted users, approval labels, and author association. The existing `approval-labels` field allows maintainers to elevate a content item's effective integrity when specific labels are present. However, there was no symmetric mechanism to *degrade* integrity — maintainers had no way to mark a specific item as unsafe-for-agent or needing human review without blocking the entire author. This created a gap for selective veto gating at the item level.

### Decision

We will introduce a `disapproval-labels` field to the MCP gateway `allow-only` policy. Labels present in this list degrade the effective integrity of the matched content item to `none`, regardless of the author's standing. The field is inserted into the precedence chain at priority level 3 (below `trusted-users`, above `approval-labels`), so a disapproval label on a trusted-user's item will still block the item. The field follows the same expression and environment variable fallback patterns (`GH_AW_GITHUB_DISAPPROVAL_LABELS`) as all other guard policy list fields.

### Alternatives Considered

#### Alternative 1: Invert approval-labels semantics with a boolean flag

A `require-approval-labels: true` flag on the existing `approval-labels` field could have caused unlabeled items to fail integrity rather than only elevating labeled ones. This was not chosen because it would change the semantics of existing configurations and because it acts on the absence of a label rather than the explicit presence of a veto signal, making the intent less clear to workflow authors.

#### Alternative 2: Extend blocked-users to per-item blocking

The existing `blocked-users` mechanism already degrades all content from a specific author to `none`. One option was to add a label-driven alias that acts like "this item is from a blocked source." This was not chosen because it conflates author-level policy with item-level policy, and because `blocked-users` applies globally while `disapproval-labels` must be a per-item gate that trusted authors can still trigger.

### Consequences

#### Positive
- Maintainers gain a fine-grained veto mechanism: they can label a single PR or issue to prevent agent action without penalizing the author broadly.
- The design is consistent with the existing guard policy pattern, requiring no new parser conventions or env-var schemes.

#### Negative
- The precedence chain grows by one level, increasing cognitive load for teams reasoning about effective integrity resolution.
- All locked workflow files that include the `parse-guard-vars` step must be regenerated, producing a large mechanical diff that obscures the intent of the change.

#### Neutral
- The `min-integrity` requirement validation already enforces that `disapproval-labels` can only be configured when a meaningful minimum is set, keeping misconfiguration surface consistent with `approval-labels`.
- Schema files and editor autocomplete data must be updated in lockstep with any field additions.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Disapproval Labels Policy Field

1. Implementations **MUST** treat any content item whose labels include at least one entry from `disapproval-labels` as having an effective integrity of `none`, regardless of the item author's trust level.
2. Implementations **MUST NOT** allow a `disapproval-labels` entry to elevate integrity; the field is a veto-only mechanism.
3. Implementations **MUST** resolve `disapproval-labels` before `approval-labels` in the precedence chain, such that a disapproval label takes priority over any approval label on the same item.
4. Implementations **MUST** support the `disapproval-labels` field as an array of strings, a `[]string` expression, or a GitHub Actions expression string, consistent with the patterns used for `approval-labels` and `blocked-users`.
5. Implementations **MUST** validate that each entry in `disapproval-labels` is a non-empty string.
6. Implementations **MUST** require that `min-integrity` is configured whenever `disapproval-labels` is non-empty.

### Environment Variable Fallback

1. Implementations **MUST** support the `GH_AW_GITHUB_DISAPPROVAL_LABELS` organization or repository variable as a fallback source for `disapproval-labels` values, following the same resolution order used for `GH_AW_GITHUB_APPROVAL_LABELS`.
2. Implementations **SHOULD** pass `disapproval-labels` values to the `parse_guard_list.sh` step via the `GH_AW_DISAPPROVAL_LABELS_VAR` and `GH_AW_DISAPPROVAL_LABELS_EXTRA` environment variables, consistent with peer guard fields.

### Schema and Autocomplete

1. Implementations **MUST** include `disapproval-labels` in the `main_workflow_schema.json` schema definition.
2. Implementations **MUST** include `disapproval-labels` in `autocomplete-data.json` so that editor tooling surfaces the field to workflow authors.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25198744827) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
