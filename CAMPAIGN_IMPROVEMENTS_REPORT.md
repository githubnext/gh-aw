# Campaign System Improvements Report

**Date:** 2026-01-05  
**Purpose:** Document improvements needed for campaign documentation and identify opportunities for enhanced reporting and learning capabilities.

---

## Executive Summary

This report analyzes the current state of GitHub Agentic Workflows campaign system, identifies documentation gaps between implementation and documentation, and proposes improvements for campaign reporting, analytics, and knowledge extraction.

### Key Findings

1. **Documentation Gaps:** Recent orchestrator implementation changes (precomputed discovery, Epic issue management) are not fully reflected in user-facing documentation
2. **Reporting Opportunities:** Campaign metrics are collected but not aggregated into actionable insights
3. **Learning Extraction:** No systematic approach to extract learnings across campaigns or workflow runs
4. **Cross-Campaign Analytics:** Limited visibility into portfolio-level performance and trends

---

## Section 1: Documentation Update Requirements

### 1.1 Precomputed Discovery System

**Current State:**
- Orchestrator now uses precomputed discovery manifests (`./.gh-aw/campaign.discovery.json`)
- Discovery runs BEFORE the agent in a separate step
- Manifest includes normalized metadata, cursor info, and summary counts

**Documentation Gap:**
- User-facing docs still imply agents perform discovery during Phase 1
- Discovery manifest schema and structure not documented
- Relationship between discovery step and agent execution unclear

**Required Updates:**

**File: `docs/src/content/docs/guides/campaigns/specs.md`**
Add new section after "Compilation and orchestrators":

```markdown
## Discovery System

Campaign orchestrators use a two-phase discovery approach:

### Phase 0: Precomputed Discovery

Before the agent executes, a JavaScript-based discovery step:
1. Loads cursor from repo-memory (if exists)
2. Searches for worker outputs using tracker labels and/or tracker-id markers
3. Applies pagination budgets from governance configuration
4. Normalizes discovered items with consistent metadata
5. Writes discovery manifest: `./.gh-aw/campaign.discovery.json`
6. Updates cursor in repo-memory for next run

### Discovery Manifest Schema

```json
{
  "schema_version": "v1",
  "campaign_id": "campaign-id",
  "generated_at": "2026-01-05T14:00:00Z",
  "discovery": {
    "total_items": 45,
    "cursor": {
      "last_updated_at": "2026-01-05T13:30:00Z",
      "last_item_id": "issue-123"
    }
  },
  "summary": {
    "needs_add_count": 5,
    "needs_update_count": 10,
    "open_count": 25,
    "closed_count": 20
  },
  "items": [
    {
      "url": "https://github.com/org/repo/issues/123",
      "content_type": "issue",
      "number": 123,
      "repo": "org/repo",
      "created_at": "2026-01-01T10:00:00Z",
      "updated_at": "2026-01-05T12:00:00Z",
      "state": "open",
      "title": "Example issue"
    }
  ]
}
```

### Phase 1: Agent Processing

The agent reads the precomputed manifest instead of performing GitHub searches:
1. Reads `./.gh-aw/campaign.discovery.json`
2. Processes normalized items deterministically
3. Makes decisions based on explicit GitHub state
4. Executes writes according to governance budgets

### Benefits

- **Deterministic:** Same inputs always produce same outputs
- **Efficient:** Reduces redundant GitHub API calls
- **Traceable:** Discovery logic separated from agent decisions
- **Resumable:** Cursor enables incremental processing across runs
```

**File: `docs/src/content/docs/guides/campaigns/getting-started.md`**
Update step 4 to clarify discovery:

```markdown
## 4) Run the orchestrator

Trigger the orchestrator workflow from GitHub Actions. The orchestrator runs in two phases:

**Phase 0 - Discovery (before agent):**
- Precomputes discovery using JavaScript step
- Finds tracker-labeled issues/PRs
- Writes discovery manifest to `./.gh-aw/campaign.discovery.json`
- Updates cursor in repo-memory

**Phase 1+ - Orchestration (agent):**
- Reads discovery manifest
- Adds items to the Project board
- Updates fields/status based on GitHub state
- Posts progress report
- Writes metrics snapshot

The discovery phase eliminates the need for agents to perform GitHub-wide searches, making orchestration more efficient and deterministic.
```

### 1.2 Epic Issue Management (Phase 0)

