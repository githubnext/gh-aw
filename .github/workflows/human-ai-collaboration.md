---
name: Human-AI Collaboration Campaign
description: AI analyzes and proposes, humans approve based on risk, AI executes with guardrails
timeout-minutes: 30
strict: true

on:
  workflow_dispatch:
    inputs:
      initiative:
        description: 'Initiative name (e.g., Q1-api-modernization)'
        required: true
      scope:
        description: 'Scope to analyze (e.g., repos:api-*, issues:label:tech-debt)'
        required: true

permissions:
  contents: read
  issues: read

engine: copilot

safe-outputs:
  create-issue:
    max: 1  # Only epic for human review

tools:
  github:
    toolsets: [repos, issues, search]
  repo-memory:
    branch-name: memory/campaigns
    file-glob: "memory/campaigns/human-ai-collab-*/**"

---

# Human-AI Collaboration Campaign Pattern

**Core Insight**: Enterprises don't want full automation - they want **AI-assisted decision-making**. AI provides intelligence and execution, humans provide judgment and approval.

**Campaign ID**: `human-ai-collab-${{ github.run_id }}`

**The Pattern**: AI analyzes → Humans decide → AI executes → Humans validate → AI learns

## Phase 1: AI Analysis (This Workflow)

**AI's Role**: Discover, analyze, prioritize, recommend

### 1. Discovery & Analysis

**Analyze scope**: ${{ github.event.inputs.scope }}

**AI discovers**:
- What needs attention (security issues, tech debt, cost waste, etc.)
- Current state metrics (coverage, cost, performance, etc.)
- Risk assessment for each item
- Effort estimates
- Dependencies and blockers
- Business impact analysis

**Store analysis** in `memory/campaigns/human-ai-collab-${{ github.run_id }}/analysis.json`:
```json
{
  "campaign_id": "human-ai-collab-${{ github.run_id }}",
  "initiative": "${{ github.event.inputs.initiative }}",
  "analyzed_at": "[timestamp]",
  "scope": "${{ github.event.inputs.scope }}",
  "findings": {
    "total_items": 87,
    "by_risk": {
      "low": { "count": 45, "items": [...] },
      "medium": { "count": 30, "items": [...] },
      "high": { "count": 10, "items": [...] },
      "critical": { "count": 2, "items": [...] }
    }
  },
  "ai_recommendations": {
    "auto_execute_low_risk": {
      "count": 45,
      "items": [...],
      "rationale": "Zero production impact, fully reversible",
      "estimated_savings": "$X/month"
    },
    "require_approval_medium": {
      "count": 30,
      "items": [...],
      "rationale": "Moderate impact, needs review",
      "estimated_effort": "Y hours"
    },
    "require_committee_high": {
      "count": 10,
      "items": [...],
      "rationale": "High impact, needs architecture review",
      "estimated_risk": "Z"
    },
    "defer_critical": {
      "count": 2,
      "items": [...],
      "rationale": "Too risky for campaign, needs dedicated project"
    }
  },
  "estimated_timeline": "6 weeks",
  "estimated_cost": "$X",
  "expected_roi": "10x"
}
```

### 2. Create Decision Brief for Humans

**Epic Issue**: "🤝 AI Analysis Ready: ${{ github.event.inputs.initiative }}"

**Labels**: `campaign-tracker`, `awaiting-human-decision`, `campaign:human-ai-collab-${{ github.run_id }}`

