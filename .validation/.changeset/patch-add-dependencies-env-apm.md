---
"gh-aw": patch
---

Added `dependencies.env` support for APM dependencies so workflows can pass environment variables to the `microsoft/apm-action` pack step (for example private registry auth), while keeping deterministic env ordering in generated workflow steps and skipping duplicate `GITHUB_TOKEN` entries when `github-app` is configured.

Upgraded default APM versions to `microsoft/apm@v0.8.2` and `microsoft/apm-action@v1.3.4`.
