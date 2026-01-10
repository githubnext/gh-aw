---
title: Schedule Syntax
description: Complete reference for fuzzy schedule syntax and cron expressions
sidebar:
  order: 405
---

This reference documents the complete schedule syntax supported by GitHub Agentic Workflows, including fuzzy schedules (recommended), time constraints, and standard cron expressions.

## Overview

GitHub Agentic Workflows supports human-friendly schedule expressions that are automatically converted to cron format. The system includes two types of schedules:

- **Fuzzy schedules** (recommended) - Automatically scatter execution times across workflows to prevent load spikes
- **Fixed schedules** - Run at specific times, but create server load when many workflows use the same time

:::tip[Use Fuzzy Schedules]
Fuzzy schedules distribute workflow execution times deterministically across all workflows in your repository. Each workflow gets a unique, consistent execution time that never changes across recompiles, preventing server load spikes.
:::

## Quick Reference

| Pattern | Example | Result | Type |
|---------|---------|--------|------|
| **Daily** | `daily` | Scattered time | Fuzzy |
| | `daily around 14:00` | 13:00-15:00 window | Fuzzy |
| | `daily between 9:00 and 17:00` | 9am-5pm window | Fuzzy |
| | `daily at 02:00` | 2:00 AM UTC | Fixed ⚠️ |
| **Hourly** | `hourly` | Scattered minute | Fuzzy |
| | `every 2h` | Every 2 hours | Fuzzy |
| **Weekly** | `weekly` | Scattered day/time | Fuzzy |
| | `weekly on monday` | Monday, scattered time | Fuzzy |
| | `weekly on friday around 5pm` | Friday 4pm-6pm | Fuzzy |
| | `weekly on monday at 09:00` | Monday 9:00 AM | Fixed ⚠️ |
| **Monthly** | `monthly on 15` | 15th at midnight | Fixed |
| | `monthly on 1 at 9am` | 1st at 9:00 AM | Fixed |
| **Intervals** | `every 10 minutes` | Every 10 minutes | Fixed |
| | `every 2 days` | Every 2 days | Fixed |
| **Cron** | `0 9 * * 1` | Standard cron | Fixed |

## Fuzzy Schedules (Recommended)

Fuzzy schedules automatically distribute workflow execution times to prevent server load spikes. The scattering is deterministic based on the workflow file path, so each workflow consistently gets the same execution time.

### Daily Schedules

Run once per day at a scattered time:

```yaml
on:
  schedule: daily
```

**Output**: Each workflow gets a unique time like `43 5 * * *` (5:43 AM)

**Use cases**:
- Daily reports
- Nightly maintenance
- Status updates

### Daily with Time Constraints

Scatter within a specific time window using `around` or `between`:

#### Around (±1 hour window)

```yaml
on:
  schedule: daily around 14:00
```

Scatters within 13:00-15:00 (±1 hour window)

```yaml
on:
  schedule: daily around 3pm
```

Scatters within 2pm-4pm

```yaml
on:
  schedule: daily around noon
```

Scatters within 11am-1pm

**Special time keywords**: `midnight` (00:00), `noon` (12:00)

#### Between (Custom time range)

```yaml
on:
  schedule: daily between 9:00 and 17:00
```

Scatters within business hours (9am-5pm)

```yaml
on:
  schedule: daily between 9am and 5pm
```

Same as above using 12-hour format

```yaml
on:
  schedule: daily between 22:00 and 02:00
```

Handles ranges that cross midnight (10pm-2am)

```yaml
on:
  schedule: daily between midnight and 6am
```

Early morning window (12am-6am)

**Use cases**:
- Business hours only execution
- Regional time windows
- Off-hours maintenance (crossing midnight)

### Hourly Schedules

Run every hour with scattered minute offset:

```yaml
on:
  schedule: hourly
```

**Output**: `58 */1 * * *` (minute offset is scattered, e.g., minute 58)

Each workflow gets a consistent minute offset (0-59) to prevent all hourly workflows from running at the same minute.

### Interval Schedules

Run every N hours with scattered minute offset:

```yaml
on:
  schedule: every 2h
```

**Output**: `53 */2 * * *` (every 2 hours at scattered minute)

```yaml
on:
  schedule: every 6h
```

**Output**: `12 */6 * * *` (every 6 hours at scattered minute)

**Supported intervals**: `1h`, `2h`, `3h`, `4h`, `6h`, `8h`, `12h`

### Weekly Schedules

Run once per week at scattered day and time:

```yaml
on:
  schedule: weekly
```

**Output**: Scattered to a random day and time like `43 5 * * 1` (Monday 5:43 AM)

Run on specific weekday at scattered time:

```yaml
on:
  schedule: weekly on monday
```

**Output**: `43 5 * * 1` (Monday at scattered time)

```yaml
on:
  schedule: weekly on friday
```

**Output**: `18 14 * * 5` (Friday at scattered time)

**Supported weekdays**: `sunday`, `monday`, `tuesday`, `wednesday`, `thursday`, `friday`, `saturday`

