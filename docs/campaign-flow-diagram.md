# Campaign Flow Architecture

This diagram shows how a campaign orchestrator coordinates worker workflows and tracks progress through a GitHub Project board.

## Campaign Flow Diagram

```mermaid
graph TB
    CampaignSpec["Campaign Spec<br/>.campaign.md"]
    
    CampaignSpec -->|"gh aw compile"| Orchestrator["Orchestrator Workflow<br/>.campaign.g.lock.yml"]
    
    Orchestrator -->|"scheduled run"| Discover["Discover Work Items<br/>(read from repo-memory)"]
    
    Discover --> Plan["Plan Updates<br/>(apply budgets)"]
    
    Plan --> Update["Update Project Board<br/>(status, dates, metadata)"]
    
    Update --> Report["Report Status<br/>(KPIs, progress)"]
    
    Workers["Worker Workflows<br/>(campaign-agnostic)"] -.->|"write files"| Memory["Repo-Memory<br/>(memory/campaigns branch)"]
    
    Memory -.->|"read files"| Discover
    
    style CampaignSpec fill:#e1f5ff
    style Orchestrator fill:#fff4e6
    style Workers fill:#e8f5e9
    style Memory fill:#fff9c4
```

## How It Works

### Campaign Spec
Define a campaign in `.github/workflows/*.campaign.md`:
```yaml
---
id: security-audit
memory-paths:
  - memory/campaigns/security-audit/**
cursor-glob: memory/campaigns/security-audit/cursor.json
project-url: https://github.com/orgs/org/projects/1
---
```

### Orchestrator
The orchestrator runs on a schedule (daily) and:
1. **Discovers** work items by reading files from repo-memory
2. **Plans** updates based on GitHub state (open/closed/merged)
3. **Updates** the project board with status and metadata
4. **Reports** KPIs and progress via project status updates

### Workers
Worker workflows are campaign-agnostic and write files to repo-memory (the `memory/campaigns` Git branch). The orchestrator discovers work by reading these files in subsequent runs.

### Project Board
The GitHub Project board is the authoritative state for the campaign, with fields for status, dates, priority, and metadata.

## References

- Campaign Specs: `pkg/campaign/spec.go`
- Orchestrator Builder: `pkg/campaign/orchestrator.go`
- Discovery Logic: `actions/setup/js/campaign_discovery.cjs`
- Project Updates: `actions/setup/js/update_project.cjs`
- Flow Documentation: `docs/src/content/docs/guides/campaigns/flow.md`
