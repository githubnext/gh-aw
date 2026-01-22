# Ralph Loop with GitHub Agentic Workflows

The **Ralph Loop** is an autonomous, iterative development pattern that enables AI agents to work persistently toward completing PRD (Product Requirements Document) goals. Named after Ralph Wiggum from The Simpsons, the pattern embodies persistence and continuous iteration—the agent keeps trying until success criteria are met.

## What is the Ralph Loop?

The Ralph Loop is a workflow pattern where an AI agent:

1. **Reads a structured PRD** - Contains features, user stories, and acceptance criteria
2. **Executes the highest-priority incomplete task** - Works on one task at a time
3. **Validates the work** - Runs tests, builds, and checks acceptance criteria
4. **Learns from failures** - Uses errors and feedback to improve the next iteration
5. **Commits successful work** - Only commits when tests pass and criteria are met
6. **Repeats** - Continues until all PRD items are complete or max iterations reached

The key difference from traditional AI agent workflows: **The loop runs autonomously without human intervention**, restarting with fresh context from persistent files (prd.json, progress.txt, git history) rather than relying on conversational memory.

## Why Ralph Loop with gh-aw?

GitHub Agentic Workflows provides native support for the Ralph Loop pattern through:

- **Workflow orchestration** - Schedule and trigger workflows automatically
- **Safe outputs** - Controlled, validated git operations
- **MCP integration** - Access to GitHub APIs, file systems, and external tools
- **Secure sandboxing** - Containerized execution with network isolation
- **Progress tracking** - Built-in memory and state management
- **Multiple AI engines** - Copilot CLI, Claude, Codex support

### Advantages over bash-based Ralph

| Feature | bash-based Ralph | gh-aw Ralph Loop |
|---------|------------------|------------------|
| **Infrastructure** | Local machine | GitHub Actions runners |
| **Security** | Runs with full system access | Sandboxed, read-only by default |
| **Scheduling** | Manual or cron | Built-in GitHub Actions scheduling |
| **State management** | Manual file handling | Safe outputs with validation |
| **Tool access** | Custom scripts | MCP servers (GitHub, files, web) |
| **Multi-repo** | Complex setup | Native campaign support |
| **Audit trail** | Git log only | Workflow runs + logs + artifacts |
| **Cost** | Local compute | GitHub Actions minutes |

## Complete Workflow: PRD to Completion

### 1. Write the PRD

Create a human-readable PRD describing the feature:

```markdown
# Feature: User Authentication System

## Overview
Add secure user authentication with JWT tokens and password hashing.

## User Stories

### US-001: User Registration
**As a** new user
**I want to** register with email and password
**So that** I can create an account

**Acceptance Criteria:**
- [ ] POST /api/register endpoint created
- [ ] Password hashed with bcrypt (10 rounds)
- [ ] Email validation implemented
- [ ] Returns JWT token on success
- [ ] Returns 400 for invalid inputs
- [ ] Unit tests pass with 100% coverage

### US-002: User Login
**As a** registered user
**I want to** log in with credentials
**So that** I can access protected resources

**Acceptance Criteria:**
- [ ] POST /api/login endpoint created
- [ ] Validates email and password
- [ ] Returns JWT token on success
- [ ] Returns 401 for invalid credentials
- [ ] Rate limiting implemented (5 attempts/15min)
- [ ] Unit tests pass with 100% coverage
```

### 2. Convert PRD to JSON

Use a workflow to convert the PRD into structured JSON:

```json
{
  "feature": "User Authentication System",
  "user_stories": [
    {
      "id": "US-001",
      "title": "User Registration",
      "priority": 1,
      "status": "pending",
      "acceptance_criteria": [
        {
          "id": "AC-001-1",
          "description": "POST /api/register endpoint created",
          "completed": false
        },
        {
          "id": "AC-001-2",
          "description": "Password hashed with bcrypt (10 rounds)",
          "completed": false
        }
      ],
      "tests": [
        {
          "name": "test_register_success",
          "status": "not_run"
        }
      ]
    }
  ]
}
```

### 3. Create Ralph Loop Workflow

Create `.github/workflows/ralph-feature-developer.md`:

