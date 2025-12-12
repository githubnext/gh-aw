---
title: Campaigns
description: Enterprise-ready patterns for finite, accountable initiatives with governance, tracking, and reporting
---

A **campaign** is a finite initiative with explicit ownership, approval gates, budget constraints, and executive visibility. Campaigns extend regular workflows with enterprise requirements: accountability, governance, cross-team coordination, stakeholder reporting, and AI-assisted decision-making at scale.

Campaigns use standard primitives (tracker labels, repo-memory, safe-outputs, scheduled triggers) with no special infrastructure required. The key difference is organizational structure: clear ownership with executive sponsors, defined budgets with ROI tracking, approval gates with compliance audit trails, and business metrics for stakeholder reporting.

## Enterprise Value

Campaigns transform automation into sanctioned initiatives that fit enterprise processes. Regular workflows execute operations (triage issues, run tests); campaigns orchestrate business initiatives with measurable outcomes. The pattern: AI discovers and analyzes → humans decide and approve → AI executes with guardrails → humans validate → capture learnings for future campaigns.

## Regular Workflow vs Campaign

| Aspect | Regular Workflow | Campaign |
|--------|------------------|----------|
| **Duration** | One run or recurring forever | Finite (days to months) |
| **Goal** | Execute operation | Achieve business outcome |
| **Ownership** | Team/developer | Named owner + executive sponsor |
| **Approval** | Code review | Formal approval gate, change control |
| **Budget** | Unknown ongoing cost | Defined budget, ROI tracking |
| **Memory** | Stateless (logs only) | Stateful (repo-memory for audit) |
| **Tracking** | Individual run status | KPIs, executive dashboards |
| **Governance** | Standard CI/CD | Compliance audit trail, review gates |
| **Coordination** | Independent execution | Cross-team/repo coordination |

**Examples:**

*Regular workflow* - `daily-issue-triage.md`: Runs every day at 9 AM processing issues. Dev team owns it. No budget tracking or executive reporting.

*Campaign* - `campaign-security-q1-2025.md`: Launcher runs once, workers execute over 6 weeks to fix 200 vulnerabilities. Security lead owns it with VP Engineering sponsor. $8K approved budget. Weekly reporting to execs with progress, ETA, and ROI metrics. Baseline, daily metrics, and learnings stored in repo-memory for audit.

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

## When to Use Campaigns

Use campaigns for:

- **Cross-repo orchestration**: Rolling out changes across 50-200+ repositories with dependency-aware phased execution
- **Governance & compliance**: Approval chains, audit trails, compliance framework mapping (SOC2, GDPR, HIPAA)
- **Incident response**: Multi-team coordination under SLA pressure with risk-tiered decision gates
- **Human-in-loop at scale**: AI analyzes hundreds of items, generates risk-tiered recommendations, humans approve, AI executes
- **Organizational learning**: Cross-initiative intelligence and pattern recognition to improve future campaigns

## Available Campaign Patterns

See `.github/workflows/*.md` for complete examples:

- **incident-response.md**: Multi-team coordination under SLA pressure with risk-tiered recommendations and approval gates
- **org-wide-rollout.md**: Cross-repo changes with dependency-aware phased execution and rollback capability
- **security-compliance.md**: Security remediation with CISO approval and compliance audit trail
- **human-ai-collaboration.md**: AI analyzes at scale, generates risk-tiered recommendations, humans approve, AI executes
- **intelligence.md**: Cross-campaign learning with pattern recognition and predictive recommendations

## Recommended Default Wiring for Campaigns

For most enterprise campaigns, we recommend a consistent wiring pattern so
ownership, tracking, and reporting feel the same every time:

- **Tracker label**: A dedicated label like `campaign:<id>` declared in the
  campaign spec (`tracker-label`) and applied to all issues/PRs for that
  campaign.
- **Epic issue**: A single tracker issue (often labeled `campaign-tracker` +
  `campaign:<id>`) that serves as the human-readable command center.
- **Project board**: A GitHub Projects board per campaign (for example,
  `Code Health: File Diet`) used as the primary dashboard. Workers use the
  `update-project` safe-output to attach refactor tasks, update status
  fields, and keep the board in sync.
- **Repo-memory metrics**: Daily JSON snapshots under
  `memory/campaigns/<campaign-id>-*/metrics/*.json` following the
  `CampaignMetricsSnapshot` shape, so `gh aw campaign status` and
  intelligence workflows can compute velocity and ETAs.
- **Monitor workflow**: A lightweight monitor (often weekly) that aggregates
  metrics, generates trend charts (via `shared/trends.md`), and posts
  executive-ready updates to the epic with embedded screenshots.

This pattern turns each campaign into a first-class, auditable initiative:
issues carry the label, the epic and project board provide day-to-day
visibility, and repo-memory + charts power long-term reporting and
cross-campaign intelligence.

