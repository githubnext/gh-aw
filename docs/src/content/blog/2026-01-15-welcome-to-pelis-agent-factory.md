---
title: "Welcome to Peli's Agent Factory"
description: "An exploration of automated agentic workflows at scale"
authors:
  - gh-next
date: 2026-01-15
draft: true
---

<img src="/peli.png" alt="Peli de Halleux" width="200" style="float: right; margin: 0 0 20px 20px; border-radius: 8px;" />

Hey there! 👋

At GitHub Next, we have the privilege of collaborating with amazing people across GitHub and the broader AI community to explore the future of software development. Today, we're excited to introduce **Peli's Agent Factory** - a research collaboration between GitHub Next and Peli de Halleux from Microsoft Research.

<div align="center">
  <img src="/gh-aw/peli.png" alt="Peli de Halleux" width="250" style="border-radius: 8px; margin: 20px 0;" />
</div>

Software development is changing rapidly. Peli's Agent Factory is our exploration into what happens when you build and operate **145 specialized AI automated agents** within a real software repository. These aren't demos or proof-of-concepts - they're working agents handling actual development tasks. This is our attempt to understand how automated agentic AI can make software development more efficient, collaborative, and maybe even enjoyable.

Welcome to the factory. Let's explore together.

## What Is Peli's Agent Factory?

Imagine a software repository where AI agents work alongside your team - not replacing developers, but handling the repetitive, time-consuming tasks that slow down collaboration and progress. That's what we're building.

The factory is an experimental collection of **over 145 autonomous agentic workflows**, each solving specific problems:

- **Triage incoming issues** the moment they arrive
- **Diagnose CI failures** and open detailed reports
- **Maintain documentation** to prevent drift
- **Improve test coverage** incrementally
- **Monitor security compliance** continuously
- **Analyze agent performance** to optimize the ecosystem
- **Execute multi-day projects** to reduce technical debt

Think of this as an incubation lab where each agent has its own mode of operation, needs, and interactions. Some are simple read-only analysts. Others proactively propose changes through pull requests. A few are meta-agents that monitor and improve the health of all the other workflows.

It's basically an agent zoo. And we're learning *so much* from watching them.

## Why Build a Factory?

When we started exploring agentic workflows, we faced a fundamental question: **What should repository-level automated agentic workflows actually do?**

Rather than trying to build one "perfect" agent, we took a gardener's approach:

1. **Embrace diversity** - Create many specialized workflows for different tasks
2. **Use them continuously** - Run them in real development workflows
3. **Observe what thrives** - Find which patterns work and which fail
4. **Share the knowledge** - Catalog the structures that make agents safe and effective

The factory becomes both an experiment and a reference collection - a living library of patterns that others can study, adapt, and remix.

## The Factory at a Glance

Here's what we've built so far:

- **145 total workflows** demonstrating diverse agent patterns
- **12 core design patterns** consolidating all observed behaviors
- **9 operational patterns** for GitHub-native agent orchestration
- **128 workflows** in the `.github/workflows` directory of the [`gh-aw`](https://github.com/githubnext/gh-aw/tree/main/.github/workflows) repository
- **17 curated workflows** in the installable [`agentics`](https://github.com/githubnext/agentics) collection
- **Dozens of MCP servers** integrated for specialized capabilities
- **Multiple trigger types**: schedules, slash commands, reactions, workflow events, issue labels

Each workflow is written in natural language using Markdown, then compiled into secure GitHub Actions that run with carefully scoped permissions. Everything is observable, auditable, and remixable.

## What We're Learning

Running this many agents in production is... quite the experience. We've watched agents succeed spectacularly, fail in interesting ways, and surprise us constantly. Some key lessons emerging:

- **Diversity beats perfection** - A collection of focused agents works better than one universal assistant
- **Guardrails enable innovation** - Strict constraints actually make it easier to experiment safely
- **Meta-agents are essential** - Agents that watch other agents become incredibly valuable
- **Personality matters** - Agents with distinct characters (like our Grumpy Reviewer or Poem Bot) are easier to trust
- **Cost-quality tradeoffs are real** - Longer analyses aren't always better

We'll dive deeper into these lessons in upcoming articles.

## What's Coming in This Series

Over the next few weeks, we'll be sharing what we've learned through a series of detailed articles. We'll be looking at the most interesting agents, the design and operational patterns we've discovered, security lessons, and practical guides for building your own workflows.

## Try It Yourself

Want to start your own agent factory? See our [Quick Start](https://githubnext.github.io/gh-aw/setup/quick-start/). Our advice is:

1. **Start small** - Pick one tedious task (issue triage, CI diagnosis, weekly summaries)
2. **Use the analyst pattern first** - Read-only agents that post to discussions are safest and least intrusive
3. **Nurture continuously** - Let it run and observe its behavior
4. **Iterate** - Refine based on what actually helps your team
5. **Plant more seeds** - Once one agent works, add complementary ones

The workflows in Peli's factory are fully remixable. Copy them, adapt them, and make them your own. Every workflow is open source and documented.

## Learn More

- **[GitHub Agentic Workflows Documentation](https://githubnext.github.io/gh-aw/)** - How to write and compile workflows
- **[GitHub Agentic Quick Start](https://githubnext.github.io/gh-aw/setup/quick-start/)** - How to write and compile workflows
- **[gh-aw Repository](https://github.com/githubnext/gh-aw)** - The factory's home
- **[The Agentics Collection](https://github.com/githubnext/agentics)** - Ready-to-install workflows
- **[The Continuous AI Project](https://githubnext.com/projects/continuous-ai)** - The broader vision

## Credits

**Peli's Agent Factory** is a research project by GitHub Next Agentic Workflows contributors and collaborators: Peli de Halleux, Don Syme, Mara Kiefer, Edward Aftandilian, Russell Horton, Jiaxiao Zhou, and many others.

This is part of GitHub Next's exploration of [Continuous AI](https://githubnext.com/projects/continuous-ai) - making AI-enriched automation as routine as CI/CD.

---

[Next Article: Meet the Workflows](/gh-aw/blog/2026-01-18-meet-the-workflows/)
