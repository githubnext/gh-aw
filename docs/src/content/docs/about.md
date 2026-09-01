---
title: About
description: About GitHub Agentic Workflows — an open-source GitHub CLI extension by GitHub that compiles natural language markdown workflows into GitHub Actions.
sidebar:
  label: About
  order: 1
---

## What is GitHub Agentic Workflows?

**GitHub Agentic Workflows** (`gh-aw`) is an open-source [GitHub CLI](https://cli.github.com/) extension from [GitHub](https://github.com/github) for defining AI-powered repository automation and running AI agents through GitHub Actions. Each agentic workflow combines YAML frontmatter for triggers, permissions, tools, and AI engine selection with a Markdown body containing natural-language instructions. The `gh aw compile` command validates that source and generates the `.lock.yml` workflow that GitHub Actions executes.

GitHub Agentic Workflows complements deterministic GitHub Actions workflows. Keep builds, tests, linting, deployments, and reproducible scripts deterministic; use an agentic workflow when a task needs reasoning, investigation, interpretation, or content and code generation. The supported agent-job path defaults to sandboxing and read-only permissions, with controlled write operations available through safe outputs.

Start with the [GitHub Agentic Workflows quickstart](/gh-aw/setup/quick-start/), then learn how to [create an agentic workflow](/gh-aw/setup/creating-workflows/), [choose an AI engine](/gh-aw/reference/engines/), [review the security architecture](/gh-aw/introduction/architecture/), or [browse the gallery by task](/gh-aw/gallery/).

## Project

| Resource | Link |
| --- | --- |
| Repository | [github/gh-aw](https://github.com/github/gh-aw) |
| Install | `gh extension install github/gh-aw` |
| License | [MIT](https://github.com/github/gh-aw/blob/main/LICENSE) |
| Changelog | [CHANGELOG.md](https://github.com/github/gh-aw/blob/main/CHANGELOG.md) |

## Team

GitHub Agentic Workflows is built and maintained by the [GitHub Next](https://githubnext.com/) team at GitHub.

## Community

Join [GitHub Discussions](https://github.com/github/gh-aw/discussions) for questions, ideas, and announcements, share feedback in the [Community discussion](https://github.com/orgs/community/discussions/186451), or chat in the [GitHub Next Discord](https://gh.io/next-discord).

## Contributing and security

Contributions are welcome. See [CONTRIBUTING.md](https://github.com/github/gh-aw/blob/main/CONTRIBUTING.md) for setup and guidelines. For vulnerability reports, see [SECURITY.md](https://github.com/github/gh-aw/blob/main/SECURITY.md).
