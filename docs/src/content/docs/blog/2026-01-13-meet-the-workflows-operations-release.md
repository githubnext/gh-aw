---
title: "Meet the Workflows: Operations & Release"
description: "A curated tour of operations and release workflows that ship software"
authors:
  - dsyme
  - peli
  - mnkiefer
date: 2026-01-13T07:00:00
sidebar:
  label: "Operations & Release"
prev:
  link: /gh-aw/blog/2026-01-13-meet-the-workflows-metrics-analytics/
  label: "Metrics & Analytics Workflows"
next:
  link: /gh-aw/blog/2026-01-13-meet-the-workflows-security-compliance/
  label: "Security & Compliance Workflows"
---

<img src="/gh-aw/peli.png" alt="Peli de Halleux" width="200" style="float: right; margin: 0 0 20px 20px; border-radius: 8px;" />

Ah! Right this way to *another extraordinary chamber* in [Peli's Agent Factory](/gh-aw/blog/2026-01-12-welcome-to-pelis-agent-factory/)!

In our [previous post](/gh-aw/blog/2026-01-13-meet-the-workflows-metrics-analytics/), we explored metrics and analytics workflows - the agents that monitor other agents, turning raw activity data into actionable insights. We built our central nervous system for the agent ecosystem, learning that you can't optimize what you don't measure.

Now comes the moment of truth: actually shipping software to users. All the quality checks, metrics tracking, and iterative improvements culminate in one critical process - the release. Operations and release workflows handle the orchestration of building, testing, generating release notes, and publishing. These workflows can't afford to be experimental; they need to be rock-solid reliable, well-tested, and yes, even a bit boring. Let's explore how automation makes shipping predictable and stress-free.

## Operations & Release Workflows

The agents that help us actually ship software:

- **[Release](https://github.com/githubnext/gh-aw/tree/bb7946527af340043f1ebb31fc21bd491dd0f42d/.github/workflows/release.md?plain=1)** - Orchestrates builds, tests, and release note generation
- **[Daily Workflow Updater](https://github.com/githubnext/gh-aw/tree/bb7946527af340043f1ebb31fc21bd491dd0f42d/.github/workflows/daily-workflow-updater.md?plain=1)** - Keeps actions and dependencies current (because dependency updates never stop)  

Shipping software is stressful enough without worrying about whether you formatted your release notes correctly. The Release workflow handles the entire orchestration - building, testing, generating coherent release notes from commits, and publishing. What's interesting here is the **reliability** requirement: these workflows can't afford to be creative or experimental. They need to be deterministic, well-tested, and boring (in a good way).

The Daily Workflow Updater taught us that maintenance is a perfect use case for agents - it's repetitive, necessary, and nobody enjoys doing it manually. These workflows handle the toil so we can focus on the interesting problems.

## Learn More

- **[GitHub Agentic Workflows](https://githubnext.github.io/gh-aw/)** - The technology behind the workflows
- **[Quick Start](https://githubnext.github.io/gh-aw/setup/quick-start/)** - How to write and compile workflows

## Next Up: Security & Compliance Workflows

After all this focus on shipping, we need to talk about the guardrails: how do we ensure these powerful agents operate safely?

Continue reading: [Security & Compliance Workflows →](/gh-aw/blog/2026-01-13-meet-the-workflows-security-compliance/)

---

*This is part 7 of a 16-part series exploring the workflows in Peli's Agent Factory.*
