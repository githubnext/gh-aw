# Introducing the Agent Nursery

**Exploring the landscape of automated agentic workflows in software development**

_The automated use of agentic coding agents opens up a broad landscape of potential applications within existing software repositories. In this project we explored a "nursery" of agentic workflows used within the development of GitHub Agentic Workflows itself and its application in a family of reusable workflows. Here we document this nursery, the patterns we explored, and the lessons for the future of repo-level automated agentic development._

## What is the Agent Nursery?

Imagine a software repository where AI agents work alongside your team - not replacing developers, but handling the repetitive, time-consuming tasks that slow down collaboration and forward progress. The **Agent Nursery** is our exploration of what happens when you take the design philosophy of _let's create a new agentic workflow for that_ as the answer to every opportunity that may present itself! What happens when you _max out on agentic workflows_ - when you make and nurture dozens of specialized AI agentic workflows in a real repository, each designed to solve a specific problem.

Over the course of this research project, we built and operated **over 100 autonomous agentic workflows** within the [`githubnext/gh-aw`](https://github.com/githubnext/gh-aw) repository and its companion [`githubnext/agentics`](https://github.com/githubnext/agentics) collection. These weren't hypothetical demos - they were working agents that:

- Triaged incoming issues and generated weekly summaries
- Diagnosed CI failures and coached teams on improvements
- Performed code reviews with personality and critical analysis
- Maintained documentation and improved test coverage
- Monitored security compliance and validated network isolation
- Analyzed agent performance and optimized workflows
- Executed multi-day projects to reduce technical debt
- Validated infrastructure through continuous smoke testing
- Even wrote poetry to boost team morale

Think of it as a real nursery: each agentic workflow has its own care requirements (triggers and permissions), nutrients (tools and data sources), and growth patterns (what it produces). Some are gentle caregivers that just read and report. Others are more proactive, proposing changes through pull requests. And a few are the gardeners themselves - meta-agents that monitor the health of all the other agents.

We know we're taking things to an extreme here. Most repositories won't need dozens of agents. No one can read all these outputs. But by pushing the boundaries, we learned valuable lessons about what works, what doesn't, and how to design safe, effective agentic workflows that teams can trust.

The information in this report is up to date as of January 2026. Most of it is highly subjective, based on our experiences running these agentic workflows over several months. We welcome feedback and contributions to improve this living document.

### By the Numbers

At the time of writing the nursery comprises:
- **128 workflows** in the main `gh-aw` repository
- **17 curated workflows** in the installable `agentics` collection
- **145 total workflows** demonstrating diverse agent patterns
- **12 core patterns** consolidating all observed behaviors
- **7 AI engines** tested (Copilot, Claude, Codex, and various MCP-based systems)
- **Dozens of MCP servers** integrated for specialized capabilities
- **Multiple trigger types**: schedules, slash commands, reactions, workflow events, issue labels

---

## Why Build a Nursery?

When we started exploring agentic workflows, we faced a fundamental question: **What should repository-level agents actually do?**

The answer wasn't obvious. So instead of trying to build one "perfect" agent, we took a gardener's approach:

1. **Embrace diversity**  -  We created diverse agents for different tasks
2. **Use them and improve them**  -  We ran them continuously in real development workflows
3. **Document what thrives**  -  We identified which patterns worked and which failed
4. **Share the knowledge**  -  We cataloged the common structures that made agents safe and effective

The nursery became both an experiment and a reference collection - a living library of patterns that others could study, adapt, and remix.

---

## Meet the Agentic Workflows

OK, it's time to meet the agentic workflows in the Agent Nursery! Below is a curated selection of the most interesting and useful agentic workflows we developed and ran continuously in the `gh-aw` repository. Each is linked to its source Markdown file so you can explore how it works in detail.

### 🏥 **Triage & Summarization Agentic Workfows**

These agents help teams stay on top of the constant flow of activity:

- **[Issue Triage Agent](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/issue-triage-agent.md)**  -  Automatically labels and categorizes new issues
- **[Weekly Issue Summary](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/weekly-issue-summary.md)**  -  Creates digestible summaries with charts and trends
- **[Daily Repo Chronicle](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/daily-repo-chronicle.md)**  -  Narrates the day's activity like a storyteller

### 🔍 **Quality & Hygiene Agentic Workfows**

These agents act as diligent caretakers, spotting problems before they grow:

- **[CI Doctor](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/ci-doctor.md)**  -  Investigates failed workflows and opens diagnostic issues
- **[Schema Consistency Checker](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/schema-consistency-checker.md)**  -  Detects drift between schemas, code, and docs
- **[Breaking Change Checker](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/breaking-change-checker.md)**  -  Watches for changes that might break users

### 📊 **Metrics & Analytics Agentic Workfows**

These agents turn raw repository activity into insights:

- **[Metrics Collector](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/metrics-collector.md)**  -  Tracks daily performance of the agent ecosystem
- **[Portfolio Analyst](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/portfolio-analyst.md)**  -  Identifies cost reduction opportunities
- **[Audit Workflows](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/audit-workflows.md)**  -  Meta-agent that audits all other agents' runs

### 🔒 **Security & Compliance Agentic Workfows**

These agents guard the perimeter and enforce policies:

- **[Security Compliance](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/security-compliance.md)**  -  Runs vulnerability campaigns with deadline tracking
- **[Firewall](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/firewall.md)**  -  Tests network security and validates rules
- **[Daily Secrets Analysis](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/daily-secrets-analysis.md)**  -  Scans for exposed credentials

### 🚀 **Operations & Release Agentic Workfows**

These agents help with the mechanics of shipping software:

- **[Release](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/release.md)**  -  Orchestrates builds, tests, and release note generation
- **[Daily Workflow Updater](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/daily-workflow-updater.md)**  -  Keeps actions and dependencies current

### 🎨 **Creative & Culture Agentic Workfows**

These agents proved that not everything needs to be serious:

- **[Poem Bot](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/poem-bot.md)**  -  Responds to `/poem-bot` commands with creative verses
- **[Daily Team Status](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/daily-team-status.md)**  -  Shares team mood and status updates
- **[Daily News](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/daily-news.md)**  -  Curates relevant news for the team

### 💬 **Interactive & ChatOps Agentic Workfows**

These agents respond to user commands and provide on-demand assistance:

- **[Q](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/q.md)**  -  Workflow optimizer that investigates performance and creates PRs
- **[Grumpy Reviewer](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/grumpy-reviewer.md)**  -  Performs critical code reviews with personality
- **[Workflow Generator](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/workflow-generator.md)**  -  Creates new workflows from issue requests

### 🔧 **Code Quality & Refactoring Agentic Workfows**

These agents improve code structure and developer experience:

- **[Terminal Stylist](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/terminal-stylist.md)**  -  Analyzes and improves console output styling
- **[Semantic Function Refactor](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/semantic-function-refactor.md)**  -  Identifies refactoring opportunities
- **[Repository Quality Improver](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/repository-quality-improver.md)**  -  Holistic code quality improvements

### 🔬 **Testing & Validation Agentic Workfows**

These agents ensure system reliability through comprehensive testing:

- **[Smoke Tests](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/smoke-codex-firewall.md)**  -  Validate engine and firewall functionality
- **[Daily Multi-Device Docs Tester](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/daily-multi-device-docs-tester.md)**  -  Tests documentation across devices
- **[CI Coach](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/ci-coach.md)**  -  Provides guidance on CI/CD improvements

### 🧰 **Tool & Infrastructure Agentic Workfows**

These agents monitor and analyze the agentic infrastructure itself:

- **[MCP Inspector](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/mcp-inspector.md)**  -  Validates Model Context Protocol configurations
- **[GitHub MCP Tools Report](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/github-mcp-tools-report.md)**  -  Analyzes available MCP tools
- **[Agent Performance Analyzer](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/agent-performance-analyzer.md)**  -  Meta-orchestrator for agent quality

### 🚀 **Multi-Phase Improver Workfows** (from Agentics Pack)

These comprehensive agents perform sustained, multi-day improvement projects:

- **[Daily Backlog Burner](https://github.com/githubnext/agentics/blob/main/workflows/daily-backlog-burner.md)**  -  Systematically works through issues and PRs to close or advance them
- **[Daily Perf Improver](https://github.com/githubnext/agentics/blob/main/workflows/daily-perf-improver.md)**  -  Three-phase performance optimization with build analysis and PRs
- **[Daily Test Improver](https://github.com/githubnext/agentics/blob/main/workflows/daily-test-improver.md)**  -  Identifies coverage gaps and implements new tests over multiple days
- **[Daily QA](https://github.com/githubnext/agentics/blob/main/workflows/daily-qa.md)**  -  Continuous quality assurance with automated testing strategies
- **[Daily Accessibility Review](https://github.com/githubnext/agentics/blob/main/workflows/daily-accessibility-review.md)**  -  WCAG compliance checking with Playwright
- **[PR Fix](https://github.com/githubnext/agentics/blob/main/workflows/pr-fix.md)**  -  On-demand slash command to fix failing CI checks in pull requests

### 📊 **Advanced Analytics & ML Agentic Workfows**

These agents use sophisticated analysis techniques to extract insights:

- **[Copilot Session Insights](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/copilot-session-insights.md)**  -  Analyzes Copilot agent usage patterns and metrics
- **[Copilot PR NLP Analysis](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/copilot-pr-nlp-analysis.md)**  -  Natural language processing on PR conversations
- **[Prompt Clustering Analysis](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/prompt-clustering-analysis.md)**  -  Clusters and categorizes agent prompts using ML
- **[Copilot Agent Analysis](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/copilot-agent-analysis.md)**  -  Deep analysis of agent behavior patterns

### 🏢 **Organization & Cross-Repo Agentic Workfows**

These agents work at organization scale, across multiple repositories:

- **[Org Health Report](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/org-health-report.md)**  -  Organization-wide repository health metrics
- **[Stale Repo Identifier](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/stale-repo-identifier.md)**  -  Identifies inactive repositories
- **[Ubuntu Image Analyzer](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/ubuntu-image-analyzer.md)**  -  Documents GitHub Actions runner environments

### 📝 **Documentation & Content Agentic Workfows**

These agents maintain high-quality documentation and content:

- **[Glossary Maintainer](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/glossary-maintainer.md)**  -  Keeps glossary synchronized with codebase
- **[Technical Doc Writer](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/technical-doc-writer.md)**  -  Generates and updates technical documentation
- **[Slide Deck Maintainer](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/slide-deck-maintainer.md)**  -  Maintains presentation slide decks
- **[Blog Auditor](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/blog-auditor.md)**  -  Reviews blog content for quality and accuracy

### 🔗 **Issue & PR Management Agentic Workfows**

These agents enhance issue and pull request workflows:

- **[Issue Arborist](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/issue-arborist.md)**  -  Links related issues as sub-issues
- **[Mergefest](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/mergefest.md)**  -  Automatically merges main branch into PR branches
- **[Sub Issue Closer](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/sub-issue-closer.md)**  -  Closes completed sub-issues automatically
- **[Issue Template Optimizer](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/issue-template-optimizer.md)**  -  Improves issue templates based on usage

### 🎯 **Campaign & Project Coordination Agentic Workfows**

These agents manage structured improvement campaigns:

- **[Campaign Generator](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/campaign-generator.md)**  -  Creates and coordinates multi-step campaigns
- **[Workflow Health Manager](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/workflow-health-manager.md)**  -  Monitors and maintains workflow health

---

## Imports & Sharing: The Agent Nursery's Secret Weapon

One of the most powerful features that enabled us to scale to 145 agents was the **imports system** - a mechanism for sharing and reusing workflow components across the entire nursery. Rather than duplicating configuration, tool setup, and instructions in every workflow, we created a library of shared components that agents could import on-demand.

### Why Imports Matter at Scale

Tending dozens of agents would be unsustainable without reuse. Imports provided several critical benefits:

**🔄 DRY Principle for Agents**  
When we improved report formatting or updated an MCP server configuration, the change automatically propagated to all workflows that imported it. No need to update 46 workflows individually.

**🧩 Composable Agent Capabilities**  
Workflows could mix and match capabilities by importing different shared components - like combining data visualization, trending analysis, and web search in a single import list.

**🎯 Separation of Concerns**  
Tools configuration, network permissions, data fetching logic, and agent instructions could be maintained independently by different experts, then composed together.

**⚡ Rapid Experimentation**  
Creating a new workflow often meant writing just the agent-specific prompt and importing 3-5 shared components. We could prototype new agents in minutes.

---

## 12 Patterns for Flourishing Agentics

After developing these 145 agents, we identified 12 fundamental design patterns of successful agentic workflows:

### 1. The Read-Only Analyst 🔬
**Observe, analyze, and report without changing anything**

Agentic Workfows that gather data, perform analysis, and publish insights through discussions or assets. No write permissions to code. Safe for continuous operation at any frequency. Includes metrics collectors, audit reports, health monitors, statistical analyzers ([`lockfile-stats`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/lockfile-stats.md), [`org-health`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/org-health-report.md)), and deep research agents ([`scout`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/scout.md), [`archie`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/archie.md)).

*Examples: Weekly metrics, [`audit workflows`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/audit-workflows.md), [`portfolio analyst`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/portfolio-analyst.md), [`session insights`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/copilot-session-insights.md), [`prompt clustering`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/prompt-clustering-analysis.md), [`org health report`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/org-health-report.md)*

### 2. The ChatOps Responder 💬
**On-demand assistance via slash commands**

Agentic Workfows activated by `/command` mentions in issues or PRs. Role-gated for security. Respond with analysis, visualizations, or actions. Includes review agents ([`grumpy-reviewer`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/grumpy-reviewer.md)), generators ([`workflow-generator`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/workflow-generator.md), [`campaign-generator`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/campaign-generator.md)), fixers ([`pr-fix`](https://github.com/githubnext/agentics/blob/main/workflows/pr-fix.md)), optimizers ([`q`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/q.md)), researchers ([`scout`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/scout.md)), and diagram creators ([`archie`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/archie.md)).

*Examples: [`Q`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/q.md), [`grumpy reviewer`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/grumpy-reviewer.md), [`poem bot`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/poem-bot.md), [`mergefest`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/mergefest.md), [`scout`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/scout.md), [`archie`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/archie.md), [`pr-fix`](https://github.com/githubnext/agentics/blob/main/workflows/pr-fix.md)*

### 3. The Continuous Janitor 🧹
**Automated cleanup and maintenance**

Agentic Workfows that propose incremental improvements through PRs. Run on schedules (daily/weekly). Create scoped changes with descriptive labels and commit messages. Human review before merging. Includes dependency updates, formatting fixes, documentation sync, file refactoring ([`daily-file-diet`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/daily-file-diet.md)), instruction maintenance ([`instructions-janitor`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/instructions-janitor.md)), and CI repairs ([`hourly-ci-cleaner`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/hourly-ci-cleaner.md)).

*Examples: [`Daily workflow updater`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/daily-workflow-updater.md), [`glossary maintainer`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/glossary-maintainer.md), [`unbloat-docs`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/unbloat-docs.md), [`daily-file-diet`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/daily-file-diet.md), [`hourly-ci-cleaner`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/hourly-ci-cleaner.md)*

### 4. The Quality Guardian 🛡️
**Continuous validation and compliance enforcement**

Agentic Workfows that validate system integrity through testing, scanning, and compliance checks. Run frequently (hourly/daily) to catch regressions early. Includes smoke tests (engine validation), security scanners ([`static-analysis-report`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/static-analysis-report.md), [`daily-malicious-code-scan`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/daily-malicious-code-scan.md)), accessibility checkers ([`daily-accessibility-review`](https://github.com/githubnext/agentics/blob/main/workflows/daily-accessibility-review.md)), and infrastructure validators ([`firewall`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/firewall.md), [`mcp-inspector`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/mcp-inspector.md)).

*Examples: Smoke tests, [`schema consistency checker`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/schema-consistency-checker.md), [`breaking change checker`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/breaking-change-checker.md), [`security compliance`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/security-compliance.md), [`static-analysis-report`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/static-analysis-report.md), [`mcp-inspector`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/mcp-inspector.md)*

### 5. The Issue & PR Manager 🔗
**Intelligent workflow automation for issues and pull requests**

Agentic Workfows that triage, link, label, close, and coordinate issues and PRs. Includes triagers ([`issue-triage-agent`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/issue-triage-agent.md), [`issue-classifier`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/issue-classifier.md)), linkers ([`issue-arborist`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/issue-arborist.md)), closers ([`sub-issue-closer`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/sub-issue-closer.md)), template optimizers, merge coordinators ([`mergefest`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/mergefest.md)), changeset generators, and auto-assigners. React to events or run on schedules.

*Examples: [`Issue triage agent`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/issue-triage-agent.md), [`issue arborist`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/issue-arborist.md), [`mergefest`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/mergefest.md), [`sub issue closer`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/sub-issue-closer.md), [`changeset generator`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/changeset.md)*

### 6. The Multi-Phase Improver Agent 🔄
**Progressive work across multiple days with human checkpoints**

Agentic Workfows that tackle complex improvements too large for single runs. Three phases: (1) Research and create plan discussion, (2) Infer/setup build infrastructure, (3) Implement changes via PR. Check state each run to determine current phase. Enables ambitious goals like backlog reduction, performance optimization, test coverage improvement.

*Examples: [`Daily backlog burner`](https://github.com/githubnext/agentics/blob/main/workflows/daily-backlog-burner.md), [`daily perf improver`](https://github.com/githubnext/agentics/blob/main/workflows/daily-perf-improver.md), [`daily test improver`](https://github.com/githubnext/agentics/blob/main/workflows/daily-test-improver.md), [`daily QA`](https://github.com/githubnext/agentics/blob/main/workflows/daily-qa.md)*

### 7. The Code Intelligence Agent 🔍
**Semantic analysis and pattern detection**

Agentic Workfows using specialized code analysis tools (Serena, ast-grep) to detect patterns, duplicates, anti-patterns, and refactoring opportunities. Language-specific ([`go-pattern-detector`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/go-pattern-detector.md), [`duplicate-code-detector`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/duplicate-code-detector.md)) or cross-language ([`semantic-function-refactor`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/semantic-function-refactor.md)). Creates issues or PRs with detailed findings. Includes type analysis ([`typist`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/typist.md)) and consistency checking.

*Examples: [`Duplicate code detector`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/duplicate-code-detector.md), [`semantic function refactor`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/semantic-function-refactor.md), [`terminal stylist`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/terminal-stylist.md), [`go-pattern-detector`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/go-pattern-detector.md), [`typist`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/typist.md)*

### 8. The Content & Documentation Agent 📝
**Maintain knowledge artifacts synchronized with code**

Agentic Workfows that keep documentation, glossaries, slide decks, blog posts, and other content fresh. Monitor codebase changes and update corresponding docs. Includes specialized analyzers ([`video-analyzer`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/video-analyzer.md), [`pdf-summary`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/pdf-summary.md), [`ubuntu-image-analyzer`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/ubuntu-image-analyzer.md) for environment docs) and general maintainers. Creates PRs with synchronized content.

*Examples: [`Glossary maintainer`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/glossary-maintainer.md), [`technical doc writer`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/technical-doc-writer.md), [`slide deck maintainer`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/slide-deck-maintainer.md), [`blog auditor`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/blog-auditor.md), [`repo-tree-map`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/repo-tree-map.md), [`ubuntu-image-analyzer`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/ubuntu-image-analyzer.md)*

### 9. The Meta-Agent Optimizer 🎯
**Agentic Workfows that monitor and optimize other agents**

Agentic Workfows that analyze the agent ecosystem itself. Download workflow logs, classify failures, detect missing tools, track performance metrics, identify cost optimization opportunities. Essential for maintaining agent health at scale. Includes performance analyzers, missing tool detectors, and workflow health managers.

*Examples: [`Audit workflows`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/audit-workflows.md), [`agent performance analyzer`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/agent-performance-analyzer.md), [`portfolio analyst`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/portfolio-analyst.md), [`workflow health manager`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/workflow-health-manager.md)*

### 10. The Meta-Agent Orchestrator 🚦
**Orchestrate multi-step workflows via state machines**

Agentic Workfows that coordinate complex workflows through campaigns or task queue patterns. Track state across runs (open/in-progress/completed). Campaign pattern uses labeled issues and project boards. Includes workflow generators and dev monitors ([`dev-hawk`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/dev-hawk.md)).

*Examples: [`Campaign generator`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/campaign-generator.md), [`workflow generator`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/workflow-generator.md), [`dev-hawk`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/dev-hawk.md)*

### 11. The ML & Analytics Agent 🤖
**Advanced insights through machine learning and NLP**

Agentic Workfows that apply clustering, NLP, statistical analysis, or ML techniques to extract patterns from historical data. Fetch sessions, prompts, PRs, or conversations. Generate visualizations and trend reports. Use repo-memory for longitudinal analysis. Discover hidden patterns not obvious from individual interactions.

*Examples: [`Copilot session insights`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/copilot-session-insights.md), [`NLP analysis`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/copilot-pr-nlp-analysis.md), [`prompt clustering`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/prompt-clustering-analysis.md), [`copilot agent analysis`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/copilot-agent-analysis.md)*

### 12. The Security & Moderation Agent 🔒
**Protect repositories from threats and enforce policies**

Agentic Workfows that guard repositories through vulnerability scanning, secret detection, spam filtering, malicious code analysis, and compliance enforcement. Includes AI moderators for comment spam, security scanners for code vulnerabilities, firewall validators, deadline trackers for security campaigns, and automated security fix generators.

*Examples: [`Security compliance`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/security-compliance.md), [`firewall`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/firewall.md), [`daily secrets analysis`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/daily-secrets-analysis.md), [`ai-moderator`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/ai-moderator.md), [`security-fix-pr`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/security-fix-pr.md), [`static-analysis-report`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/static-analysis-report.md), [`daily-malicious-code-scan`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/daily-malicious-code-scan.md)*

---

## Agentic Ops

Beyond the 12 core behavioral patterns that define what agents do, our nursery revealed several strategic design patterns for how to structure and organize agentic workflows at scale, in the context of the GitHub information primitives. These patterns emerged from building and operating workflows and represent battle-tested approaches to common challenges.

### [ChatOps](https://githubnext.github.io/gh-aw/examples/comment-triggered/chatops/): Command-Driven Interactions 💬

These are workflows triggered by slash commands (`/review`, `/deploy`, `/fix`) in issue or PR comments. Creates an interactive conversation interface where team members can invoke powerful AI capabilities with simple commands.

**Example**: [`grumpy-reviewer`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/grumpy-reviewer.md) - Triggered by `/grumpy` on PR comments, performs critical code review with cache memory to avoid duplicate feedback.

### [DailyOps](https://githubnext.github.io/gh-aw/examples/scheduled/dailyops/) 📅

These are workflows that run on weekday schedules to make small, daily progress toward large goals. Instead of overwhelming teams with major changes, work happens automatically in manageable pieces that are easy to review and integrate.

**Example**: [`daily-test-improver`](https://github.com/githubnext/agentics/blob/main/workflows/daily-test-improver.md) - Systematically identifies coverage gaps and implements new tests over multiple days with phased approval.

### [IssueOps](https://githubnext.github.io/gh-aw/examples/issue-pr-events/issueops/): Event-Driven Issue Automation 🎫

These are workflows that transform GitHub issues into automation triggers, automatically analyzing, categorizing, and responding to issues as they're created or updated. Uses safe outputs to ensure secure automated responses.

**Example**: [`issue-triage-agent`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/issue-triage-agent.md) - Automatically labels and categorizes new issues with intelligent analysis.

### [LabelOps](https://githubnext.github.io/gh-aw/examples/issue-pr-events/labelops/) 🏷️

These are workflows that use GitHub labels as triggers, metadata, and state markers. Responds to specific label changes with filtering to activate only for relevant labels while maintaining secure automated responses.

**Example**: Multiple workflows use label filtering for priority escalation and specialized routing based on issue categorization.

### [ProjectOps](https://githubnext.github.io/gh-aw/examples/issue-pr-events/projectops/): AI-Powered Project Board Management 📊

These are workflows that keep GitHub Projects v2 boards up to date using AI to analyze issues/PRs and intelligently decide routing, status, priority, and field values. Safe output architecture ensures security while automating project management.

### [ResearchPlanAssign](https://githubnext.github.io/gh-aw/guides/researchplanassign/): Scaffolded Improvement Strategy 🔬

This is a three-phase strategy that keeps developers in control while leveraging AI agents for systematic code improvements. Provides clear decision points at each phase: Research (investigate), Plan (break down work), Assign (execute).

**Example Implementations from Nursery**:

1. **Duplicate Code Detection → Plan → Refactor**: [`duplicate-code-detector`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/duplicate-code-detector.md) uses Serena MCP for semantic analysis, creates well-scoped issues (max 3 per run), pre-assigns to `@copilot` since fixes are straightforward.

2. **File Size Analysis → Plan → Refactor**: [`daily-file-diet`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/daily-file-diet.md) monitors files exceeding healthy thresholds (1000+ lines), analyzes structure for split boundaries, creates refactoring issue with concrete plan.

3. **Deep Research → Plan → Implementation**: [`scout`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/scout.md) performs deep research using multiple MCPs (Tavily, arXiv, DeepWiki), creates structured summary. Developer uses `/plan` to convert recommendations into issues.

### [MultiRepoOps](https://githubnext.github.io/gh-aw/guides/multirepoops/): Cross-Repository Coordination 🔗

These are workflows that coordinate operations across multiple GitHub repositories using cross-repository safe outputs and secure authentication. Enables feature synchronization, hub-and-spoke tracking, organization-wide enforcement, and upstream/downstream workflows.

### [SideRepoOps](https://githubnext.github.io/gh-aw/guides/siderepoops/): Isolated Automation Infrastructure 🏗️

This pattern provides an easy way to get started with agentic workflows. Run workflows from a separate "side" repository that targets your main codebase, keeping AI-generated issues, comments, and workflow runs isolated from production code.

### [TrialOps](https://githubnext.github.io/gh-aw/guides/trialops/): Safe Workflow Validation 🧪

This is a specialized testing pattern that extends SideRepoOps for validating workflows in temporary trial repositories before production deployment. Creates isolated private repositories where workflows execute and capture safe outputs without affecting actual codebases.

---

## Security Lessons

Security is critical in agentic workflows. See the [security architecture](https://githubnext.github.io/gh-aw/introduction/architecture/) and [security guide](https://githubnext.github.io/gh-aw/guides/security/). Many of the security features of GitHub Agentic Workflows were born from lessons learned while building and running this nursery.

Nurturing a nursery taught us that **safety isn't just about permissions** - it's about designing environments where agents can't accidentally cause harm:

### 🛡️ **Least Privilege, Always**
Start with read-only permissions. Add write permissions only when absolutely necessary and through constrained safe outputs.

### 🚪 **Safe Outputs as the Gateway**
All effectful operations go through safe outputs with built-in limits:
- Maximum items to create
- Expiration times
- "Close older duplicates" logic
- "If no changes" guards

### 👥 **Role-Gated Activation**
Powerful agents (fixers, optimizers) require specific roles to invoke. Not every mention or workflow event should trigger them.

### ⏱️ **Time-Limited Experiments**
Experimental agents include `stop-after: +1mo` to automatically expire, preventing forgotten demos from running indefinitely.

### 🔍 **Explicit Tool Lists**
Agentic Workfows declare exactly which tools they use. No ambient authority. This makes security review straightforward.

### 📋 **Auditable by Default**
Discussions and assets create a natural "agent ledger." You can always trace what an agent did and when.

---

## Conclusions

The Agent Nursery was an ambitious experiment in scaling agentic workflows. After months of nurturing and observing, several key insights emerged about what works, what doesn't, and how to design effective agent ecosystems.

✨ **Diversity Beats Perfection**
No single agent can do everything. A collection of focused agents, each doing one thing well, proved more practical than trying to build a universal assistant.

📊 **Guardrails Enable Innovation**
Counter-intuitively, strict constraints (safe outputs, limited permissions, allowlisted tools) made it *easier* to experiment. We knew the blast radius of any failure.

🔄 **Meta-Agentic Workfows Are Essential**
Agentic Workfows that monitor agents became some of the most valuable. They caught issues early and helped us understand aggregate behavior.

🎭 **Personality Matters**
Agentic Workfows with clear "personalities" (the meticulous auditor, the helpful janitor, the creative poet) were easier for teams to understand and trust.

⚖️ **Cost-Quality Tradeoffs Are Real**
Longer, more thorough analyses cost more but aren't always better. The portfolio analyst helped us identify which agents gave the best value.

🔄 **Multi-Phase Workflows Enable Ambitious Goals**
Breaking complex improvements into 3-phase workflows (research → setup → implement) allowed agents to tackle projects that would be too large for a single run. Each phase builds on the last, with human feedback between phases.

💬 **Slash Commands Create Natural User Interfaces**
ChatOps-style `/command` triggers made agents feel like natural team members. Users could invoke powerful capabilities with simple comments, and role-gating ensured only authorized users could trigger sensitive operations.

🧪 **Heartbeats Build Confidence**
Frequent, lightweight validation tests (every 12 hours) caught regressions quickly. These "heartbeat" agents ensured the infrastructure stayed healthy without manual monitoring.

🔧 **MCP Inspection Is Essential**
As workflows grew to use multiple MCP servers, having agents that could validate and report on tool availability became critical. The MCP inspector pattern prevented cryptic failures from misconfigured tools.

🎯 **Dispatcher Patterns Scale Command Complexity**
Instead of one monolithic agent handling all requests, dispatcher agents could route to specialized sub-agents or commands. This made the system more maintainable and allowed for progressive feature addition.

📿 **Task Queuing is Everywhere**
The task queue pattern provided a simple way to queue and distribute work across multiple workflow runs. Breaking large projects into discrete tasks allowed incremental progress with clear state tracking, recording tasks as issues, discussions or project cards.

🤖 **ML Analysis Reveals Hidden Patterns**
Applying clustering and NLP to agent interactions revealed usage patterns that weren't obvious from individual runs. This meta-analysis helped identify opportunities for consolidation and optimization.

🏢 **Cross-Repo Agentic Workfows Need Special Care**
Organization-level agents required careful permission management and rate limit awareness. They proved valuable for understanding ecosystem health but needed deliberate scoping to avoid overwhelming repositories.

📝 **Documentation Agentic Workfows Bridge Code and Context**
Agentic Workfows that maintained glossaries, technical docs, and slide decks kept documentation synchronized with rapidly evolving codebases. They acted as "knowledge janitors," reducing staleness debt.

### Challenges We Encountered

Not everything was smooth sailing. We faced several challenges that provided valuable lessons:

**Permission Creep**
As agents gained capabilities, there was a temptation to grant broader permissions. We had to constantly audit and prune permissions to maintain least privilege.

**Debugging Complexity**
When agents misbehaved, tracing the root cause through multiple workflow runs and safe outputs was challenging. Improved logging and observability are needed.

**Repository Noise**
Frequent agent runs created a lot of issues, PRs, and comments. We had to implement archival strategies to keep the repository manageable.

**Cost Management**
Running many agents incurred significant costs. The portfolio analyst helped, but ongoing cost monitoring is essential.

**User Trust**
Some team members were hesitant to engage with automated agents. Clear communication about capabilities and limitations helped build trust over time.

---

## Try It Yourself

Want to start your own agent nursery?

1. **Start Small**: Pick one tedious task (issue triage, CI diagnosis, weekly summaries)
2. **Use the Analyst Pattern**: Read-only agents that post to discussions
3. **Nurture Continuously**: Let it run and observe
4. **Iterate**: Refine based on what actually helps your team
5. **Plant More Seeds**: Once one agent works, add complementary ones

The workflows in this nursery are fully remixable. Copy them, adapt them, and make them your own.

---

## Learn More

- **[Agentic Workflows Documentation](https://githubnext.github.io/gh-aw/)**  -  How to write and compile workflows
- **[gh-aw Repository](https://github.com/githubnext/gh-aw)**  -  The nursery's home
- **[Agentics Collection](https://github.com/githubnext/agentics)**  -  Ready-to-install workflows
- **[Continuous AI Project](https://githubnext.com/projects/continuous-ai)**  -  The broader vision

---

## Credits

**Agent Nursery** was a research project by GitHub Next Agentic Workflows contributors and collaborators:

Peli de Halleux, Don Syme, Mara Kiefer, Edward Aftandilian, Krzysztof Cieślak,  Russell Horton, Ben De St Paer‑Gotch, Jiaxiao Zhou

*Part of GitHub Next's exploration of Continuous AI - making AI-enriched automation as routine as CI/CD.*

---

## Appendix: How Agentic Workfows Work

Every agent in the zoo follows the same basic lifecycle:

### 1. **Write** in Natural Language

Agentic Workfows are defined in **Markdown files** using natural language prompts and declarative frontmatter:

```
---
description: Investigates failed CI workflows to identify root causes
on:
  workflow_run:
    workflows: ["CI"]
    types: [completed]
permissions:
  contents: read
  issues: write
tools:
  github:
    toolsets: [issues, pull-requests]
---

When a CI workflow fails, investigate the logs...
```

### 2. **Compile** to Secure Workflows

The natural language workflow compiles to a **locked YAML file** that runs on GitHub Actions:

```
# ci-doctor.lock.yml
name: CI Doctor
on:
  workflow_run:
    workflows: ["CI"]
    types: [completed]
# ... validated, locked execution plan
```

This compilation step ensures:
- ✅ Permissions are minimal and explicit
- ✅ Tools are allowlisted
- ✅ Network access is controlled
- ✅ Safe outputs are properly constrained

### 3. **Run** and Produce Artifacts

Agentic Workfows execute automatically and create **team-visible artifacts**:
- 💬 Issue and PR comments
- 📝 Discussion posts with detailed reports
- 📎 Uploaded assets (charts, logs, data files)
- 🔀 Pull requests for proposed changes

Everything an agent does is auditable - no hidden side effects.

---

## Appendix: Imports and Sharing

To promote code reuse and accelerate development, the nursery extensively utilized the import feature of GitHub Agentic Workflows. This allowed workflows to share common logic, utilities, and configurations, reducing duplication and ensuring consistency across the agent ecosystem.

The nursery organized shared components into two main directories:

#### `.github/workflows/shared/` - Core Capabilities (35+ components)

These components provided fundamental capabilities that many workflows needed:

**Most Popular Shared Components:**
- [**`reporting.md`**](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/shared/reporting.md) (46 imports) - Report formatting guidelines, workflow run references, footer standards
- [**`jqschema.md`**](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/shared/jqschema.md) (17 imports) - JSON querying and schema validation utilities
- [**`python-dataviz.md`**](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/shared/python-dataviz.md) (7 imports) - Python environment with NumPy, Pandas, Matplotlib, Seaborn
- [**`trending-charts-simple.md`**](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/shared/trending-charts-simple.md) (6 imports) - Quick setup for creating trend visualizations
- [**`gh.md`**](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/shared/gh.md) (4 imports) - Safe-input wrapper for GitHub CLI with authentication
- [**`copilot-pr-data-fetch.md`**](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/shared/copilot-pr-data-fetch.md) (4 imports) - Fetch and cache GitHub Copilot PR data

**Specialized Components:**
- [**`charts-with-trending.md`**](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/shared/charts-with-trending.md) - Comprehensive trending with cache-memory integration
- [**`ci-data-analysis.md`**](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/shared/ci-data-analysis.md) - CI workflow analysis utilities
- [**`session-analysis-charts.md`**](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/shared/session-analysis-charts.md) - Copilot session visualization patterns
- [**`keep-it-short.md`**](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/shared/keep-it-short.md) - Prompt guidance for concise responses
- [**`safe-output-app.md`**](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/shared/safe-output-app.md) - Safe output application patterns

#### `.github/workflows/shared/mcp/` - MCP Server Configurations (20+ servers)

These components configured Model Context Protocol servers for specialized capabilities:

**Most Used MCP Servers:**
- [**`gh-aw.md`**](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/shared/mcp/gh-aw.md) (12 imports) - GitHub Agentic Workflows MCP server with `logs` command
- [**`tavily.md`**](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/shared/mcp/tavily.md) (5 imports) - Web search via Tavily API
- [**`markitdown.md`**](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/shared/mcp/markitdown.md) (3 imports) - Document conversion (PDF, Office, images to Markdown)
- [**`ast-grep.md`**](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/shared/mcp/ast-grep.md) (2 imports) - Structural code search and analysis
- [**`brave.md`**](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/shared/mcp/brave.md) (2 imports) - Alternative web search via Brave API
- [**`arxiv.md`**](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/shared/mcp/arxiv.md) (2 imports) - Academic paper research
- [**`notion.md`**](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/shared/mcp/notion.md) (2 imports) - Notion workspace integration

**Infrastructure & Analysis:**
- [**`jupyter.md`**](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/shared/mcp/jupyter.md) - Jupyter notebook environment with Docker services
- [**`skillz.md`**](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/shared/mcp/skillz.md) - Dynamic skill loading from `.github/skills/` directory
- [**`context7.md`**](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/shared/mcp/context7.md), [**`deepwiki.md`**](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/shared/mcp/deepwiki.md), [**`microsoft-docs.md`**](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/shared/mcp/microsoft-docs.md) - Specialized search and documentation
- [**`slack.md`**](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/shared/mcp/slack.md), [**`sentry.md`**](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/shared/mcp/sentry.md), [**`datadog.md`**](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/shared/mcp/datadog.md) - External service integrations

### Imports by the Numbers

- **84 workflows** (65% of zoo) use the imports feature
- **46 workflows** import [`reporting.md`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/shared/reporting.md) (most popular component)
- **17 workflows** import [`jqschema.md`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/shared/jqschema.md) (JSON utilities)
- **12 workflows** import [`mcp/gh-aw.md`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/shared/mcp/gh-aw.md) (meta-analysis server)
- **35+ shared components** in `.github/workflows/shared/`
- **20+ MCP server configs** in `.github/workflows/shared/mcp/`
- **Average 2-3 imports** per workflow (some have 8+!)

## Appendix: Complete Workflow Inventory

### Workflows from `.github/workflows/` (128 total)

**Incorporated into 12 consolidated patterns (95 workflows):**

*Pattern 1 - Read-Only Analyst:*
[`agent-performance-analyzer`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/agent-performance-analyzer.md), [`audit-workflows`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/audit-workflows.md), [`copilot-agent-analysis`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/copilot-agent-analysis.md), [`copilot-session-insights`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/copilot-session-insights.md), [`daily-copilot-token-report`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/daily-copilot-token-report.md), [`daily-performance-summary`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/daily-performance-summary.md), [`lockfile-stats`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/lockfile-stats.md), [`metrics-collector`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/metrics-collector.md), [`org-health-report`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/org-health-report.md), [`portfolio-analyst`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/portfolio-analyst.md), [`prompt-clustering-analysis`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/prompt-clustering-analysis.md), [`repo-tree-map`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/repo-tree-map.md), [`scout`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/scout.md), [`static-analysis-report`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/static-analysis-report.md), [`ubuntu-image-analyzer`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/ubuntu-image-analyzer.md)

*Pattern 2 - ChatOps Responder:*
[`archie`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/archie.md), [`campaign-generator`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/campaign-generator.md), [`grumpy-reviewer`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/grumpy-reviewer.md), [`mergefest`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/mergefest.md), [`poem-bot`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/poem-bot.md), [`pr-nitpick-reviewer`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/pr-nitpick-reviewer.md), [`q`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/q.md), [`workflow-generator`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/workflow-generator.md)

*Pattern 3 - Continuous Janitor:*
[`changeset`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/changeset.md), [`daily-doc-updater`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/daily-doc-updater.md), [`daily-file-diet`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/daily-file-diet.md), [`daily-workflow-updater`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/daily-workflow-updater.md), [`developer-docs-consolidator`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/developer-docs-consolidator.md), [`glossary-maintainer`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/glossary-maintainer.md), [`hourly-ci-cleaner`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/hourly-ci-cleaner.md), [`instructions-janitor`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/instructions-janitor.md), [`slide-deck-maintainer`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/slide-deck-maintainer.md), [`technical-doc-writer`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/technical-doc-writer.md), [`unbloat-docs`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/unbloat-docs.md)

*Pattern 4 - Quality Guardian:*
[`breaking-change-checker`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/breaking-change-checker.md), [`ci-coach`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/ci-coach.md), [`daily-multi-device-docs-tester`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/daily-multi-device-docs-tester.md), [`firewall-escape`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/firewall-escape.md), [`firewall`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/firewall.md), [`github-mcp-structural-analysis`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/github-mcp-structural-analysis.md), [`mcp-inspector`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/mcp-inspector.md), [`schema-consistency-checker`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/schema-consistency-checker.md), [`smoke-claude`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/smoke-claude.md), [`smoke-codex-firewall`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/smoke-codex-firewall.md), [`smoke-codex`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/smoke-codex.md), [`smoke-copilot-no-firewall`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/smoke-copilot-no-firewall.md), [`smoke-copilot-playwright`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/smoke-copilot-playwright.md), [`smoke-copilot-safe-inputs`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/smoke-copilot-safe-inputs.md), [`smoke-copilot`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/smoke-copilot.md), [`smoke-detector`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/smoke-detector.md), [`smoke-srt-custom-config`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/smoke-srt-custom-config.md), [`smoke-srt`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/smoke-srt.md)

*Pattern 5 - Issue & PR Manager:*
[`issue-arborist`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/issue-arborist.md), [`issue-classifier`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/issue-classifier.md), [`issue-template-optimizer`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/issue-template-optimizer.md), [`issue-triage-agent`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/issue-triage-agent.md), [`sub-issue-closer`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/sub-issue-closer.md)

*Pattern 7 - Code Intelligence:*
[`cli-consistency-checker`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/cli-consistency-checker.md), [`duplicate-code-detector`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/duplicate-code-detector.md), [`go-pattern-detector`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/go-pattern-detector.md), [`repository-quality-improver`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/repository-quality-improver.md), [`semantic-function-refactor`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/semantic-function-refactor.md), [`terminal-stylist`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/terminal-stylist.md), [`typist`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/typist.md)

*Pattern 9 - Content & Documentation:*
[`artifacts-summary`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/artifacts-summary.md), [`blog-auditor`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/blog-auditor.md), [`copilot-pr-merged-report`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/copilot-pr-merged-report.md), [`notion-issue-summary`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/notion-issue-summary.md), [`pdf-summary`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/pdf-summary.md), [`video-analyzer`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/video-analyzer.md), [`weekly-issue-summary`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/weekly-issue-summary.md)

*Pattern 10 - Event-Driven Coordinator:*
[`dev-hawk`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/dev-hawk.md), [`docs-quality-maintenance-project67.campaign`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/docs-quality-maintenance-project67.campaign.md), [`docs-quality-maintenance-project67.campaign.g`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/docs-quality-maintenance-project67.campaign.g.md), [`file-size-reduction-project71.campaign`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/file-size-reduction-project71.campaign.md), [`file-size-reduction-project71.campaign.g`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/file-size-reduction-project71.campaign.g.md), [`workflow-health-manager`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/workflow-health-manager.md)

*Pattern 11 - ML & Analytics:*
[`copilot-pr-nlp-analysis`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/copilot-pr-nlp-analysis.md), [`copilot-pr-prompt-analysis`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/copilot-pr-prompt-analysis.md)

*Pattern 12 - Security & Moderation:*
[`ai-moderator`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/ai-moderator.md), [`daily-malicious-code-scan`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/daily-malicious-code-scan.md), [`daily-secrets-analysis`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/daily-secrets-analysis.md), [`security-compliance`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/security-compliance.md), [`security-fix-pr`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/security-fix-pr.md), [`safe-output-health`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/safe-output-health.md)

**Specialized/experimental variants (33):**
[`brave`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/brave.md), [`ci-doctor`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/ci-doctor.md), [`cli-version-checker`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/cli-version-checker.md), [`cloclo`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/cloclo.md), [`commit-changes-analyzer`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/commit-changes-analyzer.md), [`craft`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/craft.md), [`daily-assign-issue-to-user`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/daily-assign-issue-to-user.md), [`daily-choice-test`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/daily-choice-test.md), [`daily-cli-performance`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/daily-cli-performance.md), [`daily-code-metrics`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/daily-code-metrics.md), [`daily-fact`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/daily-fact.md), [`daily-firewall-report`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/daily-firewall-report.md), [`daily-issues-report`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/daily-issues-report.md), [`daily-news`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/daily-news.md), [`daily-repo-chronicle`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/daily-repo-chronicle.md), [`daily-team-status`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/daily-team-status.md), [`deep-report`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/deep-report.md), [`dependabot-go-checker`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/dependabot-go-checker.md), [`dev`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/dev.md), [`dictation-prompt`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/dictation-prompt.md), [`docs-noob-tester`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/docs-noob-tester.md), [`example-custom-error-patterns`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/example-custom-error-patterns.md), [`example-permissions-warning`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/example-permissions-warning.md), [`example-workflow-analyzer`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/example-workflow-analyzer.md), [`github-mcp-tools-report`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/github-mcp-tools-report.md), [`go-fan`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/go-fan.md), [`go-logger`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/go-logger.md), [`issue-monster`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/issue-monster.md), [`jsweep`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/jsweep.md), [`layout-spec-maintainer`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/layout-spec-maintainer.md), [`plan`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/plan.md), [`playground-org-project-update-issue`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/playground-org-project-update-issue.md), [`playground-snapshots-refresh`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/playground-snapshots-refresh.md), [`python-data-charts`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/python-data-charts.md), [`release`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/release.md), [`research`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/research.md), [`stale-repo-identifier`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/stale-repo-identifier.md), [`super-linter`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/super-linter.md), [`tidy`](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/tidy.md)

### Workflows from `/home/dsyme/agentics/workflows` (17 total)

**All incorporated into patterns:**
- ci-doctor → Pattern 4 (Quality Guardian)
- daily-accessibility-review → Patterns 4 & 6
- daily-backlog-burner → Pattern 6 (Multi-Phase)
- daily-dependency-updates → Pattern 3 (Janitor)
- daily-perf-improver → Pattern 6 (Multi-Phase)
- daily-plan → Pattern 10 (Coordinator)
- daily-progress → Pattern 1 (Analyst)
- daily-qa → Pattern 6 (Multi-Phase)
- daily-team-status → Pattern 1 (Analyst)
- daily-test-improver → Pattern 6 (Multi-Phase)
- issue-triage → Pattern 5 (Issue & PR Manager)
- plan → Pattern 10 (Coordinator)
- pr-fix → Patterns 2 & 3
- q → Patterns 2 & 8
- repo-ask → Pattern 2 (ChatOps)
- update-docs → Pattern 9 (Documentation)
- weekly-research → Pattern 1 (Analyst)

### Coverage Summary

- **Total workflows:** 145
- **Incorporated into 12 patterns:** 112 (77%)
- **Specialized/experimental:** 33 (23%)

The 12 consolidated patterns capture the essential behaviors of all 145 workflows. The 33 specialized workflows are primarily language-specific utilities, experimental demos, or variations that don't represent fundamentally distinct patterns.

