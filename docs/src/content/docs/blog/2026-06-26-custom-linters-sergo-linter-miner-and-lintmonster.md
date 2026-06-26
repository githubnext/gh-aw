---
title: "Custom Linters in Practice: Sergo, Linter Miner, and LintMonster"
description: "How gh-aw grows, audits, and applies its custom Go linters with three cooperating workflows and a trail of linked PRs and issues."
authors:
  - copilot
date: 2026-06-26
metadata:
  seoDescription: "See how gh-aw uses custom Go linters, Linter Miner, Sergo, and LintMonster, with linked workflow files, PRs, issues, and ADRs."
---

`gh-aw` now registers **35 custom Go analyzers** in
[`cmd/linters/main.go`](https://github.com/github/gh-aw/blob/main/cmd/linters/main.go).
That linter surface is not maintained by hand alone.
It is grown, audited, and applied by three separate
workflows:

- [Linter Miner](https://github.com/github/gh-aw/blob/main/.github/workflows/linter-miner.md)
  proposes new analyzers from recurring patterns.
- [Sergo](https://github.com/github/gh-aw/blob/main/.github/workflows/sergo.md)
  stress-tests those analyzers for false positives,
  false negatives, and suppression gaps.
- [LintMonster](https://github.com/github/gh-aw/blob/main/.github/workflows/lint-monster.md)
  runs the custom suite and turns findings into
  tracked cleanup work.

The interesting part is not that each workflow exists.
It is that they form a loop: one workflow adds lint
rules, another challenges them, and a third drives the
codebase toward compliance.

## Linter Miner keeps adding new rules

The
[workflow definition](https://github.com/github/gh-aw/blob/main/.github/workflows/linter-miner.md)
is explicit about its job: mine discussions, issues,
and Go source, pick one new linter idea, implement it,
and open a PR. GitHub search currently shows a long run
of
[`[linter-miner]` PRs](https://github.com/search?q=repo%3Agithub%2Fgh-aw+is%3Apr+%22%5Blinter-miner%5D%22&type=pullrequests),
and the recent examples are concrete:

- [`fprintlnsprintf`](https://github.com/github/gh-aw/pull/34498)
  flags `fmt.Fprintln(w, fmt.Sprintf(...))` and links to
  [ADR 34498](https://github.com/github/gh-aw/blob/main/docs/adr/34498-add-fprintlnsprintf-linter.md).
- [`timeafterleak`](https://github.com/github/gh-aw/pull/39133)
  catches `time.After(...)` inside `for`+`select`
  loops.
- [`errorfwrapv`](https://github.com/github/gh-aw/pull/39263)
  flags `fmt.Errorf(...%v..., err)` where `%w` should
  preserve the error chain.
- [`wgdonenotdeferred`](https://github.com/github/gh-aw/pull/40837)
  catches non-deferred `sync.WaitGroup.Done()` calls.
- [`lenstringsplit`](https://github.com/github/gh-aw/pull/41090)
  rewrites `len(strings.Split(s, sep))` to
  `strings.Count(s, sep)+1` when the separator is
  provably non-empty.
- [`stringreplaceminusone`](https://github.com/github/gh-aw/pull/41285)
  rewrites `strings.Replace(..., -1)` to
  `strings.ReplaceAll(...)`.

This is not a one-off burst. The same theme appears in
the blog's own weekly updates:
[May 25](https://github.com/github/gh-aw/blob/main/docs/src/content/docs/blog/2026-05-25-weekly-update.md),
[June 15](https://github.com/github/gh-aw/blob/main/docs/src/content/docs/blog/2026-06-15-weekly-update.md),
and
[June 22](https://github.com/github/gh-aw/blob/main/docs/src/content/docs/blog/2026-06-22-weekly-update.md).
Those posts document `fprintlnsprintf`,
`timeafterleak`, `errorfwrapv`, and `deferinloop` as
shipped work rather than aspirational ideas.

## Sergo pressure-tests the linters after they land

Where Linter Miner expands the rule set,
[Sergo](https://github.com/github/gh-aw/blob/main/.github/workflows/sergo.md)
does the adversarial follow-up. The workflow is focused
on actionable Go analysis using Serena, and its issue
history shows a steady pattern:
find a precision gap, write a tightly scoped issue, and
let the next PR harden the analyzer.

The clearest evidence is the issue-to-PR chain:

- [Issue #40244](https://github.com/github/gh-aw/issues/40244)
  found that `errstringmatch` only handled
  `strings.Contains(err.Error(), ...)`;
  [PR #40248](https://github.com/github/gh-aw/pull/40248)
  extended coverage to `HasPrefix`, `HasSuffix`,
  `EqualFold`, `Index`, `LastIndex`, and `Compare`.
- [Issue #41377](https://github.com/github/gh-aw/issues/41377)
  found missing `//nolint:` support across four
  context-family linters;
  [PR #41382](https://github.com/github/gh-aw/pull/41382)
  added suppression parity.
- [Issue #41376](https://github.com/github/gh-aw/issues/41376)
  found a false negative in `manualmutexunlock` when
  two struct instances shared the same mutex field;
  [PR #41383](https://github.com/github/gh-aw/pull/41383)
  fixed the keying model.
- [Issue #40947](https://github.com/github/gh-aw/issues/40947)
  found that `wgdonenotdeferred` missed goroutine
  closures launched inside loops;
  [PR #41026](https://github.com/github/gh-aw/pull/41026)
  fixed the function-literal scope boundary.
- [Issue #41163](https://github.com/github/gh-aw/issues/41163)
  found that `lenstringsplit` mishandled an empty
  raw-string separator;
  [PR #41188](https://github.com/github/gh-aw/pull/41188)
  fixed the false positive and the broken autofix.

There is also useful evidence in the failures. Sergo's
[Issue #40243](https://github.com/github/gh-aw/issues/40243)
bundled several package-identity precision fixes into
one direction, and
[PR #40247](https://github.com/github/gh-aw/pull/40247)
closed unmerged after sprawling into a large branch.
The narrower follow-up work still landed, including
[PR #40248](https://github.com/github/gh-aw/pull/40248).
That is a good sign: the workflow is producing reviewable
problems, not just optimistic reports.

## LintMonster turns diagnostics into repository work

[LintMonster](https://github.com/github/gh-aw/blob/main/.github/workflows/lint-monster.md)
operates later in the loop. It runs
`make golint-custom`, groups findings by root cause,
creates or updates issues, and can assign up to three
Copilot agent sessions to fix them.

Its evidence trail is easy to follow:

- [Issue #40932](https://github.com/github/gh-aw/issues/40932)
  grouped four resource-lifecycle and context-propagation
  findings; [PR #41589](https://github.com/github/gh-aw/pull/41589)
  merged the targeted fixes.
- [Issue #40933](https://github.com/github/gh-aw/issues/40933)
  tracked hard-coded path constants;
  [PR #41611](https://github.com/github/gh-aw/pull/41611)
  replaced the flagged literals with existing constants.
- [Issue #39314](https://github.com/github/gh-aw/issues/39314)
  established an authoritative function-length backlog
  for **653 findings**.
- [Issue #41466](https://github.com/github/gh-aw/issues/41466)
  refreshed that same backlog at **660 findings** and
  kept it consolidated instead of spawning duplicate
  tracking issues.

This is what makes the custom linter suite operational
instead of decorative. Rules only matter if they change
the repository. LintMonster is the workflow that turns
diagnostics into queues, slices, assignments, and merged
cleanup work.

## Why the three-workflow loop matters

Taken together, the workflows separate three jobs that
usually get conflated:

1. **Invent a rule from a real pattern.**
   Linter Miner does this with new analyzers such as
   [`timeafterleak`](https://github.com/github/gh-aw/pull/39133)
   and
   [`lenstringsplit`](https://github.com/github/gh-aw/pull/41090).
2. **Challenge the rule's correctness.**
   Sergo does this with issues such as
   [#40947](https://github.com/github/gh-aw/issues/40947)
   and
   [#41163](https://github.com/github/gh-aw/issues/41163).
3. **Apply the rule to production code.**
   LintMonster does this with issue-to-PR chains such as
   [#40932 → #41589](https://github.com/github/gh-aw/issues/40932)
   and
   [#40933 → #41611](https://github.com/github/gh-aw/issues/40933).

That split is why the system looks durable. New rules
keep arriving. Old rules keep getting corrected. The
repository keeps absorbing the results.

## Further evidence

If you want to inspect the trail directly, start here:

- Source workflows:
  [Linter Miner](https://github.com/github/gh-aw/blob/main/.github/workflows/linter-miner.md),
  [Sergo](https://github.com/github/gh-aw/blob/main/.github/workflows/sergo.md),
  [LintMonster](https://github.com/github/gh-aw/blob/main/.github/workflows/lint-monster.md)
- Linter registry:
  [`cmd/linters/main.go`](https://github.com/github/gh-aw/blob/main/cmd/linters/main.go)
- ADRs:
  [34498](https://github.com/github/gh-aw/blob/main/docs/adr/34498-add-fprintlnsprintf-linter.md),
  [39133](https://github.com/github/gh-aw/blob/main/docs/adr/39133-custom-linter-for-time-after-leaks-in-loops.md),
  [40837](https://github.com/github/gh-aw/blob/main/docs/adr/40837-add-wgdonenotdeferred-linter.md),
  [41090](https://github.com/github/gh-aw/blob/main/docs/adr/41090-add-lenstringsplit-linter.md),
  [41285](https://github.com/github/gh-aw/blob/main/docs/adr/41285-add-stringreplaceminusone-linter.md)
- Search views:
  [`[linter-miner]` PRs](https://github.com/search?q=repo%3Agithub%2Fgh-aw+is%3Apr+%22%5Blinter-miner%5D%22&type=pullrequests),
  [`label:sergo` issues](https://github.com/search?q=repo%3Agithub%2Fgh-aw+is%3Aissue+label%3Asergo&type=issues),
  [`label:lint-monster` issues](https://github.com/search?q=repo%3Agithub%2Fgh-aw+is%3Aissue+label%3Alint-monster&type=issues)

This is a useful pattern beyond `gh-aw`: treat static
analysis as a living workflow system, not just a binary
that runs in CI.
