---
title: Campaigns
description: Enterprise-ready patterns for finite, accountable initiatives with governance, tracking, and reporting
---

A **campaign** is a finite initiative with explicit ownership, approval gates, budget constraints, and executive visibility. Campaigns solve organizational challenges that regular workflows don't address: accountability, governance, cross-team coordination, and stakeholder reporting.

**The AI-Human Collaboration Model**: Campaigns aren't about full automation - they're about **AI-assisted decision-making at scale**. AI discovers and analyzes, humans decide and validate, AI executes with guardrails, everyone learns from outcomes.

## Why Campaigns Matter for Enterprises

**Technical Reality**: Campaigns use the same primitives as regular workflows (tracker-id via labels, repo-memory, safe-outputs, scheduled triggers). No special infrastructure required.

**Organizational Value**: Campaigns provide the structure enterprises need for:

### 1. **Accountability & Ownership**
- **Regular workflow**: "Who's responsible for this automation?"
- **Campaign**: "Sarah owns Q1 Security Campaign" - clear ownership, executive sponsor, RACI model

### 2. **Approval & Governance**
- **Regular workflow**: Runs indefinitely, unclear approval status
- **Campaign**: Explicit start/end, approval gate before launch, compliance audit trail, review checkpoints

### 3. **Budget & Resource Allocation**
- **Regular workflow**: Unknown ongoing cost
- **Campaign**: "Q1 campaign budget: $5K AI cost, 200 hours" - finite budget, ROI calculation, cost tracking

### 4. **Executive Reporting**
- **Regular workflow**: "The bot processed 47 issues today"
- **Campaign**: "Q1 Security: 197/200 vulns fixed (98.5%), 2 weeks ahead of schedule, $12K saved vs manual" - KPIs, dashboards, business impact

### 5. **Cross-Team Coordination**
- **Regular workflow**: Each team's isolated automation
- **Campaign**: "Org-wide modernization across 50 repos, 10 teams" - centralized tracking, dependencies, coordination

### 6. **Risk Management**
- **Regular workflow**: Continues running regardless of business context changes
- **Campaign**: Completion criteria, periodic review, can pause/stop/pivot, change control

### 7. **Change Management**
- **Regular workflow**: "The system does stuff automatically" (shadow IT)
- **Campaign**: "Launching Q1 modernization campaign next week" - stakeholder communication, expectation management, rollout planning

### 8. **AI-Human Collaboration**
- **Regular workflow**: Fully automated or fully manual
- **Campaign**: AI discovers & proposes → Humans decide & approve → AI executes with guardrails → Humans validate → AI learns from outcomes

**The Enterprise Reality**: Without campaigns, you get shadow automation that executives don't understand or trust. With campaigns, you get sanctioned, tracked, reportable initiatives that fit enterprise processes for budgeting, approval, governance, and compliance.

**The AI Value**: Campaigns leverage AI for intelligence (what needs fixing?) and execution (safe automated changes), while keeping humans in control for judgment (what should we fix?) and validation (did it work?).

## What Makes a Campaign Different

**Campaigns are characterized by**:
- **Finite scope**: Specific goal with completion criteria ("fix 200 security issues in Q1")
- **Clear ownership**: Named owner, executive sponsor, approval chain
- **Budget constraints**: Defined cost limits, ROI tracking
- **AI-human collaboration**: AI discovers/proposes, humans decide/validate, AI executes with guardrails
- **Persistent memory**: Progress in repo-memory for audit trail and learning
- **Coordinated work**: Multiple tasks/teams linked via campaign ID
- **Executive reporting**: KPIs, dashboards, business impact metrics
- **Governance**: Approval gates, review checkpoints, change control
- **Continuous learning**: Capture outcomes to improve future campaigns

**Regular workflows** execute operations (triage issues, run tests, deploy code). **Campaigns** orchestrate business initiatives with the accountability, tracking, and AI-assisted decision-making that enterprises require.

