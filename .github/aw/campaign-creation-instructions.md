# Campaign Creation Instructions

This file consolidates campaign design logic used across campaign creation workflows.

---

## Campaign ID Generation

Convert campaign names to kebab-case identifiers:
- Remove special characters, replace spaces with hyphens, lowercase everything
- Add timeline if mentioned (e.g., "security-q1-2025")

**Examples:**
- "Security Q1 2025" → "security-q1-2025"
- "Node.js 16 to 20 Migration" → "nodejs-16-to-20-migration"

**Conflict check:** Verify `.github/workflows/<campaign-id>.campaign.md` doesn't exist. If it does, append `-v2`.

---

## Workflow Discovery

When identifying workflows for a campaign:

1. **Scan for existing workflows:**
   ```bash
   ls .github/workflows/*.md    # Agentic workflows
   ls .github/workflows/*.yml | grep -v ".lock.yml"  # Regular workflows
   ```

2. **Check workflow types:**
   - **Agentic workflows** (`.md` files): Parse frontmatter for description, triggers, safe-outputs
   - **Regular workflows** (`.yml` files): Read name, triggers, jobs - assess AI enhancement potential
   - **External workflows**: Check [agentics collection](https://github.com/githubnext/agentics) for reusable workflows

3. **Match to campaign type:**
   - **Security**: Look for workflows with "security", "vulnerability", "scan" keywords
   - **Dependencies**: Look for "dependency", "upgrade", "update" keywords
   - **Documentation**: Look for "doc", "documentation", "guide" keywords
   - **Quality**: Look for "quality", "test", "lint" keywords
   - **CI/CD**: Look for "ci", "build", "deploy" keywords

4. **Workflow patterns:**
   - **Scanner**: Identify issues → create-issue, add-comment
   - **Fixer**: Create fixes → create-pull-request, add-comment
   - **Reporter**: Generate summaries → create-discussion, update-issue
   - **Orchestrator**: Manage campaign → auto-generated

5. **Select 2-4 workflows:**
   - Prioritize existing agentic workflows
   - Identify 1-2 regular workflows that benefit from AI
   - Include relevant workflows from agentics collection
   - Create new workflows only if gaps remain

---

## Safe Output Configuration

Configure safe outputs using **least privilege** - only grant what's needed.

### Operation Order (Required)

When setting up project-based campaigns, operations must be performed in this order:

1. **create-project** - Creates the GitHub project (includes creating views)
2. **update-project** - Adds items and fields to the project
3. **update-issue** - Updates issue metadata (if needed)
4. **assign-to-agent** - Assigns agents to issues (if needed)

This order ensures fields exist before being referenced and issues exist before assignment.

### Common Patterns

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

**Project-based campaigns:**
```yaml
allowed-safe-outputs:
  - create-project      # Step 1: Create project with views
  - update-project      # Step 2: Add items and fields
  - update-issue        # Step 3: Update issue metadata (optional)
  - assign-to-agent     # Step 4: Assign agents (optional)
```

**Default (safe start):**
```yaml
allowed-safe-outputs:
  - create-issue
  - add-comment
  - create-pull-request
```

**Security note:** Only add `update-issue`, `update-pull-request`, or `create-pull-request-review-comment` if specifically required.

---

## Governance

### Risk Levels

- **High risk**: Sensitive changes, multiple repos, breaking changes → Requires 2 approvals + executive sponsor
- **Medium risk**: Cross-repo issues/PRs, automated changes → Requires 1 approval
- **Low risk**: Read-only, single repo → No approval needed

### Ownership

```yaml
owners:
  - @<username-or-team>
executive-sponsors:  # Required for high-risk
  - @<sponsor-username>
approval-policy:     # For high/medium risk
  required-approvals: <1-2>
  required-reviewers:
    - <team-name>
```

---

## Campaign File Template

```markdown
---
id: <campaign-id>
name: <Campaign Name>
description: <One-sentence description>
project-url: <GitHub Project URL>
workflows:
  - <workflow-id-1>
  - <workflow-id-2>
owners:
  - @<username>
risk-level: <low|medium|high>
state: planned
allowed-safe-outputs:
  - create-issue
  - add-comment
---

# <Campaign Name>

<Campaign purpose and goals>

## Workflows

### <workflow-id-1>
<What this workflow does>

## Timeline

- **Start**: <Date or "TBD">
- **Target**: <Date or "Ongoing">
```

---

## Compilation

Compile the campaign to generate orchestrator:

```bash
gh aw compile <campaign-id>
```

Generated files:
- `.github/workflows/<campaign-id>.campaign.g.md` (orchestrator)
- `.github/workflows/<campaign-id>.campaign.lock.yml` (compiled)

---

## Best Practices

1. **Start simple** - One clear goal per campaign
2. **Use passive mode first** - Monitor before enabling active execution
3. **Reuse workflows** - Check existing before creating new
4. **Minimal permissions** - Grant only what's needed
5. **Escalate when unsure** - Create issues for human review

### DO:
- ✅ Use unique kebab-case campaign IDs
- ✅ Scan existing workflows before suggesting new
- ✅ Apply least privilege for safe outputs
- ✅ Start with passive mode
- ✅ Follow operation order for project-based campaigns

### DON'T:
- ❌ Create duplicate campaign IDs
- ❌ Skip workflow discovery
- ❌ Grant unnecessary permissions
- ❌ Enable execute-workflows for first campaign

---

**Last Updated:** 2026-01-13
