# Campaign Creation Instructions

This file consolidates shared campaign design logic used across campaign creation workflows.
These instructions guide AI agents in creating production-ready GitHub Agentic Workflows campaigns.

---

## Campaign Design Principles

### 1. Campaign ID Generation

Convert campaign names to kebab-case identifiers:
- Remove special characters
- Replace spaces with hyphens
- Lowercase everything
- Add timeline if mentioned (e.g., "security-q1-2025")

**Examples:**
- "Security Q1 2025" → "security-q1-2025"
- "Node.js 16 to 20 Migration" → "nodejs-16-to-20-migration"
- "Legacy Auth Refactor" → "legacy-auth-refactor"

**Conflict Resolution:** Before creating, verify `.github/workflows/<campaign-id>.campaign.md` doesn't exist. If it does, append `-v2` or timestamp.

---

## Workflow Identification Strategy

### Workflow Discovery Process

When identifying workflows for a campaign, follow this systematic approach:

1. **Dynamically discover all local workflow files:**
   ```bash
   ls .github/workflows/*.md    # Agentic workflows
   ls .github/workflows/*.yml   # Regular GitHub Actions workflows (exclude *.lock.yml)
   ```
   
   **Important**: Filter out `.lock.yml` files - these are compiled agentic workflows:
   ```bash
   ls .github/workflows/*.yml | grep -v ".lock.yml"
   ```

2. **Analyze each workflow** to determine fit:
   
   **For agentic workflows (.md files):**
   - Read the YAML frontmatter to extract:
     * `description` - What the workflow does
     * `on` - Trigger configuration (when it runs)
     * `safe-outputs` or `safe_outputs` - What GitHub operations it performs
   - Match the description/name to campaign category keywords
   - Assess relevance based on:
     * Keywords in description (security, test, doc, quality, etc.)
     * Safe outputs alignment with campaign needs
     * Trigger frequency and type
   
   **For regular workflows (.yml files, excluding .lock.yml):**
   - Read the workflow name (YAML `name:` field)
   - Identify the trigger events (`on:` field) - schedule, push, pull_request, etc.
   - Scan the jobs to understand what the workflow does (testing, security, docs, etc.)
   - **Assess AI enhancement potential** by considering:
     * Could AI analyze the output/results intelligently?
     * Could AI prioritize findings or create actionable reports?
     * Could AI suggest fixes or improvements automatically?
     * Would natural language explanations add value?

