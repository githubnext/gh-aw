# ADR-67797: Verify the startup model/endpoint mismatch path with a shared-fixture replay test

**Date**: 2026-10-11
**Status**: Draft
**Deciders**: Unknown *(generated from PR #67797 by the Design Decision Gate)*

---

### Context

The Copilot startup model/endpoint mismatch fail-fast path (#67487) spans four independent stages: the harness (`actions/setup/js/copilot_harness.cjs`), the error detector (`detect_agent_errors.cjs`), the session collector (`unified_session.cjs`), and the failure handler (`handle_agent_failure.cjs`), plus the Go audit renderer in `pkg/cli`. Each stage already had a unit test, but every test constructed its own copy of `agent/model-endpoint-mismatch.json`. That meant a renamed or dropped record field, a moved record path, or a harness that spawned the CLI anyway could all keep CI green while the user-visible failure issue silently lost its cause and fix. The contract crossing these stages is a file on disk shared between a JavaScript producer and a Go consumer, so no single-language unit test can observe a break in it.

### Decision

We will verify this path with a single end-to-end **replay test** that drives the real harness, detector, collector and failure handler in sequence, where each stage reads only the files the previous stage actually wrote, and we will drive both the JavaScript and the Go tests from one shared fixture directory (`pkg/cli/testdata/model_endpoint_mismatch_startup/`) captured from a real failing run. The primary driver is contract fidelity across the language boundary: a single source of truth for the record, the `/reflect` document and the session line means a schema change on either side must fail a test rather than silently diverge. Each case is paired with a negative control using the same fixture and environment so that the assertions prove behaviour rather than absence of setup.

### Alternatives Considered

#### Alternative 1: Keep per-stage unit tests with independently constructed records

This is the status quo: each stage asserts against a record it builds itself. It is cheap, fast and already in place, and it was a close call because the new test adds process spawning and preload indirection. It was rejected because duplicated fixtures are exactly the failure mode observed — the stages can drift apart while every individual test still passes.

#### Alternative 2: Add a shared schema/golden projection instead of a replay

A golden-file or schema test over the record type would catch field renames on both sides without spawning processes. It was considered and partly exists (the model-routing golden suite), but it cannot catch behavioural regressions such as the harness failing to `process.exit(1)` and spawning the CLI anyway, or the record being written to a different path. The PR explicitly validated these behavioural breakages against the new test.

#### Alternative 3: Rely on a live integration workflow run

Running the real workflow against a mismatched model would be the highest-fidelity check. It was rejected because it requires live engine credentials and a sandbox run that is not reproducible in CI, and because failures would be slow and flaky compared with a deterministic replay of recorded run data.

### Consequences

#### Positive

- A single fixture is the source of truth for the record, `/reflect` document and session line, so a rename or drop on either the JS or the Go side fails a test.
- Behavioural regressions are caught, not just schema ones: the stub CLI records spawns, so a harness that stops fail-fasting is detected.
- The failure-issue body is asserted line by line, so the user-visible cause and fix cannot regress unnoticed.
- The test runs inside the existing `cjs.yml` vitest shards and `cgo.yml` job — no new CI surface.

#### Negative

- The test spawns a subprocess with a `--require` preload to stub `fetchAWFReflect`, which is more fragile and slower than an in-process mock, and couples the test to the harness's CLI invocation shape.
- The fixture was reconstructed from issue data rather than copied from a private run, so the recorded `/reflect` document may omit fields the real document carries.
- `agent_output.json` for the failure handler is still hand-written, so that one hop in the chain is not genuinely replayed.
- Future changes to any of the four stages now require updating a shared fixture, raising the cost of intentional schema changes.

#### Neutral

- The fixture lives under `pkg/cli/testdata/` and is reached from JavaScript by a relative URL, creating a deliberate cross-tree dependency between `actions/setup/js/` and `pkg/cli/`.
- The model-routing golden suite is unchanged; this test is additive and does not replace existing per-stage unit tests.
- The approach establishes a reusable pattern for other cross-language, file-contract paths in this repository.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
