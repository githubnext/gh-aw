---
title: "Meet the Workflows: Quality & Hygiene"
description: "A curated tour of quality and hygiene workflows that maintain codebase health"
authors:
  - dsyme
  - peli
  - mnkiefer
date: 2026-01-13T05:00:00
sidebar:
  label: "Quality & Hygiene"
prev:
  link: /gh-aw/blog/2026-01-13-meet-the-workflows-issue-management/
  label: "Issue & PR Management Workflows"
next:
  link: /gh-aw/blog/2026-01-13-meet-the-workflows-metrics-analytics/
  label: "Metrics & Analytics Workflows"
---

<img src="/gh-aw/peli.png" alt="Peli de Halleux" width="200" style="float: right; margin: 0 0 20px 20px; border-radius: 8px;" />

*Ah, splendid!* Welcome back to [Peli's Agent Factory](/gh-aw/blog/2026-01-12-welcome-to-pelis-agent-factory/)! Come, let me show you the chamber where everything is polished, everything improved, and all sparkles!

In our [previous post](/gh-aw/blog/2026-01-13-meet-the-workflows-issue-management/), we explored issue and PR management workflows - agents that enhance GitHub's collaboration features by removing tedious ceremony like linking related issues, merging main branches, and optimizing templates. These workflows make GitHub more pleasant to use by eliminating small papercuts that add up to significant friction.

Now let's shift from collaboration ceremony to codebase maintenance. While issue workflows help us handle what comes in, quality and hygiene workflows act as vigilant caretakers - spotting problems before they escalate and keeping our codebase healthy. These are the agents that investigate failed CI runs, detect schema drift, and catch breaking changes before users do.

## Quality & Hygiene Workflows

These are our diligent caretakers - the agents that spot problems before they become, well, bigger problems:

- **[CI Doctor](https://github.com/githubnext/gh-aw/tree/bb7946527af340043f1ebb31fc21bd491dd0f42d/.github/workflows/ci-doctor.md?plain=1)** - Investigates failed workflows and opens diagnostic issues (it's like having a DevOps specialist on call 24/7)  
- **[Schema Consistency Checker](https://github.com/githubnext/gh-aw/tree/bb7946527af340043f1ebb31fc21bd491dd0f42d/.github/workflows/schema-consistency-checker.md?plain=1)** - Detects when schemas, code, and docs drift apart  
- **[Breaking Change Checker](https://github.com/githubnext/gh-aw/tree/bb7946527af340043f1ebb31fc21bd491dd0f42d/.github/workflows/breaking-change-checker.md?plain=1)** - Watches for changes that might break things for users  

The CI Doctor was a revelation. Instead of drowning in CI failure notifications, we now get *timely*, *investigated* failures with actual diagnostic insights. The agent doesn't just tell us something broke - it analyzes logs, identifies patterns, searches for similar past issues, and even suggests fixes. We learned that agents excel at the tedious investigation work that humans find draining.

The Schema Consistency Checker caught drift that would have taken us days to notice manually. 

These "hygiene" workflows became our first line of defense, catching issues before they reached users.

## Learn More

- **[GitHub Agentic Workflows](https://githubnext.github.io/gh-aw/)** - The technology behind the workflows
- **[Quick Start](https://githubnext.github.io/gh-aw/setup/quick-start/)** - How to write and compile workflows

## Next Up: Metrics & Analytics Workflows

With quality and hygiene workflows maintaining our codebase health, we needed a way to understand whether they were actually working. How do you know if your agents are performing well? That's where metrics and analytics come in.

Continue reading: [Metrics & Analytics Workflows →](/gh-aw/blog/2026-01-13-meet-the-workflows-metrics-analytics/)

---

*This is part 5 of a 16-part series exploring the workflows in Peli's Agent Factory.*
