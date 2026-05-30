---
emoji: "🌊"
name: DataFlow PR & Discussion Dataset Builder
description: Builds cleaned, deduplicated, quality-scored datasets from GitHub discussions and PRs using OpenDCAI/DataFlow text processing pipelines
on:
  schedule: weekly
  workflow_dispatch:
permissions:
  contents: read
  discussions: read
  pull-requests: read
  issues: read
network:
  allowed:
    - defaults
    - python
    - github-actions
imports:
  - shared/pmg.md
  - uses: shared/discussions-data-fetch.md
  - uses: shared/repo-memory-standard.md
    with:
      branch-name: memory/dataflow-dataset
      description: Tracks dataset build statistics and run metadata for the DataFlow pipeline
  - shared/reporting.md
  - shared/otlp.md
tools:
  cli-proxy: true
  github:
    mode: gh-proxy
    min-integrity: approved
    toolsets:
      - default
      - pull_requests
      - discussions
steps:
  - name: Install DataFlow
    run: |
      python3 -m venv /tmp/gh-aw/agent/venv
      /tmp/gh-aw/agent/venv/bin/pip install --quiet open-dataflow
      /tmp/gh-aw/agent/venv/bin/python3 -c "
      import dataflow
      print('DataFlow', getattr(dataflow, '__version__', 'installed'), 'ready')
      # Print available operators for reference
      import pkgutil, dataflow.operators as ops
      available = [m.name for m in pkgutil.iter_modules(ops.__path__)]
      print('Operator modules:', available)
      "
      mkdir -p /tmp/gh-aw/agent/dataflow/{input,output,pipeline,reports}

  - name: Fetch merged PRs
    env:
      GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      bash "${RUNNER_TEMP}/gh-aw/actions/install_gh_cli.sh"

      # Fetch up to 500 merged PRs — title, body, metadata
      gh pr list \
        --repo "$GITHUB_REPOSITORY" \
        --state merged \
        --limit 500 \
        --json number,title,body,createdAt,mergedAt,url,author,labels \
        > /tmp/gh-aw/agent/dataflow/input/prs.json

      echo "Fetched $(jq 'length' /tmp/gh-aw/agent/dataflow/input/prs.json) merged PRs"

safe-outputs:
  upload-artifact:
    max-uploads: 3
    retention-days: 30
    skip-archive: false
  create-discussion:
    expires: 7d
    category: "reports"
    max: 1
    close-older-discussions: true
    title-prefix: "[dataflow-dataset] "
  messages:
    footer: "> 🌊 *Dataset built by [{workflow_name}]({run_url})*{effective_tokens_suffix}{history_link}"
    run-started: "🌊 DataFlow Dataset Builder starting! [{workflow_name}]({run_url}) is processing discussions and PRs with OpenDCAI/DataFlow..."
    run-success: "✅ DataFlow dataset ready! [{workflow_name}]({run_url}) produced a cleaned, deduplicated dataset. Artifacts uploaded. 📊"
    run-failure: "⚠️ DataFlow pipeline failed! [{workflow_name}]({run_url}) {status}. Check the run logs."
timeout-minutes: 30
tracker-id: dataflow-pr-discussion-dataset
strict: true
---
# DataFlow PR & Discussion Dataset Builder

