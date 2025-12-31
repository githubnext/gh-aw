---
name: Campaign - Incident Response
description: Coordinate multi-team incident response with SLA tracking, stakeholder updates, and post-mortem
timeout-minutes: 60
strict: true

on:
  workflow_dispatch:
    inputs:
      incident_severity:
        description: 'Incident severity'
        type: choice
        required: true
        options:
          - critical
          - high
          - medium
      incident_description:
        description: 'Brief incident description'
        required: true
      affected_services:
        description: 'Comma-separated list of affected services/repos'
        required: true
      stakeholder_issue:
        description: 'Issue number for stakeholder updates'
        required: false

permissions:
  contents: read
  issues: read
  pull-requests: read

engine: copilot

tools:
  github:
    toolsets: [repos, issues, pull_requests, search]
  repo-memory:
    branch-name: memory/campaigns
    file-glob: "memory/campaigns/incident-*/**"

safe-outputs:
  create-issue:
    labels: [campaign-tracker, incident]
  add-comment: {}
  add-labels: {}
  create-pull-request:
    labels: [campaign-fix, incident]
---

# Campaign: Incident Response Coordination

**Purpose**: Coordinate cross-team incident response that GitHub Actions/basic workflows cannot handle.

**Campaign ID**: `incident-${{ github.run_id }}`

**Severity**: `${{ github.event.inputs.incident_severity }}`

## Why Campaigns Solve This (Not GitHub Actions)

**Problem**: Production incident affecting multiple repos/teams requires:
- Central command & control
- Multi-team coordination
- Real-time status tracking
- Stakeholder communication every 30min
- SLA pressure (time-to-mitigation)
- Post-mortem with business context
- Approval gates for emergency fixes

**GitHub Actions fails**: No cross-team coordination, no SLA tracking, no stakeholder communication pattern

**Basic agentic workflow fails**: Single execution, no orchestration, no persistent command center

**Campaign solves**: Human-AI collaboration + persistent memory + coordination + governance

## Incident Response Steps

### 1. Initialize Command Center (repo-memory)

Create file `memory/campaigns/incident-${{ github.run_id }}/command-center.json`:
```json
{
  "campaign_id": "incident-${{ github.run_id }}",
  "incident_id": "INC-${{ github.run_id }}",
  "severity": "${{ github.event.inputs.incident_severity }}",
  "description": "${{ github.event.inputs.incident_description }}",
  "started": "[timestamp]",
  "affected_services": "${{ github.event.inputs.affected_services }}".split(","),
  "sla_target_minutes": {
    "critical": 30,
    "high": 120,
    "medium": 480
  }["${{ github.event.inputs.incident_severity }}"],
  "status": "investigating",
  "teams_involved": [],
  "timeline": []
}
```

### 2. Create Command Center Issue

Use `create-issue`:

**Title**: `🚨 INCIDENT [${{ github.event.inputs.incident_severity }}]: ${{ github.event.inputs.incident_description }}`

**Labels**: `campaign-tracker`, `tracker:incident-${{ github.run_id }}`, `incident`, `severity:${{ github.event.inputs.incident_severity }}`

**Body**:
```markdown
# Incident Response: INC-${{ github.run_id }}

**Campaign ID**: `incident-${{ github.run_id }}`
**Severity**: ${{ github.event.inputs.incident_severity }}
**Started**: [timestamp]
**SLA Target**: [minutes based on severity]

## Incident Details

**Description**: ${{ github.event.inputs.incident_description }}

**Affected Services**: ${{ github.event.inputs.affected_services }}

## Response Status

**Current Status**: 🔍 Investigating

**Teams Involved**: [AI to identify from affected services]

**Timeline**:
- [timestamp] - Incident detected and campaign initiated

## Coordination

**Query all incident work**:
```bash
gh issue list --search "tracker:incident-${{ github.run_id }}"
gh pr list --search "tracker:incident-${{ github.run_id }}"
```

**Command Center Data**: `memory/campaigns/incident-${{ github.run_id }}/`

## SLA Tracking

**Target Resolution**: [SLA minutes] minutes from start
**Time Remaining**: [calculated]

---

**Updates will be posted here every 30 minutes or on status changes**
```

