# ADR-54678: Deterministic Trace Grading Framework with Single Preprocessing Pass and vm Sandbox

**Date**: 2026-08-22
**Status**: Draft
**Deciders**: Unknown

---

### Context

The gh-aw agent job produces execution traces (token usage JSONL, MCP gateway logs, agent output JSON) that describe how an agent ran: how many LLM requests were made, how many tool calls succeeded, how many retries occurred, and how long execution took. Before this change, no systematic, deterministic mechanism existed to compute behavioral metrics from these traces. Evaluation was ad-hoc and relied on manual inspection. The system requires metrics that are byte-identical for equivalent inputs so downstream detection jobs and artifact diffs can rely on them without accounting for nondeterminism (timestamps, randomness, or LLM stochasticity).

### Decision

We will implement a deterministic trace grading framework (`trace_graders.cjs`) that performs a single preprocessing pass over all trace files at the start of each grading run and shares the resulting in-memory `PreprocessedTrace` object with all graders. Built-in graders are pure functions of that object. Custom (user-supplied) inline graders run inside a `node:vm` sandbox with a frozen copy of the trace, with access to `Math`, `JSON`, `Array`, `Object`, and a small `helpers` API, and with `Date`, `Math.random`, `require`, `process`, and `fetch` excluded. Output is written to `grader_results.json` with no timestamp field, making results deterministically byte-equivalent for identical inputs.

### Alternatives Considered

#### Alternative 1: LLM-Based Evaluation

Use a secondary LLM call after the agent run to grade behavior from logs. Considered because it could handle open-ended behavioral judgements beyond simple metrics. Rejected because LLM outputs are nondeterministic (same input rarely yields byte-identical output), add significant latency and cost per run, and are unavailable in sandboxed or offline environments. Determinism is a hard requirement for artifact diffing and detection staging.

#### Alternative 2: Per-Grader File Reads

Have each grader independently open and parse the trace files it needs. Considered because it simplifies the grader interface (each grader is fully self-contained). Rejected because it leads to redundant I/O proportional to grader count, makes it harder to enforce consistent parsing (e.g., JSONL size limits, malformed-line handling), and complicates sandboxing of custom graders (each would need its own file-access surface). The single-pass architecture keeps parsing logic in one place and is more efficient at runtime.

#### Alternative 3: Child-Process Isolation for Custom Scripts

Run each custom grader script in a separate child process with a restricted environment. Considered because it provides stronger OS-level isolation than `node:vm`. Rejected because `node:vm` with `codeGeneration: {strings: false, wasm: false}` and a frozen sandbox context is sufficient for the threat model (trusted repository authors running in an already-sandboxed CI environment), and avoids the latency, IPC overhead, and process-management complexity of spawning child processes per grader. The 5-second timeout and frozen trace provide adequate guardrails.

### Consequences

#### Positive
- Grader output (`grader_results.json`) is byte-deterministic for identical trace inputs, enabling reliable artifact diffs and detection-pipeline comparisons.
- The single preprocessing pass is O(1) in file I/O regardless of grader count — adding more graders does not add more disk reads.
- The `node:vm` sandbox prevents custom scripts from accessing the filesystem, network, or Node.js globals (`require`, `process`, `fetch`, `Date`), containing the blast radius of malicious or buggy user-supplied graders.
- Nine built-in graders (tool success rate, failure count, retries, loops, trajectory efficiency, step count, duration, context growth, artifact production) are available out of the box with no configuration.

#### Negative
- `node:vm` is not equivalent to OS-level process isolation; V8 sandbox escapes (though rare and typically patched quickly) could theoretically allow a custom script to access the host process. Operators with stricter security requirements should audit custom grader scripts before enabling them.
- The frozen trace is deep-cloned via `JSON.parse(JSON.stringify(...))`, so non-JSON-serializable trace fields (Dates, Buffers, circular references) are silently dropped. Graders cannot receive richer data types without extending the preprocessing layer.
- Custom scripts are synchronous and bounded by a 5-second VM timeout; long-running or async computations are not supported.

#### Neutral
- Grader output files (`grader_manifest.json`, `grader_results.json`) are written to `/tmp/gh-aw/agent/graders/` and copied into the detection staging directory, making them available to downstream jobs without changes to the artifact upload structure.
- Schema registration, canonical manifest/result metadata, and threshold configuration are deferred to follow-up work; this PR establishes the runtime and built-in grader set only.
- The grader step runs as `if: always()` after the agent, integrated via `parseGradersFromFrontmatter` in the workflow compiler — enabling opt-in via YAML frontmatter without modifying the core agent job.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
