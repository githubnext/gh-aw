---
description: "Guard policy smoke test: repos=all, min-integrity=none (most permissive)"
on:
  workflow_dispatch:
  pull_request:
    types: [labeled]
    names: ["metal"]
  status-comment: true
permissions:
  contents: read
  issues: read
  pull-requests: read
name: "Smoke Agent: all/none"
engine: codex
strict: true
tools:
  github:
    mode: local
    allowed-repos: "all"
    min-integrity: none
network:
  allowed:
    - defaults
    - github
safe-outputs:
  allowed-domains: [default-safe-outputs]
  add-comment:
    hide-older-comments: true
    max: 2
  messages:
    footer: "> 🤖 *Guard policy smoke test by [{workflow_name}]({run_url})*{effective_tokens_suffix}{history_link}"
    run-started: "🔍 [{workflow_name}]({run_url}) testing guard policy: `repos=all, min-integrity=none`..."
    run-success: "✅ [{workflow_name}]({run_url}) completed guard policy test."
    run-failure: "❌ [{workflow_name}]({run_url}) {status}. Check the logs for details."
timeout-minutes: 10
---

# Guard Policy Smoke Test: all/none (most permissive)

This workflow tests the most permissive guard policy (`repos: all`, `min-integrity: none`).
It should be able to access all repositories without restriction.

## Instructions

Test GitHub MCP tool access under this guard policy by performing these operations and reporting results.

### Step 1: Search public repositories

Use `search_repositories` to search for `topic:actions` and return the top 3 results. Record:
- Repository full name (owner/repo)
- Star count
- Whether access was allowed or denied

### Step 2: Search private/internal repositories

Use `search_repositories` to search for `org:github gh-aw` to find repositories in the github org. Record:
- Repository full name
- Visibility (public/private/internal)
- Whether access was allowed or denied

### Step 3: List issues from this repository

Use `list_issues` on `${{ github.repository }}` with `state: open` and `per_page: 3`. Record:
- Issue number and title
- Whether access was allowed or denied

### Step 4: List issues from a different public repository

Use `list_issues` on `github/docs` with `state: open` and `per_page: 3`. Record:
- Issue number and title
- Whether access was allowed or denied

### Step 5: Report results

Use the `add_comment` safe-output tool to post a summary to the current PR:

```json
{
  "type": "add_comment",
  "body": "## Guard Policy Test Results: `all/none`\n\n### Policy\n- repos: `all`\n- min-integrity: `none`\n\n### Results\n<results from each step>\n\n### Expected\nAll operations should succeed (most permissive policy)."
}
```

If there is no PR context, use the `noop` tool to report the results summary.