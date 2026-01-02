{{if .ProjectURL}}
### Project Board Integration

Execute state writes using the `update-project` safe-output. All writes must target this exact project URL:

**Project URL**: {{.ProjectURL}}
{{if .TrackerLabel}}
**Campaign ID**: Extract from tracker label `{{.TrackerLabel}}` (format: `campaign:CAMPAIGN_ID`)
{{end}}

#### Adding New Issues/PRs

When adding an issue or PR to the project board, use the **content number** (not URL):
```
update-project:
  project: "{{.ProjectURL}}"
  content_type: "issue"  # or "pull_request"
  content_number: 123  # Extract number from URL like https://github.com/owner/repo/issues/123
{{if .TrackerLabel}}  campaign_id: "CAMPAIGN_ID"  # Required: extract from tracker label {{.TrackerLabel}}
{{end}}  fields:
    status: "Todo"  # or "Done" if issue/PR is already closed/merged
```

**How to extract content_number from URLs**:
- Issue URL: `https://github.com/owner/repo/issues/123` → `content_number: 123`, `content_type: "issue"`
- PR URL: `https://github.com/owner/repo/pull/456` → `content_number: 456`, `content_type: "pull_request"`

**Note**: If your project board has `Start Date` and `End Date` fields, these will be **automatically populated** from the issue/PR timestamps:
- `Start Date` is set from the issue's `createdAt` timestamp
- `End Date` is set from the issue's `closedAt` timestamp (if closed)

No additional configuration is needed. The dates are extracted in ISO format (YYYY-MM-DD) and only populate if the fields exist and aren't already set. This enables roadmap timeline visualization.

**Recommended Custom Fields**: To enable advanced project board features (swimlanes, "Slice by" filtering), populate these fields when available:

```
update-project:
  project: "{{.ProjectURL}}"
  content_type: "issue"  # or "pull_request"
  content_number: 123  # Extract from URL
  fields:
    status: "Todo"  # or "In Progress", "Blocked", "Done"
{{if .TrackerLabel}}    campaign_id: "CAMPAIGN_ID"  # Extract from tracker label {{.TrackerLabel}}
{{end}}    worker_workflow: "WORKFLOW_ID"  # Enables swimlane grouping and filtering
    priority: "High"  # or "Medium", "Low" - enables priority-based views
    effort: "Medium"  # or "Small", "Large" - enables capacity planning
    team: "TEAM_NAME"  # Optional: for team-based grouping
    repository: "REPO_NAME"  # Optional: for cross-repository campaigns
```

**Custom Field Benefits**:
- `worker_workflow`: Groups items by workflow in Roadmap swimlanes; enables "Slice by" filtering in Table views (orchestrator populates this by discovering which worker created the item via tracker-id)
- `priority`: Enables priority-based filtering and sorting
- `effort`: Supports capacity planning and workload distribution
- `team`: Enables team-based grouping for multi-team campaigns
- `repository`: Enables repository-based grouping for cross-repository campaigns

**Worker Workflow Agnosticism**: Worker workflows remain campaign-agnostic. The orchestrator discovers which worker created an item (via tracker-id in the issue body) and populates the `worker_workflow` field. Workers don't need to know about campaigns or custom fields.

Only populate fields that exist on your project board. Field names are case-sensitive and should match exactly as configured in GitHub Projects.

#### Updating Existing Items

When updating status for an existing board item:
```
update-project:
  project: "{{.ProjectURL}}"
  content_type: "issue"  # or "pull_request"
  content_number: 123  # Extract from URL
{{if .TrackerLabel}}  campaign_id: "CAMPAIGN_ID"  # Required: extract from tracker label {{.TrackerLabel}}
{{end}}  fields:
    status: "Done"  # or "In Progress", "Todo"
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
