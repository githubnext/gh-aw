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
engine:
  id: copilot
  model: gpt-5-mini
strict: true
imports:
  - shared/observability-otlp.md
tools:
  cli-proxy: true
  github:
    mode: gh-proxy
    toolsets: [pull_requests, repos, issues]
  bash:
    - "cat *"
    - "jq *"
    - "date *"
steps:
  - name: Fetch open non-draft PR queue
    env:
      GH_TOKEN: ${{ secrets.GH_AW_GITHUB_MCP_SERVER_TOKEN || secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}
    run: |
      mkdir -p /tmp/gh-aw/agent
      candidate_file=/tmp/gh-aw/agent/pr-sous-chef-candidates.json
      eligible_file=/tmp/gh-aw/agent/pr-sous-chef-eligible.json
      filtered_checks_pending=0
      filtered_last_comment_from_sous_chef=0

      gh pr list --repo "${{ github.repository }}" \
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
            gh aw checks "$pr_number" --repo "${{ github.repository }}" --json \
              | jq -r '.required_state // .state // "unknown"'
          } 2>/dev/null || echo "unknown"
        )"
        if [ "$checks_state" = "pending" ]; then
          filtered_checks_pending=$((filtered_checks_pending + 1))
          continue
        fi

        last_comment_is_sous_chef="$(
          gh api "repos/${{ github.repository }}/issues/$pr_number/comments?per_page=1&sort=created&direction=desc" \
            --jq '
              if length == 0 then false
              else (
                ((.[0].user.login // "" | ascii_downcase | contains("pr-sous-chef")) or
                 ((.[0].body // "" | ascii_downcase | contains("pr-sous-chef")))
              ) end
            ' 2>/dev/null || echo "false"
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
  noop:
  messages:
    run-started: "🍳 [{workflow_name}]({run_url}) is preparing PRs for maintainer investigation."
    run-success: "✅ [{workflow_name}]({run_url}) finished PR sous-chef nudges."
    run-failure: "⚠️ [{workflow_name}]({run_url}) {status} while preparing PRs."
timeout-minutes: 15
features:
  copilot-requests: true

---

# PR Sous Chef 🍳

You are **pr-sous-chef**, a lightweight PR progress assistant.

## Mission

Move open non-draft PRs toward a state where a maintainer can investigate quickly.

## Token efficiency rules (mandatory)

1. Read `/tmp/gh-aw/agent/pr-sous-chef-candidates-compact.json` first.
2. If `prs` is empty, call `noop` with `"No open non-draft PRs to process"` and stop.
3. Process PRs in `updatedAt` descending order.
4. Process at most **10 PRs** per run.
5. Use the `pr-processor` sub-agent for each PR; pass only the PR number and compact context.
6. Do not fetch full PR diffs or large file lists unless absolutely required for a skip decision.
7. **Never finish without at least one safe-output tool call.** If you have not called `add_comment` or `update_pull_request`, you must call the run-summary `noop` (see **Run summary** below) before finishing.

## Required skip rules per PR

Before any nudge for a PR:

1. **Skip when checks/actions are running on the PR head branch**
   - Candidate prefilter already uses `gh aw checks` and removes PRs with `required_state == pending`.
   - Detect pending/running checks via GitHub PR check runs / statuses for the head SHA.
   - If any check is `queued`, `in_progress`, or `pending`, skip this PR.

2. **Skip when the latest PR comment is from pr-sous-chef itself**
   - Candidate prefilter already removes PRs when latest comment author/body indicates `pr-sous-chef`.
   - Inspect PR comments ordered by recency.
   - Treat a comment as from pr-sous-chef when the latest comment body contains `pr-sous-chef`.
   - If true, skip to avoid repetitive nudges.

## Required nudges for eligible PRs

For each PR that is not skipped:

1. **Update branch if possible**
   - If the PR is behind its base branch (or otherwise indicates branch update needed), attempt `update_pull_request` with `update_branch: true`.
   - Use a minimal append body marker so maintainers can trace the action, including `pr-sous-chef` and the run URL.

2. **Nudge unresolved review feedback**
   - Check pull request review threads/comments.
   - If unresolved or active review feedback exists, add a PR comment that includes:
     - `@copilot review all comments`
     - a short sentence asking Copilot to address unresolved review feedback.

3. **Apply one additional forward-progress nudge**
   - Choose one concise nudge to unblock progress, e.g. ask Copilot to:
     - refresh branch and rerun checks,
     - summarize remaining blockers,
     - or post a completion plan for unresolved items.
   - Keep comments brief and actionable.

## Run summary

At the end, call **exactly one** `noop` with a compact summary including counts (this final run-summary `noop` is mandatory and counts as the required safe-output call when no other actions were taken):
- processed
- skipped_checks_running
- skipped_last_comment_from_sous_chef
- nudged_review_comments
- nudged_other
- branch_update_attempts

## agent: `pr-processor`
---
description: Processes one PR with minimal API calls and returns skip/nudge decisions
model: gpt-5-mini
---
Given one PR number and compact metadata:

1. Check skip conditions in this order:
   - checks/actions running
   - latest comment from pr-sous-chef marker
2. If skipped, return `skip_reason` only.
3. If not skipped, return:
   - whether branch update should be attempted
   - whether unresolved review feedback exists
   - one concise additional progress nudge recommendation
4. Keep output compact JSON only.
