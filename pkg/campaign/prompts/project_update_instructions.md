{{if .ProjectURL}}
### Project Board Integration

Execute state writes using the `update-project` safe-output. All writes must target this exact project URL:

**Project URL**: {{.ProjectURL}}

#### Required project fields (must exist)

Your GitHub Project **must** have these fields configured. Do not attempt partial updates.

- `status` (single-select)
- `campaign_id` (text)
- `worker_workflow` (text)
- `repository` (text)
- `priority` (single-select: "High", "Medium", "Low")
- `size` (single-select: "Small", "Medium", "Large")
- `start_date` (date)
- `end_date` (date)

**Campaign ID**: `{{.CampaignID}}` (this exact value must be written to `campaign_id` for every item)

#### Adding New Issues/PRs

When adding an issue or PR to the project board, use the **content number** (not URL):
```
update-project:
  project: "{{.ProjectURL}}"
  content_type: "issue"  # or "pull_request"
  content_number: 123  # Extract number from URL like https://github.com/owner/repo/issues/123
  campaign_id: "{{.CampaignID}}"  # Required
  fields:
    status: "Todo"  # or "Done" if issue/PR is already closed/merged
    worker_workflow: "unknown"  # Required: use worker workflow ID when known, else "unknown"
    repository: "owner/repo"  # Required: extract from URL
    priority: "Medium"  # Required default
    size: "Medium"  # Required default
    start_date: "2026-01-03"  # Required: today's date in YYYY-MM-DD format
    end_date: "2026-01-03"  # Required: today's date in YYYY-MM-DD format
```

**How to extract content_number from URLs**:
- Issue URL: `https://github.com/owner/repo/issues/123` → `content_number: 123`, `content_type: "issue"`
- PR URL: `https://github.com/owner/repo/pull/456` → `content_number: 456`, `content_type: "pull_request"`

#### Required fields for every item

When adding or updating an item, always provide ALL required fields.

Deterministic defaults:
- `worker_workflow`: set to the worker workflow ID when the item is worker-created; otherwise set to `unknown`
- `repository`: extract `owner/repo` from the issue/PR URL
- `priority`: default to `Medium` unless explicitly known
- `size`: default to `Medium` unless explicitly known
- `start_date`: default to today's date in YYYY-MM-DD format
- `end_date`: default to today's date in YYYY-MM-DD format

```
update-project:
  project: "{{.ProjectURL}}"
  content_type: "issue"  # or "pull_request"
  content_number: 123  # Extract from URL
  fields:
    status: "Todo"  # or "In Progress", "Done"
    campaign_id: "{{.CampaignID}}"  # Required
    worker_workflow: "WORKFLOW_ID"  # Required (or "unknown" when not known)
    repository: "owner/repo"  # Required
    priority: "High"  # or "Medium", "Low"
    size: "Medium"  # or "Small", "Large"
    start_date: "2026-01-03"  # Required: YYYY-MM-DD format
    end_date: "2026-01-03"  # Required: YYYY-MM-DD format
```

**Field semantics**:
- `worker_workflow`: Enables swimlane grouping and filtering; use the worker workflow ID when known
- `repository`: Enables cross-repo views and grouping
- `priority`: Enables priority-based filtering and sorting
- `size`: Supports capacity planning and workload distribution
- `start_date`: Required for roadmap view; tracks when work begins
- `end_date`: Required for roadmap view; tracks when work completes

**Worker Workflow Agnosticism**: Worker workflows remain campaign-agnostic. The orchestrator discovers which worker created an item (via tracker-id in the issue body) and populates the `worker_workflow` field. Workers don't need to know about campaigns or custom fields.

Field names are case-sensitive and must match exactly as configured in GitHub Projects.

#### Updating Existing Items

When updating status for an existing board item:
```
update-project:
  project: "{{.ProjectURL}}"
  content_type: "issue"  # or "pull_request"
  content_number: 123  # Extract from URL
  campaign_id: "{{.CampaignID}}"  # Required
  fields:
    status: "Done"  # or "In Progress", "Todo"
    worker_workflow: "WORKFLOW_ID"  # Required (or "unknown")
    repository: "owner/repo"  # Required
    priority: "Medium"  # Required
    size: "Medium"  # Required
    start_date: "2026-01-03"  # Required: YYYY-MM-DD format
    end_date: "2026-01-03"  # Required: YYYY-MM-DD format
```

#### Idempotency

- If an issue/PR is already on the board with matching status → Skip (no-op)
- If an issue/PR is already on the board with different status → Update status field only
- If an issue/PR URL is invalid or deleted → Record failure, continue with remaining items

#### Write Operation Rules

1. **Batch writes separately** - Do not mix reads and writes in the same operation
2. **Validate before writing** - Confirm issue/PR URL exists and is accessible
3. **Record all outcomes** - Log success/failure for each write operation
4. **Never infer state** - Only update based on explicit issue/PR state (open/closed/merged)
5. **Fail gracefully** - If a write fails, record error and continue with remaining operations
{{end}}
