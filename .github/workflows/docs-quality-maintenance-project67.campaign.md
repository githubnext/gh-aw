---
id: docs-quality-maintenance-project67
name: "Documentation Quality & Maintenance Campaign (Project 67)"
description: "Systematically improve documentation quality, consistency, and maintainability. Success: all docs follow Diátaxis framework, maintain accessibility standards, and pass quality checks."
version: v1
project-url: "https://github.com/orgs/githubnext/projects/67"
project-github-token: "${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}"
workflows:
  - daily-doc-updater
  - docs-noob-tester
  - daily-multi-device-docs-tester
  - unbloat-docs
  - developer-docs-consolidator
  - technical-doc-writer
tracker-label: "campaign:docs-quality-maintenance-project67"
memory-paths:
  - "memory/campaigns/docs-quality-maintenance-project67/**"
metrics-glob: "memory/campaigns/docs-quality-maintenance-project67/metrics/*.json"
cursor-glob: "memory/campaigns/docs-quality-maintenance-project67/cursor.json"
state: active
tags:
  - documentation
  - quality
  - maintainability
  - accessibility
  - user-experience
risk-level: low
allowed-safe-outputs:
  - add-comment
  - update-project
  - create-pull-request
  - create-discussion
  - upload-asset
objective: "Maintain high-quality, accessible, and consistent documentation following the Diátaxis framework while ensuring all docs are accurate, complete, and user-friendly"
kpis:
  - name: "Documentation coverage of features"
    priority: primary
    unit: percent
    baseline: 85
    target: 95
    time-window-days: 90
    direction: increase
    source: custom
  - name: "Documentation accessibility score"
    priority: supporting
    unit: percent
    baseline: 90
    target: 98
    time-window-days: 30
    direction: increase
    source: custom
  - name: "User-reported documentation issues"
    priority: supporting
    unit: count
    baseline: 15
    target: 5
    time-window-days: 30
    direction: decrease
    source: pull_requests
governance:
  max-project-updates-per-run: 15
  max-comments-per-run: 10
  max-new-items-per-run: 8
  max-discovery-items-per-run: 100
  max-discovery-pages-per-run: 10
---

# Documentation Quality & Maintenance Campaign (Project 67)

## Overview

This campaign ensures the GitHub Agentic Workflows documentation maintains the highest quality standards, remains accurate and up-to-date, follows the Diátaxis framework, and provides an excellent user experience across all devices and accessibility requirements.

## Objective

**Maintain high-quality, accessible, and consistent documentation following the Diátaxis framework while ensuring all docs are accurate, complete, and user-friendly**

High-quality documentation is critical for user adoption and success. This campaign coordinates multiple documentation workflows to systematically improve and maintain documentation excellence.

## Success Criteria

- ✅ All documentation follows the Diátaxis framework (Tutorial, How-to, Reference, Explanation)
- ✅ Documentation coverage reaches 95% of user-facing features
- ✅ Accessibility score maintains 98% or higher
- ✅ User-reported documentation issues decrease to ≤5 per month
- ✅ All documentation passes automated quality checks
- ✅ Documentation site performs well across mobile, tablet, and desktop devices

## Key Performance Indicators

### Primary KPI: Documentation Coverage of Features
- **Baseline**: 85% (current estimated coverage)
- **Target**: 95% (comprehensive feature documentation)
- **Time Window**: 90 days
- **Direction**: Increase
- **Source**: Custom metrics from feature analysis

This KPI tracks what percentage of user-facing features have complete documentation. Coverage includes CLI commands, workflow configurations, safe outputs, tools, and all major features.

### Supporting KPI: Documentation Accessibility Score
- **Baseline**: 90% (current accessibility compliance)
- **Target**: 98% (near-perfect accessibility)
- **Time Window**: 30 days (rolling)
- **Direction**: Increase
- **Source**: Custom metrics from Playwright accessibility testing

This KPI measures WCAG 2.1 AA compliance across the documentation site, including keyboard navigation, screen reader support, color contrast, and semantic HTML.

### Supporting KPI: User-Reported Documentation Issues
- **Baseline**: 15 per month (current average)
- **Target**: 5 per month (minimal confusion)
- **Time Window**: 30 days (rolling)
- **Direction**: Decrease
- **Source**: Pull requests and issues labeled "documentation"

Lower user-reported issues indicate clearer, more complete documentation that addresses user needs effectively.

## Associated Workflows

This campaign coordinates six complementary documentation workflows:

### daily-doc-updater
**Purpose**: Automatically reviews and updates documentation based on recent code changes
- Scans merged PRs and commits from last 24 hours
- Identifies features requiring documentation updates
- Creates PRs with documentation additions/updates
- Ensures documentation stays current with codebase

**Cadence**: Daily at 6am UTC

