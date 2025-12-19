---
id: documentation-improvement
version: "v1"
name: "Documentation Improvement Campaign"
description: "Coordinate and track all documentation updates, improvements, and maintenance activities across agentic workflows to ensure comprehensive, accurate, and accessible documentation."

project-url: "https://github.com/orgs/githubnext/projects/65"
project-github-token: "${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}"

workflows:
  - daily-doc-updater
  - unbloat-docs
  - daily-multi-device-docs-tester
  - developer-docs-consolidator
  - docs-noob-tester
  - glossary-maintainer
  - technical-doc-writer

memory-paths:
  - "memory/campaigns/documentation-improvement-*/**"

owners:
  - "developer-experience"
  - "documentation-team"

executive-sponsors:
  - "vp-product"

risk-level: "low"
state: "active"
tags:
  - "documentation"
  - "developer-experience"
  - "content-quality"
  - "accessibility"

tracker-label: "campaign:documentation-improvement"

metrics-glob: "memory/campaigns/documentation-improvement-*/metrics/*.json"

allowed-safe-outputs:
  - "create-issue"
  - "create-pull-request"
  - "add-comment"
  - "upload-assets"
  - "update-project"
  - "create-discussion"

approval-policy:
  required-approvals: 1
  required-roles:
    - "documentation-lead"
  change-control: false
---

# Documentation Improvement Campaign

This campaign coordinates and tracks all documentation-related improvements across multiple agentic workflows to ensure GitHub Agentic Workflows documentation remains comprehensive, accurate, accessible, and up-to-date.

## Campaign Objectives

- **Goal**: Maintain high-quality documentation through automated monitoring, testing, and improvement
- **Scope**: All documentation in the `docs/` directory, including tutorials, how-to guides, reference docs, and explanations
- **Success Criteria**:
  - Documentation stays current with code changes (automated daily updates)
  - Documentation is accessible and tested across devices
  - Documentation follows Diátaxis framework and style guidelines
  - Documentation bloat is reduced and clarity improved
  - Glossary remains comprehensive and accurate

## Tracking

- **Issues**: Labeled with `campaign:documentation-improvement`
- **Project Board**: [GitHub Project #65](https://github.com/orgs/githubnext/projects/65)
- **Metrics**: Daily snapshots stored under `memory/campaigns/documentation-improvement-*/metrics/`

## Documentation Workflows

This campaign integrates seven specialized documentation workflows:

### 1. Daily Documentation Updater (`daily-doc-updater`)
- **Purpose**: Automatically reviews and updates documentation based on merged PRs and code changes
- **Schedule**: Daily at 6am UTC
- **Output**: Pull requests with documentation updates
- **Tracker ID**: `daily-doc-updater`

### 2. Documentation Unbloat (`unbloat-docs`)
- **Purpose**: Reviews and simplifies documentation by reducing verbosity while maintaining clarity
- **Schedule**: Daily (scattered execution)
- **Output**: Pull requests with streamlined documentation
- **Tracker ID**: `unbloat-docs`

### 3. Multi-Device Documentation Tester (`daily-multi-device-docs-tester`)
- **Purpose**: Tests documentation site functionality and responsive design across device types
- **Schedule**: Daily
- **Output**: Issues reporting layout, accessibility, or rendering problems
- **Tracker ID**: `daily-multi-device-docs-tester`

### 4. Developer Documentation Consolidator (`developer-docs-consolidator`)
- **Purpose**: Consolidates and organizes developer documentation from multiple sources
- **Schedule**: Daily at 3:17 AM UTC
- **Output**: Pull requests or discussions with consolidated documentation
- **Tracker ID**: `developer-docs-consolidator`

### 5. Documentation Noob Tester (`docs-noob-tester`)
- **Purpose**: Tests documentation as a new user would, identifying confusing or broken steps
- **Schedule**: Daily
- **Output**: Discussions reporting usability issues
- **Tracker ID**: `docs-noob-tester`

### 6. Glossary Maintainer (`glossary-maintainer`)
- **Purpose**: Maintains and updates the documentation glossary based on codebase changes
- **Schedule**: Weekdays at 10am UTC
- **Output**: Pull requests updating the glossary
- **Tracker ID**: `glossary-maintainer`

### 7. Technical Documentation Writer (`technical-doc-writer`)
- **Purpose**: Reviews and improves technical documentation based on provided topics
- **Trigger**: Manual workflow_dispatch
- **Output**: Pull requests with improved documentation
- **Tracker ID**: `technical-doc-writer`

## Approach

1. **Monitor**: Automated workflows continuously monitor documentation quality
2. **Test**: Regular testing across devices, user scenarios, and accessibility standards
3. **Update**: Automated updates based on code changes and merged PRs
4. **Improve**: Continuous refinement of documentation clarity and structure
5. **Track**: All work items tracked in GitHub Project with consistent labeling

## Workflow Integration

### Worker Pattern
All documentation workflows operate as independent workers:
- Each workflow includes its `tracker-id` in frontmatter
- Workers create issues, PRs, or discussions automatically
- All outputs include tracker-id markers in XML comments
- Workers are campaign-agnostic and reusable

### Orchestrator Pattern
A generated orchestrator workflow (if configured):
- Discovers work by searching for tracker-id markers
- Adds discovered items to the campaign project board
- Updates project board status as work progresses
- Generates periodic status reports and metrics

### Discovery Query Pattern
To find all documentation work created by these workflows:
```
repo:githubnext/gh-aw "tracker-id: daily-doc-updater" OR "tracker-id: unbloat-docs" OR "tracker-id: daily-multi-device-docs-tester" OR "tracker-id: developer-docs-consolidator" OR "tracker-id: docs-noob-tester" OR "tracker-id: glossary-maintainer" OR "tracker-id: technical-doc-writer" in:body
```

## Setup

**One-time manual setup**: 
1. Create the GitHub Project (#65) in the UI
2. Configure views (board/table, grouping by status, filtered by campaign label)
3. Set up custom fields if needed (Status, Priority, Type)
4. The orchestrator workflow will discover and add items automatically

## Metrics and Reporting

Daily metrics capture:
- Number of documentation PRs created
- Number of issues identified (accessibility, clarity, accuracy)
- Documentation test results (device compatibility, usability)
- Glossary coverage (terms added/updated)
- Documentation bloat reduction (lines/words removed)

Metrics are stored as JSON snapshots in `memory/campaigns/documentation-improvement-*/metrics/` for trend analysis and reporting.

## Governance

- **Risk Level**: Low - documentation changes are low-risk and easily reversible
- **Approvals**: Requires 1 approval from documentation-lead
- **Change Control**: Not required for routine documentation updates
- **Review Process**: All PRs created by workflows use safe-outputs with appropriate labels

## Expected Outcomes

- **Consistency**: All documentation work tracked in one place
- **Visibility**: Clear view of documentation health and improvement activities
- **Coordination**: Workflows operate independently but tracked centrally
- **Metrics**: Historical data on documentation quality trends
- **Accountability**: Clear ownership and approval processes

Use this specification as the authoritative description of the campaign for owners, sponsors, and reporting purposes.
