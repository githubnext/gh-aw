---
title: "Agent of the Day – August 20, 2026"
description: "Meet Issue Arborist, the daily workflow that quietly turns gh-aw's sprawling issue tracker into a navigable tree."
authors:
  - copilot
date: 2026-08-20
metadata:
  seoDescription: "Issue Arborist scans 100 open issues a day, links symptom issues to root causes, and publishes its reasoning in public."
  linkedPostText: "The agent pruning gh-aw's issue tracker into a tidy tree, one link at a time"
---

## Agent of the Day – August 20, 2026: The Gardener

Every fast-moving open source repo eventually grows the same problem: hundreds of open issues, some clearly related, most not linked to each other at all. A container-scan finding sits next to its parent burn-down tracker with no connection. A workflow-failure symptom issue never gets tied back to the root cause that explains it. Humans could do this triage work, but it's tedious, repetitive, and easy to defer forever. Today's spotlight exists to do exactly that triage — every single day, without getting bored.

## Agent of the Day: The Gardener

We're calling this workflow's persona **The Gardener**, and the name fits **Issue Arborist**, a scheduled `gh-aw` workflow that scans the 100 most recent open issues without a parent, looks for orphan clusters and symptom/root-cause pairs, and either links them as sub-issues or creates a new parent when a cluster is big enough to deserve one.

What makes the Gardener trustworthy isn't just that it links issues — it's *how conservative* it is about doing so. In its [August 17 run](https://github.com/github/gh-aw/actions/runs/31997474928), it reviewed 100 open issues and found seven confident matches, each with an explicit citation for why the link was safe to make:

- [#53269](https://github.com/github/gh-aw/issues/53269) and [#53270](https://github.com/github/gh-aw/issues/53270) linked under [#53268](https://github.com/github/gh-aw/issues/53268) (`lint-monster: function-length refactoring`) — both child issues explicitly described themselves as slices of that parent's backlog.
- [#52723](https://github.com/github/gh-aw/issues/52723) linked under [#53049](https://github.com/github/gh-aw/issues/53049), a safe-outputs reliability parent that already referenced it by name.
- [#52652](https://github.com/github/gh-aw/issues/52652) linked under [#52657](https://github.com/github/gh-aw/issues/52657), a container CVE burn-down tracker.
- [#53245](https://github.com/github/gh-aw/issues/53245) and [#53235](https://github.com/github/gh-aw/issues/53235) linked under [#53263](https://github.com/github/gh-aw/issues/53263) — a root-cause issue about `safe_outputs` hard-failing entire batches — because both symptom issues cited the exact same failed run IDs the root cause called out.
- [#53193](https://github.com/github/gh-aw/issues/53193) linked under [#53262](https://github.com/github/gh-aw/issues/53262) for the same reason: matching run IDs across a root-cause and a symptom report.

Just as telling is what it *didn't* link. The same run flagged four newer container-scan issues ([#53071](https://github.com/github/gh-aw/issues/53071), [#53072](https://github.com/github/gh-aw/issues/53072), [#53073](https://github.com/github/gh-aw/issues/53073), [#53075](https://github.com/github/gh-aw/issues/53075)) and one more ([#52858](https://github.com/github/gh-aw/issues/52858)) as *probably* related to the CVE burn-down parent — but held back because it wasn't confident whether maintainers wanted one rolling child per image or a daily detail issue per scan. It also noted #53263 might belong under the broader safe-outputs backlog #53049, but again declined to guess. Every decision — made and skipped — gets published as a public daily discussion report, so maintainers can see the reasoning, not just the result.

Across its last five tracked runs (roughly August 11–17), Issue Arborist has been remarkably consistent: five successful runs, zero errors, zero warnings, averaging around 8 minutes of runtime and creating 6–8 safe-output items each time, all classified as normal, uneventful automation. No orphan cluster in that window was ever large enough to justify spinning up a brand-new parent issue from scratch — which is itself a useful signal that gh-aw's existing issue hierarchy is holding up reasonably well.

It's a small, unglamorous job — but multiply "quietly link the right issues together" by 365 days a year, and you get an issue tracker that stays legible instead of turning into an unsearchable pile. That's the kind of maintenance work that's easy to skip and expensive to have skipped.

Want to see how workflows like this one are built? Check out [github/gh-aw](https://github.com/github/gh-aw).
