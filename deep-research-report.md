# How AI Agents Rewire Project Boards: Human Roles, Control, and an Adoption Blueprint

## Executive summary

GitHub project boards today are primarily **human-operated coordination surfaces**: people decide what gets tracked, keep fields up to date, interpret status, and translate board state into stakeholder narratives. GitHub provides flexible views (table/board/roadmap), built‑in automations, and charting, but the system still relies heavily on ongoing human attention—especially when work spans repositories, involves dependencies and schedule shifts, or requires richer portfolio reporting. citeturn0search15turn13search6turn0search19turn13search0

When AI agents are embedded through agentic workflows (notably GitHub Agentic Workflows, “gh-aw”), the project board’s role shifts from “a place humans curate” into a **continuously maintained operational model**: agents infer intent from issues/PRs, reason across related items, propose or apply field and schedule updates, and publish portfolio-grade summaries—while humans move “up the stack” into **policy, approvals, exception handling, and audit**. citeturn5view0turn15view0turn12view0turn12view1turn26search12

Control does not have to be traded away for automation. gh-aw is explicitly designed around **least privilege**, **scoped write capabilities (“safe outputs”)**, **sanitisation**, **threat detection**, and optional modes like **lockdown** and **staged (preview) execution** that keep humans in charge of consequential actions. citeturn11view1turn5view0turn10view2turn18search1turn20view0

## Baseline: human-centric workflows and pain points in GitHub Projects today

GitHub Projects (Projects v2) is defined as an adaptable collection of items that can be viewed as a **table**, **kanban board**, or **roadmap**, and stays up to date with GitHub data; it commonly tracks issues, pull requests, and draft items. citeturn0search15turn13search8

Built-in workflows can automate parts of a human workflow—for example updating Status when items are added/closed, or closing issues when a project status changes—plus auto-add and auto-archive workflows based on filters. citeturn26search3turn13search1turn13search0turn13search6

Projects also support charting (“Insights”), including **current** and **historical** charts configured from project data; however, Insights does not track archived or deleted items, which matters when teams use archiving to keep boards usable at scale. citeturn0search19turn13search0

The roadmap view is oriented around date/iteration fields, and supports drag-and-drop changes that affect start/target dates (or iteration), which is useful for planning—but can become labour-intensive when schedule change requires multi-item updates and dependency-aware shifts. citeturn0search25turn0search29

image_group{"layout":"carousel","aspect_ratio":"16:9","query":["GitHub Projects roadmap view screenshot","GitHub Projects board view screenshot","GitHub Projects insights charts screenshot"],"num_per_query":1}

### Evidence of recurring gaps from community discussions

Across several years of community threads, pain points cluster into “automation boundaries” and “portfolio-scale visibility”:

**Dependency and schedule mechanics**
- Users request dependency-aware schedule shifting and bulk movement on roadmaps (e.g., “dragging one bar should move subsequent dependent items”). citeturn0search5  
- Roadmap usability issues include lack of contextual fields visible in the list/timeline without opening each item. citeturn0search26

**Portfolio / multi-repo coverage**
- Auto-add workflows are described as limited when organisations want to auto-add issues across *multiple repositories* or organisation-wide criteria. citeturn13search16  
- Another thread highlights friction when project workflows become tied to a departed org member (suggesting brittleness and a need for more maintainable automation approaches). citeturn24search22

**Operational hygiene and “board truth”**
- Requests to hide closed items (or auto-archive/remove them on a schedule) appear repeatedly. citeturn1search10turn1search7  
- Sub-issues usability: users ask to hide closed sub-issues because large parent issues become slow/noisy to manage. citeturn1search0turn1search24  
- Status synchronisation friction persists (e.g., “drag to Done” doesn’t automatically close issues, requiring manual clicks). citeturn1search19turn26search11

**Workflows and reporting**
- Users ask for richer custom workflow building inside Projects (more triggers/conditions/actions) beyond today’s defaults. citeturn24search2turn24search14  
- Teams still ask how to create burn down and velocity charts using Insights, implying the built-in charting is flexible but not turnkey for common agile views in many setups. citeturn0search16turn24search31