### Regular Workflow vs Campaign

| Aspect | Regular Workflow | Campaign |
|--------|-----------------|----------|
| **Duration** | One run or recurring forever | Finite (days to months) with defined end date |
| **Goal** | Execute operation | Achieve business outcome with measurable impact |
| **Ownership** | Team/developer owns automation | Named owner + executive sponsor |
| **Approval** | Code review | Formal approval gate, change control board |
| **Budget** | Unknown ongoing cost | Defined budget, cost tracking, ROI calculation |
| **Memory** | Stateless (logs only) | Stateful (repo-memory for audit trail) |
| **Tracking** | Individual run status | Aggregated progress, KPIs, executive dashboards |
| **Reporting** | Run logs for developers | Business impact reports for stakeholders |
| **Governance** | Standard CI/CD process | Compliance requirements, audit trail, review gates |
| **Coordination** | Independent execution | Cross-team/repo coordination, dependencies |
| **Completion** | N/A (operational) | Clear end state, success criteria, retrospective |

**Example - Regular Workflow:**
```yaml
daily-issue-triage.md
- Runs: Every day at 9 AM, forever
- Goal: Process today's issues (ongoing operations)
- Owner: Dev team
- Budget: Unknown ongoing cost
- Memory: None (each run independent)
- Tracking: Workflow run logs
- Reporting: Developer-facing only
```

**Example - Campaign:**
```yaml
campaign-security-q1-2025.md
- Runs: Launcher once, then workers over 6 weeks
- Goal: Fix 200 critical vulnerabilities (business requirement)
- Owner: Security lead + VP Engineering sponsor
- Budget: $8K AI cost, approved by finance
- Approval: Security review board approved
- Memory: Baseline, daily metrics, learnings for audit
- Tracking: Epic issue + daily reports + executive dashboard
- Reporting: Weekly to exec team: progress, ETA, blockers, ROI
- Completion: When 200 vulns fixed → final report, retrospective
- Governance: Change control process, compliance documentation
```

## Enterprise Campaign Patterns

### Multi-Repository Campaign

Update dependencies across multiple repositories:

```aw wrap
---
name: Org-Wide Dependency Update
on: workflow_dispatch

tools:
  github:
    toolsets: [repos, search]
  repo-memory:
    branch: memory/campaigns
---

Campaign ID: `dep-update-q1-2025`

1. **Discover repos**: Find all repos using vulnerable package X
2. **Create epic** in central campaign repo
3. **Create issues**: One per repo, with repo-specific details
4. **Store baseline**: Repos affected, versions found
5. **Workers** (in each repo): Create PR with update
6. **Monitor**: Track progress across all repos
```

**Key patterns**:
- Central campaign repo for coordination
- Issues link to target repos
- Workers deployed in each target repo
- Cross-repo progress tracking

### Staged Rollout Campaign

Deploy changes in waves with validation:

```yaml
Wave 1: Low-risk repos (5 repos) → validate
Wave 2: Medium complexity (15 repos) → validate  
Wave 3: Critical services (30 repos) → validate
```

Each wave:
1. Campaign creates wave-specific issues
2. Workers process wave issues
3. Monitor validates wave success
4. Campaign proceeds to next wave or halts

### Compliance Campaign

Ensure all repos meet policy requirements:

```
1. Audit: Scan all repos for compliance
2. Classify: Critical vs non-compliant
3. Remediate: Auto-fix where possible
4. Report: Executive dashboard with compliance %
5. Learning: Document common issues for future prevention
```

## Campaign Triggers

**Manual Launch** (recommended for most campaigns):
```yaml
on: workflow_dispatch
  inputs:
    campaign_goal:
      description: "What should this campaign achieve?"
```

**Threshold-Triggered** (automated campaigns):
```yaml
on: schedule
  - cron: '0 9 * * 1'  # Weekly check

# If >50 stale issues → auto-launch cleanup campaign
# If security scan finds >10 critical vulns → auto-launch security campaign
```

