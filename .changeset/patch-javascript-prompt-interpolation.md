---
"gh-aw": patch
---

Use JavaScript for prompt variable interpolation instead of shell expansion

The compiler now uses `actions/github-script` to interpolate GitHub Actions expressions in prompts, replacing the previous shell expansion approach. This improves security by using literal shell variables and adds a dedicated JavaScript interpolation step after prompt creation.
