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
    └── usage/            # Compact session and usage data
```

`run_summary.json`'s `job_details` array lists every GitHub Actions job for the run (e.g. `agent`, `detection`, `activation`), each with its own `conclusion` and a `steps[]` array of `{name, status, conclusion}`. Use this to distinguish a job that genuinely ran and failed from one that never executed (job absent or `conclusion: "skipped"`) — this is essential for correctly attributing failures to specific jobs/steps rather than guessing from token usage alone.
