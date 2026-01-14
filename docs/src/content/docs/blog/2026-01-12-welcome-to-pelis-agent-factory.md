---
title: "Welcome to Peli's Agent Factory"
description: "An exploration of automated agentic workflows at scale"
authors:
  - dsyme
  - peli
date: 2026-01-12
next:
  link: /gh-aw/blog/2026-01-13-meet-the-workflows/
  label: Meet the Workflows
---

<img src="https://avatars.githubusercontent.com/pelikhan" alt="Peli de Halleux" width="200" style="float: right; margin: 0 0 20px 20px; border-radius: 8px;" />

Hi there! 👋 Welcome to Peli's Agent Factory!

Imagine a software repository where AI agents work alongside your team - not replacing developers, but handling the repetitive, time-consuming tasks that slow down collaboration and forward progress.

Peli's Agent Factory is our exploration of what happens when you take the design philosophy of **let's create a new agentic workflow for that** as the answer to every opportunity that may present itself! What happens when you **max out on automated agentic workflows** - when you make and nurture dozens of specialized, automated AI agentic workflows in a real repository.

Software development is changing rapidly. Peli's Agent Factory is our exploration into what happens when you build and operate **a collection of specialized AI automated agentic workflows** within a real software repository, handling actual development tasks. This is our attempt to understand how automated agentic AI can make software teams more efficient, collaborative, and maybe even enjoyable.

Welcome to the factory. Let's explore together!

## What Is Peli's Agent Factory?

Peli's factory is a collection of **automated agentic workflows** we use in practice. 

Over the course of this research project, we built and operated **over 100 automated agentic workflows** within the [`githubnext/gh-aw`](https://github.com/githubnext/gh-aw) repository and its companion [`githubnext/agentics`](https://github.com/githubnext/agentics) collection. These were used mostly in the context of the [`githubnext/gh-aw`](https://github.com/githubnext/gh-aw) project itself, but many have also been applied at scale to GitHub and Microsoft internal repositories, and some external repositories. These weren't hypothetical demos - they were working agents that:

- Triage incoming issues
- Diagnose CI failures
- Maintain documentation
- Improve test coverage
- Monitor security compliance
- Optimize workflow efficiency
- Execute multi-day projects
- Validate infrastructure
- Even write poetry to boost team morale

Think of this as a team where as part of our work we're creating a lot of agentic workflows. Some are simple read-only analysts. Others proactively propose changes through pull requests. A few are meta-agents that monitor and improve the health of all the other workflows.

We know we're taking things to an extreme here. Most repositories won't need dozens of agentic workflows. No one can read all these outputs (except, of course, another workflow). But by pushing the boundaries, we learned valuable lessons about what works, what doesn't, and how to design safe, effective agentic workflows that teams can trust and use.

It's basically an agentic workflow cornucopia. And we're learning *so much* from it all, we'd like to share it with you.

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

- **A comprehensive collection of workflows** demonstrating diverse agent patterns
- **12 core design patterns** consolidating all observed behaviors
- **9 operational patterns** for GitHub-native agent orchestration
- **128 workflows** in the `.github/workflows` directory of the [`gh-aw`](https://github.com/githubnext/gh-aw/tree/532a0412680638e5e93b6e8c5ea9b8074fe6be22/.github/workflows) repository
- **17 curated workflows** in the installable [`agentics`](https://github.com/githubnext/agentics) collection
- **Dozens of MCP servers** integrated for specialized capabilities
- **Multiple trigger types**: schedules, slash commands, reactions, workflow events, issue labels

Each workflow is written in natural language using Markdown, then compiled into secure GitHub Actions that run with carefully scoped permissions. Everything is observable, auditable, and remixable.

## Our First Installment: Meet the Workflows

In our first series of articles, [Meet the Workflows](/gh-aw/blog/2026-01-13-meet-the-workflows/), we'll take you on a 16-part tour of the most interesting agents in the factory. You'll see how they operate, what problems they solve, and the unique personalities we've given them.

Over the next few weeks, we'll also be sharing what we've learned through a series of detailed articles. We'll be looking at the design and operational patterns we've discovered, security lessons, and practical guides for building your own workflows.

## What We're Learning

Running this many agents in production is... quite the experience. We've watched agents succeed spectacularly, fail in interesting ways, and surprise us constantly. Some key lessons emerging:

- **Repository-level automation is incredibly powerful** - Agents embedded in the development workflow can have outsized impact
- **Diversity beats perfection** - A collection of focused agents works better than one universal assistant
- **Guardrails enable innovation** - Strict constraints actually make it easier to experiment safely
- **Meta-agents are valuable** - Agents that watch other agents become incredibly valuable
- **Personality matters** - Agents with distinct characters (like our Grumpy Reviewer or Poem Bot) are easier to trust
- **Cost-quality tradeoffs are real** - Longer analyses aren't always better

We'll dive deeper into these lessons in upcoming articles.

## Try It Yourself

Want to start with automated agentic workflows on GitHub? See our [Quick Start](https://githubnext.github.io/gh-aw/setup/quick-start/). Our advice is:

1. **Start small** - Pick one tedious task (issue triage, CI diagnosis, weekly summaries)
2. **Use the analyst pattern first** - Read-only agents that post to discussions are safest and least intrusive
3. **Nurture continuously** - Run the workflow and gather feedback
4. **Iterate** - Refine based on what actually helps your team
5. **Plant more seeds** - Once one workflow works, add complementary ones

The workflows in Peli's factory are fully remixable. You can copy them, adapt them, and make them your own.

## Learn More

- **[Meet the Workflows](/gh-aw/blog/2026-01-13-meet-the-workflows/)** - The 16-part tour of the workflows
- **[GitHub Agentic Workflows](https://githubnext.github.io/gh-aw/)** - The technology behind the workflows
- **[Quick Start](https://githubnext.github.io/gh-aw/setup/quick-start/)** - How to write and compile workflows

## Credits

**Peli's Agent Factory** is a research project by GitHub Next, Microsoft Research and collaborators, including Peli de Halleux, Don Syme, Mara Kiefer, Edward Aftandilian, Russell Horton, Jiaxiao Zhou.

This is part of GitHub Next's exploration of [Continuous AI](https://githubnext.com/projects/continuous-ai) - making AI-enriched automation as routine as CI/CD.
