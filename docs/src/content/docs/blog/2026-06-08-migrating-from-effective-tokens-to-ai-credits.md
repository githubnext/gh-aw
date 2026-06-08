---
title: "Effective Tokens replaced by AI Credits"
description: "The latest gh-aw build uses AI Credits (AIC) as the primary spend metric and keeps Effective Tokens (ET) as a legacy compatibility field."
authors:
  - copilot
date: 2026-06-08
metadata:
  seoDescription: "gh-aw now reports AI Credits (AIC) as the primary spend metric, aligned with GitHub Copilot billing and models.dev pricing."
  linkedPostText: "ET replaced by AIC in latest build"
---

In the latest gh-aw build, Effective Tokens (ET) have been
replaced by AI Credits (AIC) as the primary spend metric.

> [!IMPORTANT]
> AIC is now the default cost metric in gh-aw output. ET remains
> available only as a legacy compatibility field.

This change reflects GitHub Copilot billing and models.dev pricing.
It makes spend tracking directly aligned to monetary cost instead of
a normalized token proxy.

## What this means in practice

- `gh aw audit` and `gh aw logs` report AI Credits as the primary
  spend metric.
- Effective Tokens are deprecated in documentation and should be
  treated as legacy compatibility output.
- Cost reporting and budget discussions should use AIC values.

For repositories that need automatic workflow updates, run:

```bash
gh aw fix --write
```

## Metric reference

- **AI Credits (AIC)**: primary spend metric (1 AIC = $0.01 USD)
- **Effective Tokens (ET)**: deprecated legacy metric

## Where to read more

- [Cost Management](/gh-aw/reference/cost-management/)
- [Auditing Workflows](/gh-aw/reference/audit/)
- [AI Credits Specification](/gh-aw/specs/ai-credits-specification/)
- [Effective Tokens Specification](/gh-aw/specs/effective-tokens-specification/)
