---
title: Coding & Development 
description: Automated workflows for dependency management, documentation updates, and pull request assistance
sidebar:
  order: 3
---

Coding and development workflows streamline common development tasks through intelligent automation, reducing manual overhead and improving code quality.

You can write your own workflows tailored to your specific technology stack and development practices. Here are some sample workflows from the Agentics collection to get you started:

### 📦 Daily Dependency Updater
Update dependencies and create pull requests automatically.

- **What it does**: Checks for dependency updates, creates branches, and submits PRs with updated versions
- **Why it's valuable**: Keeps dependencies current without manual tracking and updating
- **Learn more**: [Daily Dependency Updater Documentation](https://github.com/githubnext/agentics/blob/main/docs/daily-dependency-updates.md)

### 📖 Regular Documentation Update
Update documentation automatically on code changes.

- **What it does**: Analyzes code changes and creates documentation PRs using Diátaxis methodology
- **Why it's valuable**: Ensures documentation stays current with code changes and API updates
- **Learn more**: [Regular Documentation Update Documentation](https://github.com/githubnext/agentics/blob/main/docs/update-docs.md)

### 🏥 PR Fix
Analyze failing CI checks and implement fixes for pull requests.

- **What it does**: Investigates failing PR checks, identifies root causes, and pushes fixes to PR branches
- **Why it's valuable**: Speeds up PR resolution and reduces developer context switching
- **Learn more**: [PR Fix Documentation](https://github.com/githubnext/agentics/blob/main/docs/pr-fix.md)

### 🔍 Daily Adhoc QA
Perform adhoc explorative quality assurance tasks.

- **What it does**: Follows README instructions, tests build processes, and validates user experience
- **Why it's valuable**: Catches user experience issues and documentation problems proactively
- **Learn more**: [Daily QA Documentation](https://github.com/githubnext/agentics/blob/main/docs/daily-qa.md)

## Security Considerations

> [!WARNING]
> Coding workflows have network access and execute in GitHub Actions. Review all outputs carefully before merging, as they could potentially be influenced by untrusted inputs like issue descriptions or comments.

> [!WARNING]
> GitHub Agentic Workflows is a research demonstrator, and not for production use.
