---
"gh-aw": minor
---

Change strict mode default from false to true

**⚠️ Breaking Change**: Strict mode is now enabled by default for all agentic workflows.

**What changed:**
- Workflows without an explicit `strict:` field now default to `strict: true` (previously `false`)
- The JSON schema default for the `strict` field is now `true`
- Compiler applies schema default when `strict:` is not specified in frontmatter

**Migration guide:**
- **No action needed** if your workflows already comply with strict mode requirements
- **Add `strict: false`** to workflows that need to opt out of strict mode validation
- **Review security** implications before disabling strict mode in production workflows

**Strict mode enforces:**
1. No write permissions (`contents:write`, `issues:write`, `pull-requests:write`) - use safe-outputs instead
2. Explicit network configuration (no implicit defaults)
3. No wildcard `*` in network.allowed domains
4. Network configuration required for custom MCP servers with containers
5. GitHub Actions pinned to commit SHAs (not tags/branches)
6. No deprecated frontmatter fields

**Examples:**
```yaml
# Default behavior (strict mode enabled)
on: issues
permissions:
  contents: read

# Explicitly disable strict mode
on: issues
permissions:
  contents: write  # Would fail in strict mode
strict: false      # Opt out
```

**CLI behavior unchanged:**
- `gh aw compile` still defaults to non-strict compilation at CLI level
- Schema default applies when `strict:` not in frontmatter
- `gh aw compile --strict` still overrides frontmatter settings

See [Strict Mode Documentation](https://githubnext.github.io/gh-aw/reference/frontmatter/#strict-mode-strict) for details.
