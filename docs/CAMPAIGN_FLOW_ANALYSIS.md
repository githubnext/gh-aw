# Campaign Flow Analysis Report

**Date**: 2026-01-19  
**Version**: 1.0  
**Status**: Complete

## Executive Summary

This document analyzes the agentic campaign flow (orchestration, discovery, fusion) in gh-aw to identify what works well and what doesn't in the current implementation. Campaigns enable coordinated, multi-repository initiatives with AI-driven orchestration and GitHub Project board integration.

**Key Findings**:
- ✅ **Strong Foundation**: Well-architected discovery precomputation, deterministic manifest generation, and comprehensive governance
- ⚠️ **Complexity Gaps**: Missing error recovery documentation, limited multi-repository support, and unclear worker-orchestrator contract
- ⚠️ **Usability Issues**: No live campaign examples in repository, limited observability, and missing troubleshooting guides

---

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [What Works Well](#what-works-well)
3. [What Doesn't Work Well](#what-doesnt-work-well)
4. [Detailed Analysis](#detailed-analysis)
5. [Recommendations](#recommendations)

---

## Architecture Overview

### Core Components

```
┌─────────────────────────────────────────────────────────────┐
│                     Campaign Lifecycle                       │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  1. Campaign Spec (.campaign.md)                             │
│     └─> YAML frontmatter defines goals, workflows, KPIs     │
│                                                               │
│  2. Compilation (gh aw compile)                              │
│     └─> Generates orchestrator workflow (.campaign.lock.yml) │
│                                                               │
│  3. Discovery Precomputation (campaign_discovery.cjs)        │
│     └─> Searches GitHub, creates manifest JSON              │
│                                                               │
│  4. Orchestration (AI Agent)                                 │
│     └─> Reads manifest, updates project board               │
│                                                               │
│  5. Worker Dispatch (dispatch_workflow safe output)          │
│     └─> Fire-and-forget worker execution                    │
│                                                               │
└─────────────────────────────────────────────────────────────┘
```

### Key Files

| Component | Location | Purpose |
|-----------|----------|---------|
| **Spec Definition** | `pkg/campaign/spec.go` | CampaignSpec data structure with validation |
| **Orchestrator Generation** | `pkg/campaign/orchestrator.go` | Builds orchestrator WorkflowData from spec |
| **Discovery Script** | `actions/setup/js/campaign_discovery.cjs` | Precomputation script (runs before agent) |
| **Validation** | `pkg/campaign/validation.go` | JSON schema + semantic validation |
| **CLI Commands** | `pkg/campaign/command.go` | campaign, status, new, validate commands |
| **Workflow Discovery** | `pkg/campaign/workflow_discovery.go` | Discovers existing workflows by keywords |
| **Workflow Fusion** | `pkg/campaign/workflow_fusion.go` | Adapts workflows for campaign use |

---

## What Works Well

### 1. ✅ Discovery Precomputation Architecture

**Strength**: Separating discovery from agent execution creates deterministic, cacheable manifests.

**Evidence**:
- Discovery runs **before** the agent in a separate GitHub Actions step
- Outputs stable JSON manifest to `./.gh-aw/campaign.discovery.json`
- Enforces strict pagination budgets (items, pages)
- Saves cursor for incremental discovery across runs

**Benefits**:
```javascript
// Discovery is deterministic and budget-controlled
const DEFAULT_MAX_ITEMS = 100;
const DEFAULT_MAX_PAGES = 10;

// Cursor enables incremental discovery
function saveCursor(cursorPath, cursor) {
  fs.writeFileSync(cursorPath, JSON.stringify(cursor, null, 2));
}
```

**Impact**: Agents get consistent, repeatable input instead of performing expensive GitHub-wide searches during execution.

---

### 2. ✅ Comprehensive Governance Model

**Strength**: Fine-grained budgets prevent runaway campaign operations.

**Evidence** (`pkg/campaign/spec.go`):
```go
type CampaignGovernancePolicy struct {
    MaxNewItemsPerRun       int      // Limits new work per run
    MaxDiscoveryItemsPerRun int      // Discovery pagination budget
    MaxDiscoveryPagesPerRun int      // API call budget
    OptOutLabels            []string // [no-campaign, no-bot]
    MaxProjectUpdatesPerRun int      // Project board write budget
    MaxCommentsPerRun       int      // Comment budget
    DoNotDowngradeDoneItems *bool    // Prevents status downgrades
}
```

**Benefits**:
- Prevents overwhelming GitHub API
- Controls project board spam
- Supports gradual rollout
- Enables safe experimentation

---

### 3. ✅ Clean Separation of Concerns

**Strength**: Campaign components are well-isolated with clear responsibilities.

**Architecture**:

| Layer | Responsibility | Independence |
|-------|----------------|--------------|
| **Spec** | Declarative campaign definition | Pure data, no logic |
| **Validation** | Schema + semantic checks | JSON schema + Go rules |
| **Orchestrator** | WorkflowData generation | Pure function, testable |
| **Discovery** | GitHub search precomputation | Separate JS script |
| **Fusion** | Workflow adaptation | Isolated Go package |

**Evidence**: Each component can be tested independently (validation, orchestrator, fusion all have separate test files).

---

### 4. ✅ Repo-Memory Integration

**Strength**: Durable state persistence across workflow runs.

**Evidence** (`pkg/campaign/orchestrator.go`):
```go
"repo-memory": []any{
    map[string]any{
        "id":          "campaigns",
        "branch-name": "memory/campaigns",
        "file-glob":   extractFileGlobPatterns(spec),
        "campaign-id": spec.ID,
    },
}
```

**Benefits**:
- Cursor persistence for incremental discovery
- Metrics snapshots for trend analysis
- Auditable history in git branches
- No external database required

---

### 5. ✅ Campaign Item Protection

**Strength**: Prevents non-campaign workflows from interfering with campaign-tracked items.

**Mechanism**:
```yaml
# Campaign orchestrator automatically applies labels
tracker-label: campaign:security-q1-2025

# Other workflows check and skip
if (issueLabels.some(label => label.startsWith('campaign:'))) {
  core.info(`Skipping: managed by campaign orchestrator`);
  return false;
}
```

**Benefits**:
- No duplicate actions on campaign items
- Clear ownership boundaries
- Opt-out via labels (no-campaign, no-bot)

---

### 6. ✅ Validation System

**Strength**: Multi-layer validation catches issues early.

**Layers**:
1. **JSON Schema** - Type checking, required fields, enums
2. **Semantic Validation** - ID format, URL structure, KPI consistency
3. **Workflow Existence** - Referenced workflows must exist
4. **Compilation Validation** - Runs during `gh aw compile`

**Evidence**:
```go
// ValidateSpec performs lightweight semantic validation
func ValidateSpec(spec *CampaignSpec) []string {
    var problems []string
    // JSON schema validation first
    schemaProblems := ValidateSpecWithSchema(spec)
    problems = append(problems, schemaProblems...)
    
    // Additional semantic checks
    // ... ID format, URL validation, KPI consistency
    return problems
}
```

---

## What Doesn't Work Well

### 1. ⚠️ Limited Multi-Repository Support

**Problem**: Campaigns are designed for multi-repo initiatives but lack clear cross-repo discovery patterns.

**Evidence**:
- `allowed-repos` field exists but validation is minimal
- `allowed-orgs` field added but discovery implementation unclear
- Documentation focuses on single-repo examples
- No guidance on cross-org campaigns

**Impact**:
```yaml
# Current spec supports this
allowed-repos: ["github/docs", "github/cli"]
allowed-orgs: ["microsoft"]

# But discovery script doesn't clearly show how this works
# campaign_discovery.cjs has GH_AW_DISCOVERY_REPOS but limited documentation
```

**User Pain**: Teams attempting multi-repo campaigns hit undocumented limitations.

---

### 2. ⚠️ Missing Error Recovery Documentation

**Problem**: Campaign orchestrators run on schedules, but error handling is implicit.

**Evidence**:
- Documentation mentions "campaigns are designed to keep going" but lacks specifics
- No runbook for common failure scenarios
- Limited guidance on partial failures during discovery
- No structured logging for debugging failed runs

**Documentation Gap** (from `docs/src/content/docs/guides/campaigns/flow.md`):
```markdown
## When something goes wrong

Campaigns are designed to keep going and report what happened in the Project status update.

- **Dispatch failed**: fix the worker workflow (missing, not dispatchable), then wait for the next run.
- **Project updates hit a limit**: increase governance limits or let the campaign catch up over multiple runs.
- **Permissions errors**: ensure the workflow token has the required Projects permissions.
```

**Missing**:
- How to detect partial discovery failures
- When to reset cursors
- How to recover from bad manifest data
- Debugging workflow dispatch issues

---

### 3. ⚠️ Workflow Fusion Complexity

**Problem**: Workflow fusion adds complexity without clear benefits.

**Evidence** (`pkg/campaign/workflow_fusion.go`):
```go
// FuseWorkflowForCampaign takes an existing workflow and adapts it
// by adding workflow_dispatch trigger and storing it in a campaign-specific folder
func FuseWorkflowForCampaign(rootDir, workflowID, campaignID string) (*FusionResult, error) {
    // Creates: .github/workflows/campaigns/<campaign-id>/<workflow-id>-worker.md
    campaignDir := filepath.Join(rootDir, ".github", "workflows", "campaigns", campaignID)
}
```

**Issues**:
1. **Unclear Value**: Why copy workflows instead of dispatching originals?
2. **Maintenance Burden**: Fused workflows can drift from originals
3. **Namespace Pollution**: Separate folder structure adds complexity
4. **Limited Adoption**: No examples in repository use fusion

**Documentation** (from `docs/campaign-worker-fusion.md`):
> "Campaign worker workflow fusion adapts existing workflows for campaign use..."

**Missing**:
- When to use fusion vs. dispatch-workflow
- How to keep fused workflows in sync
- Migration path from original to fused workflows

---

### 4. ⚠️ No Live Campaign Examples

**Problem**: Repository lacks working campaign examples users can reference.

**Evidence**:
```bash
# No campaign specs in .github/workflows/
$ ls .github/workflows/*.campaign.md
# (empty)

# Campaign workflow tests skip because no specs exist
TestComputeCompiledStateForCampaign_UsesLockFiles (0.00s)
  campaign_test.go:78: campaign spec not found; skipping compiled-state test
```

**Impact**:
- Users can't learn by example
- Hard to validate campaign features work end-to-end
- Documentation examples are hypothetical, not proven
- Testing is limited without real campaign specs

---

### 5. ⚠️ Unclear Worker-Orchestrator Contract

**Problem**: How workers and orchestrators communicate is implicit, not explicit.

**Evidence**:
- Workers create issues/PRs with `gh-aw-tracker-id: workflow-name`
- Discovery searches for this marker: `"gh-aw-tracker-id: ${trackerId}"`
- But there's no schema validation for tracker-id format
- No documentation on required issue/PR fields for campaign tracking

**Missing Contract**:
```yaml
# What workers MUST include
tracker-id: workflow-name         # Required for discovery
labels: [campaign:campaign-id]     # Required for protection

# What orchestrators expect
url: string                        # Issue/PR URL
content_type: "issue" | "pull_request"
state: "open" | "closed"
title: string
created_at: timestamp
updated_at: timestamp

# Optional but recommended
closed_at: timestamp               # For closed items
merged_at: timestamp               # For PRs
```

**Impact**: Workers may not include required metadata, breaking discovery.

---

### 6. ⚠️ Limited Observability

**Problem**: Hard to understand what's happening during a campaign run.

**Evidence**:
- Discovery manifest written to `./.gh-aw/campaign.discovery.json` but not surfaced in UI
- No summary of discovery results in workflow logs
- Project status updates are the only output
- No metrics on discovery performance, API usage, or budget consumption

**Missing**:
```javascript
// Discovery script doesn't output summary to workflow logs
core.info(`Discovery complete: ${allItems.length} items found`);
core.info(`  - Open: ${openCount}`);
core.info(`  - Closed: ${closedCount}`);
core.info(`  - Pages scanned: ${pagesScanned}/${maxPages}`);
core.info(`  - Items scanned: ${itemsScanned}/${maxItems}`);
// ❌ These logs don't exist or are hard to find
```

**User Pain**: Debugging campaign issues requires downloading artifacts and parsing JSON manually.

---

### 7. ⚠️ Workflow Dispatch is Fire-and-Forget

**Problem**: Orchestrators dispatch workers but don't track their completion.

**Evidence** (from documentation):
> "Dispatch is fire-and-forget: the orchestrator does not wait for worker workflows to finish. Results are picked up on later runs."

**Issues**:
1. No way to detect failed worker runs
2. No correlation between dispatch and discovered items
3. Workers might fail silently
4. Hard to debug "why aren't items appearing?"

**Missing**:
```yaml
# Orchestrators could track dispatches
dispatch-log:
  - workflow: vulnerability-scanner
    run-id: 123456789
    status: success | failed | in_progress
    dispatched-at: 2026-01-19T10:00:00Z
    completed-at: 2026-01-19T10:05:00Z
```

---

### 8. ⚠️ Schema-Prompt Alignment Risk

**Problem**: Campaign behavior is controlled by prompts, not schema enforcement.

**Evidence**:
- Orchestrator instructions in `prompt_sections.go` are text templates
- Governance limits passed to agent via markdown instructions
- No compile-time guarantee agent follows instructions

**Example** (`pkg/campaign/orchestrator.go`):
```go
// Instructions rendered as markdown text
orchestratorInstructions := RenderOrchestratorInstructions(promptData)
projectInstructions := RenderProjectUpdateInstructions(promptData)
```

**Risk**:
- Agent might ignore governance limits
- Prompt changes could break campaigns
- No validation that prompts match schema

**Mitigation**: Safe outputs enforce maxima (`Max: maxComments`), but other instructions rely on agent compliance.

---

## Detailed Analysis

### Discovery Precomputation Deep Dive

**Architecture Decision**: Why separate discovery from agent execution?

**Problem with Agent-Driven Discovery**:
```
Agent Phase 1 (Current Run):
  └─> Search GitHub for campaign items
      ├─> 100+ API calls
      ├─> Non-deterministic results
      ├─> Expensive token usage
      └─> Slow execution
```

**Solution with Precomputation**:
```
Discovery Step (Before Agent):
  └─> campaign_discovery.cjs runs
      ├─> Enforced budgets (items, pages)
      ├─> Saves cursor for continuation
      └─> Writes deterministic manifest

Agent Phase 1 (Current Run):
  └─> Read manifest from ./.gh-aw/campaign.discovery.json
      ├─> Fast (local file read)
      ├─> Deterministic (same manifest every time)
      └─> Efficient (no GitHub API calls)
```

**Benefits**:
1. **Performance**: Discovery is parallelizable, agent work is sequential
2. **Cost**: Discovery uses GitHub Actions compute, agent uses AI tokens
3. **Reliability**: Manifest generation can retry without rerunning agent
4. **Testability**: Mock manifests for agent testing

**Trade-offs**:
- Adds complexity (separate script, manifest format)
- Discovery failures block agent execution
- Manifest can become stale if discovery fails

---

### Governance Model Deep Dive

**Design Principle**: Campaigns should pace themselves, not overwhelm systems.

**Budget Types**:

| Budget | Scope | Enforcement | Purpose |
|--------|-------|-------------|---------|
| `max-discovery-items-per-run` | Discovery script | Hard limit (break loop) | Prevent unbounded GitHub API usage |
| `max-discovery-pages-per-run` | Discovery script | Hard limit (break loop) | Rate limit API calls |
| `max-project-updates-per-run` | Safe output | Hard limit (max count) | Prevent project board spam |
| `max-comments-per-run` | Safe output | Hard limit (max count) | Prevent notification fatigue |
| `max-new-items-per-run` | Agent prompt | Soft limit (instruction) | Pace campaign work |

**Enforcement Layers**:

1. **Discovery Script** (Hard Limits):
```javascript
if (itemsScanned >= maxItems || pagesScanned >= maxPages) {
  core.warning(`Reached discovery budget limits. Stopping discovery.`);
  break;
}
```

2. **Safe Outputs** (Hard Limits):
```go
safeOutputs.UpdateProjects = &workflow.UpdateProjectConfig{
    BaseSafeOutputConfig: workflow.BaseSafeOutputConfig{Max: maxProjectUpdates},
}
```

3. **Agent Prompts** (Soft Limits):
```go
fmt.Fprintf(markdownBuilder, "- Governance: max new items per run: %d\n", 
    spec.Governance.MaxNewItemsPerRun)
```

**Key Insight**: Hard limits prevent damage, soft limits guide behavior.

---

### Repo-Memory Integration Deep Dive

**Architecture Decision**: Why use git branches for campaign state?

**Alternatives Considered**:
1. **Environment Variables**: Lost between runs ❌
2. **Workflow Artifacts**: Expire after 90 days ❌
3. **External Database**: Requires infrastructure ❌
4. **Git Branches**: Versioned, auditable, durable ✅

**Repo-Memory Benefits**:

```yaml
# Campaign cursor file
memory/campaigns/security-q1-2025/cursor.json
{
  "page": 3,
  "trackerId": "vulnerability-scanner",
  "updated_at": "2026-01-19T10:00:00Z"
}

# Metrics snapshots (append-only)
memory/campaigns/security-q1-2025/metrics/2026-01-19.json
{
  "date": "2026-01-19",
  "discovered_items": 42,
  "open_items": 25,
  "closed_items": 17
}
```

**Capabilities**:
- **Cursor Persistence**: Continue discovery from last position
- **Time-Series Metrics**: Track campaign progress over time
- **Historical Analysis**: Review past campaign states
- **Branch Protection**: Prevent accidental deletion

**Trade-offs**:
- Adds git complexity (branch management)
- Repo size grows with metrics
- Requires repo-memory tool configuration

---

### Validation Architecture Deep Dive

**Multi-Layer Validation Strategy**:

```
┌─────────────────────────────────────────────┐
│        Campaign Spec Validation              │
├─────────────────────────────────────────────┤
│                                               │
│  Layer 1: JSON Schema (Type Safety)          │
│  ├─> Required fields: id, name, project-url │
│  ├─> Enum validation: state, direction      │
│  └─> Type checking: int, string, array      │
│                                               │
│  Layer 2: Semantic Validation (Logic)        │
│  ├─> ID format: lowercase, hyphens only     │
│  ├─> URL structure: GitHub Project URLs     │
│  ├─> KPI consistency: exactly 1 primary     │
│  └─> Governance: non-negative integers      │
│                                               │
│  Layer 3: Reference Validation (Context)     │
│  ├─> Workflows exist in directory           │
│  ├─> Lock files compiled                    │
│  └─> Repo-memory paths valid                │
│                                               │
└─────────────────────────────────────────────┘
```

**Example Validation Flow**:

```go
// Layer 1: JSON Schema
schema.Validate(specData) // Type checking, required fields

// Layer 2: Semantic Rules
if !isValidID(spec.ID) {
    problems = append(problems, suggestValidID(spec.ID))
}

// Layer 3: Context Checks
ValidateWorkflowsExist(spec, workflowsDir)
```

**Validation Quality Metrics**:
- ✅ Actionable error messages with examples
- ✅ Suggestions for fixing issues
- ✅ Non-blocking warnings for best practices
- ⚠️ Limited validation of multi-repo configuration

---

## Recommendations

### Priority 1: Critical Issues

#### 1.1 Add Live Campaign Examples

**Action**: Create at least one complete campaign example in the repository.

**Implementation**:
```bash
# Create example campaign spec
.github/workflows/example-security-campaign.campaign.md

---
id: example-security-campaign
name: Example Security Campaign
objective: Demonstrate campaign features with a working example
project-url: https://github.com/orgs/githubnext/projects/1
workflows:
  - example-security-scanner
allowed-repos: ["githubnext/gh-aw"]
governance:
  max-discovery-items-per-run: 50
  max-project-updates-per-run: 10
kpis:
  - name: Security Issues Resolved
    priority: primary
    baseline: 0
    target: 100
    time-window-days: 30
---

This is an example campaign demonstrating core features...
```

**Benefits**:
- Users learn by example
- Validates end-to-end functionality
- Enables better testing
- Demonstrates best practices

**Effort**: 1-2 days (includes worker workflow, documentation)

---

#### 1.2 Document Error Recovery Procedures

**Action**: Create operational runbook for campaign failures.

**Implementation**:
```markdown
# Campaign Operations Runbook

## Discovery Failures

### Symptom: Manifest empty or incomplete
**Diagnosis**:
1. Check workflow run logs for discovery step
2. Look for rate limit errors or API failures
3. Verify cursor file exists and is valid

**Recovery**:
```bash
# Reset cursor if corrupted
rm memory/campaigns/<campaign-id>/cursor.json
# Re-run orchestrator workflow
gh workflow run <campaign-id>.campaign.lock.yml
```

### Symptom: Workers not creating trackable items
**Diagnosis**:
1. Verify worker workflow includes `gh-aw-tracker-id: workflow-name`
2. Check if discovery script searches for correct tracker-id
3. Verify allowed-repos includes worker's repository

**Recovery**: Update worker to include tracker-id in issue/PR body...
```

**Effort**: 2-3 days (research common failures, write procedures)

---

#### 1.3 Enhance Multi-Repository Discovery

**Action**: Clarify and test multi-repo discovery patterns.

**Implementation**:

1. **Update Discovery Script**:
```javascript
// Add explicit per-repo discovery
async function discoverByRepo(octokit, repo, trackerId) {
  // Scope search to specific repository
  const searchQuery = `repo:${repo} "gh-aw-tracker-id: ${trackerId}"`;
  // ... pagination logic
}
```

2. **Add Validation**:
```go
// Validate allowed-repos format and accessibility
func ValidateAllowedRepos(spec *CampaignSpec) []string {
    for _, repo := range spec.AllowedRepos {
        // Check format: owner/repo
        // Verify repo exists (optional: API call)
        // Warn if permissions might be insufficient
    }
}
```

3. **Document Patterns**:
```markdown
## Multi-Repository Campaigns

### Pattern 1: Explicit Repository List
```yaml
allowed-repos:
  - github/docs
  - github/cli
  - github/gh-aw
```

### Pattern 2: Organization-Wide
```yaml
allowed-orgs:
  - github
```

### Pattern 3: Mixed (Org + Specific Repos)
```yaml
allowed-orgs:
  - microsoft
allowed-repos:
  - github/docs  # Add specific GitHub repos
```

**Effort**: 3-4 days (implementation + testing + documentation)

---

### Priority 2: Usability Improvements

#### 2.1 Add Discovery Summary Logging

**Action**: Output structured discovery results to workflow logs.

**Implementation**:
```javascript
// At end of campaign_discovery.cjs main()
function logDiscoverySummary(manifest) {
  console.log("\n" + "=".repeat(60));
  console.log("📊 DISCOVERY SUMMARY");
  console.log("=".repeat(60));
  console.log(`Campaign ID: ${manifest.campaign_id}`);
  console.log(`Total Items: ${manifest.discovery.total_items}`);
  console.log(`  - Open: ${manifest.summary.open_count}`);
  console.log(`  - Closed: ${manifest.summary.closed_count}`);
  console.log(`  - Merged PRs: ${manifest.summary.merged_count}`);
  console.log(`Items to Add: ${manifest.summary.needs_add_count}`);
  console.log(`Items to Update: ${manifest.summary.needs_update_count}`);
  console.log(`Budget Usage:`);
  console.log(`  - Items: ${manifest.discovery.items_scanned}/${manifest.discovery.max_items_budget}`);
  console.log(`  - Pages: ${manifest.discovery.pages_scanned}/${manifest.discovery.max_pages_budget}`);
  if (manifest.discovery.cursor) {
    console.log(`Cursor: Page ${manifest.discovery.cursor.page}`);
  }
  console.log("=".repeat(60) + "\n");
}
```

**Benefits**:
- Easier debugging
- Visibility into budget consumption
- Quick health checks

**Effort**: 1 day

---

#### 2.2 Create Worker-Orchestrator Contract Document

**Action**: Formalize the contract between workers and orchestrators.

**Implementation**:
```markdown
# Campaign Worker-Orchestrator Contract

## Worker Requirements

Workers MUST include the following in created issues/PRs:

### 1. Tracker ID (Required for Discovery)

**Location**: Issue/PR body
**Format**: `gh-aw-tracker-id: <workflow-name>`

**Example**:
```markdown
## Issue Description
This issue tracks a security vulnerability...

<!-- Campaign tracking -->
gh-aw-tracker-id: vulnerability-scanner
```

### 2. Campaign Label (Required for Protection)

**Location**: Issue/PR labels
**Format**: `campaign:<campaign-id>`

**Example**: `campaign:security-q1-2025`

### 3. Metadata Fields (Required)

| Field | Type | Source | Example |
|-------|------|--------|---------|
| `title` | string | Issue/PR title | "Upgrade lodash to 4.17.21" |
| `url` | string | Issue/PR URL | "https://github.com/org/repo/issues/123" |
| `state` | string | Issue/PR state | "open" or "closed" |
| `created_at` | timestamp | GitHub API | "2026-01-19T10:00:00Z" |
| `updated_at` | timestamp | GitHub API | "2026-01-19T12:00:00Z" |

### 4. Optional Fields

| Field | Type | When Required | Example |
|-------|------|---------------|---------|
| `closed_at` | timestamp | When state is "closed" | "2026-01-20T08:00:00Z" |
| `merged_at` | timestamp | For merged PRs | "2026-01-20T09:00:00Z" |

## Orchestrator Guarantees

Orchestrators MUST:

1. **Search for tracker-id**: Include all issues/PRs with matching tracker-id
2. **Respect opt-out labels**: Skip items with `no-campaign` or `no-bot`
3. **Apply campaign labels**: Add `campaign:<id>` when adding to project
4. **Honor governance limits**: Respect discovery and update budgets

## Discovery Process

1. Orchestrator runs discovery precomputation
2. Discovery searches: `"gh-aw-tracker-id: ${trackerId}" type:issue`
3. Results normalized to manifest format
4. Agent reads manifest and updates project board
```

**Effort**: 2 days (research + documentation + examples)

---

#### 2.3 Improve Workflow Fusion or Remove It

**Action**: Either clarify fusion use cases or deprecate the feature.

**Option A: Clarify Use Cases**
```markdown
## When to Use Workflow Fusion

Use fusion when:
- Worker has triggers incompatible with campaigns (push, pull_request)
- You want campaign-specific configuration without modifying original
- Original workflow is maintained by another team

Don't use fusion when:
- Worker already has workflow_dispatch
- You can modify the original workflow
- Worker is simple (fusion adds overhead)

## Example: Fusion for Third-Party Workflow

```yaml
# Original: security-scan.md (maintained by security team)
on:
  push:
    branches: [main]
  schedule: daily
```

Fusion creates `campaigns/security-q1/security-scan-worker.md`:
- Adds workflow_dispatch
- Preserves original configuration
- Allows campaign-specific overrides
```

**Option B: Deprecate Fusion**
```markdown
## Workflow Fusion Deprecation (v2.0)

Workflow fusion will be removed in v2.0.

**Migration**: Instead of fusion, update worker workflows to support workflow_dispatch:

```yaml
# Before (fusion required)
on:
  push:
    branches: [main]

# After (fusion not needed)
on:
  workflow_dispatch:  # Campaign can dispatch
  push:               # Original trigger preserved
    branches: [main]
```

**Effort**: 2-3 days (analysis + decision + documentation)

---

### Priority 3: Technical Improvements

#### 3.1 Add Worker Dispatch Tracking

**Action**: Track workflow dispatches and correlate with discovered items.

**Implementation**:
```go
// Orchestrator adds dispatch tracking to safe outputs
type DispatchTrackingConfig struct {
    RecordDispatches bool   // Save dispatch log to repo-memory
    CorrelateItems   bool   // Match dispatches to discovered items
    LogPath          string // memory/campaigns/<id>/dispatches.json
}
```

**Dispatch Log Format**:
```json
{
  "dispatches": [
    {
      "workflow": "vulnerability-scanner",
      "run_id": 123456789,
      "dispatched_at": "2026-01-19T10:00:00Z",
      "status": "success",
      "items_created": ["https://github.com/org/repo/issues/456"]
    }
  ]
}
```

**Effort**: 4-5 days (implementation + testing)

---

#### 3.2 Schema-Enforced Governance

**Action**: Move governance from prompts to schema-enforced constraints.

**Implementation**:
```go
// Instead of prompt instructions
fmt.Fprintf(markdownBuilder, "- Governance: max new items: %d\n", max)

// Use schema-validated constraints
type GovernanceConstraints struct {
    MaxNewItems       *EnforcedLimit `json:"max-new-items"`
    MaxProjectUpdates *EnforcedLimit `json:"max-project-updates"`
}

type EnforcedLimit struct {
    Value    int    `json:"value"`
    Enforced bool   `json:"enforced"` // Hard vs. soft limit
    Message  string `json:"message"`  // Error message if exceeded
}
```

**Benefits**:
- Compile-time validation
- No prompt drift risk
- Clearer enforcement semantics

**Effort**: 5-6 days (refactoring + validation updates)

---

#### 3.3 Add Campaign Health Checks

**Action**: Create `gh aw campaign health` command to diagnose issues.

**Implementation**:
```go
func RunCampaignHealth(campaignID string) error {
    // Check 1: Spec is valid
    spec := LoadCampaignSpec(campaignID)
    problems := ValidateSpec(spec)
    
    // Check 2: Referenced workflows exist and compile
    for _, wf := range spec.Workflows {
        checkWorkflowCompiles(wf)
    }
    
    // Check 3: Discovery is working
    manifest := loadDiscoveryManifest(campaignID)
    checkManifestFreshness(manifest)
    
    // Check 4: Repo-memory is accessible
    checkRepoMemoryBranch(spec)
    
    // Check 5: Project board is accessible
    checkProjectAccess(spec.ProjectURL)
    
    // Output health report
    renderHealthReport(checks)
}
```

**Output**:
```
✅ Campaign spec validation: PASS
✅ Workflows exist: PASS (3/3 workflows found)
⚠️ Discovery freshness: WARN (manifest is 2 days old)
❌ Project access: FAIL (401 Unauthorized - check project-github-token)
✅ Repo-memory: PASS (memory/campaigns branch exists)

Overall Health: DEGRADED
Recommended Actions:
  1. Update project-github-token secret with correct permissions
  2. Re-run orchestrator to refresh discovery manifest
```

**Effort**: 3-4 days (implementation + testing)

---

## Summary

### Strengths to Maintain

1. ✅ **Discovery Precomputation**: Keep this architecture; it's a key differentiator
2. ✅ **Governance Model**: Fine-grained budgets prevent runaway operations
3. ✅ **Validation System**: Multi-layer validation catches issues early
4. ✅ **Repo-Memory Integration**: Durable state without external dependencies

### Areas Requiring Improvement

1. ⚠️ **Multi-Repository Support**: Clarify and test cross-repo discovery
2. ⚠️ **Error Recovery**: Document operational procedures for failures
3. ⚠️ **Observability**: Add structured logging and health checks
4. ⚠️ **Worker Contract**: Formalize worker-orchestrator communication
5. ⚠️ **Live Examples**: Create reference campaigns users can learn from

### Implementation Roadmap

**Phase 1 (Critical - 1-2 weeks)**:
- Add live campaign example
- Document error recovery procedures
- Enhance multi-repo discovery

**Phase 2 (Usability - 2-3 weeks)**:
- Add discovery summary logging
- Create worker-orchestrator contract document
- Decide on workflow fusion future

**Phase 3 (Technical - 3-4 weeks)**:
- Implement dispatch tracking
- Schema-enforced governance
- Campaign health checks

---

## Appendix: Testing Coverage Analysis

### Existing Tests

**Campaign Package** (`pkg/campaign/*_test.go`):
- ✅ `orchestrator_test.go` - Orchestrator generation
- ✅ `validation_test.go` - Spec validation
- ✅ `campaign_test.go` - Spec loading
- ✅ `workflow_discovery_test.go` - Workflow discovery
- ✅ `workflow_fusion_test.go` - Workflow fusion
- ✅ `template_test.go` - Prompt template rendering

**CLI Package** (`pkg/cli/*_test.go`):
- ✅ `compile_campaign_validation_test.go` - Campaign validation during compile
- ✅ `compile_campaign_orchestrator_test.go` - Orchestrator compilation

### Test Gaps

**Missing Integration Tests**:
- ❌ End-to-end campaign execution
- ❌ Discovery precomputation with real GitHub API
- ❌ Multi-repository discovery
- ❌ Workflow dispatch and correlation
- ❌ Error recovery scenarios

**Recommended Test Additions**:

```go
// Test: Complete campaign lifecycle
func TestCampaignLifecycle(t *testing.T) {
    // 1. Create campaign spec
    // 2. Compile orchestrator
    // 3. Run discovery
    // 4. Validate manifest
    // 5. Verify project updates
}

// Test: Multi-repo discovery
func TestMultiRepoDiscovery(t *testing.T) {
    // Test allowed-repos and allowed-orgs
}

// Test: Error recovery
func TestDiscoveryRecovery(t *testing.T) {
    // Test cursor reset, partial failures
}
```

---

## Appendix: Architectural Decisions

### ADR-001: Discovery Precomputation

**Status**: Accepted  
**Context**: Agents performing GitHub-wide discovery are expensive and non-deterministic  
**Decision**: Separate discovery into precomputation step before agent execution  
**Consequences**:
- ✅ Deterministic manifests
- ✅ Efficient agent execution
- ⚠️ Added complexity (separate script, manifest format)

### ADR-002: Repo-Memory for State

**Status**: Accepted  
**Context**: Campaigns need durable state across runs  
**Decision**: Use git branches (repo-memory) instead of external database  
**Consequences**:
- ✅ No external dependencies
- ✅ Auditable history
- ⚠️ Repo size grows with metrics
- ⚠️ Git branch management complexity

### ADR-003: Fire-and-Forget Worker Dispatch

**Status**: Accepted  
**Context**: Orchestrators dispatch workers asynchronously  
**Decision**: Don't wait for worker completion; discover results later  
**Consequences**:
- ✅ Orchestrators don't block on workers
- ✅ Workers can run independently
- ⚠️ No immediate failure detection
- ⚠️ Hard to correlate dispatches with items

### ADR-004: Prompt-Based Governance

**Status**: Under Review  
**Context**: Governance limits need enforcement  
**Decision**: Pass limits via prompts, enforce via safe outputs  
**Consequences**:
- ✅ Flexible (easy to change prompts)
- ⚠️ No compile-time validation
- ⚠️ Agent might ignore instructions
- **Recommendation**: Move to schema-enforced constraints

---

**Document Version**: 1.0  
**Last Updated**: 2026-01-19  
**Next Review**: 2026-02-19 or after major campaign feature changes