### How other “Boards” products cover these gaps (as a comparison point)

While GitHub has added charting and automation capabilities over time (including references to cycle velocity and cumulative flow diagrams in changelog posts), community feedback shows many teams still perceive gaps in dependency planning, agile reporting, and portfolio-level rollups. citeturn24search11turn0search16turn0search5

By contrast:
- Jira’s Advanced Roadmaps explicitly supports **dependencies** and cross-project planning constructs such as cross-project releases, aligning work across multiple projects and surfacing dependency signals on timelines. citeturn14search1turn14search18  
- Azure DevOps Boards has deep, first-party **Analytics** reporting such as sprint burndown and cumulative flow diagrams integrated into the work tracking experience. citeturn14search0turn14search33turn14search6

The key implication for GitHub Projects is not that it “cannot” do these things, but that **humans end up compensating**: they build scripts, enforce conventions, and spend meeting time interpreting inconsistent board state.

## What agent-embedded Projects enables: concrete capabilities and why they’re qualitatively different

GitHub Agentic Workflows is presented as a way to author AI-powered automation in Markdown that runs inside GitHub Actions, with additional guardrails such as safe outputs and sandboxed execution. It is explicitly described as early-stage and requiring careful supervision. citeturn26search0turn26search12turn1search15

ProjectOps in gh-aw frames the core difference: when an issue or PR arrives, an agent can analyse content and decide **where it belongs**, **which status/fields to set**, and whether to create/update project structures—while project operations are executed through safe outputs in separate scoped jobs so the agent job does not directly hold Projects write credentials. citeturn5view0turn12view0

The capability change is “fundamental” because it moves Projects from **rule-trigger automation** (labels/status changes) to **contextual decision-making** plus **cross-item reasoning**:

### Intent inference and content-based routing
GitHub’s built-in project workflows are primarily event/filter driven, whereas ProjectOps explicitly positions itself as “content-based routing” and “dynamic field assignment” driven by analysing issue/PR content. citeturn5view0turn26search3

### Cross-item reasoning and dependency synthesis
A single issue rarely contains all planning context. Cross-item reasoning becomes feasible when you combine:
- Orchestrator/worker patterns (split work, dispatch specialised workers, then aggregate outcomes). citeturn8view2turn1search12  
- Projects & Monitoring patterns that treat Projects as a durable “source of truth” for what workflows discovered/decided/did, including custom fields like Tracker IDs to correlate runs and initiatives. citeturn15view0turn22view0

This can directly address community requests such as dependency-aware timeline updates: an agent can synthesise implied dependencies from links/sub-issues/labels, propose shifts, and encode decisions into explicit project fields—even if the underlying Project UI does not natively “shift the chain” in one gesture. citeturn0search5turn5view0turn12view0

### Dynamic prioritisation and automated scheduling
The `update-project` safe output supports updating status, priority, numeric estimates, iteration/sprint fields, and start/target dates (with supported field types such as DATE, NUMBER, ITERATION, SINGLE_SELECT), enabling agents to compute and apply schedule/priority updates as structured operations. citeturn12view0turn5view0

### Portfolio aggregation across repositories
MultiRepoOps enables cross-repository coordination using safe outputs with `target-repo`, combined with authenticated GitHub tool access, making organisation-wide tracking and rollups practical without requiring a separate project management system. citeturn7view2turn21search11

### Natural-language summaries and stakeholder-ready reporting
Project status updates (`create-project-status-update`) are designed to publish progress summaries, findings, trends, and status indicators to a project’s Updates tab—particularly for scheduled workflows and orchestrators. citeturn12view1turn15view0turn9view0

### Proactive suggestions (not just reactive automation)
DailyOps and DataOps patterns push towards scheduled, incremental analysis and reporting: deterministic data extraction steps feed agent interpretation, and scheduled workflows can post daily/weekly updates or make small improvements. citeturn9view0turn8view0  
This turns project management from “people remember to update the board” into “the system continuously proposes the next best action.”

