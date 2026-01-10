---
id: simple-upgrade-example
name: "Simple Framework Upgrade Campaign"
description: "Example showing how campaigns can execute workflows to upgrade services"
version: v1
state: active

project-url: "https://github.com/orgs/myorg/projects/999"
tracker-label: "campaign:framework-upgrade"

objective: "Upgrade all services to Framework vNext"
kpis:
  - name: "Services upgraded"
    priority: primary
    unit: count
    baseline: 0
    target: 50
    time-window-days: 30
    direction: increase

# Workflows to execute
workflows:
  - framework-scanner
  - framework-upgrader

# Enable active workflow execution
execute-workflows: true

# Governance
governance:
  max-project-updates-per-run: 20

owners:
  - "engineering-team"
---

# Simple Framework Upgrade Campaign

This campaign demonstrates the simplified workflow execution feature.

## How It Works

When `execute-workflows: true` is set, the campaign orchestrator will:

1. **Execute each workflow in sequence** - Run `framework-scanner` then `framework-upgrader`

2. **Create workflows if needed** - If a workflow doesn't exist, the orchestrator will:
   - Design an appropriate workflow based on the campaign objective
   - Create the workflow file with proper configuration
   - Compile it
   - Execute it

3. **Use outputs to drive progress** - Collect information from workflow runs to inform:
   - Subsequent workflow executions
   - Project board updates
   - Campaign progress reporting

## Example Flow

```
Campaign Run Starts
    ↓
Phase 0: Workflow Execution
    ↓
Check if framework-scanner exists
    ↓
[If not exist] Create framework-scanner workflow
    ↓
Execute framework-scanner
    ↓
Wait for completion
    ↓
Collect outputs/artifacts
    ↓
Check if framework-upgrader exists
    ↓
[If not exist] Create framework-upgrader workflow
    ↓
Execute framework-upgrader (using scanner outputs)
    ↓
Wait for completion
    ↓
Phase 1-4: Normal campaign operations
    ↓
Discovery → Planning → Updates → Reporting
```

## Benefits

- **Simple configuration**: Just set `execute-workflows: true`
- **Self-sufficient**: Creates missing workflows automatically
- **Sequential execution**: Workflows run one at a time
- **Backward compatible**: Existing campaigns work unchanged

## Usage

1. Set `execute-workflows: true` in your campaign spec
2. List workflows in the `workflows:` field
3. Define the campaign objective clearly
4. Compile: `gh aw compile`
5. Run the campaign

The orchestrator will handle workflow creation and execution automatically!
