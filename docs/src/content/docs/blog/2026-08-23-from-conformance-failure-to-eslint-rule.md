---
title: "From a Conformance Failure to an ESLint Rule"
description: "How a daily Safe Outputs conformance check found an MCP error-message bug, produced a targeted fix, and inspired a new ESLint rule."
authors:
  - copilot
date: 2026-08-23
metadata:
  seoDescription: "A daily Safe Outputs check found an MCP error serialization bug, drove a targeted repair, and inspired a new ESLint rule."
---

A small diagnostic failure can reveal a larger reliability pattern. On August 23, the [Daily Safe Outputs Conformance Checker](https://github.com/github/gh-aw/blob/main/.github/workflows/daily-safe-outputs-conformance.md) found an error-serialization problem in `gh-aw`'s MCP server path. That finding became [issue #55014](https://github.com/github/gh-aw/issues/55014), the targeted repair in [PR #55042](https://github.com/github/gh-aw/pull/55042), and a proposed preventative rule in [PR #55052](https://github.com/github/gh-aw/pull/55052).

The sequence is a useful example of an agentic maintenance loop: a scheduled check identifies a mismatch, an issue explains the real behavior behind it, a focused change repairs both the implementation and the check, and a separate mining workflow looks for the same risky shape elsewhere.

## The signal: MCE-006

The daily checker runs [`scripts/check-safe-outputs-conformance.sh`](https://github.com/github/gh-aw/blob/main/scripts/check-safe-outputs-conformance.sh) against the Safe Outputs implementation. Its workflow runs the script, captures its output, groups failures by check ID and severity, and creates actionable issues for important findings. High-severity failures make the script exit nonzero; the workflow can keep the resulting issue open for one day while newer runs replace stale reports.

Run [#32621246743](https://github.com/github/gh-aw/actions/runs/32621246743) reported MCE-006, the check for readable serialized error messages. Initially, the checker only looked in `mcp_server_core.cjs` for direct uses of `String(e.message)` or `String(err.message)`. The core instead delegates formatting to `getErrorMessage()` in [`error_helpers.cjs`](https://github.com/github/gh-aw/blob/main/actions/setup/js/error_helpers.cjs), so the literal match could not distinguish a missing serialization path from a shared helper.

That looked like a checker limitation, but the generated issue investigated the helper rather than dismissing the result. It found a real edge case: for a thrown plain object with a non-string `message`, the helper fell through to `String(error)`. A value such as `{ message: { reason: "x" } }` could therefore become the unhelpful user-facing text `[object Object]`, contrary to the Safe Outputs requirement that JSON-RPC error messages remain readable.

## The repair: fix the value and teach the check

[PR #55042](https://github.com/github/gh-aw/pull/55042) makes the branch explicit. When a non-`Error` object has a `message` property, `getErrorMessage()` now preserves a string message or coerces that message value; only objects without a message follow the whole-object fallback. The accompanying tests cover numeric and non-primitive message values so the behavior is exercised rather than inferred.

The same change updates MCE-006's model of the code. The checker continues to accept direct message coercion in the MCP core, but it also recognizes the shared-helper path when the core uses `getErrorMessage()` and the helper itself safely handles non-string messages. This matters because conformance checks should follow the security property, not insist on one spelling of its implementation.

## The follow-up: mine the pattern into a rule

The repair also provided evidence that the problem was structural rather than isolated. The scheduled [ESLint Miner](https://github.com/github/gh-aw/blob/main/.github/workflows/eslint-miner.md) workflow mines recent issues and discussions, scans `actions/setup/js`, selects one low-false-positive rule, validates it, and opens at most one draft PR. Its [August 23 run](https://github.com/github/gh-aw/actions/runs/32629340370) used the MCE-006 issue as that evidence.

The result, proposed in [PR #55052](https://github.com/github/gh-aw/pull/55052), is `no-string-fallback-for-non-string-message`. The rule detects a narrow conditional shape: code verifies that `x.message` is a string, returns that message when it is, then falls back to `String(x)` instead of `String(x.message)`. It is configured as a warning because the correct fallback can differ by call site, but the diagnostic makes the risky choice visible.

The miner found four existing occurrences in `actions/setup/js`: `dispatch_workflow.cjs`, `route_slash_command.cjs`, `log_parser_shared.cjs`, and `safeoutputs_cli.cjs`. The rule intentionally does not change them automatically; each needs a local decision about what a readable fallback should be. Its tests accept the safe counterpart, `String(x.message)`, and reject the container-stringifying form.

## A maintenance loop, not a one-off fix

This trail links a specification, an executable conformance check, a short-lived actionable issue, a targeted repair, and a preventative lint rule. The daily conformance workflow is responsible for detecting violations against a defined contract. ESLint Miner is responsible for turning a demonstrated repeated code pattern into a durable guardrail. Together, they move the repository from “this bug happened” to “this class of bug is easier to see before it reaches an MCP response.”

Follow [github/gh-aw](https://github.com/github/gh-aw) for the status of the repair and rule proposals, and inspect the linked workflows to adapt the same feedback loop to your own repository.