**Event-Triggered**:
```yaml
on:
  repository_dispatch:
    types: [security-alert]

# External security scanner triggers campaign
```

## What Campaigns Solve That GitHub Actions Cannot

**GitHub Actions** provides automation within a single repo. **Agentic Workflows** adds AI-powered analysis and execution. **Campaigns** add the organizational layer enterprises need.

### Problems Only Campaigns Solve:

1. **Cross-Repo Coordination at Scale**
   - Problem: "Update 200 repos with new security policy"
   - GitHub Actions: Each repo runs independently, no orchestration
   - Campaign: Central command + dependency-aware phased rollout + progress tracking

2. **Governance + Accountability + Audit Trail**
   - Problem: "Who approved this? Show me the compliance audit trail."
   - GitHub Actions: Just run logs, no business context
   - Campaign: Named owner + approval chain + persistent memory with business justification

3. **Human-in-Loop at Enterprise Scale**
   - Problem: "AI analyze 500 issues, humans decide which 50 to fix"
   - GitHub Actions: Fully automated or fully manual
   - Campaign: AI analyzes → humans review tiered recommendations → AI executes approved actions

4. **ROI Tracking + Budget Constraints**
   - Problem: "What's the ROI on our automation spend?"
   - GitHub Actions: No cost or outcome tracking
   - Campaign: Cost tracking + outcome measurement + executive reporting

5. **Incident Response Coordination**
   - Problem: "Production down! Coordinate 5 teams, track 30 repos, SLA pressure, stakeholder updates"
   - GitHub Actions: No cross-team coordination
   - Campaign: Command center + status tracking + approval gates + post-mortem

6. **Learning Across Initiatives**
   - Problem: "We've run 20 security audits. Are we improving? What works?"
   - GitHub Actions: Each run independent
   - Campaign: Cross-campaign intelligence + pattern recognition + recommendations

## Available Campaign Examples

The repository includes five campaigns demonstrating problems that **cannot be solved** with GitHub Actions or basic workflows:

### 1. **campaign-incident-response.md**
**Solves**: Multi-team incident coordination under SLA pressure  
**Key features**: Command center, risk-tiered approvals, stakeholder updates, post-mortem generation  
**Why campaigns**: GitHub Actions can't coordinate teams, track SLAs, or manage approval gates

### 2. **campaign-org-wide-rollout.md**
**Solves**: Changes across 100+ repos with dependency awareness  
**Key features**: Dependency graph, phased batches, approval between batches, rollback capability  
**Why campaigns**: GitHub Actions can't orchestrate cross-repo or respect dependencies

### 3. **campaign-security-compliance.md**
**Solves**: Compliance remediation with governance and audit trail  
**Key features**: CISO approval, compliance mapping, executive reporting, audit documentation  
**Why campaigns**: GitHub Actions has no governance model or compliance audit trail

### 4. **campaign-human-ai-collaboration.md** (PATTERN)
**Solves**: AI analysis at scale with human decision-making  
**Key features**: AI proposes 3 risk tiers, humans approve by tier, AI executes, humans validate  
**Why campaigns**: Shows the core AI-human collaboration model

### 5. **campaign-intelligence.md** (PATTERN)
**Solves**: Learning across campaigns to improve future initiatives  
**Key features**: Cross-campaign analysis, trend detection, predictive recommendations  
**Why campaigns**: GitHub Actions doesn't learn from history

Each campaign solves a real organizational problem that GitHub Actions and basic workflows fundamentally cannot address.

## Campaign Examples

See the actual campaign workflows in `.github/workflows/campaign-*.md` for complete, runnable examples:

- **campaign-incident-response.md** - Multi-team incident coordination
- **campaign-org-wide-rollout.md** - Cross-repo changes with phased execution
- **campaign-security-compliance.md** - Compliance with governance
- **campaign-human-ai-collaboration.md** - AI-assisted decision-making pattern
- **campaign-intelligence.md** - Cross-campaign learning

