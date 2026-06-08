---
title: "Migrating from Effective Tokens to AI Credits"
description: "A practical guide to updating dashboards, alerts, and workflows after gh-aw moved from Effective Tokens (ET) to AI Credits (AIC)."
authors:
  - copilot
date: 2026-06-08
metadata:
  seoDescription: "Learn how to migrate ET-based reporting to AIC in gh-aw with compatibility tips, field mappings, and rollout steps."
  linkedPostText: "Migrate ET-based reporting to AIC"
---

> [!IMPORTANT]
> **AI Credits (AIC)** are now the primary spend metric in
> gh-aw. **Effective Tokens (ET)** are deprecated and kept only
> for backward compatibility in report output.

If your dashboards, budgets, or alerts still depend on Effective
Tokens, this migration is mostly a naming and interpretation
update. The workflow data is still there, but the default metric
for cost tracking has changed to AI Credits so spend is easier to
understand and compare.

## Why the metric changed

Effective Tokens normalized token usage into a single synthetic
number. That was useful for cross-model analysis, but it was not
directly tied to money.

AI Credits are simpler for operations and planning:

- **1 AIC = $0.01 USD**
- AIC aligns with provider pricing and real spend
- AIC is easier to use for budgets, alerts, and forecasting

For the formal details, see the
[AI Credits Specification](/gh-aw/specs/ai-credits-specification/)
and the
[deprecated Effective Tokens Specification](/gh-aw/specs/effective-tokens-specification/).

## What to update first

Start with anything that drives financial decisions:

1. Dashboard primary KPI labels (`ET` → `AIC`)
2. Alert thresholds and pager rules
3. Weekly or monthly cost rollups
4. Internal docs and runbooks

Keep ET fields visible during rollout if your team still uses
historical ET baselines.

## Field mapping for migration

Use this mapping when updating parsers and report consumers:

| Existing usage | Preferred replacement |
| --- | --- |
| Effective Tokens (ET) as primary spend metric | AI Credits (AIC) as primary spend metric |
| ET budget thresholds | AIC budget thresholds |
| ET trend chart headline | AIC trend chart headline |
| ET-only incident criteria | AIC-first criteria, ET optional for diagnostics |

> [!NOTE]
> `gh aw audit` and `gh aw logs` still include legacy ET fields so
> existing integrations can migrate without a hard cutover.

## Rollout strategy that avoids breakage

Use a two-phase migration:

### Phase 1: Dual reporting

Update consumers to read AIC and ET in parallel. Display AIC first
in UI and summaries, but keep ET available for historical
comparisons.

### Phase 2: AIC-first operations

After your alerting and dashboard baselines are stable, switch
operational decisions to AIC and move ET to a secondary or hidden
field.

This phased approach reduces noisy alerts and avoids sudden changes
to long-running trend views.

## Common pitfalls

- **Renaming without recalibrating thresholds**: ET thresholds do
  not translate 1:1 to AIC. Start by taking 2–4 weeks of recent
  runs, then set AIC warning/critical thresholds from observed
  percentiles (for example, P75 warning and P95 critical).
- **Dropping ET too early**: Keep ET during transition if your
  quarterly reporting depends on historical ET curves.
- **Updating charts but not automations**: Make sure CI summaries,
  issue templates, and workflow notifications use AIC language too.

## Where to read more

- [Cost Management](/gh-aw/reference/cost-management/)
- [Auditing Workflows](/gh-aw/reference/audit/)
- [AI Credits Specification](/gh-aw/specs/ai-credits-specification/)
- [Effective Tokens Specification](/gh-aw/specs/effective-tokens-specification/)
