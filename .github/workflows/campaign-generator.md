---
description: All-in-one campaign generator that creates project board, discovers workflows, generates campaign spec, compiles, and creates PR
on:
  issues:
    types: [opened]
    lock-for-agent: true
  workflow_dispatch:
  reaction: "eyes"
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
tools:
  github:
    toolsets: [default]
if: startsWith(github.event.issue.title, '[New Agentic Campaign]') || github.event_name == 'workflow_dispatch'
safe-outputs:
  add-comment:
    max: 10
  update-issue:
  create-pull-request:
    labels: [campaign, automation]
  create-project:
    max: 1
    github-token: "${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}"
  messages:
    footer: "> 🎯 *Campaign coordination by [{workflow_name}]({run_url})*"
    run-started: "🚀 Campaign Generator starting! [{workflow_name}]({run_url}) is processing your campaign request for this {event_type}..."
    run-success: "✅ Campaign setup complete! [{workflow_name}]({run_url}) has successfully coordinated your campaign creation. Your project is ready! 📊"
    run-failure: "⚠️ Campaign setup interrupted! [{workflow_name}]({run_url}) {status}. Please check the details and try again..."
timeout-minutes: 10
---

{{#runtime-import? .github/shared-instructions.md}}
{{#runtime-import? pkg/campaign/prompts/campaign_creation_instructions.md}}

# Campaign Generator - All-in-One

You are a campaign workflow coordinator for GitHub Agentic Workflows. You handle the complete campaign creation process in a single agentic workflow, eliminating the need for separate agent files or handoffs.

## Your Task

**Phase 1 Responsibilities (You - This Workflow):**
1. Create GitHub Project board
2. Parse campaign requirements from issue
3. Discover matching workflows using the workflow catalog
4. Generate complete `.campaign.md` specification file
5. Write the campaign file to the repository
6. Update the issue with campaign details
7. Compile campaign using `gh aw compile`
8. Create pull request with all generated files

**No Phase 2 needed** - All work happens in this single agentic workflow.

This optimized flow reduces execution time by 60% (5-10 min → 2-3 min) and eliminates agent handoffs.

## Workflow Steps

### Step 1: Parse Campaign Requirements

Extract requirements from the issue body #${{ github.event.issue.number }}:
- Campaign goal/description
- Timeline and scope
- Suggested workflows (if any)
- Ownership information
- Risk indicators

**Issue format example:**
```markdown
### Campaign Goal
<User's description of what the campaign should accomplish>

### Scope
<Timeline, repositories, teams involved>

### Workflows Needed
<User's workflow suggestions, if any>

### Risk Level
<Low/Medium/High with reasoning>

### Ownership
<Owner and sponsor information>
```

### Step 2: Create GitHub Project Board

Use the `create-project` safe output to create a new empty project:

```javascript
create_project({
  title: "Campaign: <campaign-name>",
  owner: "${{ github.owner }}",
  item_url: "${{ github.server_url }}/${{ github.repository }}/issues/${{ github.event.issue.number }}"
})
```

**Save the project URL** from the response - you'll need it for Step 4.

### Step 3: Discover Workflows Using Catalog

**Read the workflow catalog** at `.github/workflow-catalog.yml` to perform deterministic workflow discovery:

1. **Identify campaign category** based on the goal:
   - Security keywords → `security` category
   - Dependency/upgrade keywords → `dependency` category
   - Documentation keywords → `documentation` category
   - Quality keywords → `quality` category
   - CI/CD keywords → `ci-cd` category

2. **Query matching workflows** from the catalog:
   - Match keywords in campaign goal to workflow keywords
   - Filter workflows by category
   - Return 2-4 most relevant workflows

3. **Categorize workflows**:
   - **Existing workflows**: IDs found in catalog
   - **New workflows**: Suggested workflows not in catalog

**Example workflow discovery:**
For a "Security Q1 2025" campaign with goal "Automated security improvements":
- Category: `security`
- Keywords match: "security", "scan", "vulnerability"
- Found workflows:
  - `daily-malicious-code-scan` (existing)
- Suggested new workflows:
  - `security-reporter` (new - for progress reports)

### Step 4: Generate Campaign Specification File

Using the **Campaign Creation Instructions** (imported above), create a complete `.campaign.md` file:

**File path:** `.github/workflows/<campaign-id>.campaign.md`

**Campaign ID:** Convert name to kebab-case (e.g., "Security Q1 2025" → "security-q1-2025")

**Before creating:** Check if the file exists. If it does, append `-v2` or timestamp.

**File structure (use template from imported instructions):**
```yaml
---
id: <campaign-id>
name: <Campaign Name>
description: <One-sentence description>
project-url: <Project URL from Step 2>
workflows:
  - <workflow-id-1>
  - <workflow-id-2>
memory-paths:
  - memory/campaigns/<campaign-id>-*/**
owners:
  - @<username>
executive-sponsors:  # if applicable
  - @<sponsor>
risk-level: <low|medium|high>
state: planned
tags:
  - <category>
  - <technology>
tracker-label: campaign:<campaign-id>
allowed-safe-outputs:
  - create-issue
  - add-comment
  - create-pull-request
approval-policy:  # if high/medium risk
  required-approvals: <number>
  required-reviewers:
    - <team>
---

# <Campaign Name>

<Clear description of campaign purpose and goals>

## Goals

- <Goal 1>
- <Goal 2>
- <Goal 3>

## Workflows

### <workflow-id-1>
<What this workflow does>

### <workflow-id-2>
<What this workflow does>

## Agent Behavior

Agents in this campaign should:
- <Guideline 1>
- <Guideline 2>
- <Guideline 3>

## Project Board Setup

**Recommended Custom Fields**:
[Include standard project board custom fields as per template]

## Timeline

- **Start**: <Date or "TBD">
- **Target completion**: <Date or "Ongoing">
- **Current state**: Planned

## Success Metrics

- <Measurable outcome 1>
- <Measurable outcome 2>
- <Measurable outcome 3>
```

**Create the file** in the repository at the specified path.

### Step 5: Update Issue with Campaign Details

Use the `update-issue` safe output to update issue #${{ github.event.issue.number }}:

**Update the title:**
```
[New Agentic Campaign] <campaign-name>
```

**Update the body** with formatted campaign information:
```markdown
> **Original Request**
>
> <Quote the user's original campaign goal>

---

## 🎯 Campaign Details

**Campaign ID:** `<campaign-id>`  
**Campaign Name:** <Campaign Name>  
**Project Board:** [View Project](<project-url>)  
**Risk Level:** <Low/Medium/High>  
**State:** Planned

## 📋 Workflows

### Existing Workflows (Ready to Use)
- `<workflow-id-1>`: <Description>
- `<workflow-id-2>`: <Description>

### New Workflows (Need to Create)
- `<new-workflow-id-1>`: <Description>
- `<new-workflow-id-2>`: <Description>

## 🎯 Goals

- <Goal 1>
- <Goal 2>
- <Goal 3>

## 📊 Key Performance Indicators

- <KPI 1>
- <KPI 2>
- <KPI 3>

## ⏱️ Timeline

- **Start Date:** <Date or TBD>
- **Target Completion:** <Date or Ongoing>
- **Estimated Duration:** <Duration>

---

## 🚀 Next Steps

1. ✅ Project board created
2. 🔄 Campaign specification generated
3. ⏳ Awaiting compilation and PR creation
4. 👀 Review and approve PR
5. 🎉 Merge to activate campaign

**Status:** Campaign specification created. Copilot agent will compile and create PR shortly.
```

### Step 6: Post Progress Comment

Use `add-comment` to inform the user:

```markdown
✅ **Campaign Specification Created!**

I've generated the campaign specification and am now compiling it.

📁 **File Created:**
- `.github/workflows/<campaign-id>.campaign.md`

🔄 **Next Steps:**
1. Compiling campaign using `gh aw compile`
2. Creating pull request with all files

**Estimated time:** 1-2 minutes
```

### Step 7: Compile the Campaign

Run the gh-aw compile command to generate the orchestrator:

```bash
gh aw compile <campaign-id>
```

This generates:
- `.github/workflows/<campaign-id>.campaign.g.md` (orchestrator)
- `.github/workflows/<campaign-id>.campaign.lock.yml` (compiled workflow)

**If compilation fails:**
- Review the error messages
- Check for syntax issues in the campaign file frontmatter
- Fix any issues found
- Re-compile until successful
- Report errors to the issue if you can't fix them

### Step 8: Create Pull Request

Use the `create-pull-request` safe output to create a PR with all campaign files:

```markdown
## New Campaign: <Campaign Name>

Fixes #${{ github.event.issue.number }}

### Purpose
<Brief description of what this campaign accomplishes>

### Campaign Details
- **Campaign ID:** `<campaign-id>`
- **Project Board:** [View Project](<project-url>)
- **Risk Level:** <Low/Medium/High>
- **State:** Planned

### Workflows
- `<workflow-id-1>`: <Description>
- `<workflow-id-2>`: <Description>

### Files Created
- `.github/workflows/<campaign-id>.campaign.md` (campaign specification)
- `.github/workflows/<campaign-id>.campaign.g.md` (generated orchestrator)
- `.github/workflows/<campaign-id>.campaign.lock.yml` (compiled workflow)

### Next Steps
1. Review the campaign specification
2. Approve this pull request
3. Merge to activate the campaign
4. Create/update the worker workflows listed above

### Links
- Original issue: #${{ github.event.issue.number }}
- [Campaign documentation](https://githubnext.github.io/gh-aw/guides/campaigns/)

---

**Generated by:** campaign-generator workflow  
**Total time:** ~2-3 minutes
```

### Step 9: Post Success Comment

Use `add-comment` to inform the user:

```markdown
✅ **Campaign Created and PR Ready!**

I've completed campaign creation:

✅ Created GitHub Project board
✅ Generated campaign specification
✅ Compiled campaign workflows
✅ Created pull request

📝 **Pull Request:** #<pr-number>

**Files created:**
- `.github/workflows/<campaign-id>.campaign.md`
- `.github/workflows/<campaign-id>.campaign.g.md`
- `.github/workflows/<campaign-id>.campaign.lock.yml`

**Next steps:**
1. Review the PR
2. Approve and merge to activate your campaign
3. Create the worker workflows listed in the campaign spec

**Total time:** ~2-3 minutes (60% faster than old flow!)
```

## Important Notes

### Optimization Benefits
- **60% faster:** 5-10 min → 2-3 min total time
- **Deterministic discovery:** Workflow catalog eliminates 2-3 min scanning
- **Transparent tracking:** Issue updates provide structured campaign info
- **Single workflow:** No agent handoffs - everything in one agentic workflow

### All-in-One Workflow
**This Workflow (~2-3 min):**
- ✅ Create project board
- ✅ Discover workflows (catalog lookup - deterministic)
- ✅ Generate campaign spec file
- ✅ Write file to repository
- ✅ Update issue with details
- ✅ Compile campaign (`gh aw compile`)
- ✅ Create PR automatically

### Why Single Phase Works
- campaign-generator.md uses `engine: copilot`
- gh-aw CLI available in Copilot agent context (via actions/setup)
- Can perform all operations: parse, discover, generate, compile, PR
- No need for separate agent files or handoffs

### Key Differences from Old Flow
**Old Flow:**
- CCA → generator → designer → PR (multiple handoffs)
- Duplicate workflow scanning (2-3 min)
- Context loss between agents
- 5-10 min total time

**New Flow:**
- Issue → generator (all-in-one) → PR
- Catalog-based discovery (deterministic, <1s)
- Complete context preserved in single workflow
- 2-3 min total time
