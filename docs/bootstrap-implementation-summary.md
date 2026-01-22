# Bootstrap + Planning Model: Implementation Summary

This document provides a comprehensive overview of the bootstrap and planning model implementation for campaign generator/orchestrator workflows.

## Problem Statement

The generator/orchestrator needed an explicit bootstrap + planning model to address these scenarios:

1. **Zero Discovery**: What happens when discovery returns 0 work items?
2. **Initial Work Creation**: How to create/select initial work items?
3. **Worker Selection**: How to deterministically choose which worker to run?
4. **Output Discoverability**: How to guarantee worker outputs are discoverable and attributable?

## Solution Architecture

### 1. Bootstrap Configuration

When discovery returns 0 items, the orchestrator can use one of three bootstrap strategies:

#### Mode: seeder-worker

Dispatch a specialized worker to discover and create initial work:

```yaml
bootstrap:
  mode: seeder-worker
  seeder-worker:
    workflow-id: security-scanner
    payload:
      scan-type: full
      ecosystems: [npm, pip, go]
    max-items: 50
```

**Flow**:
1. Orchestrator detects `discovery_count == 0`
2. Orchestrator dispatches seeder worker via `dispatch_workflow` safe output
3. Seeder scans repositories/systems and creates issues/PRs
4. Seeder applies tracker labels to all created items
5. Next orchestrator run discovers seeder's outputs (discovery > 0)

#### Mode: project-todos

Read work items from Project board's "Todo" column:

```yaml
bootstrap:
  mode: project-todos
  project-todos:
    status-field: Status
    todo-value: Backlog
    max-items: 10
    require-fields: [Priority, Assignee]
```

**Flow**:
1. Orchestrator detects `discovery_count == 0`
2. Orchestrator queries Project board for Status = "Backlog"
3. Orchestrator filters items with missing required fields
4. Orchestrator uses worker metadata to select appropriate worker
5. Orchestrator builds payload from Project field values
6. Orchestrator dispatches workers for each Todo item
7. Workers create issues/PRs with proper tracker labels
8. Next orchestrator run discovers worker outputs

#### Mode: manual

Wait for manual work item creation:

```yaml
bootstrap:
  mode: manual
```

**Flow**:
1. Orchestrator detects `discovery_count == 0`
2. Orchestrator reports waiting for manual work items
3. Users manually create issues/PRs with tracker labels
4. Next orchestrator run discovers manual items

### 2. Worker Metadata

Worker metadata provides a standardized payload schema and output contract:

```yaml
workers:
  - id: security-fixer
    name: Security Fix Worker
    description: Fixes security vulnerabilities
    
    # What this worker can do
    capabilities:
      - fix-security-alerts
      - create-pull-requests
    
    # Expected payload structure
    payload-schema:
      repository:
        type: string
        description: Target repository (owner/repo)
        required: true
        example: owner/repo
      alert_id:
        type: string
        description: Alert identifier
        required: true
        example: alert-123
      severity:
        type: string
        description: Alert severity
        required: false
        example: high
    
    # Output labeling contract
    output-labeling:
      labels: [security, automated]
      key-in-title: true
      key-format: "campaign-{campaign_id}-{repository}-{alert_id}"
      metadata-fields: [Campaign Id, Worker Workflow, Alert ID]
    
    # Idempotency strategy
    idempotency-strategy: pr-title-based
    
    # Selection priority (higher = preferred)
    priority: 10
```

Note: The campaign's tracker-label (defined at the campaign level) is automatically applied to all worker outputs.

### 3. Deterministic Worker Selection

When multiple workers exist, the orchestrator selects deterministically:

**Algorithm**:
```python
def select_worker(work_item, workers):
    # Step 1: Filter by capabilities
    matching_workers = [w for w in workers 
                        if can_handle(w.capabilities, work_item.type)]
    
    # Step 2: Validate payload requirements
    valid_workers = [w for w in matching_workers 
                     if can_build_payload(w.payload_schema, work_item)]
    
    # Step 3: Sort by priority (descending)
    valid_workers.sort(key=lambda w: w.priority, reverse=True)
    
    # Step 4: Select highest priority (or first alphabetically)
    if len(valid_workers) > 0:
        return valid_workers[0]
    
    return None
```

**Example**: Given a security alert work item:

1. **Match capabilities**: Both `security-scanner` and `security-fixer` match
2. **Validate payload**: Both can satisfy required fields
3. **Sort by priority**: `security-fixer` (priority 10) > `security-scanner` (priority 5)
4. **Select**: Dispatch `security-fixer`

### 4. Output Labeling Contract

Workers guarantee outputs are discoverable via:

#### Tracker Label

Format: `campaign:{campaign_id}`

- Applied to ALL worker-created items
- Enables discovery by campaign orchestrator
- Isolates campaign items from other workflows

#### Deterministic Keys

Format defined by `key-format` in worker metadata:

```
campaign-{campaign_id}-{repository}-{work_item_id}
```

- Included in issue/PR titles: `[{key}] {description}`
- Enables idempotency checks before creation
- Allows duplicate detection across runs

#### Metadata Fields

Workers populate Project fields for tracking:

