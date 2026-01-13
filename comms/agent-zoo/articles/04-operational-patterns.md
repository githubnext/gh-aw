# 9 Patterns for Automated Agent Ops on GitHub

**Strategic patterns for operating agents in the GitHub ecosystem**

[← Previous: Design Patterns](03-design-patterns.md) | [Back to Index](../index.md) | [Next: Imports & Sharing →](05-imports-and-sharing.md)

---

Beyond the 12 core behavioral patterns that define what agents **do**, Peli's Agent Factory revealed several strategic operational patterns for **how** agentic workflows operate in the context of GitHub's information primitives. These patterns emerged from building and operating workflows at scale and represent battle-tested approaches to common challenges.

While design patterns describe agent architecture, operational patterns describe how agents integrate with GitHub's workflow, issue, project, and event systems to create effective automation.

## Pattern 1: ChatOps - Command-Driven Interactions 💬

### Overview

Workflows triggered by slash commands (`/review`, `/deploy`, `/fix`) in issue or PR comments. Creates an interactive conversation interface where team members can invoke powerful AI capabilities with simple commands.

### When to Use

- Code reviews on demand
- Performance investigations
- Bug fixes and optimizations
- Research and documentation requests
- Any operation requiring user authorization

### How It Works

1. User comments `/command` on an issue or PR
2. Workflow triggers on `issue_comment` or `pull_request_comment` event
3. Comment is parsed for command and parameters
4. Role-gating validates user permissions
5. Agent executes and responds in thread
6. Cache-memory prevents duplicate work

### Example: Grumpy Reviewer

The [`grumpy-reviewer`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/grumpy-reviewer.md) workflow demonstrates ChatOps pattern perfectly:

- Triggered by `/grumpy` on PR comments
- Performs critical code review with distinctive personality
- Uses cache memory to avoid duplicate feedback
- Role-gated to prevent abuse
- Responds directly in PR thread

### Key Benefits

- Natural, conversational interface
- Role-based access control built-in
- Context-aware (knows which issue/PR)
- Immediate feedback
- Audit trail in comments

### Implementation Tips

- Use clear, memorable command names
- Document commands in README
- Implement role-gating for sensitive operations
- Add help text for `/command help`
- Use cache-memory to track command history

