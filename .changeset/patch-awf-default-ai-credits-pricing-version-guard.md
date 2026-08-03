---
"gh-aw": patch
---

Reject `models.default-ai-credits-pricing` when a workflow pins an AWF version older than v0.27.43, because those versions drop `apiProxy.defaultAiCreditsPricing` during config resolution before it reaches the API proxy.