**Current State:**
- Orchestrators create/manage Epic issues as campaign parents
- Epic serves as narrative hub with campaign_id marker
- All work issues should be sub-issues of the Epic
- Epic creation documented in `orchestrator_instructions.md` template

**Documentation Gap:**
- Not mentioned in user-facing documentation
- Relationship between Epic and work items unclear
- Epic issue creation behavior not documented

**Required Updates:**

**File: `docs/src/content/docs/guides/campaigns/project-management.md`**
Add new section after "Recommended Custom Fields for Campaigns":

```markdown
## Campaign Epic Issue

Each campaign automatically creates and manages an Epic issue that serves as the campaign's parent and narrative hub.

### Epic Issue Structure

**Created automatically on first orchestrator run:**
- **Title:** Campaign name (or "Campaign: {id}" if name not specified)
- **Labels:** `epic`, `type:epic`
- **Body contains:**
  - Campaign objective
  - Project board URL
  - Worker workflow list
  - Campaign ID marker for discovery
- **Added to project board** with campaign_id field

**Purpose:**
- Narrative context for the entire campaign
- Parent issue for all campaign work items
- Central place for campaign-level discussions
- Discovery anchor via `campaign_id: {id}` marker in body

### Work Item Hierarchy

**Issues:**
- All campaign work issues should be created as **sub-issues** of the Epic
- Sub-issue relationship maintained via GitHub's native sub-issues feature
- Worker workflows create sub-issues using the Epic as parent

**Pull Requests:**
- Cannot be sub-issues (GitHub limitation)
- Must reference related issue via standard GitHub linking (e.g., "Closes #123")
- Linked to project board via campaign_id field

**Field-Based Grouping:**
- Worker grouping uses the `worker_workflow` project field
- Do NOT re-parent issues based on worker assignment
- Epic remains the single parent for all campaign work items

### Finding the Epic

Search for issues with:
- Label: `epic` or `type:epic`
- Body contains: `campaign_id: {campaign-id}`

Example:
```bash
gh issue list --label epic --search "campaign_id: framework-upgrade"
```
```

### 1.3 Discovery Budget Configuration

**Current State:**
- Governance includes `max-discovery-items-per-run` and `max-discovery-pages-per-run`
- Defaults: 100 items, 10 pages
- Controls precomputed discovery pagination

**Documentation Gap:**
- Discovery budgets not clearly explained
- Relationship to write budgets unclear
- When to adjust budgets not documented

**Required Updates:**

**File: `docs/src/content/docs/guides/campaigns/specs.md`**
Expand governance section:

```markdown
## Governance (pacing & safety)

Use `governance` to keep orchestration predictable and reviewable:

```yaml
governance:
  # Discovery budgets (controls Phase 0 precomputation)
  max-discovery-items-per-run: 100      # Max items to discover per run
  max-discovery-pages-per-run: 10       # Max API pages to fetch per run
  
  # Write budgets (controls Phase 3 execution)
  max-new-items-per-run: 10             # Max new items to add to project
  max-project-updates-per-run: 50       # Max total project updates
  max-comments-per-run: 10              # Max comments to post
  
  # Behavior controls
  opt-out-labels: ["campaign:skip"]     # Items to exclude
  do-not-downgrade-done-items: true     # Prevent Done → other transitions
```

### Budget Guidelines

**Discovery Budgets:**
- **Purpose:** Control API usage during discovery phase
- **`max-discovery-items-per-run`:** Total items to process (default: 100)
- **`max-discovery-pages-per-run`:** API pagination limit (default: 10 pages)
- **When to increase:** Large campaigns with many tracked items
- **When to decrease:** API rate limit concerns or slower campaigns

**Write Budgets:**
- **Purpose:** Control project updates and comment activity
- **`max-new-items-per-run`:** Prevent overwhelming project board with additions
- **`max-project-updates-per-run`:** Total field updates allowed
- **`max-comments-per-run`:** Limit comment activity
- **When to increase:** Fast-moving campaigns needing aggressive updates
- **When to decrease:** Human review capacity constraints

**Budget Interaction:**
Discovery budgets affect what's available for processing; write budgets control what actually gets updated. Set discovery budgets higher than write budgets to maintain a processing backlog.

**Example Configuration:**

Small, careful campaign:
```yaml
governance:
  max-discovery-items-per-run: 50
  max-discovery-pages-per-run: 5
  max-new-items-per-run: 5
  max-project-updates-per-run: 10
  max-comments-per-run: 5
