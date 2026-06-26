# ADR-41719: Per-Repository Confirmation Gate for Org-Mode Create Actions

**Date**: 2026-06-26
**Status**: Draft
**Deciders**: Unknown

---

### Context

The `gh aw update --org` and `gh aw upgrade --org` commands support `--create-issue` and `--create-pull-request` flags that execute write actions (issue or PR creation) across every matching repository in an organization. Prior to this change, these create actions were applied to all repositories unconditionally — a single command could silently open issues or pull requests in dozens of repos with no per-repository confirmation. This presents a risk of unintended bulk writes, especially when the matching repo set is broader than expected. CI environments additionally require a non-interactive path since interactive prompts cannot be answered there.

### Decision

We will add an interactive per-repository confirmation gate to all org-mode create actions (`--create-issue` and `--create-pull-request`). Before each repository's create action runs, the user is shown a summary of the pending change (repo name, workflow count, version deltas) and asked to accept or skip. A `--yes` / `-y` flag auto-accepts all confirmations for non-interactive use. Org-mode create actions in CI environments without `--yes` will fail-fast with an explicit error instructing the caller to re-run with `--yes`.

### Alternatives Considered

#### Alternative 1: Two-pass dry-run + explicit confirmation flag

Show a full preview of all repositories that would be affected, then require the user to re-run the command with a `--confirmed` flag to execute. This avoids interactive prompts entirely and is familiar from tools like `terraform apply`. It was not chosen because it requires two full scans (doubling API calls and latency) and provides no per-repository granularity — the user must accept or reject the entire batch.

#### Alternative 2: Require explicit `--repos` when using create actions

Rather than prompting, require users to name repositories explicitly (or use a tight glob) whenever `--create-issue` or `--create-pull-request` is combined with `--org`. This eliminates accidental bulk writes at the argument level. It was not chosen because it degrades the primary org-sweep use case where the whole-org scan is intentional; the per-repo prompt is a lighter-weight safeguard that preserves the convenience of broad org targeting.

### Consequences

#### Positive
- Users see a per-repository change summary before any write action, reducing accidental bulk issue/PR creation.
- `--yes` flag provides a clean, explicit opt-in for CI and automation scripts.
- CI guardrail (fail-fast without `--yes`) makes the required flag self-documenting in error output.
- Skip-without-error semantics allow partial org runs without triggering failure pipelines.

#### Negative
- **Breaking change for existing automation**: Scripts using `gh aw update --org … --create-issue` or `--create-pull-request` without `--yes` will now fail in CI environments. Callers must add `--yes` to maintain prior behavior.
- Increases org-runner control-flow complexity: the loop now distinguishes between "user skipped" (not an error), "accepted but failed" (error), and "never attempted" (new `attempted == 0` short-circuit).

#### Neutral
- The confirmation prompt is wired via an injectable function variable (`orgConfirmActionFn`) to keep unit tests hermetic — this is a testability pattern already used in the codebase.
- The `--yes` flag is added to both `update` and `upgrade` commands with the same semantics, keeping the two commands consistent.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
