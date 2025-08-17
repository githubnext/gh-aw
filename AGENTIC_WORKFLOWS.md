# 🧳 Agentic Workflows Menu

> Your comprehensive guide to all AI-powered workflows in this repository

*Last Updated: December 17, 2024*

## 🤖 Agent Directory

| Agent | Triggers | Schedule | Description |
|-------|----------|----------|-------------|
| 📊 **Agent Standup** | 📅 Schedule, ⚙️ Manual | Daily 9 AM UTC | Daily summary of agentic workflow activity including runs, costs, and errors |
| 👥 **Daily Team Status** | 📅 Schedule, ⚙️ Manual | Daily 9 AM UTC | Motivational team progress report with productivity suggestions and seasonal haiku |
| 🔍 **Weekly Research** | 📅 Schedule, ⚙️ Manual | Weekly Mon 9 AM UTC | Deep research on repository and industry trends, competitive analysis |
| 📦 **Agentic Dependency Updater** | 📅 Schedule, ⚙️ Manual | Daily midnight UTC | Automated dependency updates with security analysis and bundled PRs |
| 🎯 **Agentic Triage** | 🔢 Issues (opened/reopened) | On-demand | Smart issue labeling, analysis, and debugging strategy recommendations |
| 🧹 **The Linter Maniac** | 🔄 Workflow runs (CI completed) | On-demand | Auto-fixes formatting/linting issues when CI lint jobs fail |
| 🔀 **@mergefest** | 📝 Alias trigger | On-mention | Merge assistant that updates PR branches with parent branch changes |
| 🕵️ **Scout The Researcher** | 📝 Alias trigger (`@scout`) | On-mention | Deep research assistant for GitHub issues with comprehensive analysis |
| 🧠 **RepoMind Agent** | 📝 Alias trigger (`@repomind`) | On-mention | Advanced code search and repository analysis with specialized MCP tools |
| ⚙️ **Go Module Guardian** | 🔀 Pull requests (go.mod/go.sum changes) | On-demand | Specialized Go dependency security and compatibility analysis |
| 🏥 **CI Failure Doctor** | 🔄 Workflow runs (CI completed) | On-demand | Expert analysis of failed CI runs with root cause investigation |
| 📱 **Travel Agent** | 🔢 Issues (labeled) | On-demand | Converts issues into copilot agent instruction prompts |
| 🎨 **The Terminal Stylist** | 📅 Schedule, 📝 Alias (`@glam`) | Daily 9 AM UTC | Terminal UI beautification specialist using lipgloss styling |
| 💰 **Tech Debt Collector** | 📅 Schedule, ⚙️ Manual | Daily 10 AM UTC | Systematic code quality improvement and technical debt reduction |
| 📚 **Starlight Scribe** | 📤 Push (main branch), ⚙️ Manual | On-demand | Technical documentation writer using Astro Starlight and Diátaxis methodology |
| 🎉 **Release Storyteller** | 🚀 Releases (published), ⚙️ Manual | On-demand | Engaging release notes generator with comprehensive change analysis |
| 👨‍⚕️ **Daily QA** | 📅 Schedule, ⚙️ Manual | Daily midnight UTC | QA engineer analyzing builds, tests, and documentation quality |
| 🎯 **Agentic Planner** | 📅 Schedule, ⚙️ Manual | Daily midnight UTC | Project planning assistant analyzing repository state and priorities |
| 📈 **Daily Test Coverage Improve** | 📅 Schedule, ⚙️ Manual | Weekday 2 AM UTC | Automated test coverage improvement with meaningful test additions |
| 🔬 **Deep Research with Codex** | ⚙️ Manual | On-demand | Comprehensive research using Codex engine for technical architecture analysis |
| 🧪 **Integration Test** | 📤 Push, 🔀 PR, ⚙️ Manual | On-demand | Comprehensive GitHub MCP integration testing and validation |
| 📋 **Agent Menu** | 📤 Push (workflow changes), ⚙️ Manual | On-demand | Documentation specialist maintaining this comprehensive workflow guide |
| 🔍 **Action Workflow Assessor** | 🔀 Pull requests (workflow changes) | On-demand | Security and capability assessor for agentic workflow modifications |
| 🏷️ **Issue Labeller** | 🔢 Issues (opened) | On-demand | Basic issue labeling service for newly opened issues |
| 🧪 **Test Claude** | 📤 Push (*claude* branches), ⚙️ Manual | On-demand | Code review assistant powered by Claude AI |
| 🧪 **Test Codex** | 📤 Push (*codex* branches), ⚙️ Manual | On-demand | Code review assistant powered by Codex AI |
| 👨‍⚕️ **Run Doctor** | 🔄 Workflow runs (CI completed) | On-demand | Diagnoses and provides fixes for failed CI workflow runs |

## 📅 Schedule Overview

