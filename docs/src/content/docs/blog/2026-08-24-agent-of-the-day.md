---
title: "Agent of the Day – August 24, 2026"
description: "Meet Q, the on-demand workflow doctor that answers /q comments in gh-aw and turns real complaints into pull requests."
authors:
  - copilot
date: 2026-08-24
metadata:
  seoDescription: "Q reads real /q slash-commands in gh-aw discussions and issues, then opens PRs to fix and tune workflows."
  linkedPostText: "The workflow doctor that answers /q on demand"
---

## Agent of the Day – August 24, 2026: The Workflow Doctor

Most agentic workflows in `gh-aw` run on a timer, quietly doing their thing every day whether anyone's watching or not. Today's spotlight is different: it only shows up when you call it. Type `/q` in a comment on an issue, pull request, or discussion, and this workflow wakes up, reads the room, and goes to work fixing whatever you pointed it at.

## Agent of the Day: The Workflow Doctor

We're calling this persona **The Workflow Doctor**, and it belongs to **Q**, a slash-command-triggered `gh-aw` workflow described in its own frontmatter as an "intelligent assistant that answers questions, analyzes repositories, and can create PRs for workflow optimizations." Q runs on the Copilot engine with SDK mode enabled, has read access to issues, pull requests, and discussions, and — critically — is under a hard rule never to touch its own definition file (`q.md`). It exists purely to diagnose and improve *other* workflows in the repo.

Three real runs from the last few days show exactly how varied its case load gets:

- **[Run 32726221560](https://github.com/github/gh-aw/actions/runs/32726221560)** fired from a comment on [discussion #55296](https://github.com/github/gh-aw/discussions/55296), where a maintainer asked Q to "add a job that tests the github MCP in remote mode without using any agentic workflow feature" as a canary test to rule out a runtime/compiler bug, plus a summary of the MCP handshake message. Q completed in 11.5 minutes across a single turn, burning 25.8k tokens, and wrapped up with a successful conclusion and a proposed pull request queued up.
- **[Run 32727642327](https://github.com/github/gh-aw/actions/runs/32727642327)** answered a comment on [issue #55389](https://github.com/github/gh-aw/issues/55389) asking Q to "use mai flash model to reduce cost" — a straightforward cost-tuning request that Q turned into a workflow-level model swap.
- **[Run 32726004596](https://github.com/github/gh-aw/actions/runs/32726004596)** came from [discussion #55334](https://github.com/github/gh-aw/discussions/55334), where the ask was to "update to use repo-memory to store the mined loops" — plumbing persistent state into a workflow that was previously stateless between runs.

Audit data on the discussion-triggered run classifies its behavior fingerprint as `directed` execution with `narrow` tool breadth and a `selective_write` actuation style — in plain terms, Q doesn't wander. It reads exactly what it needs (the triggering comment, the parent issue or discussion, recent logs and audits for the target workflow), forms a specific diagnosis, and proposes a scoped pull request through its `create-pull-request` safe output, complete with a `[q]` title prefix, `automation` and `workflow-optimization` labels, and Copilot as the default reviewer.

That safe-outputs configuration is worth calling out on its own: PRs expire after 2 days if unmerged, patches are capped at 500 files, and protected-file edits automatically fall back to filing an issue instead of silently failing. It's a small but deliberate guardrail set for a workflow that has write access to propose changes across the entire repo's workflow surface — tight enough to keep blast radius small, generous enough to let Q actually fix things.

The interesting part isn't any single fix — it's the range. In three runs pulled from the same short window, Q handled a low-level infrastructure canary test, a cost-optimization tweak, and a state-persistence upgrade, each triggered by a different person from a different corner of the repository. That's the value proposition of an on-demand workflow doctor: no scheduling, no queue, just `/q` and a clear ask.

Want to see how Q — or any other `gh-aw` workflow — is built? Explore the project at [github.com/github/gh-aw](https://github.com/github/gh-aw).
