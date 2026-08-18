# ADR-53725: Support Runner-Group Object Form for `safe-outputs.jobs` `runs-on` and `runner`

**Date**: 2026-08-18
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The gh-aw compiler supports three `runs-on` shapes for GitHub Actions jobs: a plain string (e.g. `"ubuntu-latest"`), a string array (e.g. `["self-hosted", "linux"]`), and a runner-group object (e.g. `{group: "my-group", labels: ["linux"]}`). This full surface is already supported for top-level `runs-on`, `runs-on-slim`, `safe-outputs.runs-on`, and `safe-outputs.threat-detection.runs-on`.

However, `safe-outputs.jobs.<id>.runs-on` and its `runner` alias only accepted the string and array shapes. Users operating runner-group–only fleets had no way to target those fleets from custom safe-jobs. When an object form was supplied it was silently dropped during compilation, causing the `runs-on` key to be omitted from emitted safe-job YAML entirely.

### Decision

We will align `safe-outputs.jobs.<id>.runs-on` and `safe-outputs.jobs.<id>.runner` with the existing `github_actions_runs_on` definition by updating the JSON schema reference, preserving the raw input value in `SafeJobConfig.rawRunsOn`, introducing a `resolveSafeJobRunsOn` render function that uses the raw value when available, and extending validation and macOS-guard checks to cover both fields across all custom safe-jobs.

The primary driver is correctness parity: all `runs-on` fields in the compiler should accept the same set of valid shapes, and the compiler should never silently omit a `runs-on` line from emitted YAML.

### Alternatives Considered

#### Alternative 1: Accept object form for `runs-on` only, not for the `runner` alias

Only update the `runs-on` field to support the object form while leaving the `runner` alias restricted to string and array. This is simpler — the `runner` alias is undocumented for object-form usage — but it creates an asymmetry: `runner` was intended as a full alias for `runs-on`, so diverging here would confuse users who already use `runner: {group: ...}` successfully at other levels.

#### Alternative 2: Reject object-form `runs-on` in safe-jobs and require label arrays instead

Require users with runner-group fleets to express their runner as a label array (e.g. `["self-hosted", "linux"]`) rather than using the object form. This keeps `SafeJobConfig` simpler (no `rawRunsOn` field) but is a breaking change relative to user expectation, since the object form works at every other level of the configuration and the issue (#52303) was filed precisely because users expected parity.

### Consequences

#### Positive
- Custom safe-jobs can now target runner-group fleets using the same `{group, labels}` YAML object form supported everywhere else in the compiler.
- Schema, parsing, rendering, and validation paths for `runs-on` / `runner` are now consistent across all job types.
- The fix is covered by a full suite of unit and compile-path integration tests (string, array, group-only, group+labels, alias, unset/default).

#### Negative
- `SafeJobConfig` now carries a secondary `rawRunsOn any` field alongside `RunsOn RunsOnValue` and `runsOnArray bool`, creating a two-representation design. The render dispatch (`resolveSafeJobRunsOn`) must be kept in sync if new `runs-on` shapes are added in the future.
- Safe-job `runs-on` validation is extended as a separate code path from top-level validation; this duplication must be maintained if the macOS-guard or other shape constraints are updated.

#### Neutral
- The `runner` alias field in the JSON schema is updated to reference `#/$defs/github_actions_runs_on` alongside examples, improving editor tooling and discoverability at no runtime cost.
- Documentation for `safe-outputs.md` and `self-hosted-runners.md` is updated to explicitly state object-form support for custom safe-jobs.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