### docs-noob-tester
**Purpose**: Tests documentation from a beginner's perspective
- Navigates documentation as a new user would
- Follows getting started guides step-by-step
- Identifies confusing or broken instructions
- Takes screenshots and reports usability issues
- Creates discussions with findings

**Cadence**: Daily (scattered execution time)

### daily-multi-device-docs-tester
**Purpose**: Tests documentation site across multiple device form factors
- Tests responsive design on mobile, tablet, desktop
- Verifies functionality across different screen sizes
- Identifies layout or navigation issues
- Takes device-specific screenshots
- Creates issues for device-specific problems

**Cadence**: Daily (scattered execution time)

### unbloat-docs
**Purpose**: Reviews and simplifies documentation by reducing verbosity
- Identifies overly verbose documentation sections
- Simplifies language while maintaining clarity
- Removes redundant explanations
- Improves readability and scannability
- Creates PRs with simplified content

**Cadence**: Daily (scattered execution time)
**Trigger**: Also available via `/unbloat` command in PR comments

### developer-docs-consolidator
**Purpose**: Consolidates developer documentation from multiple sources
- Reviews markdown files in `specs/` directory
- Ensures consistent technical tone and formatting
- Produces consolidated `developer.instructions.md`
- Uses Serena MCP for static analysis
- Creates PRs with consolidated documentation

**Cadence**: Daily at 3:17 AM UTC

### technical-doc-writer
**Purpose**: Creates or enhances technical documentation for complex features
- Identifies features lacking technical documentation
- Generates comprehensive technical explanations
- Follows documentation guidelines and framework
- Creates PRs with new technical content

**Cadence**: On-demand or scheduled

## Project Board

**URL**: https://github.com/orgs/githubnext/projects/67

The project board serves as the campaign dashboard, tracking:
- Documentation gaps and coverage
- Quality improvement tasks
- Accessibility issues
- User-reported problems
- PRs in review
- Completed improvements
- Overall campaign progress

## Tracker Label

All campaign-related issues and PRs are tagged with: `campaign:docs-quality-maintenance-project67`

## Memory Paths

Campaign state and metrics are stored in:
- `memory/campaigns/docs-quality-maintenance-project67/**`

Metrics snapshots: `memory/campaigns/docs-quality-maintenance-project67/metrics/*.json`

## Governance Policies

### Rate Limits (per run)
- **Max project updates**: 15
- **Max comments**: 10
- **Max new items added**: 8
- **Max discovery items scanned**: 100
- **Max discovery pages**: 10

These limits ensure sustainable progress while preventing API rate limit exhaustion and maintaining manageable workload for reviewers.

### Quality Standards

All documentation changes must:
1. **Follow Diátaxis framework**: Clearly categorize content as Tutorial, How-to, Reference, or Explanation
2. **Maintain accessibility**: Pass WCAG 2.1 AA standards
3. **Use proper formatting**: Follow Astro Starlight markdown conventions
4. **Include examples**: Provide practical code samples where appropriate
5. **Be technically accurate**: Match current codebase behavior
6. **Maintain consistent tone**: Neutral, technical, not promotional

### Review Requirements

- All documentation PRs require human review before merge
- Accessibility issues require immediate attention
- Breaking documentation changes need stakeholder approval
- Major restructuring requires discussion before implementation

## Risk Assessment

**Risk Level**: Low

This campaign:
- Does not modify production code
- Requires human review for all changes
- Focuses on documentation improvements only
- Uses proven workflows with established patterns
- Has rollback capability through git history
- Monitors quality through automated testing

**Potential Risks**:
- Overly aggressive simplification losing important details (mitigated by human review)
- Documentation drift from code changes (mitigated by daily-doc-updater)
- Accessibility regressions (mitigated by continuous testing)
- Resource consumption from multiple workflows (mitigated by governance rate limits)

## Campaign Lifecycle

### Phase 1: Discovery
- Identify documentation gaps through feature analysis
- Scan for outdated or inaccurate content
- Detect accessibility issues through automated testing
- Collect user feedback and reported issues

### Phase 2: Prioritization
- Rank documentation needs by user impact
- Prioritize accessibility issues (highest priority)
- Address feature documentation gaps
- Schedule quality improvements

### Phase 3: Execution
- Workflows create PRs with improvements
- Human reviewers validate changes
- Merge approved documentation updates
- Update project board with progress

### Phase 4: Verification
- Automated tests confirm accessibility standards
- Manual testing validates improvements
- User feedback collection on updated docs
- Metrics analysis shows quality trends

### Phase 5: Continuous Monitoring
- Daily workflows maintain documentation currency
- Accessibility testing runs continuously
- User satisfaction tracked through issues
- Project board reflects real-time status

## Orchestrator

