# ADR-29581: Claude Agent Retry Harness for Transient API Errors

**Date**: 2026-05-01
**Status**: Draft
**Deciders**: pelikhan

---

## Part 1 — Narrative (Human-Friendly)

### Context

The gh-aw platform runs Claude Code as an agentic worker inside sandboxed CI containers. Prior to this change, the Claude CLI was invoked directly via a shell one-liner embedded in each workflow's `awf` step. The Anthropic API returns HTTP 529 (overloaded) and HTTP 429 (rate-limited) under load, both of which are transient and retriable. Without retry logic, any such error caused the entire CI job to fail immediately, requiring a manual re-run and wasting accumulated agent progress. The Copilot engine already used a Node.js harness (`copilot_harness.cjs`) with similar concerns; the Claude engine had no equivalent wrapper.

### Decision

We will wrap the Claude Code CLI invocation in a dedicated Node.js harness (`claude_harness.cjs`) that monitors stdout for `overloaded_error` (HTTP 529) and `rate_limit_error` (HTTP 429) signals, retries up to three times with exponential backoff (5 s → 10 s → 20 s, capped at 60 s), and resumes partial sessions using `claude --continue` to avoid re-injecting the prompt. The harness mirrors the existing `copilot_harness.cjs` pattern, with shared AWF proxy reflection logic extracted into `awf_reflect.cjs`. The Go engine (`claude_engine.go`) is updated to delegate to `node claude_harness.cjs claude …` and to pass the prompt via `--prompt-file` rather than inline shell expansion.

### Alternatives Considered

#### Alternative 1: GitHub Actions step-level retry

GitHub Actions supports third-party `retry` actions (e.g., `nick-fields/retry`) that can re-run an entire workflow step. This was not chosen because a full step re-run restarts Claude from scratch, discarding any on-disk session state that `--continue` can resume, and it cannot distinguish transient API errors from fatal failures (e.g., authentication errors or a missing binary), causing unnecessary retries.

#### Alternative 2: Retry logic in the Go engine (`pkg/workflow/claude_engine.go`)

The retry loop could be implemented in Go directly, where the engine already constructs and launches the Claude subprocess. This approach was not chosen because it would conflate process-management concerns with error-classification logic that must parse Claude's streaming JSON output, and it would diverge from the established pattern where harness scripts (not the engine) own execution details — making it harder to evolve retry policy independently of the compiled binary.

### Consequences

#### Positive
- CI jobs survive transient Anthropic API overload and rate-limit errors without manual intervention.
- Partial-execution retries use `--continue`, preserving on-disk session state so Claude resumes where it left off rather than restarting from the beginning.
- Shared AWF reflection logic (`awf_reflect.cjs`) eliminates duplication between the Claude and Copilot harnesses.

#### Negative
- Each workflow invocation now has a Node.js process indirection layer; any crash or startup failure in the harness itself can mask the underlying Claude error.
- The retry heuristic (detecting error types via string matching on stdout) is brittle: if Anthropic changes its output format or error labels, retries will silently stop working.
- Introducing `--prompt-file` requires updating all existing tests that asserted the old inline `"$(cat …)"` shell expansion pattern.

#### Neutral
- The harness is distributed as a `.cjs` file alongside other setup scripts; it is not published as an npm package.
- The prompt is no longer passed as a positional CLI argument; `--prompt-file` is now the canonical mechanism for all Claude harness invocations.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Harness Invocation

1. Implementations **MUST** invoke the Claude Code CLI through `claude_harness.cjs` (or a custom harness specified in workflow frontmatter via `engine.harness`) rather than calling the `claude` binary directly.
2. Implementations **MUST** pass the agent prompt via `--prompt-file <path>` and **MUST NOT** embed the prompt inline as a shell-expanded positional argument (e.g., `"$(cat …)"`).
3. Implementations **MUST NOT** re-pass `--prompt-file` on retry runs that use `--continue`; the harness **MUST** omit the prompt argument when resuming a partial session.

### Retry Policy

1. The harness **MUST** detect `overloaded_error` (HTTP 529) and `rate_limit_error` (HTTP 429) error signals in Claude's stdout stream and treat them as retriable.
2. The harness **MUST** retry up to 3 times with exponential backoff starting at 5 seconds, doubling each attempt, capped at 60 seconds.
3. The harness **MUST** use `--continue` on retry attempts when the previous attempt produced partial output, to resume from on-disk session state.
4. The harness **MUST NOT** retry on non-transient failures (e.g., binary not found, authentication errors, zero-output exits that do not match a known transient error pattern).
5. The harness **SHOULD** log the error type and retry attempt number to stderr before sleeping.

### Shared AWF Reflection Module

1. All harnesses that require AWF API proxy reflection **MUST** import the shared `awf_reflect.cjs` module rather than duplicating its constants or functions inline.
2. `awf_reflect.cjs` **MUST** export at minimum: `AWF_API_PROXY_REFLECT_URL`, `AWF_REFLECT_OUTPUT_PATH`, `AWF_REFLECT_TIMEOUT_MS`, `AWF_MODELS_URL_TIMEOUT_MS`, `GEMINI_MODEL_NAME_PREFIX`, `extractModelIds`, `fetchModelsFromUrl`, `enrichReflectModels`, and `fetchAWFReflect`.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25228247320) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
