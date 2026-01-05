---
title: "Campaign Improvements & Future Directions"
description: "Recommendations for enhancing campaign reporting, learning systems, and orchestrator capabilities"
---

This document outlines opportunities to improve campaign functionality, particularly around summarized reporting, learning from campaign outcomes, and better integration between orchestrators and metrics.

## Current State

The campaign system provides:
- Discovery precomputation for efficient item tracking
- Cursor-based incremental processing
- Metrics snapshots written to repo-memory
- Project board synchronization
- Rate limit controls and governance policies

## Improvement Opportunities

### 1. Summarized Campaign Reports

**Current Limitation**: Campaign orchestrators write individual metrics snapshots but don't generate human-readable progress summaries.

**Proposed Enhancement**: Add a summary report generation capability to orchestrators that:

- **Aggregates metrics across runs**: Read multiple metrics snapshots from repo-memory and calculate trends
- **Generates markdown reports**: Create formatted markdown summaries with:
  - Current campaign status (tasks completed, in progress, blocked)
  - Velocity trends (tasks per day over time)
  - KPI progress toward targets
  - Top contributors (workflows with most completed items)
  - Blockers and risks (items stuck in certain states)
- **Posts to Epic issue**: Automatically update the campaign Epic issue with the latest summary as a comment
- **Configurable frequency**: Allow campaigns to specify report frequency (e.g., weekly, every 10 runs)

**Example Report Format**:

```markdown
## Campaign Progress Report (2025-01-05)

**Overall Status**: 🟢 On track

### Metrics Summary
- **Tasks Completed**: 45 / 200 (22.5%)
- **Current Velocity**: 7.5 tasks/day
- **Estimated Completion**: 2025-02-12 (38 days remaining)

### This Week's Progress
- ✅ 52 new tasks completed (+15%)
- 🚧 8 items in progress
- 🚫 3 items blocked (down from 5)

### Worker Activity
- `migration-worker`: 28 completed (top contributor)
- `daily-doc-updater`: 12 completed
- `unbloat-docs`: 7 completed

### KPI Progress
- **Services upgraded**: 45 / 50 target (90%) ⬆️
- **Incidents caused**: 0 / 0 target (✅ met)

### Blockers Resolved This Week
- Fixed API rate limit issue in discovery
- Unblocked 2 items waiting for external reviews
```

### 2. Campaign Learning System

**Current Limitation**: Campaigns don't capture or share learnings across runs or between campaigns.

**Proposed Enhancement**: Implement a structured learning system that:

- **Captures common patterns**: Identify recurring issues, successful strategies, and anti-patterns
- **Stores learnings in repo-memory**: Add `memory/campaigns/<id>/learnings.json` with structured insights
- **Shares learnings across campaigns**: Enable campaigns with similar objectives to reference learnings from completed campaigns
- **Surfaces recommendations**: Orchestrators can suggest improvements based on historical data

**Learning Categories**:

1. **Discovery Efficiency**
   - Optimal pagination budgets for different campaign scales
   - Most effective tracker label patterns
   - API rate limit patterns and mitigation strategies

2. **Worker Performance**
   - Average completion time per workflow
   - Success rates and common failure modes
   - Optimal scheduling for different workflows

3. **Project Management**
   - Field usage patterns (which fields are most valuable)
   - View configurations that work best
   - Status transition patterns (typical item lifecycle)

4. **Governance Tuning**
   - Effective rate limit configurations
   - Optimal max-items-per-run values
   - Successful opt-out label strategies

**Example Learning Entry**:

```json
{
  "campaign_id": "docs-quality-maintenance-project73",
  "date": "2025-01-05",
  "category": "discovery_efficiency",
  "insight": "Increased max-discovery-pages-per-run from 5 to 10",
  "impact": "Reduced average discovery time by 40%, improved cursor freshness",
  "recommendation": "For campaigns with >100 tracked items, start with 10 pages minimum"
}
```

### 3. Enhanced Metrics Integration

**Current Limitation**: Metrics are written but not actively used by orchestrators for decision-making.

**Proposed Enhancement**: Enable orchestrators to read and act on historical metrics:

