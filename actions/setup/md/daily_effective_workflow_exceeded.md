**⚠️ Daily Workflow ET Guardrail Exceeded**: The agent was not started because the triggering user has already consumed the configured 24-hour effective-token budget for this workflow.

- **24h ET usage:** `{total_effective_tokens}` effective tokens
- **Configured threshold:** `{threshold}` effective tokens

The agent will resume automatically once the 24-hour rolling window resets. No action is required if the current limit is appropriate for your usage.

<details>
<summary>How to raise the daily limit</summary>

Set `max-daily-effective-tokens` in your workflow frontmatter to a higher value, then recompile:

```yaml
max-daily-effective-tokens: 5M
```

Common suffix shorthands: `K` = thousands, `M` = millions (e.g. `2M` = 2,000,000).

After editing the workflow source file, regenerate the compiled lock file:

```bash
gh aw compile
```

Commit and push the updated `.lock.yml` file.

> [!NOTE]
> Raising the limit increases the number of AI inference calls the workflow can make
> per 24-hour window per triggering user. Review your Copilot or model provider billing
> before significantly increasing the threshold (for example, before doubling the current
> value or setting it above 10M tokens).

</details>

<details>
<summary>What is the daily effective token guardrail?</summary>

The `max-daily-effective-tokens` frontmatter option sets a per-user, per-workflow spending cap measured in *effective tokens* — a normalized unit that accounts for the cost of each model call across the 24-hour window before the current run.

When a triggering user's aggregated effective-token usage across all completed runs of this workflow in the last 24 hours exceeds the threshold, the activation job sets the `daily_effective_workflow_exceeded` output to `true` and the agent job is skipped for that run. The conclusion job still runs and creates this report.

The guardrail is evaluated at activation time, not retrospectively, so a single very large run that pushes usage over the threshold only blocks *subsequent* runs in the same window — it does not cancel a run that is already in progress.

</details>

<details>
<summary>How to disable this guardrail</summary>

> [!CAUTION]
> Disabling this guardrail removes the per-user spending cap. Only disable it if you have
> an alternative mechanism for controlling token usage or if the workflow is intentionally
> uncapped.

Set `max-daily-effective-tokens: -1` in the workflow frontmatter to explicitly disable the guardrail, then recompile:

```yaml
max-daily-effective-tokens: -1
```

```bash
gh aw compile
```

Alternatively, remove the `max-daily-effective-tokens` key entirely to fall back to the enterprise-wide default (if one is configured) or to run with no per-workflow cap.

</details>
