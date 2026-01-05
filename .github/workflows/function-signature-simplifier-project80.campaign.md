---
id: function-signature-simplifier-project80
name: "Campaign: Function Signature Simplifier (Project 80)"
description: "Systematically refactor Go functions with more than 7 parameters to use Options structs. Success: all functions ≤7 params, maintain coverage, no regressions."
version: v1
project-url: "https://github.com/orgs/githubnext/projects/80"
project-github-token: "${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}"
workflows:
  - semantic-function-refactor
tracker-label: "campaign:function-signature-simplifier-project80"
memory-paths:
  - "memory/campaigns/function-signature-simplifier-project80/**"
metrics-glob: "memory/campaigns/function-signature-simplifier-project80/metrics/*.json"
cursor-glob: "memory/campaigns/function-signature-simplifier-project80/cursor.json"
state: active
tags:
  - code-quality
  - maintainability
  - refactoring
  - go
  - function-signatures
risk-level: low
allowed-safe-outputs:
  - add-comment
  - update-project
  - create-pull-request
objective: "Refactor all Go functions with more than 7 parameters to use Options structs while maintaining test coverage and preventing regressions"
kpis:
  - name: "Functions refactored to use Options structs"
    priority: primary
    unit: percent
    baseline: 0
    target: 100
    time-window-days: 90
    direction: increase
    source: custom
  - name: "Test coverage maintained"
    priority: supporting
    unit: percent
    baseline: 80
    target: 80
    time-window-days: 7
    direction: increase
    source: ci
  - name: "No regressions introduced"
    priority: supporting
    unit: count
    baseline: 0
    target: 0
    time-window-days: 7
    direction: decrease
    source: ci
governance:
  max-project-updates-per-run: 10
  max-comments-per-run: 10
  max-new-items-per-run: 5
  max-discovery-items-per-run: 50
  max-discovery-pages-per-run: 5
---

# Function Signature Simplifier Campaign (Project 80)

## Overview

This campaign systematically identifies and refactors Go functions with more than 7 input parameters to use Options structs instead. This improves code maintainability, readability, and makes function calls more flexible and self-documenting.

## Objective

**Refactor all Go functions with more than 7 parameters to use Options structs while maintaining test coverage and preventing regressions**

Functions with many parameters (>7) are:
- Harder to understand and maintain
- Error-prone when calling (easy to mix up parameter order)
- Difficult to extend without breaking changes
- Less self-documenting than Options structs with named fields

## Success Criteria

- All Go functions have ≤7 parameters
- Test coverage remains at or above baseline (80%)
- No functionality regressions introduced
- Options structs follow Go best practices
- Code passes all existing tests and linting

## Current State

As of campaign creation, the codebase has **23 functions** with more than 7 parameters:

### Highest Priority (10+ parameters)
- `cli/logs_orchestrator.go:41` - DownloadWorkflowLogs (21 params) ⚠️
- `cli/trial_command.go:154` - RunWorkflowTrials (15 params)
- `cli/add_command.go:125` - AddWorkflows (12 params)
- `cli/add_command.go:399` - addWorkflowsNormal (12 params)
- `cli/add_command.go:453` - addWorkflowsWithPR (12 params)
- `cli/add_command.go:561` - addWorkflowWithTracking (12 params)
- `cli/logs_github_api.go:145` - listWorkflowRunsWithPagination (12 params)
- `cli/trial_repository.go:170` - installWorkflowInTrialMode (11 params)
- `cli/compile_workflow_processor.go:49` - compileWorkflowFile (11 params)
- `cli/compile_workflow_processor.go:162` - processCampaignSpec (11 params)
- `cli/audit.go:131` - AuditWorkflowRun (11 params)
- `cli/compile_orchestrator.go:164` - generateAndCompileCampaignOrchestrator (11 params)
- `cli/update_command.go:144` - UpdateWorkflowsWithExtensionCheck (11 params)
- `cli/run_command.go:25` - RunWorkflowOnGitHub (11 params)
- `cli/run_command.go:475` - RunWorkflowsOnGitHub (11 params)
- `cli/trial_command.go:542` - showTrialConfirmation (10 params)

### Medium Priority (9 parameters)
- `cli/compile_validation.go:125` - CompileWorkflowDataWithValidation (9 params)
- `cli/audit.go:405` - auditJobRun (9 params)
- `cli/init.go:15` - InitRepository (9 params)
- `cli/update_workflows.go:15` - UpdateWorkflows (9 params)

### Lower Priority (8 parameters)
- `workflow/metrics.go:452` - FinalizeToolMetrics (8 params)
- `cli/compile_validation.go:41` - CompileWorkflowWithValidation (8 params)
- `cli/update_workflows.go:259` - updateWorkflow (8 params)

## Key Performance Indicators

### Primary KPI: Functions Refactored to Use Options Structs
- **Baseline**: 0% (23 functions need refactoring)
- **Target**: 100% (all functions refactored)
- **Time Window**: 90 days
- **Direction**: Increase
- **Source**: Custom metrics from code analysis

This KPI tracks the percentage of functions that have been successfully refactored from >7 parameters to using Options structs.

