
---
title: Objective Mapping & Portfolio Reporting Specification
version: 1.1.0
status: Partially Implemented
date: 2026-06-09
last_updated: 2026-06-10
---

# Objective Mapping & Portfolio Reporting Specification

This specification defines a reusable label-to-objective-value mapping layer for GitHub work. It also outlines later phases where that mapping can be applied to safe output outcomes, root issue tracing, and portfolio-level impact reporting.

## Implementation Scope

### Phase 1: Implemented Now

The current implementation is no longer just the bare mapping layer:

1. A shared GitHub utility loads `.github/objective-mapping.json`
2. Labels are mapped to numeric objective values through `ObjectiveMapping`
3. CLI outcome reporting enriches outcomes with `objective_value`, `objective_labels`, and `traced_root_url`
4. Pull request outcomes trace to linked closing issues before objective values are computed
5. Outcome summaries and per-objective breakdowns aggregate attempted and accepted objective value

This phase gives GitHub work a single configurable impact vocabulary and already supports basic root-aware outcome measurement.

### Phase 2+: Later Extensions

The rest of this document describes natural extensions that may be added later:

1. Root tracing beyond the current PR-to-closing-issue path, including epic resolution
2. Campaign-level aggregation and filtering in dedicated reports
3. Portfolio reporting workflows that consume the existing objective-enriched outcome data
4. Cost-aware efficiency metrics using real AI Credits data
5. Strategic analysis over delivered impact

Those extensions are design targets, not required complexity for trying the mapping itself.

## Overview: Impact Measurement Through Root Cause Tracing

Longer term, objective mapping answers: **What impact did we create?** By connecting work back to the root problems it solves.

For the MVP, the answer is simpler: **what value do these labels represent?**

For the extended model, that grows into:

1. **Root Problems** — Issues and epics represent actual business objectives (with PM-assigned impact values)
2. **Tracing** — Safe outputs (PRs, comments) trace backward to root issues/epics
3. **Impact Scoring** — Objectives are assigned to root problems, not intermediate artifacts
4. **Portfolio Reporting** — Aggregates impact by objective to show which business goals are being achieved

This enables questions like:
- **What high-priority problems did we solve?** (Trace accepted PRs back to root issues, check labels)
- **Which initiatives made the most progress?** (Aggregate impact by epic)
- **What value did we create per objective?** (Sum impact values for accepted outcomes)
- **What's our ROI by problem domain?** (Efficiency = accepted value / attempted value)

## Extended Architecture: Trace → Root → Map → Aggregate

This architecture describes the fuller impact model beyond the current MVP mapping layer.

### Components

1. **Root Tracing** — Traces safe outputs (PRs, issues) back to root issue or epic
2. **ObjectiveMapping** — Maps root issue/epic labels to numeric impact values
3. **Configuration File** — `.github/objective-mapping.json` with objective values
4. **Outcome Enrichment** — GitHub API queries to fetch root objects and their labels
5. **Portfolio Report** — Aggregates impact by objective, showing what value was created

### Design Principles

1. **Root Source of Truth** — Objectives are assigned to root issues/epics, not PRs or comments
2. **Traceability** — All work must trace back through GitHub's native linking (PR → issue)
3. **Centralized Configuration** — Single source of truth at `.github/objective-mapping.json`
4. **PM-Assigned Impact** — Labels on root objects represent business priorities
5. **Portfolio Visibility** — Aggregates show what problems were solved, what impact created

## Campaigns & Objective Alignment

### How Campaigns Work With Objectives

A **campaign** is a bounded initiative organized around specific business objectives. Examples:

- **"Q2 Performance Month"** — Campaign to improve latency (objective label: `initiative-performance`)
- **"Auth System Redesign"** — Major initiative (objective label: `epic-auth`)
- **"Critical Bug Fixes"** — Campaign to resolve urgent issues (objective label: `critical`)
- **"Testing Infrastructure"** — Initiative to improve test coverage (objective label: `testing`)

### Campaign → Objectives → Root Issues → Impact Measurement