This campaign uses an automatically generated orchestrator workflow:
- **File**: `.github/workflows/docs-quality-maintenance-project67.campaign.g.md`
- **Schedule**: Daily at 18:00 UTC (cron: `0 18 * * *`)
- **Purpose**: Coordinate worker outputs and update project board

The orchestrator:
- Discovers worker-created issues via tracker-id
- Adds new issues to the project board
- Updates issue status based on state changes
- Aggregates metrics from all documentation workflows
- Reports campaign progress and quality trends
- Identifies coordination opportunities between workflows

## Coordination Strategy

### Workflow Coordination

The campaign orchestrator ensures workflows complement rather than conflict:

1. **Sequential Execution**: Workflows run at different times to avoid conflicts
2. **Shared Memory**: Use campaign memory paths for cross-workflow coordination
3. **Unified Tracking**: All workflows use campaign tracker label
4. **Consolidated Metrics**: Orchestrator aggregates metrics from all workflows

### Avoiding Redundancy

- Each workflow has distinct responsibilities (no overlap)
- Orchestrator detects duplicate issues before creation
- Workflows check recent activity before taking action
- Rate limits prevent overwhelming reviewers

### Cross-Workflow Intelligence

- `daily-doc-updater` outputs inform `unbloat-docs` about new content
- `docs-noob-tester` findings guide `technical-doc-writer` priorities
- `daily-multi-device-docs-tester` accessibility findings trigger fixes
- `developer-docs-consolidator` maintains consistency across all docs

## Success Metrics Dashboard

The orchestrator generates daily reports showing:

```markdown
## Documentation Quality Dashboard - [DATE]

### Coverage Metrics
- Feature documentation coverage: XX% (target: 95%)
- API reference completeness: XX% (target: 100%)
- Example coverage: XX examples (target: 50+)

### Quality Metrics
- Accessibility score: XX% (target: 98%)
- Average page load time: XXXXms (target: <1000ms)
- Broken links detected: XX (target: 0)

### User Satisfaction
- Documentation issues opened: XX (target: <5/month)
- Documentation PRs merged: XX (target: 20/90 days)
- Positive feedback count: XX

### Workflow Activity (Last 7 Days)
- daily-doc-updater: XX runs, XX PRs created
- docs-noob-tester: XX runs, XX issues found
- daily-multi-device-docs-tester: XX devices tested, XX issues found
- unbloat-docs: XX PRs created, XX words removed
- developer-docs-consolidator: XX files consolidated
- technical-doc-writer: XX new docs created

### Campaign Health
- On track to meet 90-day objectives: ✅/⚠️/❌
- Quality trend: 📈 Improving / 📊 Stable / 📉 Declining
- Resource utilization: XX% of rate limits
```

## Documentation Framework Compliance

All documentation must follow the **Diátaxis framework**:

### Tutorials (Learning-Oriented)
- Guide beginners through achieving specific outcomes
- Focus on learning by doing
- Example: "Your First Agentic Workflow"

### How-to Guides (Goal-Oriented)
- Solve specific real-world problems
- Assume some existing knowledge
- Example: "How to Configure MCP Servers"

### Reference (Information-Oriented)
- Provide accurate technical descriptions
- Comprehensive API/CLI documentation
- Example: "CLI Command Reference"

### Explanation (Understanding-Oriented)
- Clarify and illuminate concepts
- Provide context and background
- Example: "Understanding the Diátaxis Framework"

## Integration with Other Campaigns

This campaign may interact with:
- **Go File Size Reduction (Project 64)**: Developer docs may need updates when code is refactored
- **Security Campaigns**: Security documentation updates require coordination
- **Feature Campaigns**: New features require documentation coverage

The orchestrator coordinates with other campaign orchestrators through shared memory and cross-campaign issue tracking.

## Notes

- Workers remain campaign-agnostic and immutable
- All coordination and decision-making happens in the orchestrator
- The GitHub Project board is the single source of truth for campaign state
- Safe outputs include appropriate AI-generated footers for transparency
- Documentation changes preserve git history for easy rollback
- Human reviewers maintain final authority on all documentation changes

## Getting Started

To activate this campaign:

1. **Create Project Board**: Set up GitHub Project at the specified URL
2. **Configure Secrets**: Ensure `GH_AW_PROJECT_GITHUB_TOKEN` is available
3. **Enable Workflows**: Ensure all associated workflows are active
4. **Review Orchestrator**: Validate generated orchestrator workflow
5. **Monitor Dashboard**: Check project board and daily reports
6. **Provide Feedback**: Comment on issues and PRs to guide improvements

## Campaign Owner

**Owner**: Documentation Team
**Reviewers**: All team members contributing to documentation
**Escalation**: Create issue with `campaign:docs-quality-maintenance-project67` label

---

> 🤖 *This campaign specification was generated as part of the GitHub Agentic Workflows campaign system. For questions or concerns, please create an issue with the campaign tracker label.*