### Supporting KPI: Test Coverage Maintained
- **Baseline**: 80%
- **Target**: 80% (maintain or improve)
- **Time Window**: 7 days (rolling)
- **Direction**: Increase
- **Source**: CI metrics

Ensures that refactoring doesn't reduce test coverage. All refactored code must maintain or improve test coverage.

### Supporting KPI: No Regressions Introduced
- **Baseline**: 0
- **Target**: 0 (no failing tests)
- **Time Window**: 7 days (rolling)
- **Direction**: Decrease
- **Source**: CI metrics

Tracks any test failures or regressions introduced by refactoring. Goal is zero regressions.

## Associated Workflows

### semantic-function-refactor
Primary worker workflow that:
- Identifies functions with more than 7 parameters using AST analysis
- Creates issues for refactoring tasks with priority based on parameter count
- Proposes Options struct designs following Go best practices
- Tracks progress on the project board
- Embeds tracker-id for campaign correlation

## Project Board

**URL**: https://github.com/orgs/githubnext/projects/80

The project board serves as the primary campaign dashboard, tracking:
- Open refactoring tasks (prioritized by parameter count)
- In-progress work
- Completed refactorings
- Overall campaign progress
- Test coverage and CI status

## Tracker Label

All campaign-related issues and PRs are tagged with: `campaign:function-signature-simplifier-project80`

## Memory Paths

Campaign state and metrics are stored in:
- `memory/campaigns/function-signature-simplifier-project80/**`

Metrics snapshots: `memory/campaigns/function-signature-simplifier-project80/metrics/*.json`

## Governance Policies

### Rate Limits (per run)
- **Max project updates**: 10
- **Max comments**: 10
- **Max new items added**: 5
- **Max discovery items scanned**: 50
- **Max discovery pages**: 5

These limits ensure gradual, sustainable progress without overwhelming the team or API rate limits.

### Refactoring Guidelines

All refactoring PRs must:
1. **Use idiomatic Options struct pattern**: Follow Go best practices for Options structs
2. **Maintain backward compatibility**: Consider deprecation strategies if needed
3. **Update all call sites**: Ensure all function calls use the new signature
4. **Add/update tests**: Test the new Options struct and all paths
5. **Maintain or improve test coverage**: No reduction in coverage allowed
6. **Pass all CI checks**: Lint, build, and test must pass
7. **Document the Options struct**: Add godoc comments explaining each field

### Options Struct Pattern

Refactored functions should follow this pattern:

```go
// Before (10 parameters)
func ProcessData(ctx context.Context, userID string, timeout time.Duration, 
    maxRetries int, enableCache bool, cacheDir string, validateInput bool,
    logger *log.Logger, metrics *Metrics, dryRun bool) error {
    // ...
}

// After (2 parameters + Options struct)
type ProcessDataOptions struct {
    UserID        string
    Timeout       time.Duration
    MaxRetries    int
    EnableCache   bool
    CacheDir      string
    ValidateInput bool
    Logger        *log.Logger
    Metrics       *Metrics
    DryRun        bool
}

func ProcessData(ctx context.Context, opts ProcessDataOptions) error {
    // ...
}
```

### Review Requirements

- All refactoring PRs require human review before merge
- High-priority functions (10+ params) should be reviewed by maintainers
- Breaking changes need stakeholder approval
- Significant refactorings may require design review

## Risk Assessment

**Risk Level**: Low

This campaign:
- Does not modify production logic, only function signatures
- Requires human review for all changes
- Maintains test coverage requirements
- Uses incremental, reversible refactoring approaches
- Focuses on internal APIs (most affected functions are in CLI package)

## Campaign Lifecycle

1. **Discovery**: Identify functions with >7 parameters using AST analysis
2. **Prioritization**: Order by parameter count (highest first), then by call site complexity
3. **Execution**: Create issues and PRs for refactoring tasks
4. **Review**: Human developers review and approve changes
5. **Verification**: Automated tests confirm no regressions
6. **Tracking**: Update project board with progress

## Orchestrator

This campaign uses an automatically generated orchestrator workflow:
- **File**: `.github/workflows/function-signature-simplifier-project80.campaign.g.md`
- **Schedule**: Daily at 18:00 UTC (cron: `0 18 * * *`)
- **Purpose**: Coordinate worker outputs and update project board

The orchestrator:
- Discovers worker-created issues via tracker-id
- Adds new issues to the project board
- Updates issue status based on state changes
- Reports campaign progress and metrics
- Tracks which functions have been refactored

## Agent Behavior

Agents in this campaign should:
- Prioritize functions with the most parameters (21 down to 8)
- Focus on one function or related group of functions per PR
- Follow Go best practices for Options struct design
- Ensure all tests pass before and after refactoring
- Update documentation and examples when needed
- Be conservative: maintain backward compatibility when possible

## Notes

- Worker workflows remain campaign-agnostic and immutable
- All coordination and decision-making happens in the orchestrator
- The GitHub Project board is the single source of truth for campaign state
- Safe outputs include appropriate AI-generated footers for transparency
- Most affected functions are in the `pkg/cli/` package, not the compiler
- The campaign scope includes ALL functions with >7 params, not just compiler functions
