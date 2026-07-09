# Investigation: Copilot failure with model alias `small`

## Scope

- Workflow run: https://github.com/github/gh-aw/actions/runs/28986878657
- Failing job: https://github.com/github/gh-aw/actions/runs/28986878657/job/86018033327
- Failing step: `Execute GitHub Copilot CLI` (step 24)
- Workflow file: `/home/runner/work/gh-aw/gh-aw/.github/workflows/smoke-copilot-small.md`

## Observed failure

The run fails before the Copilot harness starts.

```
[ERROR] Error: model 'small' is unsupported or unrecognized by this AWF version. Did you mean 'gpt-4'?
```

The error is emitted by `awf` and the step exits with code `1`.

## Evidence gathered

1. The workflow explicitly sets:

```yaml
engine:
  id: copilot
  model: small
```

2. The runtime environment confirms:

```text
COPILOT_MODEL: small
```

3. The generated AWF config in the failing step includes builtin model aliases, including:

```json
"small": ["mini"]
```

## Current hypothesis

`small` is accepted by gh-aw as a valid alias and is emitted into AWF config, but AWF v0.27.27 still rejects the incoming model string (`small`) before alias resolution is applied for this execution path.

## Why this matters

The smoke workflow intended to validate Copilot behavior with a compact alias currently validates an integration mismatch instead, causing a deterministic failure.

## Suggested follow-up validation

1. Re-run the smoke workflow with `engine.model: mini`.
2. Re-run with `engine.model: gpt-5-mini`.
3. Compare whether failure only affects alias names vs. concrete provider model identifiers.
4. If alias handling is not supported in this AWF path, either:
   - update AWF to a version that resolves aliases in this path, or
   - keep smoke workflow on a concrete model and add a dedicated alias-compatibility test once AWF support is confirmed.
