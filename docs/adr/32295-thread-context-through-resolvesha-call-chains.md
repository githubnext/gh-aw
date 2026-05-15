## ADR-32295: Thread `context.Context` Through `ResolveSHA` Call Chains

**Date**: 2026-05-15
**Status**: Draft
**Deciders**: Unknown (auto-generated from PR diff)

---

### Part 1 — Narrative (Human-Friendly)

#### Context

`ActionResolver.ResolveSHA` performs `gh api` network calls to resolve action references to pinned commit SHAs and is invoked from compile, lock-file validation, maintenance-workflow generation, and the Copilot setup pipeline. Five production call sites passed a hardcoded `context.Background()`, which meant cobra's signal-aware `cmd.Context()` (Ctrl-C, timeouts) could not propagate into the network layer, and callers had no way to inject deadlines or cancellation. The hardcoded boundaries were spread across `pkg/workflow/` and `pkg/cli/` and reached through long internal call chains, so the fix is not local: every intermediate function must accept and forward `ctx`. The goal of this change is to eliminate all hidden `context.Background()` usage on the resolution path and confine `context.Background()` to top-level CLI entry points and tests.

#### Decision

We will thread `context.Context` end-to-end from cobra `RunE` handlers (via `cmd.Context()`) down to every `ActionResolver.ResolveSHA` invocation, requiring all intermediate functions in `pkg/workflow/` and `pkg/cli/` to take `ctx context.Context` as their first parameter. To support struct-based call paths, the `Compiler` type gains a `ctx` field with `WithContext(ctx)` option and `SetContext(ctx)` mutator, defaulting to `context.Background()` only inside `NewCompiler()` so existing struct constructions remain valid. After this change, `context.Background()` is permitted only at top-level CLI entry points (as a fallback when `cmd.Context()` is unavailable) and in tests; all other call sites **must** propagate `ctx` received from above.

#### Alternatives Considered

##### Alternative 1: Keep `context.Background()` and add timeouts inside `ResolveSHA`

Leave call signatures unchanged and apply a fixed timeout (e.g. `context.WithTimeout(context.Background(), 30s)`) inside `ResolveSHA` itself. Rejected because it solves only the hang-protection symptom and not the cancellation problem: Ctrl-C from cobra still cannot interrupt a hung `gh api` call, tests cannot inject deterministic contexts, and the timeout policy becomes invisible to the caller that actually knows the budget.

##### Alternative 2: Stash context in a package-level or `ActionResolver` field

Set the context once (e.g. on `ActionResolver`) and have `ResolveSHA` read it from `r.ctx` instead of taking a parameter. Rejected because it spreads context lifetime across goroutines and resolver reuse, contradicts the Go convention that `ctx` is the first parameter of any call that may block, and makes per-call deadline injection awkward.

##### Alternative 3: Thread `ctx` only into `Compiler` and a shallow wrapper

Add `ctx` to the `Compiler` struct (the centerpiece of the change) but leave the free functions in `pkg/workflow/` (`CheckActionSHAUpdates`, `ValidateActionSHAsInLockFile`, `ResolveSetupActionReference`, …) and their `pkg/cli/` callers unchanged, having them all read from a single ambient context. Rejected because several of these functions are reachable from CLI entry points that do not construct a `Compiler` (e.g. `update_actions.go`, `enable.go`, `add_command.go`), so a partial threading would leave the original five hardcoded boundaries intact under different names.

#### Consequences

##### Positive

- Cobra's `cmd.Context()` (which is cancelled on SIGINT/SIGTERM) now flows into every `gh api` network call, so `gh aw compile`, `gh aw update-actions`, etc. can be interrupted cleanly while a resolution is in flight.
- Per-call-site deadlines become possible without touching `ResolveSHA`: callers can wrap with `context.WithTimeout` at the boundary that knows the budget.
- Tests can inject controlled or already-cancelled contexts to exercise cancellation paths deterministically.

##### Negative

- The change has very broad signature churn — roughly two dozen functions across `pkg/workflow/` and `pkg/cli/` (compile pipeline, `CheckActionSHAUpdates`, `ValidateActionSHAsInLockFile`, `ResolveSetupActionReference`, `GenerateMaintenanceWorkflow`, the entire `copilot_setup.go` chain, `AddResolvedWorkflows`, `EnableWorkflowsByNames`, `InitRepository`, …) take a new first parameter, and any downstream branch or fork must rebase against it.
- The `Compiler` struct now has two ways to supply context (constructor option `WithContext` and post-hoc `SetContext`), with a `context.Background()` fallback inside `NewCompiler()`; future readers must understand that the fallback exists only for compatibility and is not the intended path.
- `InitOptions.Ctx` is a nullable field (`InitRepository` falls back to `context.Background()` if nil), which is inconsistent with the "ctx is the first parameter" rule applied everywhere else and may need follow-up to align.

##### Neutral

- `context.Background()` remains in the codebase, but only at top-level CLI entry points (as a fallback when `cmd.Context()` is unavailable) and inside tests; a future lint rule could enforce this boundary.
- The change is mechanical at most call sites and produces no behavior change when contexts are never cancelled, so it is low-risk to revert if needed.

---

### Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

#### Context Propagation on the Action-Resolution Path

1. Every call site of `ActionResolver.ResolveSHA` **MUST** pass a `context.Context` obtained from its caller; it **MUST NOT** pass a freshly constructed `context.Background()` or `context.TODO()`.
2. Every internal function in `pkg/workflow/` and `pkg/cli/` that transitively calls `ActionResolver.ResolveSHA` **MUST** accept a `ctx context.Context` as its first parameter and forward it to its callees.
3. `context.Background()` **MUST NOT** appear on the resolution call chain except at top-level CLI entry points (cobra `RunE` handlers, `main`) as a fallback when `cmd.Context()` is unavailable, and in test files.

#### Compiler and Init Boundaries

1. The `Compiler` type **MUST** expose a way to associate a `context.Context` with the compiler instance (e.g. `WithContext(ctx)` constructor option and/or `SetContext(ctx)` method).
2. `NewCompiler()` **MAY** default the embedded context to `context.Background()` when no context is supplied, but compile entry points reachable from a cobra command **SHOULD** call `SetContext(cmd.Context())` (or equivalent) before invoking compile.
3. CLI entry-point handlers (cobra `RunE`) **MUST** pass `cmd.Context()` into the first downstream function call rather than `context.Background()`.
4. `InitOptions.Ctx` **MAY** be nil; `InitRepository` **MUST** fall back to `context.Background()` only when `Ctx` is nil.

#### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25916052628) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
