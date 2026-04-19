# ADR-27205: Integration Testing Interactive TUI Workflows via Tuistory

**Date**: 2026-04-19
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

The `add-wizard` command drives an interactive terminal UI (TUI) flow — it prompts the user for repository information and confirmation before applying file changes. Existing integration tests in `pkg/cli/` exercise non-interactive commands by running the compiled `gh-aw` binary in a subprocess; they cannot interact with prompts or validate TUI-driven behavior. As the wizard becomes a user-critical path, there is a need to verify that the interactive flow works end-to-end: prompts appear in the expected order, keyboard input is accepted correctly, and cancellation leaves the repository in a clean state.

### Decision

We will use [tuistory](https://www.npmjs.com/package/tuistory) — a Node.js TUI automation tool — via `npx` to drive the `add-wizard` TUI in integration tests. Tests launch a named tuistory session that wraps the `gh-aw` subprocess in a pseudo-terminal, then send keyboard events and poll for expected text output. This approach treats the TUI as a black box from the user's perspective, validating the full interactive experience without modifying production code to add test hooks.

### Alternatives Considered

#### Alternative 1: Unit-test TUI components with mocked prompts

The prompt library used by `add-wizard` could be wrapped behind an interface, and unit tests could inject a fake implementation that returns predefined responses. This approach runs fast and in-process, but it does not validate that the production prompt rendering or keyboard-event handling works correctly. A bug in the TUI library integration would not be caught, and the test would diverge from the real user experience over time. [TODO: verify which prompt library is used and whether it exposes a testing interface]

#### Alternative 2: Expect-style scripting with `expect` / `pexpect`

Traditional `expect` or Python `pexpect` scripts can drive interactive CLIs by matching on stdout patterns and sending text. This is a mature and widely understood technique. However, it introduces a Python dependency into a Go project, and `expect` patterns are fragile against minor changes in prompt wording or terminal control codes. Tuistory offers the same interaction model in JavaScript with a higher-level API and named sessions that simplify cleanup.

#### Alternative 3: Extend the CLI with a non-interactive mode for all prompts

The `add-wizard` could accept all inputs as flags (e.g., `--repo`, `--confirm`) so tests can bypass TUI prompts entirely. This would enable simple subprocess-based testing. However, it alters the production CLI surface and may encourage misuse of flags in scripts where the interactive flow is the intended UX. It also leaves the interactive TUI path untested.

### Consequences

#### Positive
- The happy path and cancellation path of the interactive wizard are validated against a real subprocess, catching regressions in prompt ordering, text, and keyboard handling.
- Tests are isolated: each test creates a fresh temporary directory with an initialized git repository, ensuring no cross-test pollution.
- Tuistory's named sessions enable deterministic cleanup via a deferred `close` call, even when the test panics or is cancelled.
- The test gracefully skips when `npx` or tuistory is unavailable, preventing false failures on developer machines without Node.js.

#### Negative
- Integration tests now require Node.js (`npx`) in the CI runner and on developer machines that wish to run them locally.
- Tuistory is an external JavaScript tool installed at test time via `npx -y`; it is not pinned to a specific version, introducing a risk of breaking changes from upstream.
- TUI-based tests are inherently slower and more timing-sensitive than unit tests; `waitForTuistoryText` uses polling with hardcoded timeouts (up to 120 seconds per step).
- Adding a dedicated CI job (`integration-add-wizard-tuistory`) increases overall pipeline time and resource consumption.

#### Neutral
- A shared `gh-aw` binary artifact is introduced in the `update` CI job and downloaded by the new integration job, requiring the `update` job to run before `integration-add-wizard-tuistory`. This changes the CI dependency graph.
- The `GH_AW_INTEGRATION_BINARY` environment variable allows CI to inject a pre-built binary path into `TestMain`, avoiding a redundant build. Developers who do not set this variable get the original local-build fallback.
- `TestTuistoryAddWizardIntegration` is excluded from the CLI catch-all matrix job to prevent duplicate execution.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Test Isolation

1. Each integration test **MUST** create its own temporary directory and initialize a fresh git repository within it before launching the wizard under test.
2. Tests **MUST** clean up their temporary directory on exit, using `defer os.RemoveAll(...)` or equivalent, regardless of test outcome.
3. Tests **MUST NOT** depend on or mutate the repository in which the test process is running.

### Tuistory Session Lifecycle

1. Each test **MUST** use a unique session name (e.g., derived from the test name and `time.Now().UnixNano()`) to avoid collisions when tests run in parallel.
2. Tests **MUST** defer a `tuistory -s <session> close` call immediately after a successful launch so the session is always terminated.
3. Tests **SHOULD** skip (not fail) when `npx` or tuistory is unavailable in the environment, using `t.Skip(...)` with an explanatory message.
4. Tests **MUST NOT** assume a specific tuistory version; behavior differences across minor versions **SHOULD** be handled by keeping assertions against stable, version-independent output strings.

### CI Integration

1. The `integration-add-wizard-tuistory` CI job **MUST** depend on the `update` job and download the `gh-aw-linux-amd64` artifact rather than building the binary independently.
2. The CI job **MUST** set `GH_AW_INTEGRATION_BINARY` to the downloaded binary path so `TestMain` uses it without triggering a local build.
3. `TestTuistoryAddWizardIntegration` **MUST** be excluded from any catch-all CLI matrix job via `skip_pattern` to ensure it runs only in its dedicated job.
4. Test results **MUST** be uploaded as a CI artifact using the naming convention `test-result-integration-<suite-name>` with a retention period of at least 14 days.

### Pre-built Binary Injection

1. `TestMain` **MUST** check the `GH_AW_INTEGRATION_BINARY` environment variable before attempting a local build.
2. If the environment variable is set, `TestMain` **MUST** verify the file exists and is accessible; if not, it **MUST** panic with a descriptive error rather than silently falling back to a local build.
3. When the environment variable is absent, `TestMain` **MAY** build the binary locally using `make build`.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24633266047) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
