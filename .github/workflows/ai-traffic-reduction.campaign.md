---
id: ai-traffic-reduction
name: "Campaign: AI Traffic Reduction"
description: "Monitor and reduce GitHub AI traffic through comprehensive tracking, analysis, and optimization. Success: 20-30% reduction in daily token consumption within 90 days."
version: v1
engine: claude
project-url: "https://github.com/orgs/githubnext/projects/TBD"
workflows:
  - daily-copilot-token-report
  - metrics-collector
  - agent-performance-analyzer
  - audit-workflows
  - copilot-session-insights
  - copilot-pr-prompt-analysis
  - daily-code-metrics
  - daily-file-diet
  - duplicate-code-detector
  - ci-doctor
  - hourly-ci-cleaner
tracker-label: "campaign:ai-traffic-reduction"
memory-paths:
  - "memory/campaigns/ai-traffic-reduction/**"
  - "memory/daily-copilot-token-report/**"
  - "memory/metrics-collector/**"
  - "memory/agent-performance-analyzer/**"
  - "memory/audit-workflows/**"
  - "memory/copilot-session-insights/**"
  - "memory/copilot-pr-prompt-analysis/**"
  - "memory/daily-code-metrics/**"
  - "memory/daily-file-diet/**"
  - "memory/duplicate-code-detector/**"
  - "memory/ci-doctor/**"
  - "memory/hourly-ci-cleaner/**"
metrics-glob: "memory/campaigns/ai-traffic-reduction/metrics/*.json"
cursor-glob: "memory/campaigns/ai-traffic-reduction/cursor.json"
state: planned
tags:
  - ai-optimization
  - cost-reduction
  - performance
  - monitoring
  - code-quality
risk-level: low
allowed-safe-outputs:
  - create-issue
  - add-comment
  - update-project
objective: "Monitor and reduce GitHub AI traffic by 20-30% through comprehensive tracking, intelligent analysis, code quality improvements, and error prevention"
kpis:
  - name: "Token usage reduction"
    priority: primary
    unit: percent
    baseline: 0
    target: 25
    time-window-days: 90
    direction: decrease
    source: custom
  - name: "Workflow efficiency (success rate)"
    priority: supporting
    unit: percent
    baseline: 70
    target: 85
    time-window-days: 30
    direction: increase
    source: ci
  - name: "Error rate reduction"
    priority: supporting
    unit: percent
    baseline: 0
    target: 40
    time-window-days: 30
    direction: decrease
    source: custom
governance:
  max-issues-per-run: 5
  max-comments-per-run: 5
  max-project-updates-per-run: 20
---

# AI Traffic Reduction Campaign

## Overview

This campaign systematically monitors and reduces GitHub AI traffic across all agentic workflows through comprehensive measurement, intelligent analysis, code quality improvements, and error prevention strategies.

## Objective

**Monitor and reduce GitHub AI traffic by 20-30% through comprehensive tracking, intelligent analysis, code quality improvements, and error prevention**

By implementing a multi-layered approach combining measurement, analysis, optimization, and prevention, we aim to significantly reduce AI token consumption while maintaining or improving workflow quality and effectiveness.

## Success Criteria

- Achieve 20-30% reduction in daily token consumption within 90 days
- Measurable decrease in AI API costs per workflow execution
- Increase code health metrics and reduce technical debt
- Reduce repeated failures requiring AI intervention by 40%
- Improve workflow success rates from 70% to 85%
- Establish continuous feedback loop for ongoing optimization

## Key Performance Indicators

### Primary KPI: Token Usage Reduction
- **Baseline**: 0% (starting point)
- **Target**: 25% reduction in token consumption
- **Time Window**: 90 days
- **Direction**: Decrease
- **Source**: Custom metrics from token tracking workflows

This KPI tracks the overall success of our optimization efforts. A 25% reduction represents significant cost savings while maintaining workflow quality.

### Supporting KPI: Workflow Efficiency
- **Baseline**: 70% (current success rate)
- **Target**: 85% (improved success rate)
- **Time Window**: 30 days (rolling)
- **Direction**: Increase
- **Source**: CI metrics

This KPI ensures we're not sacrificing quality for token reduction. Higher success rates mean fewer retries and lower overall token consumption.

### Supporting KPI: Error Rate Reduction
- **Baseline**: 0% (current error rate)
- **Target**: 40% reduction in repeated errors
- **Time Window**: 30 days (rolling)
- **Direction**: Decrease
- **Source**: Custom metrics from ci-doctor and hourly-ci-cleaner

This KPI tracks how effectively we're preventing redundant AI invocations caused by repeated failures.

## Associated Workflows

### Measurement & Tracking

#### daily-copilot-token-report
**Schedule**: Daily at 9am UTC
**Purpose**: Track Copilot token consumption and costs across all agentic workflows with 30-day trend analysis

