# Portfolio Analyst Report Format Comparison

## Problem Statement (Issue #6322)

The Portfolio Analyst workflow generates reports that are "much too long and cluttered" making it difficult to quickly understand key insights and track progress week-over-week.

**Example**: The latest report (Discussion #6322) is approximately 4,000+ words with dense text sections and no visual elements.

## Solution

Transform the report format from text-heavy to visual, dashboard-style with embedded charts.

---

## Before: Text-Heavy Format

```markdown
# Portfolio Analysis Report - 2025-12-13

## Data Availability
⚠️ **Important**: This analysis is based on a single day snapshot...
[Long paragraph about data limitations]

## Executive Summary
- Total Workflows Analyzed: 172
- Current Daily Cost: $61.17
- Projected Monthly Cost: $1,835.18
- Potential Savings: $847.20/month (46% reduction)
...

## Cost Reduction Strategies

### 1. Fix Critical High-Failure Workflows ($335.81/month wasted)

These workflows are burning money on failed runs...

#### **AI Moderator** - 86.7% failure rate (26/30 runs)
- **Wasted Cost**: $3.11/day × 30 = $93.36/month
- **Root Cause**: 1034 errors across 26 failed runs
- **Action**: Investigate error logs...
[300+ lines of detailed analysis per workflow]

#### **Hourly CI Cleaner** - 50.0% failure rate (27/54 runs)
[Another 200+ lines]
...

[Continues for 4000+ words with no visual breaks]
```text

**Problems:**
- ❌ Too long (4000+ words)
- ❌ No visual elements
- ❌ Hard to scan quickly
- ❌ Difficult to compare week-over-week
- ❌ Key insights buried in text

---

## After: Visual Dashboard Format

```markdown
# 📊 Portfolio Dashboard - 2025-12-13

## Quick Overview

Analysis of 511 workflow runs over 24 hours shows $61/day in costs with significant optimization opportunities. 7 workflows require immediate attention, with potential savings of $847/month (46% reduction).

## Visual Summary

### Cost Trends (Last 30 Days)

![Cost Trends](https://raw.githubusercontent.com/.../cost_trends.png)

**Key Insights**:
- Daily average: $61.17
- Trend: Stable (±5% variation)
- Monthly total: $1,835

### Top Cost Drivers

![Top Spenders](https://raw.githubusercontent.com/.../top_spenders.png)

Top 3 workflows account for 32% of total cost:
1. `hourly-ci-cleaner.md` - $194/month (50% failure rate ⚠️)
2. `tidy.md` - $190/month (40% failure rate ⚠️)
3. `ai-moderator.md` - $93/month (87% failure rate 🔥)

### Failure Analysis

![Failure Rates](https://raw.githubusercontent.com/.../failure_rates.png)

**Wasted Spend**: $336/month on failed runs
- 7 workflows with >30% failure rate
- 12 workflows with 100% failure rate (should be disabled)

### Overall Health

![Success Overview](https://raw.githubusercontent.com/.../success_overview.png)

- ✅ Success: 69.3% (354 runs)
- ❌ Failure: 12.7% (65 runs)
- ⏸️ Cancelled: 18.0% (92 runs)

## 💰 Cost Reduction Opportunities

**Total Potential Savings: $847/month (46% reduction)**

<details>
<summary><b>Strategy 1: Fix High-Failure Workflows - $336/month</b></summary>

| Workflow | Failure Rate | Wasted Cost | Fix |
|----------|-------------|-------------|-----|
| `ai-moderator.md` | 86.7% | $93/mo | Investigate 1034 errors in logs |
| `hourly-ci-cleaner.md` | 50.0% | $97/mo | Reduce frequency to every 4h |
| `tidy.md` | 39.6% | $75/mo | Fix 1476 errors |

</details>

<details>
<summary><b>Strategy 2: Reduce Over-Scheduling - $369/month</b></summary>

| Workflow | Current | Recommended | Savings |
|----------|---------|-------------|---------|
| `hourly-ci-cleaner.md` | 54 runs/day | 6 runs/day | $172/mo |
| `tidy.md` | 53 runs/day | 4 runs/day | $176/mo |

</details>

<details>
<summary><b>Strategy 3: Disable Failed Workflows - $142/month</b></summary>

12 workflows with 0% success rate - disable immediately.

</details>

## 🎯 Priority Actions

1. **CRITICAL** - Disable `/cloclo` (100% failure, $32/mo waste)
2. **CRITICAL** - Fix AI Moderator (87% failure, $93/mo waste)
3. **HIGH** - Reduce Hourly CI Cleaner frequency (save $172/mo)

## 📈 Data Quality

- **Period Analyzed**: Dec 13, 2025 (24 hours)
- **Total Runs**: 511 workflow runs
- **Workflows**: 172 total, 75 executed
- **Confidence**: Low - single day snapshot, re-run after 7 days

---

**Methodology**: Analysis based on actual workflow execution data from `gh aw logs` for last 30 days. Costs calculated from real token usage.
```text

**Benefits:**
- ✅ Concise (1200 words vs 4000+)
- ✅ Visual charts show trends immediately
- ✅ Scannable structure with clear sections
- ✅ Easy to compare week-over-week (same format)
- ✅ Key insights upfront, details in collapsible sections
- ✅ Dashboard-like for quick decision making

---

## Key Changes Made

### 1. Added Python Data Visualization

**Import**: `shared/trending-charts-simple.md`
- Provides Python environment (pandas, matplotlib, seaborn)
- Sets up directory structure for data and charts
- Automatically uploads charts as artifacts

### 2. Added Asset Upload Capability

**Safe Output**: `upload-assets`
- Uploads generated charts to GitHub
- Returns URLs for embedding in discussions
- Charts persist across workflow runs

### 3. Updated Workflow Instructions

Added comprehensive "Visualization Requirements" section:
- Specifies 4 required charts with exact requirements
- Provides data preparation examples
- Shows chart generation workflow
- Includes quality standards (300 DPI, professional styling)

### 4. Replaced Output Format

Old format: Dense text with nested lists and long paragraphs
New format: Visual dashboard with:
- Quick 2-3 sentence overview
- 4 embedded charts in "Visual Summary"
- Key insights as bullet points
- Detailed recommendations in collapsible sections
- Priority actions as numbered list

### 5. Updated Success Criteria

Added requirements for:
- All 4 charts must be generated
- Charts must be uploaded as assets
- Charts must be embedded in report
- Dashboard-style format required
- Report length reduced to <1500 words

---

## Example Python Script

Created `examples/portfolio-analyst-charts.py` demonstrating:
1. Loading data from summary.json
2. Preparing dataframes with pandas
3. Generating all 4 required charts
4. Professional styling with seaborn
5. Saving high-quality PNG files (300 DPI)

The script can be run as-is or adapted by the AI agent during workflow execution.

---

## Impact

**Before**: Users must read 4000+ words of dense text to find key issues
**After**: Users see key issues in 4 charts within seconds, can drill down as needed

**Before**: No way to quickly compare trends week-over-week
**After**: Same chart format every week enables visual comparison

**Before**: Recommendations buried in long text sections
**After**: Top 3 priority actions clearly highlighted

The new format transforms the Portfolio Analyst from a data dump into an actionable dashboard.
