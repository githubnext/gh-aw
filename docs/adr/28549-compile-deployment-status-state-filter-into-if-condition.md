# ADR-28549: Compile `deployment_status.state` Filter into GitHub Actions `if:` Condition

**Date**: 2026-04-26
**Status**: Draft
**Deciders**: Unknown (generated from PR diff — [PR #28549](https://github.com/github/gh-aw/pull/28549))

---

## Part 1 — Narrative (Human-Friendly)

### Context

The gh-aw compiler translates a higher-level Markdown-based workflow DSL into GitHub Actions YAML. The GitHub `deployment_status` event fires for every state change in an external deployment (pending, queued, in_progress, success, failure, error, inactive, waiting). For DevOps incident automation — the primary use-case for this trigger — only the terminal failure states (`error`, `failure`) are actionable, but GitHub Actions provides no native trigger-level filter for `deployment_status` by state. Without compiler support, workflow authors must write raw `if:` expressions manually, which is inconsistent with the DSL's abstraction level and causes agents to default to suboptimal triggers when generating workflows.

### Decision

We will add a `state:` field to the `deployment_status` trigger in the gh-aw DSL schema and compiler. When present, the compiler reads `on.deployment_status.state` (accepting a single string or an array) and synthesises the equivalent GitHub Actions expression (`github.event.deployment_status.state == 'error' || ...`), merging it into the job-level `if:` condition. The `state:` lines are commented out in the compiled lock file with an explanatory note. We will also introduce natural-language trigger shorthands (e.g., `"deployment failed"`, `"deployment failed or error"`) in `trigger_parser.go` that expand to the same `deployment_status` trigger with the appropriate `state` condition, enabling both the declarative YAML form and a concise prose form.

### Alternatives Considered

#### Alternative 1: Document the Pattern Without Compiler Changes

Add a canonical example using a manually written `if: github.event.deployment_status.state == 'failure'` expression and document the approach in the workflow guide, leaving the compiler unchanged.

This was not chosen because it keeps the filtering burden on workflow authors, is inconsistent with other trigger abstractions in the DSL (e.g., `issue.state`), and does not enable natural-language shorthands. Agents generating workflows from prose descriptions would still lack a declarative signal to use.

#### Alternative 2: Runtime Filtering Inside the Agent Prompt

Instead of compile-time condition synthesis, instruct the agent (via its system prompt or workflow description) to exit early when `github.event.deployment_status.state` is not a failure state.

This was not chosen because it consumes agent tokens on every non-failure deployment event, increases latency, and places correctness-critical control flow inside an LLM response rather than in deterministic compiled infrastructure. It also makes no-op runs indistinguishable from real activations in the audit log.

### Consequences

#### Positive
- Workflow authors can express state-filtered deployment triggers declaratively (`state: [error, failure]`), consistent with other DSL trigger filters.
- Natural-language shorthands (`on: "deployment failed or error"`) lower the barrier for DevOps automation, enabling agents to generate correct workflows from prose intent.
- Compile-time `if:` conditions prevent unnecessary agent invocations for non-failure events, reducing cost and noise.
- A canonical, compilable example (`deployment-incident-monitor.md`) gives teams a tested starting point.

#### Negative
- The hardcoded `state` enum (`error`, `failure`, `pending`, `success`, `inactive`, `in_progress`, `queued`, `waiting`) must be kept in sync with GitHub's deployment status API; additions or renames require a compiler update.
- Each new trigger type with semantic sub-fields (like `state:`) increases the surface area of the compiler's extraction logic, adding maintenance burden.
- The natural-language parser introduces implicit mappings (`"deployment failed"` → `state == 'failure'`) that are opaque unless documented; future contributors may not know the shorthand exists.

#### Neutral
- The `state:` lines are intentionally commented out in the compiled lock file, which may surprise contributors inspecting the generated YAML.
- `TriggerIR.Conditions` propagation through `schedule_preprocessing.go` is a prerequisite change that affects all future NL trigger shorthands, not just `deployment_status`.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Schema and Validation

1. The `deployment_status` trigger object **MUST** accept an optional `state` property that is either a single string or an array of strings.
2. Each value in `state` **MUST** be one of the enumerated GitHub deployment status values: `error`, `failure`, `pending`, `success`, `inactive`, `in_progress`, `queued`, `waiting`.
3. An unrecognised `state` value **SHOULD** produce a compiler warning and **MUST NOT** be silently ignored.

### Compilation

1. When `on.deployment_status.state` is present, the compiler **MUST** synthesise a GitHub Actions expression of the form `github.event.deployment_status.state == '<value>'`, joining multiple values with ` || `.
2. The synthesised expression **MUST** be merged into the job-level `if:` condition of the activation job.
3. The `state:` lines in the compiled lock file **MUST** be commented out with an explanatory note indicating that state filtering was compiled into the `if:` condition.
4. The compiled lock file **MUST NOT** include a native `deployment_status.state` filter under `on:`, as GitHub Actions does not support trigger-level state filtering for this event.

### Natural-Language Trigger Parsing

1. The natural-language trigger parser **MUST** recognise the phrase `"deployment failed"` and expand it to a `deployment_status` trigger with `state == 'failure'`.
2. The natural-language trigger parser **MUST** recognise the phrase `"deployment error"` and expand it to a `deployment_status` trigger with `state == 'error'`.
3. The natural-language trigger parser **MUST** recognise the phrase `"deployment failed or error"` (and semantically equivalent phrasings) and expand it to a `deployment_status` trigger with `state == 'failure' || state == 'error'`.
4. Natural-language expansions **MUST** produce conditions that are propagated through `TriggerIR.Conditions` into the frontmatter `if:` field.
5. New natural-language deployment shorthands **SHOULD** be added to this parser rather than handled inline in calling code.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24955643779) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