```
Campaign: "Q2 Performance Month"
    ↓
Assigned Objectives: initiative-performance (300 points)
    ↓
Root Issues Created: #1234, #5678, #9012
(all labeled with "initiative-performance")
    ↓
Agent Creates PRs & Reviews (safe outputs)
    ↓
Safe Outputs Accepted/Rejected
    ↓
[Root Tracing] Trace PRs back to root issues
    ↓
[Objective Mapping] Fetch labels from root issues (initiative-performance)
    ↓
[Portfolio Report] Aggregate by campaign objectives:
    Total attempted: 1500 points (5 PRs × 300 points each)
    Delivered: 1200 points (4 accepted × 300 points)
    Efficiency: 80% → Campaign on track, good progress
```

### Campaign Reporting

Portfolio reports can be filtered by campaign to answer:

| Question | Answer | Example |
|----------|--------|---------|
| **How is this campaign doing?** | Efficiency metric | "Performance Month: 75% delivered (1500/2000 points)" |
| **Which campaigns succeeded?** | High efficiency campaigns | "Auth redesign: 90% complete, critical bugs: 85% complete" |
| **Which need intervention?** | Low efficiency campaigns | "Low-priority features: only 40% delivered, investigate blocker" |
| **What's total campaign impact?** | Sum all delivered objectives | "This quarter we delivered 12,000 points across 8 campaigns" |

### Configuration for Campaigns

Add campaign-level labels to `.github/objective-mapping.json`:

```json
{
  "label_to_value": {
    "epic-auth": 500,
    "epic-performance": 300,
    "initiative-modernize": 400,
    "campaign-q2-testing": 200,
    "critical": 100,
    "p0": 100,
    "p1": 50
  },
  "multi_label_logic": "max",
  "priority_labels": ["epic-auth", "epic-performance", "initiative-modernize", "campaign-q2-testing", "critical"]
}
```

**Strategy:** Campaigns typically use `multi_label_logic: "max"` so that a PR addressing both a campaign objective and a critical issue gets the higher value (captures the most important aspect).

### Connecting Campaigns to Execution

**Setup:**
1. PM defines campaign with clear objectives (e.g., "Ship auth redesign by EOQ")
2. Assign objective label to campaign (e.g., `epic-auth` with value 500)
3. All root issues in the campaign are labeled with that objective
4. Agent runs workflow against those issues

**Measurement:**
1. Safe outputs traced back to root issues via PR links
2. Root issues have objective label (`epic-auth`)
3. Portfolio report aggregates by campaign objective
4. Shows: "Campaign delivered 500/600 planned points (83% success)"

**Strategic Alignment:**
- Campaigns are how business divides work into initiatives
- Objectives are the labels that mark root issues as part of that campaign
- Impact measurement answers: "Did the campaign deliver what was planned?"

## Configuration

### File Format

The configuration file maps objective labels (typically on root issues/epics) to impact values:

```json
{
  "_comment": "Impact mapping for business objectives. Labels are assigned to root issues/epics by PM/team.",
  "label_to_value": {
    "epic-auth": 500,
    "initiative-performance": 300,
    "critical": 100,
    "p0": 100,
    "p1": 50
  },
  "multi_label_logic": "max",
  "priority_labels": ["epic-auth", "initiative-performance", "critical", "p0"]
}
```

### Location & Precedence

Objectives are loaded in this order (first found wins):

1. **Environment Variable** — `OBJECTIVE_MAPPING_JSON` (full JSON string or file path)
2. **Repository File** — `.github/objective-mapping.json`
3. **Built-in Defaults** — Fallback with standard objectives

### Typical Objective Labels

These are assigned by PMs/teams to root issues and epics:

- **Epics** (e.g., `epic-auth`, `initiative-modernize`) — Major initiatives worth 300–500 impact
- **Critical** (e.g., `critical`, `p0`) — Must-fix problems worth 100 impact
- **High-priority** (e.g., `p1`) — Important work worth 50 impact
- **Domains** (e.g., `security`, `performance`) — Strategic focus areas worth 30–80 impact

### Multi-Label Logic

When an outcome has multiple objective labels, the system applies one of three strategies:

| Strategy | Behavior | Use Case | Example |
|----------|----------|----------|---------|
| **max** (default) | Uses highest value | Risk-based prioritization | `[bug, p0]` → 100 |
| **sum** | Adds all values | Cumulative impact | `[performance, workflow]` → 75 |
| **first** | Uses priority order | Organizational hierarchy | `[p0, testing]` → 100 (p0 first) |

#### Example: Multi-Label Computation

Given labels `[bug, p0, testing]` with values `{bug: 70, p0: 100, testing: 75}`:

```
max:   max(70, 100, 75) = 100
sum:   70 + 100 + 75 = 245
first: depends on priority_labels order
```

