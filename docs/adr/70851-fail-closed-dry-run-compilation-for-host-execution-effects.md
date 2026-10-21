# ADR-70851: Fail Closed in Dry-Run Compilation for Host Execution Effects

**Date**: 2026-10-21
**Status**: Draft
**Deciders**: pelikhan, Unknown (review pending)

---

### Context

`gh aw compile --dry-run` produces diagnostic lock files intended for live testing with compiler-managed mutations staged rather than applied. A hosted-readiness audit found that staging the built-in safe outputs did not neutralize other execution surfaces: authored custom jobs, custom safe-output jobs, reusable workflow calls, host steps (`steps`, `pre-steps`, `pre-agent-steps`, `post-steps`, `on.steps`), custom MCP servers and `mcp-scripts`, custom safe-output scripts, Bash graders, and `id-token: write` permissions could all still run with real credentials. Built-in job names could also be overridden with `uses`/`secrets`/`steps`, bypassing the staging. As a result, a successful dry-run compile was not a trustworthy signal that unattended hosted execution was side-effect-free. Separately, the dry-run scanner gate only failed on findings *added* relative to a baseline, so pre-existing medium-or-higher findings passed silently.

### Decision

We will make dry-run compilation fail closed for execution effects that staging cannot isolate. Concretely: (1) a new `validateDryRunEffects` check in `pkg/workflow/compiler_development_safety.go` rejects compilation when authored host steps, custom MCP servers/scripts, custom safe-output scripts, enabled Bash graders, authored OIDC write permissions (including imports), or execution/secret overrides on built-in job names are present; (2) custom jobs, custom safe-output jobs, and managed reusable calls are rewritten into inert placeholders (`if: ${{ !always() }}`, `permissions: {}`, an echo step, outputs preserved by name but emptied) so the job graph remains valid without retaining executable targets or credentials; and (3) the dry-run scanner gate fails on any medium-or-higher or unclassified finding, existing or added, while still reporting differential counts and canonicalizing file identities via `filepath.EvalSymlinks`. The primary driver is that dry-run output must be a safe signal for hosted execution; rejecting is preferred over silently allowing unverified effects.

### Alternatives Considered

#### Alternative 1: Keep staging only built-in safe outputs and document the gaps

Leave custom jobs, host steps, and custom MCP/scripts executing normally in dry-run and document that dry-run only protects compiler-managed safe outputs. This is non-breaking and simplest, but it leaves the audit finding unresolved: a passing dry-run compile would continue to imply safety it does not provide, and the built-in-job-name override bypass would remain.

#### Alternative 2: Attempt to sandbox or stage every authored effect

Rewrite custom steps, MCP servers, and scripts into staged or mocked equivalents so diagnostic workflows still compile and partially run. This would preserve more workflow coverage, but arbitrary host scripts, external MCP effects, and cloud credentials cannot be reliably intercepted by the compiler; claiming isolation would be unsound. The PR explicitly avoids "claiming that dry-run is a general execution sandbox."

#### Alternative 3: Strip unsupported declarations silently instead of rejecting

Remove host steps, custom MCP servers, and OIDC permissions from the emitted lock without failing compilation (as is done for custom jobs). This keeps more workflows compiling, but silently changes agent behavior (e.g., missing tools or setup) so the diagnostic run would no longer represent the real workflow, and authors would not learn that those effects went unverified. Disabling is used only for jobs, where dependency/output names can be preserved structurally.

### Consequences

#### Positive
- A successful `--dry-run` compile becomes a meaningful precondition for unattended hosted testing: unsupported effects are either rejected or rendered inert and credential-free.
- Closes the bypass where built-in job names could be overridden with `uses`, `secrets`, `steps`, or `permissions`.
- Scanner gate no longer accepts known medium-or-higher or unclassified findings merely because they are in the baseline; symlinked paths no longer produce mismatched identities.

#### Negative
- Breaking change (major changeset): diagnostic workflows with unsupported declarations now fail compilation; per the PR, 208 of 346 fleet workflows are rejected under hardened dry-run.
- Disabled custom jobs emit empty outputs, so dependent jobs may skip and dry-run coverage of downstream logic is reduced.
- Existing scanner findings (e.g., a pre-existing poutine runner finding) now block fleet admission until resolved; no waiver mechanism is introduced.

#### Neutral
- Ordinary (non-dry-run) compilation is unchanged.
- Agent shell execution, custom engines, custom credentials, and hosted identity/runner behavior remain outside dry-run guarantees and still require independent review.
- Trigger rewriting moved from `map[string]any` to `yaml.Node` to preserve comments when adding `workflow_dispatch`.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
