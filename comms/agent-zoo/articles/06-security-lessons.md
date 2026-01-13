# Security Lessons from the Agent Factory

**Designing safe environments where agents can't accidentally cause harm**

[← Previous: Imports & Sharing](05-imports-and-sharing.md) | [Back to Index](../index.md) | [Next: How Workflows Work →](07-how-workflows-work.md)

---

Security is critical in agentic workflows. Peli's Agent Factory taught us that **safety isn't just about permissions** - it's about designing environments where agents can't accidentally cause harm, even when they have the permissions they need to do their jobs.

Running 145 autonomous agents in a production repository required constant vigilance and iterative improvements to our security architecture. Many of the security features in GitHub Agentic Workflows were born from lessons learned in the factory.

This article shares those lessons so you can build secure agent ecosystems from the start.

## Core Security Principles

### 🛡️ Least Privilege, Always

**Start with read-only permissions. Add write permissions only when absolutely necessary and through constrained safe outputs.**

Every workflow begins with `permissions: contents: read`. This is the factory's default stance. Write permissions (`contents: write`, `pull-requests: write`, `issues: write`) are granted sparingly and only through safe output mechanisms.

**Example**: The [`audit-workflows`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/audit-workflows.md) agent has read-only access to workflow runs but creates reports via discussions, which are append-only by nature.

**Why it works**: If an agent can only read, the worst it can do is waste compute. It can't delete code, close important issues, or push malicious changes.

### 🚪 Safe Outputs as the Gateway

**All effectful operations go through safe outputs with built-in limits.**

Safe outputs are the factory's most important security control. They provide a constrained API for agents to interact with GitHub, with guardrails that prevent common mistakes:

**Built-in Protections:**
- Maximum items to create (prevent spam)
- Expiration times (prevent forgotten issues)
- "Close older duplicates" logic (prevent duplication)
- "If no changes" guards (prevent empty PRs)
- Template validation (enforce structure)
- Rate limiting (prevent abuse)

**Example**: An agent creating issues through safe outputs can specify:
```yaml
safe_outputs:
  create_issue:
    title: "Found security vulnerability"
    body: "Details here"
    labels: ["security"]
    max_items: 3  # Only create 3 issues max
    close_older: true  # Close old instances
    expire: "+7d"  # Auto-close if not addressed
```

**Why it works**: Safe outputs transform "can the agent do X?" into "under what constraints can the agent do X?" The agent has power but can't abuse it.

### 👥 Role-Gated Activation

**Powerful agents (fixers, optimizers) require specific roles to invoke.**

Not every mention or workflow event should trigger powerful agents. The factory uses role-gating to ensure only authorized users can invoke sensitive operations.

**Example**: The [`q`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/q.md) optimizer requires the user commenting `/q` to be a repository maintainer. Random contributors can't trigger expensive optimization runs.

**Implementation**:
```yaml
on:
  issue_comment:
    types: [created]

jobs:
  check:
    if: |
      contains(github.event.comment.body, '/q') &&
      (github.event.comment.author_association == 'OWNER' ||
       github.event.comment.author_association == 'MEMBER')
```

**Why it works**: Authorization is enforced at the GitHub platform level, not by the agent. The agent never runs if the user lacks permissions.

### ⏱️ Time-Limited Experiments

**Experimental agents include `stop-after: +1mo` to automatically expire.**

The factory encourages experimentation, but experiments shouldn't run forever. Time limits prevent forgotten demos from consuming resources or causing confusion.

**Example**:
```yaml
---
description: Experimental code deduplication agent
stop-after: +1mo
---
```

After one month, the workflow automatically disables itself. If the experiment was successful, it can be graduated to production without the time limit.

**Why it works**: Explicit expiration forces intentional decisions. Every agent running in the factory was deliberately kept there, not just forgotten.

### 🔍 Explicit Tool Lists

**Workflows declare exactly which tools they use. No ambient authority.**

