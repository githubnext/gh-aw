---
name: Claude Token Optimizer
description: Analyzes the most expensive Claude workflow identified by the token usage analyzer and creates an optimization issue with Anthropic-specific token-saving recommendations including cache efficiency improvements
on:
  workflow_run:
    workflows:
      - "Claude Token Usage Analyzer"
    types:
      - completed
    branches:
      - main
  workflow_dispatch:
  skip-if-match: 'is:issue is:open in:title "⚡ Claude Token Optimization:"'

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
    title-prefix: "⚡ Claude Token Optimization: "
    labels: [automated-analysis, token-optimization, claude, cost-reduction]
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
  - name: Find and download artifacts from the most expensive Claude workflow
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      set -euo pipefail
      mkdir -p /tmp/token-optimizer-claude

      echo "📥 Loading Claude workflow runs from last 24 hours..."
      gh aw logs \
        --engine claude \
        --start-date -1d \
        --json \
        -c 300 \
        > /tmp/token-optimizer-claude/claude-runs.json 2>/dev/null || echo "[]" > /tmp/token-optimizer-claude/claude-runs.json

      RUN_COUNT=$(jq '. | length' /tmp/token-optimizer-claude/claude-runs.json 2>/dev/null || echo 0)
      echo "Found ${RUN_COUNT} Claude runs"

      if [ "$RUN_COUNT" -eq 0 ]; then
        echo "No Claude runs found, nothing to optimize"
        exit 0
      fi

      # Find the most expensive workflow (by total tokens across all its runs)
      echo "🔍 Identifying most expensive workflow..."
      jq -r '
        group_by(.workflowName) |
        map({
          workflow: .[0].workflowName,
          total_tokens: (map(.tokenUsage) | add),
          total_cost: (map(.estimatedCost) | add),
          run_count: length,
          avg_tokens: ((map(.tokenUsage) | add) / length),
          run_ids: map(.databaseId),
          latest_run_id: (sort_by(.createdAt) | last | .databaseId),
          latest_run_url: (sort_by(.createdAt) | last | .url)
        }) |
        sort_by(.total_tokens) | reverse | .[0]
      ' /tmp/token-optimizer-claude/claude-runs.json > /tmp/token-optimizer-claude/top-workflow.json

      WORKFLOW_NAME=$(jq -r '.workflow' /tmp/token-optimizer-claude/top-workflow.json)
      LATEST_RUN_ID=$(jq -r '.latest_run_id' /tmp/token-optimizer-claude/top-workflow.json)
      echo "Most expensive workflow: $WORKFLOW_NAME (run: $LATEST_RUN_ID)"
      echo "WORKFLOW_NAME=$WORKFLOW_NAME" >> "$GITHUB_ENV"

      # Download the firewall-audit-logs artifact from the latest run
      ARTIFACT_DIR="/tmp/token-optimizer-claude/artifacts"
      mkdir -p "$ARTIFACT_DIR"

      echo "📥 Downloading firewall-audit-logs from run $LATEST_RUN_ID..."
      gh run download "$LATEST_RUN_ID" \
        --repo "$GITHUB_REPOSITORY" \
        --name "firewall-audit-logs" \
        --dir "$ARTIFACT_DIR" \
        2>/dev/null || true

      # Also download agent artifacts
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
        cp "$USAGE_FILE" /tmp/token-optimizer-claude/token-usage.jsonl
        echo "Records: $(wc -l < /tmp/token-optimizer-claude/token-usage.jsonl)"

        # Pre-compute Anthropic-specific metrics
        echo "📊 Computing Anthropic cache efficiency metrics..."
        awk '
        BEGIN { ti=0; to=0; cr=0; cw=0; tr=0 }
        {
          if (match($0, /"input_tokens" *: *([0-9]+)/, m)) ti += m[1]+0
          if (match($0, /"output_tokens" *: *([0-9]+)/, m)) to += m[1]+0
          if (match($0, /"cache_read_tokens" *: *([0-9]+)/, m)) cr += m[1]+0
          if (match($0, /"cache_write_tokens" *: *([0-9]+)/, m)) cw += m[1]+0
          tr += 1
        }
        END {
          total = ti + to + cr + cw
          if (tr == 0) exit
          printf "Requests: %d\n", tr
          printf "Input tokens: %d\n", ti
          printf "Output tokens: %d\n", to
          printf "Cache read tokens: %d\n", cr
          printf "Cache write tokens: %d\n", cw
          printf "Total tokens: %d\n", total
          if (ti + cr > 0) printf "Cache hit rate: %.1f%%\n", (cr / (ti + cr)) * 100
          if (ti + cw > 0) printf "Cache write rate: %.1f%%\n", (cw / (ti + cw)) * 100
          if (cw > 0) printf "Cache read/write ratio: %.2f\n", (cr / cw)
        }' /tmp/token-optimizer-claude/token-usage.jsonl > /tmp/token-optimizer-claude/cache-metrics.txt
        cat /tmp/token-optimizer-claude/cache-metrics.txt
      else
        echo "No token-usage.jsonl found in artifacts"
        touch /tmp/token-optimizer-claude/token-usage.jsonl
        touch /tmp/token-optimizer-claude/cache-metrics.txt
      fi

      # Find the workflow markdown source
      WORKFLOW_MD_NAME=$(echo "$WORKFLOW_NAME" | tr '[:upper:]' '[:lower:]' | tr ' ' '-')
      WORKFLOW_MD=".github/workflows/${WORKFLOW_MD_NAME}.md"
      if [ -f "$WORKFLOW_MD" ]; then
        echo "Found workflow source: $WORKFLOW_MD"
        cp "$WORKFLOW_MD" /tmp/token-optimizer-claude/workflow-source.md
      else
        find .github/workflows -name "*.md" -exec grep -l "^name: $WORKFLOW_NAME" {} \; 2>/dev/null | head -1 | while read -r f; do
          echo "Found: $f"
          cp "$f" /tmp/token-optimizer-claude/workflow-source.md
        done
      fi

      # Extract declared tools from workflow source
      if [ -f /tmp/token-optimizer-claude/workflow-source.md ]; then
        sed -n '/^---$/,/^---$/p' /tmp/token-optimizer-claude/workflow-source.md | \
          grep -A20 "^tools:" | head -30 > /tmp/token-optimizer-claude/declared-tools.txt || true
      fi

