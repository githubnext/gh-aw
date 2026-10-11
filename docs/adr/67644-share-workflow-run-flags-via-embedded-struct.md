# ADR-67644: Share CLI Workflow Run Flags via an Embedded `WorkflowRunFlags` Struct

**Date**: 2026-10-11
**Status**: Draft
**Deciders**: pelikhan (PR #67644)

---

### Context

The `gh aw run` command feeds the same eight run-time flags (`Verbose`, `EngineOverride`, `RepoOverride`, `RefOverride`, `AutoMergePRs`, `Push`, `DryRun`, `Approve`) into two different option structs in `pkg/cli`: `RunOptions` (non-interactive dispatch) and `RunWorkflowOptions` (interactive selection). Before this change, each struct declared its own copy of every field, and `cmd/gh-aw/main.go`, `run_interactive.go`, and `add_interactive_workflow.go` each had to re-list all fields at every call site. That duplication made it easy to forget to forward a newly added flag through one of the interactive paths, a class of bug that is silent at compile time. The public `pkg/cli` package API (documented in `pkg/cli/README.md`) is consumed by external callers, so any change to these structs is an API-visible change. See issue #67198.

### Decision

We will extract the eight shared run flags into a single exported `WorkflowRunFlags` struct in `pkg/cli/run_workflow_execution.go` and **embed** it (anonymously, without a JSON tag) in both `RunOptions` and `RunWorkflowOptions`. The CLI constructs `WorkflowRunFlags` once in `cmd/gh-aw/main.go` and forwards the whole value to both entry points. Flags that are genuinely mode-specific — `Enable`, `WaitForCompletion`, `RepeatCount`, `Inputs`, `JSON` on the non-interactive side and `WorkflowName`, `requiredInputsOnly` on the interactive side — stay on their respective structs. Anonymous embedding was chosen over a named field because it preserves promoted field access (`opts.Verbose`) and flat JSON serialization, keeping the change source- and wire-compatible for field *reads* while giving a single place to add future flags.

### Alternatives Considered

#### Alternative 1: Keep duplicated fields and add a copy helper

Leave both structs as-is and add a `copyRunFlags(dst, src)` helper used at each call site. This is the smallest diff and breaks no composite-literal callers at all. It was rejected because it does not remove the root cause: a newly added flag still has to be added to three structs and the helper, so the forwarding bug remains possible and is still invisible to the compiler.

#### Alternative 2: A named field (`Flags WorkflowRunFlags`) instead of embedding

A named field makes the grouping explicit and avoids any ambiguity if a future outer field collides with a promoted name. It was rejected because it changes every read site (`opts.Verbose` → `opts.Flags.Verbose`) and nests the JSON representation under a `Flags` key, which would be a harder break for external `pkg/cli` consumers and for any serialized options. This was a close call; embedding was preferred for compatibility of field access and flat JSON.

### Consequences

#### Positive
- A new shared run flag is declared once and is automatically available to both the interactive and non-interactive paths.
- Call sites shrink substantially (`cmd/gh-aw/main.go`, `run_interactive.go`) and no longer enumerate every flag, removing the silent "forgot to forward" failure mode.
- Promoted-field access and flat JSON serialization are preserved, so read-side code and serialized payloads are unchanged.

#### Negative
- Composite literals that set these fields directly (`RunOptions{RefOverride: "main"}`) no longer compile and must be rewritten as `RunOptions{WorkflowRunFlags: WorkflowRunFlags{...}}`. This is a breaking change for external `pkg/cli` consumers, and already required updating `pkg/cli/run_command_test.go`, `run_workflow_execution_test.go`, and `add_interactive_workflow.go`.
- Embedding introduces the possibility of future silent field shadowing if an outer struct ever declares a field with a promoted name.

#### Neutral
- `pkg/cli/README.md` was updated to document `WorkflowRunFlags` and the new usage example; the generated API appendix reflects the new `RunWorkflowOptions` shape.
- New regression coverage was added in `pkg/cli/workflow_run_flags_test.go` for promoted fields, zero defaults, flat JSON round-trips, and `buildWorkflowRunArgs` dispatch arguments.
- Runtime behavior, flag defaults, and command semantics are unchanged; this is a pure structural refactor.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
