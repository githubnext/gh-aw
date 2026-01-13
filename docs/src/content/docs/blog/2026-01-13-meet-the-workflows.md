---
title: "Meet the Agentic Workflows in Peli's Agent Factory"
description: "A curated tour of the most interesting agents in the factory"
authors:
  - dsyme
date: 2026-01-12
---

<img src="/gh-aw/peli.png" alt="Peli de Halleux" width="200" style="float: right; margin: 0 0 20px 20px; border-radius: 8px;" />

Hi there! 👋 Welcome back to Peli's Agent Factory.

We're the GitHub Next team, and we've been on quite a journey. Over the past months, we've built and operated a collection of automated agentic workflows. These aren't just demos or proof-of-concepts - these are real agents doing actual work in our [`githubnext/gh-aw`](https://github.com/githubnext/gh-aw) repository and its companion [`githubnext/agentics`](https://github.com/githubnext/agentics) collection.

Think of this as your guided tour through our agent "factory". We're showcasing the workflows that caught our attention, taught us something new, or just flat-out made our lives easier. Every workflow links to its source Markdown file, so you can peek under the hood and see exactly how it works.

## 🏥 Triage & Summarization Workflows

First up: the agents that help us stay sane when things get busy. These workflows keep us on top of the constant flow of activity:

- **[Issue Triage Agent](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/issue-triage-agent.md)** - Automatically labels and categorizes new issues the moment they're opened
- **[Weekly Issue Summary](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/weekly-issue-summary.md)** - Creates digestible summaries complete with charts and trends (because who has time to read everything?)
- **[Daily Repo Chronicle](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/daily-repo-chronicle.md)** - Narrates the day's activity like a storyteller - seriously, it's kind of delightful

What surprised us most about this category? The **tone** matters way more than we expected. When the Daily Repo Chronicle started writing summaries in a narrative, almost journalistic style, people actually *wanted* to read them. We discovered that AI agents don't have to be robotic - they can have personality while still being informative. The Issue Triage Agent taught us that even simple automation (just adding labels!) dramatically reduces cognitive load when you're scanning through dozens of issues. These workflows became our daily reading habit rather than another notification to dismiss.

## 🔍 Quality & Hygiene Workflows

These are our diligent caretakers - the agents that spot problems before they become, well, bigger problems:

- **[CI Doctor](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/ci-doctor.md)** - Investigates failed workflows and opens diagnostic issues (it's like having a DevOps specialist on call 24/7)
- **[Schema Consistency Checker](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/schema-consistency-checker.md)** - Detects when schemas, code, and docs drift apart
- **[Breaking Change Checker](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/breaking-change-checker.md)** - Watches for changes that might break things for users

The CI Doctor was a revelation. Instead of drowning in CI failure notifications, we now get *timely*, *investigated* failures with actual diagnostic insights. The agent doesn't just tell us something broke - it analyzes logs, identifies patterns, searches for similar past issues, and even suggests fixes. We learned that agents excel at the tedious investigation work that humans find draining. The Schema Consistency Checker caught drift that would have taken us days to notice manually. These "hygiene" workflows became our first line of defense, catching issues before they reached users.

## 📊 Metrics & Analytics Workflows

Data nerds, rejoice! These agents turn raw repository activity into actual insights:

- **[Metrics Collector](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/metrics-collector.md)** - Tracks daily performance across the entire agent ecosystem
- **[Portfolio Analyst](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/portfolio-analyst.md)** - Identifies cost reduction opportunities (because AI isn't free!)
- **[Audit Workflows](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/audit-workflows.md)** - A meta-agent that audits all the other agents' runs - very Inception

Here's where things got meta: we built agents to monitor agents. The Metrics Collector became our central nervous system, gathering performance data that feeds into higher-level orchestrators. What we learned: **you can't optimize what you don't measure**. The Portfolio Analyst was eye-opening - it identified workflows that were costing us money unnecessarily (turns out some agents were way too chatty with their LLM calls). These workflows taught us that observability isn't optional when you're running dozens of AI agents - it's the difference between a well-oiled machine and an expensive black box.

## 🔒 Security & Compliance Workflows

These agents are our security guards, keeping watch and enforcing the rules:

- **[Security Compliance](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/security-compliance.md)** - Runs vulnerability campaigns with deadline tracking
- **[Firewall](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/firewall.md)** - Tests network security and validates rules
- **[Daily Secrets Analysis](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/daily-secrets-analysis.md)** - Scans for exposed credentials (yes, it happens)

Security workflows were where we got serious about trust boundaries. The Security Compliance agent manages entire vulnerability remediation campaigns with deadline tracking - perfect for those "audit in 3 weeks" panic moments. We learned that AI agents need guardrails just like humans need seat belts. The Firewall workflow validates that our agents can't access unauthorized resources, because an AI agent with unrestricted network access is... let's just say we sleep better with these safeguards. These workflows prove that automation and security aren't at odds - when done right, automated security is more consistent than manual reviews.

## 🚀 Operations & Release Workflows

The agents that help us actually ship software:

- **[Release](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/release.md)** - Orchestrates builds, tests, and release note generation
- **[Daily Workflow Updater](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/daily-workflow-updater.md)** - Keeps actions and dependencies current (because dependency updates never stop)

Shipping software is stressful enough without worrying about whether you formatted your release notes correctly. The Release workflow handles the entire orchestration - building, testing, generating coherent release notes from commits, and publishing. What's interesting here is the **reliability** requirement: these workflows can't afford to be creative or experimental. They need to be deterministic, well-tested, and boring (in a good way). The Daily Workflow Updater taught us that maintenance is a perfect use case for agents - it's repetitive, necessary, and nobody enjoys doing it manually. These workflows handle the toil so we can focus on the interesting problems.

## 🎨 Creative & Culture Workflows

Not everything needs to be serious! These agents remind us that work can be fun:

- **[Poem Bot](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/poem-bot.md)** - Responds to `/poem-bot` commands with creative verses (yes, really)
- **[Daily Team Status](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/daily-team-status.md)** - Shares team mood and status updates
- **[Daily News](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/daily-news.md)** - Curates relevant news for the team

Okay, hear us out: the Poem Bot started as a joke. Someone said "wouldn't it be funny if we had an agent that writes poems about our code?" and then we actually built it. And you know what? People *love* it. It became a team tradition to invoke `/poem-bot` after particularly gnarly PR merges. We learned that AI agents don't have to be all business - they can build culture and create moments of joy. The Daily News workflow curates relevant articles, but it also adds commentary and connects them to our work. These "fun" workflows have higher engagement than some of our "serious" ones, which tells you something about what makes people actually want to interact with automation.

## 💬 Interactive & ChatOps Workflows

These agents respond to commands, providing on-demand assistance whenever you need it:

- **[Q](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/q.md)** - Workflow optimizer that investigates performance and creates PRs
- **[Grumpy Reviewer](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/grumpy-reviewer.md)** - Performs critical code reviews with, well, personality
- **[Workflow Generator](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/workflow-generator.md)** - Creates new workflows from issue requests

Interactive workflows changed how we think about agent invocation. Instead of everything running on a schedule, these respond to slash commands and reactions - `/q` summons the workflow optimizer, a 🚀 reaction triggers analysis. Q (yes, named after the James Bond quartermaster) became our go-to troubleshooter - it investigates workflow performance issues and opens PRs with optimizations. The Grumpy Reviewer gave us surprisingly valuable feedback with a side of sass ("This function is so nested it has its own ZIP code"). We learned that **context is king** - these agents work because they're invoked at the right moment with the right context, not because they run on a schedule.

## 🔧 Code Quality & Refactoring Workflows

These agents make our codebase cleaner and our developer experience better:

- **[Terminal Stylist](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/terminal-stylist.md)** - Analyzes and improves console output styling (because aesthetics matter!)
- **[Semantic Function Refactor](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/semantic-function-refactor.md)** - Spots refactoring opportunities we might have missed
- **[Repository Quality Improver](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/repository-quality-improver.md)** - Takes a holistic view of code quality and suggests improvements

Code quality is where AI agents really shine - they never get bored doing the repetitive analysis that makes codebases better. The Terminal Stylist literally reads our console output code and suggests improvements to make our CLI prettier (and yes, it understands Lipgloss and modern terminal styling). The Semantic Function Refactor finds duplicated logic that's not quite identical enough for traditional duplicate detection. We learned that these agents see patterns humans miss because they can hold the entire codebase in context. The Repository Quality Improver takes a holistic view - it doesn't just find bugs, it identifies structural improvements and documentation gaps. These workflows continuously push our codebase toward better design.

## 🔬 Testing & Validation Workflows

These agents keep everything running smoothly through continuous testing:

- **[Smoke Tests](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/smoke-codex-firewall.md)** - Validate that engines and firewall are working (running every 12 hours!)
- **[Daily Multi-Device Docs Tester](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/daily-multi-device-docs-tester.md)** - Tests documentation across devices (mobile matters!)
- **[CI Coach](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/ci-coach.md)** - Provides friendly guidance on CI/CD improvements

We learned the hard way that AI infrastructure needs constant health checks. The Smoke Tests run every 12 hours to validate that our core systems (engines, firewall, MCP servers) are actually working. It's caught outages before users noticed them. The Multi-Device Docs Tester uses Playwright to test our documentation on different screen sizes - it found mobile rendering issues we never would have caught manually. The CI Coach analyzes our CI/CD pipeline and suggests optimizations ("you're running tests sequentially when they could be parallel"). These workflows embody the principle: **trust but verify**. Just because it worked yesterday doesn't mean it works today.

## 🧰 Tool & Infrastructure Workflows

These agents monitor and analyze the agentic infrastructure itself:

- **[MCP Inspector](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/mcp-inspector.md)** - Validates Model Context Protocol configurations
- **[GitHub MCP Tools Report](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/github-mcp-tools-report.md)** - Analyzes available MCP tools
- **[Agent Performance Analyzer](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/agent-performance-analyzer.md)** - Meta-orchestrator for agent quality

Infrastructure for AI agents is different from traditional infrastructure - you need to validate that tools are available, properly configured, and actually working. The MCP Inspector checks Model Context Protocol server configurations because a misconfigured MCP server means an agent can't access the tools it needs. The Agent Performance Analyzer is a meta-orchestrator that monitors all our other agents - looking for performance degradation, cost spikes, and quality issues. We learned that **layered observability** is crucial: you need monitoring at the infrastructure level (are servers up?), the tool level (can agents access what they need?), and the agent level (are they performing well?). These workflows provide visibility into the invisible.

## 🚀 Multi-Phase Improver Workflows

These are some of our most ambitious agents - they tackle big projects over multiple days:

- **[Daily Backlog Burner](https://github.com/githubnext/agentics/blob/main/workflows/daily-backlog-burner.md)** - Systematically works through issues and PRs, one day at a time
- **[Daily Perf Improver](https://github.com/githubnext/agentics/blob/main/workflows/daily-perf-improver.md)** - Three-phase performance optimization (research, setup, implement)
- **[Daily Test Improver](https://github.com/githubnext/agentics/blob/main/workflows/daily-test-improver.md)** - Identifies coverage gaps and implements new tests incrementally
- **[Daily QA](https://github.com/githubnext/agentics/blob/main/workflows/daily-qa.md)** - Continuous quality assurance that never sleeps
- **[Daily Accessibility Review](https://github.com/githubnext/agentics/blob/main/workflows/daily-accessibility-review.md)** - WCAG compliance checking with Playwright
- **[PR Fix](https://github.com/githubnext/agentics/blob/main/workflows/pr-fix.md)** - On-demand slash command to fix failing CI checks (super handy!)

This is where we got experimental with agent persistence and multi-day workflows. Traditional CI runs are ephemeral, but these workflows maintain state across days using repo-memory. The Daily Perf Improver runs in three phases - research (find bottlenecks), setup (create profiling infrastructure), implement (optimize). It's like having a performance engineer who works a little bit each day. The Daily Backlog Burner systematically tackles our issue backlog - one issue per day, methodically working through technical debt. We learned that **incremental progress beats heroic sprints** - these agents never get tired, never get distracted, and never need coffee breaks. The PR Fix workflow is our emergency responder - when CI fails, invoke `/pr-fix` and it investigates and attempts repairs. These workflows prove that AI agents can handle complex, long-running projects when given the right architecture.

## 📊 Advanced Analytics & ML Workflows

These agents use sophisticated analysis techniques to extract insights:

- **[Copilot Session Insights](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/copilot-session-insights.md)** - Analyzes Copilot agent usage patterns and metrics
- **[Copilot PR NLP Analysis](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/copilot-pr-nlp-analysis.md)** - Natural language processing on PR conversations
- **[Prompt Clustering Analysis](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/prompt-clustering-analysis.md)** - Clusters and categorizes agent prompts using ML
- **[Copilot Agent Analysis](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/copilot-agent-analysis.md)** - Deep analysis of agent behavior patterns

We got nerdy with these workflows. The Prompt Clustering Analysis uses machine learning to categorize thousands of agent prompts, revealing patterns we never noticed ("oh, 40% of our prompts are about error handling"). The Copilot PR NLP Analysis does sentiment analysis and linguistic analysis on PR conversations - it found that PRs with questions in the title get faster review. The Session Insights workflow analyzes how developers interact with Copilot agents, identifying common patterns and failure modes. What we learned: **meta-analysis is powerful** - using AI to analyze AI systems reveals insights that direct observation misses. These workflows helped us understand not just what our agents do, but *how* they behave and how users interact with them.

## 🏢 Organization & Cross-Repo Workflows

These agents work at organization scale, across multiple repositories:

- **[Org Health Report](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/org-health-report.md)** - Organization-wide repository health metrics
- **[Stale Repo Identifier](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/stale-repo-identifier.md)** - Identifies inactive repositories
- **[Ubuntu Image Analyzer](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/ubuntu-image-analyzer.md)** - Documents GitHub Actions runner environments

Scaling agents across an entire organization changes the game. The Org Health Report analyzes dozens of repositories at once, identifying patterns and outliers ("these three repos have no tests, these five haven't been updated in months"). The Stale Repo Identifier helps with organizational hygiene - finding abandoned projects that should be archived or transferred. We learned that **cross-repo insights are different** - what looks fine in one repository might be an outlier across the organization. These workflows require careful permission management (reading across repos needs organization-level tokens) and thoughtful rate limiting (you can hit API limits fast when analyzing 50+ repos). The Ubuntu Image Analyzer is wonderfully meta - it documents the very environment that runs our agents.

## 📝 Documentation & Content Workflows

These agents maintain high-quality documentation and content:

- **[Glossary Maintainer](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/glossary-maintainer.md)** - Keeps glossary synchronized with codebase
- **[Technical Doc Writer](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/technical-doc-writer.md)** - Generates and updates technical documentation
- **[Slide Deck Maintainer](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/slide-deck-maintainer.md)** - Maintains presentation slide decks
- **[Blog Auditor](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/blog-auditor.md)** - Reviews blog content for quality and accuracy

Documentation is where we challenged conventional wisdom. Can AI agents write *good* documentation? The Technical Doc Writer generates API docs from code, but more importantly, it *maintains* them - updating docs when code changes. The Glossary Maintainer caught terminology drift ("we're using three different terms for the same concept"). The Slide Deck Maintainer keeps our presentation materials current without manual updates. We learned that **AI-generated docs need human review**, but they're dramatically better than *no* docs (which is often the alternative). The Blog Auditor ensures our blog posts stay accurate as the codebase evolves - it flags outdated code examples and broken links. These workflows don't replace technical writers; they multiply their effectiveness.

## 🔗 Issue & PR Management Workflows

These agents enhance issue and pull request workflows:

- **[Issue Arborist](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/issue-arborist.md)** - Links related issues as sub-issues
- **[Issue Monster](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/issue-monster.md)** - Assigns issues to Copilot agents one at a time
- **[Mergefest](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/mergefest.md)** - Automatically merges main branch into PR branches
- **[Sub Issue Closer](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/sub-issue-closer.md)** - Closes completed sub-issues automatically
- **[Issue Template Optimizer](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/issue-template-optimizer.md)** - Improves issue templates based on usage

Issue management is tedious ceremony that developers tolerate rather than enjoy. The Issue Arborist automatically links related issues, building a dependency tree we'd never maintain manually. The Issue Monster became our task dispatcher for AI agents - it assigns one issue at a time to Copilot agents, preventing the chaos of parallel work on the same codebase. Mergefest eliminates the "please merge main" dance that happens on long-lived PRs. We learned that **tiny frustrations add up** - each of these workflows removes a small papercut, and collectively they make GitHub feel much more pleasant to use. The Issue Template Optimizer analyzes which fields in our templates actually get filled out and suggests improvements ("nobody uses the 'Expected behavior' field, remove it").

## 🎯 Campaign & Project Coordination Workflows

These agents manage structured improvement campaigns:

- **[Campaign Generator](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/campaign-generator.md)** - Creates and coordinates multi-step campaigns
- **[Workflow Health Manager](https://github.com/githubnext/gh-aw/tree/2c1f68a721ae7b3b67d0c2d93decf1fa5bcf7ee3/.github/workflows/workflow-health-manager.md)** - Monitors and maintains workflow health

Campaigns are where we tackled the "how do you coordinate multiple agents on a big project?" question. The Campaign Generator creates structured improvement campaigns - breaking down large initiatives ("migrate all workflows to new engine") into trackable sub-tasks that different agents can tackle. The Workflow Health Manager acts as a project manager, monitoring progress across campaigns and alerting when things fall behind. We learned that **coordination is the hard part** - individual agents are great at focused tasks, but orchestrating multiple agents toward a shared goal requires careful architecture. These workflows implement patterns like epic issues, progress tracking, and deadline management. They prove that AI agents can handle not just individual tasks, but entire projects when given proper coordination infrastructure.

---

## What's Next?

This collection is just the beginning! As you explore these workflows, you'll start noticing patterns - common structures, shared capabilities, and design choices that keep showing up.

We've learned *so much* from running our collection of automated agentic workflows in practice, and we're excited to share those insights with you. Coming up in this series, we'll dive into the key lessons, design patterns, operational strategies, and security considerations that emerged from this wild experiment.

Stay tuned - we've got plenty more to share!

*More articles in this series coming soon.*
