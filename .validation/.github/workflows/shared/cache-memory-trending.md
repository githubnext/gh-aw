---
# Cache-Memory Trending — Standard JSONL history management and trend analysis
#
# Provides the canonical pattern for workflows that collect daily metrics and store
# them in a rolling JSONL history file for trend analysis.
#
# Usage:
#   imports:
#     - uses: shared/cache-memory-trending.md
#       with:
#         workflow-name: "my-workflow"        # required — slug used in file paths
#         retention-days: 90                  # optional, default: 90
#         backfill-threshold: 30              # optional, default: 30
#         backfill-window: 90                 # optional, default: 90 (days of initial backfill)
#
# This import provides:
# - Canonical history file path under /tmp/gh-aw/cache-memory/trending/<workflow-name>/
# - Cache state inspection bash snippet to decide backfill vs. incremental collection
# - upsert_by_date Python function for last-write-wins deduplication by date
# - 90-day retention policy pattern
# - Metadata file conventions
# - Trend calculation helpers: 7-day / 30-day rolling average, percent change, anomaly detection
#
# The importing workflow is responsible for:
# - Declaring `tools: cache-memory: true` in its own frontmatter
# - Collecting and computing today's data (metric schema is workflow-specific)
# - Generating charts and creating the output discussion/issue

import-schema:
  workflow-name:
    type: string
    required: true
    description: "Workflow name slug used in history directory paths — use lowercase letters, digits, and hyphens only (e.g. api-consumption). Becomes the last segment of /tmp/gh-aw/cache-memory/trending/<slug>/"
  retention-days:
    type: integer
    default: 90
    description: "How many days of history to retain (default: 90)"
  backfill-threshold:
    type: integer
    default: 30
    description: "Entry count below which a full backfill window is used (default: 30)"
  backfill-window:
    type: integer
    default: 90
    description: "How many days to look back during an initial backfill (default: 90)"
---

## Cache-Memory Trending — Standard Pattern

Use this section to manage the rolling JSONL history and calculate trends.

### File Conventions

| Path | Purpose |
|------|---------|
| `/tmp/gh-aw/cache-memory/trending/${{ github.aw.import-inputs.workflow-name }}/history.jsonl` | One JSON object per line, each with a `date` field (YYYY-MM-DD) |
| `/tmp/gh-aw/cache-memory/trending/${{ github.aw.import-inputs.workflow-name }}/metadata.json` | Tracking metadata (entry count, date range, retention policy) |

Every history entry **must** include a `date` field in `YYYY-MM-DD` format. Use `recorded_at` (filesystem-safe, no colons: `YYYY-MM-DD-HH-MM-SS`) for traceability.

### Step T1 — Inspect Cache State and Choose Collection Window

Before collecting data, check how many history entries already exist so you can choose between a full backfill and an incremental update:

```bash
history_file="/tmp/gh-aw/cache-memory/trending/${{ github.aw.import-inputs.workflow-name }}/history.jsonl"
entry_count=0
if [ -f "$history_file" ]; then
  if ! entry_count=$(wc -l < "$history_file"); then
    echo "warning: unable to count existing history entries; defaulting to 0"
    entry_count=0
  fi
fi
echo "Existing history entries: $entry_count"
```

Choose the collection window based on `entry_count`. After the merge in Step T3 there is one deduplicated row per calendar day, so `entry_count` is a reliable proxy for the number of days of history. ${{ github.aw.import-inputs.backfill-threshold }} days is the minimum needed for stable 7-day and 30-day trend calculations:

- **If `entry_count >= ${{ github.aw.import-inputs.backfill-threshold }}`** (history is already rich): collect only incremental data — target the last 1–2 days.
- **If `entry_count < ${{ github.aw.import-inputs.backfill-threshold }}`** (first run, cache miss, or sparse history): run a one-time backfill — look back `${{ github.aw.import-inputs.backfill-window }}` days.

Record which mode you used (`incremental` vs `backfill`) and include it in the report's cache status block.

### Step T2 — Validate Restored Cache

After data collection, verify the cache state before updating the history:

```bash
history_file="/tmp/gh-aw/cache-memory/trending/${{ github.aw.import-inputs.workflow-name }}/history.jsonl"
if [ -f "$history_file" ] && [ -s "$history_file" ]; then
  entry_count=$(wc -l < "$history_file")
  echo "Cache restored from previous run: yes ($entry_count existing entries)"
else
  echo "Cache restored from previous run: no (first run or empty cache)"
fi
```

### Step T3 — Merge and Deduplicate History

Apply last-write-wins deduplication by `date` key, sort ascending, then prune to the retention window.

**Recommended Python implementation:**