| 🕐 Frequency | 📝 Workflow | ⏰ Schedule | 🎯 Purpose |
|-------------|-------------|-------------|------------|
| 🌅 **Daily 9 AM** | Agent Standup | `0 9 * * *` | Daily workflow activity summary |
| 🌅 **Daily 9 AM** | Daily Team Status | `0 9 * * *` | Motivational team progress report |
| 🌅 **Daily 9 AM** | Terminal Stylist | `0 9 * * *` | Terminal UI beautification |
| 🌙 **Daily 10 AM** | Tech Debt Collector | `0 10 * * *` | Systematic code quality improvements |
| 🌙 **Daily midnight** | Agentic Dependency Updater | `0 0 * * *` | Automated dependency updates |
| 🌙 **Daily midnight** | Agentic Planner | `0 0 * * *` | Project planning and priority analysis |
| 🌙 **Daily midnight** | Daily QA | `0 0 * * *` | Quality assurance and testing analysis |
| 🌆 **Weekday 2 AM** | Test Coverage Improve | `0 2 * * 1-5` | Automated test coverage enhancement |
| 📊 **Weekly Mon** | Weekly Research | `0 9 * * 1` | Comprehensive research digest |

> **💡 Pro Tip:** All times are in UTC. Workflows use GitHub Actions' cron syntax with minute, hour, day, month, and day-of-week fields.

## 🏷️ Agent Aliases

| 🤖 Agent Name | 📛 @alias | 📁 Filename |
|---------------|------------|-------------|
| **@mergefest** | `mergefest` | `mergefest.md` |
| **Scout The Researcher** | `@scout` | `scout.md` |
| **RepoMind Agent** | `@repomind` | `repomind.md` |
| **Terminal Stylist** | `@glam` | `terminal-stylist.md` |
| **Deep Research with Codex** | `deep-research-codex` | `deep-research-codex.md` |

> **🎯 Usage:** Use aliases for faster workflow management. Example: `@scout` in issue comments to trigger deep research analysis.

## 🔐 Permission Groups

### 🔓 **Read-Only Workflows**
- Integration Test
- Test Claude  
- Test Codex
- Action Workflow Assessor

### ✍️ **Issue & Comment Writers**
- Agent Standup
- Daily Team Status  
- Weekly Research
- Agentic Triage
- Scout The Researcher
- CI Failure Doctor
- Daily QA
- Agentic Planner

### 🔧 **Code & Content Modifiers**
- Agentic Dependency Updater
- The Linter Maniac
- @mergefest
- Tech Debt Collector
- Starlight Scribe
- Terminal Stylist
- Daily Test Coverage Improve

### ⚠️ **High-Permission Workflows**
- Go Module Guardian (pull-requests: write)
- Release Storyteller (pull-requests: write)

## 🛠️ MCP Tools Catalog

### 📊 **GitHub API Tools** (Most Common)
Used by nearly all workflows for repository interaction:
- `get_issue`, `create_issue`, `update_issue`, `add_issue_comment`
- `get_pull_request`, `create_pull_request`, `update_pull_request`
- `get_file_contents`, `create_or_update_file`, `push_files`
- `search_code`, `search_issues`, `search_pull_requests`

### 🧠 **Claude AI Tools** 
- **WebFetch & WebSearch**: 19 workflows (research, analysis)
- **Edit & Write**: 15 workflows (content modification)  
- **Bash**: 8 workflows (command execution)
- **MultiEdit**: 6 workflows (bulk file editing)

### 🔧 **Specialized Tools**
- **repo-mind**: RepoMind Agent (optimized code search)
- **time**: Test Claude/Codex (timezone support)

### 🎯 **Tool Usage by Category**

| 📋 Category | 🔧 Tools | 📊 Usage Count |
|-------------|----------|---------------|
| **Research** | WebFetch, WebSearch | 19 workflows |
| **Content** | Edit, Write, MultiEdit | 15 workflows |
| **Automation** | Bash, Git commands | 8 workflows |
| **Testing** | Read, LS, Glob, Grep | 6 workflows |

## 📋 Quick Reference

### 🚀 **Most Active Agents** (Scheduled)
- Agent Standup (daily activity reports)
- Daily Team Status (motivation & planning)
- Tech Debt Collector (code improvements)
- Agentic Dependency Updater (security updates)

### 🎯 **On-Demand Specialists**
- @mergefest (branch merging)
- @scout (deep research)
- @repomind (code analysis)
- Go Module Guardian (dependency security)

### 🔧 **CI/CD Integration**
- The Linter Maniac (auto-fixes lint failures)
- CI Failure Doctor (diagnoses build failures)
- Integration Test (validates GitHub MCP tools)

### 🎨 **Content & Documentation**
- Starlight Scribe (technical documentation)  
- Release Storyteller (engaging release notes)
- Terminal Stylist (beautiful CLI interfaces)

### 🧪 **Development & Testing**
- Daily Test Coverage Improve (automated testing)
- Test Claude/Codex (AI-powered code review)
- Action Workflow Assessor (workflow security)

---

## 🚀 Getting Started

1. **Browse the directory** above to find agents that match your needs
2. **Check schedules** to understand when automated agents run
3. **Use aliases** like `@scout` or `@mergefest` in issue comments for instant help
4. **Monitor permissions** when contributing new workflows or modifying existing ones
5. **Leverage MCP tools** for advanced repository analysis and automation

> **📖 Learn More:** See individual workflow files in `.github/workflows/` for detailed instructions and capabilities.

---

> AI-generated content by [Agent Menu](https://github.com/githubnext/gh-aw-internal/actions/runs/17023213151) may contain mistakes.