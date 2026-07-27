---
description: Design evidence-driven workflows that mine, refine, and apply custom linter rules without overfitting or creating unbounded remediation work.
---

# Linter Workflow Guidance

Use this guidance for any language or static-analysis framework. Treat mining, refinement, and application as a feedback loop; they may be separate workflows.

## Mine Rules From Repository Evidence

Search a bounded recent window and retain source links. Prefer evidence in this order:

1. repeated fixes in merged commits and pull requests
2. recurring review comments and resolved discussion threads
3. repeated failures or corrections in Copilot coding agent sessions
4. issues, discussions, and CI diagnostics
5. confirmed occurrences from structural or semantic code scans

Corroborate a candidate with independent events or multiple concrete occurrences. A single preference, speculative smell, or isolated mistake is insufficient.

Normalize each candidate to:

- the unsafe or undesirable pattern
- the consequence and mechanical correction
- evidence references and occurrence count
- affected languages, paths, and constructs
- likely false-positive cases

Use deterministic pre-steps to fetch, trim, and normalize large histories into `/tmp/gh-aw/agent/`. Give the agent compact evidence, not raw repositories of logs. Persist only candidate identifiers and outcomes in `cache-memory` or `repo-memory`.

## Select One High-Signal Rule

Before selection, compare candidates with:

- existing custom rules and prior proposals
- enabled standard or third-party linters
- recently rejected or reverted rules

Prefer a narrow rule with a clear diagnostic, a mechanical fix, stable syntax or semantics, and low false-positive risk. Reject candidates that are stylistic only, duplicate existing coverage, depend on intent unavailable to static analysis, or combine unrelated patterns.

Select at most one new rule per run. If no candidate clears the quality bar, emit `noop`.

## Implement and Validate

Follow the repository's existing rule layout, registration, naming, suppression, and test conventions. Keep implementation changes scoped to the rule, its registration, configuration, and tests.

Tests must cover:

- representative positive cases from the mined evidence
- nearby valid code that must not trigger
- relevant edge, aliasing, scope, or type-resolution cases
- suppression and autofix behavior when supported

Run the rule's targeted tests, build the linter, then run the configured ruleset against its target codebase. A new rule must not leave unexplained diagnostics. Open one focused draft pull request with the evidence, intended invariant, known limitations, and validation results; otherwise emit `noop`.

## Refine Existing Rules

Mine rule diagnostics, suppressions, CI failures, review feedback, reverted fixes, and follow-up commits. Classify each problem as:

- false positive
- false negative
- unclear diagnostic
- unsafe or incomplete autofix
- missing suppression or configuration behavior
- performance regression

Make the smallest change that addresses one evidenced class. Add a regression test before changing detection behavior. Do not broaden a rule merely to increase match counts; precision takes priority over coverage.

## Apply Rules

Run the linter deterministically before the agent starts and save compact diagnostics. If clean, emit `noop`.

When findings exist:

1. group them by root cause, then by subsystem when useful
2. bound each run to a small number of independent groups
3. search open and recently closed work before creating anything
4. create or update one authoritative issue per group
5. include affected paths, representative diagnostics, expected outcome, and the exact validation command
6. assign agents only for new, execution-ready work

Do not create separate work items for count changes or path slices of the same backlog. Feed fix outcomes, review comments, and validation failures back into refinement.

## Workflow Guardrails

- Keep repository access read-only in the agent job; use safe outputs for writes.
- Restrict write outputs to the rule's intended paths and fall back to an issue for protected files.
- Bound history windows, candidates, remediation groups, and agent assignments.
- Use persistent memory for compact deduplication state and cursors, not large transcripts.
- End with exactly one terminal safe output such as a pull request, issue/report, or `noop`.
- Do not finish while delegated agents are still running.