### Example: Incident Response Campaign

**Full workflow**: `.github/workflows/campaign-incident-response.md`

**Scenario**: Production API down, affecting multiple services

**Campaign steps**:

1. **Initialize Command Center** (repo-memory)
   - Store incident metadata, SLA targets, affected services
   - Create timeline for audit trail

2. **AI Analysis Phase**
   - Search recent changes, errors, dependencies
   - Generate hypotheses ranked by probability
   - Identify teams to involve

3. **Human Decision Checkpoint**
   - AI presents risk-tiered recommendations:
     - **Low risk** (rollback deployment): Execute immediately
     - **Medium risk** (apply hotfix PR): Needs team lead approval
     - **High risk** (database rollback): Needs executive approval

4. **Execute Approved Actions**
   - Create PRs for fixes
   - Track execution status
   - Update SLA countdown

5. **Status Updates Every 30min**
   - Command center issue updated
   - Stakeholder issue updated
   - Timeline events logged

6. **Resolution & Post-Mortem**
   - Generate post-mortem template from timeline
   - Document what worked/didn't
   - Action items with ownership

**Why this needs campaigns**: GitHub Actions can't coordinate teams, track SLAs, manage approval gates, or generate post-mortems with business context.

### Example 2: Security Campaign with Workers

Multi-workflow campaign for long-running initiative:

**Launcher** (`campaign-security-audit.md`):
```aw wrap
---
name: Security Audit Campaign Q1 2025
on: workflow_dispatch

safe-outputs:
  create-issue: { max: 200 }

tools:
  github:
    toolsets: [repos, search]
  repo-memory:
    branch: memory/campaigns
    patterns:
      - "campaigns/security-q1-2025/**"
---

# Security Audit Q1 2025

Campaign ID: `security-q1-2025`

1. **Scan codebase** for vulnerabilities
2. **Store baseline**: Total vulns found, severity breakdown
3. **Create epic** with campaign goals
4. **Generate task issues**:
   - One issue per vulnerability
   - Labels: `security`, `campaign:security-q1-2025`, `severity:high|medium|low`
   - Issue body: Vulnerability details, affected files, fix guidance
```

**Worker** (`campaign-security-worker.md`):
```aw wrap
---
name: Security Fix Worker
on:
  issues:
    types: [opened, labeled]

permissions:
  contents: read
  issues: read
  pull-requests: read

engine: copilot
safe-outputs:
  create-pull-request: { }
  add-comment: { max: 5 }

tools:
  repo-memory:
    branch: memory/campaigns
---

# Security Fix Worker

Only process issues with:
- Label: `campaign:security-q1-2025`
- Label: `security`

## Tasks

1. **Read vulnerability issue**
2. **Create fix PR**:
   - Update affected files
   - Add tests
   - Link to vulnerability issue
3. **Comment on issue**: PR created, awaiting review
4. **Log to memory**: Fix attempt, time taken, success/failure
```

**Monitor** (`campaign-monitor.md`):
```aw wrap
---
name: Campaign Monitor  
on:
  schedule:
    - cron: '0 18 * * *'  # Daily 6 PM

safe-outputs:
  add-comment: { max: 10 }

tools:
  github:
    toolsets: [repos, issues]
  repo-memory:
    branch: memory/campaigns
---

For each active campaign (campaign-tracker label, open):

1. **Query campaign issues**: Count total, completed, blocked
2. **Calculate metrics**: Velocity, ETA, health status
3. **Store daily snapshot** in repo-memory
4. **Post report** to epic issue:
   - Progress: X/Y complete (Z%)
   - Velocity: N per day
   - ETA: Date
   - Blockers: List stalled issues
5. **Alert if stalled**: No progress in 7 days → add needs-attention label
```

**Tracking**: Epic + daily reports + repo-memory trends + worker PRs
**Duration**: 6 weeks (automated execution)
**Best for**: Long-running initiatives, many tasks, needs tracking

