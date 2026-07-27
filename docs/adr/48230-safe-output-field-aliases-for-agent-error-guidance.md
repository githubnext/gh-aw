# ADR-48230: Safe-Output Field Aliases for Agent Error Guidance

**Date**: 2026-07-27
**Status**: Draft
**Deciders**: Unknown

---

### Context

Agents authoring agentic workflows routinely use MCP tool names (e.g. `create_issue_comment`) or underscore variants (e.g. `add_comment`) in the `safe-outputs` frontmatter section instead of the correct hyphenated field names (e.g. `add-comment`). The existing error path emits a generic "Valid fields are: ..." dump that provides no actionable signal. A Levenshtein/edit-distance approach is ineffective here because the semantic distance between a GitHub MCP tool name and its `safe-outputs` canonical equivalent can be large (e.g. `create-issue-comment` → `add-comment` is ~14 edit operations), making fuzzy matching produce wrong or empty suggestions. The schema validation layer in `pkg/parser` is the right place to intercept these errors before they reach the user.

### Decision

We will add a curated static alias map (`safeOutputAliases`) in the `pkg/parser` package that maps known agent mistakes — including MCP tool name variants and underscore ↔ hyphen swaps — to their correct `safe-outputs` canonical field names. A dedicated function (`safeOutputAliasSuggestion`) checks this map when a schema additional-properties error is raised under the `/safe-outputs` path, and returns a precise "Did you mean 'X'?" suggestion that takes priority over the generic field-list fallback. This is integrated into `generateSchemaBasedSuggestions` ahead of the existing `additionalPropertiesSuggestion` call.

### Alternatives Considered

#### Alternative 1: Levenshtein / fuzzy edit-distance matching

Compute string edit distance between the invalid field name and each valid `safe-outputs` field name, and suggest the closest match. This approach is already available as a pattern elsewhere in suggestion logic. It was rejected here because the edit distance between MCP tool name variants and their canonical `safe-outputs` equivalents is too large (e.g. `create-issue-comment` → `add-comment` is ~14 edits) — the closest fuzzy match would be a different, unrelated field, producing misleading suggestions.

#### Alternative 2: Silent normalization at parse time

Automatically normalize underscore-separated or MCP-style field names to their hyphenated canonical equivalents during frontmatter parsing, silently accepting the misspelling. This would suppress the validation error entirely, removing the learning signal for workflow authors and masking a class of mistakes that benefit from explicit correction. It would also allow non-canonical spellings to persist in workflow files, complicating future schema evolution.

### Consequences

#### Positive
- Agents and workflow authors receive precise, actionable error messages ("Did you mean 'add-comment'?") instead of an undifferentiated field dump, reducing the iteration cycle on `safe-outputs` configuration errors.
- Deduplication logic within `safeOutputAliasSuggestion` collapses multiple aliases resolving to the same canonical name into a single suggestion, keeping error output clean when multiple wrong field names appear together.

#### Negative
- The alias map must be maintained manually as new `safe-outputs` fields are added or existing fields are renamed; there is no automated enforcement that the alias table stays in sync with the schema.
- The alias map covers only pre-enumerated mistakes; novel misspellings not yet in the table fall through to the generic error message with no improvement over the prior behaviour.

#### Neutral
- The alias check is scoped exclusively to the `/safe-outputs` JSON schema path and fires only on additional-properties errors, so it has no effect on validation errors in other frontmatter sections.
- Unit tests cover alias lookups, deduplication, and non-matching paths; integration tests exercise the full pipeline using real on-disk workflow files.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
