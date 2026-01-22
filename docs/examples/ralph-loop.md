# Ralph Loop Pattern for GitHub Agentic Workflows

## Executive Summary

The Ralph Loop is an autonomous AI agent pattern that runs AI coding tools repeatedly until all requirements are complete. This document explores how the pattern can be implemented using GitHub Agentic Workflows (gh-aw), mapping Ralph's core concepts to gh-aw features and identifying both opportunities and limitations.

**Key Finding**: gh-aw can implement a Ralph-like pattern using GitHub Actions scheduling, safe-outputs for persistence, and workflow chaining. However, gh-aw's design emphasizes safety and structured outputs over full autonomy, creating a "Guided Ralph" approach with stronger guardrails.

## Overview of the Ralph Loop Pattern

### What is Ralph?

Ralph is an autonomous AI agent loop pattern developed by [Geoffrey Huntley](https://ghuntley.com/ralph/) that runs AI coding tools (Amp, Claude Code) repeatedly in a bash loop until all PRD (Product Requirements Document) items are complete. Named after Ralph Wiggum from The Simpsons, it emphasizes persistent, autonomous execution.

### Core Characteristics

1. **Fresh Context Per Iteration**: Each loop iteration spawns a new AI instance with clean context, avoiding LLM context window limitations
2. **File-Based Memory**: State persists via git history, `progress.txt`, and `prd.json` rather than in-memory conversation history
3. **Small Atomic Tasks**: PRD broken into small stories that fit within a single context window
4. **Feedback Loops**: Quality checks (typecheck, tests, lints) provide automatic backpressure
5. **Learning Persistence**: `AGENTS.md` files capture learnings for future iterations
6. **Autonomous Execution**: Runs unattended for hours/days until completion or max iterations

### Ralph Workflow

```
┌─────────────────────────────────────┐
│  1. Load prd.json + progress.txt    │
│  2. Pick highest priority task      │
│     where passes: false             │
│  3. Implement the task              │
│  4. Run quality checks              │
│  5. Commit if checks pass           │
│  6. Update prd.json (passes: true)  │
│  7. Append learnings to progress.txt│
│  8. Repeat until all complete       │
└─────────────────────────────────────┘
```

**Stop Condition**: All stories have `passes: true` → Output `<promise>COMPLETE</promise>` → Exit loop

## Mapping Ralph Concepts to gh-aw Features

| Ralph Concept | gh-aw Implementation | Notes |
|---------------|---------------------|-------|
| **Bash Loop** | GitHub Actions workflow with scheduled triggers + workflow_dispatch | Scheduled cron or event-triggered execution |
| **Fresh Context** | New workflow run = fresh AI agent instance | Each workflow run starts clean |
| **prd.json** | GitHub Discussion or Issue with structured JSON in body | Persistent task list accessible across runs |
| **progress.txt** | Discussion comments or issue comments | Append-only learning log |
| **AGENTS.md** | `.github/agents/*.agent.md` or custom instructions | Built-in support for agent instructions |
| **Git Commits** | safe-outputs: create-pull-request | Validated PR creation with safety checks |
| **Quality Checks** | tools: bash with allowed commands | Run tests/linters in workflow |
| **Single Story Focus** | Workflow reads one task from PRD, marks complete | Task selection logic in workflow prompt |
| **Autonomy** | Limited - requires safe-outputs for writes | Trade-off: safety over full autonomy |

## Proposed Workflow Structure

### Phase 1: PRD Initialization (Manual or One-Time)

Create a GitHub Discussion or Issue with PRD structure:

```markdown
---
title: "[PRD] Feature Name"
category: "ideas"
---

## User Stories

```json
{
  "branchName": "feature-name",
  "userStories": [
    {
      "id": 1,
      "title": "Add database column for user preferences",
      "description": "Create migration and add preferences column to users table",
      "priority": 1,
      "passes": false,
      "acceptanceCriteria": [
        "Migration file created",
        "Tests pass",
        "No lint errors"
      ]
    },
    {
      "id": 2,
      "title": "Update API to expose preferences",
      "description": "Add GET/POST endpoints for user preferences",
      "priority": 2,
      "passes": false,
      "acceptanceCriteria": [
        "Endpoints return correct data",
        "Tests cover edge cases",
        "API docs updated"
      ]
    }
  ]
}
```

## Progress

(Learnings will be appended here)
```

### Phase 2: Ralph Loop Workflow

File: `.github/workflows/ralph-loop.md`

```markdown
---
name: Ralph Loop - Autonomous Implementation
description: Implements PRD stories one at a time with quality checks

on:
  workflow_dispatch:
    inputs:
      prd_discussion_number:
        description: "PRD Discussion number"
        required: true
      max_iterations:
        description: "Maximum iterations (default: 10)"
        default: "10"
  schedule:
    - cron: "0 */2 * * *"  # Every 2 hours

permissions:
  contents: read
  discussions: read
  pull-requests: read

safe-outputs:
  create-pull-request:
    title-prefix: "[ralph] "
    labels: [automated, ralph-loop]
    draft: true
  update-discussion:
    allowed-discussion-numbers: [${{ inputs.prd_discussion_number }}]

tools:
  github:
    toolsets: [default]
  bash:
    - "npm test"
    - "npm run typecheck"
    - "npm run lint"
    - "make test"
    - "make lint"
    - "go test ./..."

engine: copilot
timeout-minutes: 30
---

# Ralph Loop Agent - Autonomous Story Implementation

You are the Ralph Loop Agent, implementing PRD stories autonomously with quality checks.

## Current Context

- **Repository**: ${{ github.repository }}
- **PRD Discussion**: #${{ inputs.prd_discussion_number }}
- **Max Iterations**: ${{ inputs.max_iterations || 10 }}
- **Branch Base**: main

## Your Mission

Implement ONE user story from the PRD, verify it passes quality checks, and update progress.

## Step 1: Load PRD State

1. Fetch discussion #${{ inputs.prd_discussion_number }}
2. Parse the JSON in the discussion body to extract `userStories` array
3. Read progress comments to understand previous learnings
4. Check git history to see what's already been implemented

## Step 2: Select Next Story

Find the highest priority story where `passes: false`:

```javascript
const nextStory = userStories
  .filter(s => !s.passes)
  .sort((a, b) => a.priority - b.priority)[0];
```

**If no stories remain** (all `passes: true`):
- Add discussion comment: "✅ <promise>COMPLETE</promise> - All stories implemented!"
- Exit successfully

**If story found**:
- Proceed to Step 3

## Step 3: Implement the Story

1. **Check if branch exists**: `git branch -r | grep ${{ branchName }}`
   - If not, create: `git checkout -b ${{ branchName }}`
   - If exists, checkout: `git checkout ${{ branchName }}`

2. **Implement the story**:
   - Focus ONLY on the selected story's requirements
   - Keep changes small and atomic
   - Follow acceptance criteria precisely
   - Use existing codebase patterns (check `.github/agents/*.agent.md` for conventions)

3. **Self-review**:
   - Did I complete ALL acceptance criteria?
   - Are changes minimal and focused?
   - Did I follow repository conventions?

## Step 4: Run Quality Checks

Execute quality checks as feedback loop:

```bash
# Run appropriate checks for the codebase
npm test || make test || go test ./...
npm run typecheck || true
npm run lint || make lint || true
```

**Interpret results**:
- ✅ **All pass**: Proceed to Step 5
- ❌ **Any fail**: Fix issues and re-run checks
  - Maximum 3 fix attempts
  - If still failing after 3 attempts, add comment explaining failure and exit

## Step 5: Create Pull Request

Use `safe-outputs: create-pull-request` to create a draft PR:

**PR Title**: `[ralph] Story #${{ storyId }}: ${{ storyTitle }}`

**PR Body**:
```markdown
## Story Implementation

**Story ID**: #${{ storyId }}
**Priority**: ${{ priority }}

### Description
${{ storyDescription }}

### Acceptance Criteria
${{ acceptanceCriteria.map(c => `- [x] ${c}`).join('\n') }}

### Quality Checks
- [x] Tests pass
- [x] Linting passes  
- [x] Type checks pass

### Learnings
${{ any patterns, gotchas, or insights discovered }}

---
*Automated by Ralph Loop workflow*
```

## Step 6: Update PRD

Add a comment to discussion #${{ inputs.prd_discussion_number }}:

```markdown
## ✅ Story #${{ storyId }} Implemented

**Story**: ${{ storyTitle }}
**PR**: #${{ prNumber }}
**Branch**: ${{ branchName }}

### Learnings
- ${{ learning1 }}
- ${{ learning2 }}

### Next Story
${{ nextUnfinishedStory || "All complete!" }}
```

**Update story status in PRD JSON** (in discussion body):
- Set `passes: true` for the completed story
- This requires updating the discussion body with modified JSON

## Important Guidelines

### Small Tasks Rule
Each story should be completable in ONE iteration. If a story is too large:
- Add comment explaining it needs splitting
- Suggest 2-3 smaller stories
- Mark original story as `passes: false` with `needs-split: true`

### AGENTS.md Updates
After implementing a story, consider updating `.github/agents/developer.instructions.agent.md` if you discovered:
- New patterns used in this codebase
- Gotchas to avoid (e.g., "always update index.ts exports")
- Useful context for future iterations

### Failure Handling
If quality checks fail after 3 attempts:
- Create draft PR anyway (for human review)
- Add comment to PRD discussion explaining failure
- Mark story as `passes: false` but add `attempted: true`
- Do NOT retry same story in next iteration

### Stop Conditions
1. **Success**: All stories have `passes: true` → Output `<promise>COMPLETE</promise>`
2. **Max Iterations**: Reached max_iterations limit → Report progress and exit
3. **No Completable Stories**: All remaining stories need splitting → Exit with summary

## Example Execution

```
Iteration 1:
- Load PRD: 3 stories, 0 complete
- Select: Story #1 (priority 1)
- Implement: Add database column
- Quality: ✅ All pass
- PR: Created #123
- Update: Story #1 → passes: true
- Progress: 1/3 complete

Iteration 2:
- Load PRD: 3 stories, 1 complete  
- Select: Story #2 (priority 2)
- Implement: Update API endpoints
- Quality: ✅ All pass
- PR: Created #124
- Update: Story #2 → passes: true
- Progress: 2/3 complete

Iteration 3:
- Load PRD: 3 stories, 2 complete
- Select: Story #3 (priority 3)
- Implement: Add UI component
- Quality: ✅ All pass
- PR: Created #125
- Update: Story #3 → passes: true
- Progress: 3/3 complete
- Output: ✅ <promise>COMPLETE</promise>
```

Begin execution now. Load the PRD, select the next story, and implement it with quality checks.
```

### Phase 3: Orchestration Workflow (Optional)

A controller workflow that triggers the Ralph Loop repeatedly:

```markdown
---
name: Ralph Orchestrator
on:
  workflow_dispatch:
    inputs:
      prd_discussion_number:
        required: true

permissions: read-all

jobs:
  orchestrate:
    runs-on: ubuntu-latest
    steps:
      - name: Trigger Ralph Loop
        run: |
          for i in {1..10}; do
            gh workflow run ralph-loop.yml \
              -f prd_discussion_number=${{ inputs.prd_discussion_number }}
            sleep 300  # Wait 5 minutes between iterations
          done
---
```

## Key Differences from Bash-Based Ralph

| Aspect | Bash Ralph | gh-aw Ralph |
|--------|------------|-------------|
| **Iteration Control** | Local bash while loop | GitHub Actions scheduled triggers or workflow_dispatch |
| **State Persistence** | Local files (prd.json, progress.txt) | GitHub Discussions/Issues API |
| **Autonomy** | Full - directly commits to repo | Limited - uses safe-outputs for PR creation |
| **Quality Checks** | Direct bash execution | Restricted bash commands via tools: bash allowlist |
| **Memory** | Git + local files | Git + GitHub API (discussions/issues) |
| **Development Loop** | Local development machine | GitHub Actions runners (cloud) |
| **Feedback Speed** | Immediate (local) | 2-5 minutes (GitHub Actions startup time) |
| **Cost** | Free (local compute) | GitHub Actions minutes (free tier available) |
| **Security** | Trusted local environment | Sandboxed execution with input validation |

## Gaps and Limitations

### 1. Autonomy vs Safety Trade-off

**Gap**: gh-aw prioritizes safety over full autonomy
- Ralph bash: Directly commits and pushes
- gh-aw: Requires safe-outputs for writes, creates draft PRs

**Impact**: Human review required for each PR, reducing full autonomy
**Mitigation**: Use `safe-outputs: auto-merge` (if/when available) or external automation to auto-merge draft PRs after checks pass

### 2. Iteration Speed

**Gap**: GitHub Actions startup overhead (2-5 minutes per iteration)
- Ralph bash: Instant iteration start
- gh-aw: Cold start penalty for each workflow run

**Impact**: Slower feedback loops, harder to achieve "overnight completion"
**Mitigation**: 
- Use scheduled triggers (every 30 min) instead of workflow chaining
- Optimize workflow for faster startup (pre-built containers)
- Consider self-hosted runners for faster execution

### 3. State Management Complexity

**Gap**: GitHub API adds complexity vs local files
- Ralph bash: Simple file read/write (prd.json, progress.txt)
- gh-aw: API calls to discussions/issues, JSON parsing in markdown

**Impact**: More complex state updates, potential for race conditions
**Mitigation**:
- Use GitHub Discussions as single source of truth
- Implement optimistic concurrency (check updated_at before writes)
- Consider using GitHub Projects API for structured task management

### 4. Context Window Management

**Gap**: Limited ability to continue long-running tasks
- Ralph bash: Can use auto-handoff features in Amp/Claude
- gh-aw: Each workflow run has fixed context limit (current model limits)

**Impact**: Very large stories may not fit in single iteration
**Mitigation**:
- Enforce strict story size limits (< 200 lines of code change)
- Implement story splitting detection and suggestions
- Use multi-step workflows with shared state

### 5. Local Development Testing

**Gap**: Difficult to test Ralph loop locally
- Ralph bash: Run locally with `./ralph.sh`
- gh-aw: Requires `gh aw run --interactive` or pushing to GitHub

**Impact**: Slower development cycle for workflow authors
**Mitigation**:
- Use `gh aw run --interactive` for local testing
- Create small test PRDs for validation
- Implement dry-run mode that skips PRs

### 6. Error Recovery

**Gap**: Limited error recovery mechanisms
- Ralph bash: Manual intervention in local shell
- gh-aw: Workflow fails, requires manual re-trigger

**Impact**: Blocked execution until human intervention
**Mitigation**:
- Implement retry logic with exponential backoff
- Add failure comments to PRD with debugging info
- Create separate "recovery" workflow for stuck stories

### 7. Workflow Scheduling Limitations

**Gap**: GitHub Actions scheduling constraints
- Ralph bash: Runs continuously until complete
- gh-aw: Maximum cron frequency is every 5 minutes, but practical limit for cost/rate limiting is every 30-60 min

**Impact**: Cannot achieve rapid iteration cycles
**Mitigation**:
- Use workflow_dispatch for manual iteration control
- Implement workflow chaining for faster cycles (within rate limits)
- Consider repository_dispatch for external triggering

## Implementation Recommendations

### Start Small: Single-Story Workflow

Before implementing full Ralph Loop, create a simpler workflow:

```markdown
---
name: Implement Single Story
on: workflow_dispatch

safe-outputs:
  create-pull-request:
    title-prefix: "[story] "
---

Implement story #1 from the PRD, run tests, and create a PR.
```

**Benefits**:
- Validate gh-aw capabilities
- Test safe-outputs configuration
- Understand workflow execution model
- Build confidence before full automation

### Use Existing DailyOps Pattern

gh-aw already has a similar pattern in DailyOps workflows:
- Scheduled execution (weekdays)
- Progress tracking via discussions
- Incremental changes over time
- Human approval between phases

**Recommendation**: Adapt DailyOps pattern for Ralph-like execution

### Hybrid Approach: Human-in-the-Loop Ralph

Balance autonomy with safety:

1. **Phase 1**: Ralph Loop creates draft PRs automatically
2. **Phase 2**: Human reviews and approves PRs
3. **Phase 3**: Auto-merge on approval triggers next iteration

This maintains Ralph's autonomous implementation while preserving gh-aw's safety guarantees.

### Consider Campaign Workflows

For multi-repository Ralph execution, use gh-aw campaigns:

```markdown
---
campaign:
  targets:
    - org/repo1
    - org/repo2
---

Implement the same story across multiple repositories.
```

**Use Case**: Large-scale migrations or consistent feature rollouts

## Security Considerations

### Trust Boundaries

Ralph assumes trusted local environment. gh-aw must handle untrusted inputs:

1. **PRD Validation**: Validate JSON structure to prevent injection
2. **Command Restrictions**: Only allow safe bash commands
3. **PR Review**: All changes require human review (draft PRs)
4. **Branch Protection**: Use protected branches to prevent direct commits
5. **Secrets Management**: Never expose secrets in workflow logs

### Safe-Outputs Guardrails

All write operations go through safe-outputs:
- **create-pull-request**: Validates PR title, labels, prevents malicious content
- **update-discussion**: Restricts which discussions can be modified
- **create-issue**: Enforces title prefix, label requirements

**Recommendation**: Run Ralph workflows with minimum necessary permissions

## Future Enhancements

### 1. Native Ralph Loop Support in gh-aw

Add first-class support for Ralph pattern:

```yaml
ralph-loop:
  enabled: true
  max-iterations: 10
  prd-source: discussion
  quality-checks:
    - make test
    - make lint
```

### 2. Story Status Tracking

Built-in support for PRD tracking:

```yaml
safe-outputs:
  update-prd-status:
    discussion: ${{ inputs.prd_discussion_number }}
    story-id: ${{ inputs.story_id }}
    status: complete
```

### 3. Smart Story Selection

AI-powered story prioritization:
- Analyze dependencies between stories
- Suggest optimal execution order
- Detect stories that should be combined or split

### 4. Continuous Ralph Mode

Run workflow in continuous mode with automatic re-triggering:

```yaml
ralph-loop:
  mode: continuous
  check-interval: 5m
  stop-on: all-complete
```

### 5. Multi-Agent Collaboration

Multiple Ralph agents working on different stories in parallel:

```yaml
ralph-loop:
  parallelism: 3
  strategy: priority-based
```

## Conclusion

GitHub Agentic Workflows can implement a Ralph-like pattern with modifications to accommodate its security-first design. The result is a "Guided Ralph" approach that trades some autonomy for safety guarantees, structured outputs, and GitHub-native integration.

**Best Fit Use Cases**:
1. **Incremental Migrations**: Daily scheduled tasks that chip away at technical debt
2. **Documentation Maintenance**: Automated doc updates based on code changes  
3. **Dependency Updates**: Continuous dependency upgrade workflows
4. **Phased Rollouts**: Implement features across multiple repositories over time

**Not Ideal For**:
1. **Rapid Local Development**: Better served by bash Ralph (instant iteration)
2. **Unattended Overnight Runs**: GitHub Actions scheduling limits rapid iteration
3. **Direct Commit Workflows**: gh-aw requires PR-based changes for safety

The Ralph Loop pattern represents an exciting direction for agentic workflows. As gh-aw evolves, incorporating more Ralph-inspired features while maintaining safety guardrails will enable increasingly autonomous yet secure development automation.

## References

- **Ralph Repository**: https://github.com/snarktank/ralph
- **Geoffrey Huntley's Ralph Article**: https://ghuntley.com/ralph/
- **gh-aw Documentation**: https://githubnext.github.io/gh-aw/
- **Ralph Playbook**: https://claytonfarr.github.io/ralph-playbook/
- **DailyOps Pattern**: [docs/src/content/docs/examples/scheduled/dailyops.md](../src/content/docs/examples/scheduled/dailyops.md)

## See Also

- [DailyOps Pattern](../src/content/docs/examples/scheduled/dailyops.md) - Similar incremental automation pattern in gh-aw
- [Safe Outputs Guide](https://githubnext.github.io/gh-aw/reference/safe-outputs/) - Understanding gh-aw's write operation guardrails
- [Campaign Workflows](https://githubnext.github.io/gh-aw/guides/campaigns/) - Multi-repository execution patterns
