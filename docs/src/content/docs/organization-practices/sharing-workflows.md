---
title: Sharing Workflows
description: Share, reuse, and govern workflows across repositories and organizations.
sidebar:
  badge: { text: 'Platform', variant: 'tip' }
---

Sharing workflows across repositories is an organization practice, not a single design pattern.

Some teams want a central repository that publishes common workflows. Others want reusable imports, shared components, or starter repositories that teams can adopt and customize. The right choice depends on how tightly the organization wants to synchronize behavior across repositories.

## Common Sharing Models

### Shared source repository

One repository publishes workflow sources and other repositories add them with `gh aw add` or track them via `source:` metadata. This is a good fit when a platform team owns the workflow and downstream repositories should receive updates over time.

### Reusable imports and shared components

Shared imports are useful when the organization wants common building blocks rather than one complete workflow. This works well for common MCP configuration, shared prompts, safety policy, or reusable workflow fragments.

### Starter repositories and templates

Starter repositories and workflow templates are useful when teams need a strong starting point but are expected to diverge. This model favors local ownership over synchronized updates.

### Central ops repositories

A central operations repository is useful when workflows need durable memory, reporting, review, or organization-wide coordination. In that model, individual product repositories often emit signals while the ops repository owns the long-lived automation loop.

## Choosing Between Them

Use a shared source repository when consistency matters more than local autonomy.

Use imports when the common unit is a capability or policy fragment.

Use templates when the organization wants fast adoption but expects teams to customize independently.

Use a central ops repository when workflows need shared history, review queues, reporting, or cross-repository orchestration.

## Governance Questions

When workflows are shared across an organization, the important questions are usually operational rather than technical:

- who owns the source workflow
- how updates are reviewed and promoted
- which repositories may consume or dispatch to shared workflows
- how secrets, permissions, and safe outputs are standardized
- when teams may fork a workflow rather than stay on the shared source

Those decisions affect reliability more than the file format does.

## Practical Guidance

For synchronized reuse, start with [Reusing Workflows](/gh-aw/guides/packaging-imports/), `gh aw add`, and imports.

For cross-repository control-plane designs, combine this guidance with [SideRepoOps](/gh-aw/patterns/side-repo-ops/) and [MultiRepoOps](/gh-aw/patterns/multi-repo-ops/).

For organizations introducing workflow sharing gradually, it is common to start with templates or starter repositories, then move stable concerns into imports or a shared source repository once conventions have settled.

## Related Documentation

- [Reusing Workflows](/gh-aw/guides/packaging-imports/)
- [Imports Reference](/gh-aw/reference/imports/)
- [SideRepoOps](/gh-aw/patterns/side-repo-ops/)
- [MultiRepoOps](/gh-aw/patterns/multi-repo-ops/)
- [Workflow Structure](/gh-aw/reference/workflow-structure/)
