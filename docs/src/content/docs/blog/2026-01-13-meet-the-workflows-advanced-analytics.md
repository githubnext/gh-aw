---
title: "Meet the Workflows: Advanced Analytics & ML"
description: "A curated tour of workflows that use ML to extract insights from agent behavior"
authors:
  - dsyme
  - pelikhan
  - mnkiefer
date: 2026-01-13T15:00:00
sidebar:
  label: "Advanced Analytics & ML"
prev:
  link: /gh-aw/blog/2026-01-13-meet-the-workflows-organization/
  label: "Organization & Cross-Repo Workflows"
next:
  link: /gh-aw/blog/2026-01-13-meet-the-workflows-campaigns/
  label: "Campaigns & Project Coordination Workflows"
---

<img src="/gh-aw/peli.png" alt="Peli de Halleux" width="200" style="float: right; margin: 0 0 20px 20px; border-radius: 8px;" />

*Ooh!* Time to plunge into the *data wonderland* at [Peli's Agent Factory](/gh-aw/blog/2026-01-12-welcome-to-pelis-agent-factory/)! Where numbers dance and patterns sing!

In our [previous post](/gh-aw/blog/2026-01-13-meet-the-workflows-organization/), we explored organization and cross-repo workflows that operate at enterprise scale - analyzing dozens of repositories together to find patterns and outliers that single-repo analysis would miss. We learned that perspective matters: what looks normal in isolation might signal drift at scale.

Beyond tracking basic metrics (run time, cost, success rate), we wanted deeper insights into *how* our agents actually behave and *how* developers interact with them. What patterns emerge from thousands of agent prompts? What makes some PR conversations more effective than others? How do usage patterns reveal improvement opportunities? This is where we brought out the big guns: machine learning, natural language processing, sentiment analysis, and clustering algorithms. Advanced analytics workflows don't just count things - they understand them, finding patterns and insights that direct observation would never reveal.

## Advanced Analytics & ML Workflows

These agents use sophisticated analysis techniques to extract insights:

- **[Copilot Session Insights](https://github.com/githubnext/gh-aw/tree/bb7946527af340043f1ebb31fc21bd491dd0f42d/.github/workflows/copilot-session-insights.md?plain=1)** - Analyzes Copilot agent usage patterns and metrics  
- **[Copilot PR NLP Analysis](https://github.com/githubnext/gh-aw/tree/bb7946527af340043f1ebb31fc21bd491dd0f42d/.github/workflows/copilot-pr-nlp-analysis.md?plain=1)** - Natural language processing on PR conversations  
- **[Prompt Clustering Analysis](https://github.com/githubnext/gh-aw/tree/bb7946527af340043f1ebb31fc21bd491dd0f42d/.github/workflows/prompt-clustering-analysis.md?plain=1)** - Clusters and categorizes agent prompts using ML  
- **[Copilot Agent Analysis](https://github.com/githubnext/gh-aw/tree/bb7946527af340043f1ebb31fc21bd491dd0f42d/.github/workflows/copilot-agent-analysis.md?plain=1)** - Deep analysis of agent behavior patterns  

The Prompt Clustering Analysis uses machine learning to categorize thousands of agent prompts, revealing patterns we never noticed ("oh, 40% of our prompts are about error handling").

The Copilot PR NLP Analysis does sentiment analysis and linguistic analysis on PR conversations - it found that PRs with questions in the title get faster review.

The Session Insights workflow analyzes how developers interact with Copilot agents, identifying common patterns and failure modes. What we learned: **meta-analysis is powerful** - using AI to analyze AI systems reveals insights that direct observation misses.

These workflows helped us understand not just what our agents do, but *how* they behave and how users interact with them.

## Learn More

- **[GitHub Agentic Workflows](https://githubnext.github.io/gh-aw/)** - The technology behind the workflows
- **[Quick Start](https://githubnext.github.io/gh-aw/setup/quick-start/)** - How to write and compile workflows

## Next Up: Campaigns & Project Coordination Workflows

We've reached the final stop: coordinating multiple agents toward shared, complex goals across extended timelines.

Continue reading: [Campaigns & Project Coordination Workflows →](/gh-aw/blog/2026-01-13-meet-the-workflows-campaigns/)

---

*This is part 15 of a 16-part series exploring the workflows in Peli's Agent Factory.*
