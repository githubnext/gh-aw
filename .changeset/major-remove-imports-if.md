---
"gh-aw": major
---

Remove `imports.if` support from workflow frontmatter.

**⚠️ Breaking Change**: `imports:` entries no longer accept an `if` condition because conditional imports can change workflow setup and security posture at runtime.

**Migration guide:**
- Remove any `if:` keys from `imports:` entries; imported workflow fragments must now be unconditional.
- Keep security-relevant imports unconditional so the full workflow structure is visible at compile time.
- For experiment-specific prompt variants, move the condition into the workflow body and use `{{#if experiments.<name> ...}}` together with `{{#runtime-import ...}}`.
- Example:
  ```yaml
  # Before
  imports:
    - uses: ./.github/workflows/shared/prompts/review.md
      if: ${{ experiments.new_prompt }}
  ```

  ```md
  <!-- After -->
  {{#if experiments.new_prompt}}
  {{#runtime-import ./.github/workflows/shared/prompts/review.md}}{{/runtime-import}}
  {{/if}}
  ```
