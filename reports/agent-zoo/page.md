# Introducing the Agent Zoo

**Exploring the landscape of automated agentic workflows in software development**

_The automated use of agentic coding agents opens up a broad landscape of potential applications within existing software repositories. In this project we explored a "zoo" of agentic workflows used within the development of GitHub Agentic Workflows itself and its application in a family of reusable workflows. Here we document this zoo, the patterns we explored, and the lessons for the future of repo-level automated agentic development._

## What is the Agent Zoo?

Imagine a software repository where AI agents work alongside your team - not replacing developers, but handling the repetitive, time-consuming tasks that slow down collaboration and forward progress. The **Agent Zoo** is our exploration of what happens when you take the design philosophy of _let's create a new agentic workflow for that_ as the answer to every opportunity that may present itself! What happens when you _max out on agentic workflows_ - when you let dozens of specialized AI agentic workflows loose in a real repository, each designed to solve a specific problem.

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

Think of it as a real zoo: each "species" of agent has its own habitat (triggers and permissions), diet (tools and data sources), and behavior patterns (what it produces). Some are friendly herbivores that just read and report. Others are more assertive, proposing changes through pull requests. And a few are the zookeepers themselves - meta-agents that monitor the health of all the other agents.

We know we're taking things to an extreme here. Most repositories won't need dozens of agents. No one can read all these outputs. But by pushing the boundaries, we learned valuable lessons about what works, what doesn't, and how to design safe, effective agentic workflows that teams can trust.

The information in this report is up to date as of January 2026. Most of it is highly subjective, based on our experiences running these agentic workflows over several months. We welcome feedback and contributions to improve this living document.

### By the Numbers

At the time of writing zoo comprises:
- **128 workflows** in the main `gh-aw` repository
- **17 curated workflows** in the installable `agentics` collection
- **145 total workflows** demonstrating diverse agent patterns
- **12 core patterns** consolidating all observed behaviors
- **7 AI engines** tested (Copilot, Claude, Codex, and various MCP-based systems)
- **Dozens of MCP servers** integrated for specialized capabilities
- **Multiple trigger types**: schedules, slash commands, reactions, workflow events, issue labels

---

## Why Build a Zoo?

When we started exploring agentic workflows, we faced a fundamental question: **What should repository-level agents actually do?**

The answer wasn't obvious. So instead of trying to build one "perfect" agent, we took a naturalist's approach:

1. **Let many species evolve**  -  We created diverse agents for different tasks
2. **Observe them in the wild**  -  We ran them continuously in real development workflows
3. **Document survival strategies**  -  We identified which patterns worked and which failed
4. **Extract reusable DNA**  -  We cataloged the common structures that made agents safe and effective

The zoo became both an experiment and a reference collection - a living library of patterns that others could study, adapt, and remix.

---

## Meet the Inhabitants

OK, it's time to meet the residents of the Agent Zoo! Below is a curated selection of the most interesting and useful agentic workflows we developed and ran continuously in the `gh-aw` repository. Each agent is linked to its source Markdown file so you can explore how it works in detail.

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
- **[Spec-Kit Dispatcher](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/speckit-dispatcher.md)**  -  Routes specification-driven development commands

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

### 🚀 **Multi-Phase Project Agentic Workfows** (from Agentics Pack)

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
- **[Beads Worker](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/beads-worker.md)**  -  Executes ready beads (task units) from beads-equipped repos
- **[Workflow Health Manager](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/workflow-health-manager.md)**  -  Monitors and maintains workflow health

---

## How Agentic Workfows Work

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

## The 12 Core Patterns

After running 145 agents continuously, we identified 12 fundamental patterns that capture the essential DNA of successful agentic workflows:

### 1. The Read-Only Analyst 🔬
**Observe, analyze, and report without changing anything**

Agentic Workfows that gather data, perform analysis, and publish insights through discussions or assets. No write permissions to code. Safe for continuous operation at any frequency. Includes metrics collectors, audit reports, health monitors, statistical analyzers (lockfile-stats, org-health), and deep research agents (scout, archie).

*Examples: Weekly metrics, audit workflows, portfolio analyst, session insights, prompt clustering, org health report*

### 2. The ChatOps Responder 💬
**On-demand assistance via slash commands**

Agentic Workfows activated by `/command` mentions in issues or PRs. Role-gated for security. Respond with analysis, visualizations, or actions. Includes review agents (grumpy-reviewer), generators (workflow-generator, campaign-generator), fixers (pr-fix), optimizers (q), researchers (scout), and diagram creators (archie).

*Examples: Q, grumpy reviewer, poem bot, mergefest, scout, archie, pr-fix*

### 3. The Continuous Janitor 🧹
**Automated cleanup and maintenance**

