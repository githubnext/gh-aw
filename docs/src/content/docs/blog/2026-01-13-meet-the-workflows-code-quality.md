---
title: "Meet the Workflows in Peli's Agent Factory: Code Quality & Refactoring"
description: "A curated tour of code quality workflows that make codebases cleaner"
authors:
  - dsyme
  - peli
date: 2026-01-13
sidebar:
  label: "Code Quality & Refactoring"
prev:
  link: /gh-aw/blog/2026-01-13-meet-the-workflows-interactive-chatops/
  label: "Interactive & ChatOps Workflows"
next:
  link: /gh-aw/blog/2026-01-13-meet-the-workflows-testing-validation/
  label: "Testing & Validation Workflows"
---

<img src="/gh-aw/peli.png" alt="Peli de Halleux" width="200" style="float: right; margin: 0 0 20px 20px; border-radius: 8px;" />

Welcome to the next chapter in [Peli's Agent Factory](/gh-aw/blog/2026-01-12-welcome-to-pelis-agent-factory/)!

We've explored reactive workflows (triage and quality checks), observability workflows (metrics and analytics), scheduled workflows (releases and culture), and [on-demand workflows](/gh-aw/blog/2026-01-13-meet-the-workflows-interactive-chatops/) (ChatOps and interactive agents). Each category serves a distinct purpose in our agent ecosystem.

Now let's talk about the agents that work quietly in the background, continuously pushing your codebase toward better design. Code quality and refactoring workflows never take a day off - they analyze console output styling, spot semantic duplication, identify structural improvements, and find patterns humans miss because they can hold entire codebases in context. These workflows embody the principle that *good enough* can always become *better*, and that incremental improvements compound over time. Let's meet the perfectionist agents.

## Code Quality & Refactoring Workflows

These agents make our codebase cleaner and our developer experience better:

- **[Terminal Stylist](https://github.com/githubnext/gh-aw/tree/532a0412680638e5e93b6e8c5ea9b8074fe6be22/.github/workflows/terminal-stylist.md?plain=1)** - Analyzes and improves console output styling (because aesthetics matter!)
- **[Semantic Function Refactor](https://github.com/githubnext/gh-aw/tree/532a0412680638e5e93b6e8c5ea9b8074fe6be22/.github/workflows/semantic-function-refactor.md?plain=1)** - Spots refactoring opportunities we might have missed
- **[Repository Quality Improver](https://github.com/githubnext/gh-aw/tree/532a0412680638e5e93b6e8c5ea9b8074fe6be22/.github/workflows/repository-quality-improver.md?plain=1)** - Takes a holistic view of code quality and suggests improvements

Code quality is where AI agents really shine - they never get bored doing the repetitive analysis that makes codebases better. The Terminal Stylist literally reads our console output code and suggests improvements to make our CLI prettier (and yes, it understands Lipgloss and modern terminal styling). The Semantic Function Refactor finds duplicated logic that's not quite identical enough for traditional duplicate detection. We learned that these agents see patterns humans miss because they can hold the entire codebase in context. The Repository Quality Improver takes a holistic view - it doesn't just find bugs, it identifies structural improvements and documentation gaps.

These workflows continuously push our codebase toward better design.

## Learn More

- **[GitHub Agentic Workflows](https://githubnext.github.io/gh-aw/)** - The technology behind the workflows
- **[Quick Start](https://githubnext.github.io/gh-aw/setup/quick-start/)** - How to write and compile workflows

## Next Up: Testing & Validation

Making code better is great, but we also need to ensure improvements don't break existing functionality. Trust, but verify.

Continue reading: [Testing & Validation Workflows →](/gh-aw/blog/2026-01-13-meet-the-workflows-testing-validation/)

---

*This is part 8 of a 16-part series exploring the workflows in Peli's Agent Factory.*
