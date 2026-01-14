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

Let's dive deeper into Peli's Agent Factory!

So far in our journey through Peli's Agent Factory, we've explored workflows that handle incoming activity ([triage and summarization](/gh-aw/blog/2026-01-13-meet-the-workflows/)) and maintain codebase health ([quality and hygiene](/gh-aw/blog/2026-01-13-meet-the-workflows-quality-hygiene/)). These foundational workflows keep us organized, investigate failures, and catch drift before it becomes a problem.

But here's a question: when you're running dozens of AI agents, how do you know if they're actually working well? How do you spot performance issues, cost problems, or quality degradation? That's where metrics and analytics workflows come in - they're the agents that monitor other agents, turning raw activity data into actionable insights. This is where we got meta and built our central nervous system.

## 📊 Metrics & Analytics Workflows

Data nerds, rejoice! These agents turn raw repository activity into actual insights:

- **[Metrics Collector](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/metrics-collector.md)** - Tracks daily performance across the entire agent ecosystem
- **[Portfolio Analyst](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/portfolio-analyst.md)** - Identifies cost reduction opportunities (because AI isn't free!)
- **[Audit Workflows](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/audit-workflows.md)** - A meta-agent that audits all the other agents' runs - very Inception

Here's where things got meta: we built agents to monitor agents. The Metrics Collector became our central nervous system, gathering performance data that feeds into higher-level orchestrators. What we learned: **you can't optimize what you don't measure**. The Portfolio Analyst was eye-opening - it identified workflows that were costing us money unnecessarily (turns out some agents were way too chatty with their LLM calls).

These workflows taught us that observability isn't optional when you're running dozens of AI agents - it's the difference between a well-oiled machine and an expensive black box.

## In the Next Stage of Our Journey...

Now that we've explored how metrics and analytics workflows help us optimize our agent ecosystem, let's examine the agents that keep watch over security and compliance, ensuring our AI agents operate within safe boundaries.

Continue reading: [Security & Compliance Workflows →](/gh-aw/blog/2026-01-13-meet-the-workflows-security-compliance/)

---

*This is part 3 of a 16-part series exploring the workflows in Peli's Agent Factory.*