### 3. AI Analysis Phase

**For each affected service/repo**:

1. **Search for related issues/PRs** (recent errors, deployment failures)
2. **Identify recent changes** (PRs merged in last 48h)
3. **Check dependencies** (related services that might be impacted)
4. **Analyze error patterns** (common failure modes)

**Generate hypothesis**:
- Likely root causes (ranked by probability)
- Affected blast radius (repos, users, services)
- Recommended immediate actions
- Teams that need to be involved

### 4. Human Decision Checkpoint

**AI presents findings to command center issue**:

```markdown
## 🤖 AI Analysis Complete

### Likely Root Causes (by probability)
1. [Cause A] - 60% confidence
   - Evidence: [specific findings]
   - Impact: [scope]
   - Recommended action: [specific fix]
   - Risk: [rollback complexity]
   
2. [Cause B] - 25% confidence
   - Evidence: [specific findings]
   - Recommended action: [specific fix]

### Blast Radius
- Affected repos: [list]
- Affected users: [estimate]
- Downstream dependencies: [list]

### Recommended Actions (Risk-Tiered)

**IMMEDIATE (Low Risk - AI can execute)**:
- [ ] Rollback deployment in repo X
- [ ] Increase timeout in service Y
- [ ] Scale up instances in service Z

**REQUIRES APPROVAL (Medium Risk - needs team lead)**:
- [ ] Apply hotfix PR #123
- [ ] Disable feature flag "new-api"

**REQUIRES EXECUTIVE APPROVAL (High Risk - data/security impact)**:
- [ ] Database migration rollback
- [ ] Traffic failover to backup region

### Teams to Involve
- @team-api - [reason]
- @team-database - [reason]
- @team-frontend - [reason]

---

**🚦 Awaiting incident commander decision on recommended actions**

Reply with:
- "execute-immediate" - Run all low-risk actions
- "approve-medium [action numbers]" - Approve specific medium-risk actions
- "escalate-high" - Escalate high-risk actions to executive
- "investigate-more [cause]" - Deep dive into specific cause
```

### 5. Execute Approved Actions

**Based on human decisions**:

**If "execute-immediate"**:
- Create PRs for rollbacks/hotfixes
- Apply labels: `incident-fix`, `tracker:incident-${{ github.run_id }}`
- Add comments linking back to command center
- Update command center with execution status

**If "approve-medium [actions]"**:
- Create issues for each approved action
- Assign to relevant teams
- Track progress in command center
- Update SLA countdown

**If "escalate-high"**:
- Create executive escalation issue
- Tag VP/CTO for approval
- Document business impact
- Pause until approval received

### 6. Status Updates (Every 30min)

Add comment to command center issue:

```markdown
## 🕐 Status Update: [HH:MM] ([X] minutes since start)

**SLA Status**: ⏰ [minutes remaining] / [target minutes]

**Current Status**: [investigating/mitigating/resolved]

**Actions Completed** (last 30min):
- ✅ [Action A] - [outcome]
- ✅ [Action B] - [outcome]

**Actions In Progress**:
- 🔄 [Action C] - [team] - ETA [time]
- 🔄 [Action D] - [team] - ETA [time]

**Blockers**:
- 🚫 [Blocker A] - [reason] - [action needed]

**Next Steps**:
- [Next action 1]
- [Next action 2]

**Metrics**:
- Error rate: [current vs baseline]
- User impact: [affected users]
- Service health: [status]
```

{% if github.event.inputs.stakeholder_issue %}
Also post sanitized update to stakeholder issue #${{ github.event.inputs.stakeholder_issue }}.
{% endif %}

### 7. Store Timeline Events (repo-memory)

