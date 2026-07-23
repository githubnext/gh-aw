# ADR-47517: PTY-Backed Integration Test Harness for Add-Wizard Manifest Bootstrap Ordering

**Date**: 2026-07-23
**Status**: Draft
**Deciders**: Unknown

---

### Context

PR #47462 changed `add-wizard` to surface `aw.yml` config actions before the generic engine setup flow, ensuring pre-install bootstrap steps (such as repo-variable prompts) are presented to users before engine selection. This ordering invariant was verified only by unit tests, which cannot simulate a real interactive terminal. Validating the terminal-boundary behavior — that prompts appear in the correct order on a real PTY — requires integration tests that exercise the full `add-wizard` binary under a live interactive session. The existing tuistory-based tests require `npx`, live `gh` authentication, and network access, making them unsuitable for isolated environments.

### Decision

We will introduce a PTY-backed integration test harness using `github.com/creack/pty` to drive the `add-wizard` binary in a real pseudo-terminal, combined with a fake `gh` CLI stub that satisfies the binary's GitHub API calls without external network access. Tests set up a temporary git repository containing a local package fixture with a known `aw.yml` manifest and assert on the order in which prompts appear in the PTY output stream.

### Alternatives Considered

#### Alternative 1: Extend the Existing Tuistory-Based Tests

The project already uses `tuistory` to run add-wizard sessions against a real interactive terminal. Extending those tests to cover the manifest bootstrap case would reuse existing infrastructure. However, tuistory tests require live `gh` authentication and `npx` availability; they cannot run in environments without external network access or a real GitHub login, making them unsuitable as the sole regression guard for this ordering invariant. This alternative was not chosen because the new test scenario must be isolatable from external state.

#### Alternative 2: Assert Ordering via Unit Tests with a Fake Terminal Writer

Unit tests could mock the `io.Writer` passed to the wizard and assert on the sequence of strings written. This avoids a real PTY and any external tooling. However, terminal prompt rendering in interactive CLI frameworks is sensitive to whether the output is an actual terminal; a fake writer changes the code path (skipping interactive prompts entirely in CI-like environments) and therefore cannot confirm that the visual ordering of prompts at the real terminal boundary is correct. This alternative was not chosen because it would not catch regressions in the actual interactive flow.

### Consequences

#### Positive
- Provides a regression guard that exercises the full binary at the terminal boundary, catching ordering bugs that unit tests cannot detect.
- Tests are self-contained: a fake `gh` stub and a temporary git repository eliminate all external dependencies, so the test suite runs in offline or restricted environments.

#### Negative
- Introduces `github.com/creack/pty` as a direct test dependency; PTY allocation is Unix-specific, so the new test cannot run on Windows.
- The PTY harness adds non-trivial test infrastructure (mutex-protected buffer, done channel, interrupt helper) that future contributors must understand and maintain.

#### Neutral
- The fake `gh` CLI stub hard-codes specific GitHub API responses; tests that exercise code paths relying on responses outside the stub's scope will emit an error and exit non-zero, requiring stub expansion as the wizard evolves.
- The test requires the compiled `add-wizard` binary to be present (via `globalBinaryPath`), linking it to the build step rather than running against library-level interfaces.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