**What it does**:
- Monitors daily token usage across all workflows
- Calculates costs and identifies high-consumption patterns
- Generates trend reports with 30-day historical data
- Identifies optimization opportunities
- Provides baseline metrics for improvement tracking

#### metrics-collector
**Schedule**: Daily at various times
**Purpose**: Infrastructure agent collecting daily performance metrics for the agent ecosystem

**What it does**:
- Collects comprehensive workflow execution metrics
- Tracks performance indicators and resource usage
- Aggregates data for trend analysis
- Provides foundation data for other analysis workflows

### Analysis & Auditing

#### agent-performance-analyzer
**Schedule**: Daily
**Purpose**: Meta-orchestrator analyzing AI agent performance, quality, and effectiveness

**What it does**:
- Analyzes agent behavior patterns and efficiency
- Identifies underperforming workflows
- Recommends optimization strategies
- Tracks agent quality metrics over time

#### audit-workflows
**Schedule**: Daily
**Purpose**: Daily audit of all agentic workflow runs to identify issues and improvement opportunities

**What it does**:
- Reviews all workflow executions
- Identifies failure patterns and root causes
- Discovers inefficiencies and bottlenecks
- Creates issues for systematic improvements

#### copilot-session-insights
**Schedule**: Daily
**Purpose**: Analyzes GitHub Copilot agent sessions for behavioral patterns and inefficiencies

**What it does**:
- Examines Copilot session interactions
- Identifies common failure patterns
- Discovers inefficient prompt patterns
- Suggests session optimization strategies

#### copilot-pr-prompt-analysis
**Schedule**: Daily
**Purpose**: Analyzes prompt patterns in Copilot PR interactions

**What it does**:
- Reviews PR-related Copilot interactions
- Identifies successful prompt patterns
- Discovers ineffective prompting strategies
- Provides recommendations for better AI engagement

### Code Quality Improvement

#### daily-code-metrics
**Schedule**: Daily
**Purpose**: Tracks and visualizes daily code metrics to monitor repository health

**What it does**:
- Monitors code complexity and maintainability
- Tracks technical debt accumulation
- Identifies files needing refactoring
- Provides code quality trends

#### daily-file-diet
**Schedule**: Daily
**Purpose**: Monitors file sizes and creates refactoring issues for oversized files

**What it does**:
- Identifies files exceeding size thresholds (>800 LOC)
- Creates refactoring issues for oversized files
- Tracks file size reduction progress
- Reduces AI context requirements through better code organization

#### duplicate-code-detector
**Schedule**: Daily
**Purpose**: Identifies duplicate code patterns for consolidation

**What it does**:
- Scans codebase for duplicate patterns
- Identifies consolidation opportunities
- Creates issues for code deduplication
- Reduces codebase complexity

### Error Prevention

#### ci-doctor
**Schedule**: On CI failure
**Purpose**: Investigates failed CI workflows to identify root causes and patterns

**What it does**:
- Analyzes CI failure logs and patterns
- Identifies recurring failure causes
- Builds knowledge base of common issues
- Prevents repeated AI invocations for known problems

#### hourly-ci-cleaner
**Schedule**: Hourly (only when CI fails)
**Purpose**: Optimizes token spend by running only when CI fails

**What it does**:
- Automatically fixes common CI issues
- Applies known solutions to prevent manual intervention
- Reduces need for AI analysis on simple failures
- Maintains database of automated fixes

## Project Board Setup

**Recommended Custom Fields**:

1. **Category** (Single select): Measurement, Analysis, Code Quality, Error Prevention
   - Categorizes workflow type

2. **Worker/Workflow** (Single select): List of all 11 workflows
   - Tracks which workflow is responsible

3. **Priority** (Single select): Critical, High, Medium, Low
   - Priority based on impact on token reduction

4. **Status** (Single select): Todo, In Progress, Blocked, Done
   - Current work state

5. **Effort** (Single select): Small (1 day), Medium (2-3 days), Large (1 week)
   - Estimated effort for completion

6. **Impact on Token Usage** (Single select): High, Medium, Low
   - Expected impact on token reduction goal

7. **Start Date** (Date): When work begins

8. **End Date** (Date): Target completion date

The orchestrator automatically populates these fields based on issue content, labels, and workflow assignments.

## Timeline

- **Start Date**: 2026-01-13
- **Target Completion**: Ongoing (continuous optimization)
- **Current State**: Planned

This is a continuous optimization campaign with no end date. It runs indefinitely to maintain optimal AI usage patterns.

## Success Metrics

### Cost Optimization
- **Token consumption per workflow**: Track individual workflow efficiency
- **Cost per successful execution**: Measure cost-effectiveness
- **Total monthly AI spend**: Monitor overall budget impact
- **ROI on optimization efforts**: Calculate value of improvements

### Workflow Efficiency
- **First-time success rate**: Reduce retry requirements
- **Average execution time**: Improve workflow speed
- **Error recovery time**: Reduce time to fix issues
- **Workflow completion rate**: Increase successful completions

