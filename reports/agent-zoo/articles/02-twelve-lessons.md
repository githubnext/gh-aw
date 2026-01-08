# 12 Lessons from Peli's Agent Factory

**Key insights about what works, what doesn't, and how to design effective agent ecosystems**

[← Previous: Meet the Workflows](01-meet-the-workflows.md) | [Back to Index](../index.md) | [Next: Design Patterns →](03-design-patterns.md)

---

Peli's Agent Factory was an ambitious experiment in scaling agentic workflows. After months of nurturing over 145 autonomous agents and observing their behavior in production, several key insights emerged about what works, what doesn't, and how to design effective agent ecosystems.

## The 12 Key Lessons

### ✨ Diversity Beats Perfection

No single agent can do everything. A collection of focused agents, each doing one thing well, proved more practical than trying to build a universal assistant.

Rather than spending months building the "perfect" general-purpose agent, we found it more effective to create specialized agents quickly and let them evolve based on actual usage. The zoo's diversity meant that when one pattern failed, others could succeed. This heterogeneous approach created a more resilient and adaptable system.

### 📊 Guardrails Enable Innovation

Counter-intuitively, strict constraints (safe outputs, limited permissions, allowlisted tools) made it *easier* to experiment. We knew the blast radius of any failure.

With clear boundaries in place, we could rapidly prototype new agents without fear of breaking production systems. Safe outputs prevented agents from accidentally deleting code or closing important issues. Network allowlists ensured agents couldn't leak data to unauthorized services. These guardrails didn't slow us down - they gave us confidence to move faster.

### 🔄 Meta-Agents Are Essential

Agents that monitor agents became some of the most valuable. They caught issues early and helped us understand aggregate behavior.

As the factory grew past 50 workflows, it became impossible for humans to track everything. Meta-agents like the [Audit Workflows](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/audit-workflows.md) and [Agent Performance Analyzer](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/agent-performance-analyzer.md) provided the observability layer we needed. They detected patterns across runs, identified struggling agents, and surfaced systemic issues that would have been invisible looking at individual workflows.

### 🎭 Personality Matters

Agents with clear "personalities" (the meticulous auditor, the helpful janitor, the creative poet) were easier for teams to understand and trust.

We noticed that generic agent names like "issue-handler" or "code-checker" created confusion about what the agent actually did. But giving agents distinct personalities - like the [Grumpy Reviewer](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/grumpy-reviewer.md) or [Poem Bot](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/poem-bot.md) - made their purpose immediately clear and set expectations about tone and behavior. Team members developed relationships with specific agents.

### ⚖️ Cost-Quality Tradeoffs Are Real

Longer, more thorough analyses cost more but aren't always better. The [Portfolio Analyst](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/portfolio-analyst.md) helped us identify which agents gave the best value.

We discovered that some "thorough" agents were doing redundant work or producing reports nobody read. The Portfolio Analyst tracked cost-per-insight across all agents, revealing that simple, focused agents often delivered better ROI than complex ones. This led us to consolidate overlapping agents and tune prompt lengths to balance thoroughness with cost.

### 🔄 Multi-Phase Workflows Enable Ambitious Goals

Breaking complex improvements into 3-phase workflows (research → setup → implement) allowed agents to tackle projects that would be too large for a single run. Each phase builds on the last, with human feedback between phases.

