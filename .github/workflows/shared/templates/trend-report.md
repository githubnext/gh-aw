# Trend Report Template

This template provides instructions for generating security scan trend analysis sections in reports.

## Purpose

Use this template when creating discussions or reports that need to include historical trend analysis comparing current security scan results with previous scans.

## Trend Analysis Format

When including trend analysis in your report, use the following structure:

### Example: Improvement Trend

```markdown
## 📈 Trend Analysis

**Week-over-Week Change**: ↓ 15% improvement (116 → 98 findings)

| Metric | Current | Previous | Change |
|--------|---------|----------|--------|
| Total Findings | 98 | 116 | ↓ -18 (-15.5%) |
| Zizmor | 40 | 50 | ↓ -10 (-20.0%) |
| Poutine | 28 | 30 | ↓ -2 (-6.7%) |
| Actionlint | 30 | 36 | ↓ -6 (-16.7%) |

**Improvements**:
- Resolved 25 SC2086 shellcheck warnings
- Fixed 2 medium severity template injections

**Status**: Security posture is improving ✅
```

### Example: Regression Trend

```markdown
## 📈 Trend Analysis

**Week-over-Week Change**: ↑ 20% regression (100 → 120 findings)

| Metric | Current | Previous | Change |
|--------|---------|----------|--------|
| Total Findings | 120 | 100 | ↑ +20 (+20.0%) |
| Zizmor | 55 | 45 | ↑ +10 (+22.2%) |
| Poutine | 35 | 30 | ↑ +5 (+16.7%) |
| Actionlint | 30 | 25 | ↑ +5 (+20.0%) |

**New Issues**:
- 15 new template injection vulnerabilities detected
- 5 new supply chain security issues

**Status**: Security posture regressed - immediate action needed ⚠️
```

### Example: Stable Trend

```markdown
## 📈 Trend Analysis

**Week-over-Week Change**: → No change (100 findings)

| Metric | Current | Previous | Change |
|--------|---------|----------|--------|
| Total Findings | 100 | 100 | → +0 (+0.0%) |
| Zizmor | 45 | 45 | → +0 (+0.0%) |
| Poutine | 30 | 30 | → +0 (+0.0%) |
| Actionlint | 25 | 25 | → +0 (+0.0%) |

**Status**: Security posture remains stable
```

### Example: First Scan (No Historical Data)

```markdown
## 📈 Trend Analysis

**First Scan**: No historical data available for comparison.

**Current Findings**: 116 total issues

This establishes the baseline for future trend tracking. Subsequent scans will include week-over-week comparisons to monitor security posture improvements and regressions.
```

## Trend Indicators

Use these Unicode symbols to indicate trend direction:

- **↓** (U+2193) - Improvement (fewer issues)
- **↑** (U+2191) - Regression (more issues)  
- **→** (U+2192) - Stable (no change)

## Calculation Guidelines

### Percentage Change Formula

```
Percentage Change = ((Current - Previous) / Previous) × 100
```

### Trend Classification

- **Improvement**: Total findings decreased from previous scan
- **Regression**: Total findings increased from previous scan
- **Stable**: Total findings unchanged from previous scan

### Significance Thresholds

Consider highlighting trends that exceed these thresholds:

- **Minor change**: 0-10% variation
- **Moderate change**: 10-25% variation
- **Significant change**: >25% variation

## Data Sources

Trend data should be sourced from:

- **Current scan**: Analysis results from latest scan execution
- **Historical scans**: Data from `/tmp/gh-aw/cache-memory/security-scans/`
- **Scan index**: `/tmp/gh-aw/cache-memory/security-scans/index.json`
- **Aggregates**: `/tmp/gh-aw/cache-memory/security-scans/trends/weekly.json` and `monthly.json`

## Best Practices

1. **Always include trend section**: Even for first scan, acknowledge baseline establishment
2. **Be specific about changes**: Mention what improved or regressed
3. **Provide context**: Explain what the numbers mean for security posture
4. **Use consistent formatting**: Follow the table format for comparability
5. **Include actionable insights**: Tell readers what to do based on trends
6. **Keep history rolling**: Maintain 90-day rolling history for trends
7. **Update aggregates**: Calculate weekly and monthly averages

## Integration Example

Here's how to integrate trend analysis into a full security report:

```markdown
# 🔍 Static Analysis Report - November 5, 2025

## Analysis Summary

Daily security scan completed with 116 total findings across 87 workflows.

## 📈 Trend Analysis

**Week-over-Week Change**: ↓ 15% improvement (136 → 116 findings)

| Metric | Current | Previous | Change |
|--------|---------|----------|--------|
| Total Findings | 116 | 136 | ↓ -20 (-14.7%) |
| Zizmor | 50 | 60 | ↓ -10 (-16.7%) |
| Poutine | 30 | 36 | ↓ -6 (-16.7%) |
| Actionlint | 36 | 40 | ↓ -4 (-10.0%) |

**Improvements**:
- Resolved 20 issues since last scan
- Security posture improving week-over-week

## Findings by Tool

[Rest of detailed findings...]
```

## Usage in Workflows

When using this template in an agentic workflow:

1. Load historical scan data from cache memory
2. Calculate current scan results
3. Compare current vs. previous using the formulas above
4. Format output using the template structure
5. Include in discussion or report body

## Maintenance

- Update examples as new patterns emerge
- Refine thresholds based on actual usage
- Add new trend types as needed (monthly, quarterly)
- Document any calculation changes for consistency
