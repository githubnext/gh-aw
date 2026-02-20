---
title: CentralRepoOps
description: Operate and roll out changes across many repositories from a single private control repository.
sidebar:
  badge: { text: 'Enterprise', variant: 'caution' }
draft: true
---

CentralRepoOps is a [MultiRepoOps](/gh-aw/patterns/multirepoops/) deployment variant where a single private repository acts as a control plane for large-scale operations across many repositories.

Use this pattern when you need to coordinate rollouts, policy updates, and tracking across tens or hundreds of repositories from a private central location, using cross-repository safe outputs and secure authentication to deliver consistency, control, and auditability.

## When to Use CentralRepoOps

- **Organization-wide rollouts** - Apply one change pattern across dozens or hundreds of repositories.
- **Central governance** - Keep prioritization, approvals, and reporting in one control repository.
- **Phased adoption** - Roll out by waves (pilot first, then broader groups).
- **Security operations** - Prioritize repositories with alerts or higher exposure.

## Example: Dependabot Rollout (Orchestrator + Worker)

This pattern maps directly to your Dependabot rollout pair:

- `dependabot-rollout-orchestrator.md` decides *where* to roll out next.
- `dependabot-rollout.md` executes *how* to configure each target repository.

### Orchestrator (central control)

```aw wrap
---
on:
  schedule:
    - cron: '0 9 * * 1'
  workflow_dispatch:
    inputs:
      target_repos:
        description: 'List of repos (owner/repo1, owner/repo2)'
        required: false
        type: string

tools:
  github:
    toolsets: [repos]

safe-outputs:
  dispatch-workflow:
    workflows: [dependabot-rollout]
    max: 5
---

# Dependabot Rollout Orchestrator

1. Filter repos that already have `.github/dependabot.yml`
2. Categorize candidates (simple, security, complex, conflicting)
3. Prioritize and select top repos
4. Dispatch `dependabot-rollout` worker per selected repo
5. Summarize decisions and rationale
```

### Worker (repo-local execution)

```aw wrap
---
on:
  workflow_dispatch:
    inputs:
      target_repo:
        description: 'Target repository (owner/repo format)'
        required: true
        type: string

engine:
  id: copilot
  steps:
    - name: Checkout target repository
      uses: actions/checkout@v5
      with:
        repository: ${{ github.event.inputs.target_repo }}
        token: ${{ secrets.GH_AW_GITHUB_TOKEN }}

safe-outputs:
  github-token: ${{ secrets.GH_AW_GITHUB_TOKEN }}
  create-pull-request:
    target-repo: ${{ github.event.inputs.target_repo }}
    title-prefix: '[dependabot] '
  create-issue:
    target-repo: ${{ github.event.inputs.target_repo }}
    title-prefix: '[dependabot-config] '
    max: 1
---
```

The worker analyzes each repository and either creates a customized Dependabot PR or opens an issue when conflicts (for example Renovate) require migration planning.

## Setup This Example

Use this checklist to run the Dependabot rollout pattern in a control repository:

1. **Add both workflow sources** to your control repo: `.github/workflows/dependabot-rollout-orchestrator.md` and `.github/workflows/dependabot-rollout.md`.
2. **Compile workflows** so lock files are generated: `gh aw compile`.
3. **Add required secrets** in the control repository (or organization secrets):
    * [`COPILOT_GITHUB_TOKEN`](/gh-aw/reference/auth/#copilot_github_token) for GitHub Copilot access.
    * [`GH_AW_GITHUB_TOKEN`](/gh-aw/reference/auth/#gh_aw_github_token) for cross-repo access (checkout and safe outputs):
        - Use a fine-grained PAT with owner the org that owns the repos and permissions:
            - `contents: write` (for PR creation)
            - `issues: write` (for issue creation)
            - `pull-requests: write` (for PR creation)
            - `actions: write` (for private repos)
4. Commit both `.md` and generated `.lock.yml` files.
5. Run the orchestrator with `workflow_dispatch` (`target_repos`) or let the schedule trigger run automatically.

The orchestrator decides which repositories to target; each worker run applies repo-local analysis and then creates either a PR or issue in that target repository. The safe output configuration ensures that all cross-repository operations are authenticated and controlled, with limits on the number of created items to prevent overload.

## Why CentralRepoOps Matters for Enterprises

- **Consistency at scale** - Same rollout logic and policy gates across all repositories
- **Risk reduction** - Controlled fan-out (`max`), phased prioritization, and explicit rationale
- **Auditability** - One orchestrator run provides a full decision trail of selection and outcomes
- **Operational efficiency** - Teams avoid copy-pasting workflows into every repository
- **Security posture** - Prioritize exposed or vulnerable repositories first

## Implementation Guidance

- Keep orchestrator permissions narrow; delegate repo-specific writes to workers.
- Use safe output limits (`max`) and explicit target workflow allowlists.
- Add correlation IDs to worker dispatch inputs for tracking.

## Related Patterns

- **[MultiRepoOps](/gh-aw/patterns/multirepoops/)** - Cross-repository automation capability
- **[Orchestration](/gh-aw/patterns/orchestration/)** - Generic orchestrator/worker dispatch pattern
- **[SideRepoOps (guide)](/gh-aw/guides/ephemerals/#siderepoops-pattern)** - Isolated control-plane setup