```markdown
---
on:
  workflow_dispatch:
    inputs:
      prd_file:
        description: 'Path to PRD JSON file'
        required: true
        type: string
      max_iterations:
        description: 'Maximum iterations'
        required: false
        type: string
        default: '10'
engine: copilot
permissions:
  contents: write
  pull-requests: write
safe-outputs:
  commit-changes:
    branch: "ralph/{{inputs.prd_file}}"
    message-prefix: "ralph: "
  create-pull-request:
    title-prefix: "[ralph] "
    labels: [ralph-loop, automated]
tools:
  github:
    mode: remote
    toolsets: [default]
---

## Ralph Loop: Feature Development

You are a persistent AI developer working on completing a feature PRD.

### Context Files
- `prd.json` - Structured PRD with tasks and acceptance criteria
- `progress.txt` - Learnings from previous iterations
- `AGENTS.md` - Project-specific agent instructions

### Your Process

1. **Read and understand the current state:**
   - Load `prd.json` to see all tasks and their status
   - Read `progress.txt` to learn from previous attempts
   - Check `AGENTS.md` for project-specific guidelines

2. **Select the highest-priority incomplete task:**
   - Choose the first user story with `status: "pending"` or `status: "in_progress"`
   - If all stories complete, mark the PRD as done and exit

3. **Implement the task:**
   - Write clean, tested code following project conventions
   - Focus on one acceptance criterion at a time
   - Keep changes minimal and focused

4. **Validate your work:**
   - Run the project's build command
   - Run all tests (existing + new tests)
   - Check that all acceptance criteria are met

5. **Update state based on results:**
   
   **If tests pass:**
   - Mark acceptance criteria as `completed: true` in prd.json
   - Update user story status to `"completed"` if all criteria done
   - Append learnings to progress.txt
   - Commit the changes with a descriptive message
   
   **If tests fail:**
   - DO NOT commit
   - Add detailed failure analysis to progress.txt:
     - What was attempted
     - What failed (test output, error messages)
     - Insights for the next iteration
   - Update prd.json with `status: "in_progress"` and notes

6. **Report progress:**
   - Print summary of completed vs remaining tasks
   - Exit if all tasks complete or max iterations reached

### Success Criteria
All user stories in prd.json have `status: "completed"` and all tests pass.

### Important Rules
- NEVER commit failing code
- ALWAYS run tests before committing
- ALWAYS update prd.json and progress.txt
- Keep iterations focused on one task
- Learn from previous failures in progress.txt
```

### 4. Run the Loop

Trigger the workflow manually or on a schedule:

```bash
# Manual trigger
gh workflow run ralph-feature-developer.md \
  -f prd_file="auth-system-prd.json" \
  -f max_iterations="20"

# Or use workflow_dispatch with GitHub UI
```

### 5. Monitor Progress

Track progress through:

- **Workflow runs** - See each iteration attempt
- **Git commits** - Successful implementations
- **progress.txt** - Learning and iteration history
- **prd.json** - Real-time completion status

### 6. Review and Merge

When complete, review the PR:

```bash
# Review the changes
gh pr view ralph/auth-system-prd.json

# Review the progress history
gh pr diff ralph/auth-system-prd.json

# Merge if satisfied
gh pr merge ralph/auth-system-prd.json --squash
```

## File Structure

```
project/
├── .github/workflows/
│   ├── ralph-feature-developer.md      # Ralph Loop workflow
│   └── ralph-feature-developer.lock.yml # Compiled workflow
├── prds/
│   ├── auth-system.md                  # Human PRD
│   ├── auth-system-prd.json            # Structured PRD
│   └── auth-system-progress.txt        # Iteration learnings
├── AGENTS.md                           # Agent instructions
└── src/
    └── ... (your code)
```

## Best Practices

### PRD Writing
- **Be specific** - Clear acceptance criteria, not vague goals
- **Be testable** - Every criterion should have an objective test
- **Be atomic** - Break large features into small user stories
- **Include examples** - Show expected inputs/outputs
- **Define "done"** - Explicit success criteria

### Acceptance Criteria
✅ **Good:** "POST /api/register endpoint returns 201 and JWT token for valid input"  
❌ **Bad:** "Registration should work"

✅ **Good:** "Password must be hashed with bcrypt using 10 rounds"  
❌ **Bad:** "Use secure password storage"

### Iteration Management
- **Start small** - Test the loop with simple tasks first
- **Set limits** - Use max_iterations to prevent infinite loops
- **Monitor logs** - Check workflow runs for unexpected behavior
- **Track learnings** - progress.txt should be cumulative and informative