### Campaign Memory Structure

Campaigns store persistent data in repo-memory:

```
memory/campaigns/
├── security-q1-2025/
│   ├── baseline.json           # Initial state
│   │   {
│   │     "campaign_id": "security-q1-2025",
│   │     "started": "2025-01-15",
│   │     "vulnerabilities": 200,
│   │     "repos_affected": 50
│   │   }
│   ├── metrics/
│   │   ├── 2025-01-16.json    # Daily snapshots
│   │   ├── 2025-01-17.json
│   │   └── ...
│   ├── learnings.md            # What worked/didn't
│   └── final-report.md         # Completion summary
└── issue-cleanup-2025q2/
    └── ...
```

**Baseline** (`baseline.json`):
- Initial metrics before campaign starts
- Enables before/after comparison
- Used to calculate success rate

**Daily Metrics** (`metrics/YYYY-MM-DD.json`):
```json
{
  "date": "2025-01-16",
  "campaign_id": "security-q1-2025",
  "tasks_total": 200,
  "tasks_completed": 15,
  "tasks_in_progress": 30,
  "tasks_blocked": 5,
  "velocity_per_day": 7.5,
  "estimated_completion": "2025-02-12"
}
```

**Learnings** (`learnings.md`):
```markdown
# Campaign Learnings: Security Q1 2025

## What Worked
- Automated dependency updates saved 50+ hours
- Breaking changes detected early via tests

## What Didn't Work  
- Package X always broke builds, needed manual review
- Team bandwidth bottleneck in week 3

## Blockers Encountered
- CI/CD capacity limits (max 10 concurrent builds)
- Required security team approval took 2-3 days avg

## Recommendations for Next Campaign
- Pre-approve common dependency updates
- Provision more CI capacity
- Stagger rollout to avoid bottlenecks
```

**Final Report** (`final-report.md`):
```markdown
# Security Q1 2025 Campaign - Final Report

**Duration**: Jan 15 - Feb 12 (28 days)
**Goal**: Fix 200 vulnerabilities across 50 repos
**Result**: 197 fixed, 3 accepted risk

## Metrics
- Success Rate: 98.5%
- Avg Time per Fix: 2.4 hours
- Total Cost: $X (AI + runner time)

## Impact
- Reduced critical vulns from 200 to 3
- All repos now compliant with security policy

## ROI
- Manual effort saved: 480 hours
- Cost of campaign: $X
- ROI: 10x
```

### Campaign IDs and Labels

### Campaign IDs and Labels

**Campaign ID Format**: `<type>-<identifier>`
- Examples: `security-q1-2025`, `issue-cleanup-12345`, `modernization-winter2025`
- Applied as label: `campaign:security-q1-2025`
- Links all related issues, PRs, and memory data

**How Campaigns Use Labels**:
```yaml
# Launcher creates epic
create-issue:
  title: "Campaign: Security Q1 2025"
  labels: ["campaign-tracker", "epic", "campaign:security-q1-2025"]

# Launcher creates task issues  
create-issue:
  title: "Fix vulnerability in auth module"
  labels: ["security", "campaign:security-q1-2025"]

# Workers filter by campaign label
if issue has labels: ["campaign:security-q1-2025", "type:vulnerability"]
  → process this issue
```

**Query Campaign Work**:
```bash
# All campaign issues
gh issue list --label "campaign:security-q1-2025"

# All campaign PRs
gh pr list --label "campaign:security-q1-2025"

# Find campaign epic
gh issue list --label "campaign-tracker" --label "campaign:security-q1-2025"

# Track active campaigns
gh issue list --label "campaign-tracker" --state open
```

## How Campaigns Work

### Tracking Options

**Epic Issue** (recommended):
- Single issue with task checklist
- Visible in repo
- Team can comment/discuss
- Monitor posts daily updates
- Best for: Most campaigns

