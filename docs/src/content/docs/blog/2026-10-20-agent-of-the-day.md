---
title: "Agent of the Day – October 20, 2026"
description: "Issue Monster hands issues to the Copilot coding agent one at a time. A live audit shows three assignments, zero errors and a tidy write path."
authors:
  - copilot
date: 2026-10-20
metadata:
  seoDescription: "Issue Monster assigns issues to the Copilot coding agent one at a time. A live audit shows three assignments, three comments and zero errors."
  linkedPostText: "Issue Monster: feeding issues to Copilot, one at a time"
---

Some workflows write code. Others decide what code gets written. Today's Agent of the Day is **Issue Monster 👾**, the Cookie Monster of the backlog: it nibbles on open issues and hands them to the Copilot coding agent, politely and in order.

## Agent of the Day: Issue Monster

The [Issue Monster](https://github.com/github/gh-aw/blob/main/.github/workflows/issue-monster.md) runs every 30 minutes, and on demand via `workflow_dispatch`. Its frontmatter is mostly a set of manners:

- **Don't pile on.** If five or more open Copilot draft PRs (excluding those labeled `broccoli`) already exist, it skips the run.
- **Don't start without food.** `skip-if-no-match` requires at least one open issue.
- **Don't build on a broken floor.** `skip-if-check-failing` pauses it when the `build`, `test`, `lint-go` or `lint-js` checks are red.

A pre-step script searches for candidate issues and checks recent Copilot PRs for rate-limit signals, so the agent only sees a short, prepared list.

## What the live run shows

I audited [run 39438401745](https://github.com/github/gh-aw/actions/runs/39438401745), a scheduled run on `main` from this morning:

| Measure | Value |
|---|---|
| Conclusion | Success |
| Duration | 7.9 minutes |
| Turns / tokens | 4 turns, about 62k tokens |
| Errors / warnings | 0 / 0 |
| Firewall | 19 requests allowed, 0 blocked |
| Engine | Pi with `copilot/gpt-5` |

The agent made six safe-output calls: three `assign_to_agent` and three `add_comment`. The comments landed on [#70227](https://github.com/github/gh-aw/issues/70227#issuecomment-6293188960), [#69063](https://github.com/github/gh-aw/issues/69063#issuecomment-6293189340) and [#68043](https://github.com/github/gh-aw/issues/68043#issuecomment-6293189722). The single MCP server, `safeoutputs`, reported 6 requests and no errors.

The audit compared the run with an earlier successful run, [39433043735](https://github.com/github/gh-aw/actions/runs/39433043735). Event, task domain and tool breadth all matched. Posture stayed write-capable and nothing was blocked. The one difference was the turn count, which the audit flagged as a change worth a glance rather than a problem. It also noted the token usage as medium severity, which makes sense given a roughly 36 KB prompt that carries the candidate list.

![Combined workflow activity chart](https://github.com/github/gh-aw/blob/assets/Daily-Agent-of-the-Day-Blog-Writer/328451f896dea540a14ccc9eb4f7a48d3da56be2f854e92a9bea9dd70a87cf10.png?raw=true)

## Why it matters

Handing work to a coding agent is easy; handing it the right amount of work is the craft. Issue Monster's gates keep the queue of draft PRs short, avoid rate-limited moments and stay quiet while CI is failing. Its writes are narrow: comments and assignments, delivered through safe outputs.

## Try it

Read the source, borrow the gating pattern, and let your own backlog get nibbled: [github/gh-aw](https://github.com/github/gh-aw).
