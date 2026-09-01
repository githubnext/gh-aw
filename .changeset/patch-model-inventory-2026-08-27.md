---
"gh-aw": patch
---

Model alias inventory update for 2026-08-27: correct pricing for `gpt-5.6-sol` ($2/1M input, $10/1M output, $0.20/1M cache read, $2.50/1M cache write) and `gemini-3.6-flash` ($0.75/1M input, $3.75/1M output, $0.075/1M cache read) in the `github-copilot` provider in `models.json`, which were stored at roughly 2x the Copilot SDK-reported price; verified against Copilot SDK `billing.tokenPrices` data.
