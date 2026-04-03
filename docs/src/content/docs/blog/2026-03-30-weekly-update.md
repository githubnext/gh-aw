---
title: "Weekly Update – March 30, 2026"
description: "Six releases in seven days: audit superpowers, integrity-aware cache-memory, a serious security sweep, and runner flexibility for compile-stable jobs."
authors:
  - copilot
date: 2026-03-30
---

Six releases shipped in [github/gh-aw](https://github.com/github/gh-aw) between March 24 and March 30 — that's almost one a day. From expanded audit tooling to integrity-isolated cache storage and a wave of security fixes, this was a dense week. Here's the rundown.

## Releases This Week

### [v0.64.4](https://github.com/github/gh-aw/releases/tag/v0.64.4) — March 30

The freshest release ships with quality-of-life wins for workflow authors:

- **`runs-on-slim` for compile-stable jobs** ([#23490](https://github.com/github/gh-aw/pull/23490)): Override the runner for `compile-stable` framework jobs with a new `runs-on-slim` key, giving you fine-grained control over which machine handles compilation.
- **Sibling nested imports fixed** ([#23475](https://github.com/github/gh-aw/pull/23475)): `./file.md` imports now resolve relative to the importing file's directory, not the working directory. Modular workflows that import sibling files were silently broken before — now they're not.
- **Custom tools in `<safe-output-tools>` prompt** ([#23487](https://github.com/github/gh-aw/pull/23487)): Custom jobs, scripts, and actions are now listed in the agent's `<safe-output-tools>` prompt block so the AI actually knows they exist.
- **Compile-time validation of safe-output job ordering** ([#23486](https://github.com/github/gh-aw/pull/23486)): Misconfigured `needs:` ordering on custom safe-output jobs is now caught at compile time.
- **MCP Gateway v0.2.9** ([#23513](https://github.com/github/gh-aw/pull/23513)) and **firewall v0.25.4** ([#23514](https://github.com/github/gh-aw/pull/23514)) bumped for all compiled workflows.

### [v0.64.3](https://github.com/github/gh-aw/releases/tag/v0.64.3) — March 29

A security-heavy release with one major architectural upgrade:

**Integrity-aware cache-memory** is the headline. Cache storage now uses dedicated git branches — `merged`, `approved`, `unapproved`, and `none` — to enforce integrity isolation at the storage level. A run operating at `unapproved` integrity can no longer read data written by a `merged`-integrity run, and any change to your `allow-only` guard policy automatically invalidates stale cache entries. If you upgrade and see a cache miss on your first run, that's intentional — legacy data has no integrity provenance and must be regenerated.

**`patch-format: bundle`** ([#23338](https://github.com/github/gh-aw/pull/23338)) is the other highlight: code-push flows now support `git bundle` as an alternative to `git am`, preserving merge commits, authorship, and per-commit messages that were previously dropped.

Security fixes:
- **Secret env var exclusion** ([#23360](https://github.com/github/gh-aw/pull/23360)): AWF now strips all secret-bearing env vars (tokens, API keys, MCP secrets) from the agent container's visible environment, closing a potential prompt-injection exfiltration path in `pull_request_target` workflows.
- **Argument injection fix** ([#23374](https://github.com/github/gh-aw/pull/23374)): Package and image names in `gh aw compile --validate-packages` are validated before being passed to `npm view`, `pip index versions`, `uv pip show`, and `docker`.

### [v0.64.2](https://github.com/github/gh-aw/releases/tag/v0.64.2) — March 26

The `gh aw logs` command gained cross-run report generation via the new `--format` flag:

**`gh aw logs --format`** aggregates firewall behavior across multiple workflow runs and produces an executive summary, domain inventory, and per-run breakdown:

```bash
gh aw logs agent-task --format markdown --count 10    # Markdown
gh aw logs --format markdown --json                   # JSON for dashboards
gh aw logs --format pretty                            # Console output
```

This release also includes a **YAML env injection security fix** ([#23055](https://github.com/github/gh-aw/pull/23055)): all `env:` emission sites in the compiler now use `%q`-escaped YAML scalars, preventing newlines or quote characters in frontmatter values from injecting sibling env variables into `.lock.yml` files.

### [v0.64.1](https://github.com/github/gh-aw/releases/tag/v0.64.1) — March 26

**`gh aw audit diff`** ([#22996](https://github.com/github/gh-aw/pull/22996)) lets you compare two workflow runs side-by-side — firewall behavior, MCP tool invocations, token usage, and duration — to spot regressions and behavioral drift before they become incidents:

```bash
gh aw audit diff <run1> <run2> --format markdown
```

Five new sections also landed in the standard `gh aw audit` report: Engine Configuration, Prompt Analysis, Session & Agent Performance, Safe Output Summary, and MCP Server Health. One report now gives you the full picture.

### [v0.64.0](https://github.com/github/gh-aw/releases/tag/v0.64.0) — March 25

**Bot-actor concurrency isolation**: Workflows combining `safe-outputs.github-app` with `issue_comment`-capable triggers now automatically get bot-isolated concurrency keys, preventing the workflow from cancelling itself mid-run when the bot posts a comment that re-triggers the same workflow.

### [v0.63.1](https://github.com/github/gh-aw/releases/tag/v0.63.1) — March 24

A focused patch adding the **`skip-if-check-failing`** pre-activation gate — workflows can now bail out before the agent runs if a named CI check is currently failing, avoiding wasted inference on a broken codebase. Also ships an improved fuzzy schedule algorithm with weighted preferred windows and peak avoidance to reduce queue contention on shared runners.

---

## 🤖 Agent of the Week: auto-triage-issues

The self-appointed gatekeeper of the issue tracker — reads every new issue and assigns labels so the right people see it.

This week, `auto-triage-issues` handled three runs. Two of them were textbook efficiency: triggered the moment a new issue landed, ran the pre-activation check, decided there was nothing worth labeling, and wrapped up in under 42 seconds flat. No fuss, no drama. Then came the Monday scheduled sweep. That run went a different direction: 18 turns, 817,000 tokens, and after all that contemplation... a failure. Somewhere between turn one and turn eighteen, the triage workflow decided this batch of issues deserved its most thoughtful analysis yet, burned through a frontier model's patience, and still couldn't quite close the loop.

It's the classic overachiever problem — sometimes the issues that look the simplest turn out to be the ones that take all day.

💡 **Usage tip**: If your `auto-triage-issues` scheduled runs are consistently expensive, the new `agentic_fraction` metric in `gh aw audit` can help you identify which turns are pure data-gathering and could be moved to deterministic shell steps.

→ [View the workflow on GitHub](https://github.com/github/gh-aw/blob/main/.github/workflows/auto-triage-issues.md)

---

## Try It Out

Update to [v0.64.4](https://github.com/github/gh-aw/releases/tag/v0.64.4) today with `gh extension upgrade aw`. The integrity-aware cache-memory migration will trigger a one-time cache miss on first run — expected and safe. As always, questions and contributions are welcome in [github/gh-aw](https://github.com/github/gh-aw).
