---
title: "Meet the Workflows: Testing & Validation"
description: "A curated tour of testing workflows that keep everything running smoothly"
authors:
  - dsyme
  - peli
date: 2026-01-13T11:00:00
prev:
  link: /gh-aw/blog/2026-01-13-meet-the-workflows-interactive-chatops/
  label: "Interactive & ChatOps Workflows"
next:
  link: /gh-aw/blog/2026-01-13-meet-the-workflows-tool-infrastructure/
  label: "Tool & Infrastructure Workflows"
---

<img src="/gh-aw/peli.png" alt="Peli de Halleux" width="200" style="float: right; margin: 0 0 20px 20px; border-radius: 8px;" />

Let's continue our tour of [Peli's Agent Factory](/gh-aw/blog/2026-01-12-welcome-to-pelis-agent-factory/)!

In our [previous post](/gh-aw/blog/2026-01-13-meet-the-workflows-interactive-chatops/), we explored interactive and ChatOps workflows - agents that respond to slash commands and GitHub reactions, providing on-demand assistance with full context. We learned that context is king: the right agent at the right moment is far more valuable than scheduled runs.

But making code *better* is only half the battle. We also need to ensure it keeps *working*. As we refactor, optimize, and evolve our codebase, how do we know we haven't broken something? How do we catch regressions before users do? That's where testing and validation workflows come in - the skeptical guardians that continuously verify our systems still function as expected. We learned the hard way that AI infrastructure needs constant health checks, because what worked yesterday might silently fail today. These workflows embody **trust but verify**.

## Testing & Validation Workflows

These agents keep everything running smoothly through continuous testing:

- **[Smoke Tests](https://github.com/githubnext/gh-aw/tree/532a0412680638e5e93b6e8c5ea9b8074fe6be22/.github/workflows/smoke-codex-firewall.md?plain=1)** - Validate that engines and firewall are working (running every 12 hours!)
- **[Daily Multi-Device Docs Tester](https://github.com/githubnext/gh-aw/tree/532a0412680638e5e93b6e8c5ea9b8074fe6be22/.github/workflows/daily-multi-device-docs-tester.md?plain=1)** - Tests documentation across devices (mobile matters!)
- **[CI Coach](https://github.com/githubnext/gh-aw/tree/532a0412680638e5e93b6e8c5ea9b8074fe6be22/.github/workflows/ci-coach.md?plain=1)** - Provides friendly guidance on CI/CD improvements

We learned the hard way that AI infrastructure needs constant health checks. The Smoke Tests run every 12 hours to validate that our core systems (engines, firewall, MCP servers) are actually working. It's caught outages before users noticed them. The Multi-Device Docs Tester uses Playwright to test our documentation on different screen sizes - it found mobile rendering issues we never would have caught manually. The CI Coach analyzes our CI/CD pipeline and suggests optimizations ("you're running tests sequentially when they could be parallel").

These workflows embody the principle: **trust but verify**. Just because it worked yesterday doesn't mean it works today.

## Learn More

- **[GitHub Agentic Workflows](https://githubnext.github.io/gh-aw/)** - The technology behind the workflows
- **[Quick Start](https://githubnext.github.io/gh-aw/setup/quick-start/)** - How to write and compile workflows

## Next Up: Monitoring the Monitors

But what about the infrastructure itself? Who watches the watchers? Time to go meta.

Continue reading: [Tool & Infrastructure Workflows →](/gh-aw/blog/2026-01-13-meet-the-workflows-tool-infrastructure/)

---

*This is part 11 of a 16-part series exploring the workflows in Peli's Agent Factory.*