Single-run agents are limited by token context and execution time. Multi-phase workflows like [Daily Test Improver](https://github.com/githubnext/agentics/blob/main/workflows/daily-test-improver.md) and [Daily Perf Improver](https://github.com/githubnext/agentics/blob/main/workflows/daily-perf-improver.md) could tackle ambitious projects by spreading work across multiple days. The research phase explored the problem space, the setup phase prepared infrastructure, and the implementation phase executed changes. Human checkpoints between phases ensured alignment with team goals.

### 💬 Slash Commands Create Natural User Interfaces

ChatOps-style `/command` triggers made agents feel like natural team members. Users could invoke powerful capabilities with simple comments, and role-gating ensured only authorized users could trigger sensitive operations.

Instead of remembering complex webhook URLs or GitHub Actions syntax, team members could simply comment `/grumpy` on a PR to request a critical review, or `/pr-fix` to fix failing tests. Role-gating prevented abuse while keeping the interface simple. This pattern proved so successful that most of our interactive agents adopted it.

### 🧪 Heartbeats Build Confidence

Frequent, lightweight validation tests (every 12 hours) caught regressions quickly. These "heartbeat" agents ensured the infrastructure stayed healthy without manual monitoring.

Rather than waiting for production failures, we deployed multiple [smoke test workflows](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/smoke-copilot.md) that continuously validated core functionality. When a smoke test failed, we knew immediately which component broke. This proactive monitoring prevented cascading failures and built confidence that the agent ecosystem was stable.

### 🔧 MCP Inspection Is Essential

As workflows grew to use multiple MCP servers, having agents that could validate and report on tool availability became critical. The [MCP Inspector](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/mcp-inspector.md) pattern prevented cryptic failures from misconfigured tools.

Early in the factory's development, we'd frequently see agents fail with vague errors like "tool not available" or "connection refused." The MCP Inspector agent proactively checked all MCP server configurations, validated network access, and generated status reports. This visibility transformed debugging from hours of detective work to reading a status dashboard.

### 🎯 Dispatcher Patterns Scale Command Complexity

Instead of one monolithic agent handling all requests, dispatcher agents could route to specialized sub-agents or commands. This made the system more maintainable and allowed for progressive feature addition.

The [Workflow Generator](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/workflow-generator.md) and [Campaign Generator](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/campaign-generator.md) demonstrated this pattern well. Rather than cramming all generation logic into one massive prompt, they identified user intent and dispatched to specialized generation workflows. This kept individual agents focused and made the system easier to extend.

### 📿 Task Queuing Is Everywhere

The task queue pattern provided a simple way to queue and distribute work across multiple workflow runs. Breaking large projects into discrete tasks allowed incremental progress with clear state tracking, recording tasks as issues, discussions, or project cards.

Whether managing a backlog of refactoring work, coordinating security fixes, or distributing test creation tasks, the task queue pattern appeared repeatedly. By representing work as GitHub primitives (issues, project cards), we got built-in state management, persistence, and audit trails without building custom infrastructure.

### 🤖 ML Analysis Reveals Hidden Patterns

Applying clustering and NLP to agent interactions revealed usage patterns that weren't obvious from individual runs. This meta-analysis helped identify opportunities for consolidation and optimization.

The [Prompt Clustering Analysis](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/prompt-clustering-analysis.md) and [Copilot PR NLP Analysis](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/copilot-pr-nlp-analysis.md) workflows discovered that many agents were asking similar questions or performing redundant analyses. This insight led to shared component libraries and consolidation opportunities we wouldn't have spotted through manual review.

## Challenges We Encountered

Not everything was smooth sailing. We faced several challenges that provided valuable lessons:

### Permission Creep

As agents gained capabilities, there was a temptation to grant broader permissions. We had to constantly audit and prune permissions to maintain least privilege.

The principle of least privilege requires ongoing vigilance. We established a quarterly permission audit process where we reviewed every agent's permissions against its actual behavior. This often revealed agents that had been granted write access but only needed read permissions, or agents requesting GitHub API scopes they never used.

### Debugging Complexity

When agents misbehaved, tracing the root cause through multiple workflow runs and safe outputs was challenging. Improved logging and observability are needed.

Distributed debugging across multiple agents, each generating their own logs and artifacts, proved surprisingly difficult. We improved this with structured logging, correlation IDs across related runs, and meta-agents that aggregated failure patterns. But there's still room for better tooling in this space.

### Repository Noise

Frequent agent runs created a lot of issues, PRs, and comments. We had to implement archival strategies to keep the repository manageable.

With agents creating dozens of issues and PRs daily, the repository's signal-to-noise ratio suffered. We developed cleanup agents that archived old discussions, closed stale issues, and consolidated redundant reports. Finding the right balance between transparency and clutter remains an ongoing challenge.

### Cost Management

Running many agents incurred significant costs. The Portfolio Analyst helped, but ongoing cost monitoring is essential.

AI agent operations at scale aren't free. We had to develop cost awareness into the factory's culture, with regular reviews of spend-per-agent and value-per-dollar metrics. Some expensive but low-value agents were deprecated, while high-value agents got budget increases. Cost visibility turned out to be as important as functionality.

### User Trust

Some team members were hesitant to engage with automated agents. Clear communication about capabilities and limitations helped build trust over time.

Trust isn't automatic - it's earned through consistent behavior and transparent communication. We found that agents with clear "about me" descriptions, visible limitations, and predictable behavior patterns gained acceptance faster. Failed experiments that were openly discussed as learning opportunities also helped build trust.

## Applying These Lessons

These lessons aren't just academic observations - they're practical insights you can apply when building your own agent ecosystem:

1. **Start diverse, not perfect** - Launch multiple simple agents rather than one complex one
2. **Design with guardrails first** - Constraints enable safe experimentation
3. **Build meta-agents early** - You'll need them sooner than you think
4. **Give agents personality** - It aids understanding and adoption
5. **Monitor costs from day one** - Cost awareness prevents surprises
6. **Embrace multi-phase patterns** - Break ambitious projects into manageable phases
7. **Use ChatOps interfaces** - Slash commands are intuitive and role-gatable
8. **Implement heartbeats** - Proactive monitoring beats reactive debugging
9. **Inspect your tools** - Validate tool availability before agents need them
10. **Dispatch, don't monolith** - Route requests to specialized agents
11. **Queue your work** - Task queuing enables incremental progress
12. **Analyze at meta-level** - ML can reveal patterns humans miss

## What's Next?

These lessons emerged from observing agent behavior, but understanding *how* agents behave requires understanding their fundamental design patterns.

In the next article, we'll explore the 12 core design patterns that emerged from analyzing all 145 workflows in the factory.

[← Previous: Meet the Workflows](01-meet-the-workflows.md) | [Back to Index](../index.md) | [Next: Design Patterns →](03-design-patterns.md)
