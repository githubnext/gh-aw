---
title: "Meet the Agentic Workflows in Peli's Agent Factory"
description: "A curated tour of the most interesting agents in the factory"
authors:
  - gh-next
date: 2026-01-12
---

<img src="/peli.png" alt="Peli de Halleux" width="200" style="float: right; margin: 0 0 20px 20px; border-radius: 8px;" />

Hi there! 👋 Welcome back to Peli's Agent Factory.

We're the GitHub Next team, and we've been on quite a journey. Over the past months, we've built and operated a collection of automated agentic workflows. These aren't just demos or proof-of-concepts - these are real agents doing actual work in our [`githubnext/gh-aw`](https://github.com/githubnext/gh-aw) repository and its companion [`githubnext/agentics`](https://github.com/githubnext/agentics) collection.

Think of this as your guided tour through our agent "factory". We're showcasing the workflows that caught our attention, taught us something new, or just flat-out made our lives easier. Every workflow links to its source Markdown file, so you can peek under the hood and see exactly how it works.

## 🏥 Triage & Summarization Workflows

First up: the agents that help us stay sane when things get busy. These workflows keep us on top of the constant flow of activity:

- **[Issue Triage Agent](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/issue-triage-agent.md)** - Automatically labels and categorizes new issues the moment they're opened
- **[Weekly Issue Summary](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/weekly-issue-summary.md)** - Creates digestible summaries complete with charts and trends (because who has time to read everything?)
- **[Daily Repo Chronicle](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/daily-repo-chronicle.md)** - Narrates the day's activity like a storyteller - seriously, it's kind of delightful

## 🔍 Quality & Hygiene Workflows

These are our diligent caretakers - the agents that spot problems before they become, well, bigger problems:

- **[CI Doctor](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/ci-doctor.md)** - Investigates failed workflows and opens diagnostic issues (it's like having a DevOps specialist on call 24/7)
- **[Schema Consistency Checker](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/schema-consistency-checker.md)** - Detects when schemas, code, and docs drift apart
- **[Breaking Change Checker](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/breaking-change-checker.md)** - Watches for changes that might break things for users

## 📊 Metrics & Analytics Workflows

Data nerds, rejoice! These agents turn raw repository activity into actual insights:

- **[Metrics Collector](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/metrics-collector.md)** - Tracks daily performance across the entire agent ecosystem
- **[Portfolio Analyst](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/portfolio-analyst.md)** - Identifies cost reduction opportunities (because AI isn't free!)
- **[Audit Workflows](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/audit-workflows.md)** - A meta-agent that audits all the other agents' runs - very Inception

## 🔒 Security & Compliance Workflows

These agents are our security guards, keeping watch and enforcing the rules:

- **[Security Compliance](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/security-compliance.md)** - Runs vulnerability campaigns with deadline tracking
- **[Firewall](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/firewall.md)** - Tests network security and validates rules
- **[Daily Secrets Analysis](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/daily-secrets-analysis.md)** - Scans for exposed credentials (yes, it happens)

## 🚀 Operations & Release Workflows

The agents that help us actually ship software:

- **[Release](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/release.md)** - Orchestrates builds, tests, and release note generation
- **[Daily Workflow Updater](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/daily-workflow-updater.md)** - Keeps actions and dependencies current (because dependency updates never stop)

## 🎨 Creative & Culture Workflows

Not everything needs to be serious! These agents remind us that work can be fun:

- **[Poem Bot](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/poem-bot.md)** - Responds to `/poem-bot` commands with creative verses (yes, really)
- **[Daily Team Status](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/daily-team-status.md)** - Shares team mood and status updates
- **[Daily News](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/daily-news.md)** - Curates relevant news for the team

## 💬 Interactive & ChatOps Workflows

These agents respond to commands, providing on-demand assistance whenever you need it:

- **[Q](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/q.md)** - Workflow optimizer that investigates performance and creates PRs
- **[Grumpy Reviewer](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/grumpy-reviewer.md)** - Performs critical code reviews with, well, personality
- **[Workflow Generator](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/workflow-generator.md)** - Creates new workflows from issue requests

## 🔧 Code Quality & Refactoring Workflows

These agents make our codebase cleaner and our developer experience better:

- **[Terminal Stylist](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/terminal-stylist.md)** - Analyzes and improves console output styling (because aesthetics matter!)
- **[Semantic Function Refactor](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/semantic-function-refactor.md)** - Spots refactoring opportunities we might have missed
- **[Repository Quality Improver](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/repository-quality-improver.md)** - Takes a holistic view of code quality and suggests improvements

## 🔬 Testing & Validation Workflows

These agents keep everything running smoothly through continuous testing:

- **[Smoke Tests](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/smoke-codex-firewall.md)** - Validate that engines and firewall are working (running every 12 hours!)
- **[Daily Multi-Device Docs Tester](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/daily-multi-device-docs-tester.md)** - Tests documentation across devices (mobile matters!)
- **[CI Coach](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/ci-coach.md)** - Provides friendly guidance on CI/CD improvements

## 🧰 Tool & Infrastructure Workflows

These agents monitor and analyze the agentic infrastructure itself:

- **[MCP Inspector](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/mcp-inspector.md)** - Validates Model Context Protocol configurations
- **[GitHub MCP Tools Report](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/github-mcp-tools-report.md)** - Analyzes available MCP tools
- **[Agent Performance Analyzer](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/agent-performance-analyzer.md)** - Meta-orchestrator for agent quality

## 🚀 Multi-Phase Improver Workflows

These are some of our most ambitious agents - they tackle big projects over multiple days:

- **[Daily Backlog Burner](https://github.com/githubnext/agentics/blob/main/workflows/daily-backlog-burner.md)** - Systematically works through issues and PRs, one day at a time
- **[Daily Perf Improver](https://github.com/githubnext/agentics/blob/main/workflows/daily-perf-improver.md)** - Three-phase performance optimization (research, setup, implement)
- **[Daily Test Improver](https://github.com/githubnext/agentics/blob/main/workflows/daily-test-improver.md)** - Identifies coverage gaps and implements new tests incrementally
- **[Daily QA](https://github.com/githubnext/agentics/blob/main/workflows/daily-qa.md)** - Continuous quality assurance that never sleeps
- **[Daily Accessibility Review](https://github.com/githubnext/agentics/blob/main/workflows/daily-accessibility-review.md)** - WCAG compliance checking with Playwright
- **[PR Fix](https://github.com/githubnext/agentics/blob/main/workflows/pr-fix.md)** - On-demand slash command to fix failing CI checks (super handy!)

## 📊 Advanced Analytics & ML Workflows

These agents use sophisticated analysis techniques to extract insights:

- **[Copilot Session Insights](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/copilot-session-insights.md)** - Analyzes Copilot agent usage patterns and metrics
- **[Copilot PR NLP Analysis](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/copilot-pr-nlp-analysis.md)** - Natural language processing on PR conversations
- **[Prompt Clustering Analysis](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/prompt-clustering-analysis.md)** - Clusters and categorizes agent prompts using ML
- **[Copilot Agent Analysis](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/copilot-agent-analysis.md)** - Deep analysis of agent behavior patterns

## 🏢 Organization & Cross-Repo Workflows

These agents work at organization scale, across multiple repositories:

- **[Org Health Report](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/org-health-report.md)** - Organization-wide repository health metrics
- **[Stale Repo Identifier](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/stale-repo-identifier.md)** - Identifies inactive repositories
- **[Ubuntu Image Analyzer](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/ubuntu-image-analyzer.md)** - Documents GitHub Actions runner environments

## 📝 Documentation & Content Workflows

These agents maintain high-quality documentation and content:

- **[Glossary Maintainer](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/glossary-maintainer.md)** - Keeps glossary synchronized with codebase
- **[Technical Doc Writer](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/technical-doc-writer.md)** - Generates and updates technical documentation
- **[Slide Deck Maintainer](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/slide-deck-maintainer.md)** - Maintains presentation slide decks
- **[Blog Auditor](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/blog-auditor.md)** - Reviews blog content for quality and accuracy

## 🔗 Issue & PR Management Workflows

These agents enhance issue and pull request workflows:

- **[Issue Arborist](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/issue-arborist.md)** - Links related issues as sub-issues
- **[Issue Monster](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/issue-monster.md)** - Assigns issues to Copilot agents one at a time
- **[Mergefest](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/mergefest.md)** - Automatically merges main branch into PR branches
- **[Sub Issue Closer](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/sub-issue-closer.md)** - Closes completed sub-issues automatically
- **[Issue Template Optimizer](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/issue-template-optimizer.md)** - Improves issue templates based on usage

## 🎯 Campaign & Project Coordination Workflows

These agents manage structured improvement campaigns:

- **[Campaign Generator](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/campaign-generator.md)** - Creates and coordinates multi-step campaigns
- **[Workflow Health Manager](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/workflow-health-manager.md)** - Monitors and maintains workflow health

---

## What's Next?

This collection is just the beginning! As you explore these workflows, you'll start noticing patterns - common structures, shared capabilities, and design choices that keep showing up.

We've learned *so much* from running our collection of automated agentic workflows in practice, and we're excited to share those insights with you. Coming up in this series, we'll dive into the key lessons, design patterns, operational strategies, and security considerations that emerged from this wild experiment.

Stay tuned - we've got plenty more to share!

*More articles in this series coming soon.*
