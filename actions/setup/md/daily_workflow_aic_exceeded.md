> [!WARNING]
> **Daily Workflow AIC Guardrail Exceeded**: The agent was not started because this workflow has already consumed the configured 24-hour AI Credits budget.

- **24h AIC usage:** `{total_aic}` AI Credits
- **Configured threshold:** `{threshold}` AI Credits

The agent will resume automatically once the 24-hour rolling window resets. No action is required if the current limit is appropriate for your usage.

<sub>AIC values are approximate. Consult the billing dashboards for accurate usage and charges.</sub>

<details>
<summary>How to raise the daily limit</summary>

Set `max-daily-ai-credits` in your workflow frontmatter to a higher value, then recompile:

```yaml
max-daily-ai-credits: 20K
```

Common suffix shorthands: `K` = thousands, `M` = millions (e.g. `2M` = 2,000,000).

After editing the workflow source file, regenerate the compiled lock file:

```bash
gh aw compile
```

Commit and push the updated `.lock.yml` file.

> [!NOTE]
> Raising the limit increases the number of AI inference calls the workflow can make
> per 24-hour window. Review your Copilot or model provider billing
> before significantly increasing the threshold (for example, before doubling the current
> value or setting it far above expected usage).

</details>

<details>
<summary>What is the daily AI Credits guardrail?</summary>

The `max-daily-ai-credits` frontmatter option sets a per-workflow spending cap measured in *AI Credits* across the 24-hour window before the current run. The cap is scoped to the repository and workflow — it aggregates usage across all runs of this workflow regardless of who triggered them.

When the aggregated AI Credits usage across all completed runs of this workflow in the last 24 hours exceeds the threshold, the activation job sets the `daily_effective_workflow_exceeded` output to `true` and the agent job is skipped for that run. The conclusion job still runs and creates this report.

The guardrail is evaluated at activation time, not retrospectively, so a single very large run that pushes usage over the threshold only blocks *subsequent* runs in the same window — it does not cancel a run that is already in progress.

</details>

<details>
<summary>How to disable this guardrail</summary>

> [!CAUTION]
> Disabling this guardrail removes the per-workflow spending cap. Only disable it if you have
> an alternative mechanism for controlling AI cost usage or if the workflow is intentionally
> uncapped.

Set `max-daily-ai-credits: -1` in the workflow frontmatter to explicitly disable the guardrail, then recompile:

```yaml
max-daily-ai-credits: -1
```

```bash
gh aw compile
```

Alternatively, remove the `max-daily-ai-credits` key entirely to fall back to the enterprise-wide default (if one is configured) or to run with no per-workflow cap.

</details>
