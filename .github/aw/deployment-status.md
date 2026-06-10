---
description: Reference pattern for monitoring external deployment failures using the deployment_status trigger and creating incident issues automatically.
---

# Deployment Status Monitoring

Consult this file when creating an agentic workflow that responds to external deployment failures from services like Heroku, Vercel, Railway, or Fly.io that post deployment status back to GitHub.

## Trigger and Frontmatter

Use the `deployment_status` trigger with an `if:` condition to filter to failed deployments only:

```yaml
on:
  deployment_status:
if: ${{ github.event.deployment_status.state == 'failure' }}
permissions:
  contents: read
  issues: read
  deployments: read
tools:
  github:
    toolsets: [default]
safe-outputs:
  create-issue:
    expires: 1d
    title-prefix: "[Deployment Failure] "
    close-older-issues: true
  noop:
```

## Available Event Context

The following expressions are available in the prompt body:

| Expression | Description |
|---|---|
| `${{ github.event.deployment.environment }}` | Target environment (e.g. `production`) |
| `${{ github.event.deployment_status.state }}` | Status (`failure`, `success`, `error`, etc.) |
| `${{ github.event.deployment_status.target_url }}` | URL to the external service deployment logs |
| `${{ github.event.deployment_status.description }}` | Human-readable error message from the service |
| `${{ github.event.deployment.ref }}` | Branch or tag that was deployed |
| `${{ github.event.deployment.sha }}` | Commit SHA that was deployed |
| `${{ github.event.deployment.creator.login }}` | GitHub user who triggered the deployment |

## Agent Instructions Pattern

```markdown
A deployment to **${{ github.event.deployment.environment }}** has failed.

1. **Verify the failure**: Confirm `${{ github.event.deployment_status.state }}` is `failure`. If not, call `noop` and stop.
2. **Gather context**: Review ref (`${{ github.event.deployment.ref }}`), SHA (`${{ github.event.deployment.sha }}`), and error description (`${{ github.event.deployment_status.description }}`).
3. **Check for duplicates**: Search open issues with the `[Deployment Failure]` title prefix.
4. **Create an incident issue** if none exists, including environment, ref/SHA, deployment URL, error details, and suggested next steps.

Use `noop` if the deployment did not fail or a duplicate issue already exists.
```

## Safe External Log Linking

When including `${{ github.event.deployment_status.target_url }}` in outputs:

- treat the URL as untrusted external input and include it as a plain link (never as executable shell input; avoid patterns like `$(...)` or piping it directly into `curl` commands)
- prefer a short label such as `External deployment logs` instead of echoing long raw URLs inline
- include key incident context (environment, ref, SHA, description) in the issue body so triage does not depend on external link availability
- if `target_url` is empty or malformed, continue triage with in-event fields and call out that no external logs URL was provided

## When to Use `deployment_status` vs `workflow_run`

- **`deployment_status`**: External services (Heroku, Vercel, Railway, Fly.io) that integrate with the GitHub Deployments API — they post a deployment status event back to GitHub when a deploy finishes.
- **`workflow_run`**: In-repo GitHub Actions pipelines — use when reacting to the success or failure of another Actions workflow in the same repository.