## Current Outcome Integration and Remaining Root-Tracing Gaps

This section describes what is already implemented in the CLI today and what still remains future work.

### The Problem This Solves

When a PR is merged (safe output accepted), we need to know: **What business objective did it deliver?**

- The PR itself may have no labels
- The PR links to one or more issues
- Those issues contain the real business labels
- We must trace PR → issue → get labels → map to impact value

This is how GitHub always worked: root issue describes the problem, PRs are the solution.

### Data Flow

```
Safe output created (e.g., "create_pull_request")
  ↓
[EvaluateOutcomes] → outcome = "accepted" (merged) or "rejected" (closed)
  ↓
[enrichOutcomeWithObjectiveValue]
  1. For PR outcomes: GitHub API trace via closing issues
  2. For direct issue outcomes, or if PR tracing fails: fetch labels from the issue itself
  3. Use labels from the traced root object, not PR labels
  4. Store traced_root_url for audit trail
  ↓
ObjectiveMapping.ComputeObjectiveValue(root_labels)
  ↓
OutcomeReport populated with:
  - objective_value: int
  - objective_labels: []string
  - traced_root_url: string
  ↓ [ComputeOutcomeSummary]
OutcomeSummary aggregates:
  - total_objective_value (what we attempted)
  - accepted_objective_value (what succeeded)
  - objective_efficiency (success rate by value)
  ↓ [ComputeDomainBreakdowns]
DomainBreakdown per objective:
  - attempted: count of work toward this objective
  - accepted: count successfully delivered
  - total_objective_value: impact points we attempted
  - accepted_objective_value: impact points we delivered
  - objective_efficiency: % of value we succeeded on
```

Current limitation: epic resolution is still a future extension. The implemented trace path is PR → closing issue, with fallback to direct issue labels.

### Root Resolution Algorithm

```go
func traceOutcomeRoot(obj GitHubObject, repo string) GitHubObject {
  if obj.Type == "PullRequest" {
    if len(obj.ClosingIssues) > 0 {
      return obj.ClosingIssues[0]
    }
  }

  return obj
}
```

Future extension:

```go
func traceToRootIssueOrEpic(obj GitHubObject, repo string) GitHubObject {
  root := traceOutcomeRoot(obj, repo)
  if root.Type == "Issue" && root.EpicLink != nil {
    return root.EpicLink
  }
  return root
}
```

### Why Root Tracing Matters

**Example: PR for "fix auth bug"**

Without tracing:
- PR has no labels → objective_value = 0 → shows no impact

With the currently implemented tracing:
- PR links to issue #1234 (labeled `agentic-campaign`, `security`)
- Root issue labels feed the mapping → objective_value is computed from the configured label map
- Outcome summaries and objective breakdowns reflect delivered value on the root issue labels

This is the **only way** to measure what business value was created, because PRs don't carry the semantic meaning — issues do.

## Future Portfolio Reporting: Measuring What Value Was Created

### The Question It Answers

Instead of: **"How many PRs did we merge?"** (25 PRs, so what?)

This asks: **"How much business value did we deliver?"** (We aimed for 1000 impact points on our critical objectives, delivered 750, 75% success rate)

### Impact Metrics

Each objective (label assigned to root issues/epics by PM) shows:

| Metric | Meaning | Example |
|--------|---------|---------|
| **Attempted** | Work started toward this objective | 20 PRs addressing `epic-auth` |
| **Accepted** | Work successfully delivered | 15 of 20 PRs merged |
| **Total Impact** | Value we tried to deliver | 20 attempts × 100 points = 2000 |
| **Delivered Impact** | Value we actually delivered | 15 successes × 100 points = 1500 |
| **Efficiency** | Percentage of value achieved | 1500 / 2000 = 75% ✅ Good progress |

### Objective Breakdown Metrics

Each objective is aggregated with these metrics:

| Field | Type | Meaning |
|-------|------|---------|
| `label` | string | Business objective (e.g., `epic-auth`, `critical`) |
| `attempted` | int | Total work started on this objective |
| `accepted` | int | Work successfully delivered |
| `rejected` | int | Work that failed or was rejected |
| `pending` | int | Work still in progress |
| `total_objective_value` | int | Impact value attempted (sum of all values) |
| `accepted_objective_value` | int | Impact value delivered (sum of accepted values) |
| `objective_efficiency` | float64 | accepted / total (percentage of planned value realized) |
| `acceptance_rate` | float64 | accepted / attempted (percentage of work that succeeded) |