imports:
  - shared/reporting.md
---

# Claude Token Optimizer

You are the Claude Token Optimizer. Your job is to analyze the most token-expensive Claude workflow from the past 24 hours and create a targeted optimization issue with specific, actionable recommendations — with special focus on Anthropic's unique caching economics.

## Current Context

- **Repository**: ${{ github.repository }}
- **Analysis Date**: $(date -u +%Y-%m-%d)
- **Target Workflow**: ${{ env.WORKFLOW_NAME }}

## Data Sources

All data is in `/tmp/token-optimizer-claude/`:

- **`claude-runs.json`** — All Claude runs from the last 24 hours
- **`top-workflow.json`** — Statistics for the most expensive workflow
- **`token-usage.jsonl`** — Per-request token records from the target workflow's last run (may be empty)
- **`cache-metrics.txt`** — Pre-computed Anthropic cache efficiency summary
- **`workflow-source.md`** — The workflow's markdown source (may not exist if not found)
- **`declared-tools.txt`** — Tools declared in the workflow frontmatter
- **`artifacts/agent/`** — Agent artifacts (prompt, MCP logs) from the last run

## Anthropic Pricing Context

Before analyzing, understand Anthropic's pricing model:

| Token Type | Cost Relative to Input |
|------------|----------------------|
| Input tokens | 1× (baseline) |
| Output tokens | ~3-5× more expensive than input |
| Cache write tokens | ~1.25× (investment for future savings) |
| Cache read tokens | ~0.1× (90% discount vs input) |

