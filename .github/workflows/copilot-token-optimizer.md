---
name: Copilot Token Optimizer
description: Analyzes the most expensive Copilot workflow identified by the token usage analyzer and creates an optimization issue with specific token-saving recommendations
on:
  workflow_run:
    workflows:
      - "Copilot Token Usage Analyzer"
    types:
      - completed
    branches:
      - main
  workflow_dispatch:
  skip-if-match: 'is:issue is:open in:title "⚡ Copilot Token Optimization:"'

permissions:
  contents: read
  actions: read
  issues: read
  pull-requests: read

engine: copilot
features:
  copilot-requests: true

strict: true

tools:
  bash:
    - "*"
  github:
    toolsets: [default, issues, actions, repos]

safe-outputs:
  create-issue:
    title-prefix: "⚡ Copilot Token Optimization: "
    labels: [automated-analysis, token-optimization, copilot, cost-reduction]
    expires: 7d
    max: 1
    close-older-issues: true
  noop:

network: defaults

timeout-minutes: 30

steps:
  - name: Install gh-aw CLI
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      if gh extension list | grep -q "github/gh-aw"; then
        gh extension upgrade gh-aw || true
      else
        gh extension install github/gh-aw
      fi
      gh aw --version
  - name: Find and download artifacts from the most expensive Copilot workflow
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      set -euo pipefail
      mkdir -p /tmp/token-optimizer

      # Try to use pre-fetched logs from the Token Logs Fetch workflow to avoid redundant API calls
      TODAY=$(date -u +%Y-%m-%d)
      FETCH_RUN_ID=$(gh run list \
        --workflow "token-logs-fetch.lock.yml" \
        --status success \
        --limit 1 \
        --json databaseId \
        --jq '.[0].databaseId' 2>/dev/null || echo "")
      USED_CACHE=false
      if [ -n "$FETCH_RUN_ID" ]; then
        CACHE_TMP="/tmp/token-logs-cache-optimizer"
        mkdir -p "$CACHE_TMP"
        gh run download "$FETCH_RUN_ID" \
          --repo "$GITHUB_REPOSITORY" \
          --name "cache-memory" \
          --dir "$CACHE_TMP" \
          2>/dev/null || true
        CACHE_DATE=$(cat "$CACHE_TMP/token-logs/fetch-date.txt" 2>/dev/null || echo "")
        if [ "$CACHE_DATE" = "$TODAY" ] && [ -s "$CACHE_TMP/token-logs/copilot-runs.json" ]; then
          echo "✅ Using pre-fetched logs from Token Logs Fetch run $FETCH_RUN_ID (date: $CACHE_DATE)"
          cp "$CACHE_TMP/token-logs/copilot-runs.json" /tmp/token-optimizer/copilot-runs.json
          USED_CACHE=true
        else
          echo "ℹ️ No valid cached logs found (cache date: ${CACHE_DATE:-none}, today: $TODAY)"
        fi
      fi

      if [ "$USED_CACHE" != "true" ]; then
        echo "📥 Loading Copilot workflow runs from last 24 hours..."
        gh aw logs \
          --engine copilot \
          --start-date -1d \
          --json \
          -c 300 \
          > /tmp/token-optimizer/copilot-runs-raw.json 2>/dev/null || echo '{"runs":[]}' > /tmp/token-optimizer/copilot-runs-raw.json

        # Extract runs array from the JSON output
        jq '.runs // []' /tmp/token-optimizer/copilot-runs-raw.json > /tmp/token-optimizer/copilot-runs.json 2>/dev/null || echo "[]" > /tmp/token-optimizer/copilot-runs.json
      fi

      RUN_COUNT=$(jq 'length' /tmp/token-optimizer/copilot-runs.json 2>/dev/null || echo 0)
      echo "Found ${RUN_COUNT} Copilot runs"

      if [ "$RUN_COUNT" -eq 0 ]; then
        echo "No Copilot runs found, nothing to optimize"
        exit 0
      fi

      # Find the most expensive workflow (by total tokens across all its runs)
      echo "🔍 Identifying most expensive workflow..."
      jq -r '
        sort_by(.workflow_name) |
        group_by(.workflow_name) |
        map({
          workflow: .[0].workflow_name,
          total_tokens: (map(.token_usage) | add),
          total_cost: 0,
          run_count: length,
          avg_tokens: ((map(.token_usage) | add) / length),
          run_ids: map(.database_id),
          latest_run_id: (sort_by(.created_at) | last | .database_id),
          latest_run_url: (sort_by(.created_at) | last | .url)
        }) |
        sort_by(.total_tokens) | reverse | .[0]
      ' /tmp/token-optimizer/copilot-runs.json > /tmp/token-optimizer/top-workflow.json

      WORKFLOW_NAME=$(jq -r '.workflow' /tmp/token-optimizer/top-workflow.json)
      LATEST_RUN_ID=$(jq -r '.latest_run_id' /tmp/token-optimizer/top-workflow.json)
      echo "Most expensive workflow: $WORKFLOW_NAME (run: $LATEST_RUN_ID)"
      echo "WORKFLOW_NAME=$WORKFLOW_NAME" >> "$GITHUB_ENV"

      # Download the firewall-audit-logs artifact from the latest run of that workflow
      ARTIFACT_DIR="/tmp/token-optimizer/artifacts"
      mkdir -p "$ARTIFACT_DIR"

      echo "📥 Downloading firewall-audit-logs from run $LATEST_RUN_ID..."
      gh run download "$LATEST_RUN_ID" \
        --repo "$GITHUB_REPOSITORY" \
        --name "firewall-audit-logs" \
        --dir "$ARTIFACT_DIR" \
        2>/dev/null || true

      # Also download agent artifacts (contains prompt and tool usage logs)
      echo "📥 Downloading agent artifacts from run $LATEST_RUN_ID..."
      gh run download "$LATEST_RUN_ID" \
        --repo "$GITHUB_REPOSITORY" \
        --name "agent" \
        --dir "$ARTIFACT_DIR/agent" \
        2>/dev/null || true

      # Find token-usage.jsonl
      USAGE_FILE=$(find "$ARTIFACT_DIR" -name "token-usage.jsonl" 2>/dev/null | head -1)
      if [ -n "$USAGE_FILE" ]; then
        echo "Found token-usage.jsonl: $USAGE_FILE"
        cp "$USAGE_FILE" /tmp/token-optimizer/token-usage.jsonl
        wc -l < /tmp/token-optimizer/token-usage.jsonl
      else
        echo "No token-usage.jsonl found in artifacts"
        touch /tmp/token-optimizer/token-usage.jsonl
      fi

      # Find the workflow markdown source
      WORKFLOW_MD_NAME=$(echo "$WORKFLOW_NAME" | tr '[:upper:]' '[:lower:]' | tr ' ' '-')
      WORKFLOW_MD=".github/workflows/${WORKFLOW_MD_NAME}.md"
      if [ -f "$WORKFLOW_MD" ]; then
        echo "Found workflow source: $WORKFLOW_MD"
        cp "$WORKFLOW_MD" /tmp/token-optimizer/workflow-source.md
      else
        echo "Workflow source not found at $WORKFLOW_MD, searching..."
        FOUND_MD=$(find .github/workflows -name "*.md" -exec grep -l "^name: $WORKFLOW_NAME" {} \; 2>/dev/null | head -1 || true)
        if [ -n "$FOUND_MD" ]; then
          echo "Found: $FOUND_MD"
          cp "$FOUND_MD" /tmp/token-optimizer/workflow-source.md
        fi
      fi

      # Extract declared tools from workflow source (if available)
      if [ -f /tmp/token-optimizer/workflow-source.md ]; then
        echo "📋 Extracting declared tools from workflow source..."
        # Extract tools section from frontmatter
        sed -n '/^---$/,/^---$/p' /tmp/token-optimizer/workflow-source.md | \
          grep -A20 "^tools:" | head -30 > /tmp/token-optimizer/declared-tools.txt || true
        cat /tmp/token-optimizer/declared-tools.txt
      fi

