# Token Budget Guidelines for High-Cost Workflows

## Overview

This document establishes token budget targets and optimization strategies for agentic workflows that consume significant Copilot tokens. These guidelines help maintain cost predictability while preserving analysis quality.

## Purpose

- **Cost Control**: Prevent unbounded token consumption in expensive workflows
- **Predictability**: Establish expected token ranges per workflow run
- **Quality**: Maintain useful outputs while reducing unnecessary verbosity
- **Monitoring**: Enable tracking and alerting when budgets are exceeded

## Token Budget Configuration

### Available Controls

#### 1. `max-turns` (For Claude/Custom Engines Only)

Limits the number of conversation rounds between the agent and the AI engine.

**Important:** `max-turns` is **only supported by Claude and Custom engines**, not Copilot. For Copilot workflows, use prompt optimization and timeout controls instead.

```yaml
---
# Claude engine with max-turns
engine:
  id: claude
  max-turns: 30  # Recommended for research/analysis workflows
---
```

```yaml
---
# Custom engine with max-turns
engine:
  id: custom
  max-turns: 25  # Recommended for CI/automation workflows
---
```

**How it works:**
- Each "turn" is one round-trip: agent request → AI response
- Includes all tool calls and responses within that turn
- Workflow terminates when max-turns is reached
- Earlier termination if task completes

**Engine Support:**
- ✅ **Claude**: Fully supported
- ✅ **Custom**: Fully supported
- ❌ **Copilot**: Not supported - use prompt optimization instead
- ❌ **Codex**: Not supported

#### 2. `timeout-minutes` (Secondary Control)

Prevents workflows from running indefinitely due to unexpected loops or delays.

```yaml
---
timeout-minutes: 180  # 3 hours - research workflows
timeout-minutes: 45   # 45 minutes - CI cleanup workflows
timeout-minutes: 20   # 20 minutes - single-step automation
---
```

#### 3. Prompt Optimization (Critical for Copilot Workflows)

**Primary method for Copilot token budget control** since max-turns is not available.

Explicit instructions in workflow prompts to reduce token consumption:

**Output Size Limits:**
```markdown
## Output Guidelines

- Keep responses concise and actionable
- Main report should be under 1000 words
- Use progressive disclosure (details/summary tags)
- Summarize findings instead of exhaustive documentation
```

**Scope Reduction:**
```markdown
## Execution Scope

- Test 6-8 representative scenarios (not all scenarios)
- Focus on quality over quantity
- Prioritize critical issues over complete coverage
```

**Efficiency Instructions:**
```markdown
## Efficiency Guidelines

- Avoid verbose explanations - focus on actions
- If stuck after 3 attempts, document and move on
- Complete analysis within reasonable time
- Aim for systematic approach with minimal iteration
```

## Workflow-Specific Budgets

### Agent Persona Explorer

**Engine**: Copilot (default) - max-turns not available

**Previous Configuration:**
- No token budget controls
- 600-minute timeout
- Tests all 15-20 generated scenarios
- Complete documentation

**Optimized Configuration:**
- `timeout-minutes: 180` (reduced from 600)
- Prompt optimization: Test 6-8 representative scenarios
- Output limits: Concise documentation (<1000 words)
- Progressive disclosure for detailed content

**Expected Impact:**
- **Token Reduction**: 30-40% (from ~200K-300K to ~120K-180K per run)
- **Quality**: Maintained through strategic scenario selection
- **Runtime**: Reduced from 4-6 hours to 2-3 hours

**Budget Target:**
- **Target tokens/run**: 120K-180K
- **Alert threshold**: >200K tokens
- **Cost estimate**: $2.10-3.15 per run

**Optimization Strategy:**
- Reduce test scenarios from 15-20 to 6-8 representative cases
- Enforce concise output with word limits
- Use progressive disclosure to hide verbose content
- Focus on quality insights over complete coverage

### CI Cleaner

**Engine**: Copilot - max-turns not available

**Previous Configuration:**
- No token budget controls
- 45-minute timeout
- Already has early-exit optimization

**Optimized Configuration:**
- `timeout-minutes: 45` (unchanged)
- Enhanced efficiency instructions in prompt
- Systematic fix workflow with early termination
- Concise action-focused approach

**Expected Impact:**
- **Token Reduction**: 15-25% (from ~80K-120K to ~68K-90K per run)
- **Quality**: Maintained - focuses on systematic fixes
- **Runtime**: Maintained at 20-30 minutes

**Budget Target:**
- **Target tokens/run**: 68K-90K
- **Alert threshold**: >120K tokens
- **Cost estimate**: $1.19-1.58 per run