**Cache ROI formula**: Cache write breaks even after `1.25 / (1 - 0.1)` ≈ **1.4 reads**.

**Cache TTL**: Default 5 minutes; extended TTL requires explicit configuration. If turns are spread > 5 min apart, cache may not be reused.

## Analysis Process

### Phase 1: Load Workflow Statistics

```bash
cat /tmp/token-optimizer-claude/top-workflow.json
cat /tmp/token-optimizer-claude/cache-metrics.txt
```

### Phase 2: Analyze Anthropic Cache Efficiency

The `cache-metrics.txt` file contains:
- **Cache hit rate**: `cache_read / (input + cache_read)` — higher is better
- **Cache write rate**: `cache_write / (input + cache_write)` — shows how much is being cached
- **Cache read/write ratio**: `cache_read / cache_write` — must be > 1.4 for cache to save money

**Interpret the metrics**:

| Cache Hit Rate | Assessment | Action |
|----------------|-----------|--------|
| > 60% | Excellent — caching is very effective | No action needed |
| 30-60% | Good | Minor optimizations possible |
| 10-30% | Poor | Restructure prompt to improve caching |
| < 10% | Very poor | Cache may not be working; check min token threshold (1024) |

| Cache Read/Write Ratio | Assessment | Action |
|------------------------|-----------|--------|
| > 5× | Excellent — writes amortized well | No action needed |
| 1.4-5× | Good — cache is profitable | Minor tuning possible |
| < 1.4× | Poor — not saving money | Increase cache reuse or disable writes |
| 0 (no reads) | Cache writes are pure waste | Investigate why cache isn't being read |

### Phase 3: Analyze Tool Usage

```bash
cat /tmp/token-optimizer-claude/declared-tools.txt

# Check actual tool invocations from agent logs
find /tmp/token-optimizer-claude/artifacts/agent -name "*.log" 2>/dev/null | \
  xargs grep -oh 'mcp__[a-z_]*__[a-z_]*' 2>/dev/null | sort | uniq -c | sort -rn | head -30
```

Compare declared vs. used tools. Each unused toolset injects tool descriptions into every turn of the context window (~500 tokens per toolset).

### Phase 4: Review Workflow Source for Prompt Optimization

```bash
cat /tmp/token-optimizer-claude/workflow-source.md
```

Look for:
1. **Static content repeated every turn** — good candidate for caching (must be at start, ≥ 1024 tokens)
2. **Verbose instructions** that could be shortened
3. **Missing `max-turns`** limit — unbounded conversations accumulate context
4. **High output expectations** — if the prompt asks for very long responses, output tokens are expensive for Claude

### Phase 5: Identify Specific Optimization Opportunities

Prioritize by estimated savings:

#### A. Cache Efficiency Improvements (often highest impact for Claude)
- **Low hit rate + system prompt ≥ 1024 tokens**: Already has potential — check if tool descriptions are before or after the cache boundary
- **Low hit rate + system prompt < 1024 tokens**: Add content to reach 1024 token minimum for cache activation
- **Zero cache reads with writes**: Likely the workflow is not reusing context across turns; check if turns are too far apart

#### B. Output Token Reduction (second highest impact)
- If output tokens > 30% of total: add instructions like "Be concise, use bullet points, avoid repetition"
- Look for places in the prompt where Claude is asked to produce long reports that could be structured more efficiently

#### C. Unused Tool Exclusions
- Identify tools declared but never called
- Each unused tool description in Anthropic's context costs ~500 input tokens per turn

#### D. Turn Count Reduction
- High turn count means high cost since each turn resends the (growing) context window
- Consider pre-fetching data in `steps:` before the agent runs
- Use more directive prompts to reduce exploration turns

### Phase 6: Create Optimization Issue

Create an issue with the title: `[workflow-name] (avg [N]k tokens/run)` — the prefix `⚡ Claude Token Optimization:` is automatically added.