```python
import json
from datetime import datetime, timedelta
from pathlib import Path

HISTORY_FILE = Path("/tmp/gh-aw/cache-memory/trending/${{ github.aw.import-inputs.workflow-name }}/history.jsonl")
RETENTION_DAYS = ${{ github.aw.import-inputs.retention-days }}

def load_history(path: Path) -> list[dict]:
    """Load existing JSONL history, returning an empty list if file is absent.

    Lines that cannot be parsed as JSON (e.g. from an interrupted prior write)
    are skipped with a warning so subsequent runs can self-heal.
    """
    entries = []
    if path.exists():
        with open(path) as f:
            for row_index, raw in enumerate(f, start=1):
                line = raw.strip()
                if not line:
                    continue
                try:
                    entries.append(json.loads(line))
                except json.JSONDecodeError as exc:
                    print(f"warning: skipped malformed JSON at row_index={row_index}: {exc}")
    return entries

def upsert_by_date(entries: list[dict]) -> list[dict]:
    """Deduplicate by date (last-write-wins), then sort ascending by date."""
    by_date: dict[str, dict] = {}
    for row_index, row in enumerate(entries, start=1):
        day = row.get("date")
        if day:
            by_date[day] = row
        else:
            print(f"warning: skipped history row without 'date' at row_index={row_index}")
    return [by_date[d] for d in sorted(by_date.keys())]

def apply_retention(entries: list[dict], retention_days: int) -> list[dict]:
    """Remove entries older than retention_days, keeping exactly retention_days calendar days.

    The cutoff is inclusive: entries whose date equals today minus (retention_days - 1) are kept,
    so a retention_days=90 policy retains exactly 90 days (today through 89 days ago).
    """
    cutoff = (datetime.utcnow().date() - timedelta(days=retention_days - 1)).isoformat()
    return [e for e in entries if e.get("date", "") >= cutoff]

def save_history(path: Path, entries: list[dict]) -> None:
    """Write entries back as JSONL using an atomic rename to prevent corruption.

    Writes to a sibling `.tmp` file first, then replaces the target via os.replace()
    so an interrupted write never leaves a truncated history file.
    """
    import os
    path.parent.mkdir(parents=True, exist_ok=True)
    tmp_path = path.with_suffix(".tmp")
    with open(tmp_path, "w") as f:
        for entry in entries:
            f.write(json.dumps(entry) + "\n")
    os.replace(tmp_path, path)

# --- Merge logic ---
existing = load_history(HISTORY_FILE)

merged = list(existing)
if mode == "backfill":
    merged.extend(backfill_entries)  # backfill_entries: list of per-day dicts
# Append today last so today's data always wins on same-date collisions.
merged.append(today_summary)        # today_summary: dict with "date" field

merged = upsert_by_date(merged)
merged = apply_retention(merged, RETENTION_DAYS)
save_history(HISTORY_FILE, merged)

print(f"History updated: {len(merged)} entries retained")
```

### Step T4 — Write Metadata File

After saving the merged history, update the companion metadata file:

```python
import json
from pathlib import Path

METADATA_FILE = Path("/tmp/gh-aw/cache-memory/trending/${{ github.aw.import-inputs.workflow-name }}/metadata.json")

metadata = {
    "workflow": "${{ github.aw.import-inputs.workflow-name }}",
    "started_tracking": merged[0]["date"] if merged else today_summary["date"],
    "last_updated": today_summary["date"],
    "data_points": len(merged),
    "retention_days": ${{ github.aw.import-inputs.retention-days }},
}
METADATA_FILE.parent.mkdir(parents=True, exist_ok=True)
METADATA_FILE.write_text(json.dumps(metadata, indent=2) + "\n")
```

### Trend Calculation Helpers

Use these helpers after loading the merged history into a Pandas DataFrame:

```python
import pandas as pd
import numpy as np

df = pd.DataFrame(merged)
df["date"] = pd.to_datetime(df["date"])
df = df.sort_values("date").reset_index(drop=True)

def rolling_avg(series: pd.Series, window: int) -> pd.Series:
    """Rolling mean with a minimum of 1 observation so early rows are never NaN."""
    return series.rolling(window=window, min_periods=1).mean()

def pct_change_period(series: pd.Series, window: int) -> float | None:
    """Percent change between the last `window` days and the prior `window` days.
    Returns None when there is insufficient data."""
    if len(series) < window * 2:
        return None
    recent = series.iloc[-window:].mean()
    prior = series.iloc[-(window * 2):-window].mean()
    if prior == 0:
        return None
    return round((recent - prior) / prior * 100, 1)

def detect_anomalies(series: pd.Series, z_threshold: float = 2.0) -> pd.Series:
    """Return a boolean Series marking entries more than z_threshold std devs from the mean."""
    mean = series.mean()
    std = series.std()
    if std == 0:
        return pd.Series([False] * len(series), index=series.index)
    return ((series - mean) / std).abs() > z_threshold

# Example: compute 7-day and 30-day rolling averages for a "value" column
df["rolling_7d"] = rolling_avg(df["value"], 7)
df["rolling_30d"] = rolling_avg(df["value"], 30)

# Percent change vs. prior period
trend_7d = pct_change_period(df["value"], 7)
trend_30d = pct_change_period(df["value"], 30)

# Anomaly flags
df["is_anomaly"] = detect_anomalies(df["value"])

# Trend direction label helper
# Changes of more than ±5% are considered significant trends (↑/↓);
# changes within ±5% are considered stable (→).
def trend_label(pct: float | None) -> str:
    """Return an arrow-prefixed label for a percent-change value.

    Args:
        pct: Percent change, or None when there is insufficient data.
    Returns:
        A human-readable string such as '↑ +12.3%', '↓ -8.0%', or '→ +1.2%'.
    """
    if pct is None:
        return "→ (insufficient data)"
    if pct > 5:
        return f"↑ {pct:+.1f}%"
    if pct < -5:
        return f"↓ {pct:+.1f}%"
    return f"→ {pct:+.1f}%"
```

### Cache Status Block

Include this block in your report's output (discussion, issue, or step summary) to aid debugging and transparency:

```markdown
<details>
<summary>📦 Cache Memory Status</summary>

- **Location**: `/tmp/gh-aw/cache-memory/trending/${{ github.aw.import-inputs.workflow-name }}/history.jsonl`
- **Cache restored from previous run**: {yes (N entries) / no (first run)}
- **Collection mode**: {incremental / backfill}
- **Data points stored**: {data_points}
- **Earliest entry**: {earliest_date}
- **Retention policy**: ${{ github.aw.import-inputs.retention-days }} days

</details>
```
