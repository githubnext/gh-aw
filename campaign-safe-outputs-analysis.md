# Campaign Safe Outputs Analysis: PR #11855

## Executive Summary

PR #11855 ("chore: make orchestrator dispatch only") fundamentally changed the campaign orchestrator model from allowing direct safe outputs to a **dispatch-only** model. This analysis compares which safe outputs were allowed in the old vs. new model.

### Quick Comparison

| Aspect | Old Model (Before PR #11855) | New Model (After PR #11855) |
|--------|------------------------------|----------------------------|
| **Safe Outputs Allowed** | 5 types | 1 type |
| **Direct GitHub API Writes** | ✅ Yes (15+ per run) | ❌ No (0 per run) |
| **Worker Dispatches** | ✅ Yes (implicit) | ✅ Yes (3 max per run) |
| **Architecture** | Hybrid (direct + dispatch) | Pure dispatch-only |
| **Orchestrator Complexity** | High (logic + writes) | Low (logic only) |

### Safe Outputs Comparison

| Safe Output | Old Model | New Model | Status |
|-------------|-----------|-----------|--------|
| `create-issue` | ✅ Max 1 | ❌ Removed | Must use worker |
| `add-comment` | ✅ Max 3 | ❌ Removed | Must use worker |
| `update-project` | ✅ Max 10 | ❌ Removed | Must use worker |
| `create-project-status-update` | ✅ Max 1 | ❌ Removed | Must use worker |
| `dispatch-workflow` | ✅ Max 3 | ✅ Max 3 | **Only option** |

## Key Changes

### Old Model (Before PR #11855)

Campaign orchestrators could **directly perform** multiple types of safe outputs:

| Safe Output Type | Max Allowed | Purpose | Governance Override |
|-----------------|-------------|---------|---------------------|
| `create-issue` | 1 | Create the Epic issue for the campaign (only created once) | No |
| `add-comment` | 3 | Comment on related issues/PRs as part of campaign coordination | Yes (`max-comments-per-run`) |
| `update-project` | 10 | Update the campaign's GitHub Project dashboard | Yes (`max-project-updates-per-run`) |
| `create-project-status-update` | 1 | Create project status updates for campaign summaries | No |
| `dispatch-workflow` | 3 | Dispatch worker workflows for the campaign | No |

**Total Direct Actions**: Orchestrators could perform up to **17 direct safe output operations** per run (1 + 3 + 10 + 1 + (workflows not dispatched via safe outputs in old model)).

**Custom GitHub Token Support**: The old model supported custom GitHub tokens for `update-project` and `create-project-status-update` via the `project-github-token` field in campaign specs.

### New Model (After PR #11855)

Campaign orchestrators are now **dispatch-only** and can ONLY use:

| Safe Output Type | Max Allowed | Purpose | Governance Override |
|-----------------|-------------|---------|---------------------|
| `dispatch-workflow` | 3 | Dispatch worker workflows with workflow_dispatch trigger | No |

**Total Direct Actions**: Orchestrators can perform up to **3 dispatch operations** per run.

**No Direct Writes**: All side effects (creating issues, commenting, updating projects, creating status updates) must now be performed by **dispatched worker workflows**.

## Detailed Comparison

### Safe Outputs Removed from Orchestrators

The following safe outputs were **removed** from campaign orchestrators in PR #11855:

#### 1. `create-issue` (Max: 1)
- **Old Behavior**: Orchestrator could directly create the campaign Epic issue
- **New Behavior**: Must dispatch a worker workflow to create issues
- **Impact**: Campaign orchestrators can no longer create Epic issues directly

#### 2. `add-comment` (Max: 3, Governed by `max-comments-per-run`)
- **Old Behavior**: Orchestrator could comment on up to 3 issues/PRs per run for coordination
- **New Behavior**: Must dispatch a worker workflow to add comments
- **Impact**: Campaign orchestrators can no longer comment directly on issues/PRs

#### 3. `update-project` (Max: 10, Governed by `max-project-updates-per-run`)
- **Old Behavior**: Orchestrator could update up to 10 project items per run
- **Supported Custom Token**: Yes (via `project-github-token`)
- **New Behavior**: Must dispatch a worker workflow to update projects
- **Impact**: Campaign orchestrators can no longer update GitHub Projects directly

#### 4. `create-project-status-update` (Max: 1)
- **Old Behavior**: Orchestrator could create project status update summaries directly
- **Supported Custom Token**: Yes (via `project-github-token`)
- **New Behavior**: Must dispatch a worker workflow to create status updates
- **Impact**: Campaign orchestrators can no longer create project status updates directly

### Safe Outputs Retained for Orchestrators

Only `dispatch-workflow` remains available to orchestrators:

#### `dispatch-workflow` (Max: 3)
- **Old Behavior**: Not used as a safe output in old model (workflows were dispatched differently)
- **New Behavior**: The ONLY safe output available to orchestrators
- **Max Per Run**: 3 workflow dispatches
- **Workflows**: Must be from the allowlisted workflows in the campaign spec
- **Impact**: This is now the **only** way orchestrators can take action

## Architectural Implications

### Before PR #11855: Hybrid Model

```
Campaign Orchestrator
├── Direct Actions (via safe outputs)
│   ├── Create Epic issue (1x)
│   ├── Add comments (3x)
│   ├── Update project items (10x)
│   └── Create status updates (1x)
└── Indirect Actions (via worker dispatch)
    └── Dispatch workers for domain-specific tasks
```

**Total Orchestrator Capabilities**: 15 direct actions + worker dispatches

### After PR #11855: Dispatch-Only Model

```
Campaign Orchestrator
└── Dispatch-Only Actions
    └── Dispatch worker workflows (3x max)
        ├── Workers create issues
        ├── Workers add comments
        ├── Workers update projects
        └── Workers create status updates
```

**Total Orchestrator Capabilities**: 3 dispatches only

## Rationale for Change

Based on the PR description and code changes, the new dispatch-only model provides:

1. **Clear Separation of Concerns**: Orchestrators coordinate, workers execute
2. **Simplified Permission Model**: Orchestrators don't need write permissions to GitHub API
3. **Better Scalability**: Worker workflows can be specialized and tested independently
4. **Reduced Orchestrator Complexity**: Orchestrators focus solely on decision-making and coordination
5. **Consistent Architecture**: All GitHub API writes go through dedicated worker workflows

## Impact on Governance Configuration

### Removed Governance Fields

The following governance fields no longer affect orchestrator behavior directly:

- `max-comments-per-run`: Previously limited `add-comment` safe output (max: 3 default)
- `max-project-updates-per-run`: Previously limited `update-project` safe output (max: 10 default)

These fields may still be relevant for **worker workflows** that perform these operations, but they no longer govern orchestrator behavior.

### Retained Governance Fields

- `max-discovery-items-per-run`: Still affects discovery precomputation
- `max-discovery-pages-per-run`: Still affects discovery pagination
- Worker workflows can define their own governance for safe outputs

## Code Evidence

### Old Configuration (Removed)

```go
// From pkg/campaign/orchestrator.go (before PR #11855)

safeOutputs := &workflow.SafeOutputsConfig{}

// Allow creating the Epic issue for the campaign (max: 1, only created once).
safeOutputs.CreateIssues = &workflow.CreateIssuesConfig{
    BaseSafeOutputConfig: workflow.BaseSafeOutputConfig{Max: 1}
}

// Allow commenting on related issues/PRs as part of campaign coordination.
safeOutputs.AddComments = &workflow.AddCommentsConfig{
    BaseSafeOutputConfig: workflow.BaseSafeOutputConfig{Max: maxComments}
}

// Allow updating the campaign's GitHub Project dashboard.
updateProjectConfig := &workflow.UpdateProjectConfig{
    BaseSafeOutputConfig: workflow.BaseSafeOutputConfig{Max: maxProjectUpdates}
}
if strings.TrimSpace(spec.ProjectGitHubToken) != "" {
    updateProjectConfig.GitHubToken = strings.TrimSpace(spec.ProjectGitHubToken)
}
safeOutputs.UpdateProjects = updateProjectConfig

// Allow creating project status updates for campaign summaries.
statusUpdateConfig := &workflow.CreateProjectStatusUpdateConfig{
    BaseSafeOutputConfig: workflow.BaseSafeOutputConfig{Max: 1}
}
if strings.TrimSpace(spec.ProjectGitHubToken) != "" {
    statusUpdateConfig.GitHubToken = strings.TrimSpace(spec.ProjectGitHubToken)
}
safeOutputs.CreateProjectStatusUpdates = statusUpdateConfig
```

### New Configuration (Current)

```go
// From pkg/campaign/orchestrator.go (after PR #11855)

// Campaign orchestrators are dispatch-only: they may only dispatch allowlisted
// workflows via the dispatch-workflow safe output. All side effects (Projects,
// issues/PRs, comments) must be performed by dispatched worker workflows.
safeOutputs := &workflow.SafeOutputsConfig{}
if len(spec.Workflows) > 0 {
    dispatchWorkflowConfig := &workflow.DispatchWorkflowConfig{
        BaseSafeOutputConfig: workflow.BaseSafeOutputConfig{Max: 3},
        Workflows:            spec.Workflows,
    }
    safeOutputs.DispatchWorkflow = dispatchWorkflowConfig
}
```

## Test Evidence

### Old Tests (Modified)

Before PR #11855, tests verified that orchestrators had `update-project`, `add-comment`, etc.:

```go
// Test: TestBuildOrchestrator_GovernanceOverridesSafeOutputMaxima
if data.SafeOutputs.AddComments.Max != 3 {
    t.Fatalf("unexpected add-comment max: got %d, want %d", 
        data.SafeOutputs.AddComments.Max, 3)
}
if data.SafeOutputs.UpdateProjects.Max != 4 {
    t.Fatalf("unexpected update-project max: got %d, want %d", 
        data.SafeOutputs.UpdateProjects.Max, 4)
}
```

### New Tests (Current)

After PR #11855, tests verify orchestrators ONLY have `dispatch-workflow`:

```go
// Test: TestBuildOrchestrator_GovernanceDoesNotGrantWriteSafeOutputs
if data.SafeOutputs.DispatchWorkflow == nil {
    t.Fatalf("expected dispatch-workflow safe output to be enabled")
}
if data.SafeOutputs.CreateIssues != nil || 
   data.SafeOutputs.AddComments != nil || 
   data.SafeOutputs.UpdateProjects != nil || 
   data.SafeOutputs.CreateProjectStatusUpdates != nil {
    t.Fatalf("expected no write safe outputs")
}
```

## Compiled Workflow Changes

### Old Compiled Orchestrator

The compiled `.lock.yml` file included these safe output handler configurations:

```yaml
env:
  GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG: >
    {
      "add_comment": {"max": 3},
      "create_issue": {"max": 1},
      "create_project_status_update": {"max": 1},
      "update_project": {"max": 10},
      "missing_data": {},
      "missing_tool": {},
      "noop": {"max": 1}
    }
```

**Available Tools Listed**: `add_comment`, `create_issue`, `create_project_status_update`, `missing_tool`, `noop`, `update_project`

### New Compiled Orchestrator

The compiled `.lock.yml` file now only includes dispatch-workflow:

```yaml
env:
  GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG: >
    {
      "dispatch_workflow": {
        "max": 3,
        "workflows": [
          "code-scanning-fixer",
          "security-fix-pr",
          "dependabot-bundler",
          "secret-scanning-triage"
        ]
      },
      "missing_data": {},
      "missing_tool": {},
      "noop": {"max": 1}
    }
```

**Available Tools Listed**: `dispatch_workflow`, `missing_tool`, `noop`

## Migration Path

For existing campaigns using the old model:

1. **Identify Direct Actions**: Review orchestrator logic for direct safe output usage
2. **Create Worker Workflows**: Build specialized workers for:
   - Creating campaign Epic issues
   - Adding coordination comments
   - Updating GitHub Projects
   - Creating project status updates
3. **Update Orchestrator**: Modify orchestrator to dispatch workers instead of performing actions directly
4. **Configure Allowlist**: Add worker workflows to the campaign spec's `workflows` field
5. **Test Dispatch Flow**: Verify worker workflows receive correct inputs and execute properly

## Recommendations

1. **Use Worker Workflows for All Writes**: Follow the dispatch-only model consistently
2. **Design Specialized Workers**: Create focused workers for specific operations (one worker per safe output type)
3. **Governance in Workers**: Move `max-comments-per-run` and `max-project-updates-per-run` to worker configurations
4. **Dispatch Budget**: Be mindful of the 3-dispatch limit when designing campaign flows
5. **Worker Communication**: Use workflow inputs and repo-memory for orchestrator-worker coordination

## Real-World Example: Security Alert Burndown Campaign

### Before PR #11855

The `security-alert-burndown` campaign orchestrator had access to:

```yaml
# Compiled safe outputs in .lock.yml
GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG: >
  {
    "add_comment": {"max": 3},
    "create_issue": {"max": 1},
    "create_project_status_update": {"max": 1},
    "update_project": {"max": 10},
    "missing_data": {},
    "missing_tool": {},
    "noop": {"max": 1}
  }
```

The orchestrator could:
- Create 1 campaign Epic issue
- Add 3 comments to issues/PRs
- Update 10 project items
- Create 1 project status update
- **Total**: 15 direct GitHub API write operations per run

### After PR #11855

The `security-alert-burndown` campaign orchestrator now has:

```yaml
# Compiled safe outputs in .lock.yml
GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG: >
  {
    "dispatch_workflow": {
      "max": 3,
      "workflows": [
        "code-scanning-fixer",
        "security-fix-pr",
        "dependabot-bundler",
        "secret-scanning-triage"
      ]
    },
    "missing_data": {},
    "missing_tool": {},
    "noop": {"max": 1}
  }
```

The orchestrator can:
- Dispatch up to 3 worker workflows from the allowlist
- **Total**: 3 workflow dispatches per run (no direct GitHub API writes)

All GitHub API writes (creating issues, commenting, updating projects, creating status updates) must now be performed by the dispatched worker workflows themselves.

## Safe Output Comparison Matrix

| Safe Output Type | Old Model (Max) | New Model (Max) | Migration Path |
|-----------------|-----------------|-----------------|----------------|
| `create-issue` | ✅ 1 | ❌ 0 | Create worker workflow with `create-issue` safe output |
| `add-comment` | ✅ 3 | ❌ 0 | Create worker workflow with `add-comment` safe output |
| `update-project` | ✅ 10 | ❌ 0 | Create worker workflow with `update-project` safe output |
| `create-project-status-update` | ✅ 1 | ❌ 0 | Create worker workflow with `create-project-status-update` safe output |
| `dispatch-workflow` | ✅ 3 (implicit) | ✅ 3 (only) | Already supported, now the only option |

## Conclusion

PR #11855 represents a significant architectural shift in campaign orchestrators:

- **Before**: Hybrid model with 4 direct safe outputs (create-issue, add-comment, update-project, create-project-status-update) + dispatch
- **After**: Pure dispatch-only model with only `dispatch-workflow` safe output

This change enforces a cleaner separation between coordination (orchestrators) and execution (workers), with all GitHub API writes now delegated to specialized worker workflows. The new model is more maintainable, testable, and scalable, though it requires campaigns to adopt a worker-based architecture for all side effects.

### Key Takeaways

1. **Orchestrators lost 4 safe outputs**: `create-issue`, `add-comment`, `update-project`, `create-project-status-update`
2. **Orchestrators retained 1 safe output**: `dispatch-workflow` (max: 3)
3. **Total capacity reduction**: From 15+ direct actions to 3 dispatch operations
4. **Architectural benefit**: Clear separation of coordination logic (orchestrators) from execution logic (workers)
5. **Migration requirement**: All campaigns must now use worker workflows for GitHub API writes
