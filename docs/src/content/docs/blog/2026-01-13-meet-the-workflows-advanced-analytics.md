---
title: "Meet the Workflows in Peli's Agent Factory: Advanced Analytics & ML"
description: "A curated tour of workflows that use ML to extract insights from agent behavior"
authors:
  - dsyme
  - peli
date: 2026-01-13
prev:
  link: /gh-aw/blog/2026-01-13-meet-the-workflows-multi-phase/
  label: "Multi-Phase Improver Workflows"
next:
  link: /gh-aw/blog/2026-01-13-meet-the-workflows-organization/
  label: "Organization & Cross-Repo Workflows"
---

<img src="/gh-aw/peli.png" alt="Peli de Halleux" width="200" style="float: right; margin: 0 0 20px 20px; border-radius: 8px;" />

Time to get nerdy at Peli's Agent Factory!

We just explored [multi-phase improver workflows](/gh-aw/blog/2026-01-13-meet-the-workflows-multi-phase/) - our most ambitious agents that tackle big projects over multiple days, maintaining state and making incremental progress. These workflows proved that AI agents can handle complex, long-running initiatives when given proper architecture.

Now let's get nerdy. Beyond tracking basic metrics (run time, cost, success rate), we wanted deeper insights into *how* our agents actually behave and *how* developers interact with them. What patterns emerge from thousands of agent prompts? What makes some PR conversations more effective than others? How do usage patterns reveal improvement opportunities? This is where we brought out the big guns: machine learning, natural language processing, sentiment analysis, and clustering algorithms. Advanced analytics workflows don't just count things - they understand them, finding patterns and insights that direct observation would never reveal.

## 📊 Advanced Analytics & ML Workflows

These agents use sophisticated analysis techniques to extract insights:

- **[Copilot Session Insights](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/copilot-session-insights.md)** - Analyzes Copilot agent usage patterns and metrics
- **[Copilot PR NLP Analysis](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/copilot-pr-nlp-analysis.md)** - Natural language processing on PR conversations
- **[Prompt Clustering Analysis](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/prompt-clustering-analysis.md)** - Clusters and categorizes agent prompts using ML
- **[Copilot Agent Analysis](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/copilot-agent-analysis.md)** - Deep analysis of agent behavior patterns

We got nerdy with these workflows. The Prompt Clustering Analysis uses machine learning to categorize thousands of agent prompts, revealing patterns we never noticed ("oh, 40% of our prompts are about error handling"). The Copilot PR NLP Analysis does sentiment analysis and linguistic analysis on PR conversations - it found that PRs with questions in the title get faster review. The Session Insights workflow analyzes how developers interact with Copilot agents, identifying common patterns and failure modes. What we learned: **meta-analysis is powerful** - using AI to analyze AI systems reveals insights that direct observation misses.

These workflows helped us understand not just what our agents do, but *how* they behave and how users interact with them.

## In the Next Stage of Our Journey...

Now that we've explored sophisticated analytics on individual repositories, let's zoom out to organization scale - workflows that operate across multiple repositories to provide enterprise-wide insights and management.

Continue reading: [Organization & Cross-Repo Workflows →](/gh-aw/blog/2026-01-13-meet-the-workflows-organization/)

---

*This is part 12 of a 16-part series exploring the workflows in Peli's Agent Factory.*