### Weekly with Time Constraints

Add `around` to scatter within ±1 hour of target time:

```yaml
on:
  schedule: weekly on monday around 09:00
```

Scatters Monday 8am-10am

```yaml
on:
  schedule: weekly on friday around 5pm
```

Scatters Friday 4pm-6pm

## UTC Offset Support

All time specifications support UTC offset notation to convert times to UTC:

### Syntax

- **Plus offset**: `utc+N` or `utc+HH:MM`
- **Minus offset**: `utc-N` or `utc-HH:MM`

### Examples

```yaml
on:
  schedule: daily at 14:00 utc+9
```

Converts 2:00 PM JST to 5:00 AM UTC

```yaml
on:
  schedule: daily at 9am utc-5
```

Converts 9:00 AM EST to 2:00 PM UTC

```yaml
on:
  schedule: daily around 3pm utc-8
```

Converts 3:00 PM PST (±1 hour) to UTC

```yaml
on:
  schedule: daily between 9am utc-5 and 5pm utc-5
```

Business hours EST (9am-5pm EST → 2pm-10pm UTC)

```yaml
on:
  schedule: weekly on monday at 08:00 utc+05:30
```

8:00 AM IST to UTC

**Common offsets**:
- PST/PDT: `utc-8` / `utc-7`
- EST/EDT: `utc-5` / `utc-4`
- JST: `utc+9`
- IST: `utc+05:30`

## Fixed Schedules

Fixed schedules run at specific times. Use sparingly, as many workflows with the same fixed time create server load spikes.

:::caution[Load Spikes]
Fixed schedules cause all workflows to run simultaneously. Use fuzzy schedules instead to distribute load.
:::

### Daily at Fixed Time

```yaml
on:
  schedule: daily at 02:00
```

**Output**: `0 2 * * *` (2:00 AM UTC every day)

```yaml
on:
  schedule: daily at midnight
```

**Output**: `0 0 * * *` (midnight UTC)

```yaml
on:
  schedule: daily at 3pm
```

**Output**: `0 15 * * *` (3:00 PM UTC)

### Weekly at Fixed Time

```yaml
on:
  schedule: weekly on monday at 06:30
```

**Output**: `30 6 * * 1` (Monday 6:30 AM UTC)

```yaml
on:
  schedule: weekly on friday at 5pm
```

**Output**: `0 17 * * 5` (Friday 5:00 PM UTC)

## Monthly Schedules

Monthly schedules run on a specific day of the month:

```yaml
on:
  schedule: monthly on 15
```

**Output**: `0 0 15 * *` (15th at midnight UTC)

```yaml
on:
  schedule: monthly on 1 at 9am
```

**Output**: `0 9 1 * *` (1st at 9:00 AM UTC)

```yaml
on:
  schedule: monthly on 15 at 09:00
```

**Output**: `0 9 15 * *` (15th at 9:00 AM UTC)

**Valid days**: 1-31 (note: day 31 only runs in months with 31 days)

## Interval Schedules

### Minute Intervals

Run every N minutes (minimum 5 minutes):

```yaml
on:
  schedule: every 5 minutes
```

**Output**: `*/5 * * * *`

```yaml
on:
  schedule: every 10 minutes
```

**Output**: `*/10 * * * *`

```yaml
on:
  schedule: every 30 minutes
```

**Output**: `*/30 * * * *`

**Short format**: `every 5m`, `every 10m`, `every 30m`

**Valid intervals**: `5m`, `10m`, `15m`, `20m`, `30m` (minimum 5 minutes)

:::note[Minimum Interval]
GitHub Actions enforces a minimum schedule interval of 5 minutes.
:::

### Hour Intervals

Fuzzy hour intervals (recommended):

```yaml
on:
  schedule: every 1h
```

**Output**: `FUZZY:HOURLY/1 * * *` → scatters to `58 */1 * * *`

```yaml
on:
  schedule: every 2 hours
```

**Output**: `FUZZY:HOURLY/2 * * *` → scatters to `53 */2 * * *`

Fixed hour intervals (creates load spikes):

```yaml
on:
  schedule:
    - cron: "0 */2 * * *"
```

**Output**: `0 */2 * * *` (every 2 hours at minute 0)

:::caution
Using fixed minute offsets (e.g., `0 */2 * * *`) causes all workflows to run at the same minute of each hour. Use `every 2h` instead for scattered minute offsets.
:::

### Day Intervals

```yaml
on:
  schedule: every 2 days
```

**Output**: `0 0 */2 * *` (every 2 days at midnight)

```yaml
on:
  schedule: every 1d
```

**Output**: `0 0 * * *` (same as daily at midnight)

**Short format**: `every 1d`, `every 2d`, `every 3d`

### Week Intervals

```yaml
on:
  schedule: every 1w
```

**Output**: `0 0 * * 0` (every Sunday at midnight)

```yaml
on:
  schedule: every 2w
```

