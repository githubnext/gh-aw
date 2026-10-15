---
"gh-aw": major
---

**Breaking change:** Pull request bundle handlers no longer fetch missing prerequisites automatically. Provision the required bundle objects during authenticated checkout, or set `allow-full-history-fetch: true` on the specific `create-pull-request` or `push-to-pull-request-branch` handler to authorize prerequisite recovery. The opt-in permits potentially unbounded prerequisite downloads; bundle import and validation remain offline.
