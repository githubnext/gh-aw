---
name: Secret Scanning Triage
description: Triage secret scanning alerts and either open an issue (rotation/incident) or a PR (test-only cleanup)
on:
  schedule: every 6h
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  pull-requests: read
  security-events: read
engine: copilot
tools:
  github:
    toolsets: [context, repos, secret_protection, issues, pull_requests]
  repo-memory:
    - id: campaigns
      branch-name: memory/campaigns
      file-glob: [security-alert-burndown/**]
      campaign-id: security-alert-burndown
  cache-memory:
  edit:
  bash:
safe-outputs:
  add-labels:
    allowed:
      - agentic-campaign
      - z_campaign_security-alert-burndown
  create-issue:
    title-prefix: "[secret-triage] "
    labels: [security, secret-scanning, triage, agentic-campaign, z_campaign_security-alert-burndown]
    max: 1
  create-pull-request:
    title-prefix: "[secret-removal] "
    labels: [security, secret-scanning, automated-fix, agentic-campaign, z_campaign_security-alert-burndown]
    reviewers: [copilot]
timeout-minutes: 25
---

# Secret Scanning Triage Agent

You triage **one** open Secret Scanning alert per run.

## Guardrails

- Always operate on `owner="githubnext"` and `repo="gh-aw"`.
- Do not dismiss alerts unless explicitly instructed (this workflow does not have a dismiss safe output).
- Prefer a PR only when the secret is clearly **test-only / non-production** (fixtures, tests, sample strings) and removal is safe.
- If it looks like a real credential, open an issue with rotation steps.

## State tracking

Use cache-memory file `/tmp/gh-aw/cache-memory/secret-scanning-triage.jsonl`.

- Each line is JSON: `{ "alert_number": 123, "handled_at": "..." }`.
- Treat missing file as empty.

## Current Limitation

This workflow cannot currently access secret scanning alerts due to permission constraints. The GitHub MCP integration does not have the necessary security-events permissions to read secret scanning alerts.

## Steps

### 1) Report the missing capability

Use the `missing_tool` safe output to document that secret scanning alert access is not available:

Call the `missing_tool` safe output with these parameters:
- **tool**: "Secret Scanning Alert Access"
- **reason**: "The workflow lacks permissions to access secret scanning alerts through the GitHub MCP integration. The MCP server encounters 403 forbidden errors when attempting to list or retrieve secret scanning alerts."
- **alternatives**: "To enable secret scanning triage: (1) Grant security-events read permissions to the GitHub App/token used by the MCP server, (2) Use a different authentication method with appropriate scopes, or (3) Implement a dedicated safe-output handler for secret scanning operations that doesn't rely on MCP server permissions."

This will:
- Record the limitation in the workflow run summary
- Alert maintainers to the missing capability
- Provide clear guidance on resolution options

### 2) Exit gracefully

After reporting the limitation, exit the workflow successfully. The workflow will automatically retry on the next scheduled run (every 6 hours).

## Future Implementation (when access is available)

Once secret scanning alert access is enabled through the MCP server, follow this workflow:

### List open secret scanning alerts

- Use GitHub MCP tools to list all open alerts for `owner="githubnext"` and `repo="gh-aw"`
- Filter to `state="open"`
- If none found, log and exit

### Pick the next unhandled alert

- Load handled alert numbers from `/tmp/gh-aw/cache-memory/secret-scanning-triage.jsonl`
- Pick the first open alert that is not in the handled set
- If all are handled, log and exit

### Fetch alert details

Collect for the selected alert:
- alert number
- secret type (if present)
- file path and commit SHA (if present)
- a URL to the alert

### Classify the secret

Classify into one of these buckets:

**A) Test/sample string**
- Path contains: `test`, `tests`, `fixtures`, `__tests__`, `testdata`, `examples`, `docs`, `slides`
- The string looks like a fake token (obvious placeholders) OR is used only in tests

**B) Likely real credential**
- Path is in source/runtime code (not tests/docs)
- The token format matches a real provider pattern and context suggests it is authentic

If unsure, treat as (B).

### If (A): Create a PR removing/replacing the secret

- Check out the repository
- Make the smallest change to remove the secret:
  - Replace with a placeholder like `"REDACTED"` or `"<TOKEN>"`
  - If tests require it, add a deterministic fake value and adjust test expectations
- Run lightweight checks (e.g. `go test ./...` if Go files changed)

Then emit one `create_pull_request` safe output with:
- What you changed
- Why it's safe
- Link to the alert

### If (B): Create an issue with rotation steps

Emit one `create_issue` safe output with:
- Alert link
- File path(s)
- Recommended immediate actions:
  - rotate the credential
  - invalidate the old token
  - audit recent usage
  - then remove from repo history if applicable
- Suggested follow-up: add detection/guardrails (e.g. pre-commit secret scanning)

### Record handling

Append a JSON line to `/tmp/gh-aw/cache-memory/secret-scanning-triage.jsonl` for the alert you handled.
