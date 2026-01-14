---
title: "Meet the Workflows in Peli's Agent Factory: Multi-Phase Improvers"
description: "A curated tour of multi-phase workflows that tackle long-running projects"
authors:
  - dsyme
  - peli
date: 2026-01-13
prev:
  link: /gh-aw/blog/2026-01-13-meet-the-workflows-tool-infrastructure/
  label: "Tool & Infrastructure Workflows"
next:
  link: /gh-aw/blog/2026-01-13-meet-the-workflows-advanced-analytics/
  label: "Advanced Analytics & ML Workflows"
---

<img src="/gh-aw/peli.png" alt="Peli de Halleux" width="200" style="float: right; margin: 0 0 20px 20px; border-radius: 8px;" />

Let's continue our journey through [Peli's Agent Factory](/gh-aw/blog/2026-01-12-welcome-to-pelis-agent-factory/)!

In our [previous posts](/gh-aw/blog/2026-01-13-meet-the-workflows-tool-infrastructure/), we explored the infrastructure layer - workflows that monitor MCP servers, validate tool configurations, and ensure the platform itself stays healthy. These meta-workflows provide visibility into the invisible plumbing that makes AI agents work.

Most workflows we've seen so far run once and complete: analyze this PR, triage that issue, test this deployment. They're ephemeral - they execute, produce results, and disappear. But what about projects that are too big to tackle in a single run? What about initiatives that require research, setup, and incremental implementation? Traditional CI/CD is built for stateless execution, but we discovered something powerful: workflows that maintain state across days, working a little bit each day like a persistent team member who never takes breaks. Welcome to our most ambitious experiment - multi-phase improvers that prove AI agents can handle complex, long-running projects.

## 🚀 Multi-Phase Improver Workflows

These are some of our most ambitious agents - they tackle big projects over multiple days:

- **[Daily Backlog Burner](https://github.com/githubnext/agentics/blob/main/workflows/daily-backlog-burner.md?plain=1)** - Systematically works through issues and PRs, one day at a time
- **[Daily Perf Improver](https://github.com/githubnext/agentics/blob/main/workflows/daily-perf-improver.md?plain=1)** - Three-phase performance optimization (research, setup, implement)
- **[Daily Test Improver](https://github.com/githubnext/agentics/blob/main/workflows/daily-test-improver.md?plain=1)** - Identifies coverage gaps and implements new tests incrementally
- **[Daily QA](https://github.com/githubnext/agentics/blob/main/workflows/daily-qa.md?plain=1)** - Continuous quality assurance that never sleeps
- **[Daily Accessibility Review](https://github.com/githubnext/agentics/blob/main/workflows/daily-accessibility-review.md?plain=1)** - WCAG compliance checking with Playwright
- **[PR Fix](https://github.com/githubnext/agentics/blob/main/workflows/pr-fix.md?plain=1)** - On-demand slash command to fix failing CI checks (super handy!)

This is where we got experimental with agent persistence and multi-day workflows. Traditional CI runs are ephemeral, but these workflows maintain state across days using repo-memory. The Daily Perf Improver runs in three phases - research (find bottlenecks), setup (create profiling infrastructure), implement (optimize). It's like having a performance engineer who works a little bit each day. The Daily Backlog Burner systematically tackles our issue backlog - one issue per day, methodically working through technical debt. We learned that **incremental progress beats heroic sprints** - these agents never get tired, never get distracted, and never need coffee breaks. The PR Fix workflow is our emergency responder - when CI fails, invoke `/pr-fix` and it investigates and attempts repairs.

These workflows prove that AI agents can handle complex, long-running projects when given the right architecture.

## Learn More

- **[GitHub Agentic Workflows](https://githubnext.github.io/gh-aw/)** - The technology behind the workflows
- **[Quick Start](https://githubnext.github.io/gh-aw/setup/quick-start/)** - How to write and compile workflows

## Next Up: Analytics & ML Workflows

Basic metrics are useful, but we want to go deeper - to truly understand agent behavior patterns and developer interactions.

Continue reading: [Advanced Analytics & ML Workflows →](/gh-aw/blog/2026-01-13-meet-the-workflows-advanced-analytics/)

---

*This is part 11 of a 16-part series exploring the workflows in Peli's Agent Factory.*