**Output**: `0 0 */14 * *` (every 14 days)

### Month Intervals

```yaml
on:
  schedule: every 1mo
```

**Output**: `0 0 1 * *` (1st of every month at midnight)

```yaml
on:
  schedule: every 2mo
```

**Output**: `0 0 1 */2 *` (1st of every other month)

## Time Formats

### 24-Hour Format

```yaml
HH:MM
```

Examples: `00:00`, `09:30`, `14:00`, `23:59`

### 12-Hour Format

```yaml
Ham, Hpm
```

Examples:
- `1am` → 01:00
- `3pm` → 15:00
- `12am` → 00:00 (midnight)
- `12pm` → 12:00 (noon)
- `11pm` → 23:00

### Special Keywords

- `midnight` → `00:00`
- `noon` → `12:00`

### With UTC Offset

```yaml
TIME utc+N
TIME utc-N
TIME utc+HH:MM
```

Examples:
- `14:00 utc+9` → Converts JST to UTC
- `3pm utc-5` → Converts EST to UTC
- `9am utc+05:30` → Converts IST to UTC
- `midnight utc-8` → Converts PST to UTC

## Standard Cron Expressions

You can use standard 5-field cron expressions directly:

```yaml
on:
  schedule:
    - cron: "0 9 * * 1"
```

**Format**: `minute hour day-of-month month day-of-week`

**Examples**:
- `0 9 * * 1` - Every Monday at 9:00 AM
- `*/15 * * * *` - Every 15 minutes
- `0 0 * * *` - Daily at midnight
- `0 14 * * 1-5` - Weekdays at 2:00 PM

See [GitHub's cron syntax documentation](https://docs.github.com/en/actions/using-workflows/events-that-trigger-workflows#schedule) for complete cron format details.

## Multiple Schedules

You can specify multiple schedule triggers:

```yaml
on:
  schedule:
    - cron: daily
    - cron: weekly on monday
    - cron: monthly on 15
```

or

```yaml
on:
  schedule:
    - cron: "0 9 * * 1"
    - cron: "0 14 * * 5"
```

## Shorthand Format

Use the ultra-short format in the `on:` field:

```yaml
on: daily
```

Automatically expands to:

```yaml
on:
  schedule:
    - cron: "FUZZY:DAILY * * *"
  workflow_dispatch:
```

This shorthand adds `workflow_dispatch` for manual triggering alongside the schedule.

## Best Practices

### Recommended

✅ Use fuzzy schedules to prevent load spikes:
```yaml
on: daily
on: hourly
on: weekly on monday
on: every 2h
```

✅ Use time constraints for business hours:
```yaml
on: daily between 9:00 and 17:00
on: daily around 14:00
```

✅ Use UTC offsets for regional times:
```yaml
on: daily between 9am utc-5 and 5pm utc-5
```

### Avoid

❌ Fixed times that create load spikes:
```yaml
on: daily at midnight  # All workflows run at same time
```

❌ Hourly intervals with fixed minute:
```yaml
on:
  schedule:
    - cron: "0 */2 * * *"  # All workflows run at minute 0
```

Instead use fuzzy alternatives:
```yaml
on: daily
on: every 2h
```

## How Scattering Works

Fuzzy schedules use a deterministic hash of the workflow file path to assign each workflow a unique execution time:

1. **Workflow identifier**: Full file path (e.g., `.github/workflows/daily-report.md`)
2. **Stable hash**: FNV-1a hash algorithm (consistent across platforms)
3. **Deterministic offset**: Hash modulo time range gives consistent offset
4. **Same across recompiles**: Same workflow path always gets same scattered time

**Example**:
```yaml
on: daily
```

Workflow A: `43 5 * * *` (5:43 AM)
Workflow B: `17 14 * * *` (2:17 PM)
Workflow C: `8 20 * * *` (8:08 PM)

Each workflow gets a different time, but the same workflow always gets the same time.

## Validation & Warnings

The compiler validates schedule expressions and emits warnings for patterns that create load spikes:

```text
⚠ Schedule uses fixed daily time (0:0 UTC). Consider using fuzzy
  schedule 'daily' instead to distribute workflow execution times
  and reduce load spikes.
```

```text
⚠ Schedule uses hourly interval with fixed minute offset (0).
  Consider using fuzzy schedule 'every 2h' instead to distribute
  workflow execution times and reduce load spikes.
```

```text
⚠ Schedule uses fixed weekly time (Monday 6:30 UTC). Consider using
  fuzzy schedule 'weekly on monday' instead to distribute workflow
  execution times and reduce load spikes.
```

Fix these by using the suggested fuzzy schedules.

## Related Documentation

- [Triggers](/gh-aw/reference/triggers/) - Complete trigger configuration
- [Frontmatter](/gh-aw/reference/frontmatter/) - Workflow configuration reference
- [GitHub Actions Schedule Events](https://docs.github.com/en/actions/using-workflows/events-that-trigger-workflows#schedule) - GitHub's schedule documentation
