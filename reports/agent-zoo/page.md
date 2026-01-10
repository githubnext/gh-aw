# Peli's Agent Factory
**An exploration of automated agentic workflows at scale**

At GitHub Next, we have the privilege of collaborating with amazing people across GitHub and the broader AI community to explore the future of software development. In this project, we introduce one of our collaborations - one we're calling **Peli's Agent Factory**.

<div align="right"> <img src="peli.png" alt="Peli de Halleux" width="250" /> </div>

Peli's Agent Factory is a research collaboration between GitHub Next and Peli de Halleux from Microsoft Research, exploring what happens when you build and operate dozens of specialized AI agents within a real software repository.

Software development is changing rapidly. In this project, you'll explore with us a collection of over 145 automated agentic workflows. These are working agents that handle real development tasks. Peli's Agent Factory is our attempt to understand how to harness automated agentic AI to make software development more efficient, collaborative, and enjoyable. Entering Peli's Agent Factory is like a new world of possibilities - one that may unfolds differently for each repository, company or software community. We invite you to explore and learn with us.

## What is Peli's Agent Factory?

Imagine a software repository where AI agents work alongside your team - not replacing developers, but handling the repetitive, time-consuming tasks that slow down collaboration and forward progress. Peli's Agent Factory is an experimental collection of over 145 autonomous agentic workflows that each solve specific problems:

- Triage incoming issues
- Diagnose CI failures
- Maintain documentation
- Improve test coverage
- Monitor security compliance
- Analyze agent performance
- Execute multi-day projects to reduce technical debt

Think of this as an incubation lab where each agent has its own mode of operation, needs, and interactions. Some are simple read-only analysts. Others proactively propose changes through pull requests. A few are meta-agents that monitor and improve the health of all the other workflows.

## Explore the Factory

We've documented our journey through a series of detailed articles:

1. [**Meet the Agentic Workflows**](articles/01-meet-the-workflows.md) - A curated tour of the most interesting agents in the factory
2. [**12 Lessons from Peli's Agent Factory**](articles/02-twelve-lessons.md) - Key insights about what works, what doesn't, and how to design effective agent ecosystems
3. [**12 Design Patterns from Peli's Agent Factory**](articles/03-design-patterns.md) - Fundamental behavioral patterns for successful agentic workflows
4. [**9 Patterns for Automated Agent Ops on GitHub**](articles/04-operational-patterns.md) - Strategic patterns for operating agents in the GitHub ecosystem
5. [**Imports & Sharing: Peli's Secret Weapon**](articles/05-imports-and-sharing.md) - How modular, reusable components enabled scaling to 145 agents
6. [**Security Lessons from the Agent Factory**](articles/06-security-lessons.md) - Designing safe environments where agents can't accidentally cause harm
7. [**How Agentic Workflows Work**](articles/07-how-workflows-work.md) - The technical foundation: from natural language to secure execution
8. [**Authoring New Workflows in Peli's Agent Factory**](articles/08-authoring-workflows.md) - A practical guide to creating your own agentic workflows
9. [**Getting Started with Agentic Workflows**](articles/09-getting-started.md) - Begin your journey with agentic automation

## Why Build a Factory?

When we started exploring agentic workflows, we faced a fundamental question: **What should repository-level automated agentic workflows actually do?**

Rather than trying to build one "perfect" agent, we took a gardener's approach:
1. **Embrace diversity** - Create diverse agentic workflows for different tasks
2. **Use them and improve them** - Run them continuously in real development workflows
3. **Identify what thrives** - Find which patterns worked and which failed
4. **Share the knowledge** - Catalog the structures that made agents safe and effective

The factory becomes both an experiment and a reference collection - a living library of patterns that others can study, adapt, and remix.

## The Factory at a Glance

- **145 total workflows** demonstrating diverse agent patterns
- **12 core design patterns** consolidating all observed behaviors
- **9 operational patterns** for GitHub-native agent orchestration
- **128 workflows** in the main [`gh-aw`](https://github.com/githubnext/gh-aw) repository
- **17 curated workflows** in the installable [`agentics`](https://github.com/githubnext/agentics) collection
- **Dozens of MCP servers** integrated for specialized capabilities
- **Multiple trigger types**: schedules, slash commands, reactions, workflow events, issue labels

## Try It Yourself

Want to start your own agent factory?

1. **Start Small**: Pick one tedious task (issue triage, CI diagnosis, weekly summaries)
2. **Use the Analyst Pattern**: Read-only agents that post to discussions
3. **Nurture Continuously**: Let it run and observe
4. **Iterate**: Refine based on what actually helps your team
5. **Plant More Seeds**: Once one agent works, add complementary ones

The workflows in this factory are fully remixable. Copy them, adapt them, and make them your own.

## Learn More

- **[GitHub Agentic Workflows Documentation](https://githubnext.github.io/gh-aw/)** - How to write and compile workflows
- **[gh-aw Repository](https://github.com/githubnext/gh-aw)** - The factory's home
- **[Agentics Collection](https://github.com/githubnext/agentics)** - Ready-to-install workflows
- **[Continuous AI Project](https://githubnext.com/projects/continuous-ai)** - The broader vision

## Credits

**Peli's Agent Factory** was a research project by GitHub Next Agentic Workflows contributors and collaborators:

Peli de Halleux, Don Syme, Mara Kiefer, Edward Aftandilian, Krzysztof Cieślak, Russell Horton, Ben De St Paer‑Gotch, Jiaxiao Zhou, Daniel Mappiel.

*Part of GitHub Next's exploration of [Continuous AI](https://githubnext.com/projects/continuous-ai) - making AI-enriched automation as routine as CI/CD.*

---

*Last updated: January 2026*
