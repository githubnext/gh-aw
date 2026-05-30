---
"gh-aw": patch
---

Clean up the `actions/setup` helpers by removing the old `GH_AW_ALLOWED_BOTS` branches, simplifying the bot status checks, and inlining the git auth configuration used when fetching/pushing branches so the unused helper/tests can be dropped.