Update `memory/campaigns/incident-${{ github.run_id }}/timeline.json` continuously:
```json
{
  "incident_id": "INC-${{ github.run_id }}",
  "timeline": [
    {
      "timestamp": "[time]",
      "event": "incident_detected",
      "details": "Description",
      "actor": "system"
    },
    {
      "timestamp": "[time]",
      "event": "ai_analysis_complete",
      "hypotheses": ["cause A", "cause B"],
      "confidence": {"cause A": 0.6, "cause B": 0.25},
      "actor": "ai"
    },
    {
      "timestamp": "[time]",
      "event": "human_decision",
      "decision": "execute-immediate",
      "actions_approved": ["rollback X", "scale Y"],
      "actor": "@incident-commander"
    },
    {
      "timestamp": "[time]",
      "event": "action_executed",
      "action": "rollback X",
      "outcome": "success",
      "actor": "ai"
    },
    {
      "timestamp": "[time]",
      "event": "incident_resolved",
      "resolution": "Rollback deployed, error rate back to baseline",
      "duration_minutes": 45,
      "actor": "@incident-commander"
    }
  ]
}
```

### 8. Incident Resolution

When resolved, create final summary in command center:

```markdown
## ✅ Incident Resolved

**Duration**: [X] minutes
**SLA Status**: ✅ Within SLA / ⚠️ Exceeded by [X] minutes

**Root Cause**: [confirmed cause]

**Resolution**: [what fixed it]

**Impact**:
- Affected users: [count]
- Services impacted: [list]
- Duration: [minutes]
- Data loss: Yes/No

**Teams Involved**:
- [Team A] - [contribution]
- [Team B] - [contribution]

**PRs/Fixes Applied**:
- #123 - [description]
- #124 - [description]

**Post-Mortem Required**: Yes

See full timeline and analysis: `memory/campaigns/incident-${{ github.run_id }}/`

---

Campaign closed.
```

### 9. Generate Post-Mortem Template

Create `memory/campaigns/incident-${{ github.run_id }}/post-mortem-template.md`:

```markdown
# Post-Mortem: INC-${{ github.run_id }}

**Date**: [date]
**Severity**: ${{ github.event.inputs.incident_severity }}
**Duration**: [minutes]
**Impact**: [user impact summary]

## Timeline

[AI-generated timeline from timeline.json with human-readable descriptions]

## Root Cause

[Confirmed root cause with technical details]

## What Went Well

- [AI-assisted analysis provided accurate hypothesis in X minutes]
- [Team Y responded quickly with fix]
- [Rollback procedure worked as designed]

## What Went Wrong

- [Detection delay - [X] minutes before incident noticed]
- [Communication gap between teams]
- [Missing monitoring for [specific metric]]

## Action Items

**Prevent Recurrence**:
- [ ] [Action 1] - Owner: [team] - Due: [date]
- [ ] [Action 2] - Owner: [team] - Due: [date]

**Improve Response**:
- [ ] [Action 3] - Owner: [team] - Due: [date]
- [ ] [Action 4] - Owner: [team] - Due: [date]

**Improve Detection**:
- [ ] [Action 5] - Owner: [team] - Due: [date]

## Lessons Learned

[AI-generated insights from timeline analysis]

## Campaign Metrics

- Time to detection: [minutes]
- Time to AI analysis: [minutes]
- Time to human decision: [minutes]
- Time to resolution: [minutes]
- Actions taken: [count]
- Teams coordinated: [count]
```

## Why This Campaign Cannot Be Done Without Campaigns

**Cross-team coordination**: Command center orchestrates multiple teams across repos
**SLA tracking**: Time pressure and deadline monitoring
**Human-in-loop**: AI analyzes, humans decide on risk-tiered actions
**Stakeholder communication**: Regular status updates with business context
**Audit trail**: Complete timeline with business justification for every decision
**Post-mortem**: Learning capture with action items and ownership
**Governance**: Approval gates for high-risk emergency actions

**GitHub Actions**: Can't coordinate across teams, no SLA concept, no stakeholder communication
**Basic workflows**: Single execution, no orchestration, no persistent command center

## Output

Provide summary:
- Campaign ID: `incident-${{ github.run_id }}`
- Incident ID: `INC-${{ github.run_id }}`
- Command center issue: #[number]
- Severity: ${{ github.event.inputs.incident_severity }}
- Status: [investigating/mitigating/resolved]
- Duration: [minutes] / SLA: [target minutes]
- Teams involved: [count]
- Actions taken: [count]
- Memory location: `memory/campaigns/incident-${{ github.run_id }}/`
- Post-mortem template: Ready for human review
