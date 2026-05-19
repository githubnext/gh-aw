# ADR-33273: Synthetic `pull_request_reviewer` Trigger with Centralized Lifecycle Routing

**Date**: 2026-05-19
**Status**: Draft
**Deciders**: Unknown

---

## Part 1 — Narrative (Human-Friendly)

### Context

Agentic workflows that act as PR reviewers need to react to a coordinated set of GitHub events — assignment as a reviewer (`pull_request.ready_for_review`) and review activity (`pull_request_review.submitted`/`edited`/`dismissed`) — but the existing slash-command routing model only handles comment-style invocations. Authors of reviewer workflows previously had to wire up the full event matrix, concurrency rules, and PR-state guards by hand, which led to inconsistent behavior across workflows and missed lifecycle edges (notably re-runs after a PR was already closed). The repository already adopted centralized slash-command routing in [ADR-31605](31605-centralized-slash-command-routing.md), so an obvious next step is to extend that routing surface to cover the reviewer lifecycle as a first-class trigger. The constraints are: keep the existing slash-command routing semantics intact, avoid an explosion of bespoke `on:` blocks in user workflows, and preserve compile-time validation of the new trigger.

### Decision

We will introduce a synthetic frontmatter trigger `on.pull_request_reviewer: slash_command` that compiles down to centralized routing through `agentic_commands.yml`. The compiler will: (1) extract the reviewer trigger from frontmatter; (2) emit reviewer lifecycle event coverage for `pull_request.ready_for_review` and `pull_request_review` actions; (3) apply PR-scoped concurrency with queue-max defaults; and (4) generate a `GH_AW_REVIEWER_ROUTING` payload plus reviewer routes in the central router. The runtime dispatcher (`route_slash_command.cjs`) will resolve review events back to the owning workflow via existing XML markers (`extractWorkflowId`) in the review body, and will cancel early when the PR is already closed at workflow start.

### Alternatives Considered

#### Alternative 1: Require authors to wire `pull_request` + `pull_request_review` events manually

Leave the existing primitives as-is and document the recommended event matrix. Rejected because this duplicates concurrency, PR-state guarding, and marker resolution logic across every reviewer workflow, and offers no compile-time signal that a workflow is reviewer-shaped. It also makes future reviewer-lifecycle changes a breaking change for every consumer.

#### Alternative 2: Add a separate top-level frontmatter key (e.g., `reviewer: true`) instead of a synthetic `on.*` entry

A boolean flag would be simpler to parse, but it diverges from the existing model where trigger shape is declared under `on:`. The `on.pull_request_reviewer: slash_command` form composes naturally with the centralized router (which already keys off the `slash_command` mode) and keeps trigger declarations in one place. Rejected to preserve schema consistency with [ADR-31605](31605-centralized-slash-command-routing.md).

#### Alternative 3: Dispatch review events by matching workflow name conventions

Instead of XML markers in the review body, derive the target workflow from naming heuristics. Rejected because marker-based resolution is already used elsewhere in the codebase and is robust against renames, while heuristics are fragile and would leak workflow-internal naming into the user-visible review surface.

### Consequences

#### Positive
- Reviewer workflows declare a single line of frontmatter and inherit a consistent lifecycle, concurrency, and PR-state behavior.
- Reuses the centralized router from ADR-31605, so reviewer routes share the same generation, metadata, and observability surfaces as slash commands.
- Early cancellation on PR-closed eliminates a class of zombie runs that previously consumed runner minutes after a PR was closed mid-flight.
- Marker-based resolution (`extractWorkflowId`) makes review-event dispatch resilient to workflow renames.

#### Negative
- Adds a new synthetic value (`slash_command`) under an `on.*` key that does not correspond to a real GitHub event name, increasing the conceptual surface area of the trigger schema.
- Introduces a new code path in `route_slash_command.cjs` for review-event dispatch, expanding the routing logic that must be kept in sync with reviewer lifecycle events GitHub may add later.
- The `GH_AW_REVIEWER_ROUTING` env var grows the routing payload size and adds another vector for compile-time/runtime divergence if the two halves drift.
- Queue-max concurrency defaults may surprise authors who expect the default `pull_request` behavior; documentation must cover this.

#### Neutral
- Workflow generation now stays enabled when only reviewer routes exist, even without standard slash or label routes. Downstream tooling that assumed at least one slash/label route must be reviewed.
- The reviewer routing section is emitted alongside slash and label routes in generated metadata and comments, which slightly changes the shape of the generated `agentic_commands.yml`.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Trigger Schema

1. The main workflow schema **MUST** accept `on.pull_request_reviewer` with the literal value `slash_command` as its only supported mode.
2. Workflows declaring `on.pull_request_reviewer` **MUST NOT** require any additional `pull_request` or `pull_request_review` entries to receive reviewer lifecycle events.
3. The frontmatter extractor **MUST** propagate the reviewer trigger into the compiler's workflow data so downstream stages can detect reviewer-shaped workflows.

### Compiler Behavior

1. The compiler **MUST** treat the reviewer trigger as centralized command routing equivalent in dispatch semantics to existing slash-command routes.
2. The compiler **MUST** emit event coverage for both `pull_request` (action `ready_for_review`) and `pull_request_review` (actions `submitted`, `edited`, `dismissed`) for any workflow declaring the reviewer trigger.
3. The compiler **MUST** apply PR-scoped concurrency with queue-max behavior as the default for reviewer flows.
4. Generated workflow output **MUST** remain enabled when reviewer routes exist even if no standard slash-command or label routes are present.

### Central Router (`agentic_commands.yml`)

1. The router generator **MUST** include reviewer lifecycle routes in the central router whenever at least one workflow declares the reviewer trigger.
2. The router **MUST** emit a `GH_AW_REVIEWER_ROUTING` payload describing each reviewer workflow and the events it subscribes to.
3. The generated routing metadata and comments **MUST** include a reviewer section distinct from slash-command and label routes.

### Runtime Dispatch (`route_slash_command.cjs`)

1. The dispatcher **MUST** cancel execution early when the pull request is already in a closed state at workflow start.
2. The dispatcher **MUST** dispatch reviewer workflows on `pull_request` events with action `ready_for_review`.
3. The dispatcher **MUST** dispatch reviewer workflows on `pull_request_review` events with actions `submitted`, `edited`, or `dismissed`.
4. For `pull_request_review` events, the dispatcher **MUST** resolve the target workflow by extracting the workflow identifier from XML markers in the review body using the existing `extractWorkflowId` helper.
5. The dispatcher **MUST NOT** infer the target workflow for review events from workflow name heuristics.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/26120229301) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
