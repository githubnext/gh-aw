# ADR-30035: Codex Engine Node.js Harness with Retry Logic

**Date**: 2026-05-04
**Status**: Draft
**Deciders**: Unknown

---

## Part 1 — Narrative (Human-Friendly)

### Context

The system runs multiple AI agent engines (Claude, Copilot, Codex) as sub-processes inside GitHub Actions workflows. The existing Claude and Copilot engines both use a Node.js harness script (`claude_harness.cjs`, `copilot_harness.cjs`) that wraps the underlying CLI with retry logic, prompt file handling, and diagnostic logging. The Codex engine lacked this wrapper, invoking `codex exec` directly via an inline shell command with the prompt injected as a shell variable (`$INSTRUCTION`). The OpenAI Codex CLI is susceptible to transient API failures (HTTP 429 rate limits, 500/503 server errors) and does not provide a `--continue` session-resumption flag, meaning any transient failure results in total loss of the run with no recovery path.

### Decision

We will introduce `codex_harness.cjs`, a Node.js wrapper script that follows the established harness pattern used by the Claude and Copilot engines. The harness reads the prompt from a file (`--prompt-file`) instead of a shell variable, and retries up to three times on transient failures using exponential backoff (5 s → 10 s → 20 s, capped at 60 s). The `CodexEngine` Go struct implements the `HarnessProvider` interface by returning `"codex_harness.cjs"` from `GetHarnessScriptName()`, and `GetExecutionSteps()` is updated to invoke the harness via the shared `nodeRuntimeResolutionCommand` pattern.

### Alternatives Considered

#### Alternative 1: Inline Bash Retry Loop

A `while` / `until` retry loop could be added directly to the shell command already embedded in each `.lock.yml`. This avoids adding a new file but would be duplicated across many workflow files, is difficult to unit-test, and the retry-detection logic (parsing OpenAI error patterns from output) becomes fragile in a one-liner. This approach also does not address the `$INSTRUCTION` shell-variable injection, which has a size limit and quoting hazards for large prompts.

#### Alternative 2: Upstream Fix in the Codex CLI

Asking OpenAI/Codex to add native retry and `--continue` support would provide the cleanest solution but is outside our control and has an unpredictable timeline. The workflow reliability problem exists today and needs a local mitigation.

#### Alternative 3: Shell Wrapper Script (Plain `.sh`)

A POSIX shell script could implement the retry loop without requiring Node.js. However, the existing harness infrastructure is Node.js-based (process runner, AWF reflect helpers, structured logging to stderr). Introducing a Bash harness for one engine would create an inconsistency, lose the shared `process_runner.cjs` utilities, and make it harder to write fast, isolated unit tests.

### Consequences

#### Positive
- Transient rate-limit and server errors are automatically retried, improving overall workflow success rates for Codex-based agents.
- Prompt delivery switches from shell-variable injection (`$INSTRUCTION`) to file-based (`--prompt-file`), removing shell quoting hazards and size limitations for large prompts.
- Consistent harness pattern across all three agent engines simplifies future maintenance and onboarding.
- 23 unit tests covering prompt resolution, error-pattern detection, and retry policy provide regression coverage.

#### Negative
- All retries for Codex are fresh runs (not continuations), because the Codex CLI has no `--continue` flag. A partially completed run that fails transiently will restart from scratch, wasting tokens and time for the completed portion.
- Node.js must be present in the execution environment; the harness detects it via `GH_AW_NODE_BIN` or falls back to `command -v node`, adding a soft dependency that could fail silently on unusual runners.

#### Neutral
- Every compiled `.lock.yml` that uses the Codex engine is regenerated to switch from the direct `codex exec "$INSTRUCTION"` invocation to the harness-wrapped form; this is a mechanical change with no behavioral difference beyond the retry wrapper.
- The `--prompt-file` flag is a harness-only argument stripped before passing the remaining args to `codex exec`.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Harness Script

1. The Codex engine **MUST** invoke `codex exec` through `codex_harness.cjs` rather than calling it directly from a shell command.
2. The harness **MUST** accept a `--prompt-file <path>` argument, read the file contents, and append those contents as the final positional argument to `codex exec`.
3. The `--prompt-file` flag **MUST NOT** be forwarded to the `codex exec` subprocess.
4. The harness **MUST** retry a failed run up to 3 times when the subprocess produced output before exiting with a non-zero code (partial execution).
5. The harness **MUST** apply exponential backoff between retries, starting at 5 seconds, doubling on each attempt, and capping at 60 seconds.
6. The harness **MUST NOT** retry a run that produced no output before failing, as this indicates an unrecoverable error (e.g., authentication failure) rather than a transient one.
7. The harness **SHOULD** emit structured diagnostic log lines prefixed with `[codex-harness]` to stderr so they are distinguishable in aggregated logs.

### Engine Interface

1. `CodexEngine` **MUST** implement the `HarnessProvider` interface by returning `"codex_harness.cjs"` from `GetHarnessScriptName()`.
2. `CodexEngine.GetExecutionSteps()` **MUST** construct the execution command using `nodeRuntimeResolutionCommand` and the harness script name, consistent with the pattern used by other engines that implement `HarnessProvider`.
3. Compiled workflow lock files **MUST** reflect the harness invocation pattern and **MUST NOT** use the legacy `INSTRUCTION="$(cat ...)"` shell-variable injection for Codex.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25295059484) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
