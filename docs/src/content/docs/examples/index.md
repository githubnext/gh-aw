---
title: GitHub Agentic Workflows Examples
description: Find GitHub Agentic Workflows examples by task, including issue triage, pull-request review, CI investigation, documentation, dependency analysis, reporting, maintenance, and security review.
sidebar:
  order: 1
---

GitHub Agentic Workflows examples show how Markdown workflows can run AI agents through GitHub Actions for repository tasks that require reasoning, interpretation, investigation, or generation. Use this catalog to choose a starting point; each entry explains when the pattern is useful and links to maintained guidance or workflow source.

## Examples by task

| Task | When to use it |
| --- | --- |
| [Issue triage](/gh-aw/examples/ai-issue-triage/) | Automatically classify new issues, identify duplicates, apply bounded labels, and ask for missing information. |
| [Pull-request review](/gh-aw/examples/automated-pr-review/) | Automatically inspect diffs for concrete defects and post review feedback through controlled safe outputs. |
| [Documentation maintenance](/gh-aw/examples/docs-automation/) | Automatically detect drift between code and documentation and propose reviewable updates. |
| [CI failure investigation](/gh-aw/examples/ci-failure-investigation/) | Automatically analyze failed GitHub Actions runs, correlate logs, and open diagnostic issues with likely causes. |
| [Code improvement](/gh-aw/examples/code-improvement/) | Automatically find unnecessary complexity or duplicated logic and propose focused changes for human review. |
| [Dependency analysis](/gh-aw/patterns/research-plan-assign-ops/) | Automatically research dependency usage and upstream changes before creating prioritized follow-up work. |
| [Metrics and analytics](/gh-aw/examples/metrics-analytics/) | Automatically collect workflow activity and store structured snapshots for health and performance analysis. |
| [Repository reporting](/gh-aw/examples/ai-release-notes/) | Automatically summarize repository or release activity on an event or schedule. |
| [Repository maintenance](https://github.com/githubnext/agentics/blob/main/docs/repo-assist.md) | Automatically review a backlog, perform bounded maintenance tasks, and propose controlled changes on a schedule. |
| [Security review](/gh-aw/examples/security-review/) | Automatically combine repository evidence with AI interpretation to report suspicious changes through code scanning. |

## Examples and AI engines

Most examples specify the default Copilot engine or omit `engine:` entirely, so the published example set is not evenly distributed across engines. Examples are engine-portable: to run one on Claude, Codex, Gemini, or Pi, change `engine:` in the workflow frontmatter and configure that engine's authentication secret. Engine-specific options such as `engine.agent` or `engine.harness` are not portable — see the [engine feature comparison](/gh-aw/reference/engines/#engine-feature-comparison) before switching.

## Use an example safely

Before enabling an example, review its trigger, AI engine authentication, tools, network access, permissions, and safe outputs. Compile the Markdown source with `gh aw compile`, inspect both the `.md` and generated `.lock.yml` files, and begin with the narrowest permissions and outputs that satisfy the task.

Follow the [quickstart](/gh-aw/setup/quick-start/) to install `gh-aw`, read [Creating Agentic Workflows](/gh-aw/setup/creating-workflows/) to adapt an example, compare [AI engines](/gh-aw/reference/engines/), and review the [security architecture](/gh-aw/introduction/architecture/) and [FAQ](/gh-aw/reference/faq/) before deployment.