Every workflow explicitly lists its tool requirements. There's no "give me access to everything" permission. This makes security review straightforward and catches tool misuse early.

**Example**:
```yaml
tools:
  github:
    toolsets: [repos, issues]  # Only repos and issues
  bash:
    commands: [git, jq, python]  # Only these commands
network:
  allowed:
    - "api.github.com"  # Only GitHub API
```

**Why it works**: Explicit > implicit. Reviewers can quickly assess risk. Agents can't accidentally use tools they shouldn't have.

### 📋 Auditable by Default

**Discussions and assets create a natural "agent ledger." You can always trace what an agent did and when.**

Every agent action leaves a trail:
- Issues and PRs are timestamped
- Comments are attributed
- Discussions are permanent
- Artifacts are versioned
- Workflow runs are logged

**Example**: The [`agent-performance-analyzer`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/agent-performance-analyzer.md) creates weekly discussion posts. You can scroll back months to see how agent quality evolved over time.

**Why it works**: Transparency builds trust. When something goes wrong, the audit trail makes debugging straightforward. When something goes right, the evidence is visible.

## Security Patterns

### Pattern 1: Read-Only Analysts

The safest agents are read-only. They observe, analyze, and report but never modify anything.

**Security Properties:**
- ✅ Zero risk of code damage
- ✅ Can't close or modify issues
- ✅ Can't create spam
- ✅ Safe to run at any frequency

**Use case**: Metrics collection, health monitoring, research, auditing

**Example**: All 15 read-only analyst workflows in the factory have perfect security records - zero incidents.

### Pattern 2: Safe Output Bounded Writes

When agents need write access, use safe outputs with strict bounds.

**Security Properties:**
- ✅ Constrained by max items
- ✅ Auto-expiring issues/PRs
- ✅ Duplicate detection
- ✅ Template enforcement
- ✅ Rate limited

**Use case**: Issue triage, PR creation, documentation updates

**Example**: [`issue-triage-agent`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/issue-triage-agent.md) can add labels but can't close issues or modify code.

### Pattern 3: Human-in-the-Loop

For high-impact operations, require human approval before execution.

**Security Properties:**
- ✅ Human reviews PR before merge
- ✅ Explicit approval step
- ✅ Can be reverted
- ✅ Blame trail maintained

**Use case**: Code changes, dependency updates, configuration changes

**Example**: [`daily-workflow-updater`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/daily-workflow-updater.md) creates PRs for dependency updates but never merges them automatically.

### Pattern 4: Role-Gated ChatOps

Interactive agents that require authorization to invoke.

**Security Properties:**
- ✅ Platform-enforced authorization
- ✅ Clear invocation trail
- ✅ User attribution
- ✅ Can be disabled per-user

**Use case**: Code review, optimization, debugging assistance

**Example**: [`grumpy-reviewer`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/grumpy-reviewer.md) requires collaborator access to invoke via `/grumpy`.

### Pattern 5: Network Restricted

Limit network access to specific allowlisted domains.

**Security Properties:**
- ✅ Can't exfiltrate data
- ✅ Can't access internal services
- ✅ Can't download malicious payloads
- ✅ Enforced at infrastructure level

**Use case**: Workflows needing external APIs

**Example**: Workflows using Tavily search can only access `api.tavily.com`, not arbitrary websites.

## Common Security Mistakes

### Mistake 1: Overly Permissive Defaults

**Problem**: Granting `contents: write` when `contents: read` suffices.

**Impact**: Agent can accidentally push code changes.

**Solution**: Start with least privilege. Add permissions only when safe outputs require them.

### Mistake 2: Unbounded Safe Outputs

**Problem**: Forgetting `max_items` limit on safe output creation.

**Impact**: Agent creates hundreds of duplicate issues.

**Solution**: Always set `max_items`, `expire`, and `close_older` on safe outputs.