You are a data pipeline agent that uses [OpenDCAI/DataFlow](https://github.com/OpenDCAI/DataFlow) to process GitHub discussions and pull requests into high-quality, deduplicated datasets suitable for LLM training and analysis.

## Mission

Build a cleaned, quality-scored, and deduplicated JSONL dataset from this repository's discussions and PRs, then post a summary report as a GitHub Discussion.

## Current Context

- **Repository**: ${{ github.repository }}
- **Run ID**: ${{ github.run_id }}
- **Data available**:
  - Discussions: `/tmp/gh-aw/agent/discussions-data/discussions.json` (pre-fetched by shared component)
  - PRs: `/tmp/gh-aw/agent/dataflow/input/prs.json` (pre-fetched in `steps:`)
- **DataFlow venv**: `/tmp/gh-aw/agent/venv/bin/python3`
- **Output dir**: `/tmp/gh-aw/agent/dataflow/output/`

## Pipeline Overview

```
GitHub Discussions + PRs
         │
         ▼
  ┌─────────────┐
  │ Normalise   │  Convert to unified JSONL (title + body = "text", metadata preserved)
  └──────┬──────┘
         │
         ▼
  ┌─────────────┐
  │ DataFlow    │  Text length filter → alpha-ratio filter → stop-word filter
  │ Filters     │
  └──────┬──────┘
         │
         ▼
  ┌─────────────┐
  │ DataFlow    │  Near-duplicate removal (MinHash or exact-hash)
  │ Dedup       │
  └──────┬──────┘
         │
         ▼
  ┌─────────────┐
  │ Output      │  Clean JSONL + stats report
  └─────────────┘
```

## Step-by-Step Instructions

### Step 1: Inspect Available DataFlow Operators

Before building the pipeline, discover which operators are installed:

```bash
/tmp/gh-aw/agent/venv/bin/python3 -c "
import pkgutil, dataflow.operators as ops
for m in pkgutil.iter_modules(ops.__path__):
    print(m.name)
"
```

Then list classes in the `filter` and `dedup` sub-modules (if present):

```bash
/tmp/gh-aw/agent/venv/bin/python3 -c "
import inspect
try:
    import dataflow.operators.filter as f
    print('filter:', [n for n, _ in inspect.getmembers(f, inspect.isclass)])
except Exception as e:
    print('filter module error:', e)

try:
    import dataflow.operators.dedup as d
    print('dedup:', [n for n, _ in inspect.getmembers(d, inspect.isclass)])
except Exception as e:
    print('dedup module error:', e)
"
```

Use the discovered class names throughout the pipeline below.

### Step 2: Normalise Raw Data into JSONL

Convert both discussions and PRs into a unified JSONL format with a `text` field that
DataFlow operators will read.

Write a Python script `/tmp/gh-aw/agent/dataflow/pipeline/01_normalise.py`:

```python
#!/usr/bin/env python3
"""Normalise discussions and PRs into a single JSONL file for DataFlow."""

import json
import sys
from pathlib import Path

OUT = Path("/tmp/gh-aw/agent/dataflow/input/combined_raw.jsonl")
records = []

# ── Discussions ───────────────────────────────────────────────────────────────
disc_path = Path("/tmp/gh-aw/agent/discussions-data/discussions.json")
if disc_path.exists():
    discussions = json.loads(disc_path.read_text())
    for d in discussions:
        title = (d.get("title") or "").strip()
        body = (d.get("body") or "").strip()
        text = f"{title}\n\n{body}".strip() if body else title
        if text:
            records.append({
                "id": f"discussion-{d.get('number', '')}",
                "source": "discussion",
                "text": text,
                "title": title,
                "url": d.get("url", ""),
                "author": d.get("author", ""),
                "created_at": d.get("createdAt", ""),
                "category": d.get("category", ""),
                "labels": d.get("labels", []),
            })
    print(f"Loaded {len(discussions)} discussions → {sum(1 for r in records if r['source']=='discussion')} with text")

# ── Pull Requests ─────────────────────────────────────────────────────────────
pr_path = Path("/tmp/gh-aw/agent/dataflow/input/prs.json")
if pr_path.exists():
    prs = json.loads(pr_path.read_text())
    pr_count_before = len(records)
    for p in prs:
        title = (p.get("title") or "").strip()
        body = (p.get("body") or "").strip()
        text = f"{title}\n\n{body}".strip() if body else title
        author_obj = p.get("author") or {}
        if text:
            records.append({
                "id": f"pr-{p.get('number', '')}",
                "source": "pull_request",
                "text": text,
                "title": title,
                "url": p.get("url", ""),
                "author": author_obj.get("login", "") if isinstance(author_obj, dict) else str(author_obj),
                "created_at": p.get("createdAt", ""),
                "merged_at": p.get("mergedAt", ""),
                "labels": [lb.get("name","") for lb in (p.get("labels") or [])],
            })
    print(f"Loaded {len(prs)} PRs → {len(records) - pr_count_before} with text")

# Write unified JSONL
OUT.parent.mkdir(parents=True, exist_ok=True)
with OUT.open("w") as fh:
    for r in records:
        fh.write(json.dumps(r, ensure_ascii=False) + "\n")

print(f"Total records written: {len(records)} → {OUT}")
```

Run it:

```bash
/tmp/gh-aw/agent/venv/bin/python3 /tmp/gh-aw/agent/dataflow/pipeline/01_normalise.py
```

### Step 3: Build and Run the DataFlow Pipeline

Write `/tmp/gh-aw/agent/dataflow/pipeline/02_pipeline.py`:

```python
#!/usr/bin/env python3
"""
DataFlow text processing pipeline:
  1. Load JSONL into FileStorage
  2. Text length filter   (heuristic — no LLM required)
  3. Alpha-ratio filter   (heuristic — no LLM required)
  4. Near-duplicate removal (MinHash or exact-hash — no LLM required)
  5. Save clean output
"""

import json, sys, inspect, traceback
from pathlib import Path

INPUT  = "/tmp/gh-aw/agent/dataflow/input/combined_raw.jsonl"
OUTPUT = "/tmp/gh-aw/agent/dataflow/output/dataset_clean.jsonl"
STATS  = "/tmp/gh-aw/agent/dataflow/output/pipeline_stats.json"

Path("/tmp/gh-aw/agent/dataflow/output").mkdir(parents=True, exist_ok=True)

# ── Load DataFlow storage ─────────────────────────────────────────────────────
try:
    from dataflow.utils.storage import FileStorage
    storage = FileStorage(first_entry_file_name=INPUT)
    print(f"FileStorage loaded with {len(storage)} records")
except Exception as e:
    print(f"FileStorage error: {e} — falling back to raw JSONL processing")
    storage = None

stats = {
    "input_count": 0,
    "after_length_filter": 0,
    "after_alpha_filter": 0,
    "after_dedup": 0,
    "operators_used": [],
    "fallback_mode": storage is None,
}

# ── Helper: count records in a JSONL file ────────────────────────────────────
def count_jsonl(path: str) -> int:
    try:
        return sum(1 for _ in open(path))
    except FileNotFoundError:
        return 0

# ── If FileStorage is available, attempt DataFlow operators ──────────────────
if storage is not None:
    stats["input_count"] = len(storage)
    current_file = INPUT

    # ── 1. Text length filter ─────────────────────────────────────────────────
    try:
        from dataflow.operators.filter import TextLengthFilter
        op = TextLengthFilter(min_len=50, max_len=100_000)
        op.run(storage=storage.step(), input_key="text")
        stats["after_length_filter"] = len(storage)
        stats["operators_used"].append("TextLengthFilter")
        print(f"After TextLengthFilter: {stats['after_length_filter']} records")
    except Exception as e:
        print(f"TextLengthFilter skipped: {e}")
        stats["after_length_filter"] = len(storage)

    # ── 2. Alpha-ratio filter ─────────────────────────────────────────────────
    try:
        from dataflow.operators.filter import AlphaRatioFilter
        op = AlphaRatioFilter(min_ratio=0.25)
        op.run(storage=storage.step(), input_key="text")
        stats["after_alpha_filter"] = len(storage)
        stats["operators_used"].append("AlphaRatioFilter")
        print(f"After AlphaRatioFilter: {stats['after_alpha_filter']} records")
    except Exception as e:
        print(f"AlphaRatioFilter skipped: {e}")
        stats["after_alpha_filter"] = len(storage)

    # ── 3. Near-duplicate removal ─────────────────────────────────────────────
    for module_path, class_name, kwargs in [
        ("dataflow.operators.dedup", "MinHashDeduplicator", {"threshold": 0.85}),
        ("dataflow.operators.dedup", "HashDeduplicator",    {}),
    ]:
        try:
            mod = __import__(module_path, fromlist=[class_name])
            cls = getattr(mod, class_name)
            op = cls(**kwargs)
            op.run(storage=storage.step(), input_key="text")
            stats["after_dedup"] = len(storage)
            stats["operators_used"].append(class_name)
            print(f"After {class_name}: {stats['after_dedup']} records")
            break
        except Exception as e:
            print(f"{class_name} skipped: {e}")
    else:
        stats["after_dedup"] = len(storage)

    # ── Save output ───────────────────────────────────────────────────────────
    try:
        storage.save(OUTPUT)
        print(f"DataFlow output saved to {OUTPUT}")
    except Exception as e:
        # Fallback: write current data directly
        print(f"storage.save failed ({e}), writing manually")
        records = list(storage)
        with open(OUTPUT, "w") as fh:
            for r in records:
                fh.write(json.dumps(r, ensure_ascii=False) + "\n")

# ── Fallback: lightweight Python pipeline (no DataFlow operators) ─────────────
else:
    print("Running fallback pipeline (pure Python)...")
    import unicodedata, re

    def alpha_ratio(text: str) -> float:
        letters = sum(1 for c in text if c.isalpha())
        return letters / max(len(text), 1)

    seen_hashes: set = set()
    kept: list = []

    with open(INPUT) as fh:
        raw_records = [json.loads(line) for line in fh if line.strip()]

    stats["input_count"] = len(raw_records)

    for r in raw_records:
        text = r.get("text", "")
        # Length filter
        if not (50 <= len(text) <= 100_000):
            continue
        stats["after_length_filter"] = stats.get("after_length_filter", 0) + 1
        # Alpha-ratio filter
        if alpha_ratio(text) < 0.25:
            continue
        stats["after_alpha_filter"] = stats.get("after_alpha_filter", 0) + 1
        # Exact dedup
        h = hash(text[:500])
        if h in seen_hashes:
            continue
        seen_hashes.add(h)
        kept.append(r)

    stats["after_dedup"] = len(kept)
    stats["operators_used"].append("fallback_python_pipeline")

    with open(OUTPUT, "w") as fh:
        for r in kept:
            fh.write(json.dumps(r, ensure_ascii=False) + "\n")
    print(f"Fallback pipeline output: {len(kept)} records → {OUTPUT}")

# ── Write stats ───────────────────────────────────────────────────────────────
Path(STATS).write_text(json.dumps(stats, indent=2))
print(json.dumps(stats, indent=2))
```

Run it:

```bash
/tmp/gh-aw/agent/venv/bin/python3 /tmp/gh-aw/agent/dataflow/pipeline/02_pipeline.py
```

Verify output:

```bash
echo "Output records: $(wc -l < /tmp/gh-aw/agent/dataflow/output/dataset_clean.jsonl)"
cat /tmp/gh-aw/agent/dataflow/output/pipeline_stats.json
```

### Step 4: Upload Dataset Artifact

Stage the output file and upload it as a workflow artifact:

```bash
# Stage for upload
mkdir -p "$RUNNER_TEMP/gh-aw/safeoutputs/upload-artifacts"
cp /tmp/gh-aw/agent/dataflow/output/dataset_clean.jsonl \
   "$RUNNER_TEMP/gh-aw/safeoutputs/upload-artifacts/dataset_clean.jsonl"
```

Then call the `upload_artifact` safe-output tool:

```json
{
  "type": "upload_artifact",
  "path": "dataset_clean.jsonl"
}
```

Record the returned artifact URL for use in the discussion report.

### Step 5: Update Repo-Memory

Save pipeline statistics for trend tracking across runs:

```bash
DATE=$(date '+%Y-%m-%d')
RUN_ID="${GITHUB_RUN_ID}"
STATS=$(cat /tmp/gh-aw/agent/dataflow/output/pipeline_stats.json)

# Load existing history (or start fresh)
HISTORY_FILE="/tmp/gh-aw/repo-memory/default/dataflow-runs.jsonl"
mkdir -p "$(dirname "$HISTORY_FILE")"
touch "$HISTORY_FILE"

# Append this run
python3 -c "
import json, sys, os
entry = {
    'date': '$DATE',
    'run_id': '$RUN_ID',
    **json.loads('''$STATS''')
}
with open('$HISTORY_FILE', 'a') as fh:
    fh.write(json.dumps(entry) + '\n')
print('Run appended to history')
"
```

### Step 6: Compute Quality Breakdown

Read the clean output and compute a per-source breakdown:

```bash
/tmp/gh-aw/agent/venv/bin/python3 - << 'EOF'
import json
from collections import Counter
from pathlib import Path

records = [json.loads(l) for l in open("/tmp/gh-aw/agent/dataflow/output/dataset_clean.jsonl")]
stats   = json.loads(Path("/tmp/gh-aw/agent/dataflow/output/pipeline_stats.json").read_text())

by_source = Counter(r.get("source", "unknown") for r in records)
avg_len   = sum(len(r.get("text", "")) for r in records) / max(len(records), 1)

report = {
    "total_clean_records": len(records),
    "by_source": dict(by_source),
    "avg_text_length_chars": round(avg_len, 1),
    "operators_used": stats.get("operators_used", []),
    "input_count": stats.get("input_count", 0),
    "retention_rate_pct": round(len(records) / max(stats.get("input_count", 1), 1) * 100, 1),
}

Path("/tmp/gh-aw/agent/dataflow/reports/quality_breakdown.json").write_text(json.dumps(report, indent=2))
print(json.dumps(report, indent=2))
EOF
```

### Step 7: Post Discussion Report

Build the full report body in Python (do not use shell variables for the discussion body) and post it with the `create_discussion` safe output.

Read the quality breakdown and artifact URL from files, then construct the discussion:

```python
import json
from pathlib import Path

quality = json.loads(Path("/tmp/gh-aw/agent/dataflow/reports/quality_breakdown.json").read_text())
stats   = json.loads(Path("/tmp/gh-aw/agent/dataflow/output/pipeline_stats.json").read_text())

# Read artifact URL saved after upload_artifact call
artifact_url = ""
try:
    artifact_url = Path("/tmp/gh-aw/agent/url-dataset-artifact.txt").read_text().strip()
except FileNotFoundError:
    pass

run_id   = "${{ github.run_id }}"
repo     = "${{ github.repository }}"
run_url  = f"${{ github.server_url }}/{repo}/actions/runs/{run_id}"
date_str = __import__('datetime').date.today().isoformat()

by_source_rows = "\n".join(
    f"| {src} | {cnt} |"
    for src, cnt in quality.get("by_source", {}).items()
)

artifact_section = ""
if artifact_url:
    artifact_section = f"\n### 📦 Dataset Artifact\n\n[Download dataset_clean.jsonl]({artifact_url})\n"

operators_str = ", ".join(quality.get("operators_used", ["none"])) or "none"

body = f"""### Summary

Built a cleaned, deduplicated dataset from GitHub discussions and PRs using [OpenDCAI/DataFlow](https://github.com/OpenDCAI/DataFlow).

| Metric | Value |
|--------|-------|
| Input records | {quality.get("input_count", 0):,} |
| Output (clean) records | {quality.get("total_clean_records", 0):,} |
| Retention rate | {quality.get("retention_rate_pct", 0)}% |
| Average text length | {quality.get("avg_text_length_chars", 0):,.0f} chars |
| DataFlow operators | `{operators_str}` |

### Records by Source

| Source | Count |
|--------|-------|
{by_source_rows}

### DataFlow Pipeline

The pipeline applied these processing stages:

1. **Normalise** — Merged discussions and PRs into unified JSONL (`title + body` → `text`)
2. **Text length filter** — Kept records with 50–100,000 characters
3. **Alpha-ratio filter** — Removed records with fewer than 25% alphabetic characters
4. **Near-duplicate removal** — Eliminated near-identical records using MinHash or exact-hash
{artifact_section}
### Pipeline Configuration

```yaml
DataFlow package: open-dataflow (PyPI)
Source: https://github.com/OpenDCAI/DataFlow
Input: GitHub Discussions + merged PRs from {repo}
Output: JSONL — one record per item, text field for LLM use
```

<details>
<summary>Raw pipeline statistics</summary>

```json
{json.dumps(stats, indent=2)}
```

</details>

---

*Generated by DataFlow PR & Discussion Dataset Builder — Run [#{run_id}]({run_url}) on {date_str}*
"""

Path("/tmp/gh-aw/agent/discussion_body.md").parent.mkdir(parents=True, exist_ok=True)
Path("/tmp/gh-aw/agent/discussion_body.md").write_text(body)
print("Discussion body written")
print(body[:500])
```

Then emit:

```json
{
  "type": "create_discussion",
  "title": "🌊 DataFlow Dataset Build Report — [DATE]",
  "body": "[contents of /tmp/gh-aw/agent/discussion_body.md]",
  "category": "reports"
}
```

Replace `[DATE]` with today's ISO date and `[contents of ...]` with the actual text read
from `/tmp/gh-aw/agent/discussion_body.md`.

## Success Criteria

- ✅ Both discussions and PRs loaded and normalised into unified JSONL
- ✅ DataFlow pipeline applied (with graceful fallback if operator API changes)
- ✅ Clean JSONL artifact uploaded
- ✅ Quality breakdown computed (input/output counts, retention rate, source split)
- ✅ Discussion posted in `reports` category with full pipeline stats
- ✅ Run metadata appended to repo-memory for trend tracking

## Edge Cases

### DataFlow API Changes
DataFlow is actively developed — operator class names may change between releases.
Always `import inspect` and list class names before use; adapt the import paths at runtime.

### No Data Available
If both input files are empty or missing, post a short discussion noting:
"No discussions or PRs found to process in this run."

### Insufficient Text
If all records are filtered out by the length or alpha-ratio filter, relax thresholds:
`min_len=20`, `min_ratio=0.1`, and skip dedup. Note the relaxed thresholds in the report.

## Usage

To trigger this workflow on demand:

```bash
gh aw run dataflow-pr-discussion-dataset
```

The resulting JSONL dataset is suitable for:
- **LLM fine-tuning**: Supervised fine-tuning (SFT) datasets from real developer discussions
- **RAG indexing**: Embedding-ready clean text chunks from the repository's knowledge base
- **Analytics**: Deduplicated corpus for topic modelling, sentiment analysis, clustering

{{#runtime-import shared/noop-reminder.md}}
