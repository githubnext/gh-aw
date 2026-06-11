# ADR-38544: Cap MCP Server Child Process Concurrency with a Shared Semaphore Guardrail

**Date**: 2026-06-11
**Status**: Accepted — the Go MCP server now enforces a shared 4-process subprocess cap.
**Deciders**: pelikhan, Copilot

## Context

The Go MCP server shells out to `gh` and `gh aw` child processes to service tool calls (workflow add/update/fix, logs, audit, diff, compile, inspect), as well as for supporting operations such as repository lookup, actor permission checks, config validation, and the `gh version` startup probe. Under concurrent tool usage these subprocess invocations were unbounded: every in-flight request that needed a child process spawned one immediately, so a burst of concurrent requests could fan out an arbitrary number of `gh`/`gh aw` processes and exhaust host resources (file descriptors, memory, process table). There was no central place enforcing a ceiling, and each call site managed its own `cmd.Output()` / `cmd.CombinedOutput()` independently. This change is preventive hardening: the issue was to add an explicit server-side guardrail before resource pressure showed up as flaky or host-specific failures.

## Decision

We will cap the number of simultaneously-active server-managed child processes at **4** using a single shared, context-aware guardrail. The guardrail (`mcpSubprocessGuardrail`) is a buffered-channel semaphore: each subprocess acquires one slot before executing and releases it when done. Acquisition is context-aware, so a cancelled request stops waiting rather than blocking behind queued subprocesses. All subprocess call sites route through the shared `defaultMCPSubprocessGuardrail` via the helpers `runMCPSubprocessOutput`, `runMCPSubprocessCombinedOutput`, `runMCPExecOutput`, and `runMCPExecCombinedOutput`, replacing direct `cmd.Output()`/`cmd.CombinedOutput()` calls. Output contracts and tool behavior are unchanged; only concurrency is bounded.

## Alternatives Considered

### Alternative 1: Leave subprocess spawning unbounded
Keep relying on the operating system and `gh`'s own behavior to absorb concurrent load. Rejected because the failure mode (resource exhaustion under a burst of concurrent tool calls) is silent and hard to diagnose, and the server has no backpressure mechanism of its own once limits are hit.

### Alternative 2: Per-call-site or per-tool limits
Give each tool or call site its own concurrency limit rather than one shared cap. Rejected because the resource pressure is global — total live child processes is what matters — so independent per-tool counters could still sum well past a safe ceiling, and would duplicate limiting logic across many files.

### Alternative 3: Explicit worker pool / job queue
Introduce a fixed pool of worker goroutines that own subprocess execution, with requests submitting jobs to a queue. Rejected as heavier than needed: it changes the execution model and call-site ergonomics, whereas a buffered-channel semaphore achieves the same bound with a minimal, drop-in wrapper around existing `exec.Cmd` calls.

## Consequences

### Positive
- Total concurrent server-managed child processes is bounded at 4, preventing unbounded fan-out and the associated resource exhaustion.
- A single chokepoint (`defaultMCPSubprocessGuardrail`) centralizes the limit; future call sites that use the helpers are covered automatically.
- Slot acquisition respects `context` cancellation, so cancelled or timed-out requests do not hang waiting for a slot.

### Negative
- The limit `4` is a hardcoded constant (`maxActiveMCPChildProcesses`) and is not configurable per host or deployment; tuning requires a code change. That is intentional for this first guardrail because the goal is to add one deterministic ceiling without introducing new user-facing configuration surface.
- Under high concurrency, requests block while waiting for a free slot, which can increase tail latency for tool calls that previously ran immediately.
- Correctness depends on every subprocess call site using the guardrail helpers; a direct `cmd.Output()`/`cmd.CombinedOutput()` added later would silently bypass the cap.

### Neutral
- The guardrail is process-global state (a package-level `defaultMCPSubprocessGuardrail`), shared across all MCP requests in the server process.
- Existing stdout/stderr separation (using `Output()` vs `CombinedOutput()` for JSON-producing commands) is preserved through distinct helper variants.
