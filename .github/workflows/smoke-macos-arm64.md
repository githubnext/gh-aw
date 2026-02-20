---
description: Smoke macOS ARM64
on:
  workflow_dispatch:
  pull_request:
    types: [labeled]
    names: ["smoke"]
  reaction: "eyes"
  status-comment: true
permissions:
  contents: read
  pull-requests: read
  issues: read
  discussions: read
  actions: read
name: Smoke macOS ARM64
engine: copilot
runs-on: macos-latest
imports:
  - shared/gh.md
  - shared/reporting.md
  - shared/github-queries-safe-input.md
tools:
  cache-memory: true
  edit:
  bash:
    - "*"
  web-fetch:
runtimes:
  go:
    version: "1.25"
sandbox:
  agent: false
strict: false
safe-outputs:
    add-comment:
      allowed-repos: ["github/gh-aw"]
      hide-older-comments: true
      max: 2
    create-issue:
      expires: 2h
      group: true
      close-older-issues: true
    create-discussion:
      category: announcements
      labels: [ai-generated]
      expires: 2h
      close-older-discussions: true
      max: 1
    create-pull-request-review-comment:
      max: 5
    submit-pull-request-review:
    add-labels:
      allowed: [smoke-macos-arm64]
      allowed-repos: ["github/gh-aw"]
    remove-labels:
      allowed: [smoke]
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
            required: true
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
      footer: "> 📰 *BREAKING: Report filed by [{workflow_name}]({run_url})*"
      run-started: "📰 BREAKING: [{workflow_name}]({run_url}) is now investigating this {event_type}. Sources say the story is developing..."
      run-success: "📰 VERDICT: [{workflow_name}]({run_url}) has concluded. All systems operational. This is a developing story. 🎤"
      run-failure: "📰 DEVELOPING STORY: [{workflow_name}]({run_url}) reports {status}. Our correspondents are investigating the incident..."
timeout-minutes: 15
---

# Smoke Test: macOS ARM64 Agent Validation

**IMPORTANT: Keep all outputs extremely short and concise. Use single-line responses where possible. No verbose explanations.**

**PURPOSE**: This smoke test validates that the Copilot CLI agent works correctly on macOS ARM64 (Apple Silicon) runners. The agent runs directly on the host without network firewall or containerized MCP servers, since Docker is not available on macOS GHA runners.

## Test Requirements

1. **Architecture Verification**: Run `uname -m` and `uname -s` to confirm you are running on an ARM64 macOS host. Report the architecture and OS.
2. **Environment Check**: Run `sw_vers` to report the macOS version. Run `go version` to confirm Go is available.
3. **GitHub MCP Testing**: Review the last 2 merged pull requests in ${{ github.repository }}. Report the PR numbers and titles.
4. **Safe Inputs GH CLI Testing**: Use the `safeinputs-gh` tool to query 2 pull requests from ${{ github.repository }} (use args: "pr list --repo ${{ github.repository }} --limit 2 --json number,title,author")
5. **File Writing Testing**: Create a test file `/tmp/gh-aw/agent/smoke-test-macos-arm64-${{ github.run_id }}.txt` with content "Smoke test passed for macOS ARM64 at $(date)" (create the directory if it doesn't exist)
6. **Bash Tool Testing**: Execute bash commands to verify file creation was successful (use `cat` to read the file back)
7. **Discussion Interaction Testing**:
    - Use the `github-discussion-query` safe-input tool with params: `limit=1, jq=".[0]"` to get the latest discussion from ${{ github.repository }}
    - Extract the discussion number from the result (e.g., if the result is `{"number": 123, "title": "...", ...}`, extract 123)
    - Use the `add_comment` tool with `discussion_number: <extracted_number>` to add a fun, playful comment stating that the macOS ARM64 smoke test agent was here
8. **Build gh-aw**: Run `GOCACHE=/tmp/go-cache GOMODCACHE=/tmp/go-mod make build` to verify the agent can successfully build the gh-aw project (both caches must be set to /tmp because the default cache locations are not writable). If the command fails, mark this test as ❌ and report the failure.
9. **Discussion Creation Testing**: Use the `create_discussion` safe-output tool to create a discussion in the announcements category titled "macos-arm64 was here" with the label "ai-generated"
10. **Workflow Dispatch Testing**: Use the `dispatch_workflow` safe output tool to trigger the `haiku-printer` workflow with a haiku as the message input. Create an original, creative haiku about Apple Silicon.
11. **PR Review Testing**: Review the diff of the current pull request. Leave 1-2 inline `create_pull_request_review_comment` comments on specific lines, then call `submit_pull_request_review` with a brief body summarizing your review and event `COMMENT`.
12. **Web Fetch Testing**: Use the `web-fetch` tool to fetch https://api.github.com and verify the response contains valid JSON with a `current_user_url` field.

## Output

1. **Create an issue** with a summary of the smoke test run:
   - Title: "Smoke Test: macOS ARM64 - ${{ github.run_id }}"
   - Body should include:
     - Host architecture and OS info (ARM64/macOS version)
     - Go version
     - Test results (✅ or ❌ for each test)
     - Note: Docker/containers are not tested (not available on macOS GHA runners)
     - Overall status: PASS or FAIL
     - Run URL: ${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}
     - Timestamp
     - Pull request author and assignees

2. Add a **very brief** comment (max 5-10 lines) to the current pull request with:
   - Host arch confirmation (ARM64/macOS)
   - ✅ or ❌ for each test result
   - Overall status: PASS or FAIL
   - Mention the pull request author and any assignees

3. Use the `add_comment` tool to add a **fun and creative comment** to the latest discussion (using the `discussion_number` you extracted in step 7) - be playful and entertaining in your comment

4. Use the `send_slack_message` tool to send a brief summary message (e.g., "macOS ARM64 smoke test ${{ github.run_id }}: All tests passed! ✅")

If all tests pass:
- Use the `add_labels` safe-output tool to add the label `smoke-macos-arm64` to the pull request
- Use the `remove_labels` safe-output tool to remove the label `smoke` from the pull request
