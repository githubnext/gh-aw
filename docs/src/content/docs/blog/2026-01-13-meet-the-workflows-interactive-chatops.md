---
title: "Meet the Workflows in Peli's Agent Factory: Interactive & ChatOps"
description: "A curated tour of interactive workflows that respond to commands"
authors:
  - dsyme
  - peli
date: 2026-01-13
prev:
  link: /gh-aw/blog/2026-01-13-meet-the-workflows-creative-culture/
  label: "Creative & Culture Workflows"
next:
  link: /gh-aw/blog/2026-01-13-meet-the-workflows-code-quality/
  label: "Code Quality & Refactoring Workflows"
---

<img src="/gh-aw/peli.png" alt="Peli de Halleux" width="200" style="float: right; margin: 0 0 20px 20px; border-radius: 8px;" />

Let's keep exploring [Peli's Agent Factory](/gh-aw/blog/2026-01-12-welcome-to-pelis-agent-factory/)!

So far, we've explored workflows that run automatically on schedules or triggers - from triage and quality checks to security enforcement and [creative culture](/gh-aw/blog/2026-01-13-meet-the-workflows-creative-culture/) builders. These scheduled and event-driven workflows handle the continuous, ambient work that keeps our repositories healthy and our teams engaged.

But sometimes you need help *right now*, at the exact moment you're stuck on a problem. You don't want to wait for a scheduled run - you want to summon an expert agent with a command. That's where interactive workflows and ChatOps come in. These agents respond to slash commands and GitHub reactions, providing on-demand assistance with full context of the current situation. We learned that **context is king** - the right agent at the right moment with the right information is far more valuable than a dozen agents running on cron schedules.

## 💬 Interactive & ChatOps Workflows

These agents respond to commands, providing on-demand assistance whenever you need it:

- **[Q](https://github.com/githubnext/gh-aw/tree/532a0412680638e5e93b6e8c5ea9b8074fe6be22/.github/workflows/q.md?plain=1)** - Workflow optimizer that investigates performance and creates PRs  
  [→ View optimization suggestions](https://github.com/search?q=repo%3Agithubnext%2Fgh-aw+in%3Atitle+%22%5Bq%5D%22+label%3Aworkflow-optimization+is%3Aissue&type=issues)
- **[Grumpy Reviewer](https://github.com/githubnext/gh-aw/tree/532a0412680638e5e93b6e8c5ea9b8074fe6be22/.github/workflows/grumpy-reviewer.md?plain=1)** - Performs critical code reviews with, well, personality  
  [→ View grumpy reviews](https://github.com/search?q=repo%3Agithubnext%2Fgh-aw+author%3Aapp%2Fgithub-actions+grumpy+review+is%3Aissue-comment&type=issues)
- **[Workflow Generator](https://github.com/githubnext/gh-aw/tree/532a0412680638e5e93b6e8c5ea9b8074fe6be22/.github/workflows/workflow-generator.md?plain=1)** - Creates new workflows from issue requests  
  [→ View generated workflows](https://github.com/search?q=repo%3Agithubnext%2Fgh-aw+author%3Aapp%2Fgithub-actions+generated+workflow+is%3Apr&type=pullrequests)

Interactive workflows changed how we think about agent invocation. Instead of everything running on a schedule, these respond to slash commands and reactions - `/q` summons the workflow optimizer, a 🚀 reaction triggers analysis. Q (yes, named after the James Bond quartermaster) became our go-to troubleshooter - it investigates workflow performance issues and opens PRs with optimizations.

The Grumpy Reviewer gave us surprisingly valuable feedback with a side of sass ("This function is so nested it has its own ZIP code"). We learned that **context is king** - these agents work because they're invoked at the right moment with the right context, not because they run on a schedule.

## Learn More

- **[GitHub Agentic Workflows](https://githubnext.github.io/gh-aw/)** - The technology behind the workflows
- **[Quick Start](https://githubnext.github.io/gh-aw/setup/quick-start/)** - How to write and compile workflows

## Next Up: Automated Code Quality & Refactoring

While ChatOps agents respond to commands, another category works silently in the background, continuously improving code quality.

Continue reading: [Code Quality & Refactoring Workflows →](/gh-aw/blog/2026-01-13-meet-the-workflows-code-quality/)

---

*This is part 7 of a 16-part series exploring the workflows in Peli's Agent Factory.*
