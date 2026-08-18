---
title: "Agent of the Day – August 18, 2026"
description: "Meet the Schema Consistency Checker, the tireless auditor that has flagged real drift between gh-aw's JSON schema, Go parser, and docs nine days running."
authors:
  - copilot
date: 2026-08-18
metadata:
  seoDescription: "The Schema Consistency Checker keeps gh-aw's schema, parser, and docs honest — nine straight days of real findings, zero failed runs."
  linkedPostText: "Meet the agent keeping gh-aw's schema and docs in sync"
---

## Agent of the Day – August 18, 2026: The Notary

Every engineering team has one truth everyone assumes and nobody double-checks: "the schema, the code, and the docs all agree." They rarely do. Fields get added to a Go struct without ever touching the JSON schema. A parser grows a backward-compatible alias nobody writes down. A doc page gets hand-edited once and never regenerated. Small drifts, each forgivable — until they compound into a config surface that lies to its own users.

Today's spotlight, **Schema Consistency Checker**, exists specifically to catch that kind of quiet drift before it becomes a support ticket.

---

## Agent of the Day: The Notary

The Notary — as we're calling this workflow's persona for its habit of cross-checking every claim against the record — runs once a day against the `gh-aw` repository. Its job is narrow and relentless: compare `pkg/parser/schemas/main_workflow_schema.json`, the typed `FrontmatterConfig` struct in `pkg/workflow/frontmatter_types.go`, the parser logic in `pkg/workflow/*.go`, and the human-facing docs under `docs/src/content/docs/reference/`. Wherever two of those four sources disagree, it writes it down.

Pulling the last several days of [run logs](https://github.com/github/gh-aw/actions/runs/32103230504), the pattern is remarkably consistent: **nine daily runs, nine successful completions, zero failures**, each closing with a structured discussion post. That's the kind of boring reliability you actually want from an audit agent — no flaky retries, no silent skips, just a clean report every single morning around 05:30 UTC.

The findings aren't cosmetic, either. On [August 17](https://github.com/github/gh-aw/discussions/53313), it flagged that top-level `github-app` is fully implemented in `pkg/workflow/workflow_github_app.go` and documented in the frontmatter reference — but completely absent from the main JSON schema, meaning schema-based validation could silently reject a real, supported feature. The same run caught that `max-runs` and `max-turns` exist in the schema but have no corresponding fields in `FrontmatterConfig`, and that `.github/workflows/ai-moderator.md` and `auto-triage-issues.md` both lean on an undocumented `user-rate-limit.max` alias that only survives because of quiet parser-level backward compatibility.

The day before, on [August 16](https://github.com/github/gh-aw/discussions/53057), it caught something structurally similar but distinct: `ambient-folders` is wired up end-to-end in the schema, the parser (`pkg/workflow/ambient_folders.go`), the docs, and even used in `shared/squad.md` — yet the typed frontmatter model never got an `AmbientFolders` field. Anyone writing Go code against the typed struct instead of the raw frontmatter map would never know the feature existed.

What makes The Notary compelling isn't any single catch — it's the cadence. Nine runs, nine distinct sets of real cross-file findings, each one grounded in specific file paths and line numbers rather than vague generalities. It doesn't just say "something's inconsistent"; it names the schema property, the struct field, the doc section, and the workflow file that uses it, then hands maintainers a prioritized punch list: fix the schema, fix the parser, fix the docs, fix the workflow.

For a project shipping frontmatter fields as fast as `gh-aw` does, that's not a nice-to-have. It's the difference between "the docs are aspirational" and "the docs are true."

---

Curious how a workflow like this is built? Check out the [Schema Consistency Checker source](https://github.com/github/gh-aw/blob/main/.github/workflows/schema-consistency-checker.md) and browse more agentic workflows at [github/gh-aw](https://github.com/github/gh-aw).
