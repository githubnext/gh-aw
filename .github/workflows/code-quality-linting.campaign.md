---
id: code-quality-linting
name: "Campaign: Code Quality & Linting"
description: "Systematically improve code quality through comprehensive linting, automated formatting, and code simplification. Success: 95%+ linting pass rate, 98% formatting consistency, and 50+ issues resolved in 90 days."
version: v1
engine: copilot
project-url: "https://github.com/orgs/githubnext/projects/TBD"  # To be updated when project is created
workflows:
  - super-linter
  - tidy
  - code-simplifier
tracker-label: "campaign:code-quality-linting"
memory-paths:
  - "memory/campaigns/code-quality-linting/**"
  - "memory/super-linter/**"
  - "memory/tidy/**"
  - "memory/code-simplifier/**"
metrics-glob: "memory/campaigns/code-quality-linting/metrics/*.json"
cursor-glob: "memory/campaigns/code-quality-linting/cursor.json"
state: planned
tags:
  - code-quality
  - linting
  - formatting
  - automation
  - maintenance
  - technical-debt
risk-level: low
allowed-safe-outputs:
  - create-issue
  - create-pull-request
  - add-comment
  - push-to-pull-request-branch
  - update-project
objective: "Achieve 95%+ linting pass rate across the entire codebase through systematic detection, automated fixing, and code simplification while maintaining 98% formatting consistency"
kpis:
  - name: "Linting pass rate"
    priority: primary
    unit: percent
    baseline: 85
    target: 95
    time-window-days: 90
    direction: increase
    source: custom
  - name: "Code formatting consistency"
    priority: supporting
    unit: percent
    baseline: 90
    target: 98
    time-window-days: 30
    direction: increase
    source: custom
  - name: "Linting issues resolved"
    priority: supporting
    unit: count
    baseline: 0
    target: 50
    time-window-days: 90
    direction: increase
    source: pull_requests
governance:
  max-issues-per-run: 3
  max-pull-requests-per-run: 2
  max-comments-per-run: 5
  max-project-updates-per-run: 15
---

# Code Quality & Linting Campaign

## Overview

This campaign systematically improves code quality across the GitHub Agentic Workflows codebase through comprehensive linting, automated formatting, and intelligent code simplification. By coordinating three specialized workflows, the campaign ensures consistent code quality standards while reducing technical debt.

## Objective

**Achieve 95%+ linting pass rate across the entire codebase through systematic detection, automated fixing, and code simplification while maintaining 98% formatting consistency**

High code quality is essential for maintainability, reliability, and developer productivity. This campaign coordinates detection, remediation, and improvement workflows to systematically elevate code quality standards.

## Success Criteria

- Achieve 95%+ linting pass rate across entire codebase (baseline: 85%)
- Maintain 98% code formatting consistency (baseline: 90%)
- Resolve 50+ linting issues within 90 days
- Achieve zero critical linting errors in production code
- Apply automated formatting fixes within 24 hours of detection
- Reduce technical debt through systematic code simplification
- All changes pass existing tests and CI checks
- Maintain or improve code coverage during improvements

## Key Performance Indicators

### Primary KPI: Linting Pass Rate
- **Baseline**: 85% (current pass rate)
- **Target**: 95% (comprehensive quality)
- **Time Window**: 90 days
- **Direction**: Increase
- **Source**: Custom metrics from super-linter runs

This KPI tracks the percentage of code that passes linting checks. Higher pass rates indicate better code quality, consistency, and adherence to standards.

### Supporting KPI: Code Formatting Consistency
- **Baseline**: 90% (current formatting compliance)
- **Target**: 98% (near-perfect consistency)
- **Time Window**: 30 days (rolling)
- **Direction**: Increase
- **Source**: Custom metrics from tidy workflow runs

This KPI measures formatting consistency across Go, JavaScript, TypeScript, and other code files. Higher consistency reduces cognitive load and improves code readability.

### Supporting KPI: Linting Issues Resolved
- **Baseline**: 0 (campaign start)
- **Target**: 50+ issues resolved
- **Time Window**: 90 days
- **Direction**: Increase
- **Source**: Pull requests closing linting-related issues

Tracks the number of linting issues successfully resolved. Higher numbers indicate effective remediation and continuous improvement.

## Associated Workflows

### super-linter
Runs comprehensive Markdown quality checks using Super Linter. Creates issues for violations with severity categorization.

**Schedule**: Weekdays at 2 PM UTC (cron: `0 14 * * 1-5`)

**What it does**:
- Runs Super Linter on markdown files
- Categorizes violations by severity (error, warning, info)
- Creates GitHub issues for violations (max 3 per run)
- Uploads linter logs as artifacts
- Provides detailed analysis and recommendations

**Focus areas**:
- Markdown syntax and style
- Link validity
- Documentation quality
- Formatting consistency

### tidy
Automatically formats and tidies code files (Go, JS, TypeScript). Creates or updates pull requests with formatting fixes.

