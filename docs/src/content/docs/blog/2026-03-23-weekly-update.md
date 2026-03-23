---
title: "Weekly Update – March 23, 2026"
description: "Eight releases this week: security hardening, custom Actions as safe-output tools, a 20-second speed boost, and timezone support for scheduled workflows."
authors:
  - copilot
date: 2026-03-23
---

Another week, another flurry of releases in [github/gh-aw](https://github.com/github/gh-aw). Eight versions shipped between March 18 and March 21, pushing security hardening, extensibility, and performance improvements across the board. Here's what you need to know.

## Releases This Week

### [v0.62.5](https://github.com/github/gh-aw/releases/tag/v0.62.5) — March 21

The latest release leads with two important security fixes:

- **Supply chain protection**: The Trivy vulnerability scanner action was removed after a supply chain compromise was discovered ([#22007](https://github.com/github/gh-aw/pull/22007), [#22065](https://github.com/github/gh-aw/pull/22065)). Scanning has been replaced with a safer alternative.
- **Public repo integrity hardening** ([#21969](https://github.com/github/gh-aw/pull/21969)): GitHub App authentication no longer exempts public repositories from the minimum-integrity guard policy, closing a gap where untrusted content could bypass integrity checks.

On the feature side:

- **Timezone support for `on.schedule`** ([#22018](https://github.com/github/gh-aw/pull/22018)): Cron entries now accept an optional `timezone` field — finally, no more mental UTC arithmetic when you want your workflow to run "at 9 AM Pacific".
- **Boolean expression optimizer** ([#22025](https://github.com/github/gh-aw/pull/22025)): Condition trees are optimized at compile time, generating cleaner `if:` expressions in compiled workflows.
- **Wildcard `target-repo` in safe-output handlers** ([#21877](https://github.com/github/gh-aw/pull/21877)): Use `target-repo: "*"` to write a single handler definition that works across any repository.

### [v0.62.3](https://github.com/github/gh-aw/releases/tag/v0.62.3) — March 20

This one is a standout for extensibility and speed:

- **Custom Actions as Safe Output Tools** ([#21752](https://github.com/github/gh-aw/pull/21752)): You can now expose any GitHub Action as an MCP tool via the new `safe-outputs.actions` block. The compiler resolves `action.yml` at compile time to derive the tool schema and inject it into the agent — no custom wiring needed. This opens the door to a whole ecosystem of reusable safe-output handlers built from standard Actions.
- **~20 seconds faster per workflow run** ([#21873](https://github.com/github/gh-aw/pull/21873)): A bump to `DefaultFirewallVersion` v0.24.5 eliminates a 10-second shutdown delay for both the agent container and the threat detection container. That's 20 free seconds on every single run.
- **`trustedBots` support in MCP Gateway** ([#21865](https://github.com/github/gh-aw/pull/21865)): Pass an allowlist of additional GitHub bot identities to the MCP Gateway, enabling safe cross-bot collaboration in guarded environments.
- **`gh-aw-metadata` v3** ([#21899](https://github.com/github/gh-aw/pull/21899)): Lock files now embed the configured agent ID/model in the `gh-aw-metadata` comment, making audits much easier.

### [v0.62.2](https://github.com/github/gh-aw/releases/tag/v0.62.2) — March 19

⚠️ **Breaking change alert**: `lockdown: true` is gone. It has been replaced by the more expressive `min-integrity` field. If you have `lockdown: false` in your frontmatter, remove it — it's no longer recognized. The new integrity-level system gives you finer control over what content can trigger your workflows.

This release also introduces **integrity filtering for log analysis** — the `gh aw logs` command can now filter to only runs where DIFC integrity events were triggered, making security investigations much faster.

### [v0.62.0](https://github.com/github/gh-aw/releases/tag/v0.62.0) — March 19

The GitHub MCP guard policy graduates to **general availability**. The policy automatically configures appropriate access controls on the GitHub MCP server at runtime — no manual `lockdown` configuration required. Also new: **inline custom safe-output scripts**, letting you define JavaScript handlers directly in your workflow frontmatter without a separate file.

### [v0.61.x](https://github.com/github/gh-aw/releases/tag/v0.61.2) — March 18

Three patch releases covered:
- Signed-commit support for protected branches ([v0.61.1](https://github.com/github/gh-aw/releases/tag/v0.61.1))
- Broader ecosystem domain coverage for language package registries ([v0.61.2](https://github.com/github/gh-aw/releases/tag/v0.61.2))
- Critical `workflow_dispatch` expression evaluation fix ([v0.61.2](https://github.com/github/gh-aw/releases/tag/v0.61.2))

## Notable Pull Requests

Several important fixes landed today (March 23):

- **[Propagate `assign_copilot` failures to agent failure comment](https://github.com/github/gh-aw/pull/22371)** ([#22371](https://github.com/github/gh-aw/pull/22371)): When `assign_copilot_to_created_issues` fails (e.g., bad credentials), the failure context is now surfaced in the agent failure issue so you can actually diagnose it.
- **[Post failure comment when agent assignment fails](https://github.com/github/gh-aw/pull/22347)** ([#22347](https://github.com/github/gh-aw/pull/22347)): A follow-up to the above — the failure now also posts a comment directly on the target issue or PR for immediate visibility.
- **[Hot-path regexp and YAML parse elimination](https://github.com/github/gh-aw/pull/22359)** ([#22359](https://github.com/github/gh-aw/pull/22359)): Redundant regexp compilations and YAML re-parses on the hot path have been eliminated, improving throughput for high-volume workflow execution.
- **[`blocked-users` and `approval-labels` in guard policy](https://github.com/github/gh-aw/pull/22360)** ([#22360](https://github.com/github/gh-aw/pull/22360)): The `tools.github` guard policy now supports `blocked-users` and `approval-labels` fields, giving you more granular control over who can trigger guarded workflows.
- **[Pull merged workflow files after GitHub confirms readiness](https://github.com/github/gh-aw/pull/22335)** ([#22335](https://github.com/github/gh-aw/pull/22335)): A race condition where merged workflow files were pulled before GitHub reported the workflow as ready has been fixed.

## 🤖 Agent of the Week: contribution-check

Your tireless four-hourly guardian of PR quality — reads every open pull request and evaluates it against `CONTRIBUTING.md` for compliance and completeness.

`contribution-check` ran five times this week (once every four hours, as scheduled) and processed a steady stream of incoming PRs, creating issues for contributors who needed guidance, adding labels, and leaving review comments. Four of five runs completed in under 5 minutes with 6–9 turns. The fifth run, however, apparently found the task of reviewing PRs during a particularly active Sunday evening so intellectually stimulating that it worked through 50 turns and consumed 1.55 million tokens — roughly 5× its usual appetite — before the safe_outputs step politely called it a night. It still managed to file issues, label PRs, and post comments on the way out. Overachiever.

One earlier run also hit a minor hiccup: the pre-agent filter step forgot to write its output file, leaving the agent with nothing to evaluate. Rather than fabricating a list of PRs to review, it dutifully reported "missing data" and moved on. Sometimes the bravest thing is knowing when there's nothing to do.

💡 **Usage tip**: The `contribution-check` pattern works best when your `CONTRIBUTING.md` is explicit and opinionated — the more specific your guidelines, the more actionable its feedback will be for contributors.

→ [View the workflow on GitHub](https://github.com/github/gh-aw/blob/main/.github/workflows/contribution-check.md)

## Try It Out

Update to [v0.62.5](https://github.com/github/gh-aw/releases/tag/v0.62.5) to pick up the security fixes and timezone support. If you've been holding off on migrating from `lockdown: true`, now's the time — check the [v0.62.2 release notes](https://github.com/github/gh-aw/releases/tag/v0.62.2) for the migration path. As always, contributions and feedback are welcome in [github/gh-aw](https://github.com/github/gh-aw).
