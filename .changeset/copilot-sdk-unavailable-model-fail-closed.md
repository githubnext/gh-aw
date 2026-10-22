---
"gh-aw": patch
---

Copilot SDK mode (`engine.copilot-sdk: true`) now fails closed when the configured model is not in the live AWF `/reflect` catalogue. After one bounded `/reflect` refresh, the harness refuses to start the SDK, logs the shared "The requested model is not supported" message naming the model and the endpoints checked, and records `model.resolution` as `failed` / `model_unavailable`. Previously it silently ran the first model in the catalogue while every record reported the configured model.
