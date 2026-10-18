---
"gh-aw": patch
---

Keep AWF model inventories in unified sessions when an unconfigured endpoint reports null `models` or `routing_models`. Null lists are now omitted as unknown instead of becoming empty arrays or rejecting the whole snapshot, and endpoint `configured`, routing `supportedReasoningEfforts`, and `contextWindowTokens` are retained (unified session specification 2.0.0).
