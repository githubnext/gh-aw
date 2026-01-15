---
title: "Welcome to Peli's Agent Factory"
description: "An exploration of automated agentic workflows at scale"
authors:
  - dsyme
  - peli
  - mnkiefer
date: 2026-01-12
featured: true
next:
  link: /gh-aw/blog/2026-01-13-meet-the-workflows/
  label: Meet the Workflows
---

<img src="/gh-aw/peli.png" alt="Peli de Halleux" width="200" style="float: right; margin: 0 0 20px 20px; border-radius: 8px;" />

Hi there! 👋 Welcome to Peli's Agent Factory!

Imagine a software repository where AI agents work alongside your team - not replacing developers, but handling the repetitive, time-consuming tasks that slow down collaboration and forward progress.

Peli's Agent Factory is our exploration of what happens when you take the design philosophy of **"let's create a new automated agentic workflow for that"** as the answer to almost every opportunity that arises! What happens when you **max out on automated agentic workflows** - when you make and use dozens of specialized, automated AI agentic workflows and use them in real repositories.

Software development is changing rapidly. This is our attempt to understand how automated agentic AI can make software teams more efficient, collaborative, and more enjoyable.

Welcome to the factory. Let's explore together!

## What Is Peli's Agent Factory?

Peli's factory is a collection of [**automated agentic workflows**](https://https://githubnext.github.io/gh-aw) we use in practice. Over the course of this research project, we built and operated **over 100 automated agentic workflows** within the [`githubnext/gh-aw`](https://github.com/githubnext/gh-aw) repository and its companion [`githubnext/agentics`](https://github.com/githubnext/agentics) collection. These were used mostly in the context of the [`githubnext/gh-aw`](https://github.com/githubnext/gh-aw) project itself, but some have also been applied at scale in GitHub and Microsoft internal repositories, and some external repositories. These weren't hypothetical demos - they were working agents that:

- Triage incoming issues
- Diagnose CI failures
- Maintain documentation
- Improve test coverage
- Monitor security compliance
- Optimize workflow efficiency
- Execute multi-day projects
- Validate infrastructure
- Even write poetry to boost team morale

Some workflows are "read-only analysts". Others proactively propose changes through pull requests. Some are meta-agents that monitor and improve the health of all the other workflows.

We know we're taking things to an extreme here. Most repositories won't need dozens of agentic workflows. No one can read all these outputs (except, of course, another workflow). But by pushing the boundaries, we learned valuable lessons about what works, what doesn't, and how to design safe, effective agentic workflows that teams can trust and use.

It's basically a candy shop chocolate factory of agentic workflows. And we're learning so much from it all, we'd like to share it with you.

## Why Build a Factory?

When we started exploring agentic workflows, we faced a fundamental question: **What should repository-level automated agentic workflows actually do?**

Rather than trying to build one "perfect" agent, we took a broad, heterogeneous approach:

1. **Embrace diversity** - Create many specialized workflows as we identified opportunities
2. **Use them continuously** - Run them in real development workflows
3. **Observe what works** - Find which patterns work and which fail
4. **Share the knowledge** - Catalog the structures that make agents safe and effective

The factory becomes both an experiment and a reference collection - a living library of patterns that others can study, adapt, and remix.

Here's what we've built so far:

- **A comprehensive collection of workflows** demonstrating diverse agent patterns
- **12 core design patterns** consolidating all observed behaviors
- **9 operational patterns** for GitHub-native agent orchestration
- **128 workflows** in the `.github/workflows` directory of the [`gh-aw`](https://github.com/githubnext/gh-aw/tree/bb7946527af340043f1ebb31fc21bd491dd0f42d/.github/workflows) repository
- **17 curated workflows** in the installable [`agentics`](https://github.com/githubnext/agentics) collection
- **Multiple trigger types**: schedules, slash commands, reactions, workflow events, issue labels

Each workflow is written in natural language using Markdown, then compiled into secure GitHub Actions that run with carefully scoped permissions. Everything is observable, auditable, and remixable.

## Meet the Workflows

In our first series, [Meet the Workflows](/gh-aw/blog/2026-01-13-meet-the-workflows/), we'll take you on a 16-part tour of the most interesting agents in the factory. You'll see how they operate, what problems they solve, and the unique personalities we've given them.

Each article is bite-sized. Start with [Meet the Workflows](/gh-aw/blog/2026-01-13-meet-the-workflows/) to get an overview, then dive into the ones that catch your eye. If you'd like to skip ahead, here's the full list of articles in the series:

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

## What We're Learning

Running this many agents in production is... quite the experience. We've watched agents succeed spectacularly, fail in interesting ways, and surprise us constantly. Over the next few weeks, we'll also be sharing what we've learned through a series of detailed articles. We'll be looking at the design and operational patterns we've discovered, security lessons, and practical guides for building your own workflows.

To give a taste, some key lessons are emerging:

- **Repository-level automation is incredibly powerful** - Agents embedded in the development workflow can have outsized impact
- **Diversity beats perfection** - A collection of focused agents works better than one universal assistant
- **Guardrails enable innovation** - Strict constraints actually make it easier to experiment safely
- **Meta-agents are valuable** - Agents that watch other agents become incredibly valuable
- **Cost-quality tradeoffs are real** - Longer analyses aren't always better

We'll dive deeper into these lessons in upcoming articles.

## Try It Yourself

Want to start with automated agentic workflows on GitHub? See our [Quick Start](https://githubnext.github.io/gh-aw/setup/quick-start/).

## Learn More

- **[Meet the Workflows](/gh-aw/blog/2026-01-13-meet-the-workflows/)** - The 16-part tour of the workflows
- **[GitHub Agentic Workflows](https://githubnext.github.io/gh-aw/)** - The technology behind the workflows
- **[Quick Start](https://githubnext.github.io/gh-aw/setup/quick-start/)** - How to write and compile workflows

**Peli's Agent Factory** is a research project by GitHub Next, Microsoft Research and collaborators, including Peli de Halleux, Don Syme, Mara Kiefer, Edward Aftandilian, Russell Horton, Jiaxiao Zhou.

This is part of GitHub Next's exploration of [Continuous AI](https://githubnext.com/projects/continuous-ai) - making AI-enriched automation as routine as CI/CD.
