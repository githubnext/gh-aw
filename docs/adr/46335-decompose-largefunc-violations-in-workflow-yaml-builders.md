# ADR-46335: Decompose Largefunc Violations in Workflow YAML Builders

**Date**: 2026-07-18
**Status**: Draft
**Deciders**: Unknown (automated refactor by Copilot SWE agent per lint issue #46330)

---

### Context

The `gh-aw` codebase enforces a 60-line function limit via the `largefunc` custom lint rule (`make golint-custom`). Five functions in `pkg/workflow` significantly exceeded this limit: `buildMaintenanceWorkflowYAML` (961 lines), `generatePrompt` (284 lines), `generateWorkflowHeader` (172 lines), `generateCreateAwInfo` (180 lines), and `generateOutputCollectionStep` (82 lines). These monolithic functions mix orchestration logic with per-section rendering concerns, making individual sections hard to read, test, and reason about in isolation. Lint failures blocked CI quality gates.

### Decision

We will extract focused named helper functions from each oversized YAML builder, converting each logical section (e.g., per-job block, metadata line, comment block) into its own helper. The main functions become thin orchestrators that delegate to these helpers. All external interfaces and emitted YAML output are preserved unchanged — this is a pure behavioral-preserving refactor.

### Alternatives Considered

#### Alternative 1: Raise or Disable the `largefunc` Lint Limit

Increase the threshold for these specific files or disable the rule entirely, silencing the lint failures without restructuring code. This was not chosen because the lint rule exists to improve maintainability; disabling it would perpetuate functions that are genuinely difficult to navigate, review, and test independently.

#### Alternative 2: Restructure YAML Generation into Separate Packages

Move per-job or per-section YAML generation into dedicated sub-packages or service types, defining explicit interfaces between them. This would provide stronger separation of concerns but constitutes a larger architectural change — it alters call sites, requires interface design decisions, and introduces more risk of behavioral regression. The 60-line lint rule can be satisfied by helper extraction within the same file/package without restructuring package boundaries.

### Consequences

#### Positive
- Lint compliance restored: all five functions now fall within the 60-line limit, unblocking CI quality gates.
- Each helper has a single, named responsibility (e.g., `writeLockMetadataLine`, `writeHeaderCommentBlock`), improving readability and locatability of specific rendering logic.
- Smaller units are easier to unit-test independently, enabling future tests on individual YAML sections without constructing full compiler state.
- Emitted YAML and all external interfaces are unchanged, so no downstream callers or workflow consumers are affected.

#### Negative
- Added function call indirection: readers following the rendering flow must now trace through multiple helper calls rather than reading a single linear function.
- Behavioral parity relies on the refactor being strictly equivalent; any subtle divergence in control flow would affect emitted YAML content in production workflows.
- The PR adds ~1,387 lines of net code (mostly new helper functions), increasing the total surface area in the package.

#### Neutral
- New helper functions are package-private (lowercase), keeping the public API surface unchanged.
- The split of `maintenance_workflow_yaml.go` into a main file plus `maintenance_workflow_yaml_sections.go` introduces a new file that future contributors must be aware of when navigating the maintenance workflow builder.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
