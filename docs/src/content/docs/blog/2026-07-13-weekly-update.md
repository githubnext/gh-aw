---
title: "Weekly Update – July 13, 2026"
description: "v0.82.8 ships gVisor sandbox isolation, docker-sbx runtime, private-to-public-flows control, and sbx credential refresh fixes."
authors:
  - copilot
date: 2026-07-13
metadata:
  seoDescription: "gh-aw v0.82.8: gVisor runtime, docker-sbx support, private-to-public-flows frontmatter, and Docker credential refresh for sbx workflows."
---

Another active week in [github/gh-aw](https://github.com/github/gh-aw)! We shipped [v0.82.8](https://github.com/github/gh-aw/releases/tag/v0.82.8), landed several impactful features, and squashed a frustrating Docker authentication bug that had been interrupting `sbx`-runtime workflows.

## Release: v0.82.8

[v0.82.8](https://github.com/github/gh-aw/releases/tag/v0.82.8) published on July 11th with a broad set of reliability and security improvements.

### What's New

- **gVisor container runtime** ([#44796](https://github.com/github/gh-aw/pull/44796)): Set `sandbox.agent.runtime: gvisor` in your workflow frontmatter to run the agent inside a gVisor sandbox for stronger isolation — great for workflows processing untrusted input.

- **Shared partials can declare `sandbox.agent.mounts`** ([#44500](https://github.com/github/gh-aw/pull/44500)): Partial workflow files can now define mount configurations that get merged into the parent, enabling reusable sandbox setups without copy-paste.

- **AI authorship disclosure header** ([#44497](https://github.com/github/gh-aw/pull/44497)): A new `disclosure-header` safe-output message type lets agents declare AI authorship inline in PR comments and issues.

- **`gh aw add` resolves transitive `uses:` references** ([#44763](https://github.com/github/gh-aw/pull/44763)): Importing a workflow partial now automatically pulls in any nested imports — no more manual dependency hunting.

- **OAuth token failures surface in conclusion job** ([#44777](https://github.com/github/gh-aw/pull/44777), [#44756](https://github.com/github/gh-aw/pull/44756)): Token failures are no longer silently swallowed — they now show up where you'd expect.

## Notable Pull Requests This Week

- **[docker-sbx runtime support](https://github.com/github/gh-aw/pull/45006)** — You can now run your agent inside a KVM-isolated Docker sbx microVM (`sandbox.agent.runtime: docker-sbx`) while keeping infrastructure containers on the host. Full hardware-virtualization isolation for workloads that need it.

- **[Emit sbx credential refresh before agent execution](https://github.com/github/gh-aw/pull/45146)** — Fixes those maddening intermittent `"user is not authenticated to Docker"` errors. Docker Hub OAuth tokens from the daemon-setup step could expire by the time the agent ran. Now a fresh `sbx login` runs immediately before agent execution for all `sbx`-runtime workflows.

- **[`private-to-public-flows: allow` frontmatter field](https://github.com/github/gh-aw/pull/45113)** — Wires the full frontmatter → struct → gateway JSON pipeline for `tools.github.private-to-public-flows`, letting you opt specific MCP servers out of `sink-visibility` enforcement when you explicitly trust those flows.

- **[Bump gVisor release to 20250707.0](https://github.com/github/gh-aw/pull/45101)** — Keeps the pinned gVisor release current with upstream security and reliability patches.

- **[Add missing copilot safe-output fixture files](https://github.com/github/gh-aw/pull/45004)** — Adds fixtures for `close-discussion`, `assign-to-agent`, `assign-to-user`, and `unassign-from-user`, filling gaps in the safe-output test suite.

## 🤖 Agent of the Week: aw-failure-investigator

Your on-call teammate who never sleeps — it wakes up every 6 hours, scans recent workflow run failures, and files GitHub issues so problems don't fall through the cracks.

This week `aw-failure-investigator` ran three times across July 11–12, filing 3 issues in total (2 in one run, 1 in another). Each run clocked in around 15 minutes and consumed 250+ AI credits running on claude-opus-4-8 — because when you're investigating failures, you don't want to cut corners. The July 11th run had its own failure (meta!), but bounced back cleanly on its next scheduled cycle.

In one particularly busy shift it made 13 GitHub API calls in 15 minutes, which is either impressive efficiency or evidence that it found a lot to worry about. Probably both.

💡 **Usage tip**: Pair `aw-failure-investigator` with a label-based notification rule so the right team gets pinged when it files an issue — that way failures surface asynchronously without requiring anyone to watch the Actions tab.

→ [View the workflow on GitHub](https://github.com/github/gh-aw/blob/main/.github/workflows/aw-failure-investigator.md)

## Try It Out

Update to [v0.82.8](https://github.com/github/gh-aw/releases/tag/v0.82.8) and explore the new `docker-sbx` and `gvisor` sandbox runtimes. If you've been hitting Docker auth errors on `sbx` workflows, the credential refresh fix should put those to rest. Contributions and feedback are always welcome at [github/gh-aw](https://github.com/github/gh-aw).
