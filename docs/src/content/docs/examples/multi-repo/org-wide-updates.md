---
title: Organization-Wide Updates
description: Coordinate dependency updates, security patches, and policy enforcement across multiple repositories in an organization.
sidebar:
  badge: { text: 'Multi-Repo', variant: 'note' }
---

Organization-wide update workflows enable coordinating changes across multiple repositories, from dependency updates to security patches and policy enforcement. These patterns ensure consistency while respecting repository-specific requirements.

## When to Use

- **Dependency updates** - Coordinate version updates across related projects
- **Security patches** - Apply critical security fixes organization-wide
- **Policy enforcement** - Ensure compliance with security and quality standards
- **Configuration standardization** - Maintain consistent tooling configurations
- **Breaking change coordination** - Manage API updates affecting multiple repositories

## How It Works

Workflows query organization repositories, identify those requiring updates, and create tracking issues or pull requests in each affected repository. This approach provides visibility into rollout progress while allowing repository-specific adaptations.

## Dependency Update Coordination

Identify repositories using specific dependencies and create update tracking issues:

```aw wrap
---
on:
  workflow_dispatch:
    inputs:
      package_name:
        description: 'Package name to update'
        required: true
        type: string
      current_version:
        description: 'Current version pattern (e.g., ^1.0.0)'
        required: true
        type: string
      target_version:
        description: 'Target version (e.g., ^2.0.0)'
        required: true
        type: string
permissions:
  contents: read
tools:
  github:
    toolsets: [repos]
safe-outputs:
  github-token: ${{ secrets.ORG_ADMIN_PAT }}
  create-issue:
    max: 50
    title-prefix: "[deps] "
    labels: [dependencies, org-wide-update]
---

# Coordinate Organization-Wide Dependency Update

Find all repositories using ${{ github.event.inputs.package_name }} and
create tracking issues for coordinated update.

**Update Details:**
- Package: ${{ github.event.inputs.package_name }}
- Current version: ${{ github.event.inputs.current_version }}
- Target version: ${{ github.event.inputs.target_version }}

**Process:**
1. Search code across organization for package.json/requirements.txt/go.mod files
2. Parse dependency files to find packages matching criteria
3. For each repository using the dependency:
   - Check current version
   - Assess breaking changes between versions
   - Estimate update complexity
   - Create tracking issue

**Issue should include:**
- Current version in use
- Target version with changes summary
- Breaking changes (if any)
- Migration guide link
- Estimated effort (small/medium/large)
- Testing recommendations
- Deadline (if security-related)

**Repositories to prioritize:**
- Production services first
- High-traffic applications
- Security-sensitive components
```

## Security Patch Rollout

Coordinate critical security patch deployment:

```aw wrap
---
on:
  workflow_dispatch:
    inputs:
      cve_id:
        description: 'CVE identifier'
        required: true
        type: string
      affected_package:
        description: 'Affected package name'
        required: true
        type: string
      patched_version:
        description: 'Patched version'
        required: true
        type: string
permissions:
  contents: read
tools:
  github:
    toolsets: [repos, code_security]
safe-outputs:
  github-token: ${{ secrets.SECURITY_PAT }}
  create-issue:
    max: 100
    title-prefix: "[SECURITY] "
    labels: [security, critical, cve-patch]
    assignees: [security-team]
---

# Critical Security Patch Rollout

Coordinate security patch deployment for ${{ github.event.inputs.cve_id }}
affecting ${{ github.event.inputs.affected_package }}.

**Security Details:**
- CVE: ${{ github.event.inputs.cve_id }}
- Package: ${{ github.event.inputs.affected_package }}
- Patched version: ${{ github.event.inputs.patched_version }}

**Rollout Process:**
1. Search all repositories for affected package
2. Check for Dependabot alerts related to this CVE
3. Prioritize by:
   - Production vs. development repositories
   - Public-facing vs. internal services
   - Severity of exposure
4. Create high-priority issues in each affected repo

**Issue Requirements:**
- SECURITY prefix in title
- CVE details and severity
- Link to security advisory
- Upgrade instructions
- Testing checklist
- Deployment deadline (24-48 hours for critical)

**Alert security team for:**
- Production repositories
- Public-facing services
- Repositories with existing exploitation evidence
```

## Policy Enforcement Audit

Audit and enforce organization security policies:

