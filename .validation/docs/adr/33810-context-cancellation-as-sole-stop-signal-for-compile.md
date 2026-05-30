# ADR-33810: Context Cancellation as the Sole Stop Signal for the Compile Command

**Date**: 2026-05-21
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

`gh aw compile` already wired SIGINT to a cancellable `context.Context` in `main.go` via `signal.NotifyContext`, but Ctrl+C had no observable effect once compilation was in progress: the per-file loops in `compileSpecificFiles` and `compileAllFilesInDirectory` never inspected `ctx.Done()` between iterations, so the command kept compiling every remaining workflow file before exiting. Watch mode (`watchAndCompileWorkflows`) compounded the problem by installing its own `signal.Notify(sigChan, SIGINT, SIGTERM)` handler and selecting on that channel — duplicate signal handling that competed with the context cancellation set up at the top of the process. The result was two parallel cancellation mechanisms (context-based in `main.go`, signal-channel-based in the watch loop) and several long-running loops that observed neither. The decision is whether to plumb signal channels deeper, unify on the context that already exists, or leave both mechanisms in place.

### Decision

We will treat the `context.Context` threaded through the compile pipeline as the **single, authoritative stop signal** for both batch and watch modes. The per-file loops in `compileSpecificFiles` and `compileAllFilesInDirectory` will guard every iteration with a non-blocking `select { case <-ctx.Done(): ...; default: }` and return `ctx.Err()` on cancellation. `watchAndCompileWorkflows` will delete its local `signal.Notify` handler and switch its main `select` arm from `<-sigChan` to `<-ctx.Done()`, so the same context cancellation that signals batch compile also tears down the watcher. Signal-to-context conversion is the sole responsibility of `main.go` (via `signal.NotifyContext`); nothing downstream installs its own signal handler.

### Alternatives Considered

#### Alternative 1: Thread a `chan os.Signal` through the compile call chain

We could keep `signal.Notify` and thread the resulting channel through every function that needs to be interruptible, selecting on it in each loop alongside the watcher's other event sources. This was rejected because Go's standard cancellation contract is `context.Context`, the context is already plumbed end-to-end through the compile pipeline, and adding a parallel signal channel would mean two cancellation primitives that can drift out of sync (e.g., context cancelled for non-signal reasons would still leave loops running). It also duplicates the SIGINT-to-cancellation conversion that `signal.NotifyContext` in `main.go` already performs.

#### Alternative 2: Leave the watch loop's `signal.Notify` in place and only add `ctx.Done()` guards to the batch loops

We could fix the batch-mode problem narrowly by adding `ctx.Done()` checks to the per-file loops while leaving the watch loop's local signal handler untouched. This was rejected because the watch loop's `signal.Notify(sigChan, SIGINT, SIGTERM)` overlaps with `signal.NotifyContext` in `main.go` — both deliver the same signals — and the watcher would respond to OS signals but ignore context cancellations issued for any other reason (tests, parent shutdown, future programmatic teardown). Centralising on context produces one cancellation pathway that works in every caller, including tests that cancel synthetically without sending a real signal.

#### Alternative 3: Check cancellation only at coarser boundaries (e.g., before the loop, not per iteration)

We could check `ctx.Done()` once at the top of `compileSpecificFiles` / `compileAllFilesInDirectory` and rely on per-file compilation being fast enough that a stale check is acceptable. This was rejected because the original bug report describes Ctrl+C being ignored *during* a multi-file compile — exactly the case where a top-of-function check fires too early to help. The per-iteration guard adds two lines of boilerplate per loop and makes the cancellation point obvious at the site where work is dispatched.

### Consequences

#### Positive
- Ctrl+C now visibly aborts an in-flight `gh aw compile` between files, returning `context.Canceled` and printing a "Compilation cancelled" warning, restoring the user expectation that the signal is honoured.
- Cancellation is uniform across batch and watch modes: a single `context.Context` cancellation tears down both, so callers (CLI, tests, future programmatic embedders) have one mechanism to learn.
- Watch mode no longer races between two signal sources; the `os/signal` and `syscall` imports drop out of `compile_watch.go`, reducing surface area.
- Tests can drive cancellation synchronously via `context.WithCancel` without sending real OS signals, which is what `compile_context_cancellation_test.go` does for four code paths (pre-cancel at entry, mid-loop for specific files, mid-loop for directory walks, watch-mode shutdown).
- The pipeline now follows Go's idiomatic cancellation contract end-to-end; future contributors can lean on `ctx.Done()` as the only stop signal they need to remember.