## How day-to-day human activities change when agents run the board

### From manual triage to “triage supervision”
IssueOps turns incoming issues into triggers for automated analysis and responses using safe outputs (e.g., add-comment, add-labels) while emphasising secure separation: the main AI job runs with read permissions and the write happens in a separate job. It also exposes sanitised issue text designed to reduce prompt-injection risk (e.g., removing @mentions, URIs, injections). citeturn6view0turn11view1turn10view2

In practice, that shifts a human triager’s day from *doing* classification to *deciding* whether the agent’s categorisation is correct and whether exceptions need handling (e.g., escalating security issues, reclassifying ambiguous requests).

### Planning shifts from “backlog grooming” to “policy tuning”
ProjectOps can automatically set priority/effort/status and create views/fields as configured; humans increasingly tune:
- *What* evidence the agent should use (templates, reproduction signals, linked PR state, impacted area).  
- *How* the project should encode meaning (field definitions, allowed values, default views). citeturn5view0turn12view0

### Review becomes change-control: “diff review” and rationale checking
Instead of reviewing a person’s drag-and-drop changes (often undocumented), the clean workflow becomes: agent proposes a structured set of changes → humans review a diff-like explanation → safe outputs apply changes with an auditable trail. This matches the core design rationale of safe outputs: separating read (agent) from write (safe output jobs) improves accountability and contains blast radius. citeturn11view1turn10view3

### Stakeholder communication becomes continuous, not meeting-bound
With `create-project-status-update`, scheduled workflows can publish updates with “run summary / trends / next steps” on a cadence, reducing the reliance on weekly status meetings as the primary reporting mechanism. citeturn12view1turn9view0turn15view0

### Meetings shift from status reporting to decision-making (and this is where control is won or lost)
Human factors research warns that automation changes attention: teams can become complacent (insufficient monitoring) or biased (over-reliance on flawed automation). Automation bias and complacency are documented risks in supervisory control settings, including that bias can affect both individuals and teams. citeturn2search8turn2search17turn2search20  
Therefore, the meeting agenda must explicitly incorporate **decision points**, **exceptions**, and **risk review**, rather than assuming “the board is true because an agent updated it”.

### Today vs AI-enabled state (summary comparison)

| Area | Typical today (human-centric) | AI-enabled (agentic Projects) |
|---|---|---|
| Intake & triage | Humans read, label, route; often inconsistent and time-consuming; desire for more automation remains common. citeturn24search10turn24search2 | Agents classify and route using content analysis; humans review edge cases; write actions happen via safe outputs with scoped permissions. citeturn6view0turn11view1 |
| Dependency & roadmap changes | Roadmap supports drag-to-set dates, but dependency-aware bulk shifting is a recurring request. citeturn0search25turn0search5 | Agents synthesise dependencies and propose coordinated changes via `update-project` (dates/iterations/priority), with explicit rationale and review gates. citeturn12view0turn8view2 |
| Portfolio rollups | Cross-repo auto-add and portfolio views are often requested and/or require custom tooling. citeturn13search16turn13search20 | MultiRepoOps aggregates across repos; Projects becomes the “portfolio database” plus status updates. citeturn7view2turn12view1turn15view0 |
| Reporting | Insights exist, but teams ask for common agile charts (burndown/velocity) and richer reporting. citeturn0search19turn0search16 | DataOps + scheduled updates generate tailored summaries and metrics narratives; humans validate interpretation. citeturn8view0turn9view0 |

## New human roles and responsibility boundaries

When agents act on project boards, **humans don’t disappear**; they concentrate into different roles with sharper “decision rights”:

### Approver
Owns: whether a proposed change is applied (especially anything that affects commitments, dates, priority changes, or cross-team dependencies).  
This aligns with the safe outputs model explicitly described as providing an approval gate and audit trail by separating read (agent) and write (safe output jobs). citeturn11view1turn10view3

