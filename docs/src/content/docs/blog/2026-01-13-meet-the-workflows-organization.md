---
title: "Meet the Workflows in Peli's Agent Factory: Organization & Cross-Repo"
description: "A curated tour of workflows that operate at organization scale"
authors:
  - dsyme
  - peli
date: 2026-01-13
prev:
  link: /gh-aw/blog/2026-01-13-meet-the-workflows-advanced-analytics/
  label: "Advanced Analytics & ML Workflows"
next:
  link: /gh-aw/blog/2026-01-13-meet-the-workflows-documentation/
  label: "Documentation & Content Workflows"
---

<img src="/gh-aw/peli.png" alt="Peli de Halleux" width="200" style="float: right; margin: 0 0 20px 20px; border-radius: 8px;" />

Let's zoom out at [Peli's Agent Factory](/gh-aw/blog/2026-01-12-welcome-to-pelis-agent-factory/)!

In our [previous post](/gh-aw/blog/2026-01-13-meet-the-workflows-advanced-analytics/), we explored advanced analytics workflows - using machine learning and NLP to understand agent behavior patterns, developer interactions, and conversation effectiveness. These sophisticated analyses helped us understand not just what agents do, but *how* they behave.

But all that sophisticated analysis has focused on a single repository. What happens when you zoom out to organization scale? What insights emerge when you analyze dozens or hundreds of repositories together? What looks perfectly normal in one repo might be a red flag across an organization. Organization and cross-repo workflows operate at enterprise scale, requiring careful permission management, thoughtful rate limiting, and different analytical lenses. Let's explore workflows that see the forest, not just the trees.

## 🏢 Organization & Cross-Repo Workflows

These agents work at organization scale, across multiple repositories:

- **[Org Health Report](https://github.com/githubnext/gh-aw/tree/532a0412680638e5e93b6e8c5ea9b8074fe6be22/.github/workflows/org-health-report.md?plain=1)** - Organization-wide repository health metrics  
  [→ View items](https://github.com/search?q=repo%3Agithubnext/gh-aw%20%22agentic-workflow%3A%20org-health-report%22&type=issues)
- **[Stale Repo Identifier](https://github.com/githubnext/gh-aw/tree/532a0412680638e5e93b6e8c5ea9b8074fe6be22/.github/workflows/stale-repo-identifier.md?plain=1)** - Identifies inactive repositories  
  [→ View items](https://github.com/search?q=repo%3Agithubnext/gh-aw%20%22agentic-workflow%3A%20stale-repo-identifier%22&type=issues)
- **[Ubuntu Image Analyzer](https://github.com/githubnext/gh-aw/tree/532a0412680638e5e93b6e8c5ea9b8074fe6be22/.github/workflows/ubuntu-image-analyzer.md?plain=1)** - Documents GitHub Actions runner environments  
  [→ View items](https://github.com/search?q=repo%3Agithubnext/gh-aw%20%22agentic-workflow%3A%20ubuntu-image-analyzer%22&type=issues)

Scaling agents across an entire organization changes the game. The Org Health Report analyzes dozens of repositories at once, identifying patterns and outliers ("these three repos have no tests, these five haven't been updated in months"). The Stale Repo Identifier helps with organizational hygiene - finding abandoned projects that should be archived or transferred. We learned that **cross-repo insights are different** - what looks fine in one repository might be an outlier across the organization. These workflows require careful permission management (reading across repos needs organization-level tokens) and thoughtful rate limiting (you can hit API limits fast when analyzing 50+ repos). The Ubuntu Image Analyzer is wonderfully meta - it documents the very environment that runs our agents.

## Learn More

- **[GitHub Agentic Workflows](https://githubnext.github.io/gh-aw/)** - The technology behind the workflows
- **[Quick Start](https://githubnext.github.io/gh-aw/setup/quick-start/)** - How to write and compile workflows

## Next Up: Automating the Eternal Documentation Challenge

Code evolves fast. Documentation? Not so much. Can AI agents help solve this age-old problem?

Continue reading: [Documentation & Content Workflows →](/gh-aw/blog/2026-01-13-meet-the-workflows-documentation/)

---

*This is part 13 of a 16-part series exploring the workflows in Peli's Agent Factory.*
