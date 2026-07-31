> [!WARNING]
> **AI Credits Rate Limit**
>
> The Copilot API returned a rate limit response (HTTP 429), but the workflow did not report the explicit AI credits budget-exceeded guardrail signal.{metrics_summary}

<details>
<summary>Tips for reducing rate limit issues</summary>

- Review the [cost optimization guidance](https://github.github.com/gh-aw/reference/cost-management/).
- Reduce unnecessary model or tool calls in the prompt.
- Trim large inputs or excess context that does not change the outcome.
- Split large tasks across smaller runs when possible.

</details>