**Optimization Strategy:**
- Enhanced prompt with efficiency guidelines
- Early termination conditions (stop after 3 failed attempts)
- Focus on systematic fixes without over-analysis
- Prioritize formatting/linting over complex test failures

### Issue Monster

**Engine**: Copilot (gpt-5.1-codex-mini) - max-turns not available

**Previous Configuration:**
- No token budget controls
- 30-minute timeout
- No explicit conciseness guidance
- Runs every 30 minutes (48 runs/day)

**Optimized Configuration:**
- `timeout-minutes: 30` (unchanged — already short)
- Added `## Token Budget Guidelines` section in prompt:
  - Stop immediately after assignments + comments (or noop)
  - Keep issue comments to the provided template only
  - Read only what is necessary to confirm each issue
  - Do not re-summarize the pre-fetched issue list
  - One tool call per action (assign + comment = 2 calls per issue max)

**Expected Impact:**
- **Token Reduction**: 60-75% (from ~1.99M to ~500K-800K per run)
- **Quality**: Maintained — assignments and comments unchanged
- **Runtime**: Maintained at <5 minutes per run

**Budget Target:**
- **Target tokens/run**: 50K–150K
- **Alert threshold**: >300K tokens
- **Cost estimate**: $0.88-2.63 per run

**Optimization Strategy:**
- Explicit early-stop: terminate once safe-output calls are complete
- Brevity instructions for comments and internal reasoning
- Instruction not to re-read/re-summarize the pre-fetched issue list

### CI Optimization Coach

**Engine**: Copilot - max-turns not available

**Previous Configuration:**
- No token budget controls
- 30-minute timeout
- 5 analysis phases with detailed steps
- No explicit scope cap on how many opportunities to investigate

**Optimized Configuration:**
- `timeout-minutes: 30` (unchanged)
- Added `## Token Budget Guidelines` section in prompt:
  - Focus on top 3 highest-impact opportunities only
  - Early exit: if CI is healthy after Phases 1-2, skip to noop
  - PR descriptions capped at 600 words with `<details>` for extended content
  - No duplicate data downloads
  - Stop after PR creation or noop

**Expected Impact:**
- **Token Reduction**: 50-65% (from ~1.78M to ~600K-900K per run)
- **Quality**: Maintained — highest-value optimizations are still surfaced
- **Runtime**: Maintained at <30 minutes

**Budget Target:**
- **Target tokens/run**: 300K–600K
- **Alert threshold**: >1M tokens
- **Cost estimate**: $5.25-10.50 per run

**Optimization Strategy:**
- Explicit scope cap: top 3 opportunities only
- Early-exit path for healthy CI
- Concise PR template enforcement
- No extra validation steps beyond the prescribed four commands

### Step Name Alignment

**Engine**: Claude - max-turns supported

**Previous Configuration:**
- No max-turns limit
- 30-minute timeout
- Scans all lock files comprehensively
- No issue-creation cap per run
- Re-reads glossary multiple times

**Optimized Configuration:**
- `max-turns: 30` added to engine block
- `timeout-minutes: 30` (unchanged)
- Added `## Token Budget Guidelines` section in prompt:
  - Report only top 5 most impactful issues per run
  - Create at most 2 GitHub issues per run (batch findings)
  - Load glossary once; do not re-read
  - Skip workflows reviewed in the last 3 days (use cache memory)
  - Early stop after issues created + cache updated

**Expected Impact:**
- **Token Reduction**: 65-80% (from ~1.59M to ~300K-550K per run)
- **Quality**: Maintained — most impactful issues still addressed
- **Runtime**: Reduced from 25-30 minutes to 10-15 minutes

**Budget Target:**
- **Target tokens/run**: 300K–500K
- **Alert threshold**: >800K tokens
- **Cost estimate**: $5.25-8.75 per run

**Optimization Strategy:**
- Hard max-turns limit prevents runaway analysis loops
- Issue-creation cap prevents excessive GitHub API calls
- Cache-based skip logic avoids redundant re-scanning

## Alert Thresholds (Updated)

| Workflow | Target Tokens | Alert Threshold | Critical Threshold |
|----------|--------------|-----------------|-------------------|
| Agent Persona Explorer | 120K-180K | >200K | >250K |
| CI Cleaner | 68K-90K | >120K | >150K |
| Issue Monster | 50K-150K | >300K | >500K |
| CI Optimization Coach | 300K-600K | >1M | >1.5M |
| Step Name Alignment | 300K-500K | >800K | >1.2M |

## Optimization Strategies

### 1. Reduce Iteration Scope

