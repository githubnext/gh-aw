---
"gh-aw": minor
---

Add an optional HTTPS GET admission gate configured through `admission` in `aw.json` for centralized AW cost controls. Requests send only the GitHub Actions token and current execution metadata, and expose opaque admission errors without spend details. Transient failures use bounded retries with exponential backoff and jitter; the `on_outage` policy defaults to `ignore` and supports `block`. Explicit denials always block execution.
