---
title: "One Small Error Message, One Big Feedback Loop"
description: "How a daily Safe Outputs check found an MCP error-message bug, drove a repair, and inspired a new ESLint rule."
authors:
  - copilot
  - pelikhan
date: 2026-08-23
metadata:
  seoDescription: "How a daily Safe Outputs check found an MCP error-message bug, drove a repair, and inspired a new ESLint rule."
---

Most bug reports start with a person noticing something odd. This one started with a small scheduled script.

On August 23, the [Daily Safe Outputs Conformance Checker](https://github.com/github/gh-aw/blob/main/.github/workflows/daily-safe-outputs-conformance.md) spotted a possible gap in `gh-aw`'s MCP error handling. The immediate symptom was not dramatic: under the wrong conditions, a user could receive `[object Object]` instead of a useful error message. But the story that followed is a good reminder of what automated maintenance can look like at its best.

One check became [issue #55014](https://github.com/github/gh-aw/issues/55014), a focused repair in [PR #55042](https://github.com/github/gh-aw/pull/55042), and a preventative lint rule proposed in [PR #55052](https://github.com/github/gh-aw/pull/55052). The interesting part is not any one of those artifacts. It is the loop between them.

## A tiny signal worth following

Every day, the checker runs [`scripts/check-safe-outputs-conformance.sh`](https://github.com/github/gh-aw/blob/main/scripts/check-safe-outputs-conformance.sh) against the Safe Outputs implementation. It collects the results, groups failures by severity and check ID, and turns important findings into actionable issues. High-severity failures cause a nonzero exit; the issue is then short-lived, so a newer run can replace stale information instead of growing an endless backlog.

Run [#32621246743](https://github.com/github/gh-aw/actions/runs/32621246743) raised MCE-006, the check for readable serialized error messages. At first glance, it looked like the checker had simply missed an abstraction: it searched `mcp_server_core.cjs` for direct calls such as `String(e.message)`, while the core delegates formatting to `getErrorMessage()` in [`error_helpers.cjs`](https://github.com/github/gh-aw/blob/main/actions/setup/js/error_helpers.cjs).

That could have been the end of the investigation: another false positive to tune away. Instead, the generated issue followed the helper. It uncovered a real edge case. If code threw a plain object with a non-string `message`, the helper could fall back to `String(error)`. For a value like `{ message: { reason: "x" } }`, that means the person on the other end could see `[object Object]`—technically a string, but not an explanation.

## Fix the bug—and improve the question

[PR #55042](https://github.com/github/gh-aw/pull/55042) makes the intent explicit: when a non-`Error` object has a `message` property, preserve a string message or coerce that message value. Only objects without a message use the whole-object fallback. The accompanying tests cover numeric and non-primitive messages, turning the edge case into an expected behavior.

The repair also improves MCE-006 itself. The checker still accepts direct coercion in the MCP core, but it now recognizes the shared-helper path when `getErrorMessage()` safely handles non-string messages. That is an important distinction: good conformance checks protect a property, not a particular spelling of the implementation.

## Then ask: where else does this pattern live?

The repair was not treated as a one-off. The scheduled [ESLint Miner](https://github.com/github/gh-aw/blob/main/.github/workflows/eslint-miner.md) mines recent issues and discussions, scans `actions/setup/js`, selects one low-false-positive rule, validates it, and opens at most one draft PR. Its [August 23 run](https://github.com/github/gh-aw/actions/runs/32629340370) used MCE-006 as the seed for a broader question: is this pattern hiding elsewhere?

The result is the proposed `no-string-fallback-for-non-string-message` rule in [PR #55052](https://github.com/github/gh-aw/pull/55052). It looks for a narrow shape: code confirms that `x.message` is a string, returns it when it is, then falls back to `String(x)` instead of `String(x.message)`. The rule is a warning, not an automatic rewrite, because a readable fallback still needs local judgment.

The miner found four live occurrences in `actions/setup/js`: `dispatch_workflow.cjs`, `route_slash_command.cjs`, `log_parser_shared.cjs`, and `safeoutputs_cli.cjs`. Each deserves its own fix decision. The rule simply ensures that this particular sharp edge is no longer invisible.

## The real product is the feedback loop

The lasting outcome here is not only a better error message. It is a maintenance system that keeps learning: a specification defines the promise, a daily check tests it, an issue investigates the signal, a small repair closes the gap, and a lint rule helps prevent the pattern from returning.

That is the kind of automation worth building. It does not replace engineering judgment; it creates more opportunities to apply it where it matters most. Follow [github/gh-aw](https://github.com/github/gh-aw) for the status of the repair and rule proposals, and inspect the linked workflows to adapt this feedback loop in your own repository.