```

Large, aggressive campaign:
```yaml
governance:
  max-discovery-items-per-run: 200
  max-discovery-pages-per-run: 20
  max-new-items-per-run: 20
  max-project-updates-per-run: 100
  max-comments-per-run: 20
```
```

### 1.4 Metrics Schema Requirements

**Current State:**
- Orchestrator instructions require specific metrics fields
- Schema: campaign_id, date, tasks_total, tasks_completed (minimum)
- Optional: tasks_in_progress, tasks_blocked, velocity_per_day, estimated_completion

**Documentation Gap:**
- Metrics schema not documented in user-facing docs
- Required vs optional fields unclear
- File naming convention not specified

**Required Updates:**

**File: `docs/src/content/docs/guides/campaigns/specs.md`**
Update "Durable state (repo-memory)" section:

```markdown
## Durable state (repo-memory)

If you use repo-memory for campaigns, standardize on one layout so runs are comparable:

- `memory/campaigns/<campaign-id>/cursor.json` - Checkpoint for discovery
- `memory/campaigns/<campaign-id>/metrics/<date>.json` - Daily metrics snapshots

### Cursor File

Opaque JSON object maintained by orchestrator. Contains:
- `last_updated_at`: Timestamp of most recent item processed
- `last_item_id`: Identifier for resumption
- Additional campaign-specific state

**Do not manually edit cursor files.** The orchestrator manages them automatically.

### Metrics Snapshots

One JSON file per run, named with UTC date: `YYYY-MM-DD.json`

**Required Fields (must always be present):**
```json
{
  "campaign_id": "campaign-id",
  "date": "2026-01-05",
  "tasks_total": 50,
  "tasks_completed": 25
}
```

**Optional Fields (include when available):**
```json
{
  "campaign_id": "campaign-id",
  "date": "2026-01-05",
  "tasks_total": 50,
  "tasks_completed": 25,
  "tasks_in_progress": 15,
  "tasks_blocked": 3,
  "velocity_per_day": 3.5,
  "estimated_completion": "2026-02-15"
}
```

**Field Descriptions:**
- `campaign_id`: Campaign identifier (must match spec)
- `date`: UTC date in YYYY-MM-DD format
- `tasks_total`: Total number of tasks in campaign scope (≥ 0)
- `tasks_completed`: Completed task count (≥ 0, ≤ tasks_total)
- `tasks_in_progress`: Currently active tasks (optional)
- `tasks_blocked`: Tasks awaiting resolution (optional)
- `velocity_per_day`: Average completion rate (optional)
- `estimated_completion`: Projected completion date YYYY-MM-DD (optional)

**Validation:**
Campaign tooling enforces that repo-memory writes include:
1. Valid cursor file at configured path
2. At least one metrics snapshot for current run
3. All required fields present in metrics snapshots

Typical wiring in the spec:
```yaml
memory-paths:
  - "memory/campaigns/framework-upgrade/cursor.json"
  - "memory/campaigns/framework-upgrade/metrics/*.json"
metrics-glob: "memory/campaigns/framework-upgrade/metrics/*.json"
cursor-glob: "memory/campaigns/framework-upgrade/cursor.json"
```

### Metrics Analysis

With append-only metrics, you can:
- Track daily progress and velocity trends
- Calculate burn-down and burn-up charts
- Identify blocked periods and bottlenecks
- Project completion dates based on historical velocity
- Compare performance across campaigns
```

---

## Section 2: Campaign Reporting Improvements

### 2.1 Current State Analysis

**What Exists:**
- Individual orchestrator runs post progress reports
- Metrics snapshots stored in repo-memory
- Discovery manifest includes summary counts
- Project board provides real-time status

**Gaps:**
- No aggregated reporting across runs
- No trend analysis or velocity calculations
- No cross-campaign portfolio view
- Limited insight extraction from historical data

### 2.2 Proposed: Campaign Summary Reports

**Feature:** Automated campaign summary generation from metrics snapshots

**Implementation Approach:**

Create new workflow: `.github/workflows/campaign-reporter.md`

