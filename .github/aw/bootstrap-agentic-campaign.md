# Bootstrap Instructions (Phase 0)

This phase runs when discovery returns zero work items, providing initial work for the campaign to begin.

{{ if .BootstrapMode }}
## Bootstrap Strategy: {{ .BootstrapMode }}

{{ if eq .BootstrapMode "seeder-worker" }}
### Seeder Worker Dispatch

When no work items are discovered, dispatch a seeder/scanner worker to discover initial work:

**Worker ID**: `{{ .SeederWorkerID }}`

**Payload**:
```json
{{ .SeederPayload }}
```

{{ if gt .SeederMaxItems 0 }}
**Max Items**: {{ .SeederMaxItems }} (limit how many items the seeder returns)
{{ end }}

**Implementation Steps**:

1. Check if discovery returned zero items (no worker outputs found)
2. If zero items:
   - Use the `dispatch_workflow` safe output to trigger the seeder worker
   - Pass the campaign_id and configured payload
   - Wait for the seeder worker to complete and create initial work items
3. On next orchestrator run, the discovery step will find the seeder's outputs

**Seeder Worker Contract**:
- MUST accept `campaign_id` and `payload` inputs (standard worker contract)
- MUST create discoverable outputs (issues, PRs, or discussions)
- MUST apply the tracker label: `z_campaign_{{ .CampaignID }}`
- SHOULD limit output count to configured max-items if provided
- SHOULD use deterministic work item keys for idempotency

{{ else if eq .BootstrapMode "project-todos" }}
### Project Board Todo Items

When no work items are discovered, read from the Project board's "{{ .TodoValue }}" column:

{{ if .StatusField }}
**Status Field**: `{{ .StatusField }}`
{{ else }}
**Status Field**: `Status` (default)
{{ end }}

{{ if .TodoValue }}
**Todo Value**: `{{ .TodoValue }}`
{{ else }}
**Todo Value**: `Todo` (default)
{{ end }}

{{ if gt .TodoMaxItems 0 }}
**Max Items**: {{ .TodoMaxItems }} (limit how many Todo items to process)
{{ end }}

{{ if .RequireFields }}
**Required Fields**: {{ range $index, $field := .RequireFields }}{{ if $index }}, {{ end }}`{{ $field }}`{{ end }}
  - Skip Todo items where any of these fields are empty
{{ end }}

**Implementation Steps**:

1. Check if discovery returned zero items (no worker outputs found)
2. If zero items:
   - Query the Project board at `{{ .ProjectURL }}`
   - Filter items where Status = "{{ .TodoValue }}"
   {{ if .RequireFields }}- Skip items missing required fields{{ end }}
   {{ if gt .TodoMaxItems 0 }}- Limit to {{ .TodoMaxItems }} items{{ end }}
   - Select workers based on item metadata (see Worker Selection below)
   - Dispatch appropriate worker workflows for each Todo item
3. Update Project status to "In Progress" for dispatched items
4. On next orchestrator run, the discovery step will find the worker outputs

**Project Item to Payload Mapping**:
- Read Project field values from the Todo item
- Map to worker payload schema based on worker metadata
- Include campaign_id in every worker dispatch
- Use the item's URL or number as the work_item_id

{{ else if eq .BootstrapMode "manual" }}
### Manual Bootstrap

No automatic bootstrap configured. Wait for manual work item creation:

- Work items should be created manually (issues, PRs, or discussions)
- All items MUST have the tracker label: `z_campaign_{{ .CampaignID }}`
- Items MUST follow the worker output labeling contract
- Once items exist, the orchestrator will discover them normally

{{ end }}
{{ end }}

---

## Worker Selection

{{ if .WorkerMetadata }}
When dispatching workers during bootstrap, use deterministic selection:

{{ range $index, $worker := .WorkerMetadata }}
### Worker {{ add1 $index }}: {{ .ID }}

**Capabilities**: {{ range $capIndex, $cap := .Capabilities }}{{ if $capIndex }}, {{ end }}`{{ $cap }}`{{ end }}

**Payload Schema**:
{{ range $fieldName, $fieldDef := .PayloadSchema }}- `{{ $fieldName }}` ({{ .Type }}{{ if .Required }}, required{{ end }}): {{ .Description }}
{{ end }}

**Output Labeling**:
{{ if .OutputLabeling.Labels }}- Labels: {{ range $labelIndex, $label := .OutputLabeling.Labels }}{{ if $labelIndex }}, {{ end }}`{{ $label }}`{{ end }}
{{ end }}- Key in Title: {{ .OutputLabeling.KeyInTitle }}
{{ if .OutputLabeling.KeyFormat }}- Key Format: `{{ .OutputLabeling.KeyFormat }}`
{{ end }}
- Campaign tracker label applied automatically: `z_campaign_{{ $.CampaignID }}`

**Idempotency Strategy**: {{ .IdempotencyStrategy }}

{{ if .Priority }}**Priority**: {{ .Priority }} (higher = preferred when multiple workers match)
{{ end }}

{{ end }}

**Selection Algorithm**:

1. For each work item, check which workers can handle it:
   - Match work item type/metadata to worker capabilities
   - Check if worker's payload schema requirements can be satisfied
2. If multiple workers match:
   - Select the worker with highest priority
   - If priorities are equal, select first alphabetically by ID
3. Build payload from work item metadata according to worker's payload schema
4. Dispatch worker with campaign_id and constructed payload

{{ else }}
**Note**: No worker metadata configured. Use workflow IDs from campaign spec for dispatch.
{{ end }}

---

## Bootstrap Success Criteria

After bootstrap completes:

1. ✅ At least one worker workflow dispatched (or manual items created)
2. ✅ All dispatched items will have proper tracker labels
3. ✅ Next orchestrator run will discover >= 1 work item
4. ✅ Campaign transitions from bootstrap phase to normal operation

**Idempotency**: Bootstrap only runs when discovery = 0. Once work items exist, normal orchestration takes over.
