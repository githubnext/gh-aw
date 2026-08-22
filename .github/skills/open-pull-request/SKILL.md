---
name: open-pull-request
description: Publish validated gh-aw changes as a draft pull request from a cloud agent.
generated-by: forge-agent
---

# Open Pull Request

Publish the current gh-aw branch as a draft pull request from a GitHub Copilot cloud agent after the repository's required validation and push steps have completed.

## Purpose

Package the repository's repeated publish flow into one reusable skill so agents stop improvising `gh pr create` sequences and consistently use local validation, `report_progress`, and the runtime pull-request tool.

## Conditions (C)

Use this skill when the current branch contains the intended changes, the run can call both `report_progress` and `create_pull_request`, and the task is to open or update one gh-aw pull request rather than finish an existing review cycle.

## Interface (R)

Inputs: a PR title, summary, optional draft flag, and any repository-specific validation notes. Outputs: pushed branch state via `report_progress`, one created or updated draft pull request, and a short publication summary that names the validation that ran.

## Policy (π)

Before publishing, inspect the repository for a pull-request template and follow it when present. Run repository-specific preparation before the final push: `make fmt` after Go changes, `make recompile` after workflow markdown changes, `make agent-report-progress-no-test` before any intermediate `report_progress`, and `make agent-report-progress` before the final `report_progress`. Use `report_progress` to commit and push the branch, then call `create_pull_request` exactly once to open the PR. Prefer draft mode by default. Never use `gh pr create`, direct `git push`, or ad hoc publication commands from bash in cloud-agent runs.

## Termination (T)

Success means the working tree is published through `report_progress`, the final validation target finished without unresolved errors, and exactly one draft pull request exists for the branch with a body that summarizes changes and validation. Stop after publication; use `pr-finisher` for review-thread or merge-readiness work.

## Always do

- Check for a PR template before composing the body.
- Mention the exact validation commands or targets that passed.
- Keep publication to one pull request for the current branch.
- Use draft mode unless the user explicitly wants a ready-for-review PR.

## Never do

- Never use `gh pr create` or direct Git pushes from bash.
- Never skip the final `make agent-report-progress` before the final publish push.
- Never open multiple PRs for one branch in one run.
- Never treat this skill as a replacement for review handling or merge execution.

## Gotchas / edge cases

- `report_progress` pushes changes but does not open a pull request.
- `create_pull_request` opens the PR but does not run validation or push files.
- Cloud-agent pushes do not re-trigger CI automatically, so local make targets are the authoritative pre-publication signal.
- Missing PR templates are acceptable; fall back to a concise summary and validation section.

## Assets and scripts

- `AGENTS.md`
- `.github/skills/developer/SKILL.md`
- `Makefile` targets `make fmt`, `make recompile`, `make agent-report-progress-no-test`, and `make agent-report-progress`
- Runtime tools `report_progress` and `create_pull_request`
- `.github/skills/pr-finisher/SKILL.md` for post-publication follow-up

## Scope boundaries

This skill covers publication of a new or updated pull request for the current branch. It does not decide feature scope, resolve review comments, re-run CI, or merge the pull request.

**Abstraction level:** compositional