**Before:**
```markdown
For each scenario (15-20 scenarios), test the agent response...
```

**After:**
```markdown
Test a representative subset of 6-8 scenarios to reduce token consumption...
```

**Impact**: 40-60% reduction in Phase 3 token usage

### 2. Output Compression

**Before:**
```markdown
### Detailed Analysis
[5000 word detailed report with all scenario details]
```

**After:**
```markdown
### Key Findings (3-5 bullet points max)
<details>
<summary>View Detailed Scenario Analysis</summary>
[Detailed content hidden by default]
</details>
```

**Impact**: 30-40% reduction in documentation generation tokens

### 3. Early Termination Conditions

**Before:**
```markdown
Work through each step systematically.
```

**After:**
```markdown
If all checks pass, stop immediately and create PR.
If stuck on an issue after 3 attempts, document it and move on.
```

**Impact**: 10-20% reduction by avoiding stuck states

### 4. Progressive Disclosure

Use HTML `<details>` tags to reduce initial output verbosity:

```markdown
<details>
<summary>View Detailed Examples</summary>

[Verbose content that AI generates once but doesn't repeatedly reference]

</details>
```

## Monitoring and Alerting

### Recommended Metrics

1. **Tokens per Run**: Track actual token consumption per workflow execution
2. **Cost per Run**: Calculate estimated cost based on token usage
3. **Budget Compliance**: % of runs within target budget
4. **Quality Metrics**: Ensure optimization doesn't degrade output quality

### Alert Thresholds

See the **Alert Thresholds (Updated)** table in the [Workflow-Specific Budgets](#workflow-specific-budgets) section above for all workflows including Issue Monster, CI Optimization Coach, and Step Name Alignment.

### Monitoring Tools

Use the daily Copilot token report workflow:
- Location: `.github/workflows/daily-copilot-token-report.md`
- Generates per-workflow statistics
- Tracks historical trends
- Identifies cost anomalies

## Implementation Checklist

When adding token budgets to a workflow:

- [ ] Set `max-turns` based on workflow complexity
- [ ] Adjust `timeout-minutes` to reasonable completion time
- [ ] Add output size limits in prompt instructions
- [ ] Add efficiency guidelines for agent behavior
- [ ] Document budget targets in workflow comments
- [ ] Consider scope reduction opportunities
- [ ] Add early termination conditions
- [ ] Test to verify budget compliance
- [ ] Monitor actual token consumption
- [ ] Adjust thresholds based on real-world data

## Best Practices

### DO:
- ✅ Set `max-turns` for all production workflows
- ✅ Document budget targets in workflow frontmatter comments
- ✅ Use progressive disclosure for verbose outputs
- ✅ Provide explicit output size limits
- ✅ Add early termination conditions
- ✅ Monitor token consumption trends
- ✅ Test workflows to verify budget compliance

### DON'T:
- ❌ Set max-turns so low that workflows can't complete
- ❌ Sacrifice quality for marginal token savings
- ❌ Ignore budget exceedances without investigation
- ❌ Over-optimize prompts to the point of confusion
- ❌ Remove important analysis steps without validation
- ❌ Deploy budget changes without testing

## Token Pricing Reference

**Current Copilot Pricing** (as of 2026-01):
- Input tokens: ~$0.015 per 1K tokens
- Output tokens: ~$0.020 per 1K tokens
- Average: ~$0.0175 per 1K tokens (blended)

**Cost Examples:**
- 100K tokens ≈ $1.75
- 150K tokens ≈ $2.63
- 200K tokens ≈ $3.50
- 300K tokens ≈ $5.25

## Revision History

- **2026-03-23**: Added token budget guardrails for top-cost workflows
  - Issue Monster: added prompt-level efficiency & early-stop guidelines (target 50K-150K/run)
  - CI Optimization Coach: added scope cap and early-exit path (target 300K-600K/run)
  - Step Name Alignment: added `max-turns: 30` and issue-creation cap (target 300K-500K/run)
  - Updated alert thresholds table to cover all five tracked workflows
  - See [DeepReport #23445279565](https://github.com/github/gh-aw/actions/runs/23445279565)

- **2026-01-26**: Initial guidelines created
  - Added budgets for Agent Persona Explorer and CI Cleaner
  - Established monitoring framework
  - Documented optimization strategies

## References

- [Daily Copilot Token Report](.github/workflows/daily-copilot-token-report.md)
- [Token Cost Analysis Module](.github/workflows/shared/token-cost-analysis.md)
- [DeepReport Intelligence Briefing 2026-01-26](https://github.com/github/gh-aw/actions/runs/21355400856)
