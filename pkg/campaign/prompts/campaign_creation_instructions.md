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
   - Check external collections like the "agentics" collection
   - Use keywords to find matching workflows

2. **Dynamically discover all local workflow files:**
   ```bash
   ls .github/workflows/*.md    # Agentic workflows
   ls .github/workflows/*.yml   # Regular GitHub Actions workflows (exclude *.lock.yml)
   ```
   
   **Important**: Filter out `.lock.yml` files - these are compiled agentic workflows:
   ```bash
   ls .github/workflows/*.yml | grep -v ".lock.yml"
   ```

3. **Analyze each workflow** to determine fit:
   
   **For agentic workflows (.md files):**
   - Read the workflow description (frontmatter `description` field)
   - Check the workflow name and purpose
   - Look at safe-outputs to understand what the workflow does
   - Consider triggers (`on:` field) to understand when it runs
   
   **For regular workflows (.yml files, excluding .lock.yml):**
   - Read the workflow name (YAML `name:` field)
   - Identify the trigger events (`on:` field) - schedule, push, pull_request, etc.
   - Scan the jobs to understand what the workflow does (testing, security, docs, etc.)
   - **Assess AI enhancement potential** by considering:
     * Could AI analyze the output/results intelligently?
     * Could AI prioritize findings or create actionable reports?
     * Could AI suggest fixes or improvements automatically?
     * Would natural language explanations add value?

4. **Consider three types of workflows:**
   
   **A. Agentic Workflows** (`.md` files):
   AI-powered workflows that can analyze, reason, and create GitHub content via safe-outputs.
   
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
   
   When suggesting workflows, include agentic workflows, regular workflows to enhance, and workflows from the agentics collection.

5. **Match workflows to campaign goals:**

   **For security campaigns**, dynamically discover:
   - Agentic workflows: Search catalog for "security", "vulnerability", "scan" keywords
   - Regular workflows: Look for workflows with names containing "security", "codeql", "license", "scan"
     * Example candidates: `security-scan.yml`, `codeql.yml`, `license-check.yml`
     * **AI enhancement**: Vulnerability prioritization, automated remediation PRs, natural language explanations
   - Agentics collection: `ci-doctor` (for CI security), `repo-ask` (for security questions)
   
   **For dependency/upgrade campaigns**, dynamically discover:
   - Agentic workflows: Search catalog for "dependency", "upgrade", "update" keywords
   - Regular workflows: Look for workflows related to dependency management
   - Agentics collection: `daily-dependency-updater`, `pr-fix` (for failing dependencies)
   
   **For documentation campaigns**, dynamically discover:
   - Agentic workflows: Search catalog for "doc", "documentation", "guide" keywords
   - Regular workflows: Look for workflows with names containing "docs", "link-check", "documentation"
     * Example candidates: `docs.yml`, `link-check.yml`
     * **AI enhancement**: Quality analysis, gap identification, alternative link suggestions
   - Agentics collection: `update-docs`, `weekly-research` (for documentation research)
   
   **For code quality campaigns**, dynamically discover:
   - Agentic workflows: Search catalog for "quality", "refactor", "test" keywords
   - Regular workflows: Look for CI workflows with testing, linting
     * Example candidates: `ci.yml`, `test-*.yml`
     * **AI enhancement**: Test failure analysis, flaky test detection, coverage recommendations
   - Agentics collection: `daily-test-coverage-improver`, `daily-performance-improver`, `daily-adhoc-qa`
   
   **For CI/CD optimization campaigns**, dynamically discover:
   - Agentic workflows: Search catalog for "ci", "workflow", "build" keywords
   - Regular workflows: Look for CI/CD workflows
     * Example candidates: `ci.yml`, `build.yml`, `deploy.yml`
     * **AI enhancement**: Performance optimization, failure analysis, build caching suggestions
   - Agentics collection: `ci-doctor`, `q-workflow-optimizer`, `pr-fix`
   
   **For maintenance campaigns**, dynamically discover:
   - Agentic workflows: Search catalog for "maintenance", "cleanup", "automation" keywords
   - Regular workflows: Look for maintenance and cleanup workflows
     * Example candidates: `cleanup.yml`, `maintenance.yml`, `auto-close-*.yml`
     * **AI enhancement**: Intelligent cleanup decisions, automated issue management

6. **Determine workflow strategy:**
   - **Use existing agentic**: Workflows in `.github/workflows/*.md` that already do what's needed
   - **Enhance regular workflows**: Regular `.yml` workflows (excluding `.lock.yml`) that would benefit from AI capabilities
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
**Enhancement candidates**: Regular workflows discovered dynamically by scanning `.github/workflows/*.yml` (excluding `.lock.yml`)

### Examples (Using Dynamic Discovery)

**For "Migrate to Node 20" campaign:**
1. **Scan catalog** for agentic workflows with "dependency", "upgrade" keywords
   - Found: `cli-version-checker.md` (monitors versions)
2. **Scan `.github/workflows/*.yml`** for dependency-related workflows
   - Look for workflows with "dependency", "update", "npm", "package" in name
   - Assess if they could benefit from AI (e.g., intelligent upgrade planning)
3. **Include from agentics collection**: `daily-dependency-updater`, `pr-fix`
4. **Suggest new**: `node-version-scanner` - Finds repos still on Node 16

**For "Security Q1 2025" campaign:**
1. **Scan catalog** for agentic workflows with "security", "vulnerability" keywords
   - Found: `daily-malicious-code-scan.md`
2. **Scan `.github/workflows/*.yml`** (excluding `.lock.yml`) for security workflows
   - Found candidates: `security-scan.yml`, `codeql.yml`, `license-check.yml`
   - Read each to understand what they do (Gosec/govulncheck/Trivy, CodeQL analysis, license compliance)
   - Assess AI enhancement potential: vulnerability prioritization, natural language explanations, automated fixes
3. **Include from agentics collection**: `ci-doctor` (monitors CI for security issues)
4. **Suggest new**: `security-reporter` - Weekly security posture reports

**Result**: Mix of 1 existing agentic + 2-3 enhanced regular + 1 external + 1 new = comprehensive security coverage

**For "Improve Code Quality" campaign:**
1. **Scan catalog** for agentic workflows with "quality", "refactor", "test" keywords
   - Found: `semantic-function-refactor.md`, `breaking-change-checker.md`
2. **Scan `.github/workflows/*.yml`** for CI/test workflows
   - Found candidates: `ci.yml`, `test-*.yml`, `lint.yml`
   - Assess AI enhancement: test failure analysis, flaky test detection, coverage recommendations
3. **Include from agentics collection**: `daily-test-coverage-improver`, `daily-performance-improver`, `daily-adhoc-qa`

**For "Documentation Excellence" campaign:**
1. **Scan catalog** for agentic workflows with "doc", "documentation" keywords
   - Found: `glossary-maintainer.md`, `blog-auditor.md`
2. **Scan `.github/workflows/*.yml`** for documentation workflows
   - Found candidates: `docs.yml`, `link-check.yml`
   - Assess AI enhancement: quality analysis, gap identification, alternative link suggestions
3. **Include from agentics collection**: `update-docs`

**For "Team Coordination" campaign:**
1. **Scan catalog** for agentic workflows with "team", "coordination", "planning" keywords
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
