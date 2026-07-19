# ADR-46521: Configurable Auto-Upgrade Cron Schedule via Polymorphic aw.json Field

**Date**: 2026-07-19
**Status**: Draft
**Deciders**: Unknown

---

### Context

The `agentic-auto-upgrade.yml` workflow runs on a schedule derived by scattering `FUZZY:WEEKLY` using a seed built from the repository slug. This produces a deterministic but repo-specific time that cannot be overridden. Repositories that need a predictable, fixed schedule — for example, to align with release cycles or organizational maintenance windows — have no way to configure the timing without forking or patching the generated workflow. The `aw.json` compiler already uses a polymorphic object/boolean pattern for the `maintenance` field, establishing a precedent for this kind of extension.

### Decision

We will extend the `auto_upgrade` field in `aw.json` to accept either a boolean (existing behavior) or an object `{ "cron": "<5-field POSIX expression>" }`. When the object form is used, the `cron` value is passed verbatim to `GenerateAutoUpdateWorkflow` via a new `CustomCron` option and written directly into the generated `agentic-auto-upgrade.yml`, skipping the fuzzy scatter. Omitting `cron` from the object, or using `auto_upgrade: true`, retains the existing scatter behavior. JSON schema validation rejects syntactically invalid cron strings at load time.

### Alternatives Considered

#### Alternative 1: Top-level `auto_upgrade_cron` string field

Add a separate `auto_upgrade_cron` key alongside the boolean `auto_upgrade` in `aw.json`. This is simpler to parse (no polymorphism) and keeps the schema flat. It was explored in an earlier commit on this branch (`feat: add auto_upgrade_cron to aw.json`). It was rejected because it creates a two-field API where both fields must be set together, increasing the chance of misuse (e.g., setting `auto_upgrade_cron` while `auto_upgrade` is `false` or omitted). Nesting `cron` inside `auto_upgrade` makes the relationship explicit: the object form implies enabled.

#### Alternative 2: Separate top-level `auto_upgrade_schedule` object

Introduce a new top-level `auto_upgrade_schedule: { cron: "..." }` key independent of the boolean `auto_upgrade`. This keeps backward compatibility without polymorphism but adds a third distinct configuration surface for a single feature, fragmenting the API further. It was not chosen because it diverges from the `maintenance` field pattern and forces users to set two keys to configure one behavior.

### Consequences

#### Positive
- Repositories can pin the auto-upgrade workflow to a predictable, fixed cron schedule without editing the generated file.
- Invalid cron expressions are caught at `LoadRepoConfig` time via JSON schema validation, providing early feedback before any workflow is generated.
- The object form follows the existing `maintenance` polymorphism pattern already established in `aw.json`, keeping the API internally consistent.

#### Negative
- The `auto_upgrade` field is now polymorphic (boolean | object), requiring a custom `UnmarshalJSON` implementation with try-boolean-first, fall-back-to-object logic. This adds parse complexity and a subtle order dependency.
- The object form with no `cron` key (`{ }`) and the boolean `true` are semantically identical — two ways to enable the default scatter schedule, which may confuse users reading existing configs.

#### Neutral
- The `RepoConfig` struct gains a new `AutoUpgradeCron string` field that is always empty when the boolean form is used, so callers that don't need custom cron need not change.
- All three `GenerateAutoUpdateWorkflow` call sites in `maintenance_workflow.go` are updated uniformly via a new `autoUpgradeCronFrom(cfg)` helper, keeping the change mechanical and auditable.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
