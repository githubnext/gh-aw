# ADR-46051: Recruitment Banner for Targeted Docs Research

**Date**: 2026-07-01
**Status**: Draft
**Deciders**: @pelikhan, @copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

We need a dismissible recruitment banner in the GH-AW docs site to invite selected participants to surveys or interviews. The docs site is a static Astro Starlight deployment on GitHub Pages, so there is no server-side `current_user` context available for stafftools-style targeting.

### Decision

We implement a Starlight `components.Banner` override that:

- Preserves page frontmatter banners.
- Shows the recruitment banner only when config is enabled and `ctaUrl` is set.
- Captures eligibility from `?recruit=<slug>` (and optional `&uid=<dotcom_id>`) and persists it in `localStorage`.
- Persists dismissals per slug.
- Supports optional path scoping.
- Decorates CTA links with:
  - `utm_source=inproduct_banner`
  - `utm_medium=docs`
  - `utm_campaign=<slug>`
  - `pid=<uid>` when a valid uid is present

### Alternatives Considered

#### Alternative 1: Server-side targeting

Rejected because the docs deployment is static and has no authenticated user context.

#### Alternative 2: Commit participant IDs to repo

Rejected because participant IDs must not be committed to a public repository.

#### Alternative 3: Default to enabled

Rejected to avoid accidental launch. Banner ships disabled and requires an explicit enable step.

### Consequences

#### Positive

- Works in static hosting.
- Preserves existing Starlight banner behavior.
- Enables controlled, participant-targeted recruitment.

#### Negative

- Banner does not auto-stop at quota; operators must disable it manually.
- Link distribution must be done out of band.

#### Neutral

- Audience definition is operational, not repository data.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** are interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

1. `recruitmentBanner.enabled` **MUST** default to `false` in the introducing change.
2. The banner **MUST NOT** render unless `enabled` is `true` and `ctaUrl` is non-empty.
3. The implementation **MUST** preserve native Starlight frontmatter banner behavior.
4. Eligibility **MUST** require `?recruit=<slug>` matching configured `slug`, persisted per browser.
5. When `requireUid` is `true`, eligibility **MUST** require a valid UID from the recruitment link.
6. UIDs **MUST** be treated as operational input and **MUST NOT** be committed in repository content.
7. Dismissal state **MUST** be stored per slug and respected unless frequency is `always`.
8. CTA links **MUST** include UTM parameters, and **MAY** include `pid` when UID is available.
9. CTA URLs **MUST** be restricted to `http` or `https` protocols before runtime decoration.
10. Implementations **MUST NOT** claim the banner is live while `enabled` remains `false`.

---

*This ADR is the operator guidance document for the recruitment banner, replacing the earlier draft at `docs/RECRUITMENT_BANNER.md` (removed in the same PR). Pending final review before status changes from Draft.*
