---
title: Orchestration
description: Coordinate multiple agentic workflows using workflow dispatch or reusable workflow calls (orchestrator/worker pattern).
---

Use this pattern when one workflow (the **orchestrator**) needs to fan out work to one or more **worker** workflows.

## The orchestrator/worker pattern

- **Orchestrator**: decides what to do next, splits work into units, dispatches workers.
- **Worker(s)**: do the concrete work (triage, code changes, analysis) with scoped permissions/tools.
- **Optional monitoring**: both orchestrator and workers can update a GitHub Project board for visibility.

## Dispatch workers with `dispatch-workflow`

Allow dispatching specific workflows via GitHub's `workflow_dispatch` API:

```yaml
safe-outputs:
  dispatch-workflow:
    workflows: [repo-triage-worker, dependency-audit-worker]
    max: 10
```

During compilation, gh-aw validates the target workflows exist and support `workflow_dispatch`. Workers receive a JSON payload and run asynchronously as independent workflow runs.

See [`dispatch-workflow` safe output](/gh-aw/reference/safe-outputs/#workflow-dispatch-dispatch-workflow).

## Call workers with `call-workflow`

Call reusable workflows (`workflow_call`) via compile-time fan-out—no API call at runtime:

```yaml
safe-outputs:
  call-workflow:
    workflows: [spring-boot-bugfix, frontend-dep-upgrade]
    max: 1
```

The compiler validates that each worker declares `workflow_call`, generates a typed MCP tool per worker from its inputs, and emits a conditional `uses:` job. At runtime the worker whose name the agent selected executes as part of the same workflow run—preserving `github.actor` and billing attribution.

See [`call-workflow` safe output](/gh-aw/reference/safe-outputs/#workflow-call-call-workflow).

## Choosing between the two approaches

Use `call-workflow` when actor attribution matters, workers must finish before the orchestrator concludes, or you want zero API overhead. Use `dispatch-workflow` when workers should run asynchronously, outlive the parent run, or need `workflow_dispatch` inputs.

## Passing correlation IDs

If your workers need shared context, pass an explicit input such as `tracker_id` (string) and include it in worker outputs (e.g., writing it into a Project custom field).

See also: [Monitoring](/gh-aw/patterns/monitoring/)
