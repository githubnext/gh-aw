# ADR-42794: Extend `app-to-github-app` Codemod to Cover Top-Level `app:` Key

**Date**: 2026-07-01
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The `gh aw fix` command provides automated codemods to help users migrate their workflow frontmatter across breaking changes. A previous breaking change renamed the `app:` frontmatter field to `github-app:`, and a codemod was implemented to automate that rename. However, the codemod only handled `app:` when it appeared nested under `tools.github`, `safe-outputs`, or `checkout` blocks — it did not handle the top-level `app:` key. Users whose workflows used the top-level `app:` auth configuration would run `gh aw fix` and see no changes applied, then encounter validation failures when the field was rejected by the tooling. This gap was surfaced through a daily breaking-change audit.

### Decision

We will extend the `hasDeprecatedAppField` detection function and the `renameAppToGitHubApp` rewrite function in `pkg/cli/codemod_github_app.go` to also detect and rename a top-level `app:` key in workflow YAML frontmatter. The top-level check uses the existing `isTopLevelKey` helper (shared frontmatter key detection logic) and is evaluated before the nested-section checks to avoid double-processing. This preserves the existing behavior for nested locations while closing the coverage gap.

### Alternatives Considered

#### Alternative 1: Document-only migration (no codemod for top-level)

Rather than extending the codemod, we could have documented the top-level `app:` rename in the migration guide and relied on users to make the change manually. This was rejected because the entire purpose of `gh aw fix` is to automate breaking-change migrations, and leaving one common configuration location uncovered would undermine user trust in the tool and create inconsistent migration outcomes.

#### Alternative 2: Generic "rename any `app:` key" approach

An alternative was to rename every occurrence of `app:` in any position within the frontmatter, without checking whether it is at the top level or inside a specific section. This was rejected because it would be too broad: it could incorrectly rename `app:` keys that appear inside unrelated nested structures, or inside the workflow body rather than the frontmatter, leading to false positives.

### Consequences

#### Positive
- `gh aw fix` now covers all known locations of the deprecated `app:` field, giving users a fully automated migration path for this breaking change.
- Regression tests (`codemod_github_app_test.go` and `fix_command_test.go`) were added for top-level rename coverage, increasing confidence that this behavior does not regress in future changes.

#### Negative
- The `hasDeprecatedAppField` and `renameAppToGitHubApp` functions are now more complex, with an explicit ordering dependency: top-level checks run before nested-section checks.
- Future contributors adding new `app:` rename locations must understand this ordering, which is not self-documenting.

#### Neutral
- The top-level check reuses the existing `isTopLevelKey` shared helper, avoiding any new YAML parsing logic.
- Changeset and CHANGELOG documentation was updated to reflect that both top-level and nested `app:` locations are affected by the breaking change.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