3. **Consider three types of workflows:**
   
   **A. Agentic Workflows** (`.md` files):
   AI-powered workflows that can analyze, reason, and create GitHub content via safe-outputs.
   Discovered by scanning `.github/workflows/*.md` and parsing frontmatter.
   
   **B. Regular GitHub Actions Workflows** (`.yml` files, not `.lock.yml`):
   Standard automation workflows that could be **enhanced** by converting to agentic workflows.
   
   **Identifying enhancement candidates:**
   - **Security workflows** (CodeQL, vulnerability scanning, license checks)
     → AI could prioritize vulnerabilities, explain findings, suggest fixes
   - **CI/CD workflows** (testing, building, deployment)
     → AI could analyze failures, detect flaky tests, optimize performance
   - **Documentation workflows** (builds, link checking, validation)
     → AI could assess quality, identify gaps, suggest improvements
   - **Maintenance workflows** (cleanup, automation, housekeeping)
     → AI could make intelligent decisions about what to clean/keep
   
   **C. External Workflow Collections:**
   
   **Agentics Collection** (https://github.com/githubnext/agentics):
   A family of reusable GitHub Agentic Workflows that can be installed in any repository.
   
   Categories available:
   - **Triage & Analysis**: issue-triage, ci-doctor, repo-ask, daily-accessibility-review, q-workflow-optimizer
   - **Research & Planning**: weekly-research, daily-team-status, daily-plan, plan-command
   - **Coding & Development**: daily-progress, daily-dependency-updater, update-docs, pr-fix, daily-adhoc-qa, daily-test-coverage-improver, daily-performance-improver
   
   When suggesting workflows, include discovered agentic workflows, regular workflows to enhance, and workflows from the agentics collection.

4. **Match workflows to campaign goals:**

   **For security campaigns**, dynamically discover:
   - Agentic workflows: Scan `.github/workflows/*.md` files, parse frontmatter, match descriptions containing "security", "vulnerability", "scan", "malicious" keywords
   - Regular workflows: Look for workflows with names containing "security", "codeql", "license", "scan"
     * Example candidates: `security-scan.yml`, `codeql.yml`, `license-check.yml`
     * **AI enhancement**: Vulnerability prioritization, automated remediation PRs, natural language explanations
   - Agentics collection: `ci-doctor` (for CI security), `repo-ask` (for security questions)
   
   **For dependency/upgrade campaigns**, dynamically discover:
   - Agentic workflows: Scan `.github/workflows/*.md`, match descriptions with "dependency", "upgrade", "update", "version" keywords
   - Regular workflows: Look for workflows related to dependency management
   - Agentics collection: `daily-dependency-updater`, `pr-fix` (for failing dependencies)
   
   **For documentation campaigns**, dynamically discover:
   - Agentic workflows: Scan `.github/workflows/*.md`, match descriptions with "doc", "documentation", "guide", "glossary", "blog" keywords
   - Regular workflows: Look for workflows with names containing "docs", "link-check", "documentation"
     * Example candidates: `docs.yml`, `link-check.yml`
     * **AI enhancement**: Quality analysis, gap identification, alternative link suggestions
   - Agentics collection: `update-docs`, `weekly-research` (for documentation research)
   
   **For code quality campaigns**, dynamically discover:
   - Agentic workflows: Scan `.github/workflows/*.md`, match descriptions with "quality", "refactor", "test", "lint", "metrics" keywords
   - Regular workflows: Look for CI workflows with testing, linting
     * Example candidates: `ci.yml`, `test-*.yml`
     * **AI enhancement**: Test failure analysis, flaky test detection, coverage recommendations
   - Agentics collection: `daily-test-coverage-improver`, `daily-performance-improver`, `daily-adhoc-qa`
   
   **For CI/CD optimization campaigns**, dynamically discover:
   - Agentic workflows: Scan `.github/workflows/*.md`, match descriptions with "ci", "workflow", "build", "audit", "performance" keywords
   - Regular workflows: Look for CI/CD workflows
     * Example candidates: `ci.yml`, `build.yml`, `deploy.yml`
     * **AI enhancement**: Performance optimization, failure analysis, build caching suggestions
   - Agentics collection: `ci-doctor`, `q-workflow-optimizer`, `pr-fix`
   
   **For maintenance campaigns**, dynamically discover:
   - Agentic workflows: Scan `.github/workflows/*.md`, match descriptions with "maintenance", "cleanup", "automation", "housekeeping" keywords
   - Regular workflows: Look for maintenance and cleanup workflows
     * Example candidates: `cleanup.yml`, `maintenance.yml`, `auto-close-*.yml`
     * **AI enhancement**: Intelligent cleanup decisions, automated issue management

5. **Determine workflow strategy:**
   - **Use existing agentic**: Workflows in `.github/workflows/*.md` that already do what's needed
   - **Enhance regular workflows**: Regular `.yml` workflows (excluding `.lock.yml`) that would benefit from AI capabilities
   - **Use existing from agentics**: Workflows from the agentics collection that can be installed
   - **Suggest new**: Workflows that need to be created from scratch
   - **Combination**: Mix of agentic, regular (to enhance), agentics collection, and new workflows

6. **Suggest 2-4 workflows total** (agentic + regular to enhance + agentics collection + new)
   - Prioritize existing agentic workflows
   - Identify 1-2 regular workflows that would benefit most from AI enhancement
   - Include relevant workflows from agentics collection
   - Suggest new workflows only if gaps remain

### Common Workflow Patterns

**Scanner workflows**: Identify issues (e.g., "security-scanner", "outdated-deps-scanner")
**Fixer workflows**: Create PRs (e.g., "vulnerability-fixer", "dependency-updater", "pr-fix")
**Reporter workflows**: Generate summaries (e.g., "campaign-reporter", "progress-tracker", "daily-team-status")
**Coordinator workflows**: Manage orchestration (auto-generated)
**Triage workflows**: Organize and prioritize work (e.g., "issue-triage", "plan-command")
**Enhancement candidates**: Regular workflows discovered dynamically by scanning `.github/workflows/*.yml` (excluding `.lock.yml`)

### Examples (Using Dynamic Discovery)

**For "Migrate to Node 20" campaign:**
1. **Scan `.github/workflows/*.md`** for agentic workflows with "dependency", "upgrade" keywords
   - Found: `cli-version-checker.md` (monitors versions)
2. **Scan `.github/workflows/*.yml`** for dependency-related workflows
   - Look for workflows with "dependency", "update", "npm", "package" in name
   - Assess if they could benefit from AI (e.g., intelligent upgrade planning)
3. **Include from agentics collection**: `daily-dependency-updater`, `pr-fix`
4. **Suggest new**: `node-version-scanner` - Finds repos still on Node 16

**For "Security Q1 2025" campaign:**
1. **Scan `.github/workflows/*.md`** for agentic workflows with "security", "vulnerability" keywords
   - Found: `daily-malicious-code-scan.md`
2. **Scan `.github/workflows/*.yml`** (excluding `.lock.yml`) for security workflows
   - Found candidates: `security-scan.yml`, `codeql.yml`, `license-check.yml`
   - Read each to understand what they do (Gosec/govulncheck/Trivy, CodeQL analysis, license compliance)
   - Assess AI enhancement potential: vulnerability prioritization, natural language explanations, automated fixes
3. **Include from agentics collection**: `ci-doctor` (monitors CI for security issues)
4. **Suggest new**: `security-reporter` - Weekly security posture reports

**Result**: Mix of 1 existing agentic + 2-3 enhanced regular + 1 external + 1 new = comprehensive security coverage

**For "Improve Code Quality" campaign:**
1. **Scan `.github/workflows/*.md`** for agentic workflows with "quality", "refactor", "test" keywords
   - Found: `semantic-function-refactor.md`, `breaking-change-checker.md`
2. **Scan `.github/workflows/*.yml`** for CI/test workflows
   - Found candidates: `ci.yml`, `test-*.yml`, `lint.yml`
   - Assess AI enhancement: test failure analysis, flaky test detection, coverage recommendations
3. **Include from agentics collection**: `daily-test-coverage-improver`, `daily-performance-improver`, `daily-adhoc-qa`

**For "Documentation Excellence" campaign:**
1. **Scan `.github/workflows/*.md`** for agentic workflows with "doc", "documentation" keywords
   - Found: `glossary-maintainer.md`, `blog-auditor.md`
2. **Scan `.github/workflows/*.yml`** for documentation workflows
   - Found candidates: `docs.yml`, `link-check.yml`
   - Assess AI enhancement: quality analysis, gap identification, alternative link suggestions
3. **Include from agentics collection**: `update-docs`

**For "Team Coordination" campaign:**
1. **Scan `.github/workflows/*.md`** for agentic workflows with "team", "coordination", "planning" keywords
2. **Scan `.github/workflows/*.yml`** for team coordination workflows (may find few)
3. **Include from agentics collection**: `issue-triage`, `daily-team-status`, `daily-plan`, `plan-command`

---

## Safe Output Configuration

Based on workflow needs, determine allowed safe outputs using the principle of **least privilege**.

### Common Safe Output Patterns

**Scanner workflows:**
```yaml
allowed-safe-outputs:
  - create-issue
  - add-comment
```

**Fixer workflows:**
```yaml
allowed-safe-outputs:
  - create-pull-request
  - add-comment
```

**Reporter workflows:**
```yaml
allowed-safe-outputs:
  - create-discussion
  - update-issue
```

**Default recommendation:**
```yaml
allowed-safe-outputs:
  - create-issue
  - add-comment
  - create-pull-request
```

**Security principle:** Only add `update-issue`, `update-pull-request`, or `create-pull-request-review-comment` if specifically required.

---

## Governance and Security

### Ownership Guidelines

**Default ownership:**
- Current user or requester
- Team owners for organization-wide campaigns (e.g., @security-team, @platform-team)

**Ownership structure:**
```yaml
owners:
  - @<username-or-team>
```

### Executive Sponsors

**When to require sponsors:**
- **High-risk campaigns:** REQUIRED (exec sponsor approval)
- **Medium-risk campaigns:** RECOMMENDED
- **Low-risk campaigns:** OPTIONAL

```yaml
executive-sponsors:
  - @<sponsor-username>  # For high/medium risk
```

### Risk Level Assessment

**Risk indicators:**

**High risk:**
- Sensitive changes (security, production, data handling)
- Multiple repositories affected
- Potential for breaking changes
- Requires executive oversight

**Medium risk:**
- Creating issues/PRs across repositories
- Light automation with potential side effects
- Requires team review

**Low risk:**
- Read-only operations
- Reporting and analysis
- Single repository scope

### Approval Policies

**High risk:**
```yaml
approval-policy:
  required-approvals: 2
  required-reviewers:
    - security-team
    - platform-leads
```

**Medium risk:**
```yaml
approval-policy:
  required-approvals: 1
  required-reviewers:
    - <team-name>
```

**Low risk:**
No approval policy needed (omit field).

---

## Campaign File Structure

### Complete Campaign Template

```markdown
---
id: <campaign-id>
name: <Campaign Name>
description: <One-sentence description>
project-url: <GitHub Project URL>
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

**Project Views** (automatically created):

The campaign generator creates three views optimized for campaign tracking:

1. **Campaign Roadmap** (Roadmap layout)
   - Timeline visualization with Worker/Workflow swimlanes
   - Shows work distribution across time
   - Group by Worker/Workflow field for swimlanes
   
2. **Task Tracker** (Table layout)
   - Detailed task tracking with all fields visible
   - Use "Slice by" filtering for Priority, Status, Worker/Workflow
   - Sort by Priority or Effort for prioritization
   
3. **Progress Board** (Board layout)
   - Kanban-style progress tracking
   - Group by Status field (Todo, In Progress, Blocked, Done)
   - Visual workflow state management

**Recommended Custom Fields** (must be created manually in GitHub Projects UI):

1. **Worker/Workflow** (Single select): <workflow-id-1>, <workflow-id-2>
   - Enables swimlane grouping in Roadmap views
   
2. **Priority** (Single select): High, Medium, Low
   - Priority-based filtering and sorting
   
3. **Status** (Single select): Todo, In Progress, Blocked, Done
   - Work state tracking
   
4. **Start Date** / **End Date** (Date)
   - Timeline visualization in Roadmap views
   
5. **Effort** (Single select): Small (1-3 days), Medium (1 week), Large (2+ weeks)
   - Capacity planning

The orchestrator automatically populates these fields. See the [Project Management guide](https://github.com/githubnext/gh-aw/blob/main/docs/src/content/docs/guides/campaigns/project-management.md) for detailed setup instructions and view configuration examples.

## Timeline

- **Start**: <Date or "TBD">
- **Target completion**: <Date or "Ongoing">
- **Current state**: Planned

## Success Metrics

- <Measurable outcome 1>
- <Measurable outcome 2>
- <Measurable outcome 3>
```

---

## Campaign Compilation

After creating the campaign file, compile it to generate the orchestrator:

```bash
gh aw compile <campaign-id>
```

**Generated files:**
- `.github/workflows/<campaign-id>.campaign.g.md` (orchestrator)
- `.github/workflows/<campaign-id>.campaign.lock.yml` (compiled workflow)

**If compilation fails:**
- Review error messages carefully
- Fix syntax issues in frontmatter
- Verify all required fields are present
- Re-compile until successful
- Consult `.github/aw/github-agentic-workflows.md` if needed

---

## Pull Request Template

When creating a PR for the new campaign:

```markdown
## New Campaign: <Campaign Name>

### Purpose
<Brief description of what this campaign accomplishes>

### Workflows
- `<workflow-id-1>`: <What it does>
- `<workflow-id-2>`: <What it does>

### Risk Level
**<Low/Medium/High>** - <Why this risk level>

### Next Steps
1. Review and approve this PR
2. Merge to activate the campaign
3. Create GitHub Project board using campaign template (if not already created)
4. Create/update the worker workflows listed above

### Links
- Campaign spec: `.github/workflows/<campaign-id>.campaign.md`
- [Campaign documentation](https://githubnext.github.io/gh-aw/guides/campaigns/)
```

---

## Workflow Creation Guardrails

### When to Create New Workflows

**ONLY create new workflows when:**
1. No existing workflow does the required task
2. The campaign objective clearly requires a specific capability that's missing
3. Workflows from the agentics collection don't fit the need
4. You can articulate a clear, focused purpose for the new workflow

**AVOID creating workflows when:**
- An existing workflow can be used (even if not perfect)
- The task can be handled by manual processes initially
- You're unsure about the exact requirements
- The workflow would duplicate functionality

### Workflow Creation Safety Checklist

Before suggesting a new workflow, verify:

- [ ] **Clear purpose**: Can you describe in one sentence what the workflow does?
- [ ] **Defined trigger**: Is it clear when/how the workflow should run?
- [ ] **Bounded scope**: Does it have clear input/output boundaries?
- [ ] **Testable**: Can the workflow be tested independently?
- [ ] **Reusable**: Could other campaigns use this workflow?

### Workflow Naming Guidelines

**Good names** (specific, action-oriented):
- `security-vulnerability-scanner` - Scans for vulnerabilities
- `node-version-checker` - Checks Node.js versions
- `dependency-update-pr-creator` - Creates dependency update PRs

**Poor names** (vague, too broad):
- `security-workflow` - What does it do?
- `checker` - Check what?
- `helper` - Help with what?

---

## Passive vs Active Campaigns

### Start Passive (Default)

**Passive campaigns** (recommended for beginners):
- Discover and track existing work
- Lower risk and complexity
- No workflow creation or execution
- Good for learning campaign patterns

```yaml
# Passive campaign (default)
workflows:
  - existing-workflow-1
  - existing-workflow-2
# execute-workflows: false  # Default, can omit
```

### Progress to Active (Advanced)

**Active campaigns** (for experienced users):
- Execute workflows directly
- Can create missing workflows
- Higher complexity and risk
- Requires careful testing

**Prerequisites before enabling `execute-workflows: true`:**
1. You've successfully run at least one passive campaign
2. You understand how the orchestrator coordinates work
3. You have clear success criteria and governance rules
4. You're prepared to monitor and adjust during execution

```yaml
# Active campaign (advanced)
workflows:
  - framework-scanner
  - framework-upgrader
execute-workflows: true  # Explicitly enable

governance:
  max-project-updates-per-run: 10  # Start conservative
  max-comments-per-run: 5
```

**Migration path**: Start passive → Monitor for 1-2 weeks → Add governance rules → Enable active execution

---

## Decision Rationale Guidelines

When making decisions in campaigns, always explain WHY:

### Example: Workflow Selection

**Poor**: "Use `security-scanner` workflow"

**Good**: "Use `security-scanner` workflow because:
- It scans all Go files for vulnerabilities (matches campaign scope)
- It creates issues for findings (supports our reporting needs)
- It's already tested and stable (reduces risk)"

### Example: Governance Settings

**Poor**: "Set max-project-updates-per-run to 10"

**Good**: "Set max-project-updates-per-run to 10 because:
- We have ~50 services to track (conservative pacing)
- First campaign - want to monitor impact closely
- Can increase after observing initial runs"

### Example: KPI Selection

**Poor**: "Track services upgraded"

**Good**: "Track 'Services upgraded' as primary KPI because:
- Directly measures campaign objective
- Easily quantifiable (count of completed upgrades)
- Updated automatically from project board status"

---

## Failure Handling and Recovery

### Default Failure Behaviors

**Compilation failures** - Campaign generator should:
1. Report the error clearly with context
2. Suggest specific fixes for common issues
3. Provide a link to documentation
4. **NOT** delete partially created files (for debugging)

**Runtime failures** - Orchestrator should:
1. Continue with other work items (don't stop entire campaign)
2. Report failures in status update with context
3. Suggest recovery actions when possible
4. Maintain cursor/state for next run

### First-Time User Support

**For users creating their first campaign**:

1. **Validate requirements upfront**:
   - GitHub Project board exists and is accessible
   - At least one workflow exists or will be created
   - Governance settings are appropriate for first campaign

2. **Provide conservative defaults**:
   ```yaml
   governance:
     max-new-items-per-run: 5        # Start small
     max-project-updates-per-run: 5  # Monitor impact
     max-comments-per-run: 3         # Avoid noise
   ```

3. **Include onboarding guidance in campaign body**:
   ```markdown
   ## First Campaign? Read This!
   
   This is your first campaign - here's what to expect:
   
   1. **First run** - The orchestrator will initialize the project board and add the Epic issue
   2. **Monitor** - Check the project board after the first run to verify items appear correctly
   3. **Adjust** - Based on first run, you may want to adjust governance settings
   4. **Learn** - Each run provides status updates explaining what happened and why
   ```

---

## Best Practices

### Core Campaign Principles

Follow these fundamental principles when creating campaigns:

1. **Start with one small, clear goal per campaign**
   - Focus on a single, well-defined objective
   - Avoid scope creep - multiple goals should be separate campaigns
   - Example: "Upgrade Node.js to v20" not "Upgrade Node.js and refactor auth"

2. **Use passive mode first to observe and build trust**
   - Always start with passive mode (`execute-workflows: false` or omitted)
   - Monitor 1-2 weeks to understand orchestration behavior
   - Build confidence before enabling active execution

3. **Reuse existing workflows before creating new ones**
   - Thoroughly search `.github/workflows/*.md` for existing solutions
   - Check the [agentics collection](https://github.com/githubnext/agentics) for reusable workflows
   - Only create new workflows when existing ones don't meet requirements

4. **Keep permissions minimal (issues / draft PRs, no merges)**
   - Grant only the permissions needed for the campaign's scope
   - Prefer read permissions over write when possible
   - Use draft PRs instead of direct merges for code changes
   - Example: `issues: read, pull-requests: write` for issue tracking with PR creation

5. **Make outputs standardized and predictable**
   - Use consistent safe-output configurations across workflows
   - Document expected outputs in workflow descriptions
   - Follow established patterns for issue/PR formatting

6. **Escalate to humans when unsure**
   - Don't make risky decisions autonomously
   - Create issues or comments requesting human review
   - Include context and reasoning in escalation messages
   - Example: "This change affects authentication - requesting human review"

### DO:
- ✅ Generate unique campaign IDs in kebab-case
- ✅ Scan existing workflows before suggesting new ones
- ✅ Apply principle of least privilege for safe outputs
- ✅ Assess risk level based on campaign scope
- ✅ Include clear ownership and governance
- ✅ Check for file conflicts before creating
- ✅ Compile and validate before creating PR
- ✅ Start with passive mode for first campaign
- ✅ Provide clear rationale for all decisions
- ✅ Use conservative defaults for beginners
- ✅ Test new workflows before campaign use

### DON'T:
- ❌ Create campaigns with duplicate IDs
- ❌ Suggest only new workflows without checking existing ones
- ❌ Grant unnecessary safe output permissions
- ❌ Skip risk assessment and governance
- ❌ Create campaigns without project board URL (when required)
- ❌ Skip compilation validation
- ❌ Enable execute-workflows for first campaign
- ❌ Create workflows without clear purpose
- ❌ Use high governance limits for beginners
- ❌ Make decisions without explaining rationale

---

## Reference Commands

```bash
# List all workflow files
ls .github/workflows/*.md

# Search for workflows by keyword
grep -l "security\|vulnerability" .github/workflows/*.md

# Validate all campaigns
gh aw campaign validate

# Compile specific campaign
gh aw compile <campaign-id>

# Compile with strict mode
gh aw compile --strict <campaign-id>

# List all campaigns
gh aw campaign status
```

---

**Last Updated:** 2026-01-09