Agentic Workfows that propose incremental improvements through PRs. Run on schedules (daily/weekly). Create scoped changes with descriptive labels and commit messages. Human review before merging. Includes dependency updates, formatting fixes, documentation sync, file refactoring (daily-file-diet), instruction maintenance (instructions-janitor), and CI repairs (hourly-ci-cleaner).

*Examples: Daily workflow updater, glossary maintainer, unbloat-docs, daily-file-diet, hourly-ci-cleaner*

### 4. The Quality Guardian 🛡️
**Continuous validation and compliance enforcement**

Agentic Workfows that validate system integrity through testing, scanning, and compliance checks. Run frequently (hourly/daily) to catch regressions early. Includes smoke tests (engine validation), security scanners (static-analysis-report, daily-malicious-code-scan), accessibility checkers (daily-accessibility-review), and infrastructure validators (firewall, mcp-inspector).

*Examples: Smoke tests, schema consistency checker, breaking change checker, security compliance, static-analysis-report, mcp-inspector*

### 5. The Issue & PR Manager 🔗
**Intelligent workflow automation for issues and pull requests**

Agentic Workfows that triage, link, label, close, and coordinate issues and PRs. Includes triagers (issue-triage-agent, issue-classifier), linkers (issue-arborist), closers (sub-issue-closer), template optimizers, merge coordinators (mergefest), changeset generators, and auto-assigners. React to events or run on schedules.

*Examples: Issue triage agent, issue arborist, mergefest, sub-issue closer, changeset generator*

### 6. The Multi-Phase Project Agent 🔄
**Progressive work across multiple days with human checkpoints**

Agentic Workfows that tackle complex improvements too large for single runs. Three phases: (1) Research and create plan discussion, (2) Infer/setup build infrastructure, (3) Implement changes via PR. Check state each run to determine current phase. Enables ambitious goals like backlog reduction, performance optimization, test coverage improvement.

*Examples: Daily backlog burner, daily perf improver, daily test improver, daily QA*

### 7. The Code Intelligence Agent 🔍
**Semantic analysis and pattern detection**

Agentic Workfows using specialized code analysis tools (Serena, ast-grep) to detect patterns, duplicates, anti-patterns, and refactoring opportunities. Language-specific (go-pattern-detector, duplicate-code-detector) or cross-language (semantic-function-refactor). Creates issues or PRs with detailed findings. Includes type analysis (typist) and consistency checking.

*Examples: Duplicate code detector, semantic function refactor, terminal stylist, go-pattern-detector, typist*

### 8. The Meta-Orchestrator 🎯
**Agentic Workfows that monitor and optimize other agents**

Agentic Workfows that analyze the agent ecosystem itself. Download workflow logs, classify failures, detect missing tools, track performance metrics, identify cost optimization opportunities. Essential for maintaining agent health at scale. Includes performance analyzers, missing tool detectors, and workflow health managers.

*Examples: Audit workflows, agent performance analyzer, portfolio analyst, workflow health manager*

### 9. The Content & Documentation Agent 📝
**Maintain knowledge artifacts synchronized with code**

Agentic Workfows that keep documentation, glossaries, slide decks, blog posts, and other content fresh. Monitor codebase changes and update corresponding docs. Includes specialized analyzers (video-analyzer, pdf-summary, ubuntu-image-analyzer for environment docs) and general maintainers. Creates PRs with synchronized content.

*Examples: Glossary maintainer, technical doc writer, slide deck maintainer, blog auditor, repo-tree-map, ubuntu-image-analyzer*

### 10. The Event-Driven Coordinator 🚦
**Orchestrate multi-step workflows via state machines**

Agentic Workfows that coordinate complex workflows through campaigns, beads, or spec-kit patterns. Track state across runs (open/in-progress/completed). Campaign pattern uses labeled issues and project boards. Beads pattern uses hourly workers for queued tasks. Spec-kit dispatches sub-commands for specification-driven development. Includes workflow generators and dev monitors (dev-hawk).

*Examples: Campaign generator, beads worker, speckit-dispatcher, workflow generator, dev-hawk*

### 11. The ML & Analytics Agent 🤖
**Advanced insights through machine learning and NLP**

Agentic Workfows that apply clustering, NLP, statistical analysis, or ML techniques to extract patterns from historical data. Fetch sessions, prompts, PRs, or conversations. Generate visualizations and trend reports. Use repo-memory for longitudinal analysis. Discover hidden patterns not obvious from individual interactions.

*Examples: Copilot session insights, NLP analysis, prompt clustering, copilot agent analysis*

### 12. The Security & Moderation Agent 🔒
**Protect repositories from threats and enforce policies**

