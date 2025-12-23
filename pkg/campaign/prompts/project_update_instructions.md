{{if .ProjectURL}}
### Project Board Integration

Execute state writes using the `update-project` safe-output. All writes must target this exact project URL:

**Project URL**: {{.ProjectURL}}
{{if .TrackerLabel}}
**Campaign ID**: Extract from tracker label `{{.TrackerLabel}}` (format: `campaign:CAMPAIGN_ID`)
{{end}}

#### Adding New Issues

When adding an issue to the project board:
```
update-project:
  project: "{{.ProjectURL}}"
  item_url: "ISSUE_URL"
  status: "Todo"  # or "Done" if issue is already closed
{{if .TrackerLabel}}  campaign_id: "CAMPAIGN_ID"  # Required: extract from tracker label {{.TrackerLabel}}
{{end}}```

**Note**: If your project board has `Start Date` and `End Date` fields, these will be **automatically populated** from the issue/PR timestamps:
- `Start Date` is set from the issue's `createdAt` timestamp
- `End Date` is set from the issue's `closedAt` timestamp (if closed)

No additional configuration is needed. The dates are extracted in ISO format (YYYY-MM-DD) and only populate if the fields exist and aren't already set. This enables roadmap timeline visualization.

#### Updating Existing Items

When updating status for an existing board item:
```
update-project:
  project: "{{.ProjectURL}}"
  item_url: "ISSUE_URL"
  status: "Done"  # or "In Progress", "Todo"
{{if .TrackerLabel}}  campaign_id: "CAMPAIGN_ID"  # Required: extract from tracker label {{.TrackerLabel}}
{{end}}```

#### Idempotency

- If an issue is already on the board with matching status → Skip (no-op)
- If an issue is already on the board with different status → Update status field only
- If an issue URL is invalid or deleted → Record failure, continue with remaining items

#### Write Operation Rules

1. **Batch writes separately** - Do not mix reads and writes in the same operation
2. **Validate before writing** - Confirm issue URL exists and is accessible
3. **Record all outcomes** - Log success/failure for each write operation
4. **Never infer state** - Only update based on explicit issue state (open/closed)
5. **Fail gracefully** - If a write fails, record error and continue with remaining operations
{{end}}