**Body**:
```markdown
# AI Analysis Complete - Awaiting Human Decisions

**Campaign ID**: `human-ai-collab-${{ github.run_id }}`
**Initiative**: ${{ github.event.inputs.initiative }}
**Analyzed**: [timestamp]
**AI Confidence**: 85%

## 📊 What AI Discovered

**Analyzed**: ${{ github.event.inputs.scope }}
**Found**: 87 items requiring attention

### Breakdown by Risk Level
| Risk | Count | AI Recommendation | Your Decision Needed |
|------|-------|-------------------|---------------------|
| **Critical** | 2 | ⏸️ Defer to dedicated project | ✅ Approve defer / ❌ Include anyway |
| **High** | 10 | 🔍 Requires architecture review | ✅ Approve / ❌ Reject / ⏸️ Defer |
| **Medium** | 30 | ⚠️ Needs team lead approval | ✅ Approve / ❌ Reject |
| **Low** | 45 | ✅ Safe to auto-execute | ✅ Approve / ❌ Review first |

## 🤖 AI Recommendations with Rationale

### Critical Risk (2 items) - DEFER RECOMMENDED

**Item 1**: Migrate auth service from JWT to OAuth2
- **Risk**: High - impacts all users, security-critical
- **Effort**: 3 months, dedicated team needed
- **Business Impact**: Critical service, 24/7 uptime requirement
- **AI Assessment**: Too risky for campaign automation
- **Recommendation**: ⏸️ **DEFER** - Create separate project with dedicated resources
- **Your Decision**: 
  - [ ] ✅ Agree - defer this item
  - [ ] ❌ No - include in campaign (explain why: _________)

**Item 2**: Rewrite notification service in Go
- **Risk**: High - 10M notifications/day dependency
- **Effort**: 2 months, requires extensive testing
- **Business Impact**: Revenue-impacting if broken
- **AI Assessment**: Too large for campaign, needs dedicated focus
- **Recommendation**: ⏸️ **DEFER** - Separate initiative
- **Your Decision**:
  - [ ] ✅ Agree - defer this item
  - [ ] ❌ No - include in campaign (explain why: _________)

### High Risk (10 items) - ARCHITECTURE REVIEW REQUIRED

**Sample**: Update database schema in user service
- **Risk**: High - schema changes affect multiple services
- **Effort**: 2 weeks, requires migration strategy
- **Business Impact**: 5M users affected
- **AI Assessment**: Feasible but needs careful planning
- **Recommendation**: ✅ **APPROVE** if architect reviews migration plan
- **Your Decision**:
  - [ ] ✅ Approve - proceed with architecture review gate
  - [ ] ❌ Reject - not worth the risk
  - [ ] ⏸️ Defer - not right timing

[... 9 more high-risk items with same decision template ...]

### Medium Risk (30 items) - TEAM LEAD APPROVAL NEEDED

**Sample**: Update Express.js from v4 to v5 in 15 API services
- **Risk**: Medium - breaking changes possible
- **Effort**: 1 week, automated with validation
- **Business Impact**: Non-customer-facing, can rollback
- **AI Assessment**: Good candidate for campaign
- **Recommendation**: ✅ **APPROVE** with team lead sign-off per service
- **Your Decision**:
  - [ ] ✅ Approve all 30 medium-risk items
  - [ ] ❌ Reject all medium-risk items
  - [ ] 🔍 Review individually (AI will create sub-issues)

### Low Risk (45 items) - AUTO-EXECUTE RECOMMENDED

**Examples**:
- Remove unused npm dependencies (fully reversible)
- Update README files with current setup instructions
- Add missing license headers to source files
- Fix typos in code comments
- Update copyright years

- **Risk**: Low - no functional impact
- **Effort**: Minutes per item, fully automated
- **Business Impact**: Zero
- **AI Assessment**: Safe to execute without review
- **Recommendation**: ✅ **AUTO-APPROVE** - proceed immediately
- **Your Decision**:
  - [ ] ✅ Approve - auto-execute all low-risk items
  - [ ] ❌ Review first - create issues for human review

## 💰 Business Case

**Total Estimated Value**: $50K/year savings + 20% faster development
**Total Estimated Cost**: $5K (AI + engineering time)
**ROI**: 10x in first year
**Timeline**: 6 weeks (if all approved)

**Breakdown by Approval Decision**:
- If you approve low-risk only: $5K value, 2 weeks
- If you approve low + medium: $20K value, 4 weeks  
- If you approve low + medium + high: $50K value, 6 weeks

## 📋 Next Steps Based on Your Decisions

**After you make decisions** (check boxes above):

1. **Comment on this issue with your decisions** or use reactions:
   - 👍 = Approve campaign with AI recommendations
   - 👀 = Need more information before deciding
   - ❌ = Reject campaign / need major changes

2. **AI will create execution workflow** based on your approvals:
   - Auto-execute: Low-risk items (if approved)
   - Create approval issues: Medium-risk items for team leads
   - Create review issues: High-risk items for architecture team
   - Archive: Deferred/rejected items with rationale

3. **Humans execute approval workflows**:
   - Team leads approve medium-risk items
   - Architects approve high-risk items
   - Each approval triggers AI execution worker

4. **AI executes with guardrails**:
   - Creates PRs with rollback plans
   - Runs tests automatically
   - Monitors for issues
   - Alerts on failures

5. **Humans validate outcomes**:
   - Review PR quality
   - Approve merges
   - Monitor production
   - Provide feedback

6. **AI learns from feedback**:
   - "This pattern failed" → avoid in future
   - "This worked great" → prioritize similar items
   - Updates recommendations for next campaign

## 🚨 Decision Deadline

**Please make decisions within**: 3 business days ([date])
**If no decision**: Campaign auto-executes low-risk items only (safest default)

## 📊 View Full Analysis

```bash
# Complete AI analysis with all details
cat memory/campaigns/human-ai-collab-${{ github.run_id }}/analysis.json

