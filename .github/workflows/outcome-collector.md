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

      # Get recent workflow runs from the last 3 days that had safe outputs
      RUNS=$(gh run list --limit 100 --json databaseId,conclusion,createdAt,workflowName \
        --jq '[.[] | select(.conclusion == "success")] | .[0:50] | .[].databaseId' 2>/dev/null)

      if [ -z "$RUNS" ]; then
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

      for RUN_ID in $RUNS; do
        echo "Checking run $RUN_ID..."
        RESULT=$(gh aw outcomes "$RUN_ID" --json 2>/dev/null) || continue

        ITEMS=$(echo "$RESULT" | jq '.summary.total // 0')
        if [ "$ITEMS" = "0" ] || [ "$ITEMS" = "null" ]; then
          continue
        fi

        CHECKED=$((CHECKED + 1))
        TOTAL=$((TOTAL + ITEMS))
        ACCEPTED=$((ACCEPTED + $(echo "$RESULT" | jq '.summary.accepted // 0')))
        REJECTED=$((REJECTED + $(echo "$RESULT" | jq '.summary.rejected // 0')))
        IGNORED=$((IGNORED + $(echo "$RESULT" | jq '.summary.ignored // 0')))
        PENDING=$((PENDING + $(echo "$RESULT" | jq '.summary.pending // 0')))

        # Save per-run outcome
        WORKFLOW=$(echo "$RESULT" | jq -r '.workflow // "unknown"')
        echo "$RESULT" | jq --arg wf "$WORKFLOW" '. + {workflow: $wf}' \
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
