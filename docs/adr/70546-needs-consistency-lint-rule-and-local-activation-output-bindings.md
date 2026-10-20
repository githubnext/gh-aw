# ADR-70546: Add a `needs-consistency` Lint Rule and Bind Activation Prompt Outputs to Local Steps

**Date**: 2026-10-20
**Status**: Draft
**Deciders**: Not recorded (PR #70546 author: pelikhan)

---

### Context

In GitHub Actions, a `needs.<job>.outputs.<name>` or `needs.<job>.result` expression only resolves when `<job>` is a *direct* dependency of the job that uses it. If the job is only a transitive dependency, is not a dependency at all, or does not declare the output, the expression silently evaluates to an empty value instead of failing. The repository had hit this problem: the compiled activation job's prompt steps referenced `needs.activation.outputs.label_command` (and similar outputs such as `steering_issue_number`). A job cannot read its own outputs through `needs`, so those prompt values were always empty. The `ace-editor` workflow also gated a job on `needs.activation.outputs.activated`, which the activation job does not declare. The in-process Actions/AW linter (`pkg/workflow/lint`) already has a default rule registry, but no rule checked dependency-graph or `needs` expression consistency.

### Decision

We will add a built-in `needs-consistency` rule to the default lint registry (`Default()` in `lint.go`). It targets both the Actions AST and the AW AST and reports at **Warning** severity. The rule:

- flags unknown dependencies, self-dependencies, and cycles;
- parses `needs` accesses with actionlint's expression parser, covering dot and literal bracket forms and bare job/step `if:` conditions;
- requires each referenced producer to be a direct `needs` edge, to use only the `outputs` or `result` properties, and to declare any output it references.

When a compiled Actions graph is available, it takes precedence over authored AW source. Cases the rule cannot resolve statically are skipped rather than reported, so it does not produce false positives on them. These cases are unresolved imports, compiler-generated default edges (such as the implicit `activation` dependency), built-in jobs, computed indexes, and outputs of external reusable workflows.

We will also change the compiler so that `needs.activation.outputs.X` references inside the activation job's own prompt are rewritten to the local step expressions bound to those outputs, through a shared `rewriteExpressionReferences` helper. We will remove the undeclared `activated` condition from ACE. Finally, a repository test will require every repository workflow to stay free of `needs-consistency` findings.

### Alternatives Considered

#### Alternative 1: Rely on actionlint or runtime behaviour only

We could keep relying on external `actionlint` runs and runtime behaviour to catch broken `needs` references. This avoids new rule code. However, the empty-value failure is silent at runtime. The repository already compiled and shipped workflows with self-referencing activation outputs, which shows the existing checks were not catching the problem. The existing checks also do not apply to authored AW markdown before compilation, and they are not available in the in-process or WASM lint path. This alternative was rejected.

#### Alternative 2: Make the rule an Error, or check only the compiled Actions graph

We could fail compilation on any finding, or lint only compiled `.lock.yml` graphs, where the full dependency graph is known. An error-level rule would give a stronger guarantee. However, the AW source can hide edges that are added later by imports and compiler defaults. Strict enforcement on authored source would produce false positives, while checking only compiled output would delay feedback until after compilation. The PR uses Warning severity, defers ambiguous cases, and adds a repository-level test that requires zero findings. This was a trade-off between strictness and false-positive risk.

#### Alternative 3: Fix activation prompts by adding a separate job or mirroring outputs into env

We could fix the self-output problem by moving prompt rendering into a downstream job that `needs: activation`, or by copying each output into an environment variable. However, a downstream job would change the job topology, adding latency and another runner. Mirroring outputs would duplicate the compiler's output declarations. Rewriting references to the step expressions that already back each output (wrapped in parentheses so operator precedence is kept, and with literal values turned into quoted string literals) fixes the problem in place without changing the generated graph.

### Consequences

#### Positive
- Expressions that reference transitive, missing, or undeclared producers are now reported by `gh aw lint` and the default linter for both `.md` and `.yml` workflows, instead of silently evaluating to empty.
- Activation prompt placeholders (for example label command and steering issue number) now receive real values from local step outputs instead of empty strings.
- The experiment-name rewriter and the activation-output rewriter now share one expression-aware helper, `rewriteExpressionReferences`, which skips quoted literals and property chains.
- A repository-wide test (`TestRepositoryAgenticWorkflows`) keeps all repository workflows free of these findings.

#### Negative
- Many workflow lock files had to be regenerated (about 60 `.lock.yml` diffs), and the existing ACE condition was removed. The ACE job now relies only on Actions' default rule that its direct dependency must succeed.
- The skip rules (imports, built-in jobs, compiler default `activation` edges, `safe-jobs` ID normalization, reusable-workflow outputs) encode knowledge of how the compiler behaves. They must be kept in sync as the compiler changes, or the rule will start producing false positives or missing real problems.
- Because findings are warnings, they do not block compilation for users outside this repository's test suite.
- `rewriteActivationPromptOutputs` returns a new compile error if an activation output binding is neither an expression nor a YAML string.

#### Neutral
- The rule does not flag ordering-only dependencies (jobs that are listed in `needs` but whose outputs are never referenced).
- Computed indexes and filter expressions (for example `needs[matrix.x]`, `needs.*`) are not resolved, but nested `needs` accesses inside them are still checked.
- The performance benchmark fixtures were changed to keep producer jobs, so the new rule is measured on realistic graphs.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