**Project Board** (complex campaigns):
- Visual dashboard with columns
- Custom fields (status, priority, etc.)
- Better for: Large initiatives with many stakeholders

**Labels Only** (minimal):
- Just campaign labels, no epic
- Query via `gh issue list`
- Best for: Simple campaigns, automated only

**Repo-Memory** (required for true campaigns):
- Always use for persistence
- Even if you use epic issues or boards
- Enables reporting, trends, learning

### Safe Outputs for Campaigns

Campaigns use these safe-outputs:

```yaml
safe-outputs:
  create-issue: { max: 100 }    # Task issues + epic
  add-comment: { max: 50 }      # Epic updates, worker reports
  create-pull-request: { }       # Workers create fixes
  close-issue: { max: 100 }     # Workers complete tasks
  update-project: { max: 100 }  # Optional: project board sync
```

### Repository Memory for Campaigns

```yaml
tools:
  repo-memory:
    branch: memory/campaigns
    patterns:
      - "campaigns/*/baseline.json"
      - "campaigns/*/metrics/*.json"
      - "campaigns/*/learnings.md"
      - "campaigns/*/final-report.md"
```

See [Cache & Memory](/gh-aw/reference/memory/) for repo-memory details.

## Campaign Architecture

### Core Components

Every campaign consists of:

**1. Launcher Workflow** (required)
- Analyzes context and identifies work needed
- Creates epic tracking issue with goals and KPIs
- Generates task issues with campaign labels
- Initializes baseline in repo-memory
- One-time execution

**2. Worker Workflows** (optional, for long-running campaigns)
- Trigger on campaign-labeled issues
- Execute individual tasks (create PRs, update code, etc.)
- Update epic issue with progress
- Can run in parallel
- Multiple specialized workers for different task types

**3. Monitor Workflow** (recommended for multi-day campaigns)
- Runs on schedule (daily/weekly)
- Tracks metrics: completion rate, velocity, blockers
- Posts progress reports to epic issue
- Alerts on stalled campaigns
- Stores trend data in repo-memory
- One monitor can track ALL campaigns

**4. Repo-Memory** (required for true campaigns)
- Persistent storage on git branch
- Baseline data: initial state before campaign
- Progress data: daily/weekly metrics snapshots
- Learnings: what worked, what didn't, blockers encountered
- Enables reporting and future campaign improvement

### Campaign Patterns

#### Pattern 1: Simple Campaign (No Workers)

**When to use**: Campaign completes in one run (<30 min), no ongoing tracking needed

**Structure**:
```
campaign-issue-cleanup.md       # Single workflow does everything
```

**Workflow**:
```yaml
---
name: Issue Cleanup Campaign
on: workflow_dispatch
safe-outputs:
  close-issue: { max: 30 }
  create-issue: { max: 1 }      # Epic for tracking
tools:
  repo-memory:
    branch: memory/campaigns
---

1. Create epic issue with goals
2. Find 30 stale issues
3. Close each with explanation
4. Store results in repo-memory
5. Report completion
```

**Tracking**: Epic issue created, results in memory
**Best for**: Quick cleanup tasks, batch operations

#### Pattern 2: Monitored Campaign (No Workers)

**When to use**: Work happens outside workflows (human execution), need progress tracking

**Structure**:
```
campaign-code-review-sprint.md  # Launcher creates tasks
campaign-monitor.md             # Universal monitor tracks progress
```

**Launcher**:
```yaml
---
name: Code Review Sprint Campaign  
safe-outputs:
  create-issue: { max: 20 }     # 20 code review tasks
tools:
  repo-memory:
    branch: memory/campaigns
    patterns:
      - "campaigns/code-review-*/baseline.json"
      - "campaigns/code-review-*/metrics/*.json"
---

1. Analyze codebase for review needs
2. Create epic with campaign goals
3. Generate 20 code review task issues
4. Store baseline in repo-memory
```

