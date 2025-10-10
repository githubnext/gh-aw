---
"githubnext/gh-aw": minor
---

Remove timeout requirement for strict mode and set default timeout to 20 minutes

This change makes strict mode more flexible by removing the requirement to specify `timeout_minutes`. Workflows can still set a timeout for cost control, but it's no longer mandatory for enabling strict mode's security features. The default timeout for agentic workflows has also been increased from 5 to 20 minutes to better accommodate typical workflow execution times.