# Risk assessment for each item
cat memory/campaigns/human-ai-collab-${{ github.run_id }}/risk-assessments.json
```

## 🤝 Human-AI Collaboration Model

This campaign demonstrates the ideal pattern:

1. ✅ **AI discovers** - Analyzes 87 items faster than humans
2. ✅ **AI prioritizes** - Risk-based recommendations with rationale
3. ✅ **Humans decide** - Judgment on business context, timing, risk tolerance
4. ✅ **AI executes** - Automated execution within approved guardrails
5. ✅ **Humans validate** - Quality review, production monitoring
6. ✅ **AI learns** - Improves future recommendations based on outcomes

**Not**: "AI does everything" (too risky)
**Not**: "Humans do everything" (too slow)
**Yes**: "AI amplifies human judgment at scale"

---

**Campaign Status**: 🟡 AWAITING HUMAN DECISIONS
**Your Action Required**: Review recommendations above and check decision boxes
**Timeline**: Decisions needed by [date] to meet 6-week target
```

### 3. Wait for Human Decisions

**This workflow completes here.** Execution happens in separate workflows triggered by human approval.

**Possible outcomes**:
- Humans approve via issue comments/reactions
- Execution workflows trigger based on approval labels
- AI creates appropriate issues based on risk tier
- Workers execute approved items with guardrails

## Phase 2: Risk-Tiered Execution (Separate Workflows)

### Low-Risk Auto-Execute Worker
```yaml
campaign-execute-low-risk.md:
  trigger: Issue labeled "approved:low-risk"
  permissions: write (within safe-outputs)
  
  For each approved low-risk item:
    1. Execute change (update docs, fix typos, etc.)
    2. Create PR with "ai-automated" label
    3. Auto-merge if tests pass
    4. Report success to epic
```

### Medium-Risk Approval Worker
```yaml
campaign-execute-medium-risk.md:
  trigger: Issue labeled "approved:medium-risk"
  permissions: read
  
  For each approved medium-risk item:
    1. Create approval issue for team lead
    2. Include AI analysis and recommendation
    3. Team lead adds "approved" label
    4. Worker creates PR
    5. Requires human merge approval
```

### High-Risk Review Worker
```yaml
campaign-execute-high-risk.md:
  trigger: Issue labeled "approved:high-risk"
  permissions: read
  
  For each approved high-risk item:
    1. Create architecture review issue
    2. Include detailed technical analysis
    3. Architect reviews and approves
    4. Worker creates PR with extensive testing
    5. Requires multiple approvals to merge
```