### Code Quality Impact
- **Technical debt reduction**: Measurable decrease in debt items
- **Code complexity trends**: Track complexity metrics
- **File size distribution**: Monitor file organization improvements
- **Test coverage increase**: Improve test coverage percentage

### Knowledge Base Growth
- **Known error patterns documented**: Build institutional knowledge
- **Automated fixes implemented**: Reduce manual intervention
- **Best practices identified**: Discover and share effective patterns
- **Prompt optimization patterns**: Improve AI interaction quality

## Memory and State Management

### Repo-Memory Structure

```
memory/
├── campaigns/
│   └── ai-traffic-reduction/
│       ├── metrics/
│       │   ├── weekly-token-usage.json     # Weekly token consumption
│       │   ├── monthly-trends.json         # Monthly trend analysis
│       │   └── optimization-impact.json    # Impact of optimizations
│       └── cursor.json                     # Campaign orchestration state
├── daily-copilot-token-report/
│   └── token-consumption.json              # Daily token tracking
├── metrics-collector/
│   └── workflow-metrics.json               # Performance data
├── agent-performance-analyzer/
│   └── agent-analysis.json                 # Agent efficiency data
├── audit-workflows/
│   └── workflow-audits.json                # Audit findings
├── copilot-session-insights/
│   └── session-patterns.json               # Session analysis
├── copilot-pr-prompt-analysis/
│   └── prompt-patterns.json                # Prompt effectiveness
├── daily-code-metrics/
│   └── code-quality.json                   # Code metrics
├── daily-file-diet/
│   └── file-sizes.json                     # File size tracking
├── duplicate-code-detector/
│   └── duplicates.json                     # Duplication findings
├── ci-doctor/
│   └── ci-failures.json                    # CI failure patterns
└── hourly-ci-cleaner/
    └── automated-fixes.json                # Applied fixes
```

### Metrics Tracking

The campaign maintains comprehensive metrics:
- Daily token consumption by workflow
- Cost trends and projections
- Workflow success/failure rates
- Error pattern frequencies
- Code quality improvements
- Optimization impact measurements

## Governance Policies

### Rate Limits (per run)
- **Max issues created**: 5
- **Max comments**: 5
- **Max project updates**: 20

These limits ensure sustainable operation and prevent overwhelming the team with too many issues.

### Quality Standards

All optimization efforts must meet these criteria:
1. **No quality regression**: Maintain or improve workflow effectiveness
2. **Measurable impact**: Track actual token reduction
3. **Documented changes**: Clear explanation of optimizations
4. **Reversible**: Ability to roll back if needed
5. **Tested**: Verify improvements don't break functionality

### Review Requirements

- High-impact optimizations require human review
- Code changes follow standard PR review process
- Prompt modifications need effectiveness validation
- Infrastructure changes require maintainer approval

## Risk Assessment

**Risk Level**: Low

This campaign:
- Monitors existing workflows (read-only analysis)
- Creates issues for review (requires approval to implement)
- Focuses on optimization (not production systems)
- Operates with rate limits and governance controls
- Maintains comprehensive audit trail
- Can be paused or adjusted without service impact

The workflows primarily analyze and report; actual optimizations require human approval.

## Orchestrator

This campaign uses an automatically generated orchestrator workflow:
- **File**: `.github/workflows/ai-traffic-reduction.campaign.g.md`
- **Schedule**: Daily at 18:00 UTC (cron: `0 18 * * *`)
- **Purpose**: Coordinate worker outputs and update project board

The orchestrator:
- Discovers worker-created issues via tracker-id
- Adds new issues to the project board
- Updates issue status and custom fields
- Aggregates metrics from all tracking workflows
- Reports campaign progress and token reduction trends
- Identifies high-impact optimization opportunities
- Coordinates between measurement, analysis, and action workflows

## Example Optimizations

Good examples of optimizations this campaign might identify:

### Prompt Optimization
- "Simplify agent instructions to reduce token consumption"
- "Remove redundant context from workflow prompts"
- "Optimize system prompts for efficiency"

### Code Quality
- "Reduce file size to minimize context requirements"
- "Consolidate duplicate code to simplify AI analysis"
- "Improve code organization to reduce AI confusion"

### Error Prevention
- "Document common CI failure patterns to prevent AI invocation"
- "Implement automated fixes for known issues"
- "Build error pattern database for quick resolution"

### Workflow Efficiency
- "Optimize workflow trigger conditions to reduce unnecessary runs"
- "Implement caching strategies to reduce repeated AI calls"
- "Consolidate related workflows to share context"

## Notes

- Workers remain campaign-agnostic and immutable
- All coordination happens in the orchestrator
- The GitHub Project board is the single source of truth
- Safe outputs include AI-generated footers for transparency
- This campaign focuses on optimization, not feature development
- Success depends on systematic measurement and continuous improvement
- Regular review of metrics ensures we're meeting reduction goals
