# Workflow Performance Engineering Guide

This guide helps you optimize agentic workflow execution efficiency, focusing on token usage, execution time, and cost reduction.

## Quick Performance Checks

```bash
# Analyze recent workflow runs
gh aw logs workflow-name -c 10

# Audit specific run for performance insights
gh aw audit <run-id>

# Download logs for analysis
gh aw logs --start-date -1w -o /tmp/gh-aw/perf/
```

## Performance Targets

- **Average tokens per run**: < 200k (current baseline: ~481k)
- **Average cost per run**: < $0.15 (current baseline: ~$0.30)
- **Execution time**: 2-5 minutes for typical workflows
- **Tool call efficiency**: 30% reduction in redundant calls

## Common Performance Bottlenecks

### 1. Excessive Token Usage from API Calls

**Problem**: Large API responses (especially `pull_request_read`) consume excessive tokens.

**Example Issue**: Workflow using 481k tokens, 60-70% in API responses.

**Measurement**:
```bash
# Analyze token distribution in logs
gh aw logs workflow-name -c 1
grep -i "token" /tmp/gh-aw/logs/workflow-name/*.log
```

**Optimization Strategy**:

**✓ Use Sanitized Context Text** (Recommended):
```markdown
---
on:
  issues:
    types: [opened]
---

# Analyze issue: "${{ needs.activation.outputs.text }}"
```

**Benefits**:
- Pre-sanitized content (safe from XPIA)
- Already concatenated (title + body)
- Significantly smaller than full API response
- No extra API calls needed

**✗ Avoid Large API Responses**:
```markdown
# DON'T DO THIS
Use the github tool to get full issue details with all fields.
```

**Impact**: 60-70% token reduction (481k → 150k tokens)
**Cost Savings**: ~$0.15 per run, $90-100 annually at 500 runs/year

### 2. Redundant Tool Calls and Context Fetching

**Problem**: Workflows repeatedly call the same tools or fetch the same context.

**Measurement**:
```bash
# Count tool calls in logs
grep "tool_use" /tmp/gh-aw/logs/workflow/*.log | wc -l

# Identify repeated calls
grep "tool_use" /tmp/gh-aw/logs/workflow/*.log | sort | uniq -c | sort -rn
```

**Optimization Strategy**:

**Use Cache Memory** for persistent state:
```yaml
---
tools:
  cache-memory: true
---

# Store analysis results in /tmp/gh-aw/cache-memory/
# Reuse across multiple runs
```

**Pre-compute Common Values** in workflow prompt:
```markdown
Repository: ${{ github.repository }}
Issue Number: ${{ github.event.issue.number }}
Issue Content: "${{ needs.activation.outputs.text }}"

Analyze the above issue (content already provided).
```

**Impact**: 20-30% reduction in redundant tool calls

### 3. Inefficient Pre-step Patterns

**Problem**: AI wastes turns setting up environment that could be pre-configured.

**Bad Pattern**:
```yaml
# Workflow without pre-steps
# AI will spend turns: mkdir, git config, etc.
```

**Good Pattern**:
```yaml
steps:
  setup-env:
    name: Setup Environment
    run: |
      mkdir -p /tmp/gh-aw/work
      git config --global user.name "github-actions[bot]"
      git config --global user.email "github-actions[bot]@users.noreply.github.com"
```

**Impact**: Eliminates 20-30% of redundant setup turns

### 4. Unnecessary Large Responses

**Problem**: Requesting full file contents when only excerpts needed.

**Optimization Strategy**:

**✓ Request Specific Sections**:
```markdown
Read lines 50-100 of src/main.go to analyze the Init function.
```

**✓ Use Grep/Search First**:
```markdown
Search for "func Init" in src/ directory first, then read only that function.
```

**✗ Avoid Full File Dumps**:
```markdown
Read all files in the src/ directory.  # BAD - excessive tokens
```

## Performance Optimization Techniques

### Token-Efficient Prompt Design

**✓ Good Prompt Structure**:
```markdown
---
on: issues
---

# Issue Triage for #${{ github.event.issue.number }}

**Issue Content**: "${{ needs.activation.outputs.text }}"

Tasks:
1. Categorize issue type (bug/feature/question)
2. Add label using github tool
3. Post comment with category

Use the provided issue content above. Do not fetch it again.
```

**Key Principles**:
- Provide context upfront (avoid re-fetching)
- Use sanitized context text when possible
- Be specific about required actions
- Explicitly tell AI not to re-fetch available data

### Efficient Tool Configuration

**✓ Minimal Tool Set**:
```yaml
tools:
  github:
    allowed:
      - add_labels_to_issue
      - create_issue_comment
  # Only tools needed for this specific workflow
```

**✗ Over-permissive Tools**:
```yaml
tools:
  github:  # Grants all GitHub tools - increases token usage in tool list
  web-fetch:  # Unnecessary if not fetching web content
  web-search:  # Unnecessary if not searching
```

**Impact**: Smaller tool list in system prompt = fewer tokens per turn

### Caching Strategies

**Persistent Cache Memory**:
```yaml
tools:
  cache-memory:
    key: analysis-cache-${{ github.workflow }}
```

**Use Cases**:
- Store expensive analysis results
- Cache parsed data structures
- Remember previous workflow state
- Avoid re-computing same values

