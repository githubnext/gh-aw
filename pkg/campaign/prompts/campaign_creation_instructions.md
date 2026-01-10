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

1. **Check the workflow catalog** (`.github/workflow-catalog.yml`):
   - Query **agentic workflows** (`.md` files) organized by category
   - Check **regular GitHub Actions workflows** (`.yml` files) that could become agentic
   - Check external collections like the "agentics" collection
   - Use keywords to find matching workflows

2. **List all local workflow files:**
   ```bash
   ls .github/workflows/*.md    # Agentic workflows
   ls .github/workflows/*.yml   # Regular GitHub Actions workflows
   ```

3. **Analyze each workflow** to determine fit:
   - Read the workflow description (frontmatter `description` field for .md, `name:` for .yml)
   - Check the workflow name and purpose
   - Look at safe-outputs to understand what the workflow does (agentic workflows)
   - Consider triggers (`on:` field) to understand when it runs
   - For regular workflows, assess their **agentic_potential** from the catalog

4. **Consider three types of workflows:**
   
   **A. Agentic Workflows** (`.md` files):
   AI-powered workflows that can analyze, reason, and create GitHub content via safe-outputs.
   
   **B. Regular GitHub Actions Workflows** (`.yml` files):
   Standard automation workflows that could be **enhanced** by converting to agentic workflows.
   The catalog's `regular_workflows` section identifies candidates with:
   - Current functionality and triggers
   - Agentic potential (how AI could enhance them)
   - Suggested enhancements (specific AI capabilities to add)
   
   **C. External Workflow Collections:**
   
   **Agentics Collection** (https://github.com/githubnext/agentics):
   A family of reusable GitHub Agentic Workflows that can be installed in any repository.
   
   Categories available:
   - **Triage & Analysis**: issue-triage, ci-doctor, repo-ask, daily-accessibility-review, q-workflow-optimizer
   - **Research & Planning**: weekly-research, daily-team-status, daily-plan, plan-command
   - **Coding & Development**: daily-progress, daily-dependency-updater, update-docs, pr-fix, daily-adhoc-qa, daily-test-coverage-improver, daily-performance-improver
   
   When suggesting workflows, include agentic workflows, regular workflows to enhance, and workflows from the agentics collection.

5. **Match workflows to campaign goals:**

   **For security campaigns**, look for:
   - Agentic workflows: `daily-malicious-code-scan` (local)
   - Regular workflows to enhance: `security-scan`, `codeql`, `license-check` (see catalog for AI enhancement ideas)
   - Agentics collection: `ci-doctor` (for CI security), `repo-ask` (for security questions)
   - **Enhancement opportunity**: Convert `security-scan.yml` to agentic workflow with AI-powered vulnerability prioritization and automated fix generation
   
   **For dependency/upgrade campaigns**, look for:
   - Agentic workflows: `cli-version-checker` (local)
   - Regular workflows to enhance: Consider enhancing dependency-related workflows with AI analysis
   - Agentics collection: `daily-dependency-updater`, `pr-fix` (for failing dependencies)
   
   **For documentation campaigns**, look for:
   - Agentic workflows: `glossary-maintainer`, `blog-auditor` (local)
   - Regular workflows to enhance: `docs`, `link-check` (see catalog for AI enhancement ideas)
   - Agentics collection: `update-docs`, `weekly-research` (for documentation research)
   - **Enhancement opportunity**: Convert `link-check.yml` to agentic workflow with automatic alternative link suggestions
   
   **For code quality campaigns**, look for:
   - Agentic workflows: `semantic-function-refactor`, `breaking-change-checker` (local)
   - Regular workflows to enhance: `ci` (see catalog for AI-powered test analysis)
   - Agentics collection: `daily-test-coverage-improver`, `daily-performance-improver`, `daily-adhoc-qa`
   
   **For CI/CD and workflow optimization campaigns**, look for:
   - Agentic workflows: `ci-doctor`, `ci-coach`, `audit-workflows` (local)
   - Regular workflows to enhance: `ci`, `integration-agentics` (see catalog for AI enhancement ideas)
   - Agentics collection: `ci-doctor`, `q-workflow-optimizer`, `pr-fix`
   - **Enhancement opportunity**: Enhance `ci.yml` with AI-powered failure analysis and flaky test detection
   
   **For team coordination campaigns**, look for:
   - Agentic workflows: `campaign-generator` (local)
   - Regular workflows to enhance: Consider workflow coordination patterns
   - Agentics collection: `daily-team-status`, `daily-plan`, `plan-command`, `issue-triage`

6. **Determine workflow strategy:**
   - **Use existing agentic**: Workflows in `.github/workflows/*.md` that already do what's needed
   - **Enhance regular workflows**: Regular `.yml` workflows that would benefit from AI capabilities (reference catalog's `agentic_potential`)
   - **Use existing from agentics**: Workflows from the agentics collection that can be installed
   - **Suggest new**: Workflows that need to be created from scratch
   - **Combination**: Mix of agentic, regular (to enhance), agentics collection, and new workflows

7. **Suggest 2-4 workflows total** (agentic + regular to enhance + agentics collection + new)
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
**Enhancement candidates**: Regular workflows that could benefit from AI (see catalog's `regular_workflows` section)

### Examples

**For "Migrate to Node 20" campaign:**
- Local agentic: `cli-version-checker.md` (monitors versions)
- Agentics: `daily-dependency-updater` (from agentics collection)
- New: `node-version-scanner` - Finds repos still on Node 16
- Agentics: `pr-fix` (from agentics collection - fixes failing PRs during migration)

**For "Security Q1 2025" campaign:**
- Local agentic: `daily-malicious-code-scan.md`
- Regular to enhance: `security-scan.yml` → Convert to agentic with AI vulnerability prioritization
- Regular to enhance: `codeql.yml` → Enhance with natural language explanations and automated fixes
- Agentics: `ci-doctor` (from agentics collection - monitors CI for security issues)
- New: `security-reporter` - Weekly security posture reports

**For "Improve Code Quality" campaign:**
- Local agentic: `semantic-function-refactor.md`, `breaking-change-checker.md`
- Regular to enhance: `ci.yml` → Enhance with AI-powered test failure analysis and flaky test detection
- Agentics: `daily-test-coverage-improver` (from agentics collection)
- Agentics: `daily-performance-improver` (from agentics collection)
- Agentics: `daily-adhoc-qa` (from agentics collection)

**For "Documentation Excellence" campaign:**
- Local agentic: `glossary-maintainer.md`, `blog-auditor.md`
- Regular to enhance: `docs.yml` → Add AI quality analysis and gap identification
- Regular to enhance: `link-check.yml` → Enhance with automatic alternative link suggestions and archive recommendations
- Agentics: `update-docs` (from agentics collection)

**For "Team Coordination" campaign:**
- Agentics: `issue-triage` (from agentics collection)
- Agentics: `daily-team-status` (from agentics collection)
- Agentics: `daily-plan` (from agentics collection)
- Agentics: `plan-command` (from agentics collection)

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

**Recommended Custom Fields**:

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

The orchestrator automatically populates these fields. See the [Project Management guide](https://github.com/githubnext/gh-aw/blob/main/docs/src/content/docs/guides/campaigns/project-management.md) for setup instructions.

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

## Best Practices

### DO:
- ✅ Generate unique campaign IDs in kebab-case
- ✅ Scan existing workflows before suggesting new ones
- ✅ Apply principle of least privilege for safe outputs
- ✅ Assess risk level based on campaign scope
- ✅ Include clear ownership and governance
- ✅ Check for file conflicts before creating
- ✅ Compile and validate before creating PR

### DON'T:
- ❌ Create campaigns with duplicate IDs
- ❌ Suggest only new workflows without checking existing ones
- ❌ Grant unnecessary safe output permissions
- ❌ Skip risk assessment and governance
- ❌ Create campaigns without project board URL (when required)
- ❌ Skip compilation validation

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