imports:
  - shared/reporting.md
---

# Copilot Token Optimizer

You are the Copilot Token Optimizer. Your job is to analyze the most token-expensive Copilot workflow from the past 24 hours and create a targeted optimization issue with specific, actionable recommendations.

## Current Context

- **Repository**: ${{ github.repository }}
- **Analysis Date**: $(date -u +%Y-%m-%d)
- **Target Workflow**: ${{ env.WORKFLOW_NAME }}

## Data Sources

All data is in `/tmp/token-optimizer/`:

- **`copilot-runs.json`** — All Copilot runs from the last 24 hours
- **`top-workflow.json`** — Statistics for the most expensive workflow
- **`token-usage.jsonl`** — Per-request token records from the target workflow's last run (may be empty if artifact unavailable)
- **`workflow-source.md`** — The workflow's markdown source (may not exist if not found)
- **`declared-tools.txt`** — Tools declared in the workflow frontmatter
- **`artifacts/agent/`** — Agent artifacts (prompt, MCP logs, agent-stdio.log) from the last run

## Analysis Process

### Phase 1: Load Workflow Statistics

Read the top workflow data:

```bash
cat /tmp/token-optimizer/top-workflow.json
```

Note: `avg_tokens` is the key metric. Very high `avg_tokens` means each run is expensive.

