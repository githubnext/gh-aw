---
id: framework-upgrade-example
name: "Example: Framework Upgrade Campaign"
description: "Demonstrates active workflow execution in campaigns - scans for services needing upgrades, then executes upgrades based on scan results"
version: v1
state: planned

# Project configuration
project-url: "https://github.com/orgs/myorg/projects/999"
tracker-label: "campaign:framework-upgrade-example"

# Campaign objective and KPIs
objective: "Systematically upgrade all services to Framework vNext with zero downtime"
kpis:
  - name: "Services upgraded"
    priority: primary
    unit: count
    baseline: 0
    target: 50
    time-window-days: 30
    direction: increase
    source: custom

  - name: "Failed upgrades"
    priority: supporting
    unit: count
    baseline: 0
    target: 0
    time-window-days: 30
    direction: decrease
    source: custom

# Worker workflows (for discovery and tracking)
# These workflows create issues/PRs that are discovered by the campaign
workflows:
  - framework-scanner
  - framework-upgrader

# Workflow execution configuration (NEW)
execution:
  # Define the sequence of workflows to execute
  sequence:
    # Step 1: Scan for services that need upgrading
    - workflow: framework-scanner
      inputs:
        scan_type: "comprehensive"
        framework_version: "vNext"
      outputs:
        - name: "services_to_upgrade"
          from: "artifact:scan-results.json:$.services"
          required: true
        - name: "services_count"
          from: "artifact:scan-results.json:$.total_count"
          required: true
        - name: "scan_report_url"
          from: "artifact:scan-results.json:$.report_url"
          required: false
      continue-on-failure: false

    # Step 2: Execute upgrades (only if services found)
    - workflow: framework-upgrader
      condition: "outputs.services_count > 0"
      inputs:
        services: "${{ outputs.services_to_upgrade }}"
        batch_size: "5"
        framework_version: "vNext"
        dry_run: "false"
      outputs:
        - name: "upgrade_results"
          from: "artifact:upgrade-results.json"
          required: true
        - name: "upgrade_conclusion"
          from: "conclusion"
          required: true
      continue-on-failure: true

  # Execution settings
  max-concurrent-workflows: 1  # Sequential execution
  timeout-minutes: 60          # Max 60 minutes per workflow

# Repository memory for state persistence
memory-paths:
  - "memory/campaigns/framework-upgrade-example/**"
metrics-glob: "memory/campaigns/framework-upgrade-example/metrics/*.json"
cursor-glob: "memory/campaigns/framework-upgrade-example/cursor.json"

# Governance policies
governance:
  max-project-updates-per-run: 20
  max-comments-per-run: 10
  max-new-items-per-run: 10
  max-discovery-items-per-run: 100
  max-discovery-pages-per-run: 10
  opt-out-labels: ["no-campaign", "no-bot"]

# Ownership
owners:
  - "engineering-team"
risk-level: medium

# Safe outputs
allowed-safe-outputs:
  - update-project
  - add-comment
  - create-issue
---

# Framework Upgrade Campaign (Example)

## Overview

This is an **example campaign** that demonstrates the new **active workflow execution** feature in GitHub Agentic Workflows.

Unlike traditional passive campaigns that only discover and track work, this campaign **actively runs workflows**, waits for their completion, collects outputs, and makes decisions based on the results.

## What This Campaign Does

### Phase 0: Active Workflow Execution (NEW)

**Step 1: Framework Scanner**
- **Workflow**: `framework-scanner` 
- **Purpose**: Scans all services to identify which ones need framework upgrades
- **Inputs**:
  - `scan_type`: "comprehensive"
  - `framework_version`: "vNext"
- **Outputs Collected**:
  - `services_to_upgrade`: List of services from artifact `scan-results.json`
  - `services_count`: Total count of services needing upgrades
  - `scan_report_url`: Optional URL to detailed scan report

**Step 2: Framework Upgrader** (Conditional)
- **Workflow**: `framework-upgrader`
- **Condition**: Only runs if `services_count > 0`
- **Purpose**: Executes upgrades for services identified in Step 1
- **Inputs**:
  - `services`: The list from Step 1 (`${{ outputs.services_to_upgrade }}`)
  - `batch_size`: "5" (upgrade 5 services at a time)
  - `framework_version`: "vNext"
  - `dry_run`: "false"
- **Outputs Collected**:
  - `upgrade_results`: Detailed results from artifact
  - `upgrade_conclusion`: Workflow conclusion (success/failure)

### Phases 1-4: Traditional Campaign Operations

