# ADR-29534: Frontmatter A/B Experiments with Balanced Variant Selection

**Date**: 2026-05-01
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

## Part 1 — Narrative (Human-Friendly)

### Context

Agentic workflows compiled by gh-aw use a frontmatter-driven configuration model where the prompt template and runtime behaviour are declared in a Markdown header. Teams running these workflows wanted a first-class mechanism to test different prompt variants (e.g., tone, persona, feature flags embedded in the prompt) across successive workflow runs. Without such a mechanism, variant testing was ad-hoc, untracked, and statistically unbalanced. The system needed a solution that (1) could be declared entirely in the frontmatter, (2) required no external service dependency, and (3) guaranteed approximate balance across runs without user intervention.

### Decision

We will add an `experiments` map to the workflow frontmatter schema that associates named experiments with an ordered list of variant strings (e.g., `caveman: [yes, no]`). At runtime, the activation job selects the least-used variant for each experiment using a persisted counter stored in the GitHub Actions cache, sets individual step outputs (`steps.pick-experiment.outputs.<name>`), and exposes a combined JSON blob via the activation job's outputs. Prompt templates can reference experiment values via `${{ experiments.<name> }}` expression syntax and can guard blocks with `{{#if experiments.<name> }}` Handlebars conditionals, where the string `"no"` is treated as falsy. This approach is fully self-contained within the existing gh-aw compiler and Actions runtime.

### Alternatives Considered

#### Alternative 1: External Feature-Flag Service (e.g., LaunchDarkly or Unleash)

An external feature-flag service would provide a mature A/B testing API with dashboards, targeting rules, and statistical analysis. However, it introduces a hard external dependency (network egress at runtime, credential management, billing) that conflicts with the firewall-isolated execution model of gh-aw. It was rejected as incompatible with the sandboxed runner environment.

#### Alternative 2: Random Per-Run Variant Selection Without State Persistence

Selecting a variant randomly on each run (e.g., `Math.random()`) requires no cache and no persistent state. This was rejected as the sole selection strategy because it does not guarantee balance: over N runs, some variants may appear much more (or less) often than `N/K`, making statistically meaningful comparisons impossible without a large number of runs. The least-used counter approach achieves approximate balance in far fewer runs. However, random selection is retained as the tie-breaking strategy within the least-used algorithm — when multiple variants share the minimum count (including the initial empty-cache state), one is chosen at random to avoid systematically favouring the first declared variant.

#### Alternative 3: CI/CD Environment Variables Set Externally

Teams could manually pass variant values as repository variables or dispatch inputs. This was considered too brittle: it requires human coordination across runs, does not scale to multiple simultaneous experiments, and cannot auto-rotate variants.

### Consequences

#### Positive
- Experiments are declared alongside the workflow prompt — no external tooling required.
- Balanced round-robin selection (least-used variant) ensures statistically even distribution with minimal runs.
- Step summary and artifact upload provide per-run observability for downstream analysis (`gh aw audit --artifacts experiment`).
- Template conditionals (`{{#if experiments.name }}`) allow prompt sections to be toggled based on the selected variant.

#### Negative
- Experiment state depends on the GitHub Actions cache, which has size limits, eviction policies, and is scoped per repository branch — cross-branch state is not shared.
- The `"no"` string must be treated as falsy in `isTruthy`, introducing a special-case that differs from standard JavaScript truthiness and may surprise future contributors.
- Expression rewriting (`${{ experiments.name }}` → `steps.pick-experiment.outputs.name`) adds a non-obvious indirection layer in the compiler's expression extraction phase.
- Every workflow using experiments adds three steps to the activation job (restore cache, pick, save), increasing job duration and GitHub Actions billing slightly.

#### Neutral
- The `pick_experiment.cjs` script is bundled into the `actions/setup/js/` directory alongside other runtime helpers, following the existing pattern.
- Experiment env vars (`GH_AW_EXPERIMENTS_*`) must be propagated explicitly to the interpolation and substitution steps; adding a new experiment to an existing workflow requires recompiling the lock file.
- The feature requires no schema-breaking changes — `experiments` is an optional field; workflows without it are unaffected.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Frontmatter Schema

1. Workflow frontmatter **MAY** include an `experiments` field whose value is a YAML map from experiment name (string) to a non-empty list of variant strings.
2. Each experiment name **MUST** be a valid YAML key composed of alphanumeric characters and underscores only.
3. Each variant list **MUST** contain at least two entries; a single-element variant list **MUST NOT** be accepted by the compiler.
4. Implementations **MUST NOT** require the `experiments` field — its absence **MUST** produce no change in compiled output.

### Variant Selection

5. Implementations **MUST** select the variant with the lowest cumulative invocation count across all previous runs (least-used selection).
6. When two or more variants share the lowest count (including the initial state where all counts are zero), implementations **MUST** break ties by selecting uniformly at random from the tied variants, so no variant is systematically favoured on the first run or whenever counts are equal.
7. Variant counts **MUST** be persisted between workflow runs using the GitHub Actions cache, keyed by a combination of the sanitized workflow ID and the current run ID, with a restore-key prefix that matches any prior run for that workflow ID.
8. Implementations **MUST** expose each selected variant as a named step output (`steps.pick-experiment.outputs.<experiment-name>`) and **MUST** also set a combined JSON output (`steps.pick-experiment.outputs.experiments`) containing all variant assignments.
9. Implementations **MUST** upload the experiment state directory as an artifact named `experiment` (using `if: always()`) so that assignments are available for post-run analysis even when subsequent steps fail.

### Expression and Template Integration

10. The compiler **MUST** rewrite `${{ experiments.<name> }}` expressions in the frontmatter or prompt source to `steps.pick-experiment.outputs.<name>` during expression extraction, so the runtime value is injected by the Actions expression engine.
11. Each experiment **MUST** be surfaced as an environment variable named `GH_AW_EXPERIMENTS_<NAME>` (uppercased) in every workflow step that performs prompt interpolation or template substitution.
12. Implementations **MUST** substitute `__GH_AW_EXPERIMENTS_*__` placeholders in the raw prompt text before Handlebars template rendering (Step 2.5 of `interpolate_prompt.cjs`), so that `{{#if experiments.<name> }}` conditionals evaluate the actual runtime variant value.
13. The `isTruthy` helper **MUST** treat the string `"no"` as falsy in addition to standard falsy values (`""`, `"false"`, `"0"`, `undefined`, `null`).
14. Implementations **MUST NOT** evaluate `{{#if experiments.<name> }}` before the placeholder substitution step — raw placeholders **MUST NOT** be passed to the Handlebars engine.

### Activation Job Structure

15. When the `experiments` field is present in the frontmatter, the compiled activation job **MUST** include, in order: a cache restore step, the variant selection step, a cache save step (`if: always()`), and the artifact upload step (`if: always()`).
16. The activation job **MUST** expose a `needs.activation.outputs.experiments` output containing the full JSON variant assignment object for use by downstream jobs.
17. Implementations **MUST NOT** inject experiment steps into workflows that do not declare the `experiments` frontmatter field.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25222221666) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
