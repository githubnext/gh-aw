---
"gh-aw": patch
---
Skip hardcoded pin fallback when GH_HOST is configured to a non-github.com host. When `GH_HOST` points to a GitHub Enterprise Server or GHEC instance, the dynamic action SHA resolver targets that host and fails to resolve `actions/*` repos (which live on github.com). Silently falling back to bundled hardcoded pins in that case produces unverified SHA pins and masks the real problem. The fallback is now suppressed and callers will see the standard "Unable to pin action" warning instead.