**Schedule**: Daily at 7 AM UTC (cron: `0 7 * * *`)

**Triggers**: Also runs on push to main for code files, manual workflow_dispatch, and slash commands in PRs

**What it does**:
- Runs `make fmt` to format all code
- Runs `make lint` to check for linting issues
- Automatically fixes clear, obvious issues
- Recompiles workflows with `make recompile`
- Runs tests to verify changes don't break functionality
- Creates or updates a single tidy PR with all fixes
- Pushes changes to existing tidy PR branch if one exists

**Focus areas**:
- Go code formatting (gofmt, goimports)
- JavaScript/TypeScript formatting
- Linting fixes (unused variables, imports)
- Workflow compilation

### code-simplifier
Analyzes code and creates PRs with simplifications to improve clarity and maintainability.

**Schedule**: Daily

**Skip condition**: Skips if there's already an open PR titled with "[code-simplifier]"

**What it does**:
- Analyzes recently modified code files
- Identifies opportunities for simplification
- Suggests refactorings that improve clarity
- Creates pull requests with proposed changes (expires after 7 days)
- Preserves functionality while improving maintainability
- Focuses on readability and consistency

**Focus areas**:
- Code complexity reduction
- Naming improvements
- Structure optimization
- Pattern consistency
- Error handling improvements

## Regular CI Workflows (Enhanced by Campaign)

These existing workflows benefit from the campaign's proactive linting improvements:

- **`ci.yml`**: Continuous Integration - runs golangci-lint, tests, and validation
- **`format-and-commit.yml`**: Manual formatting and linting workflow
- **`security-scan.yml`**: Security vulnerability scanning

## Tools and Make Targets

- `make lint` - Run all linters (golangci-lint, eslint, markdownlint)
- `make golint` - Run golangci-lint specifically
- `make fmt` - Format all code (Go, JS, JSON)
- `make fmt-check` - Verify formatting without changes
- `make recompile` - Recompile all workflow files
- `make test` - Run all tests
- `make agent-finish` - Complete validation (build, test, recompile, fmt, lint)

## Project Board Setup

**Recommended Custom Fields**:

1. **Issue Type** (Single select): Linting Error, Formatting Issue, Code Simplification, Technical Debt
   - Categorizes the type of quality improvement

2. **Severity** (Single select): Critical, High, Medium, Low, Info
   - Priority based on impact and urgency

3. **Source Workflow** (Single select): super-linter, tidy, code-simplifier
   - Tracks which workflow detected the issue

4. **Status** (Single select): Todo, In Progress, Fixed, Verified, Closed
   - Current work state

5. **Effort** (Single select): Small (< 1 hour), Medium (1-4 hours), Large (1+ day)
   - Estimated effort for resolution

6. **Files Affected** (Text): List of files or patterns
   - Scope of the issue

The orchestrator automatically populates these fields based on issue labels, content, and tracking metadata.

## Agent Behavior Guidelines

Agents working on tasks from this campaign should:

### Task Selection
- Prioritize critical and high-severity linting errors first
- Choose formatting consistency issues for quick wins
- Tackle code simplification after basic issues are resolved
- Consider dependencies between related issues
- Start with well-defined, isolated issues

### Implementation Standards
- Run `make fmt` before and after changes
- Run `make lint` to verify issues are resolved
- Run `make recompile` to update workflow lock files
- Run `make test` to ensure no regressions
- Follow existing code style and conventions
- Preserve functionality - only improve form, not function
- Add tests if fixing bugs discovered during linting

### Pull Request Quality
- Clear title describing the fix (use workflow prefix)
- Reference the original issue created by workflow
- Explain what was fixed and why
- Show before/after examples for clarity improvements
- Include linter output showing resolution
- Link to relevant documentation or standards

### Communication
- Comment on issues when starting work
- Update project board status
- Link PRs to issues for tracking
- Report blockers or questions promptly
- Close issues when PRs are merged

## Timeline

- **Start Date**: TBD (upon campaign approval and project board setup)
- **Target Completion**: 90 days from start
- **Current State**: Planned

### Milestones

1. **Week 1-2**: Initial scan and baseline measurement
   - Run super-linter across entire codebase
   - Establish baseline metrics
   - Categorize issues by severity and type

2. **Week 3-6**: High-priority fixes (critical/high severity)
   - Address critical linting errors
   - Fix high-severity formatting issues
   - Apply automated fixes via tidy workflow

3. **Week 7-10**: Medium-priority improvements and simplifications
   - Resolve medium-severity linting issues
   - Apply code simplifications
   - Improve consistency across modules

4. **Week 11-12**: Final improvements and validation
   - Address remaining low-severity issues
   - Verify all KPI targets met
   - Document improvements and lessons learned

## Campaign Strategy

This campaign coordinates three complementary workflows:

1. **Detection** (`super-linter`): Discovers linting issues and creates tracked issues with severity categorization
2. **Auto-Fix** (`tidy`): Automatically fixes formatting and clear linting issues with consolidated PRs
3. **Improvement** (`code-simplifier`): Suggests code simplifications for maintainability and clarity

