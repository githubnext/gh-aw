---
name: pr-finisher
description: Prepare an open pull request for merge by fixing local validation and failing checks.
---

# PR Finisher

Use this skill when asked to get a pull request to a merge-ready state.

## Goal

Take the current pull request from "almost done" to "ready to merge" by running the standard local checks, fixing failures, checking GitHub status, and pushing the resulting changes.

## Required Workflow

1. Confirm the pull request context for the current branch.
2. Run `make fmt`.
3. Run `make lint` and fix any lint failures.
4. Run `make test-unit` and fix any failing unit tests.
5. Check the current GitHub checks for the pull request and identify every failing check.
6. If `make test` is failing locally or the failing checks show the equivalent failure in CI, fix those failures too.
7. If wasm golden tests fail, or a test fix changes the expected wasm compiler output, run `make update-wasm-golden`.
8. Re-run the affected local validation until it passes.
9. Push the final changes.

## Check Inspection

- Use GitHub tools to inspect the current pull request status and check runs.
- When the user mentions CI, build, test, or workflow failures, fetch the failing job logs before deciding on a fix.
- Distinguish between failures already covered by local commands (`fmt`, `lint`, `test-unit`, `test`) and unrelated external failures.

## Command Order

Run commands in this order unless a failure requires an earlier retry:

```bash
make fmt
make lint
make test-unit
make test
```

Only run `make test` after `make test-unit` and lint are clean, or when failing PR checks indicate the broader test suite is still broken.

## Fix Scope

- Make the smallest changes needed to get the pull request green.
- Fix lint issues before test issues.
- Do not change unrelated code just because it is nearby.
- If a failure is pre-existing and unrelated to the pull request, report that explicitly in your final user or PR update instead of guessing.

## Wasm Golden Files

When a `make test` fix changes wasm compiler output or wasm golden tests fail, regenerate the wasm goldens with:

```bash
make update-wasm-golden
```

Then re-run the relevant tests.

## Completion Standard

The task is complete only when all of the following are true:

- `make fmt` has been run.
- `make lint` passes, or any unrelated pre-existing failure is explicitly identified.
- `make test-unit` passes, or any unrelated pre-existing failure is explicitly identified.
- Failing pull request checks were inspected and summarized.
- `make test` was fixed when it was part of the failing state.
- Wasm goldens were regenerated when required.
- The final changes were pushed.
