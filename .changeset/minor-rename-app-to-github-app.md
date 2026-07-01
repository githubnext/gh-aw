---
"gh-aw": major
---

Renamed the deprecated `app:` workflow field to `github-app:` and added the codemod plus schema/Go updates to keep tooling in sync.

**⚠️ Breaking Change**: The `app:` workflow field has been renamed to `github-app:`. Workflows using top-level `app:` or nested `app:` auth blocks will fail validation.

**Migration guide:**
- Replace `app:` with `github-app:` everywhere in your workflow frontmatter, including the top-level fallback and nested auth blocks
- Example:
  ```yaml
  # Before
  app:
    id: ${{ secrets.APP_ID }}
    private-key: ${{ secrets.APP_PRIVATE_KEY }}

  # After
  github-app:
    id: ${{ secrets.APP_ID }}
    private-key: ${{ secrets.APP_PRIVATE_KEY }}
  ```
- A codemod is available to automate this migration: run `gh aw fix` to update existing workflows
