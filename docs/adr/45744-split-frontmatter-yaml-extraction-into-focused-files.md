# ADR-45744: Split Frontmatter YAML Extraction into Focused Files

**Date**: 2026-07-15
**Status**: Draft
**Deciders**: pelikhan (author), copilot-swe-agent (implementer)

---

### Context

`pkg/workflow/frontmatter_extraction_yaml.go` had grown past the project's file-diet threshold into a large, mixed-responsibility file. The bulk of its complexity was concentrated in two distinct areas that have little overlap with core YAML extraction: the stateful `on:` section rewrite path (which comments out processed fields in the generated workflow YAML) and the trigger-derived condition helpers (which translate frontmatter `on.deployment_status.state` and `on.workflow_run.conclusion` into GitHub Actions expression strings). Keeping all three concerns in a single file made the `on:` rewrite path hard to navigate and buried trigger helpers inside a file nominally about YAML extraction.

### Decision

We will split `frontmatter_extraction_yaml.go` into three focused files within the same `workflow` package:

1. **`frontmatter_extraction_yaml.go`** — retains core YAML extraction and command/config parsing flow.
2. **`frontmatter_on_section_cleanup.go`** — isolates the stateful `on:` YAML rewrite logic (`commentOutProcessedFieldsInOnSection`, `addZizmorIgnoreForWorkflowRun`, `isGitHubAppNestedField`).
3. **`frontmatter_trigger_helpers.go`** — groups trigger-specific extraction and condition synthesis (`extractOnTriggerValue`, `extractOnTriggerMap`, `normalizeStringOrStringSlice`, `extractDeploymentStatusStateCondition`, `extractWorkflowRunConclusionCondition`, `isValidWorkflowRunConclusion`, `validWorkflowRunConclusions`).

No behavior is changed; all call sites remain identical.

### Alternatives Considered

#### Alternative 1: Keep all code in `frontmatter_extraction_yaml.go`

Continue adding logic to the single file and accept increasing size. No cross-file navigation is required; all helpers are co-located. This was rejected because the file had already exceeded the project's maintainability threshold and mixes three distinct concerns, making the `on:` rewrite path harder to follow without scrolling past unrelated extraction code.

#### Alternative 2: Extract to a single helper file

Move all non-core helpers (both `on:` cleanup and trigger helpers) into one `frontmatter_helpers.go`. This reduces the file count but still mixes the stateful YAML rewriting logic with the pure trigger-expression helpers, defeating the single-responsibility goal. The trigger helpers and the `on:` post-processing helpers have different scopes and rates of change; keeping them apart makes future edits safer.

### Consequences

#### Positive
- Each resulting file remains under the project's file-diet threshold.
- The `on:` YAML rewrite path is isolated in one place, making it easier to navigate and test independently.
- Trigger-condition helpers (`extractDeploymentStatusStateCondition`, `extractWorkflowRunConclusionCondition`) are discoverable without searching through the larger extraction file.
- Future contributors can extend trigger-condition logic in `frontmatter_trigger_helpers.go` without risk of inadvertently affecting `on:` cleanup logic.

#### Negative
- Three files must now be opened to get a full picture of the frontmatter compilation pipeline where one sufficed before.
- The package-level `frontmatterLog` logger is defined in `frontmatter_extraction_yaml.go` and implicitly shared with the new files via Go's package scope, creating an invisible coupling not obvious from the new files alone.

#### Neutral
- All public and package-internal call sites are unchanged; the split is transparent to callers.
- The split follows the same pattern established by earlier file-diet refactors in this package (e.g., ADR-45598, ADR-45635).

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
