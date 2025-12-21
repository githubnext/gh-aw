## Campaign Launcher Rules

This launcher maintains the campaign dashboard by ensuring the GitHub Project stays in sync with the campaign's tracker label.

### Traffic and rate limits (required)

- Minimize API calls: avoid full scans when possible and avoid repeated reads of the same data in a single run.
- Prefer incremental processing: sort deterministically (e.g., by updated time) and process a bounded slice each run.
- Use strict pagination budgets: if a query would require many pages, stop early and continue next run.
- Use a durable cursor/checkpoint: persist the last processed boundary (e.g., updatedAt cutoff + last seen ID) so the next run can continue without rescanning.
- On throttling (HTTP 429 / rate limit 403), do not retry aggressively. Use backoff and end the run after reporting what remains.

{{ if .CursorGlob }}
**Cursor file (repo-memory)**: `{{ .CursorGlob }}`
{{ end }}
{{ if gt .MaxDiscoveryItemsPerRun 0 }}
**Read budget**: max discovery items per run: {{ .MaxDiscoveryItemsPerRun }}
{{ end }}
{{ if gt .MaxDiscoveryPagesPerRun 0 }}
**Read budget**: max discovery pages per run: {{ .MaxDiscoveryPagesPerRun }}
{{ end }}

### What to do each run

1. **Discover candidate items**
   - Find open and closed issues and pull requests that match the campaign's `tracker-label`.
   - If `governance.opt-out-labels` is configured, ignore any item that has one of those labels.

2. **Decide additions (pacing)**
   - If `governance.max-new-items-per-run` is set, add at most that many new items to the Project this run.
   - Prefer adding the oldest (or least recently updated) missing items first to keep execution stable.

3. **Decide updates (no downgrade)**
   - Keep Project item status consistent with the issue/PR state.
   - If `governance.do-not-downgrade-done-items` is true, do not move items from a Done-like status back into an active status.

4. **Write changes**
   - Use `update-project` safe outputs for adds/updates.
   - Use `add-comment` safe outputs only for concise status notes (bounded by configured maxima).

5. **Report**
   - Summarize what was added/updated and what was skipped (and why).
