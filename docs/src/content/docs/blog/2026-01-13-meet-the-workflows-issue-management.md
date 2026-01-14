---
title: "Meet the Workflows in Peli's Agent Factory: Issue & PR Management"
description: "A curated tour of workflows that enhance GitHub collaboration"
authors:
  - dsyme
  - peli
date: 2026-01-13
prev:
  link: /gh-aw/blog/2026-01-13-meet-the-workflows-documentation/
  label: "Documentation & Content Workflows"
next:
  link: /gh-aw/blog/2026-01-13-meet-the-workflows-campaigns/
  label: "Campaign & Project Coordination Workflows"
---

<img src="/gh-aw/peli.png" alt="Peli de Halleux" width="200" style="float: right; margin: 0 0 20px 20px; border-radius: 8px;" />

Let's talk collaboration at Peli's Agent Factory!

In our [previous post](/gh-aw/blog/2026-01-13-meet-the-workflows-documentation/), we explored documentation and content workflows - agents that maintain glossaries, technical docs, slide decks, and blog content. We learned that AI-generated docs need human review, but they're dramatically better than having no docs at all.

Now let's talk about the daily rituals of software development: managing issues and pull requests. GitHub provides excellent primitives for collaboration, but there's a lot of ceremony involved - linking related issues, merging main into PR branches, assigning work, closing completed sub-issues, optimizing templates. These are small papercuts individually, but they add up to significant friction. Issue and PR management workflows don't replace GitHub's features; they enhance them, removing tedious ceremony and making collaboration feel effortless. Let's see how automation makes GitHub more pleasant to use.

## 🔗 Issue & PR Management Workflows

These agents enhance issue and pull request workflows:

- **[Issue Arborist](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/issue-arborist.md)** - Links related issues as sub-issues
- **[Issue Monster](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/issue-monster.md)** - Assigns issues to Copilot agents one at a time
- **[Mergefest](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/mergefest.md)** - Automatically merges main branch into PR branches
- **[Sub Issue Closer](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/sub-issue-closer.md)** - Closes completed sub-issues automatically
- **[Issue Template Optimizer](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/issue-template-optimizer.md)** - Improves issue templates based on usage

Issue management is tedious ceremony that developers tolerate rather than enjoy. The Issue Arborist automatically links related issues, building a dependency tree we'd never maintain manually. The Issue Monster became our task dispatcher for AI agents - it assigns one issue at a time to Copilot agents, preventing the chaos of parallel work on the same codebase. Mergefest eliminates the "please merge main" dance that happens on long-lived PRs. We learned that **tiny frustrations add up** - each of these workflows removes a small papercut, and collectively they make GitHub feel much more pleasant to use. The Issue Template Optimizer analyzes which fields in our templates actually get filled out and suggests improvements ("nobody uses the 'Expected behavior' field, remove it").

## In the Next Stage of Our Journey...

Now that we've explored workflows that handle individual issues and PRs, let's look at the orchestration layer - campaign and project coordination workflows that manage structured improvement initiatives across multiple agents.

Continue reading: [Campaign & Project Coordination Workflows →](/gh-aw/blog/2026-01-13-meet-the-workflows-campaigns/)

---

*This is part 15 of a 16-part series exploring the workflows in Peli's Agent Factory.*
