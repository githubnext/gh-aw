# ADR-47557: Add A/B Experiment Infrastructure to smoke-copilot-sub-agents Workflow

**Date**: 2026-07-23
**Status**: Draft
**Deciders**: pelikhan, ab-testing-advisor (automated)

---

### Context

The `smoke-copilot-sub-agents` workflow has shown reliability instability — 8 of the last 10 visible runs failed. The workflow calls three fixed inline sub-agents (`haiku-whoami`, `mini-whoami`, `nano-whoami`) and verifies exact identity strings, but it is unclear whether the current inline orchestration strategy is optimal or whether a different strategy would improve pass rate. To answer this question with statistical rigor, the team needs a mechanism to route each workflow run to one of three orchestration variants (`inline_strict`, `delegated_sequential`, `single_agent_control`), accumulate variant-level metrics automatically over ≥30 runs each, and propagate the selected variant into prompt logic at runtime.

The `gh-aw` compiler's handlebars template system already supports `experiments.<name>` variable substitution, but had no validation to prevent authors from writing fragile inline conditional syntax (e.g., `4. {{#if cond}}A{{else}}B{{/if}}` on one line), which the experiment prompt draft surfaced as a latent authoring risk.

### Decision

We will add experiment metadata to the workflow frontmatter under `experiments.sub_agent_strategy`, implement variant-conditional prompt blocks using the existing handlebars template system with separators placed on their own lines, and persist per-run assignment state to a dedicated git branch (`experiments/smokecopilotsubagents`) via GitHub Actions artifacts. We will also add a compiler validation rule (`detectMidlineTemplateSeparators`) that warns — but does not hard-error — when template separators appear mid-line in workflow markdown.

The variant assignment is performed at the start of the `activation` job via a `pick-experiment` step that reads prior state from git, selects the next variant weighted round-robin (34/33/33), and writes the result to an artifact uploaded per run. A dedicated `push_experiments_state` job (running in parallel with the agent) commits accumulated state back to the experiments branch after each run.

### Alternatives Considered

#### Alternative 1: External A/B Testing Service (e.g., LaunchDarkly, Unleash)

Use an external feature-flag or experimentation platform to assign variants at runtime. This would provide rich dashboards and statistical analysis out of the box, but adds an external service dependency, requires authentication secrets, and introduces a network call in the critical activation path. It is also disproportionate for a single-workflow experiment with a 30-run sample target. Rejected due to complexity and dependency cost.

#### Alternative 2: Static Variant Hardcoding with Manual PR Rotation

Hardcode a single variant per run and rotate it via separate PRs for each variant window. This avoids runtime infrastructure but requires human intervention per variant switch, prevents concurrent multi-variant collection, and cannot produce reliable statistical samples automatically when run frequency is low. Rejected because it defeats the purpose of automated experimentation.

#### Alternative 3: Separate Workflow Files per Variant

Create three copies of the workflow, one per variant, each running independently. This isolates variants at the file level but triples maintenance burden, makes it harder to ensure identical conditions across variants (other than the orchestration dimension), and would require three separate scheduling configurations. Rejected due to maintenance overhead and risk of configuration drift between variants.

### Consequences

#### Positive
- Statistically sound experiment data collected automatically across ≥30 runs per variant without human intervention
- Variants are self-documenting in workflow frontmatter; hypothesis and guardrails are co-located with the implementation
- Compiler warning (`detectMidlineTemplateSeparators`) prevents future authors from writing fragile inline conditional blocks; surfaces the issue at compile time rather than at runtime
- State persists in git with no external service dependency; artifact-based state survives across workflow runs

#### Negative
- The `experiments/smokecopilotsubagents` branch is a long-lived branch that must be deleted manually after the experiment concludes (~90 days); no automatic cleanup is implemented
- The `push_experiments_state` job runs unconditionally after every `activation` success, adding a small but non-zero CI overhead (checkout + git push) even between experiment campaigns
- The `single_agent_control` variant intentionally always produces FAIL status, which artificially depresses the overall workflow success rate in monitoring dashboards during the experiment window

#### Neutral
- Experiment state is stored as JSON in a git branch rather than a database; this is readable but not queryable without downloading the branch
- The compiler warning increments `WarningCount` but is not a hard error; existing workflows with inline separators will produce warning noise until authors migrate them
- The experiment infrastructure (artifact upload/download, state branch) is workflow-specific; generalizing it to other workflows would require the same boilerplate to be added to each

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
