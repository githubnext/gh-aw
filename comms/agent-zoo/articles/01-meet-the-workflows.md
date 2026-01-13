# Meet the Agentic Workflows in Peli's Agent Factory

**A curated tour of the most interesting agents in the factory**

[← Back to Peli's Agent Factory](../index.md) | [Next: 12 Lessons →](02-twelve-lessons.md)

---

Welcome to Peli's Agent Factory! Over the course of this research project, we built and operated over 145 autonomous agentic workflows. These weren't hypothetical demos - they were working agents that handled real development tasks in the [`githubnext/gh-aw`](https://github.com/githubnext/gh-aw) repository and its companion [`githubnext/agentics`](https://github.com/githubnext/agentics) collection.

Below is a curated selection of the most interesting and useful agentic workflows we developed and ran continuously. Each is linked to its source Markdown file so you can explore how it works in detail.

## 🏥 Triage & Summarization Workflows

These agents help teams stay on top of the constant flow of activity:

- **[Issue Triage Agent](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/issue-triage-agent.md)** - Automatically labels and categorizes new issues
- **[Weekly Issue Summary](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/weekly-issue-summary.md)** - Creates digestible summaries with charts and trends
- **[Daily Repo Chronicle](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/daily-repo-chronicle.md)** - Narrates the day's activity like a storyteller

## 🔍 Quality & Hygiene Workflows

These agents act as diligent caretakers, spotting problems before they grow:

- **[CI Doctor](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/ci-doctor.md)** - Investigates failed workflows and opens diagnostic issues
- **[Schema Consistency Checker](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/schema-consistency-checker.md)** - Detects drift between schemas, code, and docs
- **[Breaking Change Checker](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/breaking-change-checker.md)** - Watches for changes that might break users

## 📊 Metrics & Analytics Workflows

These agents turn raw repository activity into insights:

- **[Metrics Collector](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/metrics-collector.md)** - Tracks daily performance of the agent ecosystem
- **[Portfolio Analyst](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/portfolio-analyst.md)** - Identifies cost reduction opportunities
- **[Audit Workflows](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/audit-workflows.md)** - Meta-agent that audits all other agents' runs

## 🔒 Security & Compliance Workflows

These agents guard the perimeter and enforce policies:

- **[Security Compliance](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/security-compliance.md)** - Runs vulnerability campaigns with deadline tracking
- **[Firewall](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/firewall.md)** - Tests network security and validates rules
- **[Daily Secrets Analysis](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/daily-secrets-analysis.md)** - Scans for exposed credentials

## 🚀 Operations & Release Workflows

These agents help with the mechanics of shipping software:

- **[Release](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/release.md)** - Orchestrates builds, tests, and release note generation
- **[Daily Workflow Updater](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/daily-workflow-updater.md)** - Keeps actions and dependencies current

## 🎨 Creative & Culture Workflows

These agents proved that not everything needs to be serious:

- **[Poem Bot](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/poem-bot.md)** - Responds to `/poem-bot` commands with creative verses
- **[Daily Team Status](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/daily-team-status.md)** - Shares team mood and status updates
- **[Daily News](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/daily-news.md)** - Curates relevant news for the team

## 💬 Interactive & ChatOps Workflows

These agents respond to user commands and provide on-demand assistance:

- **[Q](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/q.md)** - Workflow optimizer that investigates performance and creates PRs
- **[Grumpy Reviewer](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/grumpy-reviewer.md)** - Performs critical code reviews with personality
- **[Workflow Generator](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/workflow-generator.md)** - Creates new workflows from issue requests

## 🔧 Code Quality & Refactoring Workflows

These agents improve code structure and developer experience:

- **[Terminal Stylist](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/terminal-stylist.md)** - Analyzes and improves console output styling
- **[Semantic Function Refactor](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/semantic-function-refactor.md)** - Identifies refactoring opportunities
- **[Repository Quality Improver](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/repository-quality-improver.md)** - Holistic code quality improvements

## 🔬 Testing & Validation Workflows

These agents ensure system reliability through comprehensive testing:

- **[Smoke Tests](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/smoke-codex-firewall.md)** - Validate engine and firewall functionality
- **[Daily Multi-Device Docs Tester](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/daily-multi-device-docs-tester.md)** - Tests documentation across devices
- **[CI Coach](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/ci-coach.md)** - Provides guidance on CI/CD improvements

## 🧰 Tool & Infrastructure Workflows

These agents monitor and analyze the agentic infrastructure itself:

- **[MCP Inspector](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/mcp-inspector.md)** - Validates Model Context Protocol configurations
- **[GitHub MCP Tools Report](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/github-mcp-tools-report.md)** - Analyzes available MCP tools
- **[Agent Performance Analyzer](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/agent-performance-analyzer.md)** - Meta-orchestrator for agent quality

## 🚀 Multi-Phase Improver Workflows

These comprehensive agents perform sustained, multi-day improvement projects:

- **[Daily Backlog Burner](https://github.com/githubnext/agentics/blob/main/workflows/daily-backlog-burner.md)** - Systematically works through issues and PRs to close or advance them
- **[Daily Perf Improver](https://github.com/githubnext/agentics/blob/main/workflows/daily-perf-improver.md)** - Three-phase performance optimization with build analysis and PRs
- **[Daily Test Improver](https://github.com/githubnext/agentics/blob/main/workflows/daily-test-improver.md)** - Identifies coverage gaps and implements new tests over multiple days
- **[Daily QA](https://github.com/githubnext/agentics/blob/main/workflows/daily-qa.md)** - Continuous quality assurance with automated testing strategies
- **[Daily Accessibility Review](https://github.com/githubnext/agentics/blob/main/workflows/daily-accessibility-review.md)** - WCAG compliance checking with Playwright
- **[PR Fix](https://github.com/githubnext/agentics/blob/main/workflows/pr-fix.md)** - On-demand slash command to fix failing CI checks in pull requests

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

This collection represents the visible tip of a much larger experimental space. As you explore these workflows, you'll notice patterns emerging - common structures, shared capabilities, and recurring design choices.

In the next article, we'll distill what we learned from running all these agents in production.

[← Back to Peli's Agent Factory](../index.md) | [Next: 12 Lessons →](02-twelve-lessons.md)