### Safety Guardrails
- **Read-only default** - Only grant write permissions explicitly
- **Branch isolation** - Each PRD gets its own branch
- **Review before merge** - Always human-review PR before merging
- **Test requirements** - Never commit without passing tests
- **Rollback plan** - Keep manual intervention options available

## Common Pitfalls

### 1. Vague Acceptance Criteria
**Problem:** Agent doesn't know when task is complete  
**Solution:** Make criteria specific and testable

### 2. No Test Coverage
**Problem:** Agent commits broken code  
**Solution:** Require tests for all acceptance criteria

### 3. Context Loss Between Iterations
**Problem:** Agent repeats mistakes  
**Solution:** Maintain detailed progress.txt with learnings

### 4. Infinite Loops
**Problem:** Agent gets stuck retrying the same approach  
**Solution:** Set max_iterations and monitor progress

### 5. Too-Large Tasks
**Problem:** Agent can't complete in one iteration  
**Solution:** Break into smaller user stories (15-30min tasks)

## Advanced Usage

### Multi-Repository Ralph
Use campaigns to run Ralph Loops across multiple repositories:

```yaml
# .github/workflows/ralph-multi-repo.campaign.md
discovery:
  repositories:
    - "org/repo1"
    - "org/repo2"
  
workers:
  - workflow: "ralph-feature-developer.md"
    inputs:
      prd_file: "{{payload.prd_file}}"
```

### Custom Validation Hooks
Add project-specific validation:

```markdown
4.5. **Run custom validation:**
   - Execute `./scripts/validate-feature.sh`
   - Check performance benchmarks
   - Validate API documentation updated
```

### Progressive PRD Refinement
Let the agent refine the PRD as it learns:

```markdown
If you discover missing requirements or edge cases:
- Add them to prd.json as new acceptance criteria
- Mark as "discovered-during-implementation"
- Continue working on the expanded scope
```

### Integration with ResearchPlanAssign
Combine Ralph Loop with existing patterns:

1. **Research Phase** - Static analysis finds refactoring opportunities
2. **Plan Phase** - Generate PRDs from findings
3. **Assign Phase** - Run Ralph Loop workflows to complete PRDs

## Tutorial Example

See [`examples/ralph/tutorial-example/`](./tutorial-example/) for a complete working example including:

- Sample feature PRD (user authentication)
- Generated prd.json with structured tasks
- Multiple iteration examples showing progress
- progress.txt with agent learnings
- AGENTS.md updates specific to the project
- Complete workflow implementation

## Comparison with bash-based Ralph

### What's Similar
- Core loop pattern (read, execute, validate, learn, repeat)
- PRD-driven task management
- Progress tracking through files
- Iterative refinement until complete

### What's Different with gh-aw

**Security:**
- Sandboxed execution environment
- Safe outputs with validation
- Read-only by default
- No local system access

**Infrastructure:**
- Runs on GitHub Actions runners
- No local setup required
- Automatic scheduling support
- Built-in artifact storage

**Tool Access:**
- MCP servers for GitHub, web, files
- Standardized tool interfaces
- Network isolation controls
- Supply chain security (SHA-pinned)

**Multi-Repository:**
- Native campaign support
- Centralized orchestration
- Consistent state management
- Organization-wide rollouts

## Learn More

- **[ResearchPlanAssign Guide](/gh-aw/guides/researchplanassign/)** - Combine with research/planning phases
- **[Campaigns Guide](/gh-aw/guides/campaigns/)** - Run Ralph Loops at scale
- **[Safe Outputs Reference](/gh-aw/reference/safe-outputs/)** - Configure git operations
- **[Security Guide](/gh-aw/guides/security/)** - Understand security model
- **[Tutorial](../../docs/src/content/docs/guides/ralph-loop.md)** - Step-by-step walkthrough

## Related Projects

- **[snarktank/ralph](https://github.com/snarktank/ralph)** - Original bash-based Ralph implementation
- **[The Agentics](https://github.com/githubnext/agentics)** - Reusable workflow components

---

> [!TIP]
> Start with a simple, single-task PRD to validate your workflow setup before attempting complex multi-story features. The Ralph Loop is powerful but requires careful PRD design and monitoring.

> [!WARNING]
> Ralph Loops run autonomously and can make many commits. Always review PRs before merging and set reasonable max_iterations limits to prevent runaway execution.
