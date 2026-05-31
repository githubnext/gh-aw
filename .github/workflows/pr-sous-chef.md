---
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
      filtered_checks_pending=0
      filtered_last_comment_from_sous_chef=0

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

        last_comment_is_sous_chef="$(
          gh api "repos/$EXPR_GITHUB_REPOSITORY/issues/$pr_number/comments?per_page=1&sort=created&direction=desc" \
            --jq '
              if length == 0 then false
              else (
                ((.[0].body // "" | contains($marker)))
               ) end
            ' --arg marker "$sous_chef_nudge_marker" 2>/dev/null || echo "false"
        )"
        if [ "$last_comment_is_sous_chef" = "true" ]; then
          filtered_last_comment_from_sous_chef=$((filtered_last_comment_from_sous_chef + 1))
          continue
        fi

        jq --argjson pr "$pr" '. + [$pr]' "$eligible_file" > "${eligible_file}.tmp" && mv "${eligible_file}.tmp" "$eligible_file"
      # Process substitution keeps the loop in the current shell so counters persist.
      done < <(jq -c '.[]' "$candidate_file")

      jq --argjson filtered_checks_pending "$filtered_checks_pending" \
         --argjson filtered_last_comment_from_sous_chef "$filtered_last_comment_from_sous_chef" '{
        fetched: (length),
        generated_at: (now | todate),
        filtered_checks_pending: $filtered_checks_pending,
        filtered_last_comment_from_sous_chef: $filtered_last_comment_from_sous_chef,
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
    uses: actions/setup-go@v6.4.0
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

## Required skip rules per PR

Before any nudge for a PR:

1. **Skip when checks/actions are running on the PR head branch**
   - Candidate prefilter already uses `gh aw checks` and removes PRs with `required_state == pending`.
   - Detect pending/running checks via GitHub PR check runs / statuses for the head SHA.
   - If any check is `queued`, `in_progress`, or `pending`, skip this PR.

2. **Skip when the latest PR comment is from pr-sous-chef itself**
   - Candidate prefilter already removes PRs when latest comment body includes the hidden marker `<!-- gh-aw-pr-sous-chef-nudge -->`.
   - Inspect PR comments ordered by recency.
   - Treat a comment as from pr-sous-chef only when the latest comment body contains exactly `<!-- gh-aw-pr-sous-chef-nudge -->`.
   - If true, skip to avoid repetitive nudges.

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

2. **Nudge unresolved review feedback**
   - Check pull request review threads/comments.
   - If unresolved or active review feedback exists, add a PR comment that includes:
     - `<!-- gh-aw-pr-sous-chef-nudge -->` as a hidden marker line.
     - @copilot review all comments
     - a short sentence asking Copilot to address unresolved review feedback.
   - Every `add_comment` must include `pr_number` set to the current PR's numeric `number` from the loop item.
   - Never emit `add_comment` without a numeric target field (`pr_number`/`pull_request_number`/`issue_number`/`item_number`) when `target: "*"` is configured.
   - Example:
     ```json
     {"type":"add_comment","pr_number":12345,"body":"<!-- gh-aw-pr-sous-chef-nudge -->\n@copilot review all comments and address unresolved review feedback."}
     ```

3. **Apply one additional forward-progress nudge**
   - Choose one concise nudge to unblock progress, e.g. ask Copilot to:
     - refresh branch and rerun checks,
     - summarize remaining blockers,
     - or post a completion plan for unresolved items.
   - Include `<!-- gh-aw-pr-sous-chef-nudge -->` in the comment body.
   - Keep comments brief and actionable.

## Run summary

At the end, call **exactly one** `noop` with a compact summary including counts (this final run-summary `noop` is mandatory and counts as the required safe-output call when no other actions were taken):
- processed
- skipped_checks_running
- skipped_last_comment_from_sous_chef
- nudged_review_comments
- nudged_other
- branch_update_attempts
- formatter_pushes (number of PRs that had formatting fixes committed and pushed)

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
model: gpt-5-mini
---
Given one PR number and compact metadata:

1. Check skip conditions in this order:
   - checks/actions running
   - latest comment contains `<!-- gh-aw-pr-sous-chef-nudge -->`
2. If skipped, return `skip_reason` only.
3. If not skipped, return:
   - whether branch update should be attempted
   - whether unresolved review feedback exists
   - one concise additional progress nudge recommendation
4. Make at most 8 tool calls total. If 8 calls are insufficient to reach a confident decision, set all fields to `null` and set `skip_reason: "insufficient_context"`.
5. Keep output compact JSON only — a single object, no prose.
6. If you cannot determine a field, set it to `null`.