### Policy author
Owns: the workflow’s “constitution”: prompts, field schemas, allowed labels/values, max operations, and which safe outputs are enabled.  
ProjectOps and safe outputs support strong configuration points (e.g., `max` operation caps, view/field configuration, allowed lists). citeturn5view0turn12view0turn6view0

### Exception handler
Owns: handling misroutes, conflicts, and ambiguous cases; maintaining a feedback loop that updates rules/policies rather than repeatedly “fixing the same class of mistake”.

### Auditor
Owns: reviewing logs, decisions, and tool usage; checking whether the system is behaving as designed. gh-aw explicitly supports operational monitoring such as `gh aw logs` and `gh aw audit`. citeturn15view0turn21search7turn21search2

## Governance and safety patterns that preserve human control

gh-aw’s security posture is explicitly “defence in depth”, including safe outputs, sandboxing, and threat detection, and acknowledges untrusted components and prompt-injection-style threats in its threat model. citeturn11view0turn10view2turn23search3  
This section translates those primitives into practical governance patterns for agent-managed Projects.

### Scoped autonomy tiers

A workable governance model is to define autonomy tiers that map to “blast radius”:

- **Tier 0 (observe)**: agents read and summarise only; no project field changes; status updates posted as commentary.  
- **Tier 1 (suggest)**: agents propose structured changes (fields/status/dates) but do not apply; outputs appear as a diff for human approval.  
- **Tier 2 (auto-apply within strict bounds)**: agents can apply low-risk, reversible updates (e.g., adding to project, setting “Needs Triage”, assigning an iteration within a fixed sprint window), capped by `max` and allowlists.  
- **Tier 3 (high-impact, gated)**: changes to commitments (target dates, priority escalations, cross-repo actions) require explicit approval triggers (e.g., slash command) and/or staged mode preview.

The foundations for this are present in gh-aw:
- Read-only-by-default permission model, with write via safe outputs in separate jobs. citeturn11view1turn10view3  
- Safe output operation caps (`max`) and structured schemas for project updates. citeturn12view0turn12view1  
- Staged mode support for previewing safe output operations without executing them (including the environment variable `GH_AW_SAFE_OUTPUTS_STAGED`). citeturn20view0

### Explainability and “rationale logging” as a control mechanism

Human trust in automation is brittle in both directions:
- People can over-trust tools (automation bias), accepting outputs without sufficient scrutiny. citeturn2search8turn2search17  
- People can also reject algorithms after observing errors (algorithm aversion), even when the algorithm is generally better. citeturn2search3turn2search0

A pragmatic mitigation is to require every agent-driven proposal to include:
1) **Evidence** (what it read), 2) **Reasoning summary** (why), 3) **Proposed change** (what), 4) **Confidence** (how sure), 5) **Escalation path** (who decides if uncertain).

This is a “control pattern” because it transforms automation from invisible state changes to explicit decision artefacts.

### Audit logs, rollback, and “time travel” for project state

gh-aw provides several primitives that can be composed into strong audit/rollback mechanics:

- **Operational monitoring tools**: status/logs/audit analysis are first-class in the monitoring pattern (inspect tool usage, failures, network patterns). citeturn15view0turn21search2  
- **Cache Memory** (time-bounded) and **Repo Memory** (branch-backed, persistent) enable storing snapshots of project state or decision logs between runs. citeturn22view1turn22view0  
- **TrialOps** runs workflows in isolated trial repos and captures safe outputs without affecting production, supporting test-and-compare before changing live governance. citeturn17view0turn21search11  
- **Ephemerals** includes workflow stop-after deadlines and auto-expiration/cleanup patterns (avoiding a runaway “automation backlog”). citeturn25view0

A robust pattern is: snapshot project fields → apply changes → if anomalies detected, restore from snapshot via a rollback workflow (using the same `update-project` safe output but with stored prior values). The feasibility is grounded in the fact that `update-project` is a structured operation over fields, and Repo Memory can persist the prior snapshot across runs. citeturn12view0turn22view0

### Threat surface management: prompt injection, untrusted input, and token isolation

