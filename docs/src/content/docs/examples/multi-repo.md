---
title: Multi-Repository Examples
description: Complete examples for managing workflows across multiple GitHub repositories, including feature synchronization, cross-repo tracking, quality monitoring, and organization-wide updates.
---

Multi-repository operations enable coordinating work across multiple GitHub repositories while maintaining security and proper access controls. These examples demonstrate common patterns for cross-repo workflows.

## Featured Examples

### [Triage from Side Repo](/gh-aw/examples/multi-repo/triage-from-side-repo/)

Runs automated issue triage on a main repository from an isolated side repository, with a slash-command bridge for real-time `/triage` response. Keeps all automation logic separate from the main codebase. Use when you want to experiment with agentic triage without touching your main repository.

### [Code Quality Monitoring](/gh-aw/examples/multi-repo/code-quality-monitoring/)

Runs weekly code quality analysis from a side repository by checking out the target codebase locally, running linters and complexity checks, and creating focused actionable issues. Use for ongoing quality gates across repositories you don't want to modify.

### [Feature Synchronization](/gh-aw/examples/multi-repo/feature-sync/)

Automates code synchronization from main repositories to sub-repositories or downstream services through pull requests with change detection, path filters, and bidirectional sync support. Use for monorepo alternatives, shared component libraries, multi-platform deployments, or fork maintenance.

### [Cross-Repository Issue Tracking](/gh-aw/examples/multi-repo/issue-tracking/)

Centralizes issue tracking by automatically creating tracking issues in a central repository with status synchronization and multi-component coordination. Use for component-based architecture visibility, multi-team coordination, cross-project initiatives, or upstream dependency tracking.

### [Dependabot Rollout](/gh-aw/examples/multi-repo/dependabot-rollout/)

Rolls out a customized Dependabot configuration across many repositories using an orchestrator + worker pair from a central control repository. The orchestrator filters and prioritizes targets, then dispatches workers that analyze each repo and create tailored pull requests. Use for org-wide config standardization, security patch rollouts, or any scheduled multi-repo operation.

## Related Documentation

- [MultiRepoOps](/gh-aw/patterns/multi-repo-ops/) - Design patterns for multi-repository workflows
- [Cross-Repository Reference](/gh-aw/reference/cross-repository/) - Checkout and target-repo configuration
- [Safe Outputs Reference](/gh-aw/reference/safe-outputs/) - Configuration options
- [GitHub Tools](/gh-aw/reference/github-tools/) - API access configuration
- [Security Best Practices](/gh-aw/introduction/architecture/) - Authentication and security
- [Reusing Workflows](/gh-aw/guides/reusing-workflows/) - Sharing workflows