#### Negative
- Every long-running loop in the compile pipeline now carries a 6-line `select { case <-ctx.Done(): ...; default: }` preamble; if more such loops appear, the boilerplate will accumulate or need to be hoisted into a helper.
- Cancellation is still cooperative — a single workflow file whose compile takes a long time will not be interrupted mid-file; users must wait for the current file to finish before the next `ctx.Done()` check fires.
- Removing the local `signal.Notify` makes `watchAndCompileWorkflows` strictly dependent on a caller that installs `signal.NotifyContext` (or equivalent); a future caller that constructs the watcher with a raw `context.Background()` will silently lose Ctrl+C handling. This contract is currently undocumented at the function signature.
- The function returns `context.Canceled` rather than `nil` on user cancel for batch mode, which callers must treat as a non-error termination if they want to suppress an error exit code.

#### Neutral
- The `os/signal` and `syscall` imports are removed from `compile_watch.go`; any grep-based tooling that searched for signal handling in the compile package will now find it only in `main.go`.
- `compile_context_cancellation_test.go` is a new test file with the `//go:build !integration` tag and four test functions covering each cancellation entry point; it depends on `pkg/testutil` and `pkg/workflow` for fixtures.
- The "Compilation cancelled" warning is emitted to `stderr` via `console.FormatWarningMessage`, matching the existing convention for user-facing warnings in this package.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Signal-to-Context Conversion

1. The process entry point (`main.go`) **MUST** be the sole site that converts OS signals (SIGINT, SIGTERM) into context cancellation for the compile command, using `signal.NotifyContext` or an equivalent mechanism.
2. Functions in the `pkg/cli` compile pipeline **MUST NOT** call `signal.Notify`, `signal.NotifyContext`, or otherwise install their own OS signal handlers.
3. Functions in the `pkg/cli` compile pipeline **MUST NOT** import `os/signal` or `syscall` solely for the purpose of receiving SIGINT / SIGTERM.

### Cancellation Guards in Compile Loops

1. `compileSpecificFiles` **MUST** evaluate `ctx.Done()` via a non-blocking `select` at the top of each iteration of its per-file loop and **MUST** return `(workflowDataList, ctx.Err())` immediately when the context is done.
2. `compileAllFilesInDirectory` **MUST** evaluate `ctx.Done()` via a non-blocking `select` at the top of each iteration of its per-file loop and **MUST** return `(workflowDataList, ctx.Err())` immediately when the context is done.
3. When cancellation is observed, the implementation **SHOULD** emit a single user-visible warning to `stderr` using the established warning formatter (e.g., `console.FormatWarningMessage("Compilation cancelled")`).
4. The cancellation guard **MUST NOT** block; it **MUST** use a `select` with a `default:` arm so that an uncancelled context proceeds without delay.

### Watch Mode Cancellation

1. `watchAndCompileWorkflows` **MUST** treat `<-ctx.Done()` as the canonical shutdown trigger in its event-loop `select`.
2. `watchAndCompileWorkflows` **MUST NOT** create or read from a local `chan os.Signal` for shutdown purposes.
3. `watchAndCompileWorkflows` **MUST** return `nil` (not `ctx.Err()`) on a clean context-cancellation shutdown, so that user-initiated stops do not produce a non-zero CLI exit.

### Test Coverage

1. Tests **SHOULD** drive cancellation via `context.WithCancel` rather than sending real OS signals, to keep the suite hermetic.
2. The suite **SHOULD** cover, at minimum: a context already cancelled at function entry, a context cancelled between iterations of the specific-files loop, a context cancelled between iterations of the directory loop, and a context cancelled during a running watch loop.
3. Tests of the batch-compile loops **MUST** assert that the returned error satisfies `errors.Is(err, context.Canceled)`.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/26241545464) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
