# Spec-Kit PR Creation Blocking Issue

## Problem Statement

The spec-kit-execute workflow successfully creates implementation files using the `create` and `edit` tools, but fails at PR creation with error:

```
McpError: MCP error -32603: No changes to commit - no commits found
```

## Root Cause Analysis

1. **File Creation Works**: Files are successfully created in the workspace:
   - `pkg/testutil/test_feature.go`
   - `pkg/testutil/test_feature_test.go`
   - `.specify/specs/001-test-feature/tasks.md` (modified)

2. **Git Status Shows Changes**: Running `git status` confirms files are created:
   ```
   Changes not staged for commit:
     modified:   .specify/specs/001-test-feature/tasks.md
   
   Untracked files:
     pkg/testutil/test_feature.go
     pkg/testutil/test_feature_test.go
   ```

3. **PR Tool Requires Commits**: The `create_pull_request` safe-output tool requires committed changes to create a PR, but fails because there are no commits.

4. **Git Commands Blocked**: Bash allowlist does not include:
   - `git add`
   - `git commit`
   - `git config`
   - `git push`

## The Circular Dependency

```
create files → files exist in workspace → git status shows changes
     ↑                                              ↓
     |                                    cannot commit (git blocked)
     |                                              ↓
     └─────── cannot create PR (no commits) ←──────┘
```

## Attempted Solutions (All Failed)

1. ❌ Direct git commands → Permission denied
2. ❌ Using make commands → Permission denied
3. ❌ Using install command → Permission denied
4. ❌ Creating directories with bash → Permission denied
5. ❌ Calling create_pull_request without commits → "No changes to commit" error

## The Critical Gap

**The workflow architecture assumes one of these:**

**Option A**: The safe-output tool automatically commits changes
- This does NOT happen - tool expects commits to exist

**Option B**: The agent can run git commands
- This is BLOCKED by bash allowlist

**Option C**: The workflow has a separate commit step
- This does NOT exist in current workflow

**Option D**: Files are committed by another mechanism
- No such mechanism exists

## Solution Required

The workflow needs to be modified to include a commit step BEFORE the agent execution phase:

```yaml
jobs:
  implement:
    steps:
      - uses: actions/checkout@v4
      
      - name: Agent Implementation
        # Agent creates files using create/edit tools
        
      - name: Commit Changes  # ← MISSING STEP
        run: |
          git config user.name "github-actions[bot]"
          git config user.email "github-actions[bot]@users.noreply.github.com"
          git add -A
          git commit -m "feat: spec-kit implementation" || echo "No changes"
      
      - name: Create PR
        # Now create_pull_request can work with committed changes
```

## Alternative Solutions

### Alt 1: Add git commands to bash allowlist
```yaml
bash:
  - "git add *"
  - "git commit -m *"
  - "git config user.* *"
```
**Issue**: Still requires manual orchestration by agent

### Alt 2: Create a safe-output for staging
```yaml
safe-outputs:
  stage-changes:
    # Tool that stages and commits all changes
```
**Benefit**: Clean abstraction, agent just calls one tool

### Alt 3: Workflow automatically commits at end
Add a post-agent step that commits any changes
**Benefit**: Simplest, works for all implementations

## Recommendation

**Alt 3** is the cleanest solution. Add a workflow step after agent execution:

```yaml
- name: Auto-commit implementation
  if: always()
  run: |
    git config user.name "spec-kit-bot[bot]"
    git config user.email "spec-kit-bot[bot]@users.noreply.github.com"
    git add -A
    if git diff --staged --quiet; then
      echo "No changes to commit"
    else
      git commit -m "feat(spec-kit): implement ${{ github.event.inputs.feature || 'automated' }}"
    fi
```

This allows the agent to focus on implementation while the workflow handles git mechanics.

## Impact

**Current State**: 0% success rate on PR creation (8 consecutive failures)
**With Fix**: Should achieve 100% success rate

## Date

2025-12-26

## Status

CRITICAL - Blocks all spec-kit automation