### Example: Portfolio Impact Report

```json
{
  "total": 50,
  "accepted": 35,
  "objective_efficiency": 0.75,
  "domain_breakdowns": [
    {
      "label": "epic-auth",
      "attempted": 20,
      "accepted": 15,
      "rejected": 4,
      "pending": 1,
      "total_objective_value": 10000,
      "accepted_objective_value": 7500,
      "objective_efficiency": 0.75,
      "acceptance_rate": 0.75
    },
    {
      "label": "critical",
      "attempted": 15,
      "accepted": 14,
      "rejected": 1,
      "pending": 0,
      "total_objective_value": 1500,
      "accepted_objective_value": 1400,
      "objective_efficiency": 0.93,
      "acceptance_rate": 0.93
    },
    {
      "label": "p1",
      "attempted": 15,
      "accepted": 6,
      "rejected": 8,
      "pending": 1,
      "total_objective_value": 750,
      "accepted_objective_value": 300,
      "objective_efficiency": 0.40,
      "acceptance_rate": 0.40
    }
  ]
}
```

### What This Report Says

- **Epic-auth**: Aimed for 10,000 impact on this initiative, delivered 7,500 (75% success) → Continue but monitor
- **Critical**: Aimed for 1,500 impact, delivered 1,400 (93% success) → Excellent, keep strategy
- **P1**: Aimed for 750 impact, delivered only 300 (40% success) → Investigate issues, may need human review

### Performance Analysis: Impact-Based Insights

The `AnalyzeDomainPerformance()` function interprets efficiency to answer: **How well are we delivering on this objective?**

| Efficiency | Status | Meaning |
|-----------|--------|---------|
| ≥ 90% | excellent | Delivering nearly all planned value → Keep strategy, scale if possible |
| ≥ 75% | good | Strong progress on objective → Monitor for regressions, maintain discipline |
| ≥ 50% | fair | Moderate success, room to improve → Review process, may need human guidance |
| < 50% | poor | Failing to deliver value → Investigate root cause, pause or redesign automation |

**Example Interpretations:**

- **epic-auth at 75%**: Started with 10,000 impact points planned, delivering 7,500. We're solving most auth problems successfully, but some are slipping. Review what's failing.
- **critical at 93%**: Nearly perfect on critical issues. Strategy is working. Could increase volume.
- **p1 at 40%**: Only delivering 40% of planned p1 work. Major problems here — investigate before continuing.

## Business Impact Model

### Key Insight: Root Tracing is Non-Negotiable

This system only works if we trace back to root issues/epics. Here's why:

1. **Root issues carry business semantics** — They're labeled by PMs with strategic intent
2. **PRs are tactical** — They're solutions, not problems; they shouldn't carry business labels
3. **GitHub's native model** — Issues represent work-to-do, PRs represent work-done
4. **Audit trail** — We can show exactly which business problems were solved

### Example: Bad vs. Good Impact Measurement

**Without Tracing (Bad):**
```
Safe output: PR #456 merged
PR labels: none
Conclusion: No impact (objectiveValue = 0)
```

**With Tracing (Good):**
```
Safe output: PR #456 merged
PR linked to: Issue #123 "Fix auth token expiry"
Issue #123 labels: epic-auth, critical
Conclusion: 100 impact points delivered (from critical label)
Portfolio: "We delivered a critical fix to the auth epic"
```

## Data: How Objectives Should Be Assigned

PMs assign objectives by labeling root issues/epics. Examples:

| Root Object | Labels | Impact | Meaning |
|---|---|---|---|
| Issue #1234 | `epic-auth` | 500 | Work toward major auth initiative |
| Issue #5678 | `critical` | 100 | Must-fix bug blocking users |
| Issue #9012 | `p1` | 50 | Important enhancement |
| Epic "Modernize API" | `initiative-api-v2` | 1000 | Major multi-quarter initiative |

When a PR closes one of these issues, it inherits the impact value.

## API & Functions

### ObjectiveMapping

