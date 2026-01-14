---
title: "Meet the Workflows in Peli's Agent Factory: Metrics & Analytics"
description: "A curated tour of metrics and analytics workflows that turn data into insights"
authors:
  - dsyme
  - peli
date: 2026-01-13
prev:
  link: /gh-aw/blog/2026-01-13-meet-the-workflows-quality-hygiene/
  label: "Quality & Hygiene Workflows"
next:
  link: /gh-aw/blog/2026-01-13-meet-the-workflows-security-compliance/
  label: "Security & Compliance Workflows"
---

<img src="/gh-aw/peli.png" alt="Peli de Halleux" width="200" style="float: right; margin: 0 0 20px 20px; border-radius: 8px;" />

Let's dive deeper into [Peli's Agent Factory](/gh-aw/blog/2026-01-12-welcome-to-pelis-agent-factory/)!

So far in our journey through [Peli's Agent Factory](/gh-aw/blog/2026-01-12-welcome-to-pelis-agent-factory/), we've explored workflows that handle incoming activity ([triage and summarization](/gh-aw/blog/2026-01-13-meet-the-workflows/)) and maintain codebase health ([quality and hygiene](/gh-aw/blog/2026-01-13-meet-the-workflows-quality-hygiene/)). These foundational workflows keep us organized, investigate failures, and catch drift before it becomes a problem.

But here's a question: when you're running dozens of AI agents, how do you know if they're actually working well? How do you spot performance issues, cost problems, or quality degradation? That's where metrics and analytics workflows come in - they're the agents that monitor other agents, turning raw activity data into actionable insights. This is where we got meta and built our central nervous system.

## 📊 Metrics & Analytics Workflows

Data nerds, rejoice! These agents turn raw repository activity into actual insights:

- **[Metrics Collector](https://github.com/githubnext/gh-aw/tree/532a0412680638e5e93b6e8c5ea9b8074fe6be22/.github/workflows/metrics-collector.md?plain=1)** - Tracks daily performance across the entire agent ecosystem
- **[Portfolio Analyst](https://github.com/githubnext/gh-aw/tree/532a0412680638e5e93b6e8c5ea9b8074fe6be22/.github/workflows/portfolio-analyst.md?plain=1)** - Identifies cost reduction opportunities (because AI isn't free!)
- **[Audit Workflows](https://github.com/githubnext/gh-aw/tree/532a0412680638e5e93b6e8c5ea9b8074fe6be22/.github/workflows/audit-workflows.md?plain=1)** - A meta-agent that audits all the other agents' runs - very Inception

Here's where things got meta: we built agents to monitor agents. The Metrics Collector became our central nervous system, gathering performance data that feeds into higher-level orchestrators. What we learned: **you can't optimize what you don't measure**. The Portfolio Analyst was eye-opening - it identified workflows that were costing us money unnecessarily (turns out some agents were way too chatty with their LLM calls).

These workflows taught us that observability isn't optional when you're running dozens of AI agents - it's the difference between a well-oiled machine and an expensive black box.

## Up Next: Building Trust Boundaries

Now that we can measure and optimize our agent ecosystem, we needed to ensure these powerful agents operate safely. Time to talk about security.

Continue reading: [Security & Compliance Workflows →](/gh-aw/blog/2026-01-13-meet-the-workflows-security-compliance/)

---

*This is part 3 of a 16-part series exploring the workflows in Peli's Agent Factory.*
