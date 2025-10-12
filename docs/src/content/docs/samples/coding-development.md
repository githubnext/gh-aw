---
title: Coding & Development 
description: Automated workflows for dependency management, documentation updates, and pull request assistance
sidebar:
  order: 300
---

Coding and development workflows streamline common development tasks through intelligent automation, reducing manual overhead and improving code quality. You can write your own workflows tailored to your specific technology stack and development practices.

## Sample Workflows

### 📦 Daily Dependency Updater
Automatically checks for dependency updates, creates branches, and submits PRs with updated versions to keep dependencies current without manual tracking. [Learn more](https://github.com/githubnext/agentics/blob/main/docs/daily-dependency-updates.md)

### 📖 Regular Documentation Update
Analyzes code changes and creates documentation PRs using Diátaxis methodology to ensure documentation stays current with code changes and API updates. [Learn more](https://github.com/githubnext/agentics/blob/main/docs/update-docs.md)

### 🏥 PR Fix
Investigates failing PR checks, identifies root causes, and pushes fixes to PR branches to speed up PR resolution and reduce developer context switching. [Learn more](https://github.com/githubnext/agentics/blob/main/docs/pr-fix.md)

### �🔍 Daily Adhoc QA
Follows README instructions, tests build processes, and validates user experience to catch user experience issues and documentation problems proactively. [Learn more](https://github.com/githubnext/agentics/blob/main/docs/daily-qa.md)

## Security Considerations

> [!WARNING]
> Coding workflows have network access and execute in GitHub Actions. Review all outputs carefully before merging, as they could potentially be influenced by untrusted inputs like issue descriptions or comments.

> [!WARNING]
> GitHub Agentic Workflows is a research demonstrator, and not for production use.
