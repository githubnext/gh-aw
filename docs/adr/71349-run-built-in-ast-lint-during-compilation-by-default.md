# ADR-71349: Run Built-in Actions/AW AST Lint During Compilation by Default

**Date**: 2026-08-05
**Status**: Draft
**Deciders**: pelikhan

---

### Context

gh-aw has an in-process Actions/AW AST lint registry (see [ADR-69813](69813-in-process-ast-lint-registry-with-secret-leak-taint-analysis.md) and [ADR-70026](70026-map-ast-lint-rules-to-compiler-threat-ids.md)) that includes a `template-injection` rule. Until this PR, that registry ran only when someone invoked `gh aw lint` separately, so `gh aw compile` could emit lock files containing patterns the linter would have flagged. The trigger was the Copilot smoke workflow. It sent an agent-generated message to `haiku-printer`, which interpolated that message straight into Bash. Shell substitutions or injected syntax could then run on the downstream runner, and the existing AST rule would have caught it if compilation had run the linter. Other compile-time analyzers are mostly opt-in flags (`--zizmor`, `--poutine`, `--actionlint`). Shellcheck is already default-on ([ADR-49762](49762-default-on-shellcheck-for-run-step-linting-in-compile-pipeline.md)).

### Decision

We will run the built-in Actions/AW AST lint registry as a default step of CLI compilation. The step runs inside `generateAndValidateYAML`, after YAML generation and before lock-file emission, and it also runs with `--no-emit`. It checks both the generated Actions graph and the authored AW source document. Error-severity diagnostics fail compilation. Warnings go to stderr and count toward the warning total. The step honors active `threat-detection` suppressions and skips warning-level `secret-leak` diagnostics. Users can opt out with `--skip-lint` (MCP: `skip_lint`). The opt-out disables only this lint pass, not compiler validation or requested external scanners. Dry-run compilation rejects it. Low-level Go callers keep their existing opt-in behavior through `workflow.WithLint(true)`. The main reason for this decision is security: injection findings should block lock-file generation instead of depending on a separate, easily forgotten lint run.

### Alternatives Considered

#### Alternative 1: Keep linting opt-in via `--lint` or standalone `gh aw lint`

This matches the existing opt-in scanner flags and avoids a breaking change. It was rejected because the haiku-printer incident shows that opt-in analysis gets skipped in practice. The rule that would have caught the injection already existed but never ran during compilation.

#### Alternative 2: Fold the lint rules into existing compiler validation (no opt-out)

Making the AST rules part of mandatory compiler validation would give the strongest guarantee. It was not chosen because lint rules change faster and have a higher false-positive risk than core validation. Users need a narrow escape hatch (`--skip-lint`) that leaves the core compiler safety checks in place.

#### Alternative 3: Enable linting by default in the low-level `workflow.Compiler` API as well

This would give library callers the same protection. It was rejected to preserve the existing Go API behavior. Default-on applies only in the CLI/MCP setup path (`createAndConfigureCompiler`), and Go callers opt in with `WithLint(true)`. [TODO: verify this rationale with the deciders]

### Consequences

#### Positive
- Injection and other AST lint errors now block lock-file generation, closing the gap that let the haiku-printer injection through.
- `--no-emit` and MCP compile also run the lint, so validation-only flows get the same coverage.
- The opt-out is narrow and auditable. It cannot be used with dry-run, and it does not weaken compiler validation or external scanners.

#### Negative
- This is a breaking (major) change. Workflows that compiled before may now fail until their findings are fixed or `--skip-lint` is passed.
- Compilation does a bit more work: it parses the generated YAML into an Actions AST and runs the rule registry for every workflow.
- CLI and Go API defaults now differ (CLI on, `workflow.Compiler` off), which can surprise library callers.

#### Neutral
- Native Actions YAML and downstream dispatch targets still need standalone `gh aw lint`. Compilation does not lint dispatch targets recursively.
- Actions AST parse/emit had to keep `cache-mode`, and authored-environment analysis had to accept sandbox shorthands, so that the default lint runs cleanly on existing workflows.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
