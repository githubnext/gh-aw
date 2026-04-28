---
description: Smoke Copilot
on: 
  workflow_dispatch:
  label_command:
    name: smoke
    events: [pull_request]
  reaction: "eyes"
  status-comment: true
  github-token: ${{ secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}
permissions:
  contents: read
  pull-requests: read
  issues: read
  discussions: read
  actions: read
name: Smoke Copilot
engine:
  id: copilot
  max-continuations: 2
  bare: true
imports:
  - shared/github-guard-policy.md
  - shared/gh.md
  - shared/reporting.md
  - shared/github-queries-mcp-script.md
  - shared/mcp/serena-go.md
network:
  allowed:
    - defaults
    - node
    - github
    - playwright
tools:
  agentic-workflows:
  cache-memory: true
  comment-memory: true
  edit:
  bash:
    - "*"
  github:
    min-integrity: approved
    trusted-users:
      - pelikhan
  playwright:
  web-fetch:
  cli-proxy: true
runtimes:
  go:
    version: "1.25"
safe-outputs:
    allowed-domains: [default-safe-outputs]
    upload-artifact:
      max-uploads: 1
      retention-days: 1
      skip-archive: true
    add-comment:
      allowed-repos: ["github/gh-aw"]
      hide-older-comments: true
      max: 2
    create-issue:
      expires: 2h
      group: true
      close-older-issues: true
      close-older-key: "smoke-copilot"
      labels: [automation, testing]
    create-discussion:
      category: announcements
      labels: [ai-generated]
      expires: 2h
      close-older-discussions: true
      close-older-key: "smoke-copilot"
      max: 1
    create-pull-request-review-comment:
      max: 5
    submit-pull-request-review:
    reply-to-pull-request-review-comment:
      max: 5
    add-labels:
      allowed: [smoke-copilot]
      allowed-repos: ["github/gh-aw"]
    remove-labels:
      allowed: [smoke]
    set-issue-type:
    dispatch-workflow:
      workflows:
        - haiku-printer
      max: 1
    jobs:
      send-slack-message:
        description: "Send a message to Slack (stub for testing)"
        runs-on: ubuntu-latest
        output: "Slack message stub executed!"
        inputs:
          message:
            description: "The message to send"
            required: false
            default: ""
            type: string
        permissions:
          contents: read
        steps:
          - name: Stub Slack message
            run: |
              echo "🎭 This is a stub - not sending to Slack"
              if [ -f "$GH_AW_AGENT_OUTPUT" ]; then
                MESSAGE=$(cat "$GH_AW_AGENT_OUTPUT" | jq -r '.items[] | select(.type == "send_slack_message") | .message')
                echo "Would send to Slack: $MESSAGE"
                {
                  echo "### 📨 Slack Message Stub"
                  echo "**Message:** $MESSAGE"
                  echo ""
                  echo "> ℹ️ This is a stub for testing purposes. No actual Slack message is sent."
                } >> "$GITHUB_STEP_SUMMARY"
              else
                echo "No agent output found"
              fi
    messages:
      append-only-comments: true
      footer: "> 📰 *BREAKING: Report filed by [{workflow_name}]({run_url})*{effective_tokens_suffix}{history_link}"
      run-started: "📰 BREAKING: [{workflow_name}]({run_url}) is now investigating this {event_type}. Sources say the story is developing..."
      run-success: "📰 VERDICT: [{workflow_name}]({run_url}) has concluded. All systems operational. This is a developing story. 🎤"
      run-failure: "📰 DEVELOPING STORY: [{workflow_name}]({run_url}) reports {status}. Our correspondents are investigating the incident..."
timeout-minutes: 15
strict: false
---

# Smoke Test: Copilot Engine Validation

**IMPORTANT: Keep all outputs extremely short and concise. Use single-line responses where possible. No verbose explanations.**

## Tool Access Overview

This workflow uses `cli-proxy: true`. The following MCP servers are **NOT available as MCP tools** — they are mounted exclusively as **shell CLI commands** (see `<mcp-clis>` section above). You **must** use them via the `bash` tool:

- **`playwright`** — use `playwright <tool> [--param value...]` in bash (e.g. `playwright browser_navigate --url ...`)
- **`serena`** — use `serena <tool> [--param value...]` in bash (e.g. `serena activate_project --path ...`)
- **`agenticworkflows`** — use `agenticworkflows <tool> [--param value...]` in bash
- **`safeoutputs`** — use `safeoutputs <tool> [--param value...]` in bash (e.g. `safeoutputs add_comment --body "..."`)
- **`mcpscripts`** — use `mcpscripts <tool> [--param value...]` in bash (e.g. `mcpscripts mcpscripts-gh --args "..."`)

The `github` MCP server is **NOT** CLI-mounted — it remains available as a normal MCP tool.

Run `<server> --help` to list all available tools for a server, or `<server> <tool> --help` for detailed parameter info.

These are **not** MCP protocol tools — they are bash executables. Call them with the `bash` tool only.

## Test Requirements

1. **GitHub MCP Testing**: Review the last 2 merged pull requests in ${{ github.repository }}
2. **MCP Scripts GH CLI Testing**: Use the `mcpscripts-gh` tool to query 2 pull requests from ${{ github.repository }} (use args: "pr list --repo ${{ github.repository }} --limit 2 --json number,title,author")
3. **Serena CLI Testing**: 
   - Use bash to run `serena activate_project --path ${{ github.workspace }}` to initialize the workspace and verify it succeeds (do NOT use bash to run go commands - use the serena CLI only)
   - After initialization, use bash to run `serena find_symbol --name_path <symbol>` to search for symbols and verify that at least 3 symbols are found in the results
4. **Playwright CLI Testing**: Use bash to run `playwright browser_navigate --url https://github.com` to navigate to <https://github.com>, then `playwright browser_snapshot` to capture the page and verify the title contains "GitHub" (do NOT try to install playwright - use the `playwright` CLI command via bash only)
5. **Web Fetch Testing**: Use the web-fetch tool to fetch https://github.com and verify the response contains "GitHub" (do NOT use bash or playwright for this test - use the web-fetch tool directly)
6. **File Writing Testing**: Create a test file `/tmp/gh-aw/agent/smoke-test-copilot-${{ github.run_id }}.txt` with content "Smoke test passed for Copilot at $(date)" (create the directory if it doesn't exist)
7. **Bash Tool Testing**: Execute bash commands to verify file creation was successful (use `cat` to read the file back)
8. **Discussion Interaction Testing**: 
   - Use the `github-discussion-query` mcp-script tool with params: `limit=1, jq=".[0]"` to get the latest discussion from ${{ github.repository }}
   - Extract the discussion number from the result (e.g., if the result is `{"number": 123, "title": "...", ...}`, extract 123)
   - Use the `add_comment` tool with `discussion_number: <extracted_number>` to add a fun, playful comment stating that the smoke test agent was here
9. **Build gh-aw**: Run `GOCACHE=/tmp/go-cache GOMODCACHE=/tmp/go-mod make build` to verify the agent can successfully build the gh-aw project (both caches must be set to /tmp because the default cache locations are not writable). If the command fails, mark this test as ❌ and report the failure.
10. **Upload gh-aw binary as artifact**: After a successful build, use bash to copy the `./gh-aw` binary into the staging directory (`mkdir -p $RUNNER_TEMP/gh-aw/safeoutputs/upload-artifacts && cp ./gh-aw $RUNNER_TEMP/gh-aw/safeoutputs/upload-artifacts/gh-aw`), then call the `upload_artifact` safe-output tool with `path: "gh-aw"`. The `upload_artifact` tool is available and configured in this workflow run — use it directly, do NOT use `missing_tool` for it. Mark this test as ❌ if the build in step 9 failed.
11. **Discussion Creation Testing**: Use the `create_discussion` safe-output tool to create a discussion in the announcements category titled "copilot was here" with the label "ai-generated"
12. **Workflow Dispatch Testing**: Use the `dispatch_workflow` safe output tool to trigger the `haiku-printer` workflow with a haiku as the message input. Create an original, creative haiku about software testing or automation.
13. **PR Review Testing**: Review the diff of the current pull request. Leave 1-2 inline `create_pull_request_review_comment` comments on specific lines, then call `submit_pull_request_review` with a brief body summarizing your review and event `COMMENT`. To test `reply_to_pull_request_review_comment`: use the `pull_request_read` tool (with `method: "get_review_comments"` and `pullNumber: ${{ github.event.pull_request.number }}`) to fetch the PR's existing review comments, then reply to the most recent one using `reply_to_pull_request_review_comment` with its actual numeric `id` as the `comment_id`. Note: `create_pull_request_review_comment` does not return a `comment_id` — you must fetch existing comment IDs from the GitHub API. If the PR has no existing review comments, skip the reply sub-test.
14. **Comment Memory Testing**: Append an original 3-line haiku to the comment-memory markdown file(s) in `/tmp/gh-aw/comment-memory/*.md` without removing existing content.

## Output

1. **Create an issue** with a summary of the smoke test run:
   - Use a temporary ID (e.g. `aw_smoke1`) for the issue so you can reference it later
   - Title: "Smoke Test: Copilot - ${{ github.run_id }}"
   - Body should include:
     - Test results (✅ or ❌ for each test)
     - Overall status: PASS or FAIL
     - Run URL: ${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}
     - Timestamp
     - Pull request author and assignees

2. **Set Issue Type**: Use the `set_issue_type` safe-output tool with `issue_number: "aw_smoke1"` (the temporary ID from step 1) and `issue_type: "Bug"` to set the type of the just-created smoke test issue.

3. **Only if this workflow was triggered by a pull_request event**: Use the `add_comment` tool to add a **very brief** comment (max 5-10 lines) to the triggering pull request (omit the `item_number` parameter to auto-target the triggering PR) with:
   - PR titles only (no descriptions)
   - ✅ or ❌ for each test result
   - Overall status: PASS or FAIL
   - Mention the pull request author and any assignees

4. Use the `add_comment` tool to add a **fun and creative comment** to the latest discussion (using the `discussion_number` you extracted in step 8) - be playful and entertaining in your comment

5. Use the `send_slack_message` tool to send a brief summary message (e.g., "Smoke test ${{ github.run_id }}: All tests passed! ✅")

If all tests pass and this workflow was triggered by a pull_request event:
- Use the `add_labels` safe-output tool to add the label `smoke-copilot` to the pull request (omit the `item_number` parameter to auto-target the triggering PR)
- Use the `remove_labels` safe-output tool to remove the label `smoke` from the pull request (omit the `item_number` parameter to auto-target the triggering PR)

{{#import shared/noop-reminder.md}}
