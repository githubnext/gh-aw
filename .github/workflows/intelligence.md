---
name: Campaign Intelligence System
description: Analyze all past campaigns to generate insights, recommendations, and organizational learning
timeout-minutes: 20
strict: true

on:
  schedule:
    - cron: '0 9 1 * *'  # Monthly on 1st at 9 AM
  workflow_dispatch:
    inputs:
      analysis_type:
        description: 'Type of analysis to run'
        type: choice
        required: false
        default: 'full'
        options:
          - full
          - trends
          - recommendations
          - benchmarks

permissions:
  contents: read
  issues: read

engine: copilot

safe-outputs:
  create-issue:
    max: 1  # Intelligence report issue

tools:
  github:
    toolsets: [repos, issues, search]
  repo-memory:
    branch-name: memory/campaigns
    file-glob: "memory/campaigns/**"  # Read all campaign data

imports:
  - shared/trends.md

---

# Campaign Intelligence System

**Purpose**: Analyze all completed campaigns to generate organizational intelligence, predict future needs, optimize campaign strategy, and build institutional knowledge.

**The Compounding Value**: Each campaign makes the next one smarter. This workflow transforms individual campaign learnings into organizational intelligence.

## Intelligence Analysis

### 1. Discover All Campaigns

**Query repo-memory** for all campaign data:
```bash
memory/campaigns/
├── incident-*/
├── org-rollout-*/
├── security-compliance-*/
├── human-ai-collab-*/
└── [any other campaign types that have run]
```

