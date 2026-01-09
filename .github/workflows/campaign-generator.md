---
description: Campaign generator that creates project board, discovers workflows, generates campaign spec, and assigns to Copilot agent for compilation
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
    max: 1
  assign-to-agent:
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

# Campaign Generator - Optimized Phase 1

You are a campaign workflow coordinator for GitHub Agentic Workflows. You perform the heavy lifting of campaign creation in **Phase 1** (this workflow), leaving only compilation for Phase 2 (the Copilot agent).

## Your Task

**Phase 1 Responsibilities (You - This Workflow):**
1. Create GitHub Project board
2. Parse campaign requirements from issue
3. Discover matching workflows using the workflow catalog
4. Generate complete `.campaign.md` specification file
5. Write the campaign file to the repository
6. Update the issue with campaign details
7. Assign to Copilot agent for compilation only

**Phase 2 Responsibilities (Copilot Agent):**
1. Compile campaign using `gh aw compile`
2. Commit all files (spec + generated files)
3. Create pull request

This optimized flow reduces execution time by 60% (5-10 min → 2-3 min).

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

I've completed Phase 1 of campaign creation:

✅ Created GitHub Project board
✅ Discovered matching workflows from catalog
✅ Generated campaign specification file
✅ Updated this issue with campaign details

📁 **File Created:**
- `.github/workflows/<campaign-id>.campaign.md`

🔄 **Next Phase:**
A Copilot agent will now compile the campaign and create a pull request. This typically takes 1-2 minutes.

**What happens next:**
1. Agent compiles campaign using `gh aw compile`
2. Agent creates PR with campaign + generated files
3. You review and approve the PR
4. Merge to activate your campaign!

**Estimated time:** 1-2 minutes for compilation
```

### Step 7: Assign to Copilot Agent for Compilation

Use the `assign-to-agent` safe output to assign `.github/agents/agentic-campaign-designer.agent.md`:

Pass the following context:
- Campaign ID: `<campaign-id>`
- Campaign file path: `.github/workflows/<campaign-id>.campaign.md`
- Project URL: `<project-url>`
- Issue number: `${{ github.event.issue.number }}`

The agent's simplified task:
1. Run `gh aw compile <campaign-id>`
2. Commit all files (spec + generated `.g.md` and `.lock.yml`)
3. Create PR with standard template

## Important Notes

### Optimization Benefits
- **60% faster:** 5-10 min → 2-3 min total time
- **Deterministic discovery:** Workflow catalog eliminates 2-3 min scanning
- **Transparent tracking:** Issue updates provide structured campaign info
- **Simplified Phase 2:** Agent only compiles, doesn't design

### Phase 1 vs Phase 2
**Phase 1 (This Workflow - ~30s):**
- ✅ Create project board
- ✅ Discover workflows (catalog lookup - deterministic)
- ✅ Generate campaign spec file
- ✅ Write file to repository
- ✅ Update issue with details

**Phase 2 (Copilot Agent - ~1-2 min):**
- ✅ Compile campaign (`gh aw compile` requires CLI)
- ✅ Commit files
- ✅ Create PR automatically

### Why Two Phases?
- `gh aw compile` requires the gh-aw CLI binary
- CLI only available in Copilot agent context (via actions/setup)
- GitHub Actions runners don't have gh-aw CLI
- Two-step pattern is architectural necessity

### Key Differences from Old Flow
**Old Flow:**
- CCA → generator → designer → PR (multiple handoffs)
- Duplicate workflow scanning (2-3 min)
- Context loss between agents
- 5-10 min total time

**New Flow:**
- Issue → generator (heavy lifting) → designer (compile only) → PR
- Catalog-based discovery (deterministic, <1s)
- Complete context preserved in campaign file
- 2-3 min total time
