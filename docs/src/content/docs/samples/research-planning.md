---
title: Research, Status & Planning
description: Automated workflows for research, team coordination, and project planning
sidebar:
  order: 100
---

Research and planning workflows help teams stay informed, coordinate activities, and maintain strategic direction through automated intelligence gathering and status reporting.

You can write your own workflows customized for your team's specific needs. Here are some sample workflows from the Agentics collection:

### 📚 Weekly Research
Automatically collects latest trends, competitive analysis, and relevant research every Monday, keeping teams informed about industry developments without manual research overhead. [Learn more](https://github.com/githubnext/agentics/blob/main/docs/weekly-research.md)

### 👥 Daily Team Status
Analyzes repository activity, pull requests, and team progress to provide automated visibility into team productivity and project health. [Learn more](https://github.com/githubnext/agentics/blob/main/docs/daily-team-status.md)

### 📋 Daily Plan
Maintains and updates project planning issues with current priorities, ensuring project plans stay current and accessible to all team members. [Learn more](https://github.com/githubnext/agentics/blob/main/docs/daily-plan.md)

### 🔍 Basic Research
Searches for information on a given topic, analyzes results, and creates structured summaries with relevant sources. Triggered manually via workflow_dispatch with research topic input. Workflow file: `.github/workflows/research.md`

### 🔍 MCP Inspector
Analyzes all MCP configuration files, extracts server details, and generates comprehensive inventory reports to maintain visibility into available MCP servers and their capabilities. Runs weekly on Mondays at 10am UTC, or manually via workflow_dispatch. Workflow file: `.github/workflows/mcp-inspector.md`

> [!WARNING]
> GitHub Agentic Workflows is a research demonstrator, and not for production use.