```go
type ObjectiveMapping struct {
    LabelToValue   map[string]int `json:"label_to_value"`
    MultiLabelLogic string         `json:"multi_label_logic"`
    PriorityLabels []string        `json:"priority_labels"`
}

// Compute value from labels using configured strategy
func (om *ObjectiveMapping) ComputeObjectiveValue(labels []string) int

// Get objective labels (mapped labels only)
func (om *ObjectiveMapping) GetObjectiveLabels(labels []string) []string

// Load from config file, env var, or defaults
func LoadObjectiveMappingFromConfig() *ObjectiveMapping

// Get built-in defaults
func DefaultObjectiveMapping() *ObjectiveMapping
```

### Objective Breakdown

```go
type DomainBreakdown struct {
    Label                  string  `json:"label"`      // Objective label (e.g., "epic-auth")
    Attempted              int     `json:"attempted"`  // Count of work toward this objective
    Accepted               int     `json:"accepted"`   // Count successfully delivered
    Rejected               int     `json:"rejected"`   // Count that failed
    Pending                int     `json:"pending"`    // Count in progress
    TotalObjectiveValue    int     `json:"total_objective_value"`    // Impact attempted
    AcceptedObjectiveValue int     `json:"accepted_objective_value"` // Impact delivered
    ObjectiveEfficiency    float64 `json:"objective_efficiency"`     // Efficiency %
    AcceptanceRate         float64 `json:"acceptance_rate"`          // Success %
}

// Aggregate outcomes by objective label
func ComputeDomainBreakdowns(reports []OutcomeReport) []DomainBreakdown

// Generate strategic insight
func AnalyzeDomainPerformance(breakdown DomainBreakdown) DomainInsight
```

## Testing

### Unit Tests

The `label_objective_mapping_test.go` covers:

- Max/sum/first combination logics with multiple label scenarios
- Case insensitivity and whitespace trimming
- Nil and empty slice handling
- Combined label computation
- Real-world scenarios (e.g., `[bug, p0]`, `[performance, workflow]`)

All tests must pass before deployment:

```bash
go test ./pkg/github -run TestObjectiveMapping
go test ./pkg/cli -run "TestComputeOutcomeSummary|TestEvaluateOutcomes"
```

### Current Integration Tests

Current tests verify:

1. Objective values are computed with the configured combination strategy
2. Pull request outcomes trace to closing issues before label evaluation
3. Direct issue-label fallback works when PR tracing is unavailable
4. Objective summaries and breakdowns aggregate correctly
5. Audit trail (`traced_root_url`) is recorded correctly

### Future Integration Tests

Additional end-to-end testing should verify:

1. Root tracing correctly follows PR → issue → epic links once epic support exists
2. Labels fetched from final root objects, not from intermediate artifacts
3. Cost-aware efficiency metrics calculate accurately when AI Credits data is available

## Extended Performance Considerations

1. **Mapping Loaded Once** — At CLI startup, reused for all outcomes
2. **GitHub API Calls** — One call per outcome to fetch labels (async batch recommended)
3. **Aggregation** — O(n) scan of outcomes to compute domains
4. **Memory** — Domain map is O(unique labels), typically < 100 entries

## Error Handling

| Error | Behavior | Recovery |
|-------|----------|----------|
| Missing config file | Use defaults | Application continues |
| Invalid JSON in config | Use defaults, log error | Application continues |
| GitHub API 404 | Skip enrichment, value = 0 | Outcome evaluates normally |
| GitHub API 5xx | Log error, skip enrichment | Retry on next evaluation cycle |
| Invalid label in config | Ignored | Mapping continues with valid labels |

## Future Extensions

1. **Batch Root Tracing** — Async batch fetching of root issues to reduce GitHub API rate limits
2. **Impact Trends** — Track efficiency trends over time (e.g., "epic-auth efficiency improved from 60% to 75%")
3. **Multi-Issue Links** — Handle PRs linked to multiple issues with different objectives
4. **Epic Hierarchies** — Support nested epics (epic → parent epic → get labels from both)
5. **Workflowized Portfolio Reports** — Dedicated reporting workflows built on the existing objective-enriched outcome data
6. **AI Credits Integration** — Factor in real run-cost data for value-per-credit reporting
7. **Predictive Efficiency** — Use historical efficiency to forecast likely delivery rate

## References

- [Safe Output Outcome Evaluation Specification](./safe-output-outcome-evaluation.md)
- [AI Credits Specification](../docs/src/content/docs/specs/ai-credits-specification.md)
- Implementation: `pkg/github/label_objective_mapping.go`, `pkg/cli/outcome_domain_breakdown.go`
