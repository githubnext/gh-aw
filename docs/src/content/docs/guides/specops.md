---
title: SpecOps
description: Maintain and propagate W3C-style specifications using agentic workflows with the w3c-specification-writer agent
sidebar:
  badge: { text: 'Advanced', variant: 'caution' }
---

SpecOps is a development pattern for maintaining formal technical specifications using agentic workflows. The `w3c-specification-writer` agent creates and updates W3C-style specifications that document APIs, protocols, and system architectures, then propagates those changes to consuming implementations automatically.

Use SpecOps when you need formal, standards-grade documentation that multiple teams or repositories depend on. The workflow ensures specifications stay synchronized with implementations through automated updates.

## What is SpecOps?

SpecOps combines three key components:

1. **Specification Agent** - The `w3c-specification-writer` agent creates formal specifications following W3C conventions
2. **Specification Document** - A markdown file with RFC 2119 requirements, conformance classes, and compliance testing
3. **Propagation Workflows** - Automated workflows that update consuming repositories when specifications change

This pattern ensures specifications remain the single source of truth while keeping implementations synchronized.

## The SpecOps Story

### Step 1: Update the Specification

Use the `w3c-specification-writer` agent to update your specification:

```yaml
---
name: Update MCP Gateway Spec
on:
  workflow_dispatch:
    inputs:
      change_description:
        description: 'What needs to change in the spec?'
        required: true
        type: string

engine: copilot
strict: true

safe-outputs:
  create-pull-request:
    title-prefix: "[spec] "
    labels: [documentation, specification]
    draft: false

tools:
  edit:
  bash:
---

# Specification Update Workflow

You are using the w3c-specification-writer agent to update the
MCP Gateway specification.

**Change Request**: ${{ inputs.change_description }}

## Your Task

1. Review the current specification at 
   `docs/src/content/docs/reference/mcp-gateway.md`

2. Apply the requested changes following W3C conventions:
   - Use RFC 2119 keywords (MUST, SHALL, SHOULD, MAY)
   - Update version number appropriately (major/minor/patch)
   - Add entry to Change Log section
   - Update Status of This Document if needed

3. Ensure all changes maintain:
   - Clear conformance requirements
   - Testable specifications
   - Complete examples
   - Proper cross-references

4. Create a pull request with the updated specification
```

The agent reviews the specification, applies changes following semantic versioning, updates the change log, and creates a pull request for review.

### Step 2: Propagate to Consumers

After the specification is updated and merged, propagate changes to consuming repositories:

```yaml
---
name: Propagate Spec Changes
on:
  push:
    branches:
      - main
    paths:
      - 'docs/src/content/docs/reference/mcp-gateway.md'
  workflow_dispatch:

engine: copilot
strict: true

safe-outputs:
  create-pull-request:
    title-prefix: "[spec-update] "
    labels: [dependencies, specification]

tools:
  github:
    toolsets: [repos, pull_requests]
  edit:
  bash:
---

# Specification Propagation Workflow

The MCP Gateway specification has been updated. Propagate changes
to consuming repositories.

## Consuming Repositories

- **gh-aw-mcpg**: Implementation repository
  - Update compliance with new requirements
  - Adjust configuration schemas
  - Update integration tests
  
- **gh-aw**: Main repository  
  - Update MCP gateway configuration validation
  - Adjust workflow compilation if needed
  - Update documentation links

## Your Task

1. Read the latest specification version and change log
2. Identify breaking changes and new requirements
3. For each consuming repository:
   - Clone or fetch latest code
   - Update implementation to match spec
   - Run tests to verify compliance
   - Create pull request with changes
4. Create tracking issue linking all PRs
```

This workflow automatically updates consuming repositories to maintain compliance with the specification.

## Example: MCP Gateway Specification

The [MCP Gateway Specification](/gh-aw/reference/mcp-gateway/) demonstrates SpecOps in action:

