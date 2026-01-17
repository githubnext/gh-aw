# GitHub Actions Cost Estimation Template

This template provides a framework for estimating the cost of GitHub Actions workflows, particularly for scheduled automation.

## Cost Estimation Formula

The total cost of a workflow depends on several factors:

```
Monthly Minutes = (Base Execution Time + Tool Multipliers) × Schedule Frequency
```

### Base Execution Time

- **Simple workflows** (basic API calls, minimal processing): ~2 minutes per run
- **Medium workflows** (data processing, multiple API calls): ~3-5 minutes per run
- **Complex workflows** (extensive analysis, file operations): ~5-10 minutes per run

### Tool Multipliers

Add these minutes to the base execution time based on tools used:

| Tool/Feature | Additional Minutes | Reason |
|--------------|-------------------|--------|
| **Playwright** | +5 minutes | Browser automation with containerization overhead |
| **repo-memory** | +1 minute | Git branch cloning and pushing operations |
| **Network API calls** (external) | +2 minutes | HTTP requests to external services |
| **cache-memory** | +0.5 minutes | Artifact upload/download operations |
| **Multiple MCP servers** (3+) | +1-2 minutes | Additional tool initialization overhead |
| **File processing** (large repos) | +2-3 minutes | Checkout and file scanning operations |

### Schedule Frequency Calculations

| Schedule | Runs per Month | Example Cron |
|----------|----------------|--------------|
| **Hourly** | 720 | `"0 * * * *"` |
| **Every 6 hours** | 120 | `"0 */6 * * *"` |
| **Daily** | 30 | `"0 2 * * *"` |
| **Weekdays only** | 22 | `"0 2 * * 1-5"` |
| **Weekly** | 4 | `"0 2 * * 1"` |
| **Bi-weekly** | 2 | `"0 2 1,15 * *"` |

## Cost Table Template

When creating scheduled workflows, include this table in the workflow documentation:

```markdown
## Estimated Cost

- **Base execution time**: ~[X] minutes per run
- **Tool overhead**: +[Y] minutes ([list tools])
- **Total per run**: ~[X+Y] minutes
- **Schedule**: [Frequency] ([runs per month] runs/month)
- **Monthly total**: ~[total] minutes (~[hours] hours)
- **Free tier impact**: [percentage]% of free tier

### Free Tier Comparison

- **Public repositories**: 2,000 minutes/month (free tier)
- **Private repositories**: 2,000 minutes/month (free tier for Pro accounts)
- **This workflow**: [total] minutes/month
- **Status**: ✅ Within free tier / ⚠️ Exceeds free tier by [X] minutes
```

## Example 1: Hourly Monitoring Workflow

```markdown
## Estimated Cost

- **Base execution time**: ~3 minutes per run
- **Tool overhead**: +2 minutes (network API calls)
- **Total per run**: ~5 minutes
- **Schedule**: Hourly (720 runs/month)
- **Monthly total**: ~3,600 minutes (~60 hours)
- **Free tier impact**: 180% (exceeds free tier by 1,600 minutes)

### Optimization Suggestions

⚠️ **This workflow exceeds the free tier.** Consider these optimizations:

1. **Reduce frequency**: Change from hourly to every 6 hours (saves 2,400 minutes/month)
2. **Use path filters**: Skip runs when monitored files haven't changed
3. **Conditional execution**: Add pre-activation checks to exit early when no action needed
```

## Example 2: Daily Code Scan

```markdown
## Estimated Cost

- **Base execution time**: ~4 minutes per run
- **Tool overhead**: +3 minutes (repo-memory + file processing)
- **Total per run**: ~7 minutes
- **Schedule**: Daily on weekdays (22 runs/month)
- **Monthly total**: ~154 minutes (~2.6 hours)
- **Free tier impact**: 7.7% of free tier

### Status

✅ **Well within free tier limits.** This workflow is cost-efficient for daily operations.
```

## Example 3: Weekly Digest with Playwright

```markdown
## Estimated Cost

- **Base execution time**: ~5 minutes per run
- **Tool overhead**: +5 minutes (Playwright browser automation)
- **Total per run**: ~10 minutes
- **Schedule**: Weekly (4 runs/month)
- **Monthly total**: ~40 minutes (~0.67 hours)
- **Free tier impact**: 2% of free tier

### Status

✅ **Minimal cost impact.** Ideal for weekly automation without concerns about usage limits.
```

## When to Include Cost Estimation

**Always include** cost estimation tables for:

- Scheduled workflows (`on: schedule`)
- Workflows that run frequently (hourly, daily)
- Workflows using expensive tools (Playwright, extensive file operations)
- Workflows in documentation examples

**Optional** for:

- Event-driven workflows (issues, pull requests)
- Manual workflows (`workflow_dispatch` only)
- One-time or rare automation tasks

## Optimization Best Practices

When a workflow exceeds 1,500 minutes/month (75% of free tier):

1. **Reduce frequency**:
   - Hourly → Every 6 hours (saves 80% of runs)
   - Daily → Weekdays only (saves 25% of runs)
   - Consider business hours only for monitoring

2. **Add path filters**:
   ```yaml
   on:
     schedule:
       - cron: "0 * * * *"
     push:
       paths:
         - 'src/**'
         - 'config/**'
   ```

3. **Use pre-activation checks**:
   ```yaml
   pre-activation:
     - name: Check if work needed
       run: |
         # Exit early if no changes detected
         if [ "$(git diff --name-only HEAD^)" = "" ]; then
           echo "No changes detected, skipping workflow"
           exit 1
         fi
   ```

4. **Optimize tool usage**:
   - Run jobs in parallel instead of sequential
   - Cache dependencies to reduce setup time
   - Use `timeout-minutes` to prevent runaway jobs
   - Minimize network calls with batching

5. **Consider alternative triggers**:
   - Replace schedule with webhook events
   - Use repository_dispatch for external triggers
   - Implement manual review gates for expensive operations

## Cost Tracking

Monitor your actual usage with:

```bash
gh aw logs --workflow [workflow-name] --stats
```

This provides:
- Average execution time
- Token usage
- Actual cost per run
- Monthly projection based on schedule