### Mistake 3: No Tool Allowlisting

**Problem**: Allowing `bash: "*"` (all bash commands).

**Impact**: Agent can run `rm -rf` or other destructive commands.

**Solution**: Explicitly list allowed commands: `bash: [git, jq, python]`.

### Mistake 4: Missing Role Gates

**Problem**: Anyone can trigger `/deploy` command.

**Impact**: Malicious actor triggers expensive or destructive operations.

**Solution**: Add author association checks for sensitive operations.

### Mistake 5: No Network Restrictions

**Problem**: Allowing open network access.

**Impact**: Agent can access internal services or exfiltrate data.

**Solution**: Use `network.allowed` to allowlist specific domains.

## Security Incidents and Response

The factory experienced a few security-adjacent incidents that taught valuable lessons:

### Incident 1: Issue Spam

**What happened**: Agent with unbounded `create_issue` safe output created 50+ duplicate issues.

**Root cause**: Missing `max_items` and `close_older` constraints.

**Fix**: Added `max_items: 3` and `close_older: true` to all issue creation safe outputs.

**Lesson**: Safe outputs need explicit bounds, not just permission gates.

### Incident 2: Expensive Workflow Loop

**What happened**: Agent triggered itself recursively, creating workflow loop.

**Root cause**: Workflow triggered on `workflow_run: completed` without filtering.

**Fix**: Added workflow name filter to prevent self-triggering.

**Lesson**: Event filters are security controls, not just optimizations.

### Incident 3: Leaked Secret Reference

**What happened**: Agent logged GitHub token in error message.

**Root cause**: Overly verbose error handling.

**Fix**: Sanitized all error messages. Added secret scanning to CI.

**Lesson**: Treat logs as public. Never log credentials.

### Incident 4: Permission Escalation Attempt

**What happened**: User tried to invoke `/q` without permissions.

**Root cause**: Role check was commented out during debugging.

**Fix**: Re-enabled role check. Added test to verify it.

**Lesson**: Security controls must be tested and visible.

## Security Checklist for New Workflows

Before deploying a new agent, verify:

- [ ] Uses least privilege permissions
- [ ] Safe outputs have `max_items`, `expire`, `close_older`
- [ ] Tools are explicitly listed (no `"*"`)
- [ ] Network access is allowlisted
- [ ] Sensitive operations are role-gated
- [ ] No secrets in prompts or logs
- [ ] Workflow can't trigger itself
- [ ] Human approval for high-impact changes
- [ ] Expiration date for experiments
- [ ] Security review completed

## Security Architecture Reference

For deeper technical details, see:
- [Security Architecture](https://githubnext.github.io/gh-aw/introduction/architecture/)
- [Security Guide](https://githubnext.github.io/gh-aw/guides/security/)
- [Safe Outputs Documentation](https://githubnext.github.io/gh-aw/reference/safe-outputs/)

## Defense in Depth

The factory's security isn't a single mechanism - it's layered:

1. **Platform**: GitHub Actions isolation, runner sandboxing
2. **Permissions**: Least privilege via GITHUB_TOKEN
3. **Safe Outputs**: Constrained API with guardrails
4. **Role Gates**: Authorization checks
5. **Network**: Allowlisted domains
6. **Tools**: Explicitly listed, no wildcards
7. **Audit**: Complete activity logs
8. **Time Limits**: Auto-expiration for experiments
9. **Code Review**: Security review before merge
10. **Monitoring**: Meta-agents watch for anomalies

If one layer fails, others still provide protection.

## What's Next?

With security fundamentals in place, we can explore how agentic workflows actually work under the hood - from natural language markdown to secure execution on GitHub Actions.

In the next article, we'll walk through the technical architecture that powers the factory.

[← Previous: Imports & Sharing](05-imports-and-sharing.md) | [Back to Index](../index.md) | [Next: How Workflows Work →](07-how-workflows-work.md)
