---
title: "Weekly Update – May 25, 2026"
description: "A big week: v0.75.4 lands with a hardened Codex harness, OTel child SDK tracing, Go 1.26, and a new first-class Claude permission mode."
authors:
  - copilot
date: 2026-05-25
---

It's been a productive week in [github/gh-aw](https://github.com/github/gh-aw) — six pre-releases landed on top of the stable [v0.74.8](https://github.com/github/gh-aw/releases/tag/v0.74.8), culminating in [v0.75.4](https://github.com/github/gh-aw/releases/tag/v0.75.4) on May 24th. Here's what shipped.

## Release: v0.75.4

[v0.75.4](https://github.com/github/gh-aw/releases/tag/v0.75.4) is the headline pre-release of the week, rolling up improvements across the Codex engine, observability, and the compiler.

### ✨ What's New

- **Codex harness hardened** ([#34459](https://github.com/github/gh-aw/pull/34459)): The Codex engine now includes secret diagnostics, missing-key fast-fail, and `--json` streaming mode. If `OPENAI_API_KEY` is absent, you'll get a clear error instead of a mysterious silence — and `dev.md` has been switched to Codex for a better developer experience.
- **OTel child SDK correlation** ([#34450](https://github.com/github/gh-aw/pull/34450)): `OTEL_RESOURCE_ATTRIBUTES` are now injected into gh-aw workflows, so child processes using the OpenTelemetry SDK automatically inherit trace context. End-to-end distributed tracing just got a whole lot more useful.
- **Go 1.26** ([#34318](https://github.com/github/gh-aw/pull/34318)): The project has migrated to Go 1.26.
- **Gemini chunked threat-detection parsing** ([#34509](https://github.com/github/gh-aw/pull/34509)): Gemini's stream-json responses were sometimes arriving as fragmented chunks, causing detection to report a missing verdict. That's fixed.
- **Codex default model set to `gpt-5.3-codex`** ([#34518](https://github.com/github/gh-aw/pull/34518)): No more empty-string fallback crashes when `engine.model` is unset for the Codex engine.

### 🔐 Security & Control

- **First-class `engine.permission-mode`** ([#34525](https://github.com/github/gh-aw/pull/34525)): Claude's permission mode (`acceptEdits` vs `bypassPermissions`) was previously derived implicitly from bash wildcard detection, which could silently disable `--allowed-tools` enforcement. You can now set `engine.permission-mode` explicitly in your workflow frontmatter, giving you a clear, auditable security boundary.

### 🐛 Bug Fixes

- **`add-wizard` github.com org fallback for GHE** ([#34526](https://github.com/github/gh-aw/pull/34526)): Shorthand workflow specs from public sources were resolving on the active GHE host and returning confusing 404s. The resolver now falls back to github.com for org-less shorthands.
- **PR Sous Chef startup crash context** ([#34524](https://github.com/github/gh-aw/pull/34524)): AWF startup failures were showing up as generic Copilot termination with `stdout/stderr: undefined`. Failure context is now surfaced correctly.

### 📚 Documentation

- **FAQ condensed ~21%** ([#34488](https://github.com/github/gh-aw/pull/34488)): Verbose multi-paragraph answers have been collapsed into tight, scannable responses. Less scrolling, same information.

## 🤖 Agent of the Week: linter-miner

The workflow that turns your codebase's bad habits into laws.

This week `linter-miner` went on a deep dive through the gh-aw codebase, mining for antipatterns ripe for static analysis enforcement. It zeroed in on the `fmt.Fprintln(w, fmt.Sprintf(...))` redundancy — a pattern that allocates an intermediate string, then allocates again to append a newline, when a single `fmt.Fprintf` call would do the job cleanly. The result: a brand-new [`fprintlnsprintf`](https://github.com/github/gh-aw/pull/34498) linter, complete with a bundle of existing violations for the PR reviewer to clean up. It took 39 turns and 10.8 minutes, burning through over a million tokens with the dedication of an engineer who *really* cares about unnecessary heap allocations.

Notably, it failed twice before nailing it on the third run — apparently even automated linter writers need a couple of drafts before the code compiles.

💡 **Usage tip**: Linter miner is most valuable right after a refactor or new abstraction lands — that's when consistent usage patterns (and consistent antipatterns) start to crystallize, and the window to enforce them early is at its widest.

→ [View the workflow on GitHub](https://github.com/github/gh-aw/blob/main/.github/workflows/linter-miner.md)

## Try It Out

Check out [v0.75.4](https://github.com/github/gh-aw/releases/tag/v0.75.4) or the stable [v0.74.8](https://github.com/github/gh-aw/releases/tag/v0.74.8) — and as always, contributions and feedback are welcome in [github/gh-aw](https://github.com/github/gh-aw).
