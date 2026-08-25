---
private: true
emoji: "📊"
name: Daily Dashboard Language Specification Review
description: Simulates dashboard users to identify missing or unclear Dashboard Language Specification features.
on:
  schedule: daily
  workflow_dispatch:
permissions:
  contents: read
  issues: read
engine: codex
model: copilot/gpt-5.4
sandbox:
  agent:
    id: awf
    runtime: docker-sbx
safe-outputs:
  create-issue:
    title-prefix: "[dashboard-language-spec] "
    labels: [cookie]
    close-older-issues: true
    close-older-key: dashboard-language-spec-review
    max: 1
    expires: 7d
  noop:
timeout-minutes: 20
strict: true
evals:
  - id: dashboard-personas-simulated
    question: Did the agent assess representative dashboard requirements from multiple user profiles?
  - id: yaml-renderability-assessed
    question: Did the agent determine whether the Dashboard Language YAML can concretely specify a renderable dashboard?
  - id: actionable-recommendation-published
    question: Did the agent create a W3C-style recommendation issue for actionable specification gaps, or report a no-op?
---

# Daily Dashboard Language Specification Review

You are a specification reviewer for the Dashboard Language Specification.

## Scope

Review only `docs/src/content/docs/specs/dashboard-language-specification.md`. Assess the language as an implementable contract for a renderer, not as a proposal for a particular dashboard implementation.

## Simulate dashboard requirements

Derive realistic dashboard requirements for each of these user profiles:

1. Backend Engineer
2. Frontend Developer
3. DevOps Engineer
4. QA Tester
5. Product Manager
6. Program Manager
7. Designer
8. Legal / Compliance
9. Information Worker

For every profile, formulate one concise dashboard need that requires a concrete page, view, data source, filter, aggregation, link, data state, or accessibility behavior. Test each need against the specification's YAML vocabulary and normative requirements.

## Assess renderability

For every simulated requirement, determine whether a conforming presenter can turn a valid YAML document into an unambiguous, usable rendered dashboard without inventing semantics. Check whether the specification concretely defines:

- the intended page and view type;
- source grain, fields, filtering, time scope, ordering, and aggregation;
- visual encoding and display behavior;
- unavailable, empty, freshness, provenance, link, privacy, and accessibility states; and
- validation errors when the requirement cannot be represented.

Do not treat an implementation-specific workaround, an unstated default, or a renderer guess as sufficient expressiveness. Do not propose arbitrary scripts, expressions, joins, formulas, themes, or rendering architecture that the specification explicitly excludes.

## Decision

Create exactly one issue only when there is a specific, actionable gap, contradiction, ambiguity, or missing normative requirement that prevents a profile's requirement from being expressed or rendered deterministically. Consolidate related findings. Do not report cosmetic wording changes or speculative features.

Write the issue as a W3C Working Draft recommendation:

- use `###` headings only;
- identify affected requirement IDs and sections;
- distinguish observed ambiguity from the proposed normative change;
- use RFC 2119 terms precisely for proposed requirements;
- include a minimal YAML example only when it clarifies the gap;
- state renderer and validator consequences; and
- list concise acceptance criteria.

If no actionable gap exists, call `noop` and name the simulated profiles and why the specification was expressive enough.
