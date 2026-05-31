---
description: Guidance for adding Python data visualization to agentic workflows with compact setup patterns and links to the full trending guide.
---

# Python Data Visualization in Agentic Workflows

## Choosing a Shared Workflow

| Import | Best for |
|---|---|
| `shared/trending-charts-simple.md` | Quick setup with cache-memory-backed trend charts |
| `shared/python-dataviz.md` | One-off charts from current-run data |
| `shared/charts-with-trending.md` | Full trending analysis with richer historical guidance |

Default to `shared/trending-charts-simple.md` for new charting workflows.

If the shared files are not present locally, import them with:

```bash
gh aw add githubnext/agentics/python-dataviz
```

## Option A: Trending Charts (Simple)

Use when you need trend charts with cache-memory persistence and minimal configuration.

```yaml
tools:
  cache-memory:
    key: trending-data-${{ github.workflow }}-${{ github.run_id }}
  bash:
    - "*"
network:
  allowed:
    - defaults
    - python
steps:
  - name: Setup Python environment
    run: |
      mkdir -p /tmp/gh-aw/python/{data,charts,artifacts}
      pip install --user --quiet numpy pandas matplotlib seaborn scipy
safe-outputs:
  upload-asset:
    max: 3
    allowed-exts: [.png, .jpg, .jpeg, .svg]
```

Agent guidance:

- write data to `/tmp/gh-aw/python/data/`
- write charts to `/tmp/gh-aw/python/charts/`
- append history to `/tmp/gh-aw/cache-memory/trending/<metric>/history.jsonl`
- use ISO 8601 timestamps
- generate charts at 300 DPI with clear labels

## Option B: Current-Run Charts Only

Use when the workflow needs charts from current data without historical tracking.

```yaml
tools:
  cache-memory: true
  bash:
    - "*"
network:
  allowed:
    - defaults
    - python
safe-outputs:
  upload-asset:
    max: 3
    allowed-exts: [.png, .jpg, .jpeg, .svg]
steps:
  - name: Setup Python environment
    run: |
      mkdir -p /tmp/gh-aw/python/{data,charts,artifacts}
      pip install --user --quiet numpy pandas matplotlib seaborn scipy
```

Rules:

- never inline dataset values directly in Python code
- store input data in files and load with pandas
- keep reusable helpers in cache-memory when that improves later runs
- save chart images under `/tmp/gh-aw/python/charts/`

## Full Trending Guide

Load [charts-trending.md](charts-trending.md) only when you need:

- detailed historical-data layouts
- moving averages, comparative trends, and retention patterns
- reporting templates with embedded chart assets
- session-analysis chart patterns
