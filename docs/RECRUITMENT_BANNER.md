# Research recruitment banner

A small, dismissible banner that invites **recruited participants** to take a survey (or book an
interview). It is adapted from the github/research-accelerator `recruitment-banner` skill, but
tailored to this **static** Astro Starlight docs site.

> **Banner:** GitHub Agentic Workflows docs research · **Slug:** `gh-aw-docs-research-2026q3`
> **Page:** all `/gh-aw/` docs pages (configurable) · **Audience:** the `dotcom_id` CSV you distribute the link to
> **Code needed:** this one-time component (it's now in the repo — future runs are config-only)

## How targeting works on a static site

The stafftools recruitment banner targets `current_user` server-side. This docs site has **no
server** — it's deployed to GitHub Pages (`https://github.github.com/gh-aw/`), so it can't know who
is viewing it. Instead:

- You distribute a **recruitment link** that carries `?recruit=gh-aw-docs-research-2026q3`
  (optionally `&uid=<dotcom_id>` for per-participant attribution) to your audience — the same CSV of
  `dotcom_id`s you'd otherwise upload in stafftools.
- Only recipients of that link see the banner. The eligibility is remembered per browser
  (`localStorage`), so it keeps showing as they navigate the docs, and hides once dismissed.
- The audience CSV is **never committed here** — committing participant user IDs to a public repo
  would expose them. The list lives wherever you send the link from (email, DM, Slack, etc.).

Tracking: the CTA link gets `utm_source=inproduct_banner&utm_medium=docs&utm_campaign=<slug>` (plus
`pid=<uid>` when present) appended automatically, and a `banner_research_cta` browser event fires on
click for any analytics you wire up.

## Files

| File | Purpose |
| --- | --- |
| `docs/src/config/recruitmentBanner.ts` | All the knobs: `enabled`, `slug`, copy, `ctaUrl`, path scoping, frequency. **Edit this.** |
| `docs/src/components/RecruitmentBanner.astro` | The banner UI + client-side eligibility logic. Overrides Starlight's `Banner` slot. |
| `docs/astro.config.mjs` | Wires the override: `components.Banner`. |

## Configure it (edit `docs/src/config/recruitmentBanner.ts`)

1. **`ctaUrl`** — your real survey / booking URL (replace the `REPLACE-WITH-YOUR-SURVEY-URL` placeholder).
2. **`title` / `message` / `ctaText`** — keep it tight and honest; mention any incentive in `message`.
3. **`slug`** — stable, kebab-case; it must match the `?recruit=<slug>` in the link you distribute,
   and **must not change** once distributed.
4. **`paths`** — leave `[]` for all docs pages, or scope to e.g. `['/gh-aw/guides/', '/gh-aw/patterns/']`.
5. **`enabled`** — **leave `false` in this PR.** Flipping it to `true` is the go-live step below.

## Go-live checklist — your gates (nothing is live yet)

The banner ships **OFF** (`enabled: false`). Turning it on is the deliberate human action — the
static-site analog of flipping "Banner is visible". Bright lines from the skill apply: you turn it
on; you don't claim it's live until it is; you create your own banner rather than editing another
team's.

- [ ] **Test first.** With `enabled: true` locally (or on a preview), open a docs page with
      `?recruit=gh-aw-docs-research-2026q3&uid=<your-own-dotcom_id>`
      ([look yours up](https://api.github.com/users/YOUR_USERNAME)). Confirm the banner renders,
      the CTA points at your survey with UTM params, and dismiss works.
- [ ] **Distribute the link** `https://github.github.com/gh-aw/?recruit=gh-aw-docs-research-2026q3`
      (or per-user with `&uid=<dotcom_id>`) to your audience CSV — out of band, not in this repo.
- [ ] **Flip `enabled: true`** and merge — and **watch it** (recruitment banners fill *fast*).
- [ ] **Set `enabled: false`** (and merge) the moment you hit your quota. There is **no auto-cap**.
      Keep the run short (≤ 24h).
- [ ] **Update the banner ownership registry** with the slug, page, owning team, and DRI.

## What this does NOT do

- It does **not** turn itself on, and merging the introducing PR does **not** make it visible.
- It does **not** commit or fabricate participant user IDs — bring your own audience and distribute
  the link to it.
- It does **not** auto-stop at a quota — monitor and flip `enabled` back to `false`.