- **Adaptive rate limiting**: Adjust discovery budgets based on recent velocity trends
- **Dynamic prioritization**: Focus on blocked items when velocity drops
- **Anomaly detection**: Alert when completion rate deviates significantly from trends
- **Capacity planning**: Estimate required orchestrator run frequency to meet targets

**Example Decision Logic**:

```yaml
# If velocity drops below 50% of average, increase discovery budget
if current_velocity < (avg_velocity * 0.5):
  max_discovery_items = max_discovery_items * 1.5
  
# If >20% of items are blocked, prioritize unblocking
if blocked_percentage > 0.2:
  focus_on_blocked = true
```

### 4. Campaign Retrospectives

**Current Limitation**: No structured retrospective process when campaigns complete.

**Proposed Enhancement**: Add campaign completion workflow that:

- **Analyzes final metrics**: Calculate success rate against KPIs
- **Generates retrospective report**: Document what worked, what didn't, and recommendations
- **Archives learnings**: Store insights in `memory/campaigns/<id>/retrospective.json`
- **Updates campaign state**: Mark campaign as `completed` with final outcomes

**Retrospective Template**:

```markdown
# Campaign Retrospective: {{ campaign_name }}

## Objectives vs Outcomes
- **Target**: {{ target_description }}
- **Achieved**: {{ actual_outcome }}
- **Success Rate**: {{ percentage }}%

## Timeline
- **Planned Duration**: {{ planned_days }} days
- **Actual Duration**: {{ actual_days }} days
- **Variance**: {{ variance }}% {{ ahead/behind }}

## What Worked Well
- {{ success_factor_1 }}
- {{ success_factor_2 }}

## Challenges Encountered
- {{ challenge_1 }}: {{ resolution }}
- {{ challenge_2 }}: {{ resolution }}

## Recommendations for Future Campaigns
1. {{ recommendation_1 }}
2. {{ recommendation_2 }}

## Worker Performance Summary
| Worker | Items Completed | Avg Time | Success Rate |
|--------|----------------|----------|--------------|
| {{ worker_1 }} | {{ count }} | {{ time }} | {{ rate }}% |
```

### 5. Cross-Campaign Analytics

**Current Limitation**: No visibility across multiple campaigns for portfolio management.

**Proposed Enhancement**: Add a campaign analytics dashboard that:

- **Aggregates metrics across campaigns**: Show portfolio-level health
- **Identifies patterns**: Highlight common blockers, top-performing workflows
- **Enables comparison**: Compare similar campaigns' performance
- **Supports resource allocation**: Help prioritize which campaigns need attention

**Dashboard Metrics**:

- Total active campaigns
- Overall completion rate
- Average velocity across campaigns
- Top blockers affecting multiple campaigns
- Worker utilization across campaigns

## Implementation Priority

**High Priority** (Immediate Value):
1. Summarized campaign reports (Epic issue updates)
2. Enhanced metrics integration (adaptive rate limiting)

**Medium Priority** (Near-term):
3. Campaign learning system (structured insights)
4. Campaign retrospectives (completion workflow)

**Low Priority** (Future):
5. Cross-campaign analytics (portfolio dashboard)

## Configuration Examples

### Enable Summarized Reporting

```yaml
# .github/workflows/my-campaign.campaign.md
governance:
  # ... existing governance ...
  reporting:
    enabled: true
    frequency: 10  # Generate report every 10 runs
    format: "markdown"
    post-to-epic: true
```

### Enable Learning Capture

```yaml
# .github/workflows/my-campaign.campaign.md
learning:
  enabled: true
  categories:
    - discovery_efficiency
    - worker_performance
    - governance_tuning
  share-with-campaigns:
    - "similar-campaign-*"  # Share learnings with similar campaigns
```

## Next Steps

To implement these improvements:

1. **Start with metrics aggregation**: Build utility functions to read and analyze historical metrics snapshots
2. **Add report generation**: Create markdown report templates and integrate with Epic issue comments
3. **Implement learning capture**: Define learning schema and storage format
4. **Build retrospective workflow**: Create workflow that triggers on campaign completion
5. **Design analytics dashboard**: Plan portfolio-level metrics and visualization

## Feedback Welcome

These are proposed enhancements based on analysis of current campaign architecture. Feedback and additional ideas are welcome—please open an issue or discussion to share your thoughts.
