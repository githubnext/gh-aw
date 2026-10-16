# ADR-69362: Gate Copilot Artifact Updates on Detected Workflow Engine During `upgrade`

**Date**: 2026-10-16
**Status**: Draft
**Deciders**: pelikhan

---

### Context

`gh aw init` only writes `.github/agents/agentic-workflows.md` and `.github/workflows/copilot-setup-steps.yml` when the Copilot engine is selected, and it exposes `--no-agent`, `--no-mcp`, and `--no-skill` to opt out of individual artifacts. `gh aw upgrade` had neither behaviour: `updateCopilotArtifacts` unconditionally wrote the dispatcher skill, the custom agent, and the Copilot setup-steps workflow on every run. In repositories whose workflows use `engine: claude` or `engine: codex`, each upgrade therefore re-created files the repository does not use, which maintainers had to delete by hand and which could fail repository formatter checks (issue #68872). The fix had to keep existing installations working — repositories that already have these files must continue to receive refreshed templates — while avoiding the introduction of unused files into non-Copilot repositories.

### Decision

We will make `gh aw upgrade` engine-aware: the Copilot custom agent and `copilot-setup-steps.yml` are written only when the repository actually uses Copilot, or when the corresponding file already exists on disk (`os.Stat` check before each write). Copilot usage is determined by `upgradeUsesCopilot`, which honours an explicit `--engine` override first and otherwise scans the workflow directory with `extractEngineIDFromFile`, treating the implicit default engine as Copilot. We will also add `--no-agent`, `--no-mcp`, and `--no-skill` flags, plumbed through `upgradeOptions`, so each artifact can be skipped even when it already exists. The dispatcher skill remains updated by default because it is engine-agnostic. `updateCopilotArtifacts` now takes the whole `upgradeOptions` struct instead of `(ctx, verbose)` so these decisions are made in one place.

### Alternatives Considered

#### Alternative 1: Opt-out flags only, without engine detection

We could have added `--no-agent`, `--no-mcp`, and `--no-skill` and left the default unconditional. This is a smaller change and keeps `upgrade` fully deterministic regardless of repository content. It was rejected because it does not fix the reported problem by default: every non-Copilot repository would still accumulate unused Copilot files on the first upgrade, and maintainers would have to remember flags on every invocation or wire them into CI.

#### Alternative 2: Only refresh files that already exist (pure "update, never create")

`upgrade` could have been narrowed to refresh existing artifacts and never create new ones, deferring creation entirely to `init`. This is the simplest rule to explain and removes the need for engine scanning. It was rejected because repositories that adopt Copilot after their initial `init` — or that were initialised by an older version of the tool — would silently never receive the agent or setup-steps files, making `upgrade` incapable of bringing a repository up to the current expected layout.

#### Alternative 3: Derive engine usage from compiled `.lock.yml` files instead of markdown frontmatter

Engine usage could be read from generated lock files rather than parsing the markdown source. It was rejected because lock files may be stale or absent (for example under `--no-compile`), whereas the markdown frontmatter is the authoritative declaration and is already parseable via the existing `extractEngineIDFromFile` helper.

### Consequences

#### Positive
- Non-Copilot repositories no longer accumulate unused Copilot agent and setup-steps files on upgrade, resolving #68872 and the associated formatter failures.
- `upgrade` and `init` now share the same mental model and the same `--no-agent` / `--no-mcp` / `--no-skill` vocabulary, reducing surprise for users moving between the two commands.
- Existing Copilot installations keep receiving template refreshes, because an existing file on disk is sufficient to trigger an update.

#### Negative
- `upgrade` behaviour now depends on repository content, so the same command can produce different file sets in different repositories; this is harder to reason about and must be documented (CLI help and `docs/src/content/docs/setup/cli.md` were updated accordingly).
- Engine detection parses every workflow markdown file in the workflow directory on each upgrade, adding I/O and another failure mode; detection failures are logged and degrade to "not Copilot", which can silently skip artifact creation.
- Three new boolean flags widen the `upgrade` surface area and the combinatorial space that tests must cover (the PR adds a 19-case table test to compensate).

#### Neutral
- A missing or unparseable `engine:` field is treated as Copilot, matching the product default, so repositories that omit the field explicitly still get the Copilot artifacts.
- `updateCopilotArtifacts` changed signature from `(ctx, verbose)` to `(opts upgradeOptions)`; this is an unexported helper, so there is no external API impact.
- The dispatcher skill is deliberately excluded from engine gating and is still written by default unless `--no-skill` is passed.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