All workflows update a central GitHub Project board to track progress toward KPI targets. The orchestrator coordinates the workflows, prevents duplication, and ensures steady progress without overwhelming reviewers.

## Memory and State Management

### Repo-Memory Structure

```
memory/
├── campaigns/
│   └── code-quality-linting/
│       ├── metrics/
│       │   └── daily-stats.json         # Daily KPI metrics
│       └── cursor.json                   # Campaign orchestration state
├── super-linter/
│   ├── recent-issues.json                # Issues created recently
│   └── severity-counts.json              # Severity distribution
├── tidy/
│   ├── formatting-stats.json             # Formatting metrics
│   └── pr-branch.txt                     # Current tidy PR branch
└── code-simplifier/
    ├── analyzed-files.json               # Files analyzed
    └── improvement-suggestions.json      # Pending suggestions
```

### Metrics Tracking

The orchestrator maintains detailed metrics:
- Linting pass rate over time
- Issues created per severity level
- Issues resolved per week
- Formatting consistency percentage
- Code simplification acceptance rate
- Average time to resolution
- PR merge rate and review time

## Governance Policies

### Rate Limits (per orchestrator run)
- **Max issues created**: 3
- **Max pull requests created**: 2
- **Max comments**: 5
- **Max project updates**: 15

These limits ensure sustainable operation and prevent overwhelming maintainers with too many changes at once.

### Quality Standards

All fixes and improvements must:
1. **Preserve functionality**: No behavioral changes unless fixing bugs
2. **Pass tests**: All existing tests must pass
3. **Follow conventions**: Match existing code style
4. **Be well-scoped**: Changes limited to specific issues
5. **Be reviewable**: Clear, focused changes
6. **Include verification**: Show linter output or test results

### Deduplication Policy

To prevent duplicate work:
- Track processed files in repo-memory
- Check for existing issues and PRs before creation
- Consolidate tidy fixes into single PR
- Use project board to coordinate work
- Review recently closed issues

### Review Requirements

- Automated formatting fixes can be auto-merged if tests pass
- Code simplifications require human review
- Critical fixes need prompt attention
- Large refactorings need tech lead approval
- Changes affecting public APIs need extra scrutiny

## Risk Assessment

**Risk Level**: Low

This campaign:
- Only improves code quality (no functional changes)
- All changes require PR review and CI checks
- Operates with rate limits and governance controls
- Focuses on maintainability, not production systems
- Maintains audit trail in repo-memory
- Cannot bypass required reviews or CI checks

The workflows create issues and PRs that must be reviewed and approved before merging. No direct code changes are made without oversight.

## Orchestrator

This campaign uses an automatically generated orchestrator workflow:
- **File**: `.github/workflows/code-quality-linting.campaign.g.md`
- **Schedule**: Daily at 18:00 UTC (cron: `0 18 * * *`)
- **Purpose**: Coordinate worker outputs and update project board

The orchestrator:
- Discovers issues and PRs created by worker workflows
- Adds new items to the project board
- Updates issue status and custom fields
- Aggregates metrics from all worker runs
- Reports campaign progress and quality trends
- Identifies high-priority issues for immediate attention
- Prevents duplicate work across workflows
- Enforces rate limits and governance policies

## Example Issues and Improvements

### Linting Issues (super-linter)
- "Markdown header hierarchy violation in docs/guides/setup.md"
- "Missing blank line before code block in README.md"
- "Inconsistent link format in .github/workflows/README.md"

### Formatting Fixes (tidy)
- "Format Go files in pkg/workflow package"
- "Fix JavaScript formatting in actions/setup/js/"
- "Remove unused imports in pkg/cli/"

### Code Simplifications (code-simplifier)
- "Simplify error handling in pkg/workflow/compiler.go"
- "Extract duplicate validation logic into shared function"
- "Reduce cyclomatic complexity in pkg/parser/frontmatter.go"

## Success Metrics

### Detection Effectiveness
- **Issues per scan**: How many issues identified per super-linter run
- **Coverage**: Percentage of codebase scanned
- **False positive rate**: Percentage of issues that are invalid

### Fix Quality
- **Auto-fix success rate**: Percentage of tidy PRs merged successfully
- **Time to resolution**: Average time from issue creation to closure
- **Test pass rate**: Percentage of fixes that pass CI on first try

### Impact Metrics
- **Code quality improvement**: Measurable reduction in linting violations
- **Formatting consistency**: Percentage of files meeting formatting standards
- **Technical debt reduction**: Number of legacy issues addressed
- **Developer satisfaction**: Feedback on code quality improvements

## Notes

- Workers remain campaign-agnostic and immutable
- All coordination happens in the orchestrator
- The GitHub Project board is the single source of truth
- Safe outputs include AI-generated footers for transparency
- This campaign complements (not replaces) regular CI checks
- Focus is on quality improvements, not feature changes
- Automated fixes prioritize safety over perfection
