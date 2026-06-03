**⚠️ Effective Token Budget Exhausted**: The run failed due to effective-token budget/rate-limit enforcement in the API proxy.

<details>
<summary>Why this happened and how to optimize</summary>

- Learn about [effective tokens]({et_spec_link}).
{usage_line}{budget_line}{run_line}
You can tune this limit with `max-effective-tokens` in workflow frontmatter.

{et_table_section}
- To optimize this workflow, follow the [token optimization instructions]({token_opt_link}).
</details>
