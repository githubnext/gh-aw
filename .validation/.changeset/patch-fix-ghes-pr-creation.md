---
"gh-aw": patch
---

Fix `add-wizard` and `add --create-pull-request` failing to create pull requests in GitHub Enterprise Server repositories. The PR creation commands now detect the GitHub host from the git `origin` remote URL and pass `--hostname` to `gh pr create`, ensuring GHES repositories are targeted correctly instead of always defaulting to github.com.