### Phase 2: Analyze Token Usage Patterns

If `token-usage.jsonl` is available (non-empty):

```bash
# Per-model breakdown
awk '{
  if (match($0, /"model" *: *"([^"]*)"/, m)) model = m[1]
  else model = "unknown"
  if (match($0, /"input_tokens" *: *([0-9]+)/, m)) input = m[1]+0; else input = 0
  if (match($0, /"output_tokens" *: *([0-9]+)/, m)) output = m[1]+0; else output = 0
  if (match($0, /"cache_read_tokens" *: *([0-9]+)/, m)) cr = m[1]+0; else cr = 0
  models[model] = 1
  mi[model] += input; mo[model] += output; mcr[model] += cr; mc[model] += 1
}
END {
  for (m in models)
    printf "Model: %s, Input: %d, Output: %d, CacheRead: %d, Requests: %d\n",
      m, mi[m], mo[m], mcr[m], mc[m]
}' /tmp/token-optimizer/token-usage.jsonl
```

Look for:
- **High input tokens per request** → large context window being used
- **Low cache hit rate** → context is not being reused across turns
- **High request count** → many back-and-forth turns; consider `max-turns` limit

### Phase 3: Analyze Declared vs. Used Tools

Review the workflow source to understand which tools are declared:

```bash
cat /tmp/token-optimizer/workflow-source.md
```

Then check agent logs for which tools were actually called:

```bash
# Find MCP tool calls in agent logs
find /tmp/token-optimizer/artifacts/agent -name "*.log" 2>/dev/null | xargs grep -l "mcp__" 2>/dev/null | head -3
grep -oh 'mcp__[a-z_]*__[a-z_]*' /tmp/token-optimizer/artifacts/agent/*.log 2>/dev/null | sort | uniq -c | sort -rn | head -30
```

Compare:
- **Declared tools** (from frontmatter) vs **tools actually invoked** (from agent logs)
- Tools declared but never called are injected into the context window every turn, wasting tokens
- Each unused tool description typically costs ~500 tokens/turn

### Phase 4: Identify Specific Optimization Opportunities

Based on the data, identify opportunities from these categories:

