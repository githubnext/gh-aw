> [!WARNING]
> **AI Credits Budget Guidance**: The run hit a legacy effective-token rate-limit signal from the API proxy. gh-aw now uses AI Credits (AIC) as the primary cost metric, so migrate per-run budgeting to `max-ai-credits`.

<details>
<summary>Why this happened and how to optimize</summary>

- Learn about [AI Credits]({ai_credits_spec_link}).
{budget_line}
- `max-effective-tokens` is deprecated; migrate to `max-ai-credits` by running `gh aw fix --write`, or update manually (1 AIC = 10,000 ET):
  ```yaml
  # before
  max-effective-tokens: 5000000
  # after
  max-ai-credits: 500
  ```

- To budget and optimize this workflow, follow the [cost management guidance]({cost_management_link}).
</details>