Agentic Workfows that guard repositories through vulnerability scanning, secret detection, spam filtering, malicious code analysis, and compliance enforcement. Includes AI moderators for comment spam, security scanners for code vulnerabilities, firewall validators, deadline trackers for security campaigns, and automated security fix generators.

*Examples: Security compliance, firewall, daily secrets analysis, ai-moderator, security-fix-pr, static-analysis-report, daily-malicious-code-scan*
Running a zoo taught us that **safety isn't just about permissions** - it's about designing environments where agents can't accidentally cause harm:

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

## What We Learned

### ✨ **Diversity Beats Perfection**
No single agent can do everything. A collection of focused agents, each doing one thing well, proved more practical than trying to build a universal assistant.

### 📊 **Guardrails Enable Innovation**
Counter-intuitively, strict constraints (safe outputs, limited permissions, allowlisted tools) made it *easier* to experiment. We knew the blast radius of any failure.

### 🔄 **Meta-Agentic Workfows Are Essential**
Agentic Workfows that monitor agents became some of the most valuable. They caught issues early and helped us understand aggregate behavior.

### 🎭 **Personality Matters**
Agentic Workfows with clear "personalities" (the meticulous auditor, the helpful janitor, the creative poet) were easier for teams to understand and trust.

### ⚖️ **Cost-Quality Tradeoffs Are Real**
Longer, more thorough analyses cost more but aren't always better. The portfolio analyst helped us identify which agents gave the best value.

### 🔄 **Multi-Phase Workflows Enable Ambitious Goals**
Breaking complex improvements into 3-phase workflows (research → setup → implement) allowed agents to tackle projects that would be too large for a single run. Each phase builds on the last, with human feedback between phases.

### 💬 **Slash Commands Create Natural User Interfaces**
ChatOps-style `/command` triggers made agents feel like natural team members. Users could invoke powerful capabilities with simple comments, and role-gating ensured only authorized users could trigger sensitive operations.

### 🧪 **Smoke Tests Build Confidence**
Frequent, lightweight validation tests (every 12 hours) caught regressions quickly. These "heartbeat" agents ensured the infrastructure stayed healthy without manual monitoring.

### 🔧 **MCP Inspection Is Essential**
As workflows grew to use multiple MCP servers, having agents that could validate and report on tool availability became critical. The MCP inspector pattern prevented cryptic failures from misconfigured tools.

### 🎯 **Dispatcher Patterns Scale Command Complexity**
Instead of one monolithic agent handling all requests, dispatcher agents could route to specialized sub-agents or commands. This made the system more maintainable and allowed for progressive feature addition.

### 📿 **Beads Enable Task Queuing**
The beads pattern provided a simple way to queue and distribute work across multiple workflow runs. Breaking large projects into discrete beads allowed incremental progress with clear state tracking.

### 🤖 **ML Analysis Reveals Hidden Patterns**
Applying clustering and NLP to agent interactions revealed usage patterns that weren't obvious from individual runs. This meta-analysis helped identify opportunities for consolidation and optimization.

### 🏢 **Cross-Repo Agentic Workfows Need Special Care**
Organization-level agents required careful permission management and rate limit awareness. They proved valuable for understanding ecosystem health but needed deliberate scoping to avoid overwhelming repositories.

### 📝 **Documentation Agentic Workfows Bridge Code and Context**
Agentic Workfows that maintained glossaries, technical docs, and slide decks kept documentation synchronized with rapidly evolving codebases. They acted as "knowledge janitors," reducing staleness debt.

---

## Research Artifacts

This project produced several resources:

