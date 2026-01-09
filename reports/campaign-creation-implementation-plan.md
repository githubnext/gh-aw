# Campaign Creation Flow: Implementation Plan

**Status**: Ready for Implementation  
**Created**: 2026-01-09  
**Type**: Refactoring Specification  
**Estimated Effort**: 2-3 weeks  

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Implementation Goals](#implementation-goals)
3. [Architecture Changes](#architecture-changes)
4. [Phase 1: Foundation](#phase-1-foundation)
5. [Phase 2: Consolidation](#phase-2-consolidation)
6. [Phase 3: Future Enhancements](#phase-3-future-enhancements)
7. [Testing Strategy](#testing-strategy)
8. [Rollback Plan](#rollback-plan)
9. [Success Metrics](#success-metrics)

---

## Executive Summary

This document provides a step-by-step implementation plan for refactoring the campaign creation flow to eliminate 52% code duplication (600 lines) across three agent files. The refactoring implements the optimized two-phase architecture documented in the flow diagram.

**Key Changes**:
- Consolidate duplicated campaign design logic into shared prompts
- Implement workflow catalog for deterministic discovery
- Add issue update safe output for transparency
- Optimize phase division: heavy lifting in Phase 1, CLI-only in Phase 2

**Expected Outcomes**:
- 69% code reduction (1,146 → 360 lines)
- 60% faster execution (5-10 min → 2-3 min)
- 67% reduction in maintenance burden (3 files → 1 file to update)

---

## Implementation Goals

### Primary Goals

1. **Eliminate Code Duplication**: Reduce 600 duplicate lines to 0
2. **Improve Maintainability**: Single source of truth for campaign design logic
3. **Optimize Performance**: Move heavy work to Phase 1 (deterministic, fast)
4. **Enhance Transparency**: Issue updates with structured campaign information
5. **Preserve Architecture**: Maintain necessary two-step flow (assign-to-agent required for CLI access)

### Non-Goals

- Changing the fundamental two-phase architecture (required for CLI access)
- Removing CCA agent (remains as optional helper)
- Modifying campaign spec format (`.campaign.md`)
- Changing compiled output (`.campaign.g.md`, `.campaign.lock.yml`)

---

## Architecture Changes

### Current Architecture

```
Issue Created
  ↓
campaign-generator.md (minimal orchestration)
  ├─ Create project board
  ├─ Post status comment
  └─ assign-to-agent
      ↓
agentic-campaign-designer.agent.md (heavy lifting)
  ├─ Scan all workflows (2-3 min)
  ├─ Generate .campaign.md spec
  ├─ Compile campaign
  └─ Create PR
```

**Problems**:
- Duplicate workflow scanning
- Designer agent does both analysis and compilation
- 600 lines of duplicated instructions across 3 files

### Target Architecture

```
Issue Created
  ↓
campaign-generator.md (Agent Step)
  ├─ Parse issue requirements
  ├─ Query workflow catalog (deterministic, <1s)
  ├─ Match workflows by category
  └─ Generate .campaign.md spec
      ↓
campaign-generator.md (Safe Outputs Step)
  ├─ Safe Output 1: Create Project Board
  ├─ Safe Output 2: Post Status Comment
  └─ Safe Output 3: Update Issue
      ├─ Title: [New Agentic Campaign] <name>
      ├─ Body: Quoted prompt + campaign details
      └─ Triggers: assign-to-agent on issue
          ↓
agentic-campaign-designer.agent.md (Copilot coding agent)
  ├─ Run: gh aw compile <campaign-id>
  ├─ Commit files to branch
  └─ Create PR automatically
```

**Improvements**:
- Workflow discovery in Phase 1 (fast, deterministic)
- Spec generation in Phase 1 (no agent needed)
- Issue updates in Phase 1 (transparency)
- Phase 2 reduced to compilation only (60% faster)

---

## Phase 1: Foundation

**Timeline**: Week 1 (5-7 days)  
**Effort**: ~15-20 hours  
**Risk**: Low (additive changes, no breaking changes)

### 1.1 Create Workflow Catalog

**File**: `.github/workflow-catalog.yml`

**Purpose**: Pre-computed workflow metadata for deterministic discovery (eliminates 2-3 min scanning)

**Structure**:
```yaml
# Workflow Catalog for Campaign Creation
# This file enables fast, deterministic workflow discovery
# Update when adding new workflows or changing categorization

version: 1.0

categories:
  security:
    description: "Security scanning, vulnerability detection, and fixes"
    keywords: [security, vulnerability, scan, cve, dependabot, codeql]
    workflows:
      - id: security-scanner
        file: security-scanner.md
        description: "Scans repository for security vulnerabilities"
        triggers: [schedule, workflow_dispatch]
        risk_level: low
      
      - id: security-fix-pr
        file: security-fix-pr.md
        description: "Creates PRs to fix security vulnerabilities"
        triggers: [issue_comment]
        risk_level: medium

  documentation:
    description: "Documentation generation, updates, and maintenance"
    keywords: [docs, documentation, readme, changelog]
    workflows:
      - id: docs-generator
        file: docs-generator.md
        description: "Generates documentation from code"
        triggers: [push, pull_request]
        risk_level: low

  testing:
    description: "Test execution, coverage, and quality assurance"
    keywords: [test, testing, qa, coverage, integration]
    workflows:
      - id: test-runner
        file: test-runner.md
        description: "Runs test suite on PR changes"
        triggers: [pull_request]
        risk_level: low

  maintenance:
    description: "Repository maintenance and housekeeping"
    keywords: [cleanup, maintenance, stale, dependencies]
    workflows:
      - id: dependency-updater
        file: dependency-updater.md
        description: "Updates dependencies weekly"
        triggers: [schedule]
        risk_level: medium

metadata:
  last_updated: "2026-01-09"
  version: "1.0.0"
  maintainer: "github-team"
```

**Implementation Steps**:

1. Create file: `.github/workflow-catalog.yml`
2. Analyze existing workflows in `.github/workflows/`
3. Categorize workflows by purpose and keywords
4. Document workflow metadata (triggers, risk levels, descriptions)
5. Add validation schema (optional but recommended)

**Acceptance Criteria**:
- [ ] File created with all existing workflows categorized
- [ ] Each category has clear description and keywords
- [ ] Each workflow has id, file, description, triggers, risk_level
- [ ] File is valid YAML
- [ ] Documentation explains how to maintain catalog

**Testing**:
- Manually verify YAML is valid: `yamllint .github/workflow-catalog.yml`
- Ensure all workflows in `.github/workflows/` are listed
- Test loading catalog in Go: `parser.ParseWorkflowCatalog()`

---

### 1.2 Create Issue Form Template

**File**: `.github/ISSUE_TEMPLATE/new-agentic-campaign.yml`

**Purpose**: Structured input for campaign creation (replaces CCA as entry point)

**Structure**:
```yaml
name: Create Agentic Campaign
description: Request creation of a new agentic campaign with automated workflows
title: "[New Agentic Campaign] "
labels: ["campaign", "automation"]
body:
  - type: markdown
    attributes:
      value: |
        ## Create Agentic Campaign
        
        This form creates a new agentic campaign that orchestrates multiple workflows toward a common goal.
        
        **What is a campaign?**
        A campaign is a coordinated set of workflows that work together to achieve a specific objective (e.g., "Security Q1 2025").

  - type: input
    id: campaign-name
    attributes:
      label: Campaign Name
      description: "Short, descriptive name for the campaign"
      placeholder: "e.g., Security Q1 2025, Documentation Sprint, Test Coverage Improvement"
    validations:
      required: true

  - type: textarea
    id: campaign-description
    attributes:
      label: Campaign Description
      description: "What problem does this campaign solve? What is the objective?"
      placeholder: |
        Example: Improve repository security by scanning for vulnerabilities weekly, 
        creating fix PRs automatically, and tracking security metrics.
    validations:
      required: true

  - type: textarea
    id: campaign-goals
    attributes:
      label: Campaign Goals
      description: "Specific, measurable goals for this campaign"
      placeholder: |
        - Reduce critical vulnerabilities to 0
        - Achieve 90% test coverage
        - Update all dependencies to latest versions
    validations:
      required: true

  - type: textarea
    id: campaign-kpis
    attributes:
      label: Key Performance Indicators (KPIs)
      description: "How will you measure success?"
      placeholder: |
        - Number of vulnerabilities fixed
        - Test coverage percentage
        - Dependency freshness score
    validations:
      required: false

  - type: dropdown
    id: risk-level
    attributes:
      label: Risk Level
      description: "What level of automation is acceptable?"
      options:
        - Low (read-only, reporting, notifications)
        - Medium (issues, PRs, comments, but requires approval)
        - High (automated merges, deployments, requires careful review)
    validations:
      required: true

  - type: textarea
    id: workflows
    attributes:
      label: Suggested Workflows (Optional)
      description: "List existing workflows or describe new workflows needed"
      placeholder: |
        Existing workflows:
        - security-scanner
        - security-fix-pr
        
        New workflows needed:
        - security-reporter (weekly summary of findings)
    validations:
      required: false

  - type: input
    id: timeline
    attributes:
      label: Timeline (Optional)
      description: "How long should this campaign run?"
      placeholder: "e.g., Q1 2025 (Jan-Mar), 4 weeks, ongoing"
    validations:
      required: false

  - type: textarea
    id: additional-context
    attributes:
      label: Additional Context
      description: "Any other relevant information"
    validations:
      required: false
```

**Implementation Steps**:

1. Create directory: `.github/ISSUE_TEMPLATE/` (if not exists)
2. Create file: `new-agentic-campaign.yml`
3. Add all fields as shown above
4. Test form in GitHub UI
5. Update documentation to reference issue form

**Acceptance Criteria**:
- [ ] File created with all fields
- [ ] Form appears in GitHub's "New Issue" menu
- [ ] All required fields enforce validation
- [ ] Form provides helpful descriptions and examples
- [ ] Submitted issues have structured, parseable content

**Testing**:
- Test form in GitHub UI: Create new issue → Select template
- Submit test issue and verify all fields are captured
- Verify issue title format: `[New Agentic Campaign] Test Campaign`

---

### 1.3 Add Issue Update Safe Output

**File**: `pkg/workflow/safe_outputs.go`

**Purpose**: Add `update-issue` safe output to update issue title and body with campaign details

**Implementation**:

```go
// UpdateIssue updates an existing issue's title and body
type UpdateIssue struct {
	Number int    `yaml:"number,omitempty"`    // Issue number (optional if context)
	Title  string `yaml:"title,omitempty"`     // New title
	Body   string `yaml:"body,omitempty"`      // New body content
	Append bool   `yaml:"append,omitempty"`    // Append to existing body vs replace
}

// Validate ensures UpdateIssue configuration is valid
func (u *UpdateIssue) Validate() error {
	if u.Number < 0 {
		return fmt.Errorf("issue number must be positive")
	}
	
	if u.Title == "" && u.Body == "" {
		return fmt.Errorf("must specify at least one of title or body to update")
	}
	
	// Title length validation
	if len(u.Title) > 256 {
		return fmt.Errorf("title length exceeds maximum of 256 characters")
	}
	
	// Body length validation
	if len(u.Body) > 65536 {
		return fmt.Errorf("body length exceeds maximum of 65536 characters")
	}
	
	return nil
}
```

**GitHub Actions Job** (`actions/update-issue/action.yml`):

```yaml
name: Update Issue
description: Updates an existing issue's title and body

inputs:
  github-token:
    description: 'GitHub token with issues:write permission'
    required: true
  issue-number:
    description: 'Issue number to update'
    required: true
  title:
    description: 'New issue title (leave empty to keep current)'
    required: false
  body:
    description: 'New issue body (leave empty to keep current)'
    required: false
  append:
    description: 'Append to existing body instead of replacing'
    required: false
    default: 'false'

runs:
  using: composite
  steps:
    - name: Update Issue
      uses: actions/github-script@v7
      with:
        github-token: ${{ inputs.github-token }}
        script: |
          const issueNumber = parseInt('${{ inputs.issue-number }}');
          const newTitle = '${{ inputs.title }}';
          const newBody = '${{ inputs.body }}';
          const append = '${{ inputs.append }}' === 'true';
          
          // Get current issue
          const { data: issue } = await github.rest.issues.get({
            owner: context.repo.owner,
            repo: context.repo.repo,
            issue_number: issueNumber
          });
          
          // Build update payload
          const updateData = {
            owner: context.repo.owner,
            repo: context.repo.repo,
            issue_number: issueNumber
          };
          
          if (newTitle) {
            updateData.title = newTitle;
          }
          
          if (newBody) {
            if (append) {
              updateData.body = issue.body + '\n\n' + newBody;
            } else {
              updateData.body = newBody;
            }
          }
          
          // Update issue
          await github.rest.issues.update(updateData);
          
          console.log(`Updated issue #${issueNumber}`);
```

**Implementation Steps**:

1. Add `UpdateIssue` struct to `pkg/workflow/safe_outputs.go`
2. Implement `Validate()` method
3. Add to `SafeOutputs` struct as `UpdateIssue *UpdateIssue`
4. Create action: `actions/update-issue/action.yml`
5. Update compilation to generate update-issue job
6. Add tests for update-issue validation

**Acceptance Criteria**:
- [ ] `UpdateIssue` struct added with validation
- [ ] Action created and tested
- [ ] Compilation generates correct GitHub Actions YAML
- [ ] Title and body updates work correctly
- [ ] Append mode works correctly
- [ ] Tests pass for valid and invalid configurations

**Testing**:
- Unit tests: `TestUpdateIssueValidation`
- Integration test: Create issue, update via safe output, verify changes
- Test append mode: Update adds to existing content

---

### 1.4 Update Campaign Generator Workflow

**File**: `.github/workflows/campaign-generator.md`

**Purpose**: Add workflow catalog query, spec generation, and issue update to Phase 1

**Changes**:

1. **Add Workflow Catalog Query** (Agent Step):

```markdown
## Agent Instructions

You are the campaign generator. Your job is to:

1. **Parse Issue Requirements**
   - Extract campaign name, description, goals, KPIs from issue body
   - Parse issue form fields (created via GitHub issue template)
   - Validate all required fields are present

2. **Query Workflow Catalog** (Deterministic Discovery)
   - Load `.github/workflow-catalog.yml`
   - Match user's campaign description and goals to workflow categories
   - Use keywords to find relevant workflows
   - Select workflows based on:
     - Category alignment with campaign goals
     - Risk level compatibility
     - Trigger types (schedule, event, manual)
   
   Example matching logic:
   - Campaign mentions "security" → Look in security category
   - Goals include "vulnerability scanning" → Match security-scanner workflow
   - Risk level "Low" → Only include low-risk workflows

3. **Generate .campaign.md Spec**
   - Create campaign spec file with:
     - Campaign metadata (name, description, goals, KPIs)
     - Matched workflows with configuration
     - Governance rules based on risk level
     - Project board configuration
   - Save to `.github/campaigns/<campaign-id>.campaign.md`

## Safe Outputs

After generating the spec, execute these safe outputs:

```yaml
safe-outputs:
  # 1. Create project board for campaign tracking
  create-project:
    title: "Campaign: {{campaign_name}}"
    description: "{{campaign_description}}"
    github-token: "${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}"
    columns:
      - "Backlog"
      - "In Progress"
      - "Completed"

  # 2. Post status comment
  add-comment:
    body: |
      🤖 **Campaign Creation Started**
      
      📊 Project Board: [Campaign: {{campaign_name}}]({{project_board_url}})
      ⏱️ Estimated time: 2-3 minutes
      
      **Progress**:
      - [x] Project board created
      - [x] Workflows identified: {{workflow_count}}
      - [x] Campaign spec generated
      - [ ] Compiling orchestrator...
      - [ ] Creating pull request...

  # 3. Update issue with campaign details
  update-issue:
    title: "[New Agentic Campaign] {{campaign_name}}"
    body: |
      > **Original Request**:
      > {{original_issue_body}}
      
      ---
      
      ## 📋 Campaign Details
      
      **Name**: {{campaign_name}}
      **Description**: {{campaign_description}}
      
      **Project Board**: [View Board]({{project_board_url}})
      
      ### Goals
      {{campaign_goals}}
      
      ### Key Performance Indicators (KPIs)
      {{campaign_kpis}}
      
      ### Workflows Selected
      {{#each workflows}}
      - **{{this.id}}**: {{this.description}}
      {{/each}}
      
      ### Timeline
      {{campaign_timeline}}
      
      ### Risk Level
      {{risk_level}}
      
      ---
      
      **Status**: ⏳ Compilation in progress...
      **PR**: Will be created automatically when ready
```

**Implementation Steps**:

1. Update agent instructions in `campaign-generator.md`
2. Add workflow catalog query logic
3. Add spec generation logic
4. Add safe outputs (create-project, add-comment, update-issue)
5. Remove assign-to-agent from generator (it's now on the issue)
6. Test end-to-end flow

**Acceptance Criteria**:
- [ ] Generator queries workflow catalog successfully
- [ ] Workflow matching uses keywords and categories
- [ ] Spec generation creates valid `.campaign.md` file
- [ ] Safe outputs execute in correct order
- [ ] Issue updated with formatted campaign details
- [ ] Project board created successfully

**Testing**:
- Create test issue with campaign request
- Verify workflow catalog is queried
- Verify workflows are matched correctly
- Verify spec file is created
- Verify issue is updated with details
- Verify project board is created

---

### 1.5 Configure assign-to-agent on Issue

**Purpose**: Trigger Copilot coding agent session from updated issue (not as safe output)

**Implementation**:

The `assign-to-agent` is configured on the issue itself (via issue labels or issue body parsing), not as a safe output from the generator workflow.

**Option A: Issue Label Trigger**

Add label `assign-to-agent` to issue after update:

```yaml
# In campaign-generator.md safe outputs
update-issue:
  title: "[New Agentic Campaign] {{campaign_name}}"
  body: "..."
  labels:
    - "assign-to-agent"
    - "campaign"
```

**Option B: Issue Body Trigger**

Add special marker in issue body that triggers agent assignment:

```markdown
---

**Status**: ⏳ Compilation in progress...

<!-- assign-to-agent: agentic-campaign-designer -->
```

**Option C: Workflow Dispatch from Issue**

Generator workflow dispatches `workflow_dispatch` event that triggers designer:

```yaml
# In campaign-generator.md (final step)
- name: Trigger Designer Agent
  uses: actions/github-script@v7
  with:
    script: |
      await github.rest.actions.createWorkflowDispatch({
        owner: context.repo.owner,
        repo: context.repo.repo,
        workflow_id: 'agentic-campaign-designer.md',
        ref: 'main',
        inputs: {
          issue_number: context.issue.number,
          campaign_id: '${{ steps.generate.outputs.campaign_id }}'
        }
      });
```

**Recommended**: Option C (Workflow Dispatch) for explicit control

**Implementation Steps**:

1. Add workflow_dispatch trigger to `agentic-campaign-designer.agent.md`
2. Add dispatch step to end of `campaign-generator.md`
3. Pass issue number and campaign ID as inputs
4. Update designer to receive inputs

**Acceptance Criteria**:
- [ ] Designer agent is triggered after issue update
- [ ] Campaign ID and issue number are passed correctly
- [ ] Designer can access campaign spec file
- [ ] Handoff is reliable and trackable

---

### Phase 1 Deliverables

- [ ] `.github/workflow-catalog.yml` created and populated
- [ ] `.github/ISSUE_TEMPLATE/new-agentic-campaign.yml` created
- [ ] `update-issue` safe output implemented
- [ ] `campaign-generator.md` updated with catalog query and spec generation
- [ ] `assign-to-agent` trigger configured (workflow dispatch)
- [ ] End-to-end flow tested: Issue → Generator → Issue Update → Designer Trigger
- [ ] Documentation updated

**Phase 1 Testing Checklist**:
- [ ] Create test campaign via issue form
- [ ] Verify workflow catalog is loaded
- [ ] Verify workflows are matched correctly
- [ ] Verify spec file is generated
- [ ] Verify issue is updated with details
- [ ] Verify project board is created
- [ ] Verify designer agent is triggered
- [ ] Verify entire flow completes successfully

---

## Phase 2: Consolidation

**Timeline**: Week 2-3 (10-14 days)  
**Effort**: ~25-30 hours  
**Risk**: Medium (refactoring existing code, maintaining backward compatibility)

### 2.1 Create Shared Campaign Prompts

**File**: `pkg/campaign/prompts/campaign_creation_instructions.md`

**Purpose**: Single source of truth for campaign design logic (eliminates 600 lines of duplication)

**Content Structure**:

```markdown
# Campaign Creation Instructions

This document contains shared instructions for campaign creation, used by multiple agents to ensure consistency.

## Workflow Identification Strategies

### Category-Based Matching

Match workflows to campaign goals using the workflow catalog:

1. **Parse campaign goals and description**
   - Extract key themes (security, testing, documentation, etc.)
   - Identify specific requirements (scanning, reporting, automation level)

2. **Query workflow catalog by category**
   - Load `.github/workflow-catalog.yml`
   - Match campaign themes to workflow categories
   - Use keyword matching for precision

3. **Filter by risk level**
   - Low risk: Read-only, reporting, notifications
   - Medium risk: Issues, PRs, requires approval
   - High risk: Automated merges, deployments

4. **Select workflows**
   - Choose 2-5 workflows per campaign (focused scope)
   - Ensure workflows complement each other
   - Avoid redundant workflows

### Example Matching

**Campaign**: "Security Q1 2025"
- **Goals**: Scan for vulnerabilities, create fix PRs, track metrics
- **Matched Category**: security
- **Workflows Selected**:
  - security-scanner (low risk, scanning)
  - security-fix-pr (medium risk, automated PRs)
  - security-reporter (low risk, reporting)

---

## Campaign File Structure

### Campaign Spec Format

Every campaign has a `.campaign.md` file with this structure:

```yaml
---
name: "Campaign Name"
description: "Brief description"
goals:
  - "Goal 1"
  - "Goal 2"
kpis:
  - "KPI 1"
  - "KPI 2"
risk_level: "low|medium|high"
timeline: "Q1 2025"
workflows:
  - id: workflow-1
    config:
      schedule: "0 0 * * 0"  # Weekly
  - id: workflow-2
    config:
      on_demand: true
governance:
  approval_required: true|false
  allowed_actions: [create_issue, create_pr]
project:
  board_id: "PVT_..."
  columns: [Backlog, In Progress, Completed]
---

# Campaign: {{name}}

{{description}}

## Objectives

{{goals}}

## Success Metrics

{{kpis}}

## Workflows

{{workflow_details}}
```

---

## Safe Output Patterns

### Project Board Creation

Always create a project board for campaign tracking:

```yaml
safe-outputs:
  create-project:
    title: "Campaign: {{name}}"
    description: "{{description}}"
    github-token: "${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}"
    columns: ["Backlog", "In Progress", "Completed"]
```

### Issue Updates

Update the campaign issue with structured details:

```yaml
safe-outputs:
  update-issue:
    title: "[New Agentic Campaign] {{name}}"
    body: |
      > **Original Request**: {{original_body}}
      
      ## Campaign Details
      - **Name**: {{name}}
      - **Board**: [Link]({{board_url}})
      - **Workflows**: {{workflow_count}}
```

### Status Comments

Post progress updates to the issue:

```yaml
safe-outputs:
  add-comment:
    body: |
      🤖 **Status Update**
      - [x] Workflows identified: {{workflows}}
      - [x] Spec generated
      - [ ] Compiling...
```

---

## Governance Policies

### Risk-Based Governance

**Low Risk Campaigns**:
- Read-only operations only
- Reporting and notifications
- No approval required
- Allowed actions: add_comment, create_issue (read-only)

**Medium Risk Campaigns**:
- Can create issues and PRs
- Requires approval before merge
- Allowed actions: create_issue, create_pr, add_comment
- Must have approval_required: true

**High Risk Campaigns**:
- Can perform automated merges
- Critical review required
- Allowed actions: all
- Must have explicit sign-off from maintainers
- Additional security checks

### Approval Rules

```yaml
governance:
  approval_required: true
  approvers:
    - github-team
  required_approvals: 2
  allowed_actions:
    - create_issue
    - create_pr
    - add_comment
```

---

## Project Board Configuration

### Standard Columns

Every campaign project board should have:

1. **Backlog**: Planned work items
2. **In Progress**: Currently active workflows
3. **Completed**: Finished items

### Automation Rules

```yaml
project:
  automation:
    issue_added: "Backlog"
    pr_opened: "In Progress"
    pr_merged: "Completed"
    issue_closed: "Completed"
```

---

## Risk Assessment Rules

### Workflow Risk Evaluation

Evaluate each workflow's risk level:

1. **Read-only operations**: Low risk
   - Scanning, reporting, notifications
   - No repository modifications

2. **Write operations with approval**: Medium risk
   - Creating issues, PRs
   - Requires review before merge

3. **Automated modifications**: High risk
   - Automated merges, deployments
   - Requires strict governance

### Campaign Risk Level

Campaign risk = Maximum risk of included workflows

- If ANY workflow is high risk → Campaign is high risk
- If ANY workflow is medium risk → Campaign is medium risk
- If ALL workflows are low risk → Campaign is low risk

---

## Error Handling

### Missing Workflows

If requested workflow doesn't exist:

```
❌ Workflow "non-existent-workflow" not found in catalog
💡 Available workflows in category: [list]
💡 Suggestion: Did you mean "similar-workflow"?
```

### Invalid Configuration

If campaign config is invalid:

```
❌ Invalid campaign configuration
- Risk level "high" requires approval_required: true
- Timeline format should be "Q1 2025" or "YYYY-MM-DD"
```

### Compilation Errors

If campaign fails to compile:

```
❌ Compilation failed
Error: Missing required field "name" in campaign spec
📝 Please update .campaign.md and try again
```

---

## Compilation Best Practices

### Pre-Compilation Checks

Before compiling:

1. Validate campaign spec against schema
2. Verify all referenced workflows exist
3. Check risk level consistency
4. Ensure governance rules are complete

### Compilation Command

```bash
gh aw compile <campaign-id>
```

This generates:
- `.campaign.g.md` (generated markdown)
- `.campaign.lock.yml` (locked YAML)

### Post-Compilation

After compilation:

1. Verify generated files are valid
2. Check for compilation warnings
3. Run validation tests
4. Create PR with all 3 files (.campaign.md, .campaign.g.md, .campaign.lock.yml)
```

**Implementation Steps**:

1. Create directory: `pkg/campaign/prompts/`
2. Create file: `campaign_creation_instructions.md`
3. Extract common logic from 3 agent files
4. Consolidate into single document
5. Add examples and error handling
6. Update agents to import this file

**Acceptance Criteria**:
- [ ] File created with all shared instructions
- [ ] All duplicated logic extracted from agent files
- [ ] Instructions are clear and complete
- [ ] Examples provided for each section
- [ ] Error handling documented

---

### 2.2 Update Campaign Generator

**File**: `.github/workflows/campaign-generator.md`

**Purpose**: Import shared instructions instead of duplicating them

**Changes**:

```markdown
---
description: Campaign generator - orchestrates campaign creation
on:
  issues:
    types: [opened, reopened]
engine: copilot
safe-outputs:
  create-project:
    github-token: "${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}"
  add-comment: {}
  update-issue: {}
---

{{#runtime-import pkg/campaign/prompts/campaign_creation_instructions.md}}

# Campaign Generator

You are the campaign generator workflow. Your responsibilities:

1. **Parse issue created via issue form template**
   - Extract: name, description, goals, KPIs, risk level, timeline
   - Validate required fields are present

2. **Query workflow catalog** (see Campaign Creation Instructions above)
   - Load `.github/workflow-catalog.yml`
   - Match workflows to campaign goals
   - Filter by risk level

3. **Generate campaign spec** (see Campaign File Structure above)
   - Create `.github/campaigns/<campaign-id>.campaign.md`
   - Include all matched workflows
   - Apply governance rules based on risk level

4. **Execute safe outputs**:
   - Create project board
   - Post status comment
   - Update issue with campaign details

## Implementation

Use the shared Campaign Creation Instructions above for:
- Workflow identification strategies
- Campaign file structure
- Safe output patterns
- Governance policies
- Risk assessment rules

Do NOT duplicate any logic from the shared instructions - reference them instead.
```

**Implementation Steps**:

1. Add `{{#runtime-import}}` directive to generator
2. Remove all duplicated logic (keep only generator-specific code)
3. Reference shared instructions for common patterns
4. Test that import works correctly
5. Verify behavior is unchanged

**Acceptance Criteria**:
- [ ] Import directive works
- [ ] Duplicated logic removed (400 lines → 40 lines)
- [ ] Generator behavior unchanged
- [ ] Tests pass

---

### 2.3 Update Designer Agent

**File**: `.github/agents/agentic-campaign-designer.agent.md`

**Purpose**: Import shared instructions, focus only on compilation

**Changes**:

```markdown
---
description: Campaign designer - compiles campaign files
---

{{#runtime-import pkg/campaign/prompts/campaign_creation_instructions.md}}

# Campaign Designer

You are the campaign designer agent (Copilot coding agent session). Your sole responsibility is:

## Compile Campaign

1. **Locate campaign spec** from issue context
   - Issue number provided via workflow input
   - Campaign spec at `.github/campaigns/<campaign-id>.campaign.md`

2. **Run compilation**:
   ```bash
   gh aw compile <campaign-id>
   ```

3. **Verify generated files**:
   - `.github/campaigns/<campaign-id>.campaign.g.md`
   - `.github/campaigns/<campaign-id>.campaign.lock.yml`

4. **Commit and create PR**:
   - Add all 3 files (.campaign.md, .campaign.g.md, .campaign.lock.yml)
   - Commit message: "feat: Add <campaign-name> campaign"
   - PR title: "Campaign: <campaign-name>"
   - PR body: Include campaign details from spec

## Notes

- You have access to `gh aw` CLI (provided by actions/setup)
- Copilot coding agent sessions create PRs automatically
- No safe outputs needed - you have direct git access
- Focus ONLY on compilation - spec generation is done in Phase 1

## Error Handling

If compilation fails:
- Add comment to issue with error details
- Do NOT create PR
- Exit with error

See Campaign Creation Instructions above for:
- Campaign file structure
- Risk assessment rules
- Governance policies
```

**Implementation Steps**:

1. Add `{{#runtime-import}}` directive
2. Remove all duplicated logic (200 lines → 60 lines)
3. Focus solely on compilation
4. Remove workflow scanning (done in Phase 1)
5. Remove spec generation (done in Phase 1)
6. Test compilation-only flow

**Acceptance Criteria**:
- [ ] Import directive works
- [ ] Duplicated logic removed (200 lines → 60 lines)
- [ ] Designer only compiles (no scanning or spec generation)
- [ ] PR creation works automatically
- [ ] Tests pass

---

### 2.4 Update Template File

**File**: `pkg/cli/templates/agentic-campaign-designer.agent.md`

**Purpose**: Remove (it's now a duplicate of the agent file)

**Changes**:

This file is 100% duplicate of `.github/agents/agentic-campaign-designer.agent.md`. After updating the agent file:

**Option A**: Delete template file

**Option B**: Make it a symlink

```bash
cd pkg/cli/templates
rm agentic-campaign-designer.agent.md
ln -s ../../../.github/agents/agentic-campaign-designer.agent.md .
```

**Option C**: Keep for backward compatibility but mark as deprecated

```markdown
---
description: DEPRECATED - Use .github/agents/agentic-campaign-designer.agent.md
---

**⚠️ DEPRECATED**: This file is deprecated and will be removed in a future version.

Use `.github/agents/agentic-campaign-designer.agent.md` instead.

This file is kept for backward compatibility only.
```

**Recommended**: Option A (delete) after ensuring no code references it

**Implementation Steps**:

1. Search codebase for references to template file
2. Update references to use agent file directly
3. Delete template file
4. Update tests
5. Document removal in changelog

**Acceptance Criteria**:
- [ ] No code references template file
- [ ] Template file deleted or deprecated
- [ ] Tests updated
- [ ] Documentation updated

---

### 2.5 Remove CCA Agent (Optional)

**File**: `.github/agents/create-agentic-campaign.agent.md`

**Purpose**: Evaluate if CCA agent is still needed

**Analysis**:

**Current CCA Responsibilities**:
1. Conversational requirement gathering
2. Workflow suggestions
3. Creates issue to trigger generator

**With New Flow**:
1. Users create issues directly via issue form ✅
2. Workflow catalog provides suggestions ✅
3. No need for CCA to create issue ✅

**Decision**: CCA agent is now **optional** - can be removed or repurposed

**Option A**: Remove CCA entirely

Users use issue form directly. CCA is not needed.

**Option B**: Keep CCA as conversational helper

CCA helps users understand campaigns and guides them to create issue, but doesn't create issue itself:

```markdown
# Create Agentic Campaign Agent

{{#runtime-import pkg/campaign/prompts/campaign_creation_instructions.md}}

## Your Role

You are a helpful assistant that helps users create agentic campaigns. You:

1. **Explain what campaigns are** and provide examples
2. **Help users define goals and KPIs**
3. **Suggest workflows** based on user's needs (using workflow catalog)
4. **Guide users to create issue** using the issue form template

## What You DON'T Do

- You do NOT create the issue yourself
- You do NOT directly trigger the campaign generator
- You do NOT scan workflows (use catalog instead)

## Workflow

1. User asks about creating campaign
2. You help them define requirements
3. You suggest workflows from catalog
4. You provide link to issue form: "Go to Issues → New Issue → Create Agentic Campaign"
5. User creates issue
6. Campaign generator takes over

This is a conversational assistant only - the issue form is the actual entry point.
```

**Recommended**: Option B (keep as helper, but simplified)

**Implementation Steps**:

1. Decide: Remove or repurpose CCA
2. If remove: Delete file, update docs
3. If repurpose: Update instructions, remove issue creation
4. Add `{{#runtime-import}}` directive
5. Test both flows: Direct issue creation, CCA-assisted creation

**Acceptance Criteria**:
- [ ] Decision made: Remove or repurpose
- [ ] If repurpose: CCA updated with shared instructions
- [ ] If remove: File deleted, docs updated
- [ ] Both entry points tested

---

### Phase 2 Deliverables

- [ ] `pkg/campaign/prompts/campaign_creation_instructions.md` created
- [ ] All duplicated logic consolidated (600 lines → 0)
- [ ] `campaign-generator.md` imports shared instructions
- [ ] `agentic-campaign-designer.agent.md` imports shared instructions
- [ ] Template file handled (deleted or deprecated)
- [ ] CCA agent decision made and implemented
- [ ] All agents use shared instructions
- [ ] Zero code duplication achieved
- [ ] Tests updated and passing
- [ ] Documentation updated

**Phase 2 Testing Checklist**:
- [ ] Import directives work correctly
- [ ] Campaign generator behavior unchanged
- [ ] Designer agent compiles successfully
- [ ] No duplicate logic in agent files
- [ ] End-to-end flow works: Issue → Generator → Designer → PR
- [ ] All tests pass
- [ ] Performance improved (60% faster execution)

---

## Phase 3: Future Enhancements

**Timeline**: Future (post-initial implementation)  
**Effort**: ~20-30 hours  
**Risk**: Low (purely additive features)

### 3.1 Advanced UX Optimizations

**Features**:

1. **Dry-Run Mode**
   - Add `dry-run` label to issue
   - Generator creates spec but doesn't trigger designer
   - Posts preview comment with campaign details
   - User can review and approve before compilation

2. **Webhook Notifications**
   - Add webhook URL to issue form
   - Post notifications to Slack/Teams/Discord
   - Events: Campaign created, PR ready, compilation failed

3. **Performance Metrics**
   - Track campaign creation times
   - Store metrics in repo-memory
   - Generate performance reports
   - Identify bottlenecks

4. **Workflow Health Checks**
   - Validate workflow files before including in campaign
   - Check for syntax errors
   - Verify workflow can compile
   - Prevent broken campaigns

### 3.2 Enhanced Workflow Catalog

**Features**:

1. **Automatic Catalog Updates**
   - Workflow for scanning `.github/workflows/`
   - Automatically updates catalog when workflows change
   - Suggests categories for new workflows

2. **Catalog Validation**
   - JSON schema for catalog format
   - CI job validates catalog on PR
   - Prevents invalid catalog entries

3. **Workflow Recommendations**
   - ML-based workflow suggestions
   - Based on repository characteristics
   - Historical campaign data

### 3.3 Campaign Analytics

**Features**:

1. **Campaign Dashboard**
   - Web UI showing all active campaigns
   - Progress tracking
   - KPI visualization

2. **Success Metrics**
   - Track campaign goal completion
   - Measure KPI achievement
   - Generate success reports

3. **Trend Analysis**
   - Compare campaigns over time
   - Identify successful patterns
   - Suggest optimizations

---

## Testing Strategy

### Unit Tests

**Files to Test**:
- `pkg/workflow/safe_outputs.go` (UpdateIssue validation)
- `pkg/parser/workflow_catalog.go` (catalog parsing)
- `pkg/campaign/prompts_test.go` (shared instructions)

**Test Cases**:
```go
func TestUpdateIssueValidation(t *testing.T) {
	tests := []struct {
		name      string
		input     UpdateIssue
		expectErr bool
	}{
		{
			name: "valid update with title and body",
			input: UpdateIssue{
				Number: 123,
				Title:  "New Title",
				Body:   "New Body",
			},
			expectErr: false,
		},
		{
			name: "invalid - no title or body",
			input: UpdateIssue{
				Number: 123,
			},
			expectErr: true,
		},
		{
			name: "invalid - title too long",
			input: UpdateIssue{
				Number: 123,
				Title:  strings.Repeat("a", 257),
			},
			expectErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if tt.expectErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestWorkflowCatalogParsing(t *testing.T) {
	catalog := `
version: 1.0
categories:
  security:
    keywords: [security, vulnerability]
    workflows:
      - id: security-scanner
        file: security-scanner.md
        description: "Scans for vulnerabilities"
`
	
	result, err := ParseWorkflowCatalog([]byte(catalog))
	if err != nil {
		t.Fatalf("failed to parse catalog: %v", err)
	}
	
	if len(result.Categories) != 1 {
		t.Errorf("expected 1 category, got %d", len(result.Categories))
	}
	
	securityCat := result.Categories["security"]
	if len(securityCat.Workflows) != 1 {
		t.Errorf("expected 1 workflow, got %d", len(securityCat.Workflows))
	}
}
```

### Integration Tests

**Test Scenarios**:

1. **End-to-End Campaign Creation**
   ```bash
   # Create issue via API
   gh issue create --title "[New Agentic Campaign] Test" --body "..."
   
   # Verify generator runs
   # Verify workflow catalog is queried
   # Verify spec is generated
   # Verify issue is updated
   # Verify designer is triggered
   # Verify PR is created
   ```

2. **Workflow Catalog Query**
   ```bash
   # Test matching logic
   # Input: Campaign with "security" in description
   # Expected: Security workflows matched
   # Input: Campaign with "testing" in goals
   # Expected: Testing workflows matched
   ```

3. **Issue Update**
   ```bash
   # Create test issue
   # Trigger update-issue safe output
   # Verify title updated
   # Verify body updated
   # Verify append mode works
   ```

### Manual Testing

**Test Checklist**:

- [ ] Create campaign via issue form (GitHub UI)
- [ ] Verify form validation works
- [ ] Verify all fields captured correctly
- [ ] Verify generator workflow triggers
- [ ] Verify workflow catalog is loaded
- [ ] Verify workflows matched to campaign goals
- [ ] Verify campaign spec generated correctly
- [ ] Verify project board created
- [ ] Verify issue updated with details
- [ ] Verify issue title formatted correctly
- [ ] Verify designer agent triggered
- [ ] Verify compilation succeeds
- [ ] Verify PR created automatically
- [ ] Verify PR contains all 3 files
- [ ] Verify PR title and body correct
- [ ] Test error cases (invalid input, missing workflows, compilation failure)

---

## Rollback Plan

### If Issues Arise During Implementation

**Phase 1 Rollback**:

1. Revert workflow-catalog.yml
2. Revert issue form template
3. Revert update-issue safe output
4. Revert campaign-generator changes
5. Restore original generator logic

**Phase 2 Rollback**:

1. Revert shared instructions import
2. Restore duplicated logic in agent files
3. Restore template file
4. Restore original CCA agent

### Feature Flags

**Gradual Rollout**:

Add feature flag to enable new flow:

```yaml
# In campaign-generator.md
features:
  use_workflow_catalog: true
  use_shared_instructions: true
```

If `false`, use old logic. Allows A/B testing and gradual rollout.

### Backward Compatibility

**Maintain for 1 release**:

- Keep old CCA agent working
- Support both entry points: CCA and issue form
- Deprecation warnings in old agents
- Remove in next major version

---

## Success Metrics

### Code Metrics

| Metric | Before | After | Target |
|--------|--------|-------|--------|
| **Total Lines** | 1,146 | 360 | 69% reduction |
| **Duplicate Lines** | 600 | 0 | 100% elimination |
| **Files to Update** | 3 | 1 | 67% reduction |
| **Update Time** | 15-20 min | 3-5 min | 75% faster |

### Performance Metrics

| Metric | Before | After | Target |
|--------|--------|-------|--------|
| **Phase 1 Time** | ~30s | ~30s | Same |
| **Phase 2 Time** | 5-10 min | 1-2 min | 70% faster |
| **Total Execution** | 5-10 min | 2-3 min | 60% faster |
| **Workflow Scanning** | 2-3 min | <1s | 99% faster |

### Quality Metrics

| Metric | Before | After | Target |
|--------|--------|-------|--------|
| **Code Drift Risk** | High | Zero | Eliminated |
| **Maintenance Burden** | 3 files | 1 file | 67% reduction |
| **Test Coverage** | ~70% | >90% | 20% increase |
| **Documentation** | Partial | Complete | 100% |

### User Experience Metrics

| Metric | Before | After | Target |
|--------|--------|-------|--------|
| **Entry Point Clarity** | Unclear | Clear | Improved |
| **Transparency** | Low | High | Issue updates |
| **Observability** | Limited | Comprehensive | Dashboard |
| **Error Messages** | Generic | Specific | Actionable |

---

## Acceptance Criteria

### Phase 1 Complete When

- [ ] Workflow catalog created and populated with all workflows
- [ ] Issue form template created with all required fields
- [ ] update-issue safe output implemented and tested
- [ ] campaign-generator.md updated with catalog query and spec generation
- [ ] assign-to-agent trigger configured (workflow dispatch)
- [ ] End-to-end flow works: Issue → Generator → Issue Update → Designer
- [ ] All Phase 1 tests pass
- [ ] Documentation complete

### Phase 2 Complete When

- [ ] Shared instructions created in pkg/campaign/prompts/
- [ ] All duplicate logic consolidated (600 lines → 0)
- [ ] campaign-generator.md imports shared instructions
- [ ] agentic-campaign-designer.agent.md imports shared instructions
- [ ] Template file handled (deleted/deprecated)
- [ ] CCA agent decision implemented
- [ ] Zero code duplication verified
- [ ] All Phase 2 tests pass
- [ ] Performance targets met (60% faster)
- [ ] Documentation complete

### Overall Success Criteria

- [ ] 69% code reduction achieved (1,146 → 360 lines)
- [ ] 100% duplicate elimination (600 lines → 0)
- [ ] 60% performance improvement (5-10 min → 2-3 min)
- [ ] 67% maintenance reduction (3 files → 1 file)
- [ ] All tests pass (unit, integration, manual)
- [ ] Documentation complete and reviewed
- [ ] No regressions in functionality
- [ ] User experience improved (transparency, observability)
- [ ] Code review approved
- [ ] Stakeholder sign-off

---

## Implementation Timeline

### Week 1: Phase 1 Foundation

**Days 1-2**:
- Create workflow catalog
- Create issue form template

**Days 3-4**:
- Implement update-issue safe output
- Add to campaign-generator

**Days 5-7**:
- Test end-to-end flow
- Fix bugs
- Update documentation

### Week 2-3: Phase 2 Consolidation

**Days 8-10**:
- Create shared instructions
- Extract duplicated logic

**Days 11-13**:
- Update all agents with imports
- Remove duplicate code
- Handle template file

**Days 14-16**:
- Test consolidated flow
- Performance testing
- Bug fixes

**Days 17-18**:
- Final testing
- Documentation review
- Code review

**Day 19-20**:
- Stakeholder review
- Deploy to production

---

## Questions & Decisions

### Open Questions

1. **Workflow catalog format**: YAML vs JSON?
   - **Recommendation**: YAML (more human-readable)

2. **assign-to-agent trigger**: Label, body marker, or workflow dispatch?
   - **Recommendation**: Workflow dispatch (explicit, reliable)

3. **CCA agent**: Remove, keep, or repurpose?
   - **Recommendation**: Repurpose as conversational helper

4. **Template file**: Delete, symlink, or deprecate?
   - **Recommendation**: Delete after migration

5. **Backward compatibility**: How long to maintain?
   - **Recommendation**: 1 release cycle (2-3 months)

### Decisions Made

- [x] Use workflow catalog for deterministic discovery
- [x] Move heavy work to Phase 1 (Agent step)
- [x] Use issue updates for transparency
- [x] Consolidate logic into shared prompts
- [x] Preserve two-phase architecture (CLI requirement)
- [x] Issue form as primary entry point
- [x] Workflow dispatch for agent trigger

---

## References

- [Campaign Creation Flow Analysis](./campaign-creation-flow-analysis.md)
- [Campaign Creation Flow Summary](./campaign-creation-flow-summary.md)
- [Campaign Flow Visual Comparison](./campaign-flow-visual-comparison.md)
- [Flow Diagram](./campaign-creation-flow-analysis.md#flow-diagram) (Mermaid)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Safe Outputs Documentation](../pkg/workflow/safe_outputs.go)

---

## Appendix

### File Structure After Implementation

```
.github/
├── ISSUE_TEMPLATE/
│   └── new-agentic-campaign.yml          # NEW: Issue form template
├── workflows/
│   ├── campaign-generator.md             # UPDATED: Uses catalog, generates spec
│   └── workflow-catalog.yml              # NEW: Workflow metadata
├── agents/
│   ├── create-agentic-campaign.agent.md  # UPDATED: Optional helper
│   └── agentic-campaign-designer.agent.md # UPDATED: Compilation only
├── campaigns/
│   └── <campaign-id>.campaign.md         # Generated by Phase 1

pkg/
├── campaign/
│   └── prompts/
│       └── campaign_creation_instructions.md # NEW: Shared instructions
├── workflow/
│   ├── safe_outputs.go                   # UPDATED: Add UpdateIssue
│   └── workflow_catalog.go               # NEW: Catalog parser

actions/
└── update-issue/
    └── action.yml                        # NEW: Update issue action
```

### Code Size Comparison

**Before**:
- campaign-generator.md: 50 lines (minimal)
- agentic-campaign-designer.agent.md: 286 lines (heavy)
- create-agentic-campaign.agent.md: 574 lines (heavy)
- template file: 286 lines (duplicate)
- **Total**: 1,196 lines

**After**:
- campaign-generator.md: 100 lines (spec generation)
- agentic-campaign-designer.agent.md: 60 lines (compilation only)
- create-agentic-campaign.agent.md: 100 lines (optional helper)
- campaign_creation_instructions.md: 200 lines (shared)
- workflow-catalog.yml: 100 lines (metadata)
- **Total**: 560 lines

**Reduction**: 636 lines (53% smaller)

---

**Document Version**: 1.0  
**Last Updated**: 2026-01-09  
**Next Review**: After Phase 1 completion
