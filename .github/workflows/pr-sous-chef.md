---
private: true
emoji: "👨‍🍳"
name: PR Sous Chef
description: Keeps open non-draft PRs moving toward maintainer investigation by posting targeted Copilot nudges
on:
  schedule: every 15m
  workflow_dispatch:
  skip-if-no-match: "is:pr is:open -is:draft"
permissions:
  contents: read
  pull-requests: read
  issues: read
  actions: read
  copilot-requests: write

sandbox:
  agent:
    sudo: false

checkout:
  fetch: ["refs/pulls/open/*"]
  fetch-depth: 0
network:
  allowed: ["defaults", "go"]
engine:
  id: copilot
  model: gpt-5-mini
strict: true
imports:
  - shared/otlp.md
tools:
  cli-proxy: true
  github:
    mode: gh-proxy
    min-integrity: approved
    toolsets: [pull_requests, repos, issues]
  edit:
  bash:
    - "cat *"
    - "jq *"
    - "date *"
    - "git fetch:*"
    - "git checkout:*"
    - "git diff:*"
    - "git status"
    - "git restore:*"
    - "make fmt"
steps:
  - name: Fetch open non-draft PR queue
    id: fetch-prs
    env:
      GH_TOKEN: ${{ secrets.GH_AW_GITHUB_MCP_SERVER_TOKEN || secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}
      EXPR_GITHUB_REPOSITORY: ${{ github.repository }}
    run: |
      mkdir -p /tmp/gh-aw/agent
      candidate_file=/tmp/gh-aw/agent/pr-sous-chef-candidates.json
      eligible_file=/tmp/gh-aw/agent/pr-sous-chef-eligible.json
      sous_chef_nudge_marker='<!-- gh-aw-pr-sous-chef-nudge -->'
      cooldown_seconds=1800
      filtered_checks_pending=0
      filtered_last_comment_from_sous_chef=0
      filtered_cooldown=0

      gh pr list --repo "$EXPR_GITHUB_REPOSITORY" \
        --state open \
        --search "is:pr is:open -is:draft sort:updated-desc" \
        --limit 30 \
        --json number,title,url,headRefOid,headRefName,updatedAt,author,mergeStateStatus \
        > "$candidate_file"

      jq -n '[]' > "$eligible_file"

      while IFS= read -r pr; do
        pr_number="$(jq -r '.number' <<<"$pr")"
        if [ -z "$pr_number" ] || [ "$pr_number" = "null" ]; then
          continue
        fi

        checks_state="$(
          {
            gh aw checks "$pr_number" --repo "$EXPR_GITHUB_REPOSITORY" --json \
              | jq -r '.required_state // .state // "unknown"'
          } 2>/dev/null || echo "unknown"
        )"
        if [ "$checks_state" = "pending" ]; then
          filtered_checks_pending=$((filtered_checks_pending + 1))
          continue
        fi

        # Fetch the 10 most-recent issue comments once; used for the skip checks below.
        recent_comments_json="$(
          gh api "repos/$EXPR_GITHUB_REPOSITORY/issues/$pr_number/comments?per_page=10&sort=created&direction=desc" \
            2>/dev/null || echo "[]"
        )"

        # Skip if the very last comment was posted by pr-sous-chef (never add two in a row).
        last_comment_is_sous_chef="$(
          jq -r --arg marker "$sous_chef_nudge_marker" '
            if length == 0 then "false"
            elif (.[0].body // "" | contains($marker)) then "true"
            else "false"
            end
          ' <<<"$recent_comments_json"
        )"
        if [ "$last_comment_is_sous_chef" = "true" ]; then
          filtered_last_comment_from_sous_chef=$((filtered_last_comment_from_sous_chef + 1))
          continue
        fi

        # Skip if pr-sous-chef commented within the last 30 minutes (cooldown period).
        last_sous_chef_comment_at="$(
          jq -r --arg marker "$sous_chef_nudge_marker" '
            [.[] | select(.body // "" | contains($marker))] | .[0].created_at // ""
          ' <<<"$recent_comments_json"
        )"
        if [ -n "$last_sous_chef_comment_at" ]; then
          comment_epoch="$(date -d "$last_sous_chef_comment_at" +%s 2>/dev/null || echo 0)"
          current_epoch="$(date +%s)"
          if [ $(( current_epoch - comment_epoch )) -lt "$cooldown_seconds" ]; then
            filtered_cooldown=$((filtered_cooldown + 1))
            continue
          fi
        fi

        jq --argjson pr "$pr" '. + [$pr]' "$eligible_file" > "${eligible_file}.tmp" && mv "${eligible_file}.tmp" "$eligible_file"
      # Process substitution keeps the loop in the current shell so counters persist.
      done < <(jq -c '.[]' "$candidate_file")

      jq --argjson filtered_checks_pending "$filtered_checks_pending" \
         --argjson filtered_last_comment_from_sous_chef "$filtered_last_comment_from_sous_chef" \
         --argjson filtered_cooldown "$filtered_cooldown" '{
        fetched: (length),
        generated_at: (now | todate),
        filtered_checks_pending: $filtered_checks_pending,
        filtered_last_comment_from_sous_chef: $filtered_last_comment_from_sous_chef,
        filtered_cooldown: $filtered_cooldown,
        prs: map({
          number,
          title,
          url,
          headRefOid,
          headRefName,
          updatedAt,
          author: (.author.login // "unknown"),
          mergeStateStatus
        })
      }' "$eligible_file" \
        > /tmp/gh-aw/agent/pr-sous-chef-candidates-compact.json
      echo "eligible_count=$(jq '.prs | length' /tmp/gh-aw/agent/pr-sous-chef-candidates-compact.json || echo 0)" >> "$GITHUB_OUTPUT"
  - name: Setup Go
    if: steps.fetch-prs.outputs.eligible_count != '0'
    uses: actions/setup-go@v6.5.0
    with:
      go-version-file: go.mod
      cache: true
  - name: Setup Node.js
    if: steps.fetch-prs.outputs.eligible_count != '0'
    uses: actions/setup-node@v6.4.0
    with:
      node-version: "24"
      cache: npm
      cache-dependency-path: actions/setup/js/package-lock.json
  - name: Install formatter dependencies
    if: steps.fetch-prs.outputs.eligible_count != '0'
    run: npm ci --prefix actions/setup/js
safe-outputs:
  add-comment:
    max: 20
    target: "*"
    github-token: ${{ secrets.AWI_MAINTENANCE_TOKEN }}
  update-pull-request:
    title: false
    body: true
    operation: append
    update-branch: true
    max: 10
    target: "*"
  push-to-pull-request-branch:
    target: "*"
    if-no-changes: ignore
    commit-title-suffix: " [pr-sous-chef]"
    excluded-files:
      - ".github/workflows/**"
    max: 10
  mentions:
    allowed: ["@copilot"]
  noop:
  messages:
    run-started: "🍳 [{workflow_name}]({run_url}) is preparing PRs for maintainer investigation."
    run-success: "✅ [{workflow_name}]({run_url}) finished PR sous-chef nudges."
    run-failure: "⚠️ [{workflow_name}]({run_url}) {status} while preparing PRs."
timeout-minutes: 25
---

# PR Sous Chef 🍳

You are **pr-sous-chef**, a lightweight PR progress assistant.

## Mission

Move open non-draft PRs toward a state where a maintainer can investigate quickly.

## Token efficiency rules (mandatory)

1. Read `/tmp/gh-aw/agent/pr-sous-chef-candidates-compact.json` first.
2. If `prs` is empty, call `noop` with `"No open non-draft PRs to process"` and stop.
3. Process PRs in `updatedAt` descending order.
4. Process at most **5 PRs** per run. Remaining eligible PRs will be handled in the next scheduled run.
5. Use the `pr-processor` sub-agent for each PR; pass only the PR number and compact context.
6. If a `pr-processor` call returns non-JSON or an error, record `{pr_number: <N>, skip_reason: "sub_agent_error"}` in the `skipped` array of the run-summary noop payload and move to the next PR without retrying.
7. Do not fetch full PR diffs or large file lists unless absolutely required for a skip decision.
8. **Never finish without at least one safe-output tool call.** If you have not called `add_comment` or `update_pull_request`, you must call the run-summary `noop` (see **Run summary** below) before finishing.
9. Use `safeoutputs <tool> --param value` shell commands for all safe-output operations (`add_comment`, `update_pull_request`, `push_to_pull_request_branch`, `noop`, `report_incomplete`). Do **not** use `gh pr comment`, `gh pr update-branch`, `gh api ... -X POST`, or any GitHub API write calls outside of `safeoutputs`. Do **not** pipe `safeoutputs` to other commands or run `safeoutputs --help` — the tool schemas are already provided; use the examples below directly.

## Required skip rules per PR

Before any nudge for a PR:

1. **Skip when checks/actions are running on the PR head branch**
   - Candidate prefilter already uses `gh aw checks` and removes PRs with `required_state == pending`.
   - Detect pending/running checks via GitHub PR check runs / statuses for the head SHA.
   - If any check is `queued`, `in_progress`, or `pending`, skip this PR.

2. **Skip when the latest PR comment is from pr-sous-chef itself**
   - Candidate prefilter already removes PRs when the latest issue comment body includes the hidden marker `<!-- gh-aw-pr-sous-chef-nudge -->`.
   - Inspect PR comments ordered by recency.
   - Treat a comment as from pr-sous-chef only when the latest comment body contains `<!-- gh-aw-pr-sous-chef-nudge -->`.
   - If true, skip to avoid back-to-back nudges.

3. **Skip during the 30-minute cooldown after a pr-sous-chef comment**
   - Candidate prefilter already removes PRs where the most recent sous-chef comment was posted within the last 30 minutes.
   - If any recent comment contains `<!-- gh-aw-pr-sous-chef-nudge -->` and was created less than 30 minutes ago, skip this PR.

## Required nudges for eligible PRs

For each PR that is not skipped:

0. **Run formatters and push if needed**
   - Checkout the PR branch: `git checkout <headRefName>`
   - Run `make fmt` to format all code (Go, JavaScript, JSON)
   - Check for changes: `git diff --quiet || echo "dirty"`
   - If dirty, call `push_to_pull_request_branch` with the PR number to push the formatting fixes
   - Return to the original branch: `git checkout -`
   - Skip this step silently if `make fmt` exits non-zero (tools unavailable)

1. **Update branch if possible**
   - If the PR is behind its base branch (or otherwise indicates branch update needed), attempt `update_pull_request` with `update_branch: true`.
   - Use a minimal append body marker so maintainers can trace the action, including `pr-sous-chef` and the run URL.
   - Example (`update_pull_request` shell call):
     ```bash
     safeoutputs update_pull_request --pull_request_number 12345 --update_branch true --body "<!-- pr-sous-chef branch update -->" --operation append
     ```

2. **Post exactly one combined nudge comment**
   - **At most ONE `add_comment` call per PR per run.** Never post two comments to the same PR in a single run.
   - Inspect PR review threads and comments for unresolved feedback.
   - If unresolved PR reviews exist, include an explicit unresolved-reviews list in the nudge comment (reviewer + direct link for each unresolved review thread, newest first).
   - Combine all nudges (unresolved review feedback, branch-refresh request, completion plan, etc.) into **one single comment** that includes:
     - `<!-- gh-aw-pr-sous-chef-nudge -->` as the first hidden marker line (required — this is how the cooldown and duplicate-comment checks detect sous-chef).
     - @copilot mention with a concise, actionable instruction covering all relevant nudges in one message, including a direct instruction to run the `pr-finisher` skill.
   - Every `add_comment` must include `pr_number` set to the current PR's numeric `number` from the loop item.
   - Never emit `add_comment` without a numeric target field (`pr_number`/`pull_request_number`/`issue_number`/`item_number`) when `target: "*"` is configured.
   - Example (`add_comment` shell call):
     ```bash
     safeoutputs add_comment --pr_number 12345 --body $'<!-- gh-aw-pr-sous-chef-nudge -->\n@copilot please run the `pr-finisher` skill, address unresolved review comments, and rerun checks once the branch is up to date.'
     ```

### Run summary

At the end, call **exactly one** `noop` with a compact summary including counts (this final run-summary `noop` is mandatory and counts as the required safe-output call when no other actions were taken):
- processed
- skipped_checks_running
- skipped_last_comment_from_sous_chef
- skipped_cooldown
- nudged
- branch_update_attempts
- formatter_pushes (number of PRs that had formatting fixes committed and pushed)

Example (`noop` shell call):

```bash
safeoutputs noop --message "processed=4; skipped_checks_running=0; skipped_last_comment_from_sous_chef=1; skipped_cooldown=1; nudged=2; branch_update_attempts=0; formatter_pushes=0"
```

## Formatting Requirements

- **Header Levels**: Use h3 (`###`) or lower for all headers in your report to maintain proper document hierarchy. Never use h1 (`#`) or h2 (`##`) headers.
- **Progressive Disclosure**: Wrap long sections or verbose details in `<details><summary>Section Name</summary>` tags to improve readability and reduce scrolling.
- Keep critical information visible (summary, key outcomes, and recommendations) and use collapsible sections for secondary details.

### Recommended Report Structure

1. **Overview**: 1-2 paragraphs summarizing key findings (always visible)
2. **Critical Information**: Key metrics, status, critical issues (always visible)
3. **Details**: Use `<details><summary>Section Name</summary>` for expanded content
4. **Recommendations**: Actionable next steps (always visible)

## agent: `pr-processor`
---
description: Processes one PR with minimal API calls and returns skip/nudge decisions
model: claude-haiku-4.5
---
Given one PR number and compact metadata:

1. Check skip conditions in this order:
   - checks/actions running
   - latest comment contains `<!-- gh-aw-pr-sous-chef-nudge -->`
   - any recent comment contains `<!-- gh-aw-pr-sous-chef-nudge -->` and was posted within the last 30 minutes
2. If skipped, return `skip_reason` only.
3. If not skipped, return:
   - whether branch update should be attempted
   - a single combined nudge comment body (covering unresolved review feedback, branch refresh, and any other forward-progress action) — one comment only, never two; if unresolved PR reviews exist, include an explicit unresolved-reviews list (reviewer + direct link per unresolved review thread)
4. Make at most 8 tool calls total. If 8 calls are insufficient to reach a confident decision, set all fields to `null` and set `skip_reason: "insufficient_context"`.
5. Keep output compact JSON only — a single object, no prose.
6. If you cannot determine a field, set it to `null`.