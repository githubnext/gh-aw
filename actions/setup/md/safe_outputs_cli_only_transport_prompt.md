<safe-outputs>
gh CLI is NOT authenticated. Use the configured safe-output tools for GitHub writes and completion signaling. This engine has no MCP tool-call interface, so the `safeoutputs` CLI on `PATH` is the ONLY way to invoke the tools listed in `<safe-output-tools>`: run `safeoutputs <tool_name> <json>` from bash (see the `<mcp-clis>` section). No command is named after an individual tool, so shell discovery such as `type create_pull_request` or `compgen -c` will never find one; always go through the `safeoutputs` binary.

Before using the CLI, check `command -v safeoutputs` and `command -v jq` in the agent's shell; runner setup logs do not establish sandbox availability. If `safeoutputs` is missing, no supported transport is available for this engine: stop and report the missing command in the final response. Do not claim the action was emitted, retry commands that are not installed, or use authenticated GitHub commands as a substitute.

For a large body, write the content to a file with a quoted heredoc and serialize it with `jq -Rs` as described in `<mcp-clis>`. If `jq` is missing, check `command -v node` and use Node.js instead. Do not substitute hand-escaped inline JSON for a large body:
```bash
node - /tmp/gh-aw/body.md <<'JS' > /tmp/gh-aw/payload.json
const fs = require('node:fs');
process.stdout.write(JSON.stringify({
  title: 'My title',
  body: fs.readFileSync(process.argv[2], 'utf8')
}));
JS
```
Only if serialization succeeds, submit it with `safeoutputs create_discussion . < /tmp/gh-aw/payload.json`. Adapt the fields to the enabled tool's schema. If neither `jq` nor `node` is available, call `missing_tool` when configured using a small inline JSON argument describing the unavailable serializer. Reserve `report_incomplete` for failed write attempts. If the diagnostic tool is unavailable, report the limitation in the final response.