Prompt injection is broadly recognised as a top vulnerability class for LLM applications, and systematic studies and security bodies treat it as a serious risk. citeturn23search3turn23news45  
gh-aw introduces multiple mitigations that are directly relevant to Projects automation:

- **Sanitised issue context** via IssueOps (`needs.activation.outputs.text`) to reduce injection vectors like @mentions and URIs. citeturn6view0  
- **Threat detection** jobs that run when safe outputs are configured, scanning agent output and code changes for prompt injection attempts, secret leaks, and malicious patches before applying them. citeturn10view2turn10view3  
- **Token isolation** for project operations: ProjectOps highlights that the agent job does not see the Projects token; safe outputs execute in separate scoped jobs. citeturn5view0  
- **Lockdown mode** for public repositories: filters visible content to trusted contributors (push access) to reduce exposure to malicious issues/comments. citeturn18search1turn18search4

### Sample safe-output policy table

The table below is an example of how to encode “control” into configuration. It is intentionally conservative. The concrete knobs and constraints map to documented safe output parameters (for example `max`, allowlists, project URL scoping, and staged mode). citeturn12view0turn12view1turn6view0turn20view0

| Safe output | Allowed scope | Autonomy tier | Guardrails (example) | Human decision point |
|---|---|---|---|---|
| `update-project` | Single project URL (portfolio board) citeturn12view0 | Tier 2 for low-risk fields; Tier 1 for dates/priority | `max: 10`; allow field list: status, triage state, team; date changes only in staged mode; require rationale text and evidence links | Approver reviews any date/priority changes (apply via slash command) |
| `create-project-status-update` | Same project URL citeturn12view1 | Tier 2 | `max: 1`; must include “What changed” + “Risks” + “Requests for decision” | Stakeholders can challenge “status” and request audit |
| `add-labels` | Repository allowlist, label allowlist citeturn6view0 | Tier 2 | `allowed: [bug, needs-info, question, doc]`; `max: 2` | Exception handler reviews mislabels weekly |
| `add-comment` | Triggering issue/PR only citeturn6view0 | Tier 2 | `max: 1`; must include “next action required” not just summary | Human owns final response tone and commitments |
| Custom safe output (e.g., Slack notify) | Specific channel/webhook | Tier 1 until stable | Require staged mode preview; only allow read-only MCP tools; write job holds secret; log everything citeturn19view1turn20view0 | Approver enables auto-send after pilot metrics meet thresholds |
| `dispatch-workflow` | Worker list allowlist citeturn8view2 | Tier 2 | Only dispatch pre-approved worker workflows; `max: 10`; pass tracker_id for traceability | Auditor reviews worker outputs and failure rates |

## UX and interaction models for human–agent project control

### Core UX choice: “suggest-only” vs “auto-apply”
This is less a UI question than a governance decision. Given known human factors risks (over-reliance or aversion), the safest default is “suggest-only” for any action that changes **commitments** (priority, schedule, scope), and “auto-apply” only for **mechanical hygiene** updates with bounded impact. citeturn2search8turn2search3turn11view1

In gh-aw terms, this maps naturally to:
- safe outputs with strict caps (`max`) and allowlists; citeturn12view0turn6view0  
- staged mode previews; citeturn20view0  
- TrialOps for trying UX patterns without production risk. citeturn17view0

### Interaction pattern: inline rationale + change-diff
A durable pattern is to treat project changes like code changes:

- **Proposed Changes** (structured list of field edits with before/after values)  
- **Why** (evidence + reasoning summary)  
- **Confidence and uncertainty**  
- **Apply** (button or slash command that triggers safe output execution)  
- **Rollback** (re-run with stored snapshot)

### Notification design: fewer, higher-signal messages
Ephemerals includes features like hiding older comments from the same workflow before posting new ones, which directly addresses “AI noise” and keeps timelines readable. citeturn25view0

### Mermaid flowchart: agent–human decision flow with safety gates