**Extract from each campaign**:
- Type (incident-response, org-rollout, security-compliance, human-ai-collab, etc.)
- Duration (start → complete)
- Scope (repos affected, items processed)
- Outcomes (success rate, items completed)
- Cost (AI usage, engineering hours)
- ROI (value delivered vs cost)
- Learnings (what worked, what didn't)
- Human decisions (approval rates, deferrals)

### 1.5. Prepare Metrics for Trend Charts

Using the Python data visualization environment from `shared/trends.md` (which imports `shared/python-dataviz.md`):

- Aggregate a **flat metrics table** across campaigns with columns like:
  - `date` (e.g., campaign completion date)
  - `campaign_id`
  - `type` (incident, rollout, security, human-ai)
  - `tasks_total`, `tasks_completed`, `velocity_per_day`
  - `success_rate`, `roi`, `cost_per_item`
- Write this table to `/tmp/gh-aw/python/data/campaign-metrics.csv` (or `.json`).
- This file will be used for trend visualizations.

### 2. Cross-Campaign Analysis

**Store intelligence** in `memory/intelligence/campaign-analysis-${{ github.run_id }}.json`:

```json
{
  "analysis_date": "[timestamp]",
  "campaigns_analyzed": 23,
  "date_range": "2024-Q1 to 2025-Q4",
  
  "by_type": {
    "incident_response": {
      "count": 8,
      "avg_duration_hours": 4.2,
      "avg_success_rate": "94%",
      "avg_time_to_resolution_minutes": 65,
      "avg_cost": "$2,800",
      "total_value_delivered": "$340K",
      "trend": "improving - AI analysis reducing MTTR by 40%"
    },
    "org_rollout": {
      "count": 12,
      "avg_duration_hours": 18.5,
      "avg_success_rate": "91%",
      "avg_repos_affected": 147,
      "avg_cost": "$6,500",
      "time_saved_vs_manual_hours": 120,
      "trend": "improving - dependency detection getting more accurate"
    },
    "security_compliance": {
      "count": 6,
      "avg_duration_weeks": 3.5,
      "avg_success_rate": "98%",
      "avg_vulnerabilities_fixed": 87,
      "avg_cost": "$8,200",
      "audit_pass_rate": "100%",
      "trend": "stable - maintaining high audit pass rate"
    },
    "human_ai_collab": {
      "count": 2,
      "avg_duration_weeks": 5.5,
      "human_approval_rate": "78%",
      "ai_recommendation_accuracy": "91%",
      "trend": "learning - AI getting more accurate"
    }
  },
  
  "organizational_trends": {
    "velocity": {
      "2024_Q1": "15 items/week",
      "2024_Q4": "28 items/week",
      "2025_Q4": "42 items/week",
      "trend": "improving 180% over 2 years"
    },
    "success_rate": {
      "first_5_campaigns": "82%",
      "last_5_campaigns": "94%",
      "trend": "learning from experience"
    },
    "roi": {
      "2024_avg": "8.5x",
      "2025_avg": "13.2x",
      "trend": "improving efficiency"
    },
    "cost_per_item": {
      "2024_avg": "$180/item",
      "2025_avg": "$95/item",
      "trend": "47% cost reduction through learning"
    }
  },
  
  "seasonal_patterns": {
    "Q1": {
      "most_common": "security (compliance deadlines)",
      "avg_duration": "4 weeks",
      "success_rate": "91%"
    },
    "Q2": {
      "most_common": "tech-debt (post-planning)",
      "avg_duration": "6 weeks",
      "success_rate": "88%"
    },
    "Q3": {
      "most_common": "dependency (summer updates)",
      "avg_duration": "2 weeks",
      "success_rate": "92%"
    },
    "Q4": {
      "most_common": "cost-optimization (budget planning)",
      "avg_duration": "5 weeks",
      "success_rate": "95%"
    }
  },
  
  "success_factors": {
    "what_predicts_success": [
      "Clear scope (specific item count) → 96% success",
      "Executive sponsor engagement → 94% success",
      "Staged rollout strategy → 93% success",
      "Human-AI collaboration → 91% success",
      "Budget approved upfront → 89% success"
    ],
    "what_predicts_failure": [
      "Vague scope → 65% success",
      "No executive sponsor → 71% success",
      "Big-bang deployment → 73% success",
      "Full automation (no human review) → 78% success"
    ]
  },
  
  "common_failure_patterns": {
    "underestimated_effort": {
      "frequency": "23% of campaigns",
      "avg_overrun": "40% over estimate",
      "root_cause": "Complex dependencies not discovered upfront",
      "recommendation": "Add 2-week discovery phase before committing to timeline"
    },
    "scope_creep": {
      "frequency": "18% of campaigns",
      "avg_overrun": "65% over scope",
      "root_cause": "Additional items added during execution",
      "recommendation": "Lock scope after approval, defer new items to next campaign"
    },
    "human_bottleneck": {
      "frequency": "15% of campaigns",
      "avg_delay": "1.5 weeks",
      "root_cause": "Approvals waiting for humans",
      "recommendation": "Set SLA for human approvals (48 hours), auto-escalate"
    }
  },
  
  "ai_learning_trends": {
    "risk_assessment_accuracy": {
      "first_5_campaigns": "78%",
      "last_5_campaigns": "91%",
      "improvement": "13 percentage points"
    },
    "effort_estimation_accuracy": {
      "first_5_campaigns": "±45%",
      "last_5_campaigns": "±18%",
      "improvement": "60% more accurate"
    },
    "recommendation_acceptance_rate": {
      "low_risk_items": "97% accepted by humans",
      "medium_risk_items": "78% accepted",
      "high_risk_items": "62% accepted",
      "trend": "Humans trust AI more on low-risk, question high-risk (appropriate)"
    }
  },
  
  "organizational_maturity": {
    "current_level": "Advanced",
    "progression": [
      "2024-Q1: Beginner - First campaigns, learning how it works",
      "2024-Q3: Intermediate - Consistent execution, starting to optimize",
      "2025-Q2: Advanced - Predictable outcomes, continuous improvement",
      "2025-Q4: Expert - Proactive campaign planning, high efficiency"
    ],
    "maturity_indicators": {
      "campaign_success_rate": "94% (Expert level: >90%)",
      "roi": "13.2x (Expert level: >10x)",
      "velocity_improvement": "180% over 2 years (Expert level: >100%)",
      "cost_reduction": "47% (Expert level: >30%)"
    }
  }
}
```

  ### 2.5. Visualize Campaign Trends

  Using the `shared/trends.md` import and the metrics file written to `/tmp/gh-aw/python/data/campaign-metrics.csv`:

  1. Load the metrics into a pandas DataFrame and set `date` as the index.
  2. Generate **trend charts** (velocity, success rate, ROI, cost per item) over time using the examples from `shared/trends.md`.
  3. Save charts under `/tmp/gh-aw/python/charts/`, for example:

    ```python
    plt.savefig('/tmp/gh-aw/python/charts/campaign_velocity_trend.png',
            dpi=300,
            bbox_inches='tight',
            facecolor='white',
            edgecolor='none')
    ```

  4. The shared Python data viz import will automatically upload these PNGs as workflow artifacts, so stakeholders can download high-quality screenshots from the workflow run.

  5. Optionally, copy 1–2 key charts into a stable path in repo-memory (for example, `memory/intelligence/charts/`) and reference them from:
    - A **monthly intelligence issue** (created via `safe-outputs.create-issue`)
    - Per-campaign final report Markdown under `memory/campaigns/.../final-report.md`
    - A pinned "Campaign Intelligence" GitHub Discussion that links to each monthly run

  **Recommended monthly intelligence issue format**:

  - **Title**: `Campaign Intelligence – ${ANALYSIS_YEAR}-${ANALYSIS_MONTH}` (for example, `Campaign Intelligence – 2025-12`)
  - **Body sections**:
    - Summary metrics (campaigns analyzed, success rate, ROI, velocity)
    - Key trends (bullet list with 3–5 bullets)
    - Top recommendations for the next 1–2 quarters
    - Links:
      - To the workflow run that produced this report
      - To the chart artifacts (trend PNGs)
      - To any updated playbooks or campaign specs

### 3. Generate Recommendations

**Predictive intelligence** for future campaigns:

```json
{
  "recommendations": {
    "immediate": [
      {
        "type": "security",
        "confidence": "high",
        "rationale": "Q1 approaching, historically best time for security campaigns",
        "suggested_scope": "75 vulnerabilities (based on your velocity)",
        "estimated_duration": "3.5 weeks (improving from 4.2 week average)",
        "estimated_cost": "$3,800 (trend shows decreasing costs)",
        "estimated_roi": "14x (above your 12x average)",
        "optimal_start_date": "2026-01-15",
        "key_success_factors": [
          "Assign executive sponsor early",
          "Use staged rollout (proven 93% success)",
          "Include 1-week discovery phase"
        ]
      },
      {
        "type": "tech-debt",
        "confidence": "medium",
        "rationale": "Last tech-debt campaign was 6 months ago, code quality metrics declining",
        "suggested_scope": "40 repos (your sweet spot based on past campaigns)",
        "estimated_duration": "6 weeks",
        "warning": "Tech-debt campaigns have high variance - recommend better upfront scoping"
      }
    ],
    
    "avoid": [
      {
        "type": "dependency",
        "timing": "Q4 2025",
        "rationale": "Holiday season - historically 40% higher failure rate due to team availability",
        "alternative": "Schedule for Q1 2026 instead"
      }
    ],
    
    "optimize": [
      {
        "finding": "Cost-optimization campaigns have 18x ROI (highest)",
        "recommendation": "Run cost-optimization campaigns quarterly instead of annually",
        "projected_value": "Additional $75K/year savings"
      },
      {
        "finding": "Human approval bottleneck causes 1.5 week delays",
        "recommendation": "Implement 48-hour SLA for approvals with auto-escalation",
        "projected_impact": "20% faster campaigns"
      },
      {
        "finding": "AI risk assessment now 91% accurate",
        "recommendation": "Increase auto-execution threshold for low-risk items",
        "projected_impact": "30% reduction in human review burden"
      }
    ]
  }
}
```

### 4. Benchmark Against Best Practices

**Compare your organization to patterns**:

```json
{
  "benchmarks": {
    "your_organization": {
      "campaigns_per_year": 12,
      "avg_success_rate": "94%",
      "avg_roi": "13.2x",
      "avg_velocity": "42 items/week",
      "cost_efficiency": "$95/item"
    },
    
    "patterns_observed": {
      "high_performing_orgs": {
        "campaigns_per_year": "10-15",
        "success_rate": ">90%",
        "roi": ">10x",
        "velocity": ">35 items/week"
      },
      "your_status": "High performing - in top 20%",
      "strengths": [
        "Excellent success rate (94% vs 90% benchmark)",
        "Outstanding ROI (13.2x vs 10x benchmark)",
        "High velocity (42 vs 35 benchmark)"
      ],
      "opportunities": [
        "Could run 15 campaigns/year based on velocity",
        "Cost optimization campaigns underutilized (3/year vs optimal 4/year)"
      ]
    },
    
    "industry_insights": {
      "common_mistakes": [
        "Insufficient discovery phase (you: doing well)",
        "No executive sponsorship (you: doing well)",
        "Big-bang deployments (you: using staged rollout ✓)",
        "No learning capture (you: tracking everything ✓)"
      ],
      "emerging_patterns": [
        "AI-human collaboration increasing (you're early adopter)",
        "Risk-tiered execution becoming standard (you're implementing)",
        "Continuous learning systems gaining traction (you're leading)"
      ]
    }
  }
}
```

### 5. Build Institutional Knowledge

**Organizational playbooks** based on experience:

```json
{
  "playbooks": {
    "security_campaign": {
      "based_on": "8 campaigns over 2 years",
      "proven_approach": {
        "discovery": "1 week - scan all repos, prioritize by CVSS score",
        "planning": "Use staged rollout: canary → early → majority → critical",
        "execution": "Auto-execute low-risk, require security team review for high-risk",
        "timeline": "4 weeks average (trending down to 3.5 weeks)",
        "budget": "$4,500 average",
        "success_factors": [
          "CISO as executive sponsor (100% success when present)",
          "Weekly updates to security team (reduces delays)",
          "Automated testing before merge (prevents regressions)"
        ],
        "pitfalls_to_avoid": [
          "Don't underestimate breaking changes in major versions",
          "Always check changelog before updating",
          "Schedule security reviews early (not bottleneck at end)"
        ]
      }
    },
    
    "cost_optimization_campaign": {
      "based_on": "3 campaigns, $180K total savings",
      "proven_approach": {
        "discovery": "2 weeks - full cloud resource audit",
        "pareto_analysis": "Focus on top 20% of resources (80% of waste)",
        "waves": "Quick wins → Low risk → Medium risk → High risk",
        "timeline": "6 weeks",
        "roi": "18x average (best performing campaign type)",
        "success_factors": [
          "CFO approval critical (changes budget planning)",
          "Pareto analysis focuses effort effectively",
          "Monthly savings tracking proves sustained value"
        ]
      }
    },
    
    "dependency_update_campaign": {
      "based_on": "6 campaigns across 200+ repos",
      "proven_approach": {
        "fastest_campaign_type": "2.1 weeks average",
        "automation_friendly": "89% success with high automation",
        "staged_rollout": "Essential - catches breaking changes early",
        "common_issue": "Major version updates - always check for breaking changes",
        "best_timing": "Q3 (summer) - fewer conflicts with other initiatives"
      }
    }
  }
}
```

### 6. Create Intelligence Report

**Issue for executives**: "📊 Monthly Campaign Intelligence Report"

**Labels**: `intelligence-report`, `executive-briefing`

**Body**:
```markdown
# Campaign Intelligence Report - [Month Year]

**Analysis Period**: [Date range]
**Campaigns Analyzed**: 23 campaigns over 2 years
**Report Generated**: [timestamp]

## 🎯 Executive Summary

Your organization has achieved **Expert-level campaign maturity**:
- ✅ 94% success rate (Expert benchmark: >90%)
- ✅ 13.2x average ROI (Expert benchmark: >10x)
- ✅ $2.4M total value delivered over 2 years
- ✅ 180% velocity improvement
- ✅ 47% cost reduction per item

**Status**: Top 20% of organizations using campaign patterns

## 📈 Trends (2-Year View)

### Performance Improving
- Velocity: 15 → 42 items/week (+180%)
- Success rate: 82% → 94% (+12 pts)
- ROI: 8.5x → 13.2x (+55%)
- Cost per item: $180 → $95 (-47%)

**Trend**: Getting faster, cheaper, and more reliable over time ✅

### AI Learning Improving
- Risk assessment accuracy: 78% → 91% (+13 pts)
- Effort estimation: ±45% → ±18% error (+60% accuracy)
- Human trust in AI recommendations: Increasing appropriately

**Trend**: AI getting smarter from each campaign ✅

## 🎯 Recommendations for Next Quarter

### 1. Run Q1 Security Campaign (HIGH CONFIDENCE)
- **Why**: Q1 is optimal timing based on 2-year history
- **Scope**: 75 vulnerabilities (based on your velocity)
- **Duration**: 3.5 weeks (improving)
- **Cost**: $3,800
- **Expected ROI**: 14x
- **Start Date**: January 15, 2026
- **Value**: $53K estimated

### 2. Increase Cost-Optimization Cadence (HIGH VALUE)
- **Why**: 18x ROI (your highest performing campaign type)
- **Current**: 3 times/year
- **Recommended**: 4 times/year (quarterly)
- **Additional Value**: $75K/year
- **Risk**: Low (proven success)

### 3. Implement Approval SLA (EFFICIENCY GAIN)
- **Issue**: Human approvals cause 1.5 week delays
- **Fix**: 48-hour SLA with auto-escalation
- **Impact**: 20% faster campaigns
- **Cost**: Minimal (process change)

## 💡 Key Insights

### What's Working
1. **Staged rollouts**: 93% success rate (vs 73% big-bang)
2. **Executive sponsorship**: 94% success (vs 71% without)
3. **Human-AI collaboration**: 91% AI accuracy with human oversight
4. **Cost optimization focus**: 18x ROI (highest value type)

### What to Improve
1. **Tech-debt campaigns**: High variance (87% success, wide range)
   - Recommendation: Better upfront scoping, add discovery phase
2. **Scope creep**: 18% of campaigns exceed scope
   - Recommendation: Lock scope after approval, defer to next campaign
3. **Underutilization**: Could run 15 campaigns/year based on velocity
   - Recommendation: Plan quarterly campaign calendar

## 🏆 Organizational Maturity: EXPERT LEVEL

**Progression**:
- 2024-Q1: Beginner (learning)
- 2024-Q3: Intermediate (consistent)
- 2025-Q2: Advanced (optimizing) 
- **2025-Q4: EXPERT** (predictable, efficient, improving) ← YOU ARE HERE

**Indicators of Expert Status**:
- ✅ Success rate >90% (yours: 94%)
- ✅ ROI >10x (yours: 13.2x)
- ✅ Velocity improvement >100% (yours: 180%)
- ✅ Cost reduction >30% (yours: 47%)
- ✅ Continuous learning system operational
- ✅ Predictive recommendations working

## 📊 Value Delivered (2-Year Total)

**Financial**:
- Total value: $2,400,000
- Total cost: $182,000
- Net value: $2,218,000
- Overall ROI: 13.2x

**Operational**:
- Security vulnerabilities fixed: 1,247
- Dependencies updated: 856
- Tech debt items resolved: 312
- Monthly cloud savings: $87K sustained

## 🚀 Next Steps

1. **Approve Q1 security campaign** (start Jan 15)
2. **Plan quarterly cost-optimization** cadence
3. **Implement approval SLA** process improvement
4. **Schedule quarterly campaign planning** sessions

**Questions?** See full analysis: `memory/intelligence/campaign-analysis-${{ github.run_id }}.json`

---
**Intelligence System Status**: Running monthly, continuously learning
**Next Report**: [next month]
**Organization Status**: Expert-level campaign maturity 🏆
```

## Output

Intelligence analysis complete:
- **Campaigns Analyzed**: 23 over 2 years
- **Intelligence Report**: #[issue-number]
- **Full Analysis**: `memory/intelligence/campaign-analysis-${{ github.run_id }}.json`
- **Recommendations Generated**: 3 high-confidence + 2 optimizations
- **Organizational Status**: Expert level
- **Next Analysis**: [next month]

**This is the compounding value**: Each campaign makes your organization smarter, faster, and more efficient.
