---
title: Research, Status & Planning
description: Automated workflows for research, team coordination, and project planning
sidebar:
  order: 100
---

Research and planning workflows help teams stay informed, coordinate activities, and maintain strategic direction through automated intelligence gathering and status reporting.

You can write your own workflows customized for your team's specific needs. Here are some sample workflows from the Agentics collection to get you started:

### 📚 Weekly Research
Collect research updates and industry trends automatically every Monday.

- **What it does**: Searches for latest trends, competitive analysis, and relevant research
- **Why it's valuable**: Keeps teams informed about industry developments without manual research overhead
- **Learn more**: [Weekly Research Documentation](https://github.com/githubnext/agentics/blob/main/docs/weekly-research.md)

### 👥 Daily Team Status  
Assess repository activity and create comprehensive status reports.

- **What it does**: Analyzes repository activity, pull requests, and team progress
- **Why it's valuable**: Provides automated visibility into team productivity and project health
- **Learn more**: [Daily Team Status Documentation](https://github.com/githubnext/agentics/blob/main/docs/daily-team-status.md)

### 📋 Daily Plan
Update planning issues for team coordination and priority alignment.

- **What it does**: Maintains and updates project planning issues with current priorities
- **Why it's valuable**: Ensures project plans stay current and accessible to all team members
- **Learn more**: [Daily Plan Documentation](https://github.com/githubnext/agentics/blob/main/docs/daily-plan.md)

### 🔍 Basic Research
Perform simple web research and summarization using Tavily.

- **What it does**: Searches for information on a given topic, analyzes results, and creates a summary with key findings
- **Why it's valuable**: Automates research tasks and provides structured summaries with relevant sources
- **Trigger**: Manual via workflow_dispatch with research topic input
- **Workflow file**: `.github/workflows/research.md`

### 🔍 MCP Inspector
Systematically audit and document all MCP server configurations.

- **What it does**: Analyzes all MCP configuration files, extracts server details, and generates comprehensive inventory reports
- **Why it's valuable**: Maintains visibility into available MCP servers, their capabilities, and configuration status
- **Trigger**: Weekly on Mondays at 10am UTC, or manual via workflow_dispatch
- **Workflow file**: `.github/workflows/mcp-inspector.md`

> [!WARNING]
> GitHub Agentic Workflows is a research demonstrator, and not for production use.