```yaml
---
name: "Campaign Reporter"
description: "Generate summarized reports from campaign metrics"
engine: copilot
on:
  workflow_dispatch:
    inputs:
      campaign-id:
        description: 'Campaign ID to report on'
        required: true
  schedule:
    - cron: "0 10 * * 1"  # Weekly Monday 10am UTC
tools:
  github:
    toolsets: [default]
  repo-memory:
    - id: campaigns
      branch-name: memory/campaigns
      file-glob: ["*/metrics/*.json"]
  bash: ["*"]
safe-outputs:
  create-discussion:
    max: 1
---

# Campaign Reporter

Generate comprehensive summary reports from campaign metrics snapshots.

## Objectives

1. Aggregate metrics across multiple runs
2. Calculate trends, velocity, and projections
3. Identify patterns and anomalies
4. Generate actionable insights
5. Create visual burn-down/burn-up representations

## Report Structure

### Executive Summary
- Campaign objective and current status
- Overall progress percentage
- Estimated completion date
- Key blockers and risks

### Metrics Analysis
- Tasks total/completed/in-progress/blocked trends
- Daily velocity chart (ASCII art or markdown table)
- Burn-down projection
- Historical velocity trends

### Insights
- Performance trends (improving/declining/stable)
- Bottleneck identification
- Recommendations for governance adjustments
- Comparison to similar campaigns

### Next Steps
- Immediate actions needed
- Governance changes to consider
- Resource allocation suggestions

## Implementation

Read all metrics files for the campaign from repo-memory:
```bash
ls -1 /tmp/gh-aw/repo-memory/campaigns/$CAMPAIGN_ID/metrics/*.json
```

Parse JSON, calculate:
- Completion rate trend
- Average daily velocity
- Days until completion (projected)
- Variance in velocity
- Blocked task percentage

Generate markdown report and post as discussion with label: `campaign-report`

## Output Format

Post summary as GitHub Discussion:

```markdown
## Campaign Summary Report: {campaign-name}
**Report Date:** {date}
**Campaign Period:** {start-date} to {latest-date}
**Status:** {On Track / At Risk / Blocked}

### Progress Overview

| Metric | Current | Change (7d) | Change (30d) |
|--------|---------|-------------|--------------|
| Tasks Completed | 45/100 | +12 | +38 |
| Completion Rate | 45% | +12% | +38% |
| Avg Velocity | 3.2/day | +0.5 | +0.3 |
| Blocked Tasks | 3 | +1 | -2 |

### Burn-Down Chart

```text
100 │                                        ◉ Target
    │                                   ◉ 
 75 │                              ◉
    │                         ◉    
 50 │                    ◉ ◉  
    │               ◉ ◉        ● Actual
 25 │          ● ●              
    │     ● ●                   
  0 │ ●                              
    └────────────────────────────────────
     Week 1  Week 2  Week 3  Week 4
```

### Key Insights

✅ **Strengths:**
- Velocity increased by 15% over last 7 days
- Blocked task count decreased
- Consistent daily progress

⚠️ **Concerns:**
- Current velocity suggests completion in 18 days vs target 14 days
- 3 tasks blocked for > 5 days
- Weekend velocity drops to near-zero

💡 **Recommendations:**
1. Increase `max-project-updates-per-run` from 10 to 15
2. Investigate blocked tasks and create unblock issues
3. Consider weekend runs for time-sensitive campaigns

### Projected Completion

**At current velocity:** 2026-01-23 (18 days)
**At target velocity:** 2026-01-19 (14 days)
**Confidence:** Moderate (velocity variance: ±0.8 tasks/day)
```
```

**Benefits:**
- Historical context for decision-making
- Early warning of issues
- Data-driven governance adjustments
- Stakeholder communication material

### 2.3 Proposed: Cross-Campaign Analytics

**Feature:** Portfolio-level campaign analytics and insights

**Implementation Approach:**

Enhance Campaign Manager workflow with analytics capabilities:

```markdown
## Campaign Portfolio Analytics

Analyze all active campaigns to identify:

### Performance Patterns
- High-performing vs struggling campaigns
- Common blockers across campaigns
- Velocity correlations with governance settings
- Resource utilization patterns

### Comparative Analysis

| Campaign | Progress | Velocity | Blocked | Status | Days to Complete |
|----------|----------|----------|---------|--------|------------------|
| docs-quality | 67% | 4.2/day | 2 | ✅ On Track | 12 |
| framework-upgrade | 34% | 2.1/day | 8 | ⚠️ At Risk | 31 |
| security-audit | 89% | 3.8/day | 0 | ✅ On Track | 3 |

### Resource Optimization
- Identify over/under-utilized workers
- Suggest budget redistributions
- Recommend campaign prioritization

### Trend Detection
- Campaigns consistently falling behind
- Governance patterns that improve velocity
- Worker workflows needing attention
- Systemic blockers affecting multiple campaigns
```

**Output:** Weekly executive discussion with portfolio summary

**Benefits:**
- Strategic resource allocation
- Early risk identification
- Best practice identification
- Cross-campaign learning

### 2.4 Proposed: Learning Extraction System

**Feature:** Systematic extraction and documentation of campaign learnings

**Implementation Approach:**

Create learning extraction workflow that analyzes:

1. **Successful Patterns:**
   - What governance settings led to on-time completion?
   - Which worker combinations were most effective?
   - What project board configurations helped visibility?

2. **Failure Analysis:**
   - Common causes of campaign delays
   - Bottleneck patterns
   - Governance settings that hindered progress

3. **Worker Performance:**
   - Which workers create highest-quality outputs?
   - Which workers have highest success rates?
   - Which workers work well together?

4. **Recommendations:**
   - Template campaign configurations for common use cases
   - Governance presets (aggressive/balanced/conservative)
   - Worker workflow combinations that work well

**Output Format:**

Create structured learning documents:

```markdown
# Campaign Learning: {campaign-id}

## Campaign Profile
- Duration: {days}
- Total Tasks: {count}
- Average Velocity: {rate}
- Final Status: {success/partial/failed}

## What Worked Well

### Governance Configuration
- `max-discovery-items-per-run: 100` - Provided good balance
- `max-project-updates-per-run: 50` - Never hit limit, could reduce

### Worker Performance
- `daily-doc-updater`: 95% success rate, high-quality outputs
- `unbloat-docs`: 87% success rate, occasional over-aggressive edits

### Project Board Setup
- Roadmap view with swimlanes by worker was highly effective
- Start/End date automation saved manual effort

## What Could Improve

### Challenges Faced
- Discovery budget too low in first 2 weeks, increased to resolve
- Blocked tasks not surfaced quickly enough
- Weekend gaps in progress

### Recommendations for Similar Campaigns
1. Start with higher discovery budget, tune down after initial discovery
2. Configure Slack/Teams notifications for blocked tasks > 3 days
3. Consider weekend runs for time-sensitive initiatives
4. Use aggressive governance settings for first 2 weeks, then moderate

## Key Metrics Summary

| Metric | Target | Actual | Variance |
|--------|--------|--------|----------|
| Completion Time | 30 days | 28 days | -2 days ✅ |
| Total Tasks | 100 | 98 | -2 ✅ |
| Avg Velocity | 3.3/day | 3.5/day | +0.2 ✅ |
| Blocked Tasks (peak) | <5 | 8 | +3 ⚠️ |

## Reusable Assets

### Campaign Spec Template
[Link to refined spec based on learnings]

### Worker Workflow Improvements
- Added better error handling to daily-doc-updater
- Enhanced unbloat-docs to be less aggressive

### Project Board Template
[Link to project board export]
```

**Storage:** Store in `memory/campaigns/{id}/learnings.md`

**Benefits:**
- Institutional knowledge capture
- Faster campaign setup for similar initiatives
- Continuous improvement feedback loop
- Training material for new campaign authors

---

## Section 3: Implementation Roadmap

### Phase 1: Documentation Updates (Immediate)
**Priority:** High  
**Effort:** 2-3 hours  
**Impact:** High - Aligns docs with current implementation

Tasks:
- [ ] Update specs.md with discovery system documentation
- [ ] Update getting-started.md with two-phase orchestration
- [ ] Update project-management.md with Epic issue documentation
- [ ] Update specs.md with governance budget guidelines
- [ ] Update specs.md with metrics schema documentation

### Phase 2: Campaign Summary Reports (Short-term)
**Priority:** High  
**Effort:** 1-2 days  
**Impact:** High - Immediate value to campaign users

Tasks:
- [ ] Create campaign-reporter.md workflow
- [ ] Implement metrics aggregation logic
- [ ] Create burn-down chart generator (ASCII art in markdown)
- [ ] Implement trend analysis calculations
- [ ] Create report template and formatting
- [ ] Test with existing campaign metrics

### Phase 3: Learning Extraction (Medium-term)
**Priority:** Medium  
**Effort:** 2-3 days  
**Impact:** Medium - Long-term continuous improvement

Tasks:
- [ ] Define learning document schema
- [ ] Create learning extraction workflow
- [ ] Implement pattern detection algorithms
- [ ] Create recommendation engine based on historical data
- [ ] Document learning extraction process
- [ ] Create example learnings from existing campaigns

### Phase 4: Cross-Campaign Analytics (Long-term)
**Priority:** Medium  
**Effort:** 3-5 days  
**Impact:** Medium - Strategic planning value

Tasks:
- [ ] Enhance campaign-manager with analytics capabilities
- [ ] Implement portfolio-level metrics aggregation
- [ ] Create comparative analysis reports
- [ ] Build trend detection across campaigns
- [ ] Create executive summary dashboard (markdown tables)
- [ ] Implement resource optimization recommendations

---

## Section 4: Success Metrics

### Documentation Quality
- [ ] All new orchestrator features documented
- [ ] User feedback indicates docs are clear and complete
- [ ] Example configurations include all new fields
- [ ] Troubleshooting guides address common issues

### Reporting Effectiveness
- [ ] Campaign summary reports generated weekly without errors
- [ ] Report insights lead to 2+ governance adjustments per month
- [ ] Stakeholders use reports for decision-making
- [ ] Reports identify blockers before they impact deadlines

### Learning Extraction Value
- [ ] 5+ learning documents created from completed campaigns
- [ ] New campaigns reference learnings when configuring
- [ ] Governance presets based on learnings adopted by 50% of campaigns
- [ ] Campaign completion rates improve by 15% using learning-based configs

### Portfolio Analytics Impact
- [ ] Executive summaries inform strategic decisions
- [ ] Resource reallocation based on analytics improves overall velocity
- [ ] Cross-campaign patterns identified and addressed
- [ ] Portfolio-level success rate improves

---

## Section 5: Recommendations

### Immediate Actions (This Week)

1. **Update Campaign Documentation** - Highest priority, required for user clarity
   - Focus on discovery system, Epic issues, governance budgets
   - Add examples and clear explanations
   - Update getting-started flow

2. **Create Basic Campaign Reporter** - High value, moderate effort
   - Start with simple metrics aggregation
   - Generate weekly summary for one campaign as proof-of-concept
   - Iterate based on feedback

### Short-term Actions (This Month)

3. **Enhance Reporting with Trends** - Builds on basic reporter
   - Add velocity calculations and projections
   - Implement burn-down visualization
   - Generate actionable recommendations

4. **Document Learning Extraction Process** - Foundation for improvement
   - Define what constitutes a "learning"
   - Create template for learning documents
   - Extract learnings from 2-3 completed campaigns manually

### Long-term Actions (This Quarter)

5. **Automate Learning Extraction** - Scale learning capture
   - Build workflow to extract learnings automatically
   - Create pattern detection algorithms
   - Generate recommendations for new campaigns

6. **Build Portfolio Analytics** - Strategic planning capability
   - Aggregate metrics across all campaigns
   - Create comparative analysis dashboards
   - Implement resource optimization recommendations

---

## Section 6: Technical Considerations

### Performance
- Metrics aggregation should complete in < 30 seconds
- Discovery manifest processing is already optimized
- Report generation should not exceed GitHub Actions time limits
- Consider caching aggregated metrics for large campaigns

### Scalability
- Design for 50+ active campaigns
- Ensure repo-memory branch doesn't grow unbounded
- Consider archiving old metrics after campaign completion
- Implement pagination for portfolio analytics

### Maintenance
- Reports should be self-maintaining (no manual intervention)
- Learning extraction should be semi-automated (human review for quality)
- Analytics should update automatically as campaigns progress
- Documentation should stay synchronized with implementation changes

### Security & Permissions
- Reports should respect repository visibility
- Cross-campaign analytics should only access permitted campaigns
- Learning documents should not expose sensitive information
- Portfolio views should respect organizational boundaries

---

## Conclusion

The campaign system in GitHub Agentic Workflows has evolved significantly with the introduction of precomputed discovery, Epic issue management, and enhanced governance controls. This report identifies critical documentation gaps and proposes a comprehensive set of improvements focused on:

1. **Immediate documentation updates** to align with current implementation
2. **Enhanced reporting capabilities** to extract value from collected metrics
3. **Systematic learning extraction** to improve future campaign success
4. **Portfolio-level analytics** for strategic decision-making

By implementing these improvements in phases, the campaign system will provide greater visibility, better decision support, and continuous improvement feedback loops that benefit all campaign users.

### Next Steps

1. **Review this report** with stakeholders
2. **Prioritize documentation updates** (Phase 1)
3. **Prototype campaign reporter** (Phase 2)
4. **Gather user feedback** on reporting needs
5. **Iterate based on real-world usage**

---

**Report Authors:** GitHub Copilot  
**Review Status:** Draft  
**Last Updated:** 2026-01-05