## Phase 3: Learning & Feedback

**Monitor workflow** tracks outcomes:
```yaml
campaign-monitor-learn.md:
  runs: daily
  
  For this campaign:
    1. Track success/failure rates by risk tier
    2. Identify patterns (what worked, what didn't)
    3. Update risk models for future campaigns
    4. Report learning to humans in epic issue
    
  Learnings stored in:
    memory/campaigns/human-ai-collab-${{ github.run_id }}/learnings.json
```

**Example learnings**:
```json
{
  "campaign_id": "human-ai-collab-${{ github.run_id }}",
  "completed_at": "[timestamp]",
  "outcomes": {
    "low_risk": {
      "attempted": 45,
      "successful": 44,
      "failed": 1,
      "success_rate": "97.8%",
      "failure_reason": "Test flake unrelated to change"
    },
    "medium_risk": {
      "attempted": 30,
      "successful": 27,
      "failed": 2,
      "deferred_by_human": 1,
      "success_rate": "90%",
      "failures": [
        {
          "item": "Update Express v4→v5",
          "reason": "Breaking change in middleware",
          "learning": "Always check changelog for Express major versions"
        }
      ]
    },
    "high_risk": {
      "attempted": 10,
      "successful": 8,
      "failed": 1,
      "deferred_by_human": 1,
      "success_rate": "80%"
    }
  },
  "ai_learnings": {
    "patterns_that_worked": [
      "Dependency updates with no config changes = safe",
      "Documentation updates = always low risk"
    ],
    "patterns_that_failed": [
      "Express major version updates = underestimated risk",
      "Database schema changes = need more human review time"
    ],
    "recommendation_accuracy": {
      "low_risk": "97.8% accurate",
      "medium_risk": "93.3% accurate", 
      "high_risk": "80% accurate"
    },
    "improvements_for_next_time": [
      "Increase risk tier for major version updates",
      "Add 1 week buffer for database changes",
      "Low-risk recommendations were spot-on"
    ]
  },
  "human_feedback": {
    "satisfaction": "85% satisfied with AI recommendations",
    "comments": [
      "AI correctly identified risky items",
      "Would like more context on effort estimates",
      "Auto-execution of low-risk saved huge time"
    ]
  }
}
```

## Key Principles

### 1. **AI Proposes, Humans Dispose**
- AI generates recommendations
- Humans make final decisions
- AI respects human judgment

### 2. **Risk-Based Approval Chains**
- Low risk → auto-execute
- Medium risk → team lead approval
- High risk → architecture review
- Critical risk → dedicated project

### 3. **Guardrails Always Active**
- safe-outputs prevent dangerous operations
- Tests must pass before merge
- Rollback plans required
- Monitoring for issues

### 4. **Transparency & Explainability**
- AI explains its reasoning
- Risk assessments documented
- Decisions traceable
- Outcomes visible

### 5. **Continuous Learning**
- Capture what worked/failed
- Improve risk models
- Better recommendations next time
- Share learnings across organization

## Output

AI analysis complete and ready for human review:
- **Campaign ID**: `human-ai-collab-${{ github.run_id }}`
- **Epic Issue**: #[number] - awaiting your decisions
- **Items Analyzed**: 87 (2 critical, 10 high, 30 medium, 45 low risk)
- **AI Recommendations**: Ready for review
- **Decision Deadline**: [3 business days]
- **Analysis Data**: `memory/campaigns/human-ai-collab-${{ github.run_id }}/analysis.json`

**Your action**: Review epic issue #[number] and make approval decisions

**Once you approve**:
- Low-risk items execute automatically
- Medium/high-risk items create approval workflows
- You validate outcomes
- AI learns from results

**This is the future**: AI provides intelligence and execution, humans provide judgment and accountability.
