---
id: documentation-tasks
name: "Documentation Quality & Completeness Campaign"
description: "Track, monitor, and review all documentation-related tasks to ensure comprehensive, accurate, and up-to-date documentation across the repository"
version: v1
project-url: https://github.com/orgs/githubnext/projects/TBD
project-github-token: "${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}"
workflows:
  - documentation-tracker
tracker-label: "campaign:documentation-tasks"
memory-paths:
  - "memory/campaigns/documentation-tasks/**"
metrics-glob: "memory/campaigns/documentation-tasks/metrics/*.json"
cursor-glob: "memory/campaigns/documentation-tasks/cursor.json"
state: active
tags:
  - documentation
  - content-quality
  - developer-experience
risk-level: low
allowed-safe-outputs:
  - add-comment
  - update-project
  - create-issue
objective: "Ensure all documentation is complete, accurate, up-to-date, and follows the Diátaxis framework and Astro Starlight conventions"
kpis:
  - name: "Documentation coverage"
    priority: primary
    unit: percent
    baseline: 0
    target: 95
    time-window-days: 90
    direction: increase
    source: custom
  - name: "Broken links fixed"
    priority: supporting
    unit: count
    baseline: 0
    target: 0
    time-window-days: 30
    direction: decrease
    source: custom
  - name: "Documentation freshness"
    priority: supporting
    unit: percent
    baseline: 0
    target: 90
    time-window-days: 60
    direction: increase
    source: custom
governance:
  max-project-updates-per-run: 15
  max-comments-per-run: 15
  max-new-items-per-run: 10
  max-discovery-items-per-run: 100
  max-discovery-pages-per-run: 10
  opt-out-labels:
    - no-campaign
    - no-bot
  do-not-downgrade-done-items: true
---

# Documentation Quality & Completeness Campaign

## Overview

This campaign ensures that all documentation in the GitHub Agentic Workflows repository is comprehensive, accurate, up-to-date, and follows established conventions and frameworks.

## Objective

**Ensure all documentation is complete, accurate, up-to-date, and follows the Diátaxis framework and Astro Starlight conventions**

High-quality documentation is critical for adoption, developer experience, and project success. This campaign tracks all documentation-related work to ensure nothing falls through the cracks.

## Success Criteria

- ✅ All documentation pages follow the Diátaxis framework (Tutorial, How-to, Reference, Explanation)
- ✅ All code examples are tested and working
- ✅ All links are functional (no broken links)
- ✅ All documentation uses proper Astro Starlight syntax and components
- ✅ All documentation is updated within 30 days of related code changes
- ✅ Documentation coverage reaches 95% of features and use cases

## Key Performance Indicators

### Primary KPI: Documentation Coverage
- **Baseline**: 0% (starting point)
- **Target**: 95% (comprehensive coverage of all features)
- **Time Window**: 90 days
- **Direction**: Increase
- **Source**: Custom analysis of documentation vs. features

### Supporting KPI: Broken Links Fixed
- **Baseline**: 0 (current state)
- **Target**: 0 (no broken links)
- **Time Window**: 30 days
- **Direction**: Decrease (eliminate all broken links)
- **Source**: Link checker analysis

### Supporting KPI: Documentation Freshness
- **Baseline**: 0% (starting point)
- **Target**: 90% (most docs updated recently)
- **Time Window**: 60 days
- **Direction**: Increase
- **Source**: Git history analysis

## Documentation Categories Tracked

### 1. User Documentation (`docs/src/content/docs/`)
- **Setup guides**: Quick start, CLI, VS Code, MCP server setup
- **Reference documentation**: Commands, configuration, API reference
- **How-to guides**: Task-oriented documentation for specific use cases
- **Examples**: Sample workflows and patterns
- **Troubleshooting**: Common issues and error resolution