### Visualizing Campaign Trends

The `intelligence.md` workflow can turn your campaign metrics into **trend charts** using the shared Python visualization imports.

- The workflow imports `shared/trends.md`, which brings in a Python data viz environment and best practices for trend charts.
- As part of the analysis, it aggregates a flat metrics table across campaigns (date, campaign_id, type, velocity, success rate, ROI, cost per item) and writes it to `/tmp/gh-aw/python/data/campaign-metrics.csv`.
- Python code (generated by the agent using the `shared/trends.md` examples) loads this file and saves charts under `/tmp/gh-aw/python/charts/*.png`.
- The shared viz import automatically uploads these PNGs as workflow artifacts.

**Where to surface the charts**:

- Link to the artifacts from the **monthly intelligence issue** (created by `intelligence.md` via `safe-outputs.create-issue`).
- Embed 1–2 key charts in each campaign's final report under `memory/campaigns/.../final-report.md`.
- Optionally maintain a pinned "Campaign Intelligence" GitHub Discussion that links to monthly issues and includes the most important charts inline.

## First-Class Campaign Definitions (Spec & CLI)

In addition to the Markdown workflows under `.github/workflows/`, you can now declare
**first-class campaign definitions** as Markdown files with YAML frontmatter and inspect them via the CLI.

### Campaign Spec Files

Campaign spec files are stored alongside regular workflows in `.github/workflows/` with
a `.campaign.md` suffix. Each file describes a single campaign pattern, with a YAML frontmatter
block that defines the spec:

```yaml
# .github/workflows/incident-response.campaign.md
id: incident-response
version: "v1"
name: "Incident Response Campaign"
description: "Multi-team incident coordination with command center, SLA tracking, and post-mortem."

workflows:
  - incident-response

memory-paths:
  - "memory/campaigns/incident-*/**"

owners:
  - "oncall-incident-commander"
  - "sre-team"

executive-sponsors:
  - "vp-engineering"

risk-level: "high"
state: "planned"
tags:
  - "incident"
  - "operations"

tracker-label: "campaign:incident-response"

allowed-safe-outputs:
  - "create-issue"
  - "add-comment"
  - "create-pull-request"

approval-policy:
  required-approvals: 1
  required-roles:
    - "incident-commander"
  change-control: false
```

**Fields**:
- `id`: Stable identifier (defaults from filename when omitted)
- `version`: Optional spec version string (defaults to `v1` when omitted)
- `name`: Human-friendly name (falls back to `id`)
- `description`: Short description of the campaign pattern
- `workflows`: Workflow IDs (Markdown basenames) that implement this campaign
- `memory-paths`: Where campaign data is stored in repo-memory
- `metrics-glob`: Optional glob (relative to repo root) used to locate JSON metrics snapshots on the `memory/campaigns` branch
- `owners`: Primary human owners for this campaign
- `executive-sponsors`: Executive stakeholders accountable for the outcome
- `risk-level`: Optional free-form risk level (e.g. low/medium/high)
- `state`: Lifecycle state (`planned`, `active`, `paused`, `completed`, or `archived`)
- `tags`: Optional labels for reporting (e.g. `security`, `modernization`)
- `tracker-label`: Label used to associate issues/PRs with the campaign
- `allowed-safe-outputs`: Documented safe-outputs operations this campaign is expected to use
- `approval-policy`: High-level approval expectations (required approvals, roles, change control)

These specs do **not** replace workflows – they sit **on top** of them as a
single, declarative source of truth for how a campaign is defined.

### Inspecting Campaigns with the CLI

Use the `campaign` command to list and inspect configured campaigns:

```bash
gh aw campaign                     # List all campaigns from .github/workflows/*.campaign.md
gh aw campaign security            # Filter by ID or name substring
gh aw campaign --json              # JSON output for tooling or dashboards

# Show live status (compiled workflows + issues/PRs)
gh aw campaign status              # Status for all campaigns
gh aw campaign status incident     # Filter by ID or name substring
gh aw campaign status --json       # JSON status output

# Create and validate campaign specs
gh aw campaign new security-q1-2025         # Scaffold a new campaign spec
gh aw campaign validate                     # Validate all campaign specs
gh aw campaign validate --json              # JSON validation report
gh aw campaign validate --strict=false      # Report problems without failing
```

This gives you a centralized, Git-backed catalog of campaigns in the
repository, aligned with the executable workflows in `.github/workflows/` and
the data they write into repo-memory.

### Compile Integration and Orchestrators

Campaign spec files participate in the normal `compile` workflow:

