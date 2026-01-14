---
title: "Meet the Workflows in Peli's Agent Factory: Tool & Infrastructure"
description: "A curated tour of infrastructure workflows that monitor the agentic systems"
authors:
  - dsyme
  - peli
date: 2026-01-13
prev:
  link: /gh-aw/blog/2026-01-13-meet-the-workflows-testing-validation/
  label: "Testing & Validation Workflows"
next:
  link: /gh-aw/blog/2026-01-13-meet-the-workflows-multi-phase/
  label: "Multi-Phase Improver Workflows"
---

<img src="/gh-aw/peli.png" alt="Peli de Halleux" width="200" style="float: right; margin: 0 0 20px 20px; border-radius: 8px;" />

Welcome back to our journey through [Peli's Agent Factory](/gh-aw/blog/2026-01-12-welcome-to-pelis-agent-factory/)!

We've covered workflows that improve code quality and [validate that everything keeps working](/gh-aw/blog/2026-01-13-meet-the-workflows-testing-validation/). Testing and validation workflows run continuously, checking that our systems function correctly and catching regressions before they reach users.

But here's a question that kept us up at night: what if the *infrastructure itself* fails? What if MCP servers are misconfigured, tools become unavailable, or agents can't access the capabilities they need? Testing the *application* is one thing; monitoring the *platform* that runs AI agents is another beast entirely. Tool and infrastructure workflows provide meta-monitoring - they watch the watchers, validate configurations, and ensure the invisible plumbing stays functional. Welcome to the layer where we monitor agents monitoring agents monitoring code. Yes, it gets very meta.

## 🧰 Tool & Infrastructure Workflows

These agents monitor and analyze the agentic infrastructure itself:

- **[MCP Inspector](https://github.com/githubnext/gh-aw/tree/532a0412680638e5e93b6e8c5ea9b8074fe6be22/.github/workflows/mcp-inspector.md?plain=1)** - Validates Model Context Protocol configurations
- **[GitHub MCP Tools Report](https://github.com/githubnext/gh-aw/tree/532a0412680638e5e93b6e8c5ea9b8074fe6be22/.github/workflows/github-mcp-tools-report.md?plain=1)** - Analyzes available MCP tools
- **[Agent Performance Analyzer](https://github.com/githubnext/gh-aw/tree/532a0412680638e5e93b6e8c5ea9b8074fe6be22/.github/workflows/agent-performance-analyzer.md?plain=1)** - Meta-orchestrator for agent quality

Infrastructure for AI agents is different from traditional infrastructure - you need to validate that tools are available, properly configured, and actually working. The MCP Inspector checks Model Context Protocol server configurations because a misconfigured MCP server means an agent can't access the tools it needs. The Agent Performance Analyzer is a meta-orchestrator that monitors all our other agents - looking for performance degradation, cost spikes, and quality issues. We learned that **layered observability** is crucial: you need monitoring at the infrastructure level (are servers up?), the tool level (can agents access what they need?), and the agent level (are they performing well?).

These workflows provide visibility into the invisible.

## Breaking the Ephemeral Mold

Most workflows we've seen are stateless - they run, complete, and disappear. But what if agents could maintain memory across days?

Continue reading: [Multi-Phase Improver Workflows →](/gh-aw/blog/2026-01-13-meet-the-workflows-multi-phase/)

---

*This is part 10 of a 16-part series exploring the workflows in Peli's Agent Factory.*