**Specification Document** (`docs/src/content/docs/reference/mcp-gateway.md`):
- Formal W3C-style specification
- RFC 2119 requirement keywords
- Conformance classes and compliance testing
- Semantic versioning with change log

**Implementation Repository** ([gh-aw-mcpg](https://github.com/githubnext/gh-aw-mcpg)):
- Implements the MCP Gateway specification
- Provides Docker container for gateway service
- Must conform to all MUST/SHALL requirements

**Maintenance Workflow** (`layout-spec-maintainer.md`):
- Automatically scans codebase for patterns
- Updates specification with discovered patterns
- Creates pull requests when changes detected

## Key Components

### W3C Specification Writer Agent

Located at `.github/agents/w3c-specification-writer.agent.md`, this agent:

- Creates formal specifications following W3C conventions
- Uses RFC 2119 keywords correctly (MUST, SHALL, SHOULD, MAY)
- Includes conformance requirements and compliance testing
- Maintains semantic versioning and change logs
- Separates normative and informative content

**Specification Structure**:
```markdown
---
title: Your Specification Name
description: One-line description
---

# Your Specification Name

**Version**: X.Y.Z
**Status**: Draft/Candidate/Recommendation/Final

## Abstract
[One paragraph summary]

## Status of This Document
[Publication status and governance]

## 1. Introduction
### 1.1 Purpose
### 1.2 Scope  
### 1.3 Design Goals

## 2. Conformance
### 2.1 Conformance Classes
### 2.2 Requirements Notation
### 2.3 Compliance Levels

## [Numbered Core Sections]
[Technical requirements with RFC 2119 keywords]

## [N]. Compliance Testing
[Test requirements and procedures]

## References
### Normative References
### Informative References

## Change Log
### Version X.Y.Z (Status)
- [Changes]
```

### Semantic Versioning for Specifications

Specification versions follow semantic versioning:

- **Major (X.0.0)** - Breaking changes, incompatible API changes
- **Minor (0.Y.0)** - New features, backward-compatible additions  
- **Patch (0.0.Z)** - Bug fixes, clarifications, editorial changes

Example version progression:
```text
1.0.0 → 1.1.0  (Added optional timeout feature)
1.1.0 → 1.1.1  (Fixed typo in Section 3.2)
1.1.1 → 2.0.0  (Removed deprecated field, breaking change)
```

### Specification Consumers

Consumers are repositories or systems that implement the specification:

**Implementation Repository** (e.g., `gh-aw-mcpg`):
- Implements all MUST/SHALL requirements
- May implement SHOULD/MAY features
- Runs compliance tests to verify conformance
- Documents compliance level

**Dependent Repositories** (e.g., `gh-aw`):
- Use the specification for integration
- Validate configuration against spec schemas
- Update when specification changes
- May contribute feedback to specification

## Propagation Strategies

### Pull-Based Propagation

Consumers periodically check for specification updates:

```yaml
on:
  schedule:
    - cron: '0 8 * * 1'  # Weekly Monday 8am
  workflow_dispatch:
```

The workflow compares local implementation against latest specification version and creates update PRs when differences are detected.

### Push-Based Propagation  

Specification repository triggers updates when changes are published:

```yaml
on:
  push:
    branches:
      - main
    paths:
      - 'docs/src/content/docs/reference/*.md'
```

The workflow immediately notifies or updates consuming repositories using GitHub API or repository dispatch events.

### Hybrid Approach

Combine both strategies:
- Push-based for critical breaking changes (immediate updates)
- Pull-based for minor changes (scheduled reviews)
- Manual workflow_dispatch for testing and special cases

## Compliance Testing

Specifications should define testable requirements:

```markdown
## 10. Compliance Testing

### 10.1 Configuration Tests

- **T-CFG-001**: Gateway MUST reject configuration with 
  unknown top-level fields
- **T-CFG-002**: Gateway MUST validate server container 
  image format
- **T-CFG-003**: Gateway SHALL enforce authentication 
  when token is configured

### 10.2 Protocol Tests  

- **T-PRO-001**: Gateway MUST translate stdio to HTTP correctly
- **T-PRO-002**: Gateway MUST isolate server instances
- **T-PRO-003**: Gateway SHOULD handle timeout gracefully

### 10.3 Compliance Checklist

| Requirement | Test ID | Level | Status |
|-------------|---------|-------|--------|
| Config validation | T-CFG-001 | 1 | Required |
| Image format | T-CFG-002 | 1 | Required |
| Authentication | T-CFG-003 | 1 | Required |
| Protocol translation | T-PRO-001 | 2 | Required |
| Server isolation | T-PRO-002 | 2 | Required |
| Timeout handling | T-PRO-003 | 3 | Optional |
```

Consuming repositories implement these tests to verify compliance.

## Workflow Automation Patterns

### Pattern 1: Spec-First Development

Update specification before implementation:

```text
1. Propose spec change via PR
2. Review and merge spec update
3. Propagation workflow creates implementation PRs
4. Implement changes to satisfy new requirements
5. Run compliance tests
6. Merge implementation PRs
```

### Pattern 2: Implementation-First Discovery

Discover patterns from implementation, then formalize in spec:

```text
1. Implement feature in consumer repository
2. Layout-spec-maintainer workflow scans code
3. Extracts patterns and updates specification
4. Creates PR with discovered patterns
5. Review and formalize in specification
6. Update other consumers with formalized patterns
```

This is how `layout-spec-maintainer` works - it scans compiled workflows and codebases to discover patterns, then documents them in `specs/layout.md`.

### Pattern 3: Continuous Compliance

Regularly verify implementation matches specification:

```yaml
---
name: Compliance Check
on:
  schedule:
    - cron: '0 6 * * *'  # Daily at 6am
  workflow_dispatch:

engine: copilot

tools:
  github:
    toolsets: [default]
---

# Daily Compliance Check

Verify gh-aw-mcpg implementation matches MCP Gateway 
specification requirements.

## Your Task

1. Read the MCP Gateway specification
2. Review gh-aw-mcpg implementation  
3. Run compliance tests from specification section 10
4. Report any non-conformance as issues
5. Create PR to fix any compliance gaps
```

## Best Practices

### Specification Maintenance

**Clear Requirements**: Use RFC 2119 keywords consistently
```markdown
✅ GOOD: "The gateway MUST validate all configuration fields"
❌ AVOID: "The gateway should probably validate configuration"
```

**Testable Statements**: Every MUST/SHALL requirement should be testable
```markdown
✅ GOOD: "The gateway MUST return HTTP 401 when token is invalid"
❌ AVOID: "The gateway should handle authentication properly"
```

**Version Discipline**: Follow semantic versioning strictly
```markdown
✅ GOOD: Version 2.0.0 for removing deprecated feature
❌ AVOID: Version 1.1.0 for breaking change
```

### Change Management

**Document Everything**: Update change log with every modification
```markdown
### Version 1.5.0 (Draft)
- **Added**: Support for HTTP-based MCP servers
- **Changed**: Clarified server isolation requirements  
- **Fixed**: Corrected authentication flow example
```

**Communicate Breaking Changes**: Highlight breaking changes clearly
```markdown
> [!CAUTION]
> **Breaking Change in Version 2.0.0**: The `legacy-field` 
> configuration option has been removed. Use `new-field` instead.
```

**Provide Migration Guides**: Help consumers update implementations
```markdown
## Migration from 1.x to 2.0

1. Replace `legacy-field: value` with `new-field: value`
2. Update authentication tokens to new format
3. Run compliance tests to verify migration
```

### Automation Tips

**Use Caching**: Store specification state between runs
```yaml
cache:
  - key: spec-state-${{ github.run_id }}
    path: /tmp/gh-aw/spec-cache
    restore-keys: |
      spec-state-
```

**Parallel Updates**: Update multiple consumers simultaneously
```yaml
strategy:
  matrix:
    repo: [consumer-1, consumer-2, consumer-3]
```

**Track Dependencies**: Maintain list of specification consumers
```yaml
# .github/spec-consumers.yml
consumers:
  - repo: githubnext/gh-aw-mcpg
    compliance_level: full
  - repo: githubnext/gh-aw
    compliance_level: partial
```

## Common Challenges

### Challenge: Breaking Changes

**Problem**: Specification changes break existing implementations

**Solution**: 
- Use semantic versioning to signal breaking changes
- Maintain backward compatibility when possible
- Provide deprecation notices before removal
- Create automated migration workflows

### Challenge: Spec Drift

**Problem**: Implementation diverges from specification over time

**Solution**:
- Run regular compliance checks (daily/weekly)
- Automate spec-to-implementation comparisons
- Block merges that violate spec requirements
- Use continuous compliance workflows

### Challenge: Multi-Consumer Coordination

**Problem**: Multiple consumers need coordinated updates

**Solution**:
- Create tracking issues that link all consumer PRs
- Use GitHub Projects for coordination dashboard
- Implement rollout phases (canary → staging → production)
- Test consumer dependencies before propagation

## Example Workflows

### Example 1: Automated Spec Update

```yaml
---
name: Layout Spec Maintainer
on:
  schedule:
    - cron: '0 7 * * 1-5'  # Weekday mornings
  workflow_dispatch:

engine: copilot
strict: true

safe-outputs:
  create-pull-request:
    title-prefix: "[specs] "
    labels: [documentation, automation]

tools:
  github:
    toolsets: [default]
  bash:
    - "find .github/workflows -name '*.lock.yml'"
    - "yq '.*' .github/workflows/*.lock.yml"
---

# Layout Specification Maintainer

Scan compiled workflows and extract patterns, then update
`specs/layout.md` with discovered patterns.

[Workflow details in `.github/workflows/layout-spec-maintainer.md`]
```

### Example 2: Cross-Repository Propagation

```yaml
---
name: Spec Change Propagator
on:
  repository_dispatch:
    types: [spec-updated]

engine: copilot

safe-outputs:
  create-pull-request:
    title-prefix: "[spec-compliance] "

tools:
  github:
    toolsets: [repos, pull_requests, issues]
---

# Specification Change Propagator

A specification has been updated. Propagate changes to
all consuming repositories.

**Updated Spec**: ${{ github.event.client_payload.spec_name }}
**Version**: ${{ github.event.client_payload.version }}

## Tasks

1. Read specification at ${{ github.event.client_payload.spec_url }}
2. Get list of consumers from `.github/spec-consumers.yml`
3. For each consumer:
   - Update implementation files
   - Run compliance tests  
   - Create pull request
4. Create tracking issue with links to all PRs
```

## Related Patterns

- **[Campaign Specs](/gh-aw/guides/campaigns/specs/)** - Define campaigns with spec files for reviewable contracts
- **[MultiRepoOps](/gh-aw/guides/multirepoops/)** - Coordinate work across multiple repositories
- **[SideRepoOps](/gh-aw/guides/siderepoops/)** - Run workflows from separate repository

## References

- [W3C Specification Writer Agent](https://github.com/githubnext/gh-aw/blob/main/.github/agents/w3c-specification-writer.agent.md)
- [MCP Gateway Specification](/gh-aw/reference/mcp-gateway/)
- [MCP Gateway Implementation (gh-aw-mcpg)](https://github.com/githubnext/gh-aw-mcpg)
- [Layout Spec Maintainer Workflow](https://github.com/githubnext/gh-aw/blob/main/.github/workflows/layout-spec-maintainer.md)
- [RFC 2119: Key words for use in RFCs to Indicate Requirement Levels](https://www.ietf.org/rfc/rfc2119.txt)