- `gh aw compile` automatically validates all `.github/workflows/*.campaign.md` specs before compiling workflows.
- Validation checks both the spec itself (IDs, tracker labels, lifecycle state, etc.) and that all referenced `workflows` exist in `.github/workflows/`.
- If any campaign spec has problems, `compile` fails with a `campaign validation failed` error until issues are fixed.

**Orchestrator workflows** are automatically generated for each campaign spec:

```bash wrap
gh aw compile
```

When specs are valid:

- Each `<name>.campaign.md` generates an orchestrator markdown workflow named `<name>-campaign.md` next to the spec.
- The orchestrator is compiled like any other workflow to `<name>-campaign.lock.yml`.
- This makes campaigns first-class, compilable entry points while keeping specs declarative.
- Orchestrators are only generated when the campaign spec has meaningful details (tracker labels, workflows, memory paths, or metrics glob).

### Interactive Campaign Designer

For a more guided, conversational experience, this repo also includes a
reusable **custom agent**:

- `.github/agents/campaign-designer.agent.md` ("Campaign Designer for gh-aw Campaigns")

You can:

- Use `gh aw campaign new <id>` to scaffold a minimal spec, then
  open the **Campaign Designer** agent in GitHub Copilot Chat or your
  preferred agent UI to refine fields like `owners`, `memory-paths`, and
  `approval-policy`, or
- Start directly with the agent and let it propose a complete
  `.campaign.md` spec (and optional starter workflow) based on your answers.

### Example: Incident Response Campaign

**Full workflow**: `.github/workflows/campaign-incident-response.md`

**Scenario**: Production API experiencing failures across multiple services

**How it works**:

1. **Command Center** (repo-memory)
   - Initialize incident metadata: severity, affected services, SLA target
   - Create timeline for audit trail

2. **AI Analysis**
   - Search recent changes, errors, related issues
   - Generate hypotheses ranked by probability
   - Identify teams to involve
   - Estimate blast radius

3. **Risk-Tiered Recommendations**
   - **Low risk** (e.g., rollback deployment): "Safe to execute immediately"
   - **Medium risk** (e.g., apply hotfix): "Needs team lead approval"
   - **High risk** (e.g., database rollback): "Needs executive approval"

4. **Human Decision Point**
   - Incident commander reviews recommendations
   - Approves actions by risk tier
   - Can defer high-risk actions for investigation

5. **Execution**
   - AI creates PRs for approved fixes
   - Tracks status in command center
   - Updates SLA countdown

6. **Communication**
   - Status updates every 30 minutes to command center
   - Stakeholder updates with sanitized info
   - Timeline continuously logged

7. **Resolution**
   - Generate post-mortem template from timeline
   - Document what worked/didn't work
   - Create action items with ownership

**The pattern**: Centralized coordination + AI intelligence + human judgment + audit trail for high-pressure situations requiring multiple teams.

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

**Campaign ID Format**: `<type>-<identifier>` (e.g., `security-q1-2025`, `modernization-winter2025`)

Apply as label `campaign:<id>` to link all related issues, PRs, and memory data. Use `campaign-tracker` label for epic issues.

**Query campaign work**:
```bash
gh issue list --label "campaign:security-q1-2025"  # All campaign issues
gh pr list --label "campaign:security-q1-2025"     # All campaign PRs
gh issue list --label "campaign-tracker" --state open  # Active campaigns
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

**Planning**: Define measurable goals and completion criteria. Identify stakeholders (owner, exec sponsor, approvers). Establish budget with ROI target. Get change control and budget approval. Estimate duration based on similar past campaigns from repo-memory learnings.

**Execution**: Store baseline in repo-memory for audit trail. Use consistent `campaign:<id>` labeling. Update epic regularly with progress. Handle failures gracefully by logging errors and escalating blockers. Track costs for budget compliance.

**Monitoring**: Run monitor daily to catch stalled work. Track velocity for realistic ETAs. Identify blockers stuck >7 days for escalation. Alert stakeholders when campaigns are at risk.

**Completion**: Generate final report with metrics, learnings, and ROI in repo-memory. Calculate ROI (campaign cost vs manual effort saved vs business value). Document what worked and didn't work to improve future campaigns. Preserve audit trail for compliance.

**Learning**: Compare campaigns over time to measure improvement. Build playbooks based on learnings. Share successful patterns as organizational templates. Maintain centralized knowledge base of campaign learnings.

## Decision Guide

**Use campaigns** for finite business goals requiring executive approval, budget tracking, cross-team coordination, stakeholder reporting, governance/compliance, or organizational learning.

**Use regular workflows** for operational automation, single-team scope, developer-only audience, standard CI/CD, or simple reporting.

**Examples**: Fix 200 security vulns before audit (campaign). Update 100 repos org-wide (campaign). Migrate services to new platform (campaign). Daily issue triage (workflow). Run tests on PRs (workflow).

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
