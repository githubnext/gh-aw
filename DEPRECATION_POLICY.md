# Deprecation Policy

This document defines the policy and process for deprecating and removing features from GitHub Agentic Workflows. It ensures consistent decision-making and provides clear guidance for maintainers and users.

## Table of Contents

- [Overview](#overview)
- [Criteria for Deprecation](#criteria-for-deprecation)
- [Deprecation Process](#deprecation-process)
- [Communication Requirements](#communication-requirements)
- [Grace Periods](#grace-periods)
- [Migration Support](#migration-support)
- [Breaking Change Releases](#breaking-change-releases)
- [Special Considerations](#special-considerations)
- [Examples](#examples)

## Overview

GitHub Agentic Workflows is committed to maintaining stability while evolving the platform. This policy balances the needs of existing users with the project's long-term health by providing a predictable process for feature deprecation.

**Key Principles:**
- **User-first**: Minimize disruption to existing workflows
- **Transparency**: Communicate changes clearly and early
- **Consistency**: Apply criteria uniformly across all features
- **Gradual transition**: Provide adequate time for migration

## Criteria for Deprecation

Features are evaluated for deprecation based on adoption, maintenance burden, and strategic alignment.

### Usage Thresholds

Usage data guides deprecation decisions but is not the sole factor:

| Usage Level | Threshold | Action |
|-------------|-----------|--------|
| **High adoption** | > 3% of workflows | Keep and maintain |
| **Moderate adoption** | 1-3% of workflows | Monitor, improve discoverability, or consider deprecation if maintenance burden is high |
| **Low adoption** | < 1% of workflows | Strong candidate for deprecation |
| **No adoption** | 0% usage after 2+ releases | Candidate for immediate deprecation |

**Note**: Usage is measured across public and internal workflows where data is available.

### Other Deprecation Criteria

Beyond usage statistics, consider:

1. **Maintenance Burden**
   - Complexity of maintaining the feature
   - Frequency of bug reports or issues
   - Dependencies on deprecated external libraries
   - Security implications

2. **Better Alternatives Available**
   - More powerful or flexible replacement exists
   - Alternative aligns better with current architecture
   - Alternative provides better user experience

3. **Strategic Alignment**
   - Feature conflicts with project direction
   - Feature requires significant rearchitecture
   - Feature blocks important improvements

4. **Dead Code**
   - Schema fields defined but never used in implementation
   - Features that have never functioned correctly
   - Experimental features that were never completed

### Examples of Deprecation Decisions

**Deprecate**: A schema field with < 1% usage that adds significant validation complexity and has a better alternative.

**Keep**: A feature with 2% usage that is simple to maintain and has no replacement.

**Improve**: A feature with 1.5% usage that may be underutilized due to poor documentation.

## Deprecation Process

The deprecation process follows a phased approach to ensure users have adequate time to migrate.

### Phase 1: Evaluation (Before Deprecation)

1. **Gather data**: Measure usage, collect user feedback, assess maintenance cost
2. **Identify alternatives**: Document what users should use instead
3. **Propose deprecation**: Create a GitHub discussion with rationale and proposed timeline
4. **Community feedback**: Allow at least 2 weeks for community input
5. **Decision**: Maintainers make final decision based on criteria and feedback

### Phase 2: Announcement (Deprecation Declared)

1. **Update documentation**: Mark feature as deprecated in all docs
2. **Add warnings**: Implement deprecation warnings in code (where applicable)
3. **Create migration guide**: Document how to migrate away from the feature
4. **Communicate broadly**:
   - GitHub discussion announcement
   - CHANGELOG entry
   - Release notes
   - Community channels (#continuous-ai Discord)

### Phase 3: Grace Period (Deprecated but Functional)

1. **Duration**: Minimum grace period (see [Grace Periods](#grace-periods) section)
2. **Support**: Feature remains fully functional with deprecation warnings
3. **Migration assistance**: Help users migrate via discussions and documentation
4. **Monitor migration**: Track usage decline during grace period

### Phase 4: Removal (Breaking Change)

1. **Bundle with v2.0**: Schedule removal for next major version release (see [Breaking Change Releases](#breaking-change-releases))
2. **Final notice**: Announce removal at least 1 month before major release
3. **Remove code**: Remove deprecated feature in major version
4. **Update schema**: Remove deprecated fields from schema
5. **Document removal**: Add migration guide to CHANGELOG with clear before/after examples

## Communication Requirements

Effective communication ensures users are not surprised by deprecations.

### Required Communication Channels

For every deprecation, announce through:

1. **GitHub Discussion**
   - Create a discussion in the repository
   - Title format: "Deprecation Notice: [Feature Name]"
   - Include rationale, alternatives, and timeline
   - Pin discussion during grace period

2. **CHANGELOG.md**
   - Add entry in "Deprecated" section under current version
   - Include migration guidance
   - Example format:
     ```markdown
     ### Deprecated
     
     - **`field-name` in workflow schema**: This field is deprecated and will be removed in v2.0. Use `new-field-name` instead. See [migration guide](#).
     ```

3. **Schema Warnings** (where applicable)
   - Add deprecation notice to JSON schema descriptions
   - Example:
     ```json
     {
       "deprecated": true,
       "description": "DEPRECATED: Use 'new-field' instead. This field will be removed in v2.0. See https://example.com/migration"
     }
     ```

4. **Runtime Warnings** (where applicable)
   - Print deprecation warning to stderr when feature is used
   - Use console formatting: `console.FormatWarningMessage()`
   - Include migration guidance in warning

5. **Documentation Updates**
   - Mark as deprecated in all documentation
   - Add visible warning boxes
   - Update examples to show alternatives

### Communication Timeline

- **Initial announcement**: When deprecation is declared (Phase 2 start)
- **Periodic reminders**: At each minor release during grace period
- **Final warning**: 1 month before removal (before Phase 4)
- **Removal notice**: In major release notes (Phase 4)

## Grace Periods

Grace periods provide users time to adapt to changes without disruption.

### Minimum Grace Periods

| Deprecation Type | Minimum Grace Period | Reasoning |
|-----------------|---------------------|-----------|
| **CLI commands/flags** | 3 months or 2 minor releases | Scripts and automation need update time |
| **Schema fields (high usage >3%)** | 6 months or 3 minor releases | Workflows need review and updates |
| **Schema fields (low usage <1%)** | 3 months or 2 minor releases | Fewer users affected |
| **Dead code (0% usage)** | 1 month or 1 minor release | No user impact |
| **Security-related** | Immediate removal allowed | Security takes priority |

**Note**: "Minor releases" refers to releases that bump the minor version number (e.g., v0.33.0 → v0.34.0).

### Grace Period Extensions

Grace periods may be extended if:
- Significant number of users still use the feature
- Migration path is more complex than anticipated
- Users request more time with valid reasoning
- Alternative implementation is delayed

Maintainers may announce extensions via GitHub discussion and CHANGELOG.

## Migration Support

Help users transition away from deprecated features.

### Migration Guide Requirements

Every deprecation must include a migration guide with:

1. **Clear problem statement**: What is being deprecated and why
2. **Alternative approach**: What to use instead
3. **Before/after examples**: Show code transformation
4. **Breaking changes**: List what will stop working
5. **Timeline**: When feature will be removed
6. **Support channels**: Where to get help

### Migration Guide Template

```markdown
## Migration Guide: [Feature Name]

### What's Changing

[Brief description of what's being deprecated]

### Why This Change

[Rationale for deprecation]

### Alternative Approach

Use [new feature] instead. [Brief description of alternative]

### Migration Steps

1. [Step 1]
2. [Step 2]
3. [Step 3]

### Before (Deprecated)

```yaml
# Old approach
deprecated-field: value
```

### After (Recommended)

```yaml
# New approach
new-field: value
```

### Timeline

- **Deprecated**: [Date]
- **Removal**: [Planned version]
- **Grace Period**: [Duration]

### Getting Help

- GitHub Discussion: [Link]
- Discord: #continuous-ai channel
```

### Migration Assistance

During the grace period:

1. **Answer questions**: Respond to migration questions in discussions
2. **Provide examples**: Add real-world migration examples
3. **Update tooling**: Consider providing automated migration tools for complex changes
4. **Monitor issues**: Track migration-related issues and provide guidance

## Breaking Change Releases

Major version releases (e.g., v2.0) are used to bundle breaking changes including feature removals.

### v2.0 Planning Process

1. **Create milestone**: Create a v2.0 milestone in GitHub Issues
2. **Track deprecations**: Add all planned removals to the milestone
3. **Bundle removals**: Schedule multiple deprecated feature removals for same major release
4. **Plan release date**: Ensure all grace periods are satisfied
5. **Prepare documentation**: Create comprehensive v2.0 migration guide
6. **Release preparation**: Final testing and validation before release

### Release Requirements

Before creating a major version release:

- [ ] All deprecated features have completed minimum grace period
- [ ] Migration guides are complete and tested
- [ ] v2.0 migration guide consolidates all breaking changes
- [ ] CHANGELOG includes all breaking changes with migration guidance
- [ ] Pre-release announcement (at least 1 month before release)
- [ ] Community has been notified and questions answered

### Bundling Strategy

Bundle related deprecations together:
- Group by domain (e.g., all schema field removals)
- Group by workflow type (e.g., all CLI command changes)
- Consider user impact (avoid too many changes in one domain)

### Major Version Frequency

- Target one major version per year maximum
- Allow time between major versions for users to stabilize
- Avoid surprise major versions - always plan and communicate early

## Special Considerations

### Security-Related Deprecations

Security issues override normal grace periods:

- **Critical vulnerabilities**: May be removed immediately
- **High severity**: Minimum 1 week notice if possible
- **Medium severity**: Follow standard grace period but prioritize removal
- **Communication**: Explain security rationale clearly

Example: If a feature enables template injection attacks, it may be removed immediately with a patch release and clear security advisory.

### Experimental Features

Features marked as experimental follow different rules:

- **Grace period**: 1 month or 1 minor release minimum
- **Removal threshold**: May be removed with lower usage thresholds
- **Communication**: Same requirements but emphasize experimental nature

Mark experimental features clearly:
```yaml
# Schema example
experimental-feature:
  type: string
  description: "⚠️ EXPERIMENTAL: This feature is experimental and may change or be removed without following standard deprecation policy."
```

### Backward Compatibility Maintenance

When possible, maintain backward compatibility:

- **Aliases**: Provide aliases for renamed commands/fields
- **Automatic migration**: Transform old format to new automatically
- **Default behavior**: Keep defaults consistent where possible

Only use breaking changes when backward compatibility is not feasible or creates technical debt.

## Examples

### Example 1: Low-Adoption Schema Field

**Scenario**: A schema field with 0.5% usage that adds validation complexity.

**Process**:
1. **Evaluation**: Usage data shows < 1%, validation code is complex, better alternative exists
2. **Proposal**: GitHub discussion proposing deprecation with 3-month grace period
3. **Decision**: Approved after 2-week feedback period
4. **Announcement**: CHANGELOG entry, schema deprecation warning, migration guide created
5. **Grace Period**: 3 months (2 minor releases), feature remains functional with warnings
6. **Removal**: Scheduled for v2.0, removed after grace period ends

**Timeline**:
- **Month 1**: Deprecation announced in v0.34.0
- **Month 2**: Reminder in v0.35.0
- **Month 3**: Final warning before v0.36.0
- **Month 4+**: Removal in v2.0.0

### Example 2: Dead Code Field

**Scenario**: A schema field that was never implemented in the workflow compiler.

**Process**:
1. **Evaluation**: Code review shows field is parsed but never used, 0% usage
2. **Proposal**: Quick deprecation with 1-month grace period (dead code)
3. **Decision**: Approved as dead code removal
4. **Announcement**: CHANGELOG entry and migration guide (noting no action needed)
5. **Grace Period**: 1 month (1 minor release)
6. **Removal**: Next major version

**Timeline**:
- **Week 1**: Deprecation announced in v0.34.0
- **Week 5**: Removal in v2.0.0

### Example 3: CLI Command Deprecation

**Scenario**: A CLI command with 2% usage being replaced by a better alternative.

**Process**:
1. **Evaluation**: Usage data shows moderate adoption, new command provides better UX
2. **Proposal**: Deprecation with 6-month grace period and alias support
3. **Community Feedback**: Some users request extended grace period
4. **Decision**: Approved with 6-month grace period
5. **Implementation**: Add alias, deprecation warning, update documentation
6. **Grace Period**: 6 months (3 minor releases)
7. **Removal**: v2.0 with alias removed

**Timeline**:
- **Month 1**: Deprecation announced in v0.34.0, alias added
- **Month 3**: Reminder in v0.35.0
- **Month 5**: Reminder in v0.36.0
- **Month 6**: Final warning 1 month before v2.0
- **Month 7**: Removal in v2.0.0

## Related Documentation

- [Breaking CLI Rules](specs/breaking-cli-rules.md) - Defines what constitutes a breaking change
- [Changesets](specs/changesets.md) - Version management and release process
- [CONTRIBUTING.md](CONTRIBUTING.md) - How to contribute changes
- [CHANGELOG.md](CHANGELOG.md) - Historical record of changes

## Questions?

For questions about this policy:
- Open a GitHub discussion
- Ask in #continuous-ai Discord channel
- Reference this policy in issues

---

**Last Updated**: 2026-01-01
**Version**: 1.0
