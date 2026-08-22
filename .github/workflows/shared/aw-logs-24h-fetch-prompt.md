## Agentic Workflow Logs (Last 24h)

Workflow logs have been pre-downloaded to `/tmp/gh-aw/aw-mcp/logs/`.

**IMPORTANT**: Do NOT run `./gh-aw` or `gh aw` CLI commands directly — the binary is not authenticated in the agent environment. Use the `agentic-workflows` MCP server tools (`status`, `logs`, `audit`) instead for all additional queries.

### Log Directory Structure

```
/tmp/gh-aw/aw-mcp/logs/
└── run-(id)/             # One directory per workflow run
    ├── aw_info.json      # Run metadata (engine, workflow, status, tokens)
    ├── run_summary.json  # Per-job/step conclusions (job_details[].name/conclusion/steps[])
    ├── activation/       # Activation job logs
    └── agent/            # Agent job logs
```

`run_summary.json`'s `job_details` array lists every GitHub Actions job for the run (e.g. `agent`, `detection`, `activation`), each with its own `conclusion` and a `steps[]` array of `{name, status, conclusion}`. Use this to distinguish a job that genuinely ran and failed from one that never executed (job absent or `conclusion: "skipped"`) — this is essential for correctly attributing failures to specific jobs/steps rather than guessing from token usage alone.
