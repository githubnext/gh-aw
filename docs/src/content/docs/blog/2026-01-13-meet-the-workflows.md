---
title: "Meet the Workflows: Triage & Summarization"
description: "A curated tour of triage and summarization workflows in the factory"
authors:
  - dsyme
  - peli
  - mnkiefer
date: 2026-01-13T01:00:00
prev:
  link: /gh-aw/blog/2026-01-12-welcome-to-pelis-agent-factory/
  label: Welcome to Peli's Agent Factory
next:
  link: /gh-aw/blog/2026-01-13-meet-the-workflows-code-quality/
  label: "Code Quality & Refactoring Workflows"
---

<img src="/gh-aw/peli.png" alt="Peli de Halleux" width="200" style="float: right; margin: 0 0 20px 20px; border-radius: 8px;" />

Hi there! 👋 Welcome back to [Peli's Agent Factory](/gh-aw/blog/2026-01-12-welcome-to-pelis-agent-factory/).

We're the GitHub Next team, and we've been on quite a journey. Over the past months, we've built and operated a collection of automated agentic workflows. These aren't just demos or proof-of-concepts - these are real agents doing actual work in our [`githubnext/gh-aw`](https://github.com/githubnext/gh-aw) repository and its companion [`githubnext/agentics`](https://github.com/githubnext/agentics) collection.

Think of this as your guided tour through our agent "factory". We're showcasing the workflows that caught our attention, taught us something new, or just flat-out made our lives easier. Every workflow links to its source Markdown file, so you can peek under the hood and see exactly how it works.

## Starting Simple: Issue Triage

To start the tour, let's begin with one of the simple workflows that **handles incoming activity** - issue triage. This represents the "hello world" of automated agentic workflows: practical, immediately useful, relatively simple, and surprisingly impactful.

Our **[Issue Triage Agent](https://github.com/githubnext/gh-aw/tree/532a0412680638e5e93b6e8c5ea9b8074fe6be22/.github/workflows/issue-triage-agent.md?plain=1)** automatically labels and categorizes new issues the moment they're opened. Let's take a look at the full workflow:

```markdown
---
timeout-minutes: 5

on:
  schedule: "0 14 * * 1-5"
  workflow_dispatch:

permissions:
  issues: read

tools:
  github:
    toolsets: [issues, labels]

safe-outputs:
  add-labels:
    allowed: [bug, feature, enhancement, documentation, question, help-wanted, good-first-issue]
  add-comment: {}
---

# Issue Triage Agent

List open issues in ${{ github.repository }} that have no labels. For each unlabeled issue, analyze the title and body, then add one of the allowed labels: `bug`, `feature`, `enhancement`, `documentation`, `question`, `help-wanted`, or `good-first-issue`. 

Skip issues that:
- Already have any of these labels
- Have been assigned to any user (especially non-bot users)

After adding the label to an issue, mention the issue author in a comment explaining why the label was added.
```

Note how concise and readable this is - it's almost like reading a to-do list for the agent. The workflow runs every weekday at 14:00 UTC, checks for unlabeled issues, and applies appropriate labels based on content analysis. It even leaves a friendly comment explaining the label choice.

In the frontmatter, we define permissions, tools, and safe outputs. This ensures the agent only has access to what it needs and can't perform any unsafe actions. The natural language instructions in the body guide the agent's behavior in a clear, human-readable way.

## Summarization Workflows

To continue the tour, let's look briefly at two summarization workflows that help us stay on top of repository activity. These agents digest large amounts of information and present it in a concise, readable format.

First, the **[Weekly Issue Summary](https://github.com/githubnext/gh-aw/tree/532a0412680638e5e93b6e8c5ea9b8074fe6be22/.github/workflows/weekly-issue-summary.md?plain=1)** creates digestible summaries complete with charts and trends (because who has time to read everything?)

Next, the **[Daily Repo Chronicle](https://github.com/githubnext/gh-aw/tree/532a0412680638e5e93b6e8c5ea9b8074fe6be22/.github/workflows/daily-repo-chronicle.md?plain=1)** marrates the day's activity like a storyteller - seriously, it's kind of delightful.

## Learnings

What surprised us most about this category?

First, the **reduction of cognitive load**. Having these agents handle triage and summarization freed up mental bandwidth for more important work. We no longer had to constantly monitor incoming issues or sift through activity logs - the agents did it for us, delivering only the essentials. This drastically reduced context switching and decision fatigue.

Second, the **tone** matters way more than we expected. When the Daily Repo Chronicle started writing summaries in a narrative, almost journalistic style, people actually *wanted* to read them. We discovered that AI agents don't have to be robotic - they can have personality while still being informative. The

Third, **customization** is key. Triage differs in every repository. Team needs for activity summaries and actions that arise from them differ in every repository. Tailoring these workflows to our specific context made them far more effective. Generic agents are okay, but customized ones are game-changers.

Finally, these workflows became part of our routine. The Daily Repo Chronicle was a morning coffee companion, giving us a quick overview of what happened overnight while we sipped. For teams that move fast using agents, these are key.

## Next Up: Code Quality & Refactoring Workflows

Now that we've explored how triage and summarization workflows help us stay on top of incoming activity, let's turn to the agents that continuously improve code quality. These perfectionist agents work quietly in the background, spotting refactoring opportunities and pushing your codebase toward better design.

If you'd like to skip ahead, here's the full list of articles in the series:

1. [Triage & Summarization Workflows](/gh-aw/blog/2026-01-13-meet-the-workflows/)
2. [Code Quality & Refactoring Workflows](/gh-aw/blog/2026-01-13-meet-the-workflows-code-quality/)
3. [Documentation & Content Workflows](/gh-aw/blog/2026-01-13-meet-the-workflows-documentation/)
4. [Issue & PR Management Workflows](/gh-aw/blog/2026-01-13-meet-the-workflows-issue-management/)
5. [Quality & Hygiene Workflows](/gh-aw/blog/2026-01-13-meet-the-workflows-quality-hygiene/)
6. [Metrics & Analytics Workflows](/gh-aw/blog/2026-01-13-meet-the-workflows-metrics-analytics/)
7. [Operations & Release Workflows](/gh-aw/blog/2026-01-13-meet-the-workflows-operations-release/)
8. [Security & Compliance Workflows](/gh-aw/blog/2026-01-13-meet-the-workflows-security-compliance/)
9. [Creative & Culture Workflows](/gh-aw/blog/2026-01-13-meet-the-workflows-creative-culture/)
10. [Interactive & ChatOps Workflows](/gh-aw/blog/2026-01-13-meet-the-workflows-interactive-chatops/)
11. [Testing & Validation Workflows](/gh-aw/blog/2026-01-13-meet-the-workflows-testing-validation/)
12. [Tool & Infrastructure Workflows](/gh-aw/blog/2026-01-13-meet-the-workflows-tool-infrastructure/)
13. [Multi-Phase Improver Workflows](/gh-aw/blog/2026-01-13-meet-the-workflows-multi-phase/)
14. [Organization & Cross-Repo Workflows](/gh-aw/blog/2026-01-13-meet-the-workflows-organization/)
15. [Advanced Analytics & ML Workflows](/gh-aw/blog/2026-01-13-meet-the-workflows-advanced-analytics/)
16. [Campaigns & Project Coordination Workflows](/gh-aw/blog/2026-01-13-meet-the-workflows-campaigns/)

Continue reading: [Code Quality & Refactoring Workflows →](/gh-aw/blog/2026-01-13-meet-the-workflows-code-quality/)

## Learn More

- **[GitHub Agentic Workflows](https://githubnext.github.io/gh-aw/)** - The technology behind the workflows
- **[Quick Start](https://githubnext.github.io/gh-aw/setup/quick-start/)** - How to write and compile workflows

---

*This is part 1 of a 16-part series exploring the workflows in Peli's Agent Factory.*
