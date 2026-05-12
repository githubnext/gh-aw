# ADR-31605: Centralized Slash-Command Routing via Generated Agentic Router Workflow

**Date**: 2026-05-12
**Status**: Draft
**Deciders**: Unknown — *[TODO: PR author to confirm]*

---

## Part 1 — Narrative (Human-Friendly)

### Context

Each slash-command workflow in this repository previously registered its own listeners for `issues`, `issue_comment`, `pull_request`, `pull_request_review_comment`, `discussion`, and `discussion_comment`. With many such workflows (e.g. `/archie`, `/cloclo`, and a growing fleet), this caused every comment, issue, or PR event to wake up many lock-files, each evaluating long `if:` expressions to decide whether its slash command matched. The duplication inflated GitHub Actions usage, made permissions sprawl across workflows, and produced large compiled `if:` predicates that were hard to read and maintain. The compiler also lacked guidance to nudge authors toward a shared router as the slash-command fleet grew.

### Decision

We will introduce a `centralized` strategy for `on.slash_command` and have the compiler generate a single shared router workflow at `.github/workflows/agentic_commands.yml` that owns the merged set of slash-command events and dispatches matching target workflows via `workflow_dispatch` with an `aw_context` input. Participating workflows (those declaring `strategy: centralized`) compile to `workflow_dispatch`-only triggers, retaining their non-slash events (e.g. label-only triggers) but delegating slash detection to the central router. The compiler additionally emits a warning recommending `strategy: centralized` once three or more slash commands are detected and some remain non-centralized, so the convention scales as the fleet grows.

### Alternatives Considered

#### Alternative 1: Keep per-workflow inline listeners (status quo)

Each slash-command workflow continues to declare its own event listeners and inline `if:` predicate. This is simple and decentralized — each workflow is self-contained — but it scales poorly: every comment/issue/PR event fans out to N workflows, each compiling N progressively longer `if:` expressions. It was rejected because the cost is visible today (duplicated runs, large generated predicates) and grows linearly with the slash-command fleet.

#### Alternative 2: One static, hand-maintained router workflow

Maintain `agentic_commands.yml` by hand and require workflow authors to register their command in it manually. This avoids compiler complexity but reintroduces a long-running coordination problem: every new slash command requires editing a shared file, and the registry can drift from the per-workflow frontmatter. Rejected because compiler-generation of the router from frontmatter (`strategy: centralized`) preserves a single source of truth and avoids merge conflicts on the shared router.

#### Alternative 3: Use a `repository_dispatch` or external broker

Forward slash events to an external service (or `repository_dispatch`) that then triggers the right workflow. This decouples GitHub event listeners entirely but adds an out-of-repo dependency, new auth surface, and operational risk. Rejected because `workflow_dispatch` + a generated router stays inside GitHub Actions and requires no new infrastructure.

### Consequences

#### Positive
- One shared workflow (`agentic_commands.yml`) handles slash-event listening for all participating commands, replacing N copies of the same listener set.
- Generated `if:` predicates on participating workflows shrink from large slash-text expressions to simple `workflow_dispatch`-gated activations, improving readability of lock files.
- Workflow-level permissions on participating lock files are reduced (e.g. dropping `issues: write` / `pull-requests: write` from activation jobs that no longer need them) because routing logic lives in the central router with its own scoped `actions: write` job permission.
- A compile-time warning nudges authors toward `strategy: centralized` once three or more slash commands exist, making the convention discoverable without a manual migration push.

#### Negative
- Adds a new generated file (`.github/workflows/agentic_commands.yml`) whose lifecycle is owned by the compiler — contributors must understand it is regenerated and not edited by hand.
- Introduces an indirection: a slash command now arrives via `workflow_dispatch` triggered by another workflow, which complicates debugging (two runs to inspect instead of one) and shifts some auth context onto `aw_context`.
- The router holds `actions: write` permission to dispatch other workflows; a bug in routing logic could dispatch the wrong workflow, so the route map and event-matching filter must remain trustworthy.
- Legacy trigger-file handling for `agentics-slash-command-trigger.yml` was removed; any external references to that file name become stale.

#### Neutral
- Two existing workflows (`/archie`, `/cloclo`) were migrated as part of this PR; remaining slash workflows continue to use the default inline strategy until opted in.
- The router serializes inbound command resolution via `aw_context.command_name`, requiring `check_command_position.cjs` to learn a new `workflow_dispatch` code path (added in this PR with tests).
- Documentation under `docs/src/content/docs/reference/command-triggers.md` was updated to describe both strategies side-by-side.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Centralized Strategy Selection

1. A slash-command workflow **MAY** opt into centralized routing by setting `on.slash_command.strategy: centralized` in its frontmatter.
2. The compiler **MUST** treat `strategy: centralized` as the participation flag — a workflow without that key **MUST NOT** be wired into the central router.
3. When at least one workflow opts into `strategy: centralized`, the compiler **MUST** generate exactly one router workflow file at `.github/workflows/agentic_commands.yml`.
4. The generated router file **MUST** be regenerable from frontmatter alone and **MUST NOT** be hand-edited; contributors **SHOULD** treat it as compiler output.

### Router Workflow Structure

1. The generated router **MUST** declare `permissions: {}` at the top level (no workflow-wide permissions).
2. The router job named `route` **MUST** declare scoped job-level permissions of at minimum `actions: write` and `contents: read`, and **MUST NOT** declare broader permissions than required to dispatch participating workflows.
3. The router **MUST** listen on the **union** of slash-event types declared by participating workflows (e.g. `issues`, `issue_comment`, `pull_request`, `pull_request_review_comment`, `discussion`, `discussion_comment`) and **MUST NOT** listen on events for which no participating workflow has subscribed.
4. The router **MUST** dispatch a participating workflow only when both the command name (parsed from the first token of the payload body) and the inbound event identifier match an entry in the generated route map.
5. The router **MUST** pass an `aw_context` JSON input containing at least `command_name` to the dispatched workflow.

### Participating Workflow Compilation

1. A workflow with `strategy: centralized` **MUST** compile with `workflow_dispatch` as a trigger and **MUST** accept an `aw_context` string input.
2. A workflow with `strategy: centralized` **MUST NOT** re-declare slash-text matching on `issue_comment`, `pull_request_review_comment`, `discussion`, or `discussion_comment` in its compiled lock file; slash matching is the router's responsibility.
3. A workflow with `strategy: centralized` **MAY** retain non-slash listeners that do not collide with slash routing (for example, label-only triggers on `issues`, `pull_request`, or `discussion`).
4. The compiled activation `if:` predicate of a centralized workflow **MUST NOT** include slash-text inspection of payload bodies.

### Inbound Command Resolution

1. Setup logic processing `workflow_dispatch` events **MUST** read `command_name` from `aw_context.inputs.aw_context` JSON when present.
2. If `aw_context.command_name` is present, the setup logic **MUST** verify it is in the configured commands list and **MUST** fail the command-position check (emit a denial summary) when it is not.
3. Manual `workflow_dispatch` invocations without `aw_context.command_name` **SHOULD** pass the command-position check to preserve existing manual-run behavior.

### Compiler Guidance

1. The compiler **SHOULD** emit a warning recommending `strategy: centralized` when three or more slash commands are detected in the repository and at least one of them does not declare `strategy: centralized`.
2. The warning **MUST NOT** block compilation; it is advisory only.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25712590786) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
