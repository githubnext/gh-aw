# ADR-53977: Extend Custom Safe-Job `runs-on` to Support Runner-Group Objects and Remove `runner` Alias

**Date**: 2026-08-20
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

Custom safe jobs (`safe-outputs.jobs.<job>`) had their own limited `runs-on` parser that only accepted strings and label arrays. Runner-group object form (`{group: ..., labels: [...]}`) was not supported, leaving custom safe jobs unable to run on self-hosted runner groups. All other `runs-on` configuration surfaces in the framework (top-level `runs-on`, `safe-outputs.runs-on`, `safe-outputs.threat-detection.runs-on`) already accepted all three forms via the shared `extractCustomJobRunsOn` parser. The legacy `runner` key was a deprecated alias for `runs-on` that duplicated the configuration surface and required a separate code path.

### Decision

We will reuse the shared `extractCustomJobRunsOn` parser for custom safe jobs, making their `runs-on` field accept the same three forms (string, label array, runner-group object) as every other runner configuration surface. The deprecated `runner` alias will be removed as a breaking change (major version bump), and a `gh aw fix` codemod (`safe-job-runner-to-runs-on`) will be provided to automatically migrate existing workflows.

### Alternatives Considered

#### Alternative 1: Extend the Custom Safe-Job Parser In-Place

Extend the existing `toRunsOnValue`/`isRunsOnArrayValue` helpers to also handle `map[string]any` (runner-group objects) without delegating to the shared parser. This avoids a code-sharing dependency, but duplicates validation logic (macOS label rejection, empty-object rejection, unknown key rejection) that is already tested and maintained in `extractCustomJobRunsOn`. Any future change to runner-group validation would need to be applied in two places.

#### Alternative 2: Keep the `runner` Alias as a Deprecated No-Op

Retain `runner` as a tolerated (but warned) alias rather than removing it outright, making the change non-breaking. This avoids the need for a migration codemod and a major version bump. However, it perpetuates two parallel configuration keys for the same concept and increases schema surface area indefinitely. Given that `gh aw fix` can automate the rename, the migration cost is low enough to justify a clean removal.

### Consequences

#### Positive
- Custom safe jobs now have full parity with all other runner configuration surfaces, enabling use of runner groups.
- Validation logic (macOS rejection, empty-object rejection, unknown-key rejection) is exercised from a single code path, reducing the risk of inconsistencies.
- The schema is simplified: one canonical key (`runs-on`) replaces two (`runs-on` and `runner`).
- The automated codemod minimizes user effort for migration.

#### Negative
- This is a breaking change: workflows using `safe-outputs.jobs.<job>.runner` will fail validation until migrated. Users must run `gh aw fix` or manually rename the key.
- The major version bump signals a broader API break even though only one deprecated alias is removed, which may cause friction for teams managing dependency pins.

#### Neutral
- The `SafeJobConfig.RunsOn` field type changes from `RunsOnValue` (a `[]string`) to `string` (the pre-serialized YAML snippet), aligning with how the shared compiler represents runner configuration internally.
- Two helper functions (`toRunsOnValue`, `isRunsOnArrayValue`) are deleted from `runs_on_snippet.go` as they are now unused.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