#### Issue Body Structure

```markdown
### Target Workflow: [workflow-name]

**Why this workflow?** Highest total token consumption across Claude workflows in the last 24 hours.

| Metric | Value |
|--------|-------|
| Runs (24h) | [N] |
| Avg tokens/run | [N]k |
| Total est. cost (24h) | $[X] |
| Avg turns/run | [N] |

### Anthropic Cache Analysis

| Metric | Value | Assessment |
|--------|-------|------------|
| Cache hit rate | [pct]% | ✅/⚠️/❌ |
| Cache write rate | [pct]% | ✅/⚠️/❌ |
| Cache read/write ratio | [ratio]× | ✅/⚠️/❌ |
| Estimated cache savings | ~$[X] (vs no cache) | — |

_(from token-usage.jsonl — if unavailable, based on run-level metrics)_

### Optimization Recommendations

#### 1. [Highest Impact — e.g., Cache Efficiency]

**Potential savings**: ~[N]k tokens/run (~$[X]/run)

**Current state**: [What is happening — e.g., "Cache write rate is 25% but cache read rate is 2%, meaning cache writes are not being amortized (ratio: 0.08×, break-even is 1.4×)"]

**Root cause**: [e.g., "System prompt is only 800 tokens — below Claude's 1024-token minimum for automatic caching"]

**Recommended change**:
```diff
# In .github/workflows/[workflow-name].md frontmatter or prompt:
- [current state]
+ [recommended state]
```

**Why this helps**: [Explanation of mechanism]

#### 2. [Second Recommendation — e.g., Output Reduction]

**Potential savings**: ~[N]k output tokens/run (~$[X]/run)

...

#### 3. [Third Recommendation — e.g., Unused Tools]

...

<details>
<summary><b>Tool Usage Analysis</b></summary>

**Declared tools** (from frontmatter):
[list from declared-tools.txt]

**Tools actually invoked** (from agent logs):
[list from mcp call analysis]

**Unused tools** (candidates for removal):
- `[toolset/tool]` — never called, saves ~500 tokens/turn if removed

</details>

<details>
<summary><b>Token Breakdown</b></summary>

| Token Type | Count | % of Total | Est. Cost Weight |
|------------|-------|------------|-----------------|
| Input | [n] | [pct]% | 1× |
| Output | [n] | [pct]% | ~4× |
| Cache Read | [n] | [pct]% | 0.1× |
| Cache Write | [n] | [pct]% | 1.25× |

</details>

### Implementation Checklist

- [ ] Apply recommended changes to `.github/workflows/[workflow-name].md`
- [ ] Run `make recompile` to regenerate the lock file
- [ ] Trigger a manual run via `workflow_dispatch` to verify
- [ ] Check cache metrics improve in next analyzer report (look for higher cache hit rate)

### References

- [Last run of [workflow-name]](LATEST_RUN_URL)
- Analysis triggered by: [§RUN_ID](RUN_URL)
```

## Important Guidelines

- **Anthropic caching is nuanced**: Don't flag cache writes as waste unless the read/write ratio is clearly below break-even (< 1.4×). Cache writes at 125% cost are an investment.
- **Output tokens are disproportionately expensive for Claude** (3-5× input cost) — reducing verbose output has high ROI.
- **1024-token minimum**: Claude won't cache a prompt shorter than 1024 tokens. If the system prompt is shorter, caching isn't available regardless of configuration.
- **Be specific**: Name exact tools, exact token counts, exact cost estimates.
- **Prioritize by impact**: List recommendations from highest to lowest savings.
- **`noop` when appropriate**: If the workflow is already well-optimized or no meaningful data is available, call `noop` with explanation.

**Important**: You MUST call a safe-output tool (`create-issue` or `noop`) at the end of your analysis. Failing to call any safe-output tool is the most common cause of workflow failures.

```json
{"noop": {"message": "No action needed: [brief explanation]"}}
```