After workflow execution completes:
1. **Phase 1**: Discover issues/PRs created by workers
2. **Phase 2**: Plan project board updates
3. **Phase 3**: Update project board with new items
4. **Phase 4**: Report campaign progress and KPIs

## Key Features Demonstrated

### 1. Sequential Workflow Execution
The campaign runs workflows in order, waiting for each to complete before proceeding to the next.

### 2. Conditional Execution
The upgrader workflow only runs if the scanner found services to upgrade (`services_count > 0`).

### 3. Output-Based Decision Making
Outputs from the scanner workflow are passed as inputs to the upgrader workflow, enabling data-driven execution.

### 4. Multiple Output Sources
- **Artifacts**: Extract structured data from JSON artifacts
- **JSONPath**: Navigate to specific fields (e.g., `$.services`)
- **Conclusion**: Capture workflow success/failure status

### 5. Error Handling
- Scanner failure stops execution (critical)
- Upgrader failure continues (allows partial progress reporting)

## How It Works

### Campaign Orchestrator Agent

The orchestrator agent uses **GitHub MCP tools** to:

1. **Trigger workflows**: `mcp__github__run_workflow`
2. **Check status**: `mcp__github__get_workflow_run`
3. **Wait for completion**: Poll every 30-60 seconds
4. **Collect outputs**: `mcp__github__download_workflow_run_artifact`
5. **Make decisions**: Evaluate conditions using collected outputs

### Example Execution Flow

```
Campaign Run Starts
    ↓
[Phase 0: Workflow Execution]
    ↓
1. Trigger framework-scanner
    ↓
2. Poll for completion (every 30s)
    ↓
3. Scanner completes ✓
    ↓
4. Download artifact: scan-results.json
    ↓
5. Extract outputs:
   - services_to_upgrade = ["service-a", "service-b", "service-c"]
   - services_count = 3
    ↓
6. Evaluate condition: services_count > 0? YES
    ↓
7. Trigger framework-upgrader with inputs
    ↓
8. Poll for completion (every 30s)
    ↓
9. Upgrader completes ✓
    ↓
10. Download artifact: upgrade-results.json
    ↓
[Phase 1: Discovery]
    ↓
11. Discover issues/PRs created by workers
    ↓
[Phase 2-4: Plan, Update, Report]
    ↓
12. Update project board
13. Report campaign progress
    ↓
Campaign Run Ends
```

## Expected Outcomes

When this campaign runs:

1. **Workflow Executions**: Scanner and upgrader workflows are triggered automatically
2. **Data Flow**: Scan results flow into upgrader inputs
3. **Conditional Logic**: Upgrader only runs if services are found
4. **Status Update**: Orchestrator reports workflow execution results including:
   - Which workflows ran
   - Execution times
   - Outputs collected
   - Success/failure status
   - Next steps

## Prerequisites

To use this example campaign:

1. **Create Worker Workflows**:
   - `.github/workflows/framework-scanner.md` - Scans for services
   - `.github/workflows/framework-upgrader.md` - Performs upgrades

2. **Worker Requirements**:
   - Both workflows must support `workflow_dispatch`
   - Scanner must create `scan-results.json` artifact
   - Upgrader must create `upgrade-results.json` artifact

3. **GitHub Project**:
   - Create a GitHub Project (replace URL in spec)
   - Configure project fields: status, campaign_id, worker_workflow

4. **Compile Campaign**:
   ```bash
   gh aw compile
   ```

5. **Run Campaign**:
   - Manually trigger from GitHub Actions tab
   - Or wait for scheduled run (daily at 18:00 UTC)

## Benefits vs Traditional Campaigns

| Traditional Campaign | This Campaign (Active Execution) |
|---------------------|----------------------------------|
| Discovers work done by workers | **Actively drives work forward** |
| Tracks progress passively | **Makes decisions based on results** |
| Workers run independently | **Orchestrates workflow sequence** |
| No data flow between workers | **Passes outputs between workflows** |
| Manual coordination needed | **Automated coordination** |

## Learn More

- [Campaign Specs Documentation](/gh-aw/guides/campaigns/specs/)
- [Workflow Execution Guide](/gh-aw/guides/campaigns/workflow-execution/)
- [Campaign Best Practices](/gh-aw/guides/campaigns/best-practices/)

## Notes

- This is an **example for demonstration purposes**
- The worker workflows (`framework-scanner`, `framework-upgrader`) are placeholders
- Adapt this pattern to your actual use cases
- Test thoroughly before production use
