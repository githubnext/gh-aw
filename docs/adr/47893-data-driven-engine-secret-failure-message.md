# ADR-47893: Data-Driven Engine Secret Failure Message via Compiler-Emitted Env Var

**Date**: 2026-07-25
**Status**: Draft
**Deciders**: Unknown

---

### Context

When an agentic workflow's `validate-secret` step fails, the activation job fails and the agent job is **skipped** (not `failure`). The `handle_agent_failure.cjs` handler previously checked only `agentConclusion !== "failure"` and bailed out silently — never creating a failure issue. Additionally, when building the failure issue body, the handler contained a hardcoded `engineId === "copilot"` check to inject a Copilot-specific alternative message (use `permissions: copilot-requests: write` instead of a PAT). This hardcoding meant that adding support for a new engine's custom failure message required modifying the JavaScript handler itself, violating separation of concerns between the compiler and the runtime handler.

### Decision

We will extend the Go compiler's engine abstraction (`WorkflowExecutor`) with a `GetSecretFailureMessage(workflowData *WorkflowData) string` method. Each engine's compiler implementation overrides this method to return a custom message string (or `""` for engines with no alternative). The compiler emits the resolved message as the `GH_AW_ENGINE_SECRET_FAILURE_MESSAGE` environment variable in the conclusion job for any workflow compiled with a validate-secret step. The `handle_agent_failure.cjs` handler reads this env var instead of performing any engine-ID-specific logic, and also adds `hasSecretVerificationFailed` to its early-return guard so that skipped-but-failed-secret-verification runs are not silently ignored.

### Alternatives Considered

#### Alternative 1: Keep the Hardcoded `engineId === "copilot"` Check in the Handler

The existing approach reads `GH_AW_ENGINE_ID` and branches in JavaScript. It works for the current single special case but requires touching the handler for every new engine that needs a custom failure message. The handler becomes a growing switch/if-else as more engines are added, and it cannot be tested independently of a running workflow.

#### Alternative 2: Static Engine-to-Message Mapping in a Shared Config File

Maintain a JSON or YAML config file mapping engine IDs to their failure messages, committed alongside the handler. The handler reads the file at runtime. This keeps the handler generic but introduces a second file that must stay in sync with each engine's compiler implementation — an implicit coupling that is harder to enforce than a typed Go interface override.

### Consequences

#### Positive
- Engine-specific failure messages are co-located with their engine's Go compiler code; the interface constraint makes it impossible to add a new engine without explicitly deciding whether it has a custom message.
- The `handle_agent_failure.cjs` handler becomes engine-agnostic — no engine-specific branches remain, and it can be tested with any value of `GH_AW_ENGINE_SECRET_FAILURE_MESSAGE` without knowing engine details.
- The compiler output (lock files) is the canonical source of truth; the Go test suite (`TestCopilotEngineSecretFailureMessageWiredInConclusionJob`, `TestNonCopilotEngineHasNoSecretFailureMessage`) verifies correct emission.

#### Negative
- The failure message string (potentially multi-line markdown) is embedded verbatim in every compiled lock file — 101 lock files updated in this PR — increasing lock file size and making the message harder to update without recompiling all workflows.
- Adding or changing a message requires a full recompile and lock file regeneration, which touches many files and can create noisy diffs.

#### Neutral
- The `hasSecretVerificationFailed` guard addition to the early-return check is a correctness fix that is independent of the data-driven message approach; it would be needed under any design.
- Engines without a custom message (e.g. Claude) return `""` from `GetSecretFailureMessage()`, and the compiler omits the env var; the handler treats absence of the var the same as an empty string.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
