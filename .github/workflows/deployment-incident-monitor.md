---
description: Monitors deployment failures and automatically creates deduplicated incident issues with root cause analysis.
on:
  deployment_status:
    state: [error, failure]
  skip-if-match: "is:issue is:open label:incident label:deployment-failure"
permissions:
  contents: read
  actions: read
  deployments: read
engine: copilot
tools:
  github:
    toolsets: [repos, actions]
safe-outputs:
  create-issue:
    expires: 7d
    title-prefix: "[Incident] "
    labels: [incident, deployment-failure]
    close-older-issues: true
  noop:
timeout-minutes: 10
---

# Deployment Incident Monitor

A deployment to **${{ github.event.deployment.environment }}** has failed with state `${{ github.event.deployment_status.state }}`.

## Your Task

Perform a root cause analysis of this deployment failure and create a focused incident issue.

## Deployment Context

- **Environment**: ${{ github.event.deployment.environment }}
- **Status**: ${{ github.event.deployment_status.state }}
- **Repository**: ${{ github.repository }}

## Investigation Steps

1. **Check for an existing open incident issue**: Look for open issues with both `incident` and `deployment-failure` labels. If one already exists for this environment and recent timeframe, call `noop` with a brief explanation.

2. **Gather context** using the available GitHub MCP tools:
   - Look up recent workflow runs and job logs in the `actions` toolset to identify what failed
   - Review recent commits to the deployed branch to identify changes that may have caused the failure
   - Check if there were any related CI failures preceding the deployment

3. **Create an incident issue** if no duplicate exists. The issue should include:
   - **Environment** and the deployment failure state
   - **Summary** of likely root cause based on available evidence
   - **Evidence**: relevant log excerpts, failing steps, or recent commits linked to the failure
   - **Suggested remediation** steps for the on-call team
   - A link to the failing deployment for quick access

## Output Guidelines

- Use `noop` if a duplicate open incident issue already exists.
- Keep the issue concise and actionable — focus on what the on-call engineer needs to know immediately.
- Do not create speculative issues; only create one when there is concrete evidence of a failure.