```mermaid
flowchart TD
  A[Repo / Project event<br/>Issue opened, PR ready, daily schedule] --> B[Activation job<br/>Sanitise context, extract text]
  B --> C[Agent job (read-only)<br/>Infer intent, reason across items,<br/>propose structured actions]
  C --> D[Threat detection gate<br/>Scan for injection, secret leaks,<br/>malicious patches]
  D -->|Blocked| E[Escalate<br/>Create failure issue / status update<br/>Request human triage]
  D -->|Allowed| F{Autonomy tier?}
  F -->|Suggest-only| G[Post proposal comment / status update<br/>with change-diff + rationale]
  G --> H{Human approves?}
  H -->|No| I[Close loop<br/>Record rejection reason]
  H -->|Yes| J[Safe output job(s)<br/>Apply updates with scoped token]
  F -->|Auto-apply within bounds| J
  J --> K[Update project status update<br/>+ log/audit artefacts]
  K --> L[Optional snapshot to Repo Memory<br/>for rollback + trend tracking]
```

### Sample UI mock text snippets (illustrative)

> **🤖 ProjectOps proposal (requires approval)**  
> **Reasoning (short):** This issue matches “Performance regression” (mentions latency spike, recent release, impacted endpoint). Similar incidents last month were triaged as P1.  
> **Proposed board updates:**  
> • Status: `Needs triage` → `In progress`  
> • Priority: `Unset` → `High`  
> • Iteration: `Sprint 12` → `Sprint 12` (no change)  
> • Start date: `—` → `2026-03-03` *(requires approval)*  
> **Confidence:** 0.72 (uncertainty: missing reproduction steps)  
> **Decision needed:** Approve priority + start date?  
> **Actions:** `/approve-projectops` or `/request-more-info`

> **✅ Applied (auto-apply policy)**  
> Added to Portfolio board, set Status=`Needs triage`, Team=`Platform`.  
> No date/priority changes applied under current policy.

These UI patterns are “control devices”: they make agent drift visible, capture human decision rationale, and support later audit.

## Benefits, risks, and a staged adoption plan with concrete workflow specs

### Measurable benefits (what to expect and how to measure)

Direct empirical studies on AI pair-programming show material productivity gains for coding tasks; for example controlled experiments with GitHub Copilot found significantly faster completion times in specific programming tasks. citeturn2search1turn2search4  
While project management work is not identical, these findings justify treating “time saved” as a measurable hypothesis rather than wishful thinking. citeturn2search19turn23search0

For Projects specifically, the most defensible benefit metrics are **operational** and **quality-of-process**:

- **Board freshness**: % of items whose Status/Priority/Iteration matches defined rules.  
- **Triage lead time**: issue opened → correctly routed + labelled.  
- **Planning throughput**: number of items moved from “Proposed” → “Ready” per week.  
- **Meeting load**: minutes spent preparing status vs deciding actions (track via self-report for a pilot).  
- **Exception rate**: % of agent proposals rejected or rolled back (should trend down as policy improves).  
- **Noise ratio**: number of notifications/comments created per meaningful action (Ephemerals can drive this down). citeturn25view0  
- **Security posture**: number of threat detection blocks; lockdown mode efficacy in public repos. citeturn10view2turn18search1turn18search4

### Risks (and why governance must be designed, not bolted on)

- **Over-automation and trust erosion**: automation bias can cause teams to accept wrong updates; algorithm aversion can cause teams to abandon automation after a visible mistake. citeturn2search8turn2search3  
- **Prompt injection / untrusted input**: prompt injection is widely treated as a serious vulnerability; if project automation acts on untrusted issues/comments, it can be coerced into harmful actions. citeturn23search3turn23news45  
- **Bias in prioritisation**: AI systems can encode or amplify biases present in data and conventions; surveys document many sources of bias and fairness failure modes in ML systems. citeturn23search5turn23search0  
- **Governance drift**: without explicit policy ownership, “prompt edits” become the new “silent configuration changes,” undermining auditability—hence the value of versioned workflows and audit tooling. citeturn26search1turn15view0

### Staged adoption plan (pilot → scale) with rollback criteria

