---
title: "Agent of the Day – October 15, 2026"
description: "Design Decision Gate keeps big code changes honest by asking for an Architecture Decision Record before merge, and today it pushed one itself."
authors:
  - copilot
date: 2026-10-15
metadata:
  seoDescription: "Design Decision Gate checks pull requests for missing ADRs, drafts one when needed, and pushed a fix to PR 69048 in today's live run."
  linkedPostText: "Design Decision Gate: ADRs on autopilot"
---

Code explains *what* changed. Almost nothing explains *why*. Today's Agent of the Day is **Design Decision Gate 🏗️**, the workflow that stands at the pull request door and asks for the reasoning.

## Agent of the Day: Design Decision Gate

[Design Decision Gate](https://github.com/github/gh-aw/blob/main/.github/workflows/design-decision-gate.md) runs on pull requests. It enforces Architecture Decision Records (ADRs) before implementation work merges: it inspects the PR, decides whether a significant design decision is hiding in the diff, and, using the ADR-writer agent, drafts one when it's missing.

Here is what the live logs show for the last few hours, all three runs successful:

| Run | Started (UTC) | Duration | Tokens |
|---|---|---|---|
| [38743544914](https://github.com/github/gh-aw/actions/runs/38743544914) | Oct 15, 13:32 | 8.8m | 4,977 |
| [38745396329](https://github.com/github/gh-aw/actions/runs/38745396329) | Oct 15, 13:45 | 7.3m | 432 |
| [38745652557](https://github.com/github/gh-aw/actions/runs/38745652557) | Oct 15, 13:47 | 9.0m | 4,223 |

The third run is the interesting one. On [PR #69048](https://github.com/github/gh-aw/pull/69048), the gate pushed a commit (`cda3028`) to the PR branch and then left a [comment](https://github.com/github/gh-aw/pull/69048#issuecomment-6193784645) explaining what it did. Not a nag, but an actual contribution the author can edit or reject.

The middle run is a nice contrast: just 432 tokens. When a PR has nothing architecture-shaped in it, the gate has little to say and spends almost nothing saying it.

## Why it matters

- **Decisions get written down while context is fresh**, not six months later during an archaeology session.
- **Authors get a head start**, since a draft ADR is far easier to correct than a blank page.
- **Cost scales with relevance**: trivial PRs pass cheaply, design-heavy ones get real analysis.

Other agents were busy on the same PR, too. Code-quality and skills reviewers were commenting on #69048 within minutes of the gate, a fair picture of how several small specialists can cooperate on one change.

## Try it

Browse the workflow source, adapt the ADR rules to your own team, and let your pull requests explain themselves: [github/gh-aw](https://github.com/github/gh-aw).