```aw wrap
---
on:
  schedule:
    - cron: "0 10 * * 1"  # Monday 10AM
permissions:
  contents: read
tools:
  github:
    toolsets: [repos, code_security]
safe-outputs:
  github-token: ${{ secrets.ORG_ADMIN_PAT }}
  create-issue:
    max: 20
    title-prefix: "[policy] "
    labels: [compliance, security-policy]
---

# Organization Security Policy Audit

Audit all repositories for compliance with organization security policies.

**Policies to check:**
1. Branch protection on default branch
2. Required pull request reviews (minimum 1)
3. Required status checks before merge
4. Secret scanning enabled
5. Dependabot alerts enabled
6. Code scanning enabled (for appropriate languages)
7. No force push to default branch
8. Signed commits required

**Process:**
For each repository in organization:
1. Check repository settings against policy requirements
2. Identify non-compliant repositories
3. Create issue for each non-compliant repo with:
   - List of policy violations
   - Remediation steps
   - Links to security policy documentation
   - Compliance deadline
   - Escalation path

**Issue template:**
```
## Security Policy Compliance Required

This repository is not compliant with organization security policies.

### Violations:
- [ ] Missing branch protection
- [ ] No required reviews
- [ ] Secret scanning disabled
- [ ] Dependabot not enabled

### Remediation:
1. Enable branch protection: Settings → Branches → Add rule
2. Configure required reviews: Require 1+ approvals
3. Enable security features: Settings → Security

**Compliance deadline:** [Date - 14 days from issue creation]
**Documentation:** [Link to security policy]
```

**Exempt repositories:** Skip archived, template, or specifically exempt repos
```

## Configuration Standardization

Standardize tooling configurations across repositories:

```aw wrap
---
on:
  workflow_dispatch:
    inputs:
      config_type:
        description: 'Configuration type to standardize'
        required: true
        type: choice
        options:
          - 'eslint'
          - 'prettier'
          - 'dependabot'
          - 'codeowners'
permissions:
  contents: read
  actions: read
tools:
  github:
    toolsets: [repos]
  edit:
  bash:
    - "git:*"
safe-outputs:
  github-token: ${{ secrets.ORG_ADMIN_PAT }}
  create-pull-request:
    max: 30
    title-prefix: "[config] "
    labels: [configuration, standardization]
    draft: true
---

# Standardize Organization Configuration

Apply standard ${{ github.event.inputs.config_type }} configuration
across organization repositories.

**Configuration type:** ${{ github.event.inputs.config_type }}

**Process:**
1. Identify repositories needing this configuration
   - For eslint/prettier: JavaScript/TypeScript repos
   - For dependabot: All repos with dependencies
   - For codeowners: Repos with multiple teams

2. For each target repository:
   - Check if configuration exists
   - Compare with organization standard
   - If different or missing:
     - Create branch with standard config
     - Create PR with changes
     - Include rationale and migration notes

**Standard configurations from:**
- `org/.github` repository (organization defaults)
- Apply appropriate config file for each type

**PR description template:**
```
## Configuration Standardization: [config_type]

This PR applies the organization standard configuration for [config_type].

### Changes:
- Adds/updates [config file]
- Aligns with organization standards
- Maintains existing custom rules where applicable

### Benefits:
- Consistent tooling across organization
- Reduced maintenance overhead
- Better code quality

### Testing:
- [ ] Configuration validated
- [ ] Existing code checked
- [ ] No breaking changes

**Review:** Please review customizations vs. standard config
```
```

## Breaking Change Migration

Coordinate breaking API changes across dependent repositories:

```aw wrap
---
on:
  release:
    types: [published]
permissions:
  contents: read
  actions: read
tools:
  github:
    toolsets: [repos, issues]
safe-outputs:
  github-token: ${{ secrets.ORG_ADMIN_PAT }}
  create-issue:
    max: 50
    title-prefix: "[breaking-change] "
    labels: [breaking-change, api-migration]
---

# Coordinate Breaking Change Migration

When a new release with breaking changes is published, create migration
issues in all dependent repositories.

**Release:** ${{ github.event.release.tag_name }}
**Breaking changes:** Parse release notes for BREAKING CHANGE markers

**Process:**
1. Identify breaking changes from release notes
2. Search organization for repositories depending on this one
3. For each dependent repository:
   - Assess impact of breaking changes
   - Estimate migration effort
   - Create migration tracking issue

**Migration issue includes:**
- Link to release notes
- List of breaking changes affecting this repo
- Migration guide (step-by-step)
- Code examples (before/after)
- Testing requirements
- Recommended migration timeline
- Support contact

**Priority levels:**
- High: Production services, public APIs
- Medium: Internal services, development tools
- Low: Example repositories, archived projects

**Include deprecation timeline if applicable**
```

## License Compliance Check

Audit dependencies for license compliance:

