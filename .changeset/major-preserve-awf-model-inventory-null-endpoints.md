---
"gh-aw": major
---

Keep AWF model inventories in unified sessions when an unconfigured endpoint reports null `models` or `routing_models`. Breaking change (unified session specification 2.0.0): `model.inventory` endpoint `models` and `routingModels` may now be absent, meaning unknown; readers must not treat absence as an empty list. Explicit empty arrays still mean a known-empty inventory. Null lists are now omitted as unknown instead of becoming empty arrays or rejecting the whole snapshot, and endpoint `configured`, routing `supportedReasoningEfforts`, and `contextWindowTokens` are retained.
