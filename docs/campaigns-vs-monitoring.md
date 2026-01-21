# Campaigns vs Monitoring: Architectural Separation

This document explains the distinction between **Campaigns** and **Monitoring** in GitHub Agentic Workflows.

## Summary

- **Campaigns**: Active orchestration of worker workflows toward goals
- **Monitoring**: Standalone ProjectOps feature for passive tracking
- **Relationship**: Campaigns can use monitoring as a modular component

## Campaigns (Active Orchestration)

### Purpose
Campaigns coordinate multiple agents working toward a shared objective, dispatching workflows, building components on the fly, and verifying results with users.

### Key Characteristics
- **Active coordination**: Orchestrates worker workflows each iteration
- **Goal-driven**: Drives toward an explicit objective with measurable KPIs
- **Creates new work**: Dispatches workflows that create issues, PRs, and other outputs
- **Modular components**: Uses worker workflows that can be recycled or created as needed
- **Decision-making**: The orchestrator adapts strategy based on progress
- **Requires workflows**: Must specify worker workflows to orchestrate via `workflow_dispatch`

### When to Use Campaigns
- Orchestrate worker workflows toward a goal
- Create new issues/PRs through coordinated workflows
- Make decisions and adapt strategy during execution
- Drive a time-bound initiative requiring active coordination
- Build and verify workflows dynamically with user feedback

### File Format
- **Location**: `.github/workflows/*.campaign.md`
- **Required fields**: `id`, `name`, `project-url`, `workflows`, `objective`, `kpis`
- **Compilation**: Generates orchestrator workflow (`.campaign.lock.yml`)

### Examples
- Security vulnerability remediation campaigns
- Framework upgrade initiatives across multiple services
- Technical debt reduction with measurable KPIs
- Multi-repository feature rollouts

## Monitoring (Passive Tracking)

### Purpose
ProjectOps feature for passive tracking and board management without orchestrating new work.

### Key Characteristics
- **Passive observation**: Tracks existing issues and PRs
- **Board management**: Updates project boards based on repository state
- **Metrics collection**: Aggregates progress data without dispatching workflows
- **No orchestration**: Does not create new work or dispatch workflows
- **Event-driven**: Responds to issue/PR events to keep boards synchronized

### When to Use Monitoring
- Track and update project boards based on existing issues/PRs
- Monitor progress without dispatching workflows
- Aggregate metrics from ongoing work
- Keep project boards synchronized with repository state
- Dashboard and reporting use cases

### Implementation
Monitoring is implemented as ProjectOps workflows, not campaign specs:
- **Event triggers**: `issues: opened`, `pull_request: opened`, etc.
- **Safe outputs**: `update-project`, `add-comment` for board updates
- **Tools**: GitHub MCP server with `projects` toolset
- **No compilation**: Standard workflow files, not campaign specs

### Examples
- Automatic issue triage to project boards
- PR status tracking and board updates
- Sprint board automation
- Progress dashboards

See: [ProjectOps documentation](/gh-aw/examples/issue-pr-events/projectops/) for examples.

## Relationship: Campaigns Use Monitoring as a Component

Campaigns can use monitoring capabilities as modular components, just like they use workflow components:

### Via Tracker Labels
Campaigns use `tracker-label` to help monitoring discover campaign-created issues/PRs:

```yaml
# Campaign spec
tracker-label: "campaign:security-q1-2025"
```

Worker workflows apply this label to issues/PRs they create, allowing the campaign orchestrator to discover and track them.

### Via Discovery
Campaigns use discovery mechanisms to find and monitor progress of worker-created items:

```yaml
# Campaign spec
discovery-repos:
  - "org/repo1"
  - "org/repo2"
```

The campaign orchestrator discovers items in these repositories and monitors their status without needing to be a "monitoring campaign."

### Via Project Boards
Both campaigns and monitoring workflows can update the same project boards:

```yaml
# Campaign spec
project-url: "https://github.com/orgs/ORG/projects/1"

# Monitoring workflow safe-output
safe-outputs:
  update-project:
    max: 10
    github-token: ${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}
```

The campaign orchestrates workers while monitoring workflows keep the board synchronized.

## Architectural Benefits

### Separation of Concerns
- Campaigns focus on **active coordination** and **goal achievement**
- Monitoring focuses on **passive tracking** and **board synchronization**
- Clear boundaries prevent feature creep and confusion

### Modularity
- Monitoring is a reusable component
- Campaigns compose monitoring + orchestration + workflows
- Each component can evolve independently

### Clarity
- Users understand campaigns = active work toward goals
- Users understand monitoring = passive tracking without action
- No confusion about "passive campaigns" or "monitoring campaigns"

## Migration from Old Thinking

### Before (Confusing)
❌ "Passive campaigns" that only monitor
❌ Campaign `mode` field with `monitoring` vs `active`
❌ Campaigns without workflows

### After (Clear)
✅ Campaigns always orchestrate workflows (active)
✅ Monitoring is a separate ProjectOps feature
✅ Campaigns can use monitoring as a component

## Decision Tree

```
Need to dispatch workflows toward a goal?
├─ YES → Use Campaigns
│  └─ Can use monitoring components for tracking
└─ NO  → Use ProjectOps Monitoring
   └─ Event-driven workflows for board management
```

## Implementation Files

### Campaigns
- **Spec**: `pkg/campaign/spec.go` - Campaign data structures
- **Validation**: `pkg/campaign/validation.go` - Requires workflows field
- **Schema**: `pkg/campaign/schemas/campaign_spec_schema.json`
- **Docs**: `docs/src/content/docs/guides/campaigns/`

### Monitoring
- **Implementation**: Regular workflow files (`.md` → `.lock.yml`)
- **Pattern**: ProjectOps pattern for project board automation
- **Docs**: `docs/src/content/docs/examples/issue-pr-events/projectops.md`

## Summary Table

| Aspect | Campaigns | Monitoring |
|--------|-----------|-----------|
| **Purpose** | Active orchestration | Passive tracking |
| **File Type** | `.campaign.md` | `.md` (regular workflow) |
| **Workflows** | Required | N/A (no orchestration) |
| **Creates Work** | Yes (via workers) | No |
| **Dispatches** | Yes (`workflow_dispatch`) | No |
| **Project Boards** | Updates via orchestrator | Updates via safe-outputs |
| **Use Case** | Goal-driven initiatives | Board synchronization |
| **Component** | No (top-level) | Yes (used by campaigns) |

## Conclusion

Monitoring is to campaigns what workflows are to campaigns - a modular component that can be used, not a type of campaign. This architectural separation maintains clean boundaries, clear purpose, and allows both concepts to evolve independently while working together effectively.