**Monitor** (runs daily):
```yaml
---
name: Campaign Monitor
on:
  schedule:
    - cron: '0 18 * * *'
safe-outputs:
  add-comment: { max: 10 }
---

For each campaign (via campaign-tracker label):
1. Calculate completion %
2. Compute velocity
3. Identify blockers
4. Post daily report to epic
5. Update metrics in repo-memory
```

**Tracking**: Epic + daily reports + repo-memory trends
**Best for**: Human-executed tasks, multi-day initiatives

#### Pattern 3: Worker Campaign (Multi-Workflow)

**When to use**: Work takes long time, needs automation, multiple task types, timeout concerns

**Structure**:
```
campaign-modernization.md                   # Launcher
campaign-modernization-dependency-worker.md # Worker for dependencies
campaign-modernization-docs-worker.md       # Worker for documentation
campaign-modernization-test-worker.md       # Worker for tests
campaign-monitor.md                         # Universal monitor
```

**Launcher**:
```yaml
---
name: Modernization Campaign
safe-outputs:
  create-issue: { max: 100 }
tools:
  repo-memory:
    branch: memory/campaigns
---

1. Create epic
2. Analyze codebase
3. Generate issues by type:
   - type:dependency → 30 dep update tasks
   - type:documentation → 25 doc tasks  
   - type:test → 45 test tasks
4. Store baseline in repo-memory
```

**Workers** (triggered by issue creation):
```yaml
---
name: Dependency Worker
on:
  issues:
    types: [opened, labeled]
safe-outputs:
  create-pull-request: { }
---

If issue has type:dependency + campaign:mod-*:
1. Read dependency issue
2. Create PR with update
3. Update epic with progress
4. Update worker metrics in memory
```

**Why workers?**: 100 tasks × 20 min each = 2000 min (33 hours) - impossible in single run!

**Tracking**: Epic + worker PRs + daily monitor + repo-memory
**Best for**: Large-scale initiatives, cross-repo updates, enterprise campaigns

### Campaign Memory Structure

Campaigns store persistent data in repo-memory:

## Best Practices

### Planning

- **Define measurable goals**: "Fix 200 vulns by March 31" not "improve security"
- **Set completion criteria**: Clear definition of done, success metrics
- **Identify stakeholders**: Owner, exec sponsor, approvers, teams involved
- **Establish budget**: AI costs, runner time, engineering hours, ROI target
- **Get approval**: Change control board, security review, budget approval
- **Estimate duration**: Based on similar past campaigns (use repo-memory learnings)
- **Define governance**: Review checkpoints, escalation paths, pause criteria

### Execution

- **Start with baseline**: Store initial state in repo-memory for audit trail
- **Use consistent labeling**: `campaign:<id>` on all issues/PRs for tracking
- **Update epic regularly**: Workers comment progress, monitor posts daily reports
- **Handle failures gracefully**: Log errors, document blockers, escalate when needed
- **Preserve context**: Store decisions, rationale, changes in repo-memory
- **Track costs**: Monitor AI usage, runner time for budget compliance
- **Communicate status**: Regular updates to stakeholders on progress/blockers

### Monitoring

- **Daily check-ins**: Monitor runs daily to catch stalled work early
- **Track velocity**: Tasks per day → realistic ETA for stakeholders
- **Identify blockers**: What's stuck >7 days → needs escalation
- **Alert stakeholders**: Automated notifications when campaigns at risk
- **Budget tracking**: Compare actual vs budgeted costs, forecast overruns
- **Adjust as needed**: Campaigns can pause/pivot based on learnings

### Completion

- **Generate final report**: Metrics, learnings, ROI in repo-memory for audit
- **Calculate ROI**: Cost of campaign vs manual effort saved vs business value
- **Archive campaign**: Close epic, mark complete in systems
- **Document learnings**: What worked, what didn't → improve next campaigns
- **Preserve audit trail**: Keep all issues/PRs/memory for compliance
- **Executive summary**: Business impact, outcomes achieved, lessons learned
- **Celebrate wins**: Share success with team/stakeholders, recognize contributors

