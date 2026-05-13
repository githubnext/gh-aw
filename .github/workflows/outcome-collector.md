---
name: Outcome Collector
description: Daily evaluation of safe output outcomes to measure workflow value and acceptance rates
on:
  schedule:
    - cron: daily around 06:00
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  pull-requests: read
  actions: read
  discussions: read
tracker-id: outcome-collector
engine:
  id: copilot
  model: claude-haiku-4.5
  bare: true
strict: true
timeout-minutes: 20
network:
  allowed:
    - defaults
    - github
tools:
  bash: true
  github:
    mode: gh-proxy
    toolsets: [default]
safe-outputs:
  create-issue:
    title-prefix: "[Outcome Report]"
    labels: [automation, observability, outcomes]
    close-older-issues: true
    group-by-day: true
    expires: 7d
  noop:
  messages:
    footer: "> 📊 *Measured by [{workflow_name}]({run_url})*{effective_tokens_suffix}"
    run-started: "📊 [{workflow_name}]({run_url}) is evaluating safe output outcomes..."
    run-success: "📊 [{workflow_name}]({run_url}) outcome evaluation complete!"
    run-failure: "📊 [{workflow_name}]({run_url}) {status}"
imports:
  - shared/observability-otlp.md
pre-agent-steps:
  - name: Evaluate outcomes for recent runs
    env:
      GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      echo "Evaluating safe output outcomes for recent workflow runs..."

      REPO="${GITHUB_REPOSITORY}"

      # Get recent successful workflow runs
      RUNS=$(gh run list --repo "$REPO" --limit 100 --json databaseId,conclusion,workflowName \
        --jq '[.[] | select(.conclusion == "success")] | .[0:50]' 2>/dev/null)

      if [ -z "$RUNS" ] || [ "$RUNS" = "[]" ] || [ "$RUNS" = "null" ]; then
        echo "No recent successful runs found"
        echo '{"runs_checked": 0, "total_outcomes": 0}' > /tmp/gh-aw/outcome-summary.json
        exit 0
      fi

      mkdir -p /tmp/gh-aw/outcomes

      CHECKED=0
      ACCEPTED=0
      REJECTED=0
      IGNORED=0
      PENDING=0
      TOTAL=0
      EVAL_JSONL="/tmp/gh-aw/outcome-evaluations.jsonl"
      > "$EVAL_JSONL"

      for RUN_ID in $(echo "$RUNS" | jq -r '.[].databaseId'); do
        # Try to download safe-outputs-items artifact (skip runs without it)
        ITEM_DIR="/tmp/gh-aw/outcomes/run-${RUN_ID}"
        gh run download "$RUN_ID" --repo "$REPO" --name safe-outputs-items --dir "$ITEM_DIR" 2>/dev/null || continue

        MANIFEST="$ITEM_DIR/safe-output-items.jsonl"
        if [ ! -f "$MANIFEST" ]; then
          continue
        fi

        # Count actionable items (exclude noop, missing_tool, missing_data, report_incomplete, empty objects)
        ITEMS=$(jq -r 'select(.type != null and .type != "" and .type != "noop" and .type != "missing_tool" and .type != "missing_data" and .type != "report_incomplete") | .type' "$MANIFEST" 2>/dev/null | wc -l | tr -d ' ')

        if [ "$ITEMS" = "0" ]; then
          continue
        fi

        WF=$(echo "$RUNS" | jq -r ".[] | select(.databaseId == $RUN_ID) | .workflowName")
        echo "Run $RUN_ID ($WF): $ITEMS item(s)"

        CHECKED=$((CHECKED + 1))
        TOTAL=$((TOTAL + ITEMS))

        # Basic outcome evaluation per item using GitHub API
        while IFS= read -r line; do
          TYPE=$(echo "$line" | jq -r '.type // empty')
          case "$TYPE" in
            ""|noop|missing_tool|missing_data|report_incomplete) continue ;;
          esac

          URL=$(echo "$line" | jq -r '.url // empty')
          ITEM_REPO=$(echo "$line" | jq -r '.repo // empty')
          TIMESTAMP=$(echo "$line" | jq -r '.timestamp // empty')
          [ -z "$ITEM_REPO" ] && ITEM_REPO="$REPO"

          RESULT="pending"
          DETAIL=""

          if [ -z "$URL" ]; then
            RESULT="pending"
            DETAIL="no url"
            PENDING=$((PENDING + 1))
          elif echo "$URL" | grep -qE '/issues/[0-9]+|/issuecomment-'; then
            NUM=$(echo "$URL" | grep -oE '/(issues|pull)/[0-9]+' | grep -oE '[0-9]+' | head -1)
            if [ -n "$NUM" ]; then
              STATE=$(gh api "repos/$ITEM_REPO/issues/$NUM" --jq '.state' 2>/dev/null || echo "")
              if [ "$STATE" = "open" ] || [ "$STATE" = "closed" ]; then
                RESULT="accepted"
                DETAIL="$STATE"
                ACCEPTED=$((ACCEPTED + 1))
              else
                RESULT="pending"
                DETAIL="api error"
                PENDING=$((PENDING + 1))
              fi
            else
              RESULT="pending"
              DETAIL="no number"
              PENDING=$((PENDING + 1))
            fi
          elif echo "$URL" | grep -qE '/pull/[0-9]+'; then
            NUM=$(echo "$URL" | grep -oE '/pull/[0-9]+' | grep -oE '[0-9]+')
            if [ -n "$NUM" ]; then
              MERGED=$(gh api "repos/$ITEM_REPO/pulls/$NUM" --jq '.merged' 2>/dev/null || echo "")
              STATE=$(gh api "repos/$ITEM_REPO/pulls/$NUM" --jq '.state' 2>/dev/null || echo "")
              if [ "$MERGED" = "true" ]; then
                RESULT="accepted"
                DETAIL="merged"
                ACCEPTED=$((ACCEPTED + 1))
              elif [ "$STATE" = "closed" ]; then
                RESULT="rejected"
                DETAIL="closed"
                REJECTED=$((REJECTED + 1))
              elif [ "$STATE" = "open" ]; then
                RESULT="pending"
                DETAIL="open"
                PENDING=$((PENDING + 1))
              else
                RESULT="pending"
                DETAIL="api error"
                PENDING=$((PENDING + 1))
              fi
            else
              RESULT="pending"
              DETAIL="no number"
              PENDING=$((PENDING + 1))
            fi
          else
            # Comments, labels, etc. — if URL exists, the item was created
            RESULT="accepted"
            DETAIL="object exists"
            ACCEPTED=$((ACCEPTED + 1))
          fi

          # Write per-item evaluation for OTEL export
          jq -n -c \
            --arg type "$TYPE" \
            --arg url "$URL" \
            --arg repo "$ITEM_REPO" \
            --arg result "$RESULT" \
            --arg detail "$DETAIL" \
            --arg workflow "$WF" \
            --argjson run_id "$RUN_ID" \
            --arg timestamp "$TIMESTAMP" \
            '{type: $type, url: $url, repo: $repo, result: $result, detail: $detail, workflow: $workflow, run_id: $run_id, timestamp: $timestamp}' \
            >> "$EVAL_JSONL"
        done < "$MANIFEST"

        # Save per-run data
        jq -n --arg wf "$WF" --argjson items "$ITEMS" --argjson run_id "$RUN_ID" \
          '{workflow: $wf, run_id: $run_id, items: $items}' \
          > "/tmp/gh-aw/outcomes/run-${RUN_ID}.json"
      done

      # Compute fleet summary
      RESOLVED=$((ACCEPTED + REJECTED))
      if [ "$RESOLVED" -gt 0 ]; then
        ACCEPTANCE_RATE=$(echo "scale=4; $ACCEPTED / $RESOLVED" | bc)
      else
        ACCEPTANCE_RATE="0"
      fi
      if [ "$TOTAL" -gt 0 ]; then
        WASTE_RATE=$(echo "scale=4; $REJECTED / $TOTAL" | bc)
      else
        WASTE_RATE="0"
      fi

      jq -n \
        --argjson checked "$CHECKED" \
        --argjson total "$TOTAL" \
        --argjson accepted "$ACCEPTED" \
        --argjson rejected "$REJECTED" \
        --argjson ignored "$IGNORED" \
        --argjson pending "$PENDING" \
        --arg acceptance_rate "$ACCEPTANCE_RATE" \
        --arg waste_rate "$WASTE_RATE" \
        '{
          runs_checked: $checked,
          total_outcomes: $total,
          accepted: $accepted,
          rejected: $rejected,
          ignored: $ignored,
          pending: $pending,
          acceptance_rate: ($acceptance_rate | tonumber),
          waste_rate: ($waste_rate | tonumber),
          date: (now | strftime("%Y-%m-%d"))
        }' > /tmp/gh-aw/outcome-summary.json

      echo "✓ Checked $CHECKED runs, $TOTAL outcomes"
      echo "  Accepted: $ACCEPTED, Rejected: $REJECTED, Ignored: $IGNORED, Pending: $PENDING"
      echo "  Acceptance rate: $ACCEPTANCE_RATE"
      cat /tmp/gh-aw/outcome-summary.json
  - name: Export outcome telemetry
    run: |
      if [ -f /tmp/gh-aw/outcome-evaluations.jsonl ] && [ -s /tmp/gh-aw/outcome-evaluations.jsonl ]; then
        node "${RUNNER_TEMP}/gh-aw/actions/emit_outcome_spans.cjs"
      else
        echo "No outcome evaluations to export"
      fi
