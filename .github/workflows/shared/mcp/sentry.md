---
mcp-servers:
  sentry:
    command: "npx"
    args: ["@sentry/mcp-server@0.33.0"]
    allowed:
      - whoami
      - find_organizations
      - find_teams
      - find_projects
      - find_releases
      - get_issue_details
      - get_trace_details
      - get_event_attachment
      - search_events
      - search_issues
      - list_events # fallback when search_events is unavailable (no LLM provider)
      - list_issue_events # fallback when search_issue_events is unavailable (no LLM provider)
      - find_dsns
      - analyze_issue_with_seer
      - search_docs # requires SENTRY_OPENAI_API_KEY
      - get_doc
    env:
      SENTRY_ACCESS_TOKEN: ${{ secrets.SENTRY_ACCESS_TOKEN }}
      SENTRY_HOST: ${{ env.SENTRY_HOST || 'sentry.io' }} # Optional - hostname only, not a full URL
      OPENAI_API_KEY: ${{ secrets.SENTRY_OPENAI_API_KEY }} # Optional
---

<!-- 

https://github.com/getsentry/sentry-mcp 

To utilize the stdio transport, you'll need to create an User Auth Token in Sentry with the necessary scopes. As of writing this is:

```
org:read
project:read
project:write
team:read
team:write
event:write
```
-->

Use this Sentry MCP import for read-only reliability triage with evidence-first reporting.

Preferred query loop:

1. Validate connectivity (`whoami`) and discover the target org/project.
2. Query spans first (`search_events`; fallback to `list_events` when unavailable in the MCP build).
3. Run companion checks on `errors` and `logs` datasets, even when empty, and state that result explicitly.
4. For each major finding, verify one representative trace:
   - use `get_trace_details` when available
   - otherwise use `list_events` filtered by `trace:<id>` to confirm continuity
5. Every priority finding must cite exact evidence (query scope + metric/count + trace or run link).
6. If expected fields are missing, cross-check emit-side semantics in `actions/setup/js/send_otlp_span.cjs` before proposing fixes (notably `gh-aw.workflow.name`, OTLP `status.code`, `gh-aw.run.status`, `gen_ai.response.finish_reasons`, and resource `service.version`).

Grounding rules:

- Prefer recurring patterns over one-off outliers.
- Separate confirmed failures from observability gaps when core attributes are null/missing.
- Call out unsupported tools or backend limitations in the report notes instead of silently skipping checks.
