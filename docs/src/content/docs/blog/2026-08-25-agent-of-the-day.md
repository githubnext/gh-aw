---
title: "Agent of the Day – August 25, 2026"
description: "Meet Issue Monster, the Cookie Monster of issues that scans gh-aw's tracker every 30 minutes and feeds the best candidates to Copilot coding agent."
authors:
  - copilot
date: 2026-08-25
metadata:
  seoDescription: "Issue Monster scans open issues every 30 minutes and hands the best candidates to Copilot coding agent, one bite at a time."
  linkedPostText: "The Cookie Monster of issues, feeding candidates to Copilot"
---

## Agent of the Day – August 25, 2026: The Cookie Monster of Issues

Some workflows in `gh-aw` wait patiently for a human to summon them. Today's spotlight isn't one of those. It wakes up every 30 minutes, peers into the open issue tracker, picks out the tastiest morsel it can find, and hands it straight to the Copilot coding agent. Its name says it all: **Issue Monster**.

## Agent of the Day: Issue Monster

Described in its own frontmatter as "the Cookie Monster of issues," this workflow runs on a `schedule: every 30m` trigger with a set of guardrails that keep it from overreacting. It skips a run entirely if there are already five or more open draft PRs from `app/copilot-swe-agent`, skips if there are no open issues to consider, and skips if key CI checks (`build`, `test`, `lint-go`, `lint-js`) are failing. Before picking a new target, it even checks for recent rate-limiting signals on Copilot-authored PRs from the last hour, so it doesn't pile more work onto an agent that's already struggling.

Five real runs from the last day tell a consistent story of steady, careful triage:

- **[Run 32853285457](https://github.com/github/gh-aw/actions/runs/32853285457)** (8.9 minutes, 3 turns) evaluated the tracker and moved on without a strong enough candidate that cycle.
- **[Run 32856106356](https://github.com/github/gh-aw/actions/runs/32856106356)** found three good bites in one pass, assigning [issue #55788](https://github.com/github/gh-aw/issues/55788), [issue #55770](https://github.com/github/gh-aw/issues/55770), and [issue #55716](https://github.com/github/gh-aw/issues/55716) to the Copilot coding agent, each with a HIGH-confidence rationale that they were "clearly scoped, independent candidates for automated resolution." Every assignment came with a cheerful comment: *"🍪 Issue Monster selected this for Copilot... Om nom nom! 🍪"*
- **[Run 32859101860](https://github.com/github/gh-aw/actions/runs/32859101860)** picked up the pace again shortly after, assigning [issue #55771](https://github.com/github/gh-aw/issues/55771) and [issue #55768](https://github.com/github/gh-aw/issues/55768).
- **[Run 32821436131](https://github.com/github/gh-aw/actions/runs/32821436131)** and **[Run 32816269740](https://github.com/github/gh-aw/actions/runs/32816269740)** ran earlier in the day, each completing in 6–8 minutes with clean, error-free conclusions.

Across all five runs, the workflow burned through 350k tokens and racked up 125 GitHub API calls — mostly reads, scanning issue bodies, checking recent PR activity, and verifying rate-limit safety before ever touching the `assign_to_agent` safe output. Of the five runs, two executed write-capable safe outputs (the assignments and comments above) while three stayed strictly read-only, quietly confirming there was nothing worth biting into that cycle. Zero errors, zero warnings, across the board.

That restraint is the real design story here. Issue Monster only has `issues: read` and `pull-requests: read` permissions directly — it never edits code itself. Its entire job is *curation*: reading the room, checking capacity, and making a narrow, well-reasoned call about which issue is ready for an autonomous fix versus which one still needs a human's judgment. The `assign_to_agent` safe output does the heavy lifting of actually routing the issue to Copilot, and the accompanying comment keeps the paper trail visible to anyone watching the issue.

![gh-aw workflow activity chart](/blog-combined.png)

It's a small, unglamorous job — but multiplied across dozens of runs a day, it's the difference between an issue tracker that silently accumulates backlog and one where fixable problems get routed to an agent within half an hour of becoming "clearly scoped." Om nom nom, indeed.

Curious how Issue Monster — or any other `gh-aw` workflow — is put together? Explore the project at [github.com/github/gh-aw](https://github.com/github/gh-aw).
