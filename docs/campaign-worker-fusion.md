# Campaign Worker Workflow Fusion

This document explains the campaign worker workflow fusion feature and its folder structure.

## Overview

Campaign worker workflow fusion allows existing workflows to be adapted for campaign use by:

1. **Discovery**: Automatically finding workflows that match campaign goals
2. **Fusion**: Adding `workflow_dispatch` trigger to enable manual/programmatic execution
3. **Organization**: Storing adapted workflows in campaign-specific folders
4. **Orchestration**: Using `dispatch_workflow` safe output to trigger workers from campaign orchestrators

## Folder Structure

```
.github/workflows/
├── security-scanner.md              # Original workflow
├── dependency-updater.md            # Original workflow
├── my-campaign.campaign.md          # Campaign spec
├── my-campaign.campaign.lock.yml    # Compiled campaign orchestrator
└── campaigns/
    └── my-campaign/                 # Campaign-specific folder
        ├── security-scanner-worker.md      # Fused worker workflow
        ├── security-scanner-worker.lock.yml
        ├── dependency-updater-worker.md
        └── dependency-updater-worker.lock.yml
```

## Why Separate Folders?

Campaign-specific folders serve multiple purposes:

1. **Pattern Analysis**: Track which workflows work best for different campaign types
2. **Reusability**: Identify successful patterns for future campaigns
3. **Organization**: Keep campaign workers separate from regular workflows
4. **Traceability**: Maintain clear lineage from original to fused workflows

## Workflow Fusion Process

### Automatic Fusion (Future)

Future enhancements may include automatic fusion during campaign generation:

```go
// During campaign generation
campaignID := "security-audit-2026"
goals := []string{"security", "dependencies"}

// 1. Discover matching workflows
matches, _ := campaign.DiscoverWorkflows(rootDir, goals)

// 2. Fuse top matches for campaign use
workflowIDs := []string{"security-scanner", "dependency-updater"}
results, _ := campaign.FuseMultipleWorkflows(rootDir, workflowIDs, campaignID)

// 3. Update campaign spec with fused workflow IDs
// spec.Workflows = append(spec.Workflows, results[0].CampaignWorkflowID, ...)
```

### Manual Fusion (Current)

Currently, users can manually fuse workflows:

```bash
# Create campaign folder
mkdir -p .github/workflows/campaigns/my-campaign

# Copy and adapt workflow
cp .github/workflows/security-scanner.md \
   .github/workflows/campaigns/my-campaign/security-scanner-worker.md

# Edit worker workflow to add workflow_dispatch and campaign metadata
```

## Fused Workflow Metadata

Fused workflows include metadata to track their origin:

```yaml
---
name: Security Scanner Worker
campaign-worker: true
campaign-id: security-audit-2026
source-workflow: security-scanner
on:
  workflow_dispatch:    # Added by fusion
  schedule:             # Original trigger preserved
    - cron: "0 9 * * 1"
---
```

## Campaign Orchestrator Integration

Campaign orchestrators can dispatch worker workflows using the `dispatch_workflow` safe output:

```yaml
safe-outputs:
  dispatch-workflow:
    workflows:
      - security-scanner-worker
      - dependency-updater-worker
    max: 3
```

## Pattern Analysis (Future)

The separate folder structure enables future analysis capabilities:

1. **Success Metrics**: Track which workflow patterns achieve campaign goals
2. **Adaptation Analysis**: Compare original vs. fused workflows
3. **Reusability Scoring**: Identify workflows that work across multiple campaigns
4. **Best Practices**: Generate recommendations based on historical data

### Example Analysis Queries

```bash
# Find most-used worker workflows
find .github/workflows/campaigns -name "*.md" | 
  xargs grep "source-workflow:" | 
  cut -d: -f2 | 
  sort | 
  uniq -c | 
  sort -rn

# Find campaigns by goal type
grep -r "objective:" .github/workflows/*.campaign.md |
  grep -i "security" |
  cut -d: -f1
```

## Best Practices

### Naming Conventions

- **Campaign folders**: Use campaign ID (e.g., `campaigns/security-audit-2026/`)
- **Worker workflows**: Add `-worker` suffix (e.g., `security-scanner-worker.md`)
- **Lock files**: Same name with `.lock.yml` extension

### Metadata Standards

Always include these fields in fused workflows:

```yaml
campaign-worker: true         # Identifies as campaign worker
campaign-id: <campaign-id>    # Links to parent campaign
source-workflow: <original>   # Tracks lineage
```

### Trigger Preservation

When adding `workflow_dispatch`:

- **Preserve** original triggers (schedule, issues, etc.)
- **Add** workflow_dispatch alongside existing triggers
- **Document** fusion changes in workflow description

### Folder Organization

- One folder per campaign under `campaigns/`
- All worker workflows for a campaign in its folder
- Include lock files in the same folder
- Commit fused workflows for pattern analysis

## Examples

Example campaign workflows demonstrating these concepts can be found in:
- [Campaign Examples](./docs/src/content/docs/examples/campaigns.md) - Example campaign specs and worker workflows
- Example security audit campaign with 3 worker workflows
- Example security scanner workflow with tracker-id usage

## Related Documentation

- [Campaign Specs](./specs/campaigns-files.md) - Campaign specification format
- [Dispatch Workflow](./docs/src/content/docs/guides/dispatchops.md) - Using workflow_dispatch
- [Safe Outputs](./docs/src/content/docs/reference/safe-outputs.md) - Safe output reference

## Future Enhancements

1. **Automatic Fusion**: Generate fused workflows during campaign creation
2. **Pattern Library**: Curated library of proven workflow patterns
3. **Smart Recommendations**: AI-powered workflow suggestions based on campaign goals
4. **Performance Metrics**: Track and display fusion effectiveness
5. **Dependency Tracking**: Automatically update fused workflows when originals change
