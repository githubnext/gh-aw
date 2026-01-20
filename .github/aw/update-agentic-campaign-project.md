{{if .ProjectURL}}
# Project Update Instructions (Authoritative Write Contract)

Defines ONLY allowed rules for writing to project board. THIS FILE TAKES PRECEDENCE.

## Hard Requirements
- Use `update-project` safe-output only
- Target: `{{.ProjectURL}}`
- Every item includes: `campaign_id: "{{.CampaignID}}"`

## Project Fields (Must Exist)
| Field | Type | Values |
|---|---|---|
| status | single-select | Todo / In Progress / Review required / Blocked / Done |
| campaign_id | text | {{.CampaignID}} |
| worker_workflow | text | workflow-id or "unknown" |
| repository | text | owner/repo |
| priority | single-select | High / Medium / Low |
| size | single-select | Small / Medium / Large |
| start_date | date | YYYY-MM-DD |
| end_date | date | YYYY-MM-DD |

## Rules
**Content ID**: Use content_number (integer), not URL. Issue 123 = `content_type: "issue"`, `content_number: 123`

**Field Defaults**:
- campaign_id: always `{{.CampaignID}}`
- worker_workflow: ID if known, else `"unknown"`
- repository: extract from URL as `owner/repo`
- priority/size: default `Medium`
- start_date: `created_at` as YYYY-MM-DD
- end_date: `closed_at`/`merged_at` if closed, else today (YYYY-MM-DD)

**Read-Write Separation**: Read all state first, then write. Never interleave.

**Adding Items** (first write): ALL required fields
```yaml
update-project:
  project: "{{.ProjectURL}}"
  campaign_id: "{{.CampaignID}}"
  content_type: "issue"  # or "pull_request"
  content_number: 123
  fields:
    status: "Todo"  # "Done" if closed
    campaign_id: "{{.CampaignID}}"
    worker_workflow: "unknown"
    repository: "owner/repo"
    priority: "Medium"
    size: "Medium"
    start_date: "2025-12-15"
    end_date: "2026-01-03"
```

**Updating Items**: Status-only unless backfill repair needed
```yaml
update-project:
  project: "{{.ProjectURL}}"
  campaign_id: "{{.CampaignID}}"
  content_type: "issue"
  content_number: 123
  fields:
    status: "Done"
```

**Idempotency**: Trust safe-output deduplication. Matching status = no-op, different status = update, invalid item = log & continue.

**Logging**: Record content_type, content_number, repository, action (noop/add/status_update/backfill/failed), errors. Don't stop on failures.

**Epic/Hierarchy**: One Epic issue per board with campaign_id, worker_workflow `"unknown"`. Work issues as sub-issues. PRs link via "Closes #123".

{{end}}
