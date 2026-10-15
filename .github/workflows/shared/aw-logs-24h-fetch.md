---
# Pre-fetch last 24 hours of agentic workflow logs for analysis
# Saves logs to /tmp/gh-aw/aw-mcp/logs/
#
# NOTE: --count defaults to 10 and is applied *in addition to* --start-date
# (it caps the number of matching runs returned, not just how far back to
# look). On a high-volume fleet the default silently truncates the "24h"
# window to only the most recent handful of runs, so an explicit high
# --count is required here to actually cover the full 24h window. 3000 is
# intentionally generous (the fleet has been observed running 100+ runs/hour,
# i.e. up to ~2400+/day) to keep headroom above expected peak daily volume;
# if the fleet ever exceeds this, the CLI reports a truncation/continuation
# signal that should be surfaced rather than silently dropped.
# Fetch compact metadata and usage, not activation snapshots. Limit storage to 1 GB.

tools:
  agentic-workflows:
  cache-memory: true
  timeout: 300

steps:
  - name: Download logs from last 24 hours
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: ./gh-aw logs --start-date -1d --count 3000 --artifacts info,usage --audit --max-storage 1024 --prune-older-runs -o /tmp/gh-aw/aw-mcp/logs
---

## Agentic Workflow Logs (Last 24h)

Compact workflow metadata and usage have been pre-downloaded to `/tmp/gh-aw/aw-mcp/logs/` with a 1 GB storage budget. If collection reaches a storage or count limit, report the analyzed subset and incomplete coverage; do not claim the entire 24-hour window was analyzed.

**IMPORTANT**: Do NOT run `./gh-aw` or `gh aw` CLI commands directly — the binary is not authenticated in the agent environment. Use the `agentic-workflows` MCP server tools (`status`, `logs`, `audit`) instead for all additional queries.

Project `summary`, `failure_investigation`, and `continuation` from `/tmp/gh-aw/aw-mcp/logs/summary.json` first; do not read every run. Start with `failure_investigation.selected_runs`, a deduplicated initial audit plan, and read their cached `audit_path`/`logs_path` before fetching more evidence. Use selected cluster IDs to read `failure_clusters`: prioritize specific `failure_boundary` signals over generic `aftermath`, and use typed `signal.job`/`signal.step`. Clusters overlap, so count unique run IDs rather than summing counts. Membership describes symptoms, not proven causes; intentional failures are excluded from recommendations and `metadata_only` groups remain unknown. Follow every continuation parameter, including the updated end date, through the MCP tools or state incomplete coverage. Report inspected run URLs and remaining uncertainty.

### Log Directory Structure

```
/tmp/gh-aw/aw-mcp/logs/
└── run-(id)/             # One directory per workflow run
    ├── aw_info.json      # Run metadata (engine, workflow, status, tokens)
    ├── run_summary.json  # Per-job/step conclusions (job_details[].name/conclusion/steps[])
    ├── audit.json        # Per-run audit report built from downloaded data
    └── usage/            # Compact session and usage data
```

`run_summary.json`'s `job_details` array lists every GitHub Actions job for the run (e.g. `agent`, `detection`, `activation`), each with its own `conclusion` and a `steps[]` array of `{name, status, conclusion}`. Use this to distinguish a job that genuinely ran and failed from one that never executed (job absent or `conclusion: "skipped"`) — this is essential for correctly attributing failures to specific jobs/steps rather than guessing from token usage alone.
