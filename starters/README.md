# GitHub Agentic Workflow Starters

Tiny but useful starter workflows — one per Ops pattern — to help you get started with GitHub Agentic Workflows. Copy any file to `.github/workflows/` in your repository, adjust the placeholders, then run `gh aw compile <name>` to generate the lock file.

## Pattern Summary

| Pattern | Starter | Trigger | What It Does | Why It's Useful |
|---------|---------|---------|--------------|-----------------|
| **[ChatOps](https://github.github.com/gh-aw/patterns/chat-ops/)** | [`chat-ops/pr-explain.md`](chat-ops/pr-explain.md) | `/explain` in PR comment | Summarizes PR changes in plain English | Non-engineers understand code changes without reading diffs |
| **[IssueOps](https://github.github.com/gh-aw/patterns/issue-ops/)** | [`issue-ops/first-contributor-welcome.md`](issue-ops/first-contributor-welcome.md) | Issue or PR opened | Greets first-time contributors with links to contributing guides | Reduces friction and drop-off for open-source newcomers |
| **[DailyOps](https://github.github.com/gh-aw/patterns/daily-ops/)** | [`daily-ops/stale-pr-nudge.md`](daily-ops/stale-pr-nudge.md) | Daily schedule | Comments on PRs with no activity for 7+ days | Prevents reviews from silently stalling in the backlog |
| **[LabelOps](https://github.github.com/gh-aw/patterns/label-ops/)** | [`label-ops/close-wontfix.md`](label-ops/close-wontfix.md) | `wontfix` label applied | Closes the issue with a contextual explanation | Automates backlog housekeeping without boilerplate comments |
| **[DataOps](https://github.github.com/gh-aw/patterns/data-ops/)** | [`data-ops/weekly-contribution-report.md`](data-ops/weekly-contribution-report.md) | Weekly schedule (Monday 9 AM) | Posts a merged-PR digest with top contributors to Discussions | Gives teams a recurring pulse on contribution activity |
| **[DispatchOps](https://github.github.com/gh-aw/patterns/dispatch-ops/)** | [`dispatch-ops/codebase-qa.md`](dispatch-ops/codebase-qa.md) | `workflow_dispatch` with a question | Answers natural-language questions about the codebase and documents the answer as an issue | Turns one-off Slack questions into persistent documentation |
| **[ProjectOps](https://github.github.com/gh-aw/patterns/project-ops/)** | [`project-ops/board-router.md`](project-ops/board-router.md) | Issue opened or labeled | Adds issues to a project board and sets Priority / Status fields by label | Eliminates manual triage clicks to keep boards accurate |
| **[WorkQueueOps](https://github.github.com/gh-aw/patterns/workqueue-ops/)** | [`workqueue-ops/checklist-processor.md`](workqueue-ops/checklist-processor.md) | `workflow_dispatch` with an issue number | Processes unchecked items in a GitHub issue checklist one batch at a time | Drives long-running migrations or audits across many runs |
| **[MultiRepoOps](https://github.github.com/gh-aw/patterns/multi-repo-ops/)** | [`multi-repo-ops/cross-repo-issue-mirror.md`](multi-repo-ops/cross-repo-issue-mirror.md) | `upstream` label applied | Mirrors an issue to a companion repository for cross-team tracking | Keeps upstream maintainers informed without leaving your repo |
| **[BatchOps](https://github.github.com/gh-aw/patterns/batch-ops/)** | [`batch-ops/bulk-stale-labeler.md`](batch-ops/bulk-stale-labeler.md) | Weekly schedule (Monday 3 AM) | Labels up to 20 inactive issues as `stale` each run | Scales stale-issue hygiene across large backlogs incrementally |

## Usage

```bash
# 1. Copy a starter to your workflow directory
cp starters/chat-ops/pr-explain.md .github/workflows/pr-explain.md

# 2. Edit placeholders (e.g. YOUR-ORG, project URLs, secrets)
#    then compile to generate the lock file
gh aw compile pr-explain

# 3. Trigger it
gh aw run pr-explain
```

> **Placeholders to replace**
> - `YOUR-ORG/YOUR-UPSTREAM-REPO` in `multi-repo-ops/cross-repo-issue-mirror.md`
> - `https://github.com/orgs/YOUR-ORG/projects/1` in `project-ops/board-router.md`
> - Secret names prefixed with `GH_AW_` that require a PAT or cross-repo token
