/**
 * Research recruitment banner configuration.
 *
 * Adapted from the github/research-accelerator `recruitment-banner` skill, but tailored to a
 * **static** Astro Starlight site (this docs site is deployed to GitHub Pages at
 * https://github.github.com/gh-aw/). There is no server, so there is no `current_user` and no
 * stafftools UI. Targeting is therefore enforced by *who you distribute the recruitment link to*
 * — your CSV of `dotcom_id`s — not by an allowlist committed to this public repo (committing
 * participant user IDs to a public repo would expose them).
 *
 * The recruitment link you send to targeted users carries `?recruit=<slug>` (and optionally
 * `&uid=<dotcom_id>` for attribution). Only recipients of that link see the banner; it then
 * sticks (per browser) so it keeps showing as they navigate the docs.
 *
 * ── Bright lines (kept faithfully from the skill) ───────────────────────────────────────────
 *   • Merging this PR does NOT turn the banner on. `enabled` ships as `false`.
 *     Turning it on is a deliberate, separate human action (set `enabled: true` and merge) —
 *     the static-site analog of flipping "Banner is visible".
 *   • The audience (`dotcom_id` CSV) is distributed out of band and is NOT committed here.
 *   • Don't change `slug` once the link has been distributed, or the banner silently stops
 *     matching the link recipients.
 */
export interface RecruitmentBannerConfig {
/**
 * Master switch. Leave `false` in the PR that introduces the banner; flip to `true` only
 * after you have tested it with your own recruitment link (see docs/adr/46051-recruitment-banner-for-targeted-docs-research.md).
 */
enabled: boolean;

/**
 * Stable, kebab-case identifier that ties the banner, the recruitment link (`?recruit=<slug>`)
 * and the UTM campaign together. Must not change after the link is distributed.
 */
slug: string;

/** Short, honest banner title. */
title: string;

/** One-line message. Mention any incentive honestly here if one is offered. */
message: string;

/** Call-to-action button text (e.g. "Take the survey"). */
ctaText: string;

/**
 * The survey / booking URL. UTM params (`utm_source=inproduct_banner`, `utm_medium=docs`,
 * `utm_campaign=<slug>`) and an optional participant id (`pid=<uid>`) are appended at runtime.
 */
ctaUrl: string;

/**
 * Optional path scoping. Each entry is matched as a path prefix of the current pathname
 * (e.g. `/gh-aw/guides/`). Empty array = show on every docs page.
 */
paths: string[];

/** `once` (default): show until dismissed, then remember the dismissal. `always`: ignore dismissals. */
frequency: 'once' | 'always';

/**
 * When `true`, only show the banner if the recruitment link carried a `uid` (stricter — the
 * link must be the per-user one). When `false`, any visitor who arrives with `?recruit=<slug>`
 * is eligible.
 */
requireUid: boolean;
}

export const recruitmentBanner: RecruitmentBannerConfig = {
// Ships OFF. Flip to true only after testing — see docs/adr/46051-recruitment-banner-for-targeted-docs-research.md.
enabled: false,

// TODO(researcher): finalize before distributing the link. Do not change after distribution.
slug: 'gh-aw-docs-research-2026q3',

title: 'Help shape the future of GitHub Agentic Workflows',
message: "Got 5 minutes? Get a gift card by telling us how you're using Agentic Workflows.",
ctaText: 'Take the survey',

ctaUrl: 'https://survey.alchemer.com/s3/8896945/GitHub-Agentic-Workflows',

// Empty = all docs pages. Example to scope: ['/gh-aw/guides/', '/gh-aw/patterns/'].
paths: [],

frequency: 'once',
requireUid: false,
};