---

# Outcome Collector

You are the Outcome Collector. Your job is to create a concise daily report of safe output outcomes.

## Input

The pre-agent step has already evaluated outcomes for recent workflow runs. Results are in:

- `/tmp/gh-aw/outcome-summary.json` — fleet-wide summary
- `/tmp/gh-aw/outcomes/run-*.json` — per-run outcome details

## Task

1. Read `/tmp/gh-aw/outcome-summary.json`
2. If `total_outcomes` is 0, call `noop` with "No safe output outcomes to report today"
3. Otherwise, create a report issue with the summary

## Report Format

Create an issue with this structure:

```markdown
## Safe Output Outcomes — {date}

### Fleet Summary

| Metric | Value |
|--------|-------|
| Runs checked | {runs_checked} |
| Total outcomes | {total_outcomes} |
| Accepted | {accepted} |
| Rejected | {rejected} |
| Ignored | {ignored} |
| Pending | {pending} |
| **Acceptance rate** | **{acceptance_rate}%** |
| Waste rate | {waste_rate}% |

### Per-Workflow Breakdown

For each workflow with outcomes, show:
- Workflow name
- Outcomes: accepted / rejected / ignored
- Acceptance rate

### Key Observations

- Which workflows have the highest acceptance rate?
- Which workflows have the highest waste rate?
- Any workflows with all outcomes ignored (noise signal)?
```

## Guidelines

- Keep the report factual — numbers only, no speculation
- Do not re-evaluate outcomes — use the pre-computed data
- If no outcomes exist, use `noop`
- Stop immediately after creating the issue
