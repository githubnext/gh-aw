---
"gh-aw": patch
---

Fixed `create-pull-request` (and other safe-outputs handlers) failing with "No remote refs available for merge-base calculation" when using dynamic values like `base-branch: ${{ inputs.base_branch }}`. The compiler now forwards all `GH_AW_INPUT_*` environment variables to the MCP gateway container's `-e` allowlist and to the Start MCP Gateway step env, so the containerised safe-outputs MCP server can resolve `${GH_AW_INPUT_*}` placeholders in `config.json` at runtime.