- `Campaign Id`: Links to campaign
- `Worker Workflow`: Which worker created this
- Custom fields: Alert ID, Severity, Package Name, etc.

## End-to-End Example

### Scenario: Security Alert Burndown Campaign

**Campaign Configuration**:
```yaml
id: security-q1-2025
name: Security Alert Burndown
project-url: https://github.com/orgs/example/projects/1

bootstrap:
  mode: seeder-worker
  seeder-worker:
    workflow-id: security-scanner
    payload:
      severity: high
      max-alerts: 20

workers:
  - id: security-scanner
    capabilities: [scan-security-alerts]
    payload-schema:
      severity: {type: string, required: true}
      max-alerts: {type: number, required: false}
    output-labeling:
      key-in-title: true
      key-format: "scan-{repository}"
    idempotency-strategy: issue-title-based
    priority: 5

  - id: security-fixer
    capabilities: [fix-security-alerts, create-pull-requests]
    payload-schema:
      repository: {type: string, required: true}
      alert_id: {type: string, required: true}
    output-labeling:
      key-in-title: true
      key-format: "campaign-{campaign_id}-{repository}-{alert_id}"
    idempotency-strategy: pr-title-based
    priority: 10

tracker-label: campaign:security-q1-2025
```

Note: The tracker-label is defined once at the campaign level and automatically applied by all workers.

**Execution Flow**:

1. **Run 1: Bootstrap (discovery = 0)**
   - Orchestrator detects no work items
   - Dispatches `security-scanner` with `{severity: "high", max-alerts: 20}`
   - Scanner finds 15 high-severity alerts
   - Scanner creates 15 issues with:
     - Label: `campaign:security-q1-2025`
     - Title: `[scan-owner-repo] High severity alerts found`
   
2. **Run 2: Discovery (discovery = 15)**
   - Discovery finds 15 issues with tracker label
   - Orchestrator reads issue metadata
   - For each issue, orchestrator:
     - Parses alert details from issue body
     - Selects `security-fixer` (highest priority with matching capabilities)
     - Builds payload: `{repository: "owner/repo", alert_id: "alert-123"}`
     - Dispatches `security-fixer`
   - Each fixer worker:
     - Checks for existing PR with key in title
     - Creates PR if not exists
     - Applies labels and metadata
   
3. **Run 3+: Normal Operation (discovery > 0)**
   - Discovery finds PRs created by workers
   - Orchestrator updates Project board
   - Tracks progress via KPIs
   - Reports on completion

## Implementation Details

### Code Structure

```
pkg/campaign/
├── spec.go                     # CampaignBootstrapConfig, WorkerMetadata types
├── orchestrator.go             # Bootstrap integration
├── template.go                 # RenderBootstrapInstructions()
├── schemas/
│   └── campaign_spec_schema.json  # JSON schema validation
└── bootstrap_test.go          # Unit tests

.github/aw/
└── bootstrap-agentic-campaign.md  # Bootstrap template

docs/
├── campaign-workers.md        # Updated documentation
└── src/content/docs/examples/campaigns/
    └── dependency-upgrade-example.campaign.md  # Example
```

### Key Types

```go
type CampaignBootstrapConfig struct {
    Mode           string
    SeederWorker   *SeederWorkerConfig
    ProjectTodos   *ProjectTodosConfig
}

type WorkerMetadata struct {
    ID                  string
    Capabilities        []string
    PayloadSchema       map[string]WorkerPayloadField
    OutputLabeling      WorkerOutputLabeling
    IdempotencyStrategy string
    Priority            int
}

type WorkerOutputLabeling struct {
    Labels          []string
    KeyInTitle      bool
    KeyFormat       string
    MetadataFields  []string
}
```

## Testing

13 unit tests covering:
- ✅ Bootstrap config parsing (3 modes)
- ✅ Worker metadata parsing
- ✅ Payload schema validation
- ✅ Output labeling contracts
- ✅ JSON/YAML serialization
- ✅ Combined bootstrap + workers scenarios

All tests passing in `pkg/campaign/bootstrap_test.go`.

## Benefits

1. **Explicit Bootstrap**: No ambiguity about how campaigns start
2. **Deterministic Selection**: Workers chosen by capabilities and priority
3. **Guaranteed Discoverability**: Output labeling contract ensures items are found
4. **Idempotency**: Keys and strategies prevent duplicate work
5. **Flexibility**: Three bootstrap modes for different scenarios
6. **Type Safety**: Payload schemas validated at campaign definition time

## Migration Guide

Existing campaigns continue to work without changes. To adopt the new features:

1. **Add Bootstrap**: Choose a mode based on your scenario
2. **Define Workers**: Document capabilities and schemas
3. **Test**: Manually trigger with discovery = 0 to verify bootstrap
4. **Iterate**: Adjust worker priorities and payload schemas as needed

## Future Enhancements

Potential improvements:
- [ ] Runtime payload schema validation
- [ ] Worker capability matching with semantic rules
- [ ] Auto-generation of worker metadata from workflow analysis
- [ ] Bootstrap mode: `hybrid` (try project-todos, fallback to seeder-worker)
- [ ] Worker selection telemetry and optimization