**Learn more**: [ChatOps Examples](https://githubnext.github.io/gh-aw/examples/comment-triggered/chatops/)

---

## Pattern 2: DailyOps - Scheduled Incremental Progress 📅

### Overview

Workflows that run on weekday schedules to make small, daily progress toward large goals. Instead of overwhelming teams with major changes, work happens automatically in manageable pieces that are easy to review and integrate.

### When to Use

- Test coverage improvements
- Performance optimization
- Documentation updates
- Technical debt reduction
- Dependency management

### How It Works

1. Workflow runs on schedule (e.g., `0 9 * * 1-5` for weekdays at 9am)
2. Agent checks state from previous runs
3. Makes incremental progress (1-3 small changes)
4. Creates PR or issue with results
5. Next day, continues from where it left off

### Example: Daily Test Improver

The [`daily-test-improver`](https://github.com/githubnext/agentics/blob/main/workflows/daily-test-improver.md) workflow systematically identifies coverage gaps and implements new tests over multiple days:

**Phase 1 (Day 1-2)**: Research coverage gaps and create plan
**Phase 2 (Day 3-4)**: Set up test infrastructure
**Phase 3 (Day 5+)**: Implement tests incrementally with phased approval

### Key Benefits

- Sustainable, non-disruptive improvements
- Easy to review small changes
- Builds momentum over time
- Human checkpoints between phases
- Natural task breaking

### Implementation Tips

- Use repo-memory for state persistence
- Limit changes per run (1-3 items)
- Create daily PRs with descriptive titles
- Include progress reports in PR descriptions
- Allow human intervention at any phase

**Learn more**: [DailyOps Examples](https://githubnext.github.io/gh-aw/examples/scheduled/dailyops/)

---

## Pattern 3: IssueOps - Event-Driven Issue Automation 🎫

### Overview

Workflows that transform GitHub issues into automation triggers, automatically analyzing, categorizing, and responding to issues as they're created or updated. Uses safe outputs to ensure secure automated responses.

### When to Use

- Automatic issue triage
- Issue classification and labeling
- Template validation
- Initial response automation
- Related issue linking

### How It Works

1. Workflow triggers on `issues: opened` or `issues: edited`
2. Agent analyzes issue content
3. Determines appropriate labels, assignees, projects
4. Uses safe outputs to apply changes
5. Optional: Posts welcome comment or requests clarification

### Example: Issue Triage Agent

The [`issue-triage-agent`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/issue-triage-agent.md) automatically labels and categorizes new issues:

- Analyzes issue title and body
- Applies relevant labels (bug, feature, documentation, etc.)
- Estimates priority based on content
- Routes to appropriate team
- Posts helpful resources

### Key Benefits

- Immediate issue processing
- Consistent categorization
- Reduces manual triage burden
- Improves issue quality over time
- Creates audit trail

### Implementation Tips

- Use safe outputs for all modifications
- Include confidence scores in labels
- Allow manual override
- Track triage accuracy
- Update classification rules based on feedback

**Learn more**: [IssueOps Examples](https://githubnext.github.io/gh-aw/examples/issue-pr-events/issueops/)

---

## Pattern 4: LabelOps - Label-Driven Workflow Automation 🏷️

### Overview

Workflows that use GitHub labels as triggers, metadata, and state markers. Responds to specific label changes with filtering to activate only for relevant labels while maintaining secure automated responses.

### When to Use

- Priority escalation
- Workflow routing
- State machine implementation
- Feature flagging
- Team assignment

### How It Works

1. Workflow triggers on `issues: labeled` or `pull_request: labeled`
2. Filter checks for specific label(s)
3. Agent takes label-specific action
4. Updates state via additional labels or project fields
5. May create follow-up issues or notifications

### Example: Priority Escalation

When `priority: critical` label is added:
- Notifies team leads
- Adds to urgent project board
- Creates daily reminder
- Updates SLA tracking

### Key Benefits

- Visual state representation
- User-friendly trigger mechanism
- Easy to understand workflows
- GitHub-native pattern
- Queryable via label filters

### Implementation Tips

- Use consistent label naming conventions
- Document label meanings
- Implement label hierarchies
- Avoid label proliferation
- Use label descriptions

**Learn more**: [LabelOps Examples](https://githubnext.github.io/gh-aw/examples/issue-pr-events/labelops/)

---

## Pattern 5: ProjectOps - AI-Powered Project Board Management 📊

### Overview

Workflows that keep GitHub Projects v2 boards up to date using AI to analyze issues/PRs and intelligently decide routing, status, priority, and field values. Safe output architecture ensures security while automating project management.

### When to Use

- Automatic project board updates
- Sprint planning assistance
- Priority management
- Status tracking
- Resource allocation

### How It Works

1. Workflow triggers on issue/PR events
2. Agent analyzes content and context
3. Determines appropriate project, status, fields
4. Uses safe outputs to update project
5. Notifies relevant stakeholders

### Example: Automatic Board Population

When issue is created:
- AI determines which project(s) it belongs to
- Sets initial status (Backlog, To Do, etc.)
- Estimates size/effort
- Assigns priority
- Sets sprint/milestone if applicable

### Key Benefits

- Always up-to-date project boards
- Reduces manual project management
- Consistent field population
- AI-powered classification
- Integrates with existing workflows

### Implementation Tips

- Use project field types effectively
- Define clear status transitions
- Implement confidence thresholds
- Allow manual overrides
- Track automation accuracy

**Learn more**: [ProjectOps Examples](https://githubnext.github.io/gh-aw/examples/issue-pr-events/projectops/)

---

## Pattern 6: ResearchPlanAssign - Scaffolded Improvement Strategy 🔬

### Overview

A three-phase strategy that keeps developers in control while leveraging AI agents for systematic code improvements. Provides clear decision points at each phase: Research (investigate), Plan (break down work), Assign (execute).

### When to Use

- Large refactoring initiatives
- Code quality campaigns
- Architecture improvements
- Systematic cleanup projects
- Knowledge transfer projects

### How It Works

**Phase 1: Research**
- Agent analyzes codebase
- Identifies improvement opportunities
- Creates research discussion with findings
- Human reviews and approves direction

**Phase 2: Plan**
- Agent creates detailed implementation plan
- Breaks work into manageable issues
- Estimates effort and dependencies
- Human reviews and prioritizes

**Phase 3: Assign**
- Issues assigned to agents or developers
- Work proceeds incrementally
- Progress tracked via issues/PRs
- Human reviews each completion

### Example: Duplicate Code Detection

The [`duplicate-code-detector`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/duplicate-code-detector.md) uses ResearchPlanAssign:

**Research**: Uses Serena MCP for semantic analysis, creates report
**Plan**: Creates well-scoped issues (max 3 per run) with refactoring strategies
**Assign**: Pre-assigns to `@copilot` since fixes are straightforward

### Key Benefits

- Human control at decision points
- Prevents runaway automation
- Clear work breakdown
- Incremental progress
- Knowledge captured in issues

### Implementation Tips

- Use discussions for research phase
- Create issues for plan phase
- Track assignments explicitly
- Include acceptance criteria
- Review and iterate

**Learn more**: [ResearchPlanAssign Guide](https://githubnext.github.io/gh-aw/guides/researchplanassign/)

---

## Pattern 7: MultiRepoOps - Cross-Repository Coordination 🔗

### Overview

Workflows that coordinate operations across multiple GitHub repositories using cross-repository safe outputs and secure authentication. Enables feature synchronization, hub-and-spoke tracking, organization-wide enforcement, and upstream/downstream workflows.

### When to Use

- Organization-wide policies
- Dependency updates across repos
- Synchronized feature rollouts
- Security compliance enforcement
- Cross-repo health monitoring

### How It Works

1. Workflow runs in "hub" repository
2. Uses GitHub App or PAT for authentication
3. Queries multiple repositories
4. Analyzes cross-repo patterns
5. Uses safe outputs to create issues/PRs in target repos
6. Aggregates results back to hub

### Example: Org Health Report

The [`org-health-report`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/org-health-report.md) analyzes health metrics across all organization repositories:

- Checks for outdated dependencies
- Validates security policies
- Monitors CI health
- Creates issues in problematic repos
- Generates org-wide report

### Key Benefits

- Organization-wide visibility
- Consistent policy enforcement
- Centralized coordination
- Reduces duplication
- Scales to many repos

### Implementation Tips

- Use GitHub Apps for authentication
- Implement rate limiting
- Respect repository permissions
- Batch operations efficiently
- Monitor cross-repo dependencies

**Learn more**: [MultiRepoOps Guide](https://githubnext.github.io/gh-aw/guides/multirepoops/)

---

## Pattern 8: SideRepoOps - Isolated Automation Infrastructure 🏗️

### Overview

Run workflows from a separate "side" repository that targets your main codebase, keeping AI-generated issues, comments, and workflow runs isolated from production code. Provides an easy way to get started with agentic workflows without cluttering your main repository.

### When to Use

- Experimenting with agents
- High-volume workflow runs
- Sensitive or noisy operations
- Testing before production
- Organizational separation

### How It Works

1. Create separate "automation" repository
2. Install workflows in automation repo
3. Configure to target main repository
4. Workflows create issues/PRs in main repo
5. Discussions/artifacts stay in automation repo
6. Monitor both repos for results

### Example: Separate Analysis Repository

Main repo: `company/product` (production code)
Side repo: `company/product-automation` (workflows)

Workflows in `product-automation` analyze `product` codebase and create issues/PRs in `product` when appropriate, but keep noisy discussions in `product-automation`.

### Key Benefits

- Keeps main repo clean
- Easy to experiment
- Clear separation of concerns
- Can be more permissive in side repo
- Easy to disable all automation

### Implementation Tips

- Use GitHub Apps for cross-repo access
- Document the relationship clearly
- Consider visibility (public vs private)
- Set up appropriate notifications
- Plan for eventual migration if successful

**Learn more**: [SideRepoOps Guide](https://githubnext.github.io/gh-aw/guides/siderepoops/)

---

## Pattern 9: TrialOps - Safe Workflow Validation 🧪

### Overview

A specialized testing pattern that extends SideRepoOps for validating workflows in temporary trial repositories before production deployment. Creates isolated private repositories where workflows execute and capture safe outputs without affecting actual codebases.

### When to Use

- Testing new workflows
- Validating workflow changes
- Training and demonstrations
- Compliance verification
- Regression testing

### How It Works

1. Create temporary private repository
2. Install workflow under test
3. Populate with test data
4. Execute workflow
5. Capture and validate outputs
6. Delete trial repo or keep for reference

### Example: Workflow Testing Pipeline

Before deploying new triage agent:
1. Create `test-triage-agent` repo
2. Populate with sample issues
3. Run triage agent
4. Verify labeling accuracy
5. Adjust prompts as needed
6. Deploy to production

### Key Benefits

- Safe testing environment
- No production impact
- Repeatable validation
- Confidence before deployment
- Compliance documentation

### Implementation Tips

- Automate trial repo creation
- Use realistic test data
- Validate all safe outputs
- Document test scenarios
- Clean up trial repos regularly

**Learn more**: [TrialOps Guide](https://githubnext.github.io/gh-aw/guides/trialops/)

---

## Combining Operational Patterns

Many successful agent systems combine multiple operational patterns:

- **ChatOps + IssueOps**: User triggers analysis via `/analyze`, which creates issue with results
- **DailyOps + MultiRepoOps**: Daily dependency updates across organization
- **ResearchPlanAssign + ProjectOps**: Research creates project board populated with planned work
- **SideRepoOps + TrialOps**: Test in trial repo, then deploy to side repo, then main repo

## Choosing the Right Operational Pattern

When designing agent operations, consider:

1. **Trigger mechanism**: Manual (ChatOps), scheduled (DailyOps), or event-driven (IssueOps, LabelOps)?
2. **Scope**: Single repo or multi-repo (MultiRepoOps)?
3. **Isolation needs**: Production or separate (SideRepoOps, TrialOps)?
4. **Coordination**: Simple or complex (ProjectOps, ResearchPlanAssign)?
5. **State management**: Stateless or stateful (LabelOps, ProjectOps)?

## What's Next?

These operational patterns work effectively because they build on a foundation of reusable, composable components. The secret weapon that enabled Peli's Agent Factory to scale to 145 workflows wasn't just good patterns - it was the ability to share and reuse components across all those workflows.

In the next article, we'll explore the imports and sharing system that made this scalability possible.

[← Previous: Design Patterns](03-design-patterns.md) | [Back to Index](../index.md) | [Next: Imports & Sharing →](05-imports-and-sharing.md)
