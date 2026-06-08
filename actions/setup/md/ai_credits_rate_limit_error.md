> [!WARNING]
> **AI Credits Budget Exceeded**
>
> The workflow hit the configured `max-ai-credits` guardrail.{metrics_table}

<details>
<summary>Tips for reducing AI credit usage</summary>

- Review the [cost optimization guidance](https://github.github.com/gh-aw/reference/cost-management/).
- Increase the `max-ai-credits` limit in the workflow frontmatter if the task legitimately requires more credits.
- Reduce unnecessary model or tool calls in the prompt.
- Trim large inputs or excess context that does not change the outcome.
- Split large tasks across smaller runs when possible.

</details>
