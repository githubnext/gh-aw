# ADR-43413: Classify SDK Session-Idle Timeouts as Agentic Engine Timeout

**Date**: 2026-07-05
**Status**: Accepted
**Deciders**: @pelikhan, @copilot

---

### Context

The `detect_agent_errors` module (`actions/setup/js/detect_agent_errors.cjs`) classifies Copilot engine log output into typed error categories consumed by downstream workflows via `steps.detect-agent-errors.outputs.agentic_engine_timeout`. The original `AGENTIC_ENGINE_TIMEOUT_PATTERN` only matched signal-based termination (`signal=SIGTERM|SIGKILL|SIGINT`), which occurs when GitHub Actions' `timeout-minutes` cancels a step. A second timeout form—the Copilot SDK emitting `Timeout after <n>ms waiting for session.idle` during long runs—was not matched, causing those failures to go unclassified and skewing failure accounting in all consuming workflows.

### Decision

We will extend `AGENTIC_ENGINE_TIMEOUT_PATTERN` with an alternation to also match the SDK idle-timeout log signature (`Timeout after \d+ms waiting for session.idle`). Both timeout signatures are unified in a single regex so all timeout forms produce `agenticEngineTimeout: true` without introducing a new output flag or secondary detection path. The comment block above the constant is updated to document both covered cases.

### Alternatives Considered

#### Alternative 1: Add a Separate `sdkIdleTimeout` Output Flag

Introduce a distinct `sdkIdleTimeout` boolean alongside `agenticEngineTimeout`, keeping the two timeout kinds separately classifiable by consumers. This would allow finer-grained downstream handling but requires every consuming workflow to add a new condition for a case that semantically is still "the run timed out," adding consumer coupling with no practical benefit for current use cases.

#### Alternative 2: Handle Idle-Timeout Upstream in the SDK Driver

Remap the SDK's `session.idle` error to a signal exit before it reaches the log output seen by `detect_agent_errors` (e.g., by wrapping it in the driver's close handler). This avoids changing the classifier but pushes responsibility into the engine driver layer, is harder to test in isolation, and would not retroactively fix the misclassification in logs already emitted by deployed versions of the driver.

### Consequences

#### Positive
- Downstream workflows consuming `agentic_engine_timeout` now receive the correct flag for SDK idle-timeout runs, eliminating silent misclassification.
- Timeout detection remains centralized in a single module; no consuming workflow changes are required.

#### Negative
- The regex alternation slightly increases pattern complexity; reviewers must understand both timeout forms to audit the pattern correctly.
- If the Copilot SDK changes the format of its idle-timeout message in a future version, the pattern will silently stop matching, requiring ongoing maintenance awareness.

#### Neutral
- Regression tests are added for both the direct `AGENTIC_ENGINE_TIMEOUT_PATTERN.test()` match and the `detectErrors()` return value, establishing explicit coverage for this timeout signature.

---
