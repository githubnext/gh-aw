# ADR-51517: Extract Method Refactoring for Add Interactive Workflow Run Prompt

**Date**: 2026-08-09
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The codebase enforces a function-length lint rule ("large-function limit") across `pkg/`. The function `checkStatusAndOfferRun` in `pkg/cli/add_interactive_workflow.go` handles multiple distinct concerns — status polling, user-facing messaging, environment detection (Codespaces), user confirmation, local branch updating, and workflow execution — all within a single function body. This multi-concern design caused the function to exceed the length limit and fail the linter. The PR is part of an ongoing "function-length backlog" that systematically reduces oversized CLI functions using surgical helper extraction without behavior changes.

### Decision

We will apply the Extract Method refactoring pattern to `checkStatusAndOfferRun`, splitting it into nine focused single-responsibility helper methods: `waitForWorkflowStatus`, `checkWorkflowStatusAttempt`, `shouldOfferAddedWorkflowRun`, `showWorkflowStatusUnavailableInstructions`, `showCodespaceRunInstructions`, `confirmRunAddedWorkflow`, `runAddedWorkflowOnce`, `updateLocalBranchBeforeWorkflowRun`, and `showWorkflowRunURL`. Each helper encapsulates one sub-concern, bringing every function below the lint threshold. No behavior is changed.

### Alternatives Considered

#### Alternative 1: Suppress the lint rule for this function

Add a per-function or per-file lint directive to exempt `checkStatusAndOfferRun` from the function-length check, avoiding any restructuring. This was not chosen because it would perpetuate the underlying maintainability problem (a function handling too many concerns), accumulate technical debt, and contradict the project's stated goal of working down the function-length backlog.

#### Alternative 2: Restructure as a state machine

Replace the imperative control flow with an explicit state machine (states: Polling → Ready → Dispatch → Confirming → Executing) or a strategy/command pattern, providing a stronger formal separation of concerns. This was not chosen for this PR because it requires a larger structural rewrite that would exceed the stated "surgical helper extraction without behavior changes" scope, and would be a distinct architectural decision warranting its own ADR and review.

### Consequences

#### Positive
- All extracted functions pass the function-length linter check.
- Individual helpers are independently unit-testable; the new `TestShouldOfferAddedWorkflowRun` table test demonstrates this.
- Each method has a clear single responsibility, improving readability and reducing cognitive load when navigating the file.

#### Negative
- Increased function-call indirection: the top-level orchestrator now delegates through multiple helpers, requiring readers to follow more call chains.
- Private method receivers (on `AddInteractiveConfig`) limit reuse of helpers outside the struct boundary; `confirmRunAddedWorkflow` is a package-level function with a slightly inconsistent scope compared to the other helpers.

#### Neutral
- This is a pure structural refactoring; all existing integration tests continue to exercise the combined flow unchanged.
- The overall file size grows slightly due to function signatures and new method declarations, even though total logic lines decrease.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
