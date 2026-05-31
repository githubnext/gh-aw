**⚠️ Daily Workflow ET Guardrail Exceeded**: The activation job blocked this workflow because the triggering user already consumed the configured 24-hour effective-token budget for this workflow.

- Aggregated 24-hour ET usage: `{total_effective_tokens}`
- Configured threshold: `{threshold}`

Wait for the 24-hour window to age out or raise `max-daily-effective-tokens` in the workflow frontmatter if the higher budget is intentional.
