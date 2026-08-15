---
# Report formatting guidelines, exposed as an inline skill (see
# /gh-aw/reference/inline-sub-agents/#explicit-end-marker and the
# "Use Inline Skills to Reduce Context" section of the cost-management guide)
# so agents invoke it on demand instead of always carrying the guidance in the
# main prompt. The explicit "## end skill:" marker bounds this block exactly,
# so importing it does not swallow any content that follows in the workflow
# that imports it.
#
# New workflows: import this file (`imports: [shared/reporting.md]`) instead of
# inlining report-formatting prose in the prompt, so the rules stay
# single-sourced and do not drift.
---

Use the `reporting` skill when producing a report, issue, or PR comment to follow the formatting guidelines below.

## skill: `reporting`
---
description: Report formatting guidelines for gh-aw workflows (header levels, collapsible details, run-ID links)
---

- Use `###` (or lower) headers only.
- Keep summary and critical actions visible; move long detail into `<details>` blocks.
- Structure reports as: overview → key metrics/issues → collapsible detail → next actions.
- Format run IDs as links: `[§12345](https://github.com/owner/repo/actions/runs/12345)`.
- Include up to 3 most relevant run URLs at end under `**References:**`.
- Do NOT add footer attribution (system adds automatically).

## end skill: `reporting`