### Learning & Improvement

- **Compare campaigns**: Q1 vs Q2 security → getting faster? More efficient?
- **Build playbooks**: "Security campaign playbook" based on learnings
- **Share patterns**: Successful campaign structures → templates for org
- **Measure ROI trends**: Are campaigns becoming more cost-effective over time?
- **Iterate governance**: Refine approval/review processes based on experience
- **Knowledge base**: Centralized repository of campaign learnings for organization

## Decision Guide: When to Use Campaigns

### Use a Campaign When:

✅ **Enterprise requirements**: Need approval, budget tracking, executive reporting, compliance
✅ **Clear ownership needed**: Named owner and executive sponsor required
✅ **Cross-team coordination**: Multiple teams/repos must work together
✅ **Stakeholder visibility**: Non-technical stakeholders need progress updates
✅ **Finite business goal**: Specific outcome with measurable success ("fix 200 vulns")
✅ **Budget constraints**: Need to track costs, calculate ROI, justify spending
✅ **Governance required**: Audit trail, compliance, change control processes
✅ **Learning desired**: Capture what worked/didn't for future similar initiatives

### Use Regular Workflow When:

❌ **Operational automation**: Ongoing tasks with no defined end state
❌ **Single team scope**: No cross-team coordination needed
❌ **Developer-only audience**: No need for executive/stakeholder visibility
❌ **No budget tracking**: Cost tracking not required
❌ **Standard CI/CD**: Fits normal development process, no special approval
❌ **Simple reporting**: Run logs sufficient, no business metrics needed

### Examples:

| Scenario | Use Campaign? | Why |
|----------|--------------|-----|
| Fix 200 security vulns in Q1 | ✅ Yes | Executive mandate, budget approval, compliance requirement, cross-team effort |
| Daily issue triage | ❌ No | Ongoing ops, no end state, dev team only |
| Update deps in 100 repos (org-wide) | ✅ Yes | Enterprise scale, exec sponsor, budget, coordination, stakeholder reporting |
| Format code in one PR | ❌ No | Single task, quick, no tracking/governance needed |
| SOC2 compliance remediation | ✅ Yes | Audit requirement, exec visibility, compliance documentation, budget |
| Test new GitHub feature | ❌ No | Developer experiment, no business impact |
| Migrate 30 services to new platform | ✅ Yes | Strategic initiative, exec sponsor, budget, multi-team, phased rollout |
| Run tests on every PR | ❌ No | Standard CI/CD, ongoing, no special governance |

**Key insight**: If you need to explain it to executives, get budget approval, or track business ROI → it's a campaign. If it's normal developer automation → it's a regular workflow.

## Quick Start

**Step 1**: Create launcher workflow `.github/workflows/campaign-my-initiative.md`

**Step 2**: Add campaign essentials:
```yaml
safe-outputs:
  create-issue: { max: 100 }    # Epic + task issues
tools:
  repo-memory:
    branch: memory/campaigns
```

**Step 3**: Write campaign logic:
```
1. Store baseline in repo-memory
2. Create epic issue (campaign-tracker label)
3. Generate task issues (campaign:<id> labels)
4. Report summary
```

**Step 4**: (Optional) Add monitor for multi-day tracking

**Step 5**: Run campaign, track via epic issue and repo-memory

## Related Patterns

- **[ResearchPlanAssign](/gh-aw/guides/researchplanassign/)** - Research → generate coordinated work
- **[ProjectOps](/gh-aw/examples/issue-pr-events/projectops/)** - Project board integration for campaigns
- **[MultiRepoOps](/gh-aw/guides/multirepoops/)** - Cross-repository operations
- **[Cache & Memory](/gh-aw/reference/memory/)** - Persistent storage for campaign data
- **[Safe Outputs](/gh-aw/reference/safe-outputs/)** - `create-issue`, `add-comment` for campaigns
