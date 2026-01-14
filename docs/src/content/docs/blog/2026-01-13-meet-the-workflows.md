---
title: "Meet the Workflows in Peli's Agent Factory: Triage & Summarization"
description: "A curated tour of triage and summarization workflows in the factory"
authors:
  - dsyme
  - peli
date: 2026-01-13
prev:
  link: /gh-aw/blog/2026-01-12-welcome-to-pelis-agent-factory/
  label: Welcome to Peli's Agent Factory
next:
  link: /gh-aw/blog/2026-01-13-meet-the-workflows-quality-hygiene/
  label: "Quality & Hygiene Workflows"
---

<img src="/gh-aw/peli.png" alt="Peli de Halleux" width="200" style="float: right; margin: 0 0 20px 20px; border-radius: 8px;" />

Hi there! 👋 Welcome back to Peli's Agent Factory.

We're the GitHub Next team, and we've been on quite a journey. Over the past months, we've built and operated a collection of automated agentic workflows. These aren't just demos or proof-of-concepts - these are real agents doing actual work in our [`githubnext/gh-aw`](https://github.com/githubnext/gh-aw) repository and its companion [`githubnext/agentics`](https://github.com/githubnext/agentics) collection.

Think of this as your guided tour through our agent "factory". We're showcasing the workflows that caught our attention, taught us something new, or just flat-out made our lives easier. Every workflow links to its source Markdown file, so you can peek under the hood and see exactly how it works.

## 🏥 Triage & Summarization Workflows

First up: the agents that help us stay sane when things get busy. These workflows keep us on top of the constant flow of activity:

- **[Issue Triage Agent](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/issue-triage-agent.md)** - Automatically labels and categorizes new issues the moment they're opened
- **[Weekly Issue Summary](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/weekly-issue-summary.md)** - Creates digestible summaries complete with charts and trends (because who has time to read everything?)
- **[Daily Repo Chronicle](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/daily-repo-chronicle.md)** - Narrates the day's activity like a storyteller - seriously, it's kind of delightful

What surprised us most about this category? The **tone** matters way more than we expected. When the Daily Repo Chronicle started writing summaries in a narrative, almost journalistic style, people actually *wanted* to read them. We discovered that AI agents don't have to be robotic - they can have personality while still being informative. The Issue Triage Agent taught us that even simple automation (just adding labels!) dramatically reduces cognitive load when you're scanning through dozens of issues.

These workflows became our daily reading habit rather than another notification to dismiss.

## In the Next Stage of Our Journey...

Now that we've explored how triage and summarization workflows help us stay on top of incoming activity, let's look next at the agents that maintain quality and hygiene in our repository. These diligent caretakers spot problems before they escalate and keep our codebase healthy.

Continue reading: [Quality & Hygiene Workflows →](/gh-aw/blog/2026-01-13-meet-the-workflows-quality-hygiene/)

---

*This is part 1 of a 16-part series exploring the workflows in Peli's Agent Factory.*
