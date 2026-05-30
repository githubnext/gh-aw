# ADR-29123: Compile-Time Validation for Dangerous Shell Expansion in safe-outputs Steps

**Date**: 2026-04-29
**Status**: Draft
**Deciders**: pelikhan

---

## Part 1 — Narrative (Human-Friendly)

### Context

The safe-outputs security harness blocks a set of dangerous bash expansion patterns at runtime — specifically `${var@operator}` (parameter transformation), `${!var}` (indirect expansion), `$(...)` (command substitution), and backtick substitution — to prevent shell injection attacks in GitHub Actions steps that post results to GitHub APIs. Prior to this change, workflow authors had no compile-time feedback when their `run:` scripts contained these patterns; the first signal was a confusing runtime failure when the harness silently rejected the step. The Copilot PR NLP Analysis workflow exposed this gap when it used `$(upload_asset ...)` command substitution to capture chart upload URLs inside a safe-outputs step.

### Decision

We will add a compile-time validator (`validateSafeOutputsStepsShellExpansion`) that scans every `run:` script in `safe-outputs.steps[]` for the four blocked shell expansion patterns and fails compilation with a descriptive error before the workflow reaches runtime. This closes the feedback loop between the runtime harness rules and the authoring experience, making violations detectable and actionable during `gh aw compile` instead of at execution time.

### Alternatives Considered

#### Alternative 1: Improved Runtime Error Messages Only

The runtime harness already blocked these patterns; the simplest fix was to enrich the runtime error message with remediation guidance rather than adding a compile step. This was not chosen because it leaves the feedback loop long: an author must deploy and run the workflow before seeing the error, and the agent that runs the workflow may have already produced non-idempotent side effects (uploaded assets, posted partial comments) before the step fails.

#### Alternative 2: Allowlist Specific Patterns in the safe-outputs Harness

An alternative was to relax the harness to permit specific safe uses of `$(...)` (e.g., `$(cat /tmp/file.txt)`) rather than blocking all command substitution. This was not chosen because it significantly increases the attack surface of the harness — distinguishing safe from unsafe command substitutions reliably requires a full shell parser — and the pattern of writing intermediate values to files and reading them with `cat` in the safe-outputs step is a viable and already-supported escape hatch.

### Consequences

#### Positive
- Workflow authors receive an actionable compiler error with the offending snippet and remediation guidance, instead of a cryptic runtime failure.
- The compiler and the runtime harness stay in sync: any new pattern blocked by the harness can be added to the validator in the same change.

#### Negative
- The validator must be maintained in tandem with the runtime harness. If the harness gains or removes a blocked pattern without updating the validator, they drift out of sync.
- The regex-based approach requires post-match filtering for false positives (arithmetic `$((`, GitHub Actions `${{ }}` expressions) because Go's RE2 engine does not support lookaheads. Edge cases in the filtering logic can yield false positives or false negatives.

#### Neutral
- The validator is registered in `compiler_validators.go` after `validateSafeOutputsMax`, making it part of the standard compilation pipeline with no change to the CLI interface.
- The companion fix to the Copilot PR NLP Analysis prompt (writing URLs to files instead of using `$(...)`) demonstrates the recommended pattern that workflow authors should follow.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Shell Expansion Validation

1. The compiler **MUST** validate every `run:` script in `safe-outputs.steps[]` for dangerous shell expansion patterns before emitting a compiled workflow artifact.
2. The compiler **MUST** reject any `run:` script that contains `${var@operator}` (bash parameter transformation), `${!var}` (indirect expansion), `$(...)` (command substitution), or `` `...` `` (backtick command substitution).
3. The compiler **MUST NOT** reject `run:` scripts that contain only `$VAR`, `${VAR}` (without operators), or `${{ expression }}` GitHub Actions template expressions.
4. The compiler **MUST NOT** reject `run:` scripts that contain `$((` arithmetic expansion.
5. Compiler error messages **MUST** include the zero-based step index, the offending snippet (truncated to 60 characters), and remediation guidance directing the author to write dynamic values to files in `/tmp/gh-aw/agent/` and read them with `cat` in the safe-outputs step.

### Safe-Outputs Step Authoring

1. Workflow authors **MUST NOT** use command substitution (`$(...)` or backticks) inside `safe-outputs.steps[].run` scripts to capture dynamic values.
2. Workflow authors **MUST NOT** use indirect expansion (`${!var}`) or parameter transformation (`${var@operator}`) in `safe-outputs.steps[].run` scripts.
3. When a safe-outputs step needs a value computed during the agent turn (e.g., an uploaded asset URL), the agent **MUST** write that value to a file in `/tmp/gh-aw/agent/` during its regular bash turn, and the safe-outputs step **MUST** read it using `cat` without shell expansion.
4. Workflow authors **SHOULD** use Python scripts to assemble complex multi-value payloads (e.g., discussion bodies) before passing them to a safe-outputs step, inserting literal strings rather than shell variables.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25114923934) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
