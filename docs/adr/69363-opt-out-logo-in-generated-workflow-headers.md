# ADR-69363: Dual Opt-Out for the ASCII Logo in Generated Workflow Headers

**Date**: 2026-10-16
**Status**: Draft
**Deciders**: pelikhan (PR #69363), issue #68782 reporter

---

### Context

Every compiled `.lock.yml` and every companion workflow (maintenance, work-queue, central slash command, auto-update, side-repo) starts with a 13-line "Agentic Workflows" ASCII logo written unconditionally by `GenerateWorkflowHeader` in `pkg/workflow/header.go`. Repositories that review lock-file diffs must scroll past ~15 content-free comment lines before reaching `gh-aw-metadata`, `gh-aw-manifest`, secrets and actions. Removing the logo by hand does not persist: the next `gh aw compile` (run by a teammate, CI, or `gh aw upgrade`) restores it. A CLI-only flag therefore cannot solve the problem, because the opt-out must survive recompilation by anyone. The existing `--banner` flag only affects terminal output and is unrelated. `GenerateWorkflowHeader` is also called from six places, so any signature change has to avoid churn across all generators.

### Decision

We will make the logo omissible through **two independent opt-outs that OR together**: the one-off CLI flag `gh aw compile --no-logo` and the persistent repository setting `"lock_file_logo": false` in `aw.json`. Either one disables the logo; the default remains logo-on, so existing lock files are unchanged byte-for-byte unless a user opts out. Mechanically, `GenerateWorkflowHeader` gains a **variadic** `logo ...bool` parameter (`!slices.Contains(logo, false)`) so that all existing three-argument call sites keep compiling; the compiler carries a `noLogo` field set via `SetLockFileLogo`, `RepoConfig.LockFileLogo *bool` models "unset = enabled", and `config.NoLogo` is threaded through the post-processing wrappers to the companion-workflow generators. The persistent setting is chosen to mirror `require_self_hosted_runners`, the closest existing precedent for a repo-wide compiler toggle.

### Alternatives Considered

#### Alternative 1: CLI flag only (`--no-logo`, no `aw.json` key)

The smallest possible change: a single cobra flag threaded into `CompileConfig`. Rejected because the opt-out would not persist — any recompilation without the flag (CI, `gh aw upgrade`, another contributor) reintroduces the logo, which is precisely the failure mode reported in issue #68782.

#### Alternative 2: Non-variadic `logo bool` parameter (or an options struct) on `GenerateWorkflowHeader`

The issue's implementation plan suggested a plain `logo bool` parameter or a small options struct, updating all six callers. This is more explicit and self-documenting than a variadic, and an options struct would scale better if more header knobs appear later. Rejected (for now) in favour of the variadic because it keeps the three-argument call sites in external/companion generators source-compatible and keeps the diff narrow; the cost is a less obvious API. *(This was a close call — [TODO: confirm the team prefers source compatibility over an explicit parameter.])*

#### Alternative 3: Post-process/strip the logo after generation

Generate headers as today and filter the logo block out of the emitted YAML when opted out. Rejected because it duplicates knowledge of the header layout in a second place and makes the generated output dependent on fragile text matching rather than on the generator itself.

### Consequences

#### Positive

- Lock-file diffs become materially easier to review in repositories that opt out: ~15 noise lines removed per generated file.
- The opt-out is durable — `aw.json` makes it a repository-wide property, so recompilation by CI or other contributors preserves the choice.
- Default behaviour is unchanged, so no existing repository sees a lock-file churn from this change.
- The new setting follows the established `require_self_hosted_runners` pattern (schema entry in `repo_config_schema.json`, accessor `IsLockFileLogoEnabled`), keeping `aw.json` consistent.

#### Negative

- `GenerateWorkflowHeader(sourceFile, generatedBy, customInstructions, logo ...bool)` is a variadic boolean — an API shape that is easy to call incorrectly and that hides the parameter's meaning at call sites.
- The effective setting is now computed from two sources (`--no-logo` and `aw.json`), so debugging "why is/isn't the logo there?" requires checking both; the OR semantics also mean the flag cannot re-enable a logo disabled by `aw.json`.
- `noLogo` must be plumbed through `CompileConfig` → post-processing wrappers → each companion-workflow generator; a future generator that forgets to pass it will silently emit the logo.
- `generateWorkflowHeader` now loads `aw.json` during header generation and can fail there, adding a new error path to a previously infallible step.

#### Neutral

- `RepoConfig.LockFileLogo` is a `*bool` to distinguish "unset" from "false"; this matches other tri-state repo config fields.
- Documentation is updated in two places (`docs/src/content/docs/setup/cli.md` and `reference/compilation-process.md`) because the feature has two entry points.
- Metadata, manifest, "DO NOT EDIT" and regeneration instructions are explicitly preserved — only the logo block is conditional.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
