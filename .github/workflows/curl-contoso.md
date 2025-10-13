---
on:
  workflow_dispatch:
permissions: read-all
engine: copilot
tools:
  bash:
    - "curl contoso.com*"
safe-outputs: {}
network:
  allowed:
    - "contoso.com"
---

# Curl contoso

Run a single, explicit curl to contoso.com and return a concise summary.

## Instructions for the agent

- Execute exactly one shell command using the `bash` tool: `curl -sS --max-time 10 contoso.com`
- Capture the HTTP status code and the first 200 characters of the response body.
- Do not make any additional network requests or external calls.
- If the request fails or times out, return a short error message describing the failure.

## Output

- Provide a JSON object with keys: `status` (HTTP status code or null), `body_preview` (string), and `error` (null or error message).

## Notes

- This workflow is intentionally minimal and uses least-privilege permissions.
