---
title: GitHub Agentic Workflows Examples
description: Find GitHub Agentic Workflows (gh-aw) examples by task, including issue triage, pull-request review, CI investigation, documentation, dependency analysis, reporting, maintenance, and security review.
sidebar:
  order: 1
---

GitHub Agentic Workflows (`gh-aw`) examples show how Markdown workflows can run AI agents through GitHub Actions for repository tasks that require reasoning, interpretation, investigation, or generation. Use this catalog to choose a starting point; each entry explains when the pattern is useful and links to maintained guidance or workflow source.

## Examples by task

| Task | When to use it | Example |
| --- | --- | --- |
| Issue triage | Classify new issues, identify duplicates, apply bounded labels, and ask for missing information. | [AI issue triage on GitHub](/gh-aw/guides/ai-issue-triage/) |
| Pull-request review | Inspect diffs for concrete defects and post review feedback through controlled safe outputs. | [Automated AI pull-request review](/gh-aw/guides/automated-pr-review/) |
| Documentation maintenance | Detect drift between code and documentation and propose reviewable updates. | [Keeping documentation up to date automatically](/gh-aw/guides/docs-automation/) |
| CI failure investigation | Analyze failed GitHub Actions runs, correlate logs, and open diagnostic issues with likely causes. | [CI Doctor and fault-investigation workflows](/gh-aw/blog/2026-01-13-meet-the-workflows-quality-hygiene/) |
| Code improvement | Find unnecessary complexity or duplicated logic and propose focused changes for human review. | [Continuous simplicity workflows](/gh-aw/blog/2026-01-13-meet-the-workflows-continuous-simplicity/) |
| Dependency analysis | Research dependency usage and upstream changes before creating prioritized follow-up work. | [ResearchPlanAssignOps dependency analysis](/gh-aw/patterns/research-plan-assign-ops/) |
| Repository reporting | Summarize repository or release activity on an event or schedule. | [AI-generated release notes and reports](/gh-aw/guides/ai-release-notes/) |
| Scheduled maintenance | Review a backlog regularly, select bounded maintenance tasks, and propose controlled changes. | [Automated repository maintenance](/gh-aw/examples/maintaining-repos/) |
| Security review | Combine deterministic security tools with AI interpretation to report suspicious changes or compliance work. | [Security-related workflows](/gh-aw/blog/2026-01-13-meet-the-workflows-security-compliance/) |

## Examples and AI engines

Most examples specify the default Copilot engine or omit `engine:` entirely, so the published example set is not evenly distributed across engines. Examples are engine-portable: to run one on Claude, Codex, Gemini, or Pi, change `engine:` in the workflow frontmatter and configure that engine's authentication secret. Engine-specific options such as `engine.agent` or `engine.harness` are not portable — see the [engine feature comparison](/gh-aw/reference/engines/#engine-feature-comparison) before switching.

## Use an example safely

Before enabling an example, review its trigger, AI engine authentication, tools, network access, permissions, and safe outputs. Compile the Markdown source with `gh aw compile`, inspect both the `.md` and generated `.lock.yml` files, and begin with the narrowest permissions and outputs that satisfy the task.

Follow the [quickstart](/gh-aw/setup/quick-start/) to install `gh-aw`, read [Creating Agentic Workflows](/gh-aw/setup/creating-workflows/) to adapt an example, compare [AI engines](/gh-aw/reference/engines/), and review the [security architecture](/gh-aw/introduction/architecture/) and [FAQ](/gh-aw/reference/faq/) before deployment.