**Example Usage in Prompt**:
```markdown
Store your analysis in /tmp/gh-aw/cache-memory/analysis.json for reuse.
Check if analysis already exists before starting.
```

**Multi-Cache Pattern** (for complex workflows):
```yaml
tools:
  cache-memory:
    - id: results
      key: results-${{ github.workflow }}
    - id: state
      key: state-${{ github.run_id }}
```

### Pre-step Optimization Patterns

**Setup Git Properly**:
```yaml
steps:
  setup-git:
    run: |
      git config --global user.name "github-actions[bot]"
      git config --global user.email "github-actions[bot]@users.noreply.github.com"
```

**Pre-create Work Directories**:
```yaml
steps:
  setup-dirs:
    run: |
      mkdir -p /tmp/gh-aw/work
      mkdir -p /tmp/gh-aw/output
```

**Pre-install Tools** (if needed):
```yaml
steps:
  setup-tools:
    run: |
      npm install -g prettier
      pip install black
```

## Performance Measurement Workflow

### 1. Establish Baseline

```bash
# Run workflow and capture metrics
gh aw logs workflow-name -c 5

# Analyze logs for:
# - Total tokens used
# - Number of turns
# - Execution time
# - Cost estimate
```

### 2. Identify Bottlenecks

```bash
# Look for:
# - Large API responses (>10k tokens)
# - Repeated tool calls
# - Unnecessary file reads
# - Redundant context fetching

grep -i "tokens" logs/*.log | sort -t: -k2 -rn | head -20
```

### 3. Apply Optimization

Make targeted changes:
- Replace API calls with sanitized context
- Add pre-steps for common setup
- Enable cache-memory where appropriate
- Reduce tool permissions to minimum needed

### 4. Measure Impact

```bash
# Run optimized workflow
gh aw logs workflow-name -c 5

# Compare metrics:
# - Token reduction %
# - Turn reduction %
# - Time improvement
# - Cost savings
```

### 5. Validate Correctness

```bash
# Ensure workflow still works correctly
# Check output quality hasn't degraded
# Verify all required actions complete
```

## Token Usage Patterns

### High-Value Token Spending

✓ **Worth the tokens**:
- Actual work (code changes, analysis, writing)
- Necessary GitHub API operations
- Critical tool executions
- User-facing output

✗ **Wasteful token spending**:
- Re-fetching already available context
- Reading entire files when only excerpt needed
- Repeated identical API calls
- Verbose debug output in production

### Cost Calculation

```
Tokens per run: 200,000
Cost per 1M tokens (input): ~$0.30
Cost per 1M tokens (output): ~$0.60
Average cost per run: ~$0.15

Annual cost (500 runs): ~$75
```

**Optimization Impact**:
- 60% token reduction: Save ~$45/year
- 30% turn reduction: Save ~$20/year
- Combined: ~$65/year savings

## Common Anti-Patterns

### ✗ Pattern: Fetch-Then-Summarize

```markdown
Get the full issue details, then summarize the description.
```

**Problem**: Fetches large response just to extract what's already in `needs.activation.outputs.text`.

**Solution**:
```markdown
Issue: "${{ needs.activation.outputs.text }}"
Summarize the above issue content.
```

### ✗ Pattern: Repeated Setup

```markdown
# Workflow runs multiple times
# Each time: mkdir, git config, npm install
```

**Problem**: Setup commands consume turns every execution.

**Solution**: Use `steps:` for one-time setup, use `cache-memory` to persist state.

### ✗ Pattern: Over-Fetching

```markdown
Read all files in the repository to understand the project structure.
```

**Problem**: Massive token usage, slow execution, expensive.

**Solution**:
```markdown
Read README.md and package.json to understand the project.
List files in src/ to see structure (don't read all).
```

## Performance Testing Checklist

Before committing workflow performance changes:

- [ ] Measure baseline token usage: `gh aw logs -c 5`
- [ ] Apply optimization (context text, caching, pre-steps)
- [ ] Measure optimized token usage
- [ ] Calculate improvement % and cost savings
- [ ] Test workflow completes successfully
- [ ] Verify output quality unchanged
- [ ] Document optimization in commit message

## Troubleshooting

### Problem: Workflow uses too many tokens

**Diagnosis**:
```bash
# Analyze token distribution
gh aw logs workflow-name -c 1
grep -A5 "token_usage" logs/*.log
```

**Solution**:
1. Replace API calls with `needs.activation.outputs.text`
2. Reduce tool permissions to minimum
3. Add explicit "don't re-fetch" instructions

### Problem: Workflow takes too many turns

**Diagnosis**:
```bash
# Count turns in logs
grep "turn" logs/*.log | wc -l
```

**Solution**:
1. Add pre-steps for setup tasks
2. Provide more context upfront
3. Use cache-memory for state
4. Be more directive in prompt

### Problem: High cost per run

**Diagnosis**:
```bash
# Calculate cost from logs
# Input tokens * $0.30/1M + Output tokens * $0.60/1M
```

**Solution**:
1. Apply token reduction techniques
2. Reduce max-turns if workflow is looping
3. Optimize tool selection
4. Use caching aggressively

## Resources

- Issue #1728: Token usage optimization case study (481k → 150k)
- Issue #2012: Workflow performance analysis patterns
- `gh aw logs` documentation: Analyzing workflow execution
- Sanitized context text: Using `needs.activation.outputs.text`
