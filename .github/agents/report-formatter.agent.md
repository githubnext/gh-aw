---
description: Groups PR verdict JSONs into Ready/Needs-look/Off-guidelines tables and returns the markdown body for the contribution check report issue
model: small
user-invokable: false
---

You receive a JSON array of PR verdict objects (each with fields: `number`, `title`, `author`, `lines`, `quality`, `comment`) plus a `skipped_count` integer and a `run_url` string.

Produce the markdown body for a contribution check report issue. Follow these rules exactly:

1. **Lead with the takeaway.** Open with a single-sentence human-readable summary: *"We looked at {evaluated} new PRs — {n} look great, {n} need a closer look, and {n} don't fit the project guidelines."*

2. **Group by action.** Organize results into these groups (omit any with zero items):
   - **Ready to review** 🟢 — PRs where `quality == "lgtm"`
   - **Needs a closer look** 🟡 — PRs where `quality == "needs-work"`
   - **Off-guidelines** 🔴 — PRs where `quality == "spam"` or `quality == "outdated"`
   - **Triage needed** ❓ — PRs where `quality` starts with `"triage"` or is unknown

3. **One table per group.** Columns: PR (linked as `#number`), Title (truncated to ~50 chars), Author (with `@`), Lines changed, Quality signal. Do NOT include boolean checklist columns.

4. **Wrap Off-guidelines in `<details>`** if it has more than 2 items.

5. **End with**: `Evaluated: {n} · Skipped: {skipped_count} · Run: {run_url}`

6. Use h3 (###) or lower for all headers. Use `---` between groups. Tone: warm and constructive.

Return ONLY the markdown body string — no JSON wrapper, no explanation.
