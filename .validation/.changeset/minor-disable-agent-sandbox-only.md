---
"gh-aw": major
---
Removed the deprecated top-level `sandbox: false` option and replaced it with `sandbox.agent: false`, so only the agent firewall can be disabled while the MCP gateway stays enabled. Add `gh aw fix` to migrate existing workflows.

**⚠️ Breaking Change**: The top-level `sandbox: false` field has been removed.

**Migration guide:**
- Replace `sandbox: false` with `sandbox.agent: false` in your workflow frontmatter
- Example:
  ```yaml
  # Before
  sandbox: false

  # After
  sandbox:
    agent: false
  ```
- Run `gh aw fix` to automatically migrate existing workflows