```aw wrap
---
on:
  schedule:
    - cron: "0 9 1 * *"  # First day of month, 9AM
permissions:
  contents: read
tools:
  github:
    toolsets: [repos]
safe-outputs:
  github-token: ${{ secrets.ORG_ADMIN_PAT }}
  create-issue:
    max: 30
    title-prefix: "[license] "
    labels: [compliance, license-audit]
---

# Organization License Compliance Audit

Audit all repositories for dependency license compliance.

**Approved licenses:**
- MIT, Apache-2.0, BSD-3-Clause, ISC
- Others require legal review

**Process:**
1. For each repository with dependencies:
   - Scan package manager files
   - Identify dependency licenses
   - Flag non-approved licenses
   - Create issue for non-compliant repos

**Issue details:**
- List of dependencies with non-approved licenses
- License types found
- Risk assessment
- Recommended actions:
  - Replace with approved alternative
  - Request legal review
  - Document exception rationale

**Compliance requirements:**
- All production dependencies must use approved licenses
- Development dependencies: more flexibility
- Document all exceptions

**Include links to:**
- Organization license policy
- Legal review request process
- Approved license list
```

## Workflow Template Deployment

Deploy standardized workflow templates across repositories:

```aw wrap
---
on:
  workflow_dispatch:
    inputs:
      workflow_name:
        description: 'Workflow template to deploy'
        required: true
        type: choice
        options:
          - 'ci-tests'
          - 'security-scan'
          - 'dependency-review'
          - 'stale-issues'
permissions:
  contents: read
  actions: read
tools:
  github:
    toolsets: [repos, actions]
  edit:
  bash:
    - "git:*"
safe-outputs:
  github-token: ${{ secrets.ORG_ADMIN_PAT }}
  create-pull-request:
    max: 40
    title-prefix: "[workflow] "
    labels: [github-actions, automation]
    draft: true
---

# Deploy Workflow Template Organization-Wide

Deploy ${{ github.event.inputs.workflow_name }} workflow template
to all appropriate repositories.

**Workflow:** ${{ github.event.inputs.workflow_name }}

**Target identification:**
- ci-tests: Repositories with test directories
- security-scan: All repositories with code
- dependency-review: Repositories with dependencies
- stale-issues: Active repositories with issues enabled

**Deployment process:**
1. Identify eligible repositories
2. Check if workflow already exists
3. For new deployments:
   - Add workflow file to .github/workflows/
   - Configure for repository specifics
   - Create PR with workflow
4. For existing workflows:
   - Compare with template
   - Update if outdated
   - Preserve custom configurations

**PR includes:**
- Workflow file
- Configuration documentation
- Usage examples
- Maintenance notes

**Allow repository-specific customization:**
- Document customization points
- Preserve existing custom workflows
```

## Authentication and Permissions

Organization-wide operations require elevated permissions:

### Organization Admin PAT

```bash
# Create PAT with organization access
gh auth token

# Required permissions:
# - admin:org (for organization settings)
# - repo (for all repositories)
# - workflow (for GitHub Actions)

# Store as organization secret
gh secret set ORG_ADMIN_PAT --org myorg --body "ghp_your_token_here"
```

### GitHub App for Organization

```yaml wrap
safe-outputs:
  app:
    app-id: ${{ vars.ORG_APP_ID }}
    private-key: ${{ secrets.ORG_APP_PRIVATE_KEY }}
    owner: "myorg"  # Organization name
    # Empty repositories list = all repos in installation
  create-issue:
    max: 100
```

**Benefits:**
- Fine-grained permissions
- Automatic token revocation
- Clear audit trail
- No personal token needed

## Best Practices

### Planning and Coordination

1. **Communicate in advance** - Notify teams before org-wide changes
2. **Pilot with subset** - Test on sample repos before full rollout
3. **Document expectations** - Clear requirements and deadlines
4. **Provide support** - Dedicated help channel during rollout
5. **Monitor progress** - Track completion across repositories

### Issue Management

1. **Use consistent labels** - Enable organization-wide queries
2. **Set reasonable max limits** - Avoid overwhelming teams with issues
3. **Prioritize appropriately** - Critical vs. routine updates
4. **Include clear deadlines** - Especially for security patches
5. **Provide escape hatches** - Exception process for special cases

### Rollout Strategy

1. **Phased approach** - Critical repos first, then others
2. **Time windows** - Give teams reasonable time to respond
3. **Automated follow-up** - Reminder issues for incomplete rollouts
4. **Metrics tracking** - Measure completion rates
5. **Retrospectives** - Learn from each org-wide initiative

### Error Handling

1. **Handle permission errors** - Not all repos may be accessible
2. **Respect repository settings** - Don't override local policies
3. **Provide rollback** - Clear instructions to undo changes
4. **Monitor for failures** - Alert on failed updates
5. **Document exceptions** - Track repos with special requirements

## Related Documentation

- [Multi-Repo Operations Guide](/gh-aw/guides/multi-repo-ops/) - Complete multi-repo overview
- [Feature Synchronization](/gh-aw/examples/multi-repo/feature-sync/) - Code sync patterns
- [Cross-Repo Issue Tracking](/gh-aw/examples/multi-repo/issue-tracking/) - Issue management
- [Safe Outputs Reference](/gh-aw/reference/safe-outputs/) - Configuration options
- [Security Best Practices](/gh-aw/guides/security/) - Security guidelines