- **[100+ Working Workflows](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/)**  -  The actual zoo, ready to explore
- **[Agentics Pack](https://github.com/githubnext/agentics)**  -  Curated collection of installable workflows for common development tasks:
  - **ci-doctor.md**  -  Diagnoses and reports on CI failures
  - **daily-backlog-burner.md**  -  Systematically works through issue/PR backlog
  - **daily-perf-improver.md**  -  Three-phase performance optimization workflow
  - **daily-test-improver.md**  -  Identifies and implements test coverage improvements
  - **daily-qa.md**  -  Continuous quality assurance automation
  - **issue-triage.md**  -  Automatically categorizes and labels issues
  - **q.md**  -  Interactive workflow optimizer and investigator
  - **repo-ask.md**  -  Natural language repository queries
  - **update-docs.md**  -  Documentation maintenance agent
  - **weekly-research.md**  -  Research summaries and insights
- **[Pattern Catalog](/reports/agent-zoo/patterns.md)**  -  Documented scaffolds and anti-patterns
- **[Research Plan](/reports/agent-zoo/plan.md)**  -  Detailed methodology and evaluation framework

---

## Try It Yourself

Want to start your own agent zoo?

1. **Start Small**: Pick one tedious task (issue triage, CI diagnosis, weekly summaries)
2. **Use the Analyst Pattern**: Read-only agents that post to discussions
3. **Run Continuously**: Let it run for a week and observe
4. **Iterate**: Refine based on what actually helps your team
5. **Add More Species**: Once one agent works, add complementary ones

The workflows in this zoo are fully remixable. Copy them, adapt them, and make them your own.

---

## Learn More

- **[Agentic Workflows Documentation](https://githubnext.github.io/gh-aw/)**  -  How to write and compile workflows
- **[gh-aw Repository](https://github.com/githubnext/gh-aw)**  -  The zoo's home
- **[Agentics Collection](https://github.com/githubnext/agentics)**  -  Ready-to-install workflows
- **[Continuous AI Project](https://githubnext.com/projects/continuous-ai)**  -  The broader vision

---

## Credits

**Agent Zoo** was a research project by GitHub Next Agentic Workflows contributors and collaborators:

Peli de Halleux, Don Syme, Mara Kiefer, Edward Aftandilian, Krzysztof Cieślak,  Russell Horton, Ben De St Paer‑Gotch, Jiaxiao Zhou

*Part of GitHub Next's exploration of Continuous AI - making AI-enriched automation as routine as CI/CD.*
---

## Appendix: Complete Workflow Inventory

### Workflows from `.github/workflows/` (128 total)

**Incorporated into 12 consolidated patterns (95 workflows):**

*Pattern 1 - Read-Only Analyst:*
agent-performance-analyzer, audit-workflows, copilot-agent-analysis, copilot-session-insights, daily-copilot-token-report, daily-performance-summary, lockfile-stats, metrics-collector, org-health-report, portfolio-analyst, prompt-clustering-analysis, repo-tree-map, scout, static-analysis-report, ubuntu-image-analyzer

*Pattern 2 - ChatOps Responder:*
archie, campaign-generator, grumpy-reviewer, mergefest, poem-bot, pr-nitpick-reviewer, q, speckit-dispatcher, workflow-generator

*Pattern 3 - Continuous Janitor:*
changeset, daily-doc-updater, daily-file-diet, daily-workflow-updater, developer-docs-consolidator, glossary-maintainer, hourly-ci-cleaner, instructions-janitor, slide-deck-maintainer, technical-doc-writer, unbloat-docs

*Pattern 4 - Quality Guardian:*
breaking-change-checker, ci-coach, daily-multi-device-docs-tester, firewall-escape, firewall, github-mcp-structural-analysis, mcp-inspector, schema-consistency-checker, smoke-claude, smoke-codex-firewall, smoke-codex, smoke-copilot-no-firewall, smoke-copilot-playwright, smoke-copilot-safe-inputs, smoke-copilot, smoke-detector, smoke-srt-custom-config, smoke-srt

*Pattern 5 - Issue & PR Manager:*
issue-arborist, issue-classifier, issue-template-optimizer, issue-triage-agent, sub-issue-closer

*Pattern 7 - Code Intelligence:*
cli-consistency-checker, duplicate-code-detector, go-pattern-detector, repository-quality-improver, semantic-function-refactor, terminal-stylist, typist

*Pattern 9 - Content & Documentation:*
artifacts-summary, blog-auditor, copilot-pr-merged-report, notion-issue-summary, pdf-summary, video-analyzer, weekly-issue-summary

*Pattern 10 - Event-Driven Coordinator:*
beads-worker, dev-hawk, docs-quality-maintenance-project67.campaign, docs-quality-maintenance-project67.campaign.g, file-size-reduction-project71.campaign, file-size-reduction-project71.campaign.g, spec-kit-execute, spec-kit-executor, workflow-health-manager

*Pattern 11 - ML & Analytics:*
copilot-pr-nlp-analysis, copilot-pr-prompt-analysis

*Pattern 12 - Security & Moderation:*
ai-moderator, daily-malicious-code-scan, daily-secrets-analysis, security-compliance, security-fix-pr, safe-output-health

**Specialized/experimental variants (33):**
brave, ci-doctor, cli-version-checker, cloclo, commit-changes-analyzer, craft, daily-assign-issue-to-user, daily-choice-test, daily-cli-performance, daily-code-metrics, daily-fact, daily-firewall-report, daily-issues-report, daily-news, daily-repo-chronicle, daily-team-status, deep-report, dependabot-go-checker, dev, dictation-prompt, docs-noob-tester, example-custom-error-patterns, example-permissions-warning, example-workflow-analyzer, github-mcp-tools-report, go-fan, go-logger, issue-monster, jsweep, layout-spec-maintainer, plan, playground-org-project-update-issue, playground-snapshots-refresh, python-data-charts, release, research, stale-repo-identifier, super-linter, tidy

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

