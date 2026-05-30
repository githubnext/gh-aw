---
title: Custom Steps and Jobs
description: "Add deterministic pre-processing steps and custom GitHub Actions jobs to agentic workflows using steps:, pre-agent-steps:, post-steps:, and jobs:"
---

Custom steps and jobs let you mix deterministic computation with agentic execution. All custom steps and jobs run **outside the firewall sandbox** with standard GitHub Actions security.

See [DeterministicOps](/gh-aw/patterns/deterministic-ops/) for patterns combining computation with AI reasoning.

## Custom Steps (`steps:`)

Add custom steps before agentic execution. If unspecified, a default checkout step is added automatically.

```yaml wrap
steps:
  - name: Install dependencies
    run: npm ci
```

Use custom steps to precompute data, filter triggers, or prepare context for AI agents.

## Custom Pre-Agent Steps (`pre-agent-steps:`)

Add custom steps before MCP gateway startup in the agent job so prerequisite MCP installation/configuration can happen first.

```yaml wrap
pre-agent-steps:
  - name: Finalize Context
    run: ./scripts/prepare-agent-context.sh
```

Use pre-agent steps when work must happen right before the engine runs (for example, final context preparation or last-moment validations).

## Custom Post-Execution Steps (`post-steps:`)

Add custom steps after agentic execution. Run after the AI engine completes regardless of success/failure (unless conditional expressions are used).

```yaml wrap
post-steps:
  - name: Upload Results
    if: always()
    uses: actions/upload-artifact@v4
    with:
      name: workflow-results
      path: /tmp/gh-aw/
      retention-days: 7
```

Useful for artifact uploads, summaries, cleanup, or triggering downstream workflows.

## Custom Jobs (`jobs:`)

Define custom jobs that run before agentic execution. The agentic execution job waits for all custom jobs to complete. Custom jobs can share data with the agent through artifacts or job outputs.

```yaml wrap
jobs:
  super_linter:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
      - name: Run Super-Linter
        uses: super-linter/super-linter@v7
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

### Supported Job-Level Fields

| Field | Description |
|---|---|
| `name` | Display name for the job |
| `needs` | Jobs that must complete before this job runs |
| `runs-on` | Runner label — string, array, or object form |
| `if` | Conditional expression to control job execution |
| `permissions` | GitHub token permissions for this job |
| `outputs` | Values exposed to downstream jobs |
| `env` | Environment variables available to all steps |
| `timeout-minutes` | Maximum job duration (GitHub Actions default: 360) |
| `concurrency` | Concurrency group to prevent parallel runs |
| `continue-on-error` | Allow the workflow to continue if this job fails |
| `container` | Docker container to run steps in |
| `services` | Service containers (e.g. databases) |
| `pre-steps` | Steps injected after compiler setup steps and before checkout/`steps` in that job |
| `steps` | List of steps — supports complete GitHub Actions step specification |
| `uses` | Reusable workflow to call |
| `with` | Input parameters for a reusable workflow |
| `secrets` | Secrets passed to a reusable workflow |

The `strategy` field (matrix builds) is not supported.

`runs-on` accepts a string, an array of runner labels, or the object form:

```yaml wrap
jobs:
  build:
    runs-on:
      group: my-runner-group
      labels: [self-hosted, linux]
    steps:
      - uses: actions/checkout@v6
```

When `jobs.<job-id>.pre-steps` is set, step execution order is deterministic:

1. Compiler-injected setup steps
2. `jobs.<job-id>.pre-steps`
3. Checkout steps
4. Remaining `jobs.<job-id>.steps`

Example using `timeout-minutes` and `env`:

```yaml wrap
jobs:
  build:
    runs-on: ubuntu-latest
    timeout-minutes: 15
    env:
      NODE_ENV: production
    steps:
      - uses: actions/checkout@v6
      - run: npm ci && npm run build
```

### Job Outputs

Custom jobs can expose outputs accessible in the agentic execution prompt via `${{ needs.job-name.outputs.output-name }}`:

```yaml wrap
jobs:
  release:
    outputs:
      release_id: ${{ steps.get_release.outputs.release_id }}
      version: ${{ steps.get_release.outputs.version }}
    steps:
      - id: get_release
        run: echo "version=${{ github.event.release.tag_name }}" >> $GITHUB_OUTPUT
---

Generate highlights for release ${{ needs.release.outputs.version }}.
```

Job outputs must be string values.

## Related Documentation

- [DeterministicOps](/gh-aw/patterns/deterministic-ops/) — Patterns combining deterministic steps with AI reasoning
- [Frontmatter Reference](/gh-aw/reference/frontmatter/) — Complete frontmatter field reference
- [Custom Safe Outputs](/gh-aw/reference/custom-safe-outputs/) — Custom post-processing jobs for agentic outputs
- [Imports](/gh-aw/reference/imports/) — Composing `pre-agent-steps` and `post-steps` across shared workflows