This plan uses gh-aw’s own patterns (TrialOps, staged mode, monitoring, ephemerals) to keep adoption reversible and measurable. citeturn17view0turn20view0turn15view0turn25view0

**Pilot scope**
- One portfolio board (single team) + one primary repository.  
- Start with Tier 1 (suggest-only) for priority/date/schedule and Tier 2 (auto-apply) only for “add to project + set triage status”.

**Instrumentation and metrics**
- Require every proposal to include: evidence links + rationale + explicit change list.  
- Store snapshots of “before state” in Repo Memory to enable rollback experiments. citeturn22view0turn12view0  
- Use `create-project-status-update` weekly to publish metrics: proposals made, accepted, rejected, rolled back, threat detection blocks. citeturn12view1turn10view2

**Rollback criteria (examples)**
- >5% of applied changes require manual correction within 48h.  
- Any threat detection block involving credential leakage or suspicious external references. citeturn10view2turn25view0  
- Rising “noise ratio” (comment volume without matching value).

**Scale-out**
- Expand to MultiRepoOps only after single-repo policies stabilise, because cross-repo automation increases blast radius and auth complexity. citeturn7view2turn21search11  
- Use TrialOps to validate any policy/prompt change before deploying to production repos. citeturn17view0  
- Use Ephemerals stop-after on experimental workflows so pilots automatically sunset unless renewed (prevents “automation forever”). citeturn25view0

### Concrete workflow specs (ProjectOps + IssueOps + DataOps combos)

These are intentionally short “specs” showing how the patterns compose.

#### Intelligent intake triage (IssueOps + ProjectOps)
Goal: reduce manual routing and ensure every new issue lands in the right project with consistent fields.

```yaml
on:
  issues:
    types: [opened]

permissions:
  contents: read
  actions: read

tools:
  github:
    toolsets: [default, projects]
    github-token: ${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}

safe-outputs:
  update-project:
    project: https://github.com/orgs/ORG/projects/PORTFOLIO
    max: 1
    github-token: ${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}
  add-comment:
    max: 1
```

Why this is safe by design: Projects write operations are executed through safe outputs, not the agent job, consistent with the ProjectOps model. citeturn5view0turn11view1turn12view0

#### Weekly portfolio health report (DataOps + Monitoring)
Goal: pull deterministic metrics (counts, cycle-time proxies, blockers) and publish a consistent executive update.

```yaml
on:
  schedule: weekly on monday

tools:
  cache-memory: true

safe-outputs:
  create-project-status-update:
    project: https://github.com/orgs/ORG/projects/PORTFOLIO
    max: 1
    github-token: ${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}
```

Why this is powerful: DataOps formalises the separation between extraction and interpretation, and Projects & Monitoring treats Projects as the durable system of record for agent decisions and run summaries. citeturn8view0turn15view0turn12view1

#### Dependency-aware replanning loop (ProjectOps + Orchestration + Staged control)
Goal: respond to schedule shocks by proposing a coherent replan without silently moving dates.

```yaml
on:
  workflow_dispatch:

safe-outputs:
  dispatch-workflow:
    workflows: [dependency-scan-worker, schedule-proposal-worker]
    max: 10
  update-project:
    project: https://github.com/orgs/ORG/projects/PORTFOLIO
    max: 20
  create-project-status-update:
    project: https://github.com/orgs/ORG/projects/PORTFOLIO
    max: 1
```

Operating mode: workers propose changes; orchestrator posts a change-diff; `update-project` runs only after explicit approval, or in staged mode for preview during testing. citeturn8view2turn12view0turn20view0

### The strategic takeaway

Embedding agents turns GitHub project boards into a **living coordination layer**: continuously updated, decision-aware, and portfolio-capable. The real transformation is not “automation of board updates,” but a reallocation of human effort towards **governance, decision quality, and system design**, backed by concrete guardrails (safe outputs, threat detection, lockdown, staged mode, trial repos, and durable audit/memory). citeturn11view1turn10view2turn18search1turn17view0turn22view0turn26search0