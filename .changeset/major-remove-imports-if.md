"gh-aw": major

Remove `imports.if` support from workflow frontmatter.

**⚠️ Breaking Change**: `imports:` entries no longer accept an `if` condition because conditional imports can change workflow setup and security posture at runtime.

**Migration guide:**
- Keep security-relevant imports unconditional.
- For experiment-specific prompt variants, use `{{#if experiments.<name> ...}}` with `{{#runtime-import ...}}` in the workflow body instead.