#### A. Unused Tool Exclusions
Tools declared in frontmatter but never invoked during the run. Removing these saves tokens every turn.

Example finding: "GitHub `toolsets: [default, actions, repos]` declared but only `issues` MCP tools called — exclude `actions` and `repos` toolsets to save ~500 tokens/turn"

#### B. Context Window Reduction
If input tokens per request are very high (> 50k), the context window may be bloated by:
- Large file reads (suggest chunking or streaming)
- Verbose MCP responses (suggest pagination with smaller `perPage`)
- Long conversation history (suggest `max-turns` reduction)

#### C. Turn Reduction
If request count per run is high (> 10 turns), consider:
- More specific prompt instructions to reduce back-and-forth
- Pre-computing data in `steps:` before the agent runs
- Using `strict: true` to fail fast on unexpected tool calls

#### D. Prompt Optimization
- System prompt restructuring for better cache hit rate
- Removing verbose instructions that are rarely needed
- Using shared imports for common instructions instead of duplicating text

### Phase 5: Create Optimization Issue

Create an issue with the title: `[workflow-name] (avg [N]k tokens/run)` — the prefix `⚡ Copilot Token Optimization:` is automatically added.

#### Issue Body Structure

```markdown
### Target Workflow: [workflow-name]

**Why this workflow?** Highest total token consumption in the last 24 hours.

| Metric | Value |
|--------|-------|
| Runs (24h) | [N] |
| Avg tokens/run | [N]k |
| Total est. cost (24h) | $[X] |
| Avg turns/run | [N] |

### Token Usage Breakdown

_(from token-usage.jsonl — if available)_

| Model | Input | Output | Cache Read | Cache Hit % |
|-------|-------|--------|------------|-------------|
| [model] | [n] | [n] | [n] | [pct]% |

### Optimization Recommendations

#### 1. [Highest Impact Recommendation]

**Potential savings**: ~[N]k tokens/run (~$[X]/run × [N] runs/day = ~$[X]/day)

**Current state**: [What the workflow currently does that's expensive]

**Recommended change**:
```diff
- [current config/prompt line]
+ [optimized config/prompt line]
```

**Why this helps**: [Explanation]

#### 2. [Second Recommendation]

...

<details>
<summary><b>Tool Usage Analysis</b></summary>

**Declared tools** (from frontmatter):
[list from declared-tools.txt]

**Tools actually invoked** (from agent logs):
[list from mcp call analysis]

**Unused tools** (candidates for removal):
- `[toolset/tool]` — never called, saves ~[N] tokens/turn if removed

</details>

<details>
<summary><b>Raw Metrics</b></summary>

[token-usage.jsonl summary]

</details>

### Implementation Checklist

- [ ] Apply recommended changes to `.github/workflows/[workflow-name].md`
- [ ] Run `make recompile` to regenerate the lock file
- [ ] Trigger a manual run via `workflow_dispatch` to verify
- [ ] Compare token usage in next analyzer report

### References

- [Last run of [workflow-name]](LATEST_RUN_URL)
- Analysis triggered by: [§RUN_ID](RUN_URL)
```

## Important Guidelines

- **Be specific**: Name exact tools, exact token counts, exact cost estimates.
- **Prioritize by impact**: List the highest token-saving opportunity first.
- **Be conservative**: Only recommend removing tools you're confident are unused (verify from logs).
- **If no data**: If both `token-usage.jsonl` and agent logs are unavailable, base recommendations on workflow source analysis only, and note the limitation.
- **`noop` when appropriate**: If the workflow is already well-optimized (< 10k tokens/run average) or if you cannot find meaningful optimization opportunities, call `noop` instead of creating a low-value issue.

**Important**: You MUST call a safe-output tool (`create-issue` or `noop`) at the end of your analysis. Failing to call any safe-output tool is the most common cause of workflow failures.

```json
{"noop": {"message": "No action needed: [brief explanation]"}}
```
