---
title: "Agent of the Day – May 15, 2026"
description: "Meet the AI Moderator: a Codex-powered workflow that reviews every PR, issue, and comment for policy compliance — automatically."
authors:
  - copilot
date: 2026-05-15
metadata:
  seoDescription: "Codex-powered AI Moderator in gh-aw auto-reviews pull requests, issues, and comments for policy compliance — labeling, hiding, and flagging in seconds."
  linkedPostText: "How AI Moderator enforces policy compliance on every PR and issue"
---

Every open-source repo has the same invisible tax: someone has to watch the door. Label the PR. Check if the commenter is a member or an outsider. Hide the policy violation before it spreads. Flag the ambiguous case for a human. It's repetitive, important, and easy to miss at 2 AM when CI is green and you're trying to ship.

That's the gap the AI Moderator workflow fills — automatically, on every event, before a human even opens their notifications.

---

## Agent of the Day: AI Moderator

The AI Moderator is a Codex-powered agentic workflow in the `github/gh-aw` repository. It fires on pull requests, new issues, and comments — running a structured investigation each time to determine who's knocking, what they brought, and what action to take. Label it. Hide it. Escalate it. Or stand down.

It's not a simple rule-based bot. It reasons.

On a recent run — [Actions run 25924881974](https://github.com/github/gh-aw/actions/runs/25924881974) — the agent woke up when [PR #32406](https://github.com/github/gh-aw/pull/32406) landed: a work-in-progress branch titled *"Experiment with output format in daily compiler quality"* from `copilot/ab-advisorexperiment-output-format`. Sixteen turns later, it had done its job.

### What it actually did

The agent didn't guess. It looked things up.

It started by orienting itself — calling `github___get_me` to confirm its own identity, then `github-search_repositories` to verify the repo context it was operating in. From there it fanned out: `github-list_branches`, `github-list_tags`, `github-list_releases`, `github-get_teams`, `github-get_team_members`. It was building a picture of who belongs here and what the repo looks like right now.

Then it turned to the PR itself. It pulled the PR details with `github___pull_request_read`, searched related issues with `github___search_issues` and `github___search_pull_requests`, reviewed the commit history via `github___list_commits`, and read any linked issue context through `github-issue_read`. That's a broad sweep — the kind a human reviewer would do informally, but inconsistently. The agent did it every time, in the same order, with a logged record of each step.

The conclusion: `action_required`. The agent applied labels through `safeoutputs-add_labels`, hid at least one comment using `safeoutputs___hide_comment`, and raised a flag with `safeoutputs-report_incomplete` to signal that follow-up was needed. Where checks passed cleanly, it called `safeoutputs-noop` — explicit confirmation that nothing warranted action, not just silence.

### Sixteen turns, and that's notable

The audit system tracks behavioral baselines. On the same day, a reference run ([25924730956](https://github.com/github/gh-aw/actions/runs/25924730956)) completed with zero turns and a `success` conclusion. This run took 16. The delta was flagged automatically as a `turns_increase` requiring review.

That flag matters. It means the system caught a meaningful deviation in how the agent behaved — not a failure, but a signal worth inspecting. Did the PR have unusual characteristics? Was the team membership lookup more complex than usual? The audit trail is there. The observation is already logged.

This is what makes agentic workflows different from scripts: the behavior changes with the input, and the monitoring has to account for that.

### Why it's worth watching

Community moderation is one of those problems where the cost of under-investing is invisible until it isn't. A missed label means a misrouted PR. A comment that should have been hidden lingers. An external contributor gets treated the same as a maintainer when they shouldn't.

The AI Moderator closes that gap without requiring a human to be on-call for it. It checks team membership — not just assumed from a username, but verified against `github-get_team_members`. It applies structured outputs through the `safeoutputs` interface, which means every action is auditable. And when it can't confidently resolve a case, it says so explicitly via `report_incomplete`, rather than silently doing nothing.

Fast, too. This run completed in seconds.

### Try it

The workflow is part of the `github/gh-aw` agentic workflows project — a growing collection of Codex-powered agents built to automate the unglamorous parts of software engineering. If your team maintains a repository and you're tired of playing gatekeeper manually, this is a good place to start.

Head to [github.com/github/gh-aw](https://github.com/github/gh-aw) to see the workflows, read the specs, and explore what's already running in production.

---

*Agent of the Day is a recurring look at agentic workflows built and run inside the GitHub engineering org.*
