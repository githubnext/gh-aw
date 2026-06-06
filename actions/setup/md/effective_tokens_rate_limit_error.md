**⚠️ AI Credits Budget Guidance**: The run hit a legacy effective-token rate-limit signal from the API proxy. gh-aw now uses AI Credits (AIC) as the primary cost metric, so migrate per-run budgeting to `max-ai-credits`.

<details>
<summary>Why this happened and how to optimize</summary>

- Learn about [AI Credits]({ai_credits_spec_link}).
{usage_line}{budget_line}{run_line}
- `max-effective-tokens` is deprecated; use `max-ai-credits` in workflow frontmatter.
You can tune this limit with `max-ai-credits` in workflow frontmatter.

{et_table_section}
- To budget and optimize this workflow, follow the [cost management guidance]({cost_management_link}).
</details>
