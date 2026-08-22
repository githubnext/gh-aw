# ADR-54678: Deterministic Trace Grading Framework with Isolated Custom Script Execution

**Date**: 2026-08-22
**Status**: Accepted
**Deciders**: gh-aw maintainers

---

### Context

The gh-aw agent job produces execution traces (token usage JSONL, MCP gateway logs, agent output JSON) that describe how an agent ran: how many LLM requests were made, how many tool calls succeeded, how many retries occurred, and how long execution took. Before this change, no systematic, deterministic mechanism existed to compute behavioral metrics from these traces. Evaluation was ad-hoc and relied on manual inspection. The system requires metrics that are byte-identical for equivalent inputs so downstream detection jobs and artifact diffs can rely on them without accounting for nondeterminism (timestamps, randomness, or LLM stochasticity).

### Decision

We will implement a deterministic trace grading framework (`trace_graders.cjs`) that performs a single preprocessing pass over all trace files at the start of each grading run and shares the resulting in-memory `PreprocessedTrace` object with all graders. Built-in graders are pure functions of that object. Custom (user-supplied) inline graders execute in a dedicated worker subprocess (`trace_graders_worker.cjs`) that evaluates scripts in a restricted `node:vm` context with timeout enforcement around invocation and a serialized, frozen trace/config payload. This isolates grader execution from the main grader process and prevents direct access to host globals (`process`, `require`, `fetch`, `Date`) in the main runtime. Output is written to `grader_results.json` with no timestamp field, making results deterministically byte-equivalent for identical inputs.

### Alternatives Considered

#### Alternative 1: LLM-Based Evaluation

Use a secondary LLM call after the agent run to grade behavior from logs. Considered because it could handle open-ended behavioral judgements beyond simple metrics. Rejected because LLM outputs are nondeterministic (same input rarely yields byte-identical output), add significant latency and cost per run, and are unavailable in sandboxed or offline environments. Determinism is a hard requirement for artifact diffing and detection staging.

#### Alternative 2: Per-Grader File Reads

Have each grader independently open and parse the trace files it needs. Considered because it simplifies the grader interface (each grader is fully self-contained). Rejected because it leads to redundant I/O proportional to grader count, makes it harder to enforce consistent parsing (e.g., JSONL size limits, malformed-line handling), and complicates sandboxing of custom graders (each would need its own file-access surface). The single-pass architecture keeps parsing logic in one place and is more efficient at runtime.

#### Alternative 3: In-Process `node:vm` Execution

Run custom scripts directly in the main grader process with `node:vm` only. Considered because it is simpler and avoids subprocess startup overhead. Rejected because in-process execution leaves the grading controller and summary writer in the same trust boundary as untrusted script evaluation, making runtime-safety and escape-impact concerns harder to contain. We prefer an isolated subprocess boundary even with modest overhead.

### Consequences

#### Positive
- Grader output (`grader_results.json`) is byte-deterministic for identical trace inputs, enabling reliable artifact diffs and detection-pipeline comparisons.
- The single preprocessing pass is O(1) in file I/O regardless of grader count — adding more graders does not add more disk reads.
- The custom-grader subprocess boundary contains failures and sandbox escapes to a short-lived worker process, reducing impact on the main grading controller.
- Nine built-in graders (tool success rate, failure count, retries, loops, trajectory efficiency, step count, duration, context growth, artifact production) are available out of the box with no configuration.

#### Negative
- The worker still uses `node:vm`; a sandbox escape could compromise the worker process. This is stronger than pure in-process execution but not equivalent to a hardened container or VM boundary.
- The frozen trace is deep-cloned via `JSON.parse(JSON.stringify(...))`, so non-JSON-serializable trace fields (Dates, Buffers, circular references) are silently dropped. Graders cannot receive richer data types without extending the preprocessing layer.
- Custom scripts are synchronous and bounded by timeout enforcement; long-running or async computations are not supported.

#### Neutral
- Grader output files (`grader_manifest.json`, `grader_results.json`) are written to `/tmp/gh-aw/agent/graders/` and copied into the detection staging directory, making them available to downstream jobs without changes to the artifact upload structure.
- Schema registration, canonical manifest/result metadata, and threshold configuration are deferred to follow-up work; this PR establishes the runtime and built-in grader set only.
- The grader step runs as `if: always()` after the agent, integrated via `parseGradersFromFrontmatter` in the workflow compiler — enabling opt-in via YAML frontmatter without modifying the core agent job.

---

*This ADR is accepted with follow-up work tracked in PR review threads for schema hardening and detection integration.*