### 2. Developer Documentation
- **DEVGUIDE.md**: Development setup and workflows
- **CONTRIBUTING.md**: Contribution guidelines
- **AGENTS.md**: AI agent instructions
- **specs/**: Technical specifications and architecture

### 3. Skills Documentation (`skills/`)
- Domain-specific skill documentation
- Integration guides
- Best practices and patterns

### 4. In-Code Documentation
- README files throughout the repository
- Code comments and docstrings
- Workflow documentation

## Associated Workflows

### documentation-tracker
A placeholder workflow that enables campaign tracking:
- Serves as the campaign's registered workflow
- Can be enhanced in the future to automatically detect documentation gaps
- Currently used for manual tracking via the campaign label system

## Project Board

**URL**: https://github.com/orgs/githubnext/projects/TBD

The project board serves as the primary campaign dashboard, tracking:
- Documentation gaps and missing content
- Documentation updates needed
- Broken links to fix
- Style/format improvements
- Overall campaign progress

**Note**: A GitHub Project board will need to be created and linked here.

## Tracker Label

All campaign-related issues and PRs are tagged with: `campaign:documentation-tasks`

Additional labels used:
- `documentation` - General documentation label
- `docs:setup` - Setup/getting started documentation
- `docs:reference` - Reference documentation
- `docs:guide` - How-to guides and tutorials
- `docs:example` - Example workflows
- `docs:specs` - Technical specifications

## Memory Paths

Campaign state and metrics are stored in:
- `memory/campaigns/documentation-tasks/**`

Metrics snapshots: `memory/campaigns/documentation-tasks/metrics/*.json`
Cursor state: `memory/campaigns/documentation-tasks/cursor.json`

## Governance Policies

### Rate Limits (per run)
- **Max project updates**: 15
- **Max comments**: 15
- **Max new items added**: 10
- **Max discovery items scanned**: 100
- **Max discovery pages**: 10

These limits ensure gradual, sustainable progress without overwhelming the team or API rate limits.

### Opt-out Labels
Issues/PRs with these labels will not be tracked:
- `no-campaign` - Explicitly excluded from campaign tracking
- `no-bot` - No automated actions

### Protection Policies
- **Do not downgrade done items**: Once an item is marked as done, don't automatically change its status

## Risk Assessment

**Risk Level**: Low

This campaign:
- Does not modify code or production systems
- Focuses on non-critical documentation improvements
- Requires human review for all changes
- Uses incremental, reversible improvements

## Types of Documentation Tasks Tracked

### 1. Documentation Gaps
- Missing documentation for new features
- Undocumented configuration options
- Missing examples for complex use cases

### 2. Documentation Updates
- Outdated content after code changes
- Deprecated information that needs updating
- Version-specific updates

### 3. Quality Improvements
- Fixing broken links
- Improving clarity and readability
- Adding missing code examples
- Correcting technical inaccuracies

### 4. Structure & Organization
- Applying Diátaxis framework principles
- Improving navigation and discoverability
- Consolidating duplicate content
- Organizing related content

### 5. Accessibility & Usability
- Ensuring proper heading hierarchy
- Adding alt text to images
- Improving search keywords
- Enhancing mobile experience

## Campaign Lifecycle

1. **Discovery**: Automated scans identify documentation issues
2. **Prioritization**: Issues are prioritized by impact and effort
3. **Assignment**: Issues are created and added to project board
4. **Review**: Human contributors implement documentation changes
5. **Validation**: Automated checks verify quality and accuracy
6. **Tracking**: Project board reflects current status

## Orchestrator (To Be Generated)

When this campaign is compiled, an orchestrator workflow will be generated:
- **File**: `.github/workflows/documentation-tasks.campaign.g.md`
- **Schedule**: Daily or on-demand
- **Purpose**: Coordinate documentation work and update project board

The orchestrator will:
- Discover documentation-related issues via tracker label
- Add new issues to the project board
- Update issue status based on state changes
- Report campaign progress and metrics
- Track KPIs and measure progress toward goals

## How to Participate

### For Documentation Contributors

1. **Find work**: Check the project board for open documentation tasks
2. **Assign yourself**: Pick an issue that matches your skills
3. **Make changes**: Update documentation following guidelines in `skills/documentation/SKILL.md`
4. **Submit PR**: Create a pull request with your changes
5. **Label properly**: Ensure PR has `campaign:documentation-tasks` label

### For Maintainers

1. **Create issues**: When you identify documentation gaps, create issues with `campaign:documentation-tasks` label
2. **Review PRs**: Prioritize documentation PRs for review
3. **Monitor board**: Check project board for blockers or stalled work
4. **Provide feedback**: Help documentation contributors succeed

### For AI Agents

Documentation-related workflows can automatically:
- Detect documentation gaps when code changes
- Create issues for missing or outdated documentation
- Check for broken links
- Validate documentation structure
- Monitor documentation freshness

## Success Metrics

Campaign success will be measured by:
- Increase in documentation coverage percentage
- Reduction in broken links to zero
- Improved documentation freshness scores
- Positive feedback from users on documentation quality
- Reduced time to find relevant documentation

## Related Resources

- **Documentation Skill**: `skills/documentation/SKILL.md`
- **Documentation Guide**: `docs/src/content/docs/guides/`
- **Diátaxis Framework**: https://diataxis.fr/
- **Astro Starlight**: https://starlight.astro.build/

---

**Campaign Status**: Active (requires project board setup)

**Next Steps**:
1. Create GitHub Project board for documentation tracking
2. Update `project-url` field with actual project URL
3. Create `documentation-health-checker` workflow
4. Compile campaign to generate orchestrator
5. Start tracking existing documentation issues
