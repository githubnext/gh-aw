---
title: Billing
description: How GitHub Agentic Workflows are billed, including GitHub Actions minutes, AI inference costs, and Copilot licensing.
sidebar:
  order: 295
---

Running an agentic workflow incurs two types of cost: **GitHub Actions minutes** for compute, and **AI inference** charged by the model provider. Both appear independently on your bill.

## GitHub Actions Minutes

Every workflow job consumes Actions compute time at standard [GitHub Actions pricing](https://docs.github.com/en/billing/managing-billing-for-your-products/managing-billing-for-github-actions/about-billing-for-github-actions). A typical run includes a short pre-activation job (10–30 seconds) and an agent job (1–15 minutes). Each job also incurs approximately 1.5 minutes of runner setup overhead.

Actions minutes are billed to the organization or user that owns the repository.

## AI Inference

Inference costs depend on which engine the workflow uses.

### GitHub Copilot (default engine)

There are two billing paths for the Copilot engine (`engine: copilot`, the default):

**Organization billing** — Inference is charged as AI Credits (AIC) against the organization's Copilot tenant. This requires all three of the following:

1. The organization has centralized billing enabled for Copilot requests in its Copilot policies. Enable it under **Organization → Settings → Copilot → Policies → Copilot CLI → "Allow use of Copilot CLI billed to the organization"**. See also [Authentication](/gh-aw/reference/auth/#copilot-requests-write-permission).

2. The workflow declares `copilot_requests: write` under `permissions`:

       permissions:
         contents: read
         copilot-requests: write

3. The workflow has been compiled (`gh aw compile`) and the updated `.lock.yml` committed to the repository.

**Individual/seat billing** — If the above conditions are not met, the workflow must be configured with a user-supplied [`COPILOT_GITHUB_TOKEN`](/gh-aw/reference/auth/#copilot_github_token). In this case inference is attributed to (and limited by) the PAT owner's Copilot entitlements rather than being billed centrally through the organization.

See [Engines](/gh-aw/reference/engines/) for a full list of engines and their authentication requirements, and [Authentication](/gh-aw/reference/auth/) for configuration details. For Copilot model pricing and AIC rates, see [GitHub Copilot models and pricing](https://docs.github.com/copilot/reference/copilot-billing/models-and-pricing).

### Anthropic (Claude)

When using [`engine: claude`](/gh-aw/reference/engines/) or passing `ANTHROPIC_API_KEY`, inference is billed directly to your [Anthropic account](https://console.anthropic.com/settings/billing).

### OpenAI (Codex / GPT)

When using [`engine: codex`](/gh-aw/reference/engines/) or passing `OPENAI_API_KEY`, inference is billed directly to your [OpenAI account](https://platform.openai.com/account/billing).

### Google (Gemini)

When using [`engine: gemini`](/gh-aw/reference/engines/) or passing `GEMINI_API_KEY`, inference is billed directly to your [Google Cloud / AI Studio account](https://aistudio.google.com/billing).

## Estimating and Monitoring Costs

Use `gh aw logs` to view per-run AI Credits (AIC) and duration, and `gh aw audit <run-id>` for a detailed breakdown of token usage and inference spend. See [Cost Management](/gh-aw/reference/cost-management/) for monitoring strategies, cost controls, and techniques for reducing spend.
