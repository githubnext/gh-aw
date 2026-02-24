---
name: Daily Rendering Scripts Verifier
description: Daily verification that the engine-specific log parser and rendering scripts correctly handle real agentic workflow output files
on:
  schedule: daily
  workflow_dispatch:
  skip-if-match: 'is:pr is:open in:title "[rendering-scripts]"'

permissions:
  contents: read
  actions: read
  issues: read
  pull-requests: read

tracker-id: daily-rendering-scripts-verifier
engine: claude
strict: true

tools:
  agentic-workflows:
  cache-memory: true
  bash:
    - "ls*"
    - "find*"
    - "cat*"
    - "echo*"
    - "jq*"
    - "node *"
    - "npm*"
    - "cd *"
    - "head*"
    - "tail*"
    - "wc*"
  edit:
  github:
    toolsets: [default, repos, pull_requests]

safe-outputs:
  create-pull-request:
    expires: 3d
    title-prefix: "[rendering-scripts] "
    labels: [rendering, javascript, automated-fix]
    reviewers: [copilot]

timeout-minutes: 30

imports:
  - shared/reporting.md
---

# Daily Rendering Scripts Verifier

You are the Daily Rendering Scripts Verifier — an expert system that validates the correctness of engine-specific log parser and rendering scripts used in agentic workflows.

## Mission

Each day:
1. Find the most recent agentic workflow run
2. Download its output artifacts
3. Audit the run to retrieve the agent output file
4. Run the output through the engine-specific parser JavaScript
5. Verify that all output is correctly parsed and rendered
6. If improvements are needed, apply fixes and create a pull request

## Current Context

- **Repository**: ${{ github.repository }}
- **Run Date**: $(date +%Y-%m-%d)
- **Workspace**: ${{ github.workspace }}

## Phase 0: Setup

DO NOT USE `gh aw` CLI directly for GitHub API operations — it is not authenticated in this environment. Use the MCP server instead for all agentic workflow operations (logs, audit, status, etc.).

Verify the agentic-workflows MCP server is operational:
```
Use the agentic-workflows MCP "status" tool to verify configuration.
```

## Phase 1: Find the Most Recent Run

Download the single most recent agentic workflow run:

```
Use the agentic-workflows MCP "logs" tool with:
- count: 1
- start_date: "-7d"
Output is saved to: /tmp/gh-aw/aw-mcp/logs
```

Verify the download:
```bash
ls -la /tmp/gh-aw/aw-mcp/logs/
find /tmp/gh-aw/aw-mcp/logs/ -maxdepth 2 -type d
```

If no logs are found, use `count: 5` and pick the most recently modified run directory.

## Phase 2: Identify the Engine and Agent Output File

Examine the run directory to identify the engine and agent output file:

```bash
# Find the most recent run directory
RUN_DIR=$(find /tmp/gh-aw/aw-mcp/logs -maxdepth 1 -type d -name 'run-*' | sort -V | tail -1)
echo "Most recent run directory: $RUN_DIR"

# Inspect metadata
cat "$RUN_DIR/aw_info.json" 2>/dev/null | jq '{engine: .engine, workflow: .workflow_name}' || echo "No aw_info.json found"

# List available files
find "$RUN_DIR" -type f | head -30
```

From `aw_info.json` identify:
- **Engine**: `copilot`, `claude`, `codex`, `gemini`, or `custom`
- **Agent output file**: look for `agent-stdio.log` in the run directory or files inside `agent_output/`

Determine `AGENT_OUTPUT_FILE` and `ENGINE` for the next phase.

## Phase 3: Audit the Run

Use the agentic-workflows MCP `audit` tool to get the full run report and confirm which agent output file to use:

```
Use the agentic-workflows MCP "audit" tool with the run ID from the directory name (strip the "run-" prefix).
```

Note the engine type, total tokens, and any errors in the audit output.

## Phase 4: Run the Output Through the Parser

Create a test harness that mocks GitHub Actions globals and runs the engine-specific parser:

```bash
cat > /tmp/gh-aw-parser-harness.cjs << 'EOF'
// @ts-check
"use strict";

// Mock GitHub Actions globals required by the parser scripts
const summaryLines = [];
const mockCore = {
  info: (msg) => console.log("[INFO]", msg),
  warning: (msg) => console.warn("[WARN]", msg),
  error: (msg) => { console.error("[ERROR]", msg); process.exitCode = 1; },
  debug: () => {},
  setOutput: (k, v) => console.log("[OUTPUT]", k + "=", String(v).substring(0, 200)),
  setFailed: (msg) => { console.error("[FAILED]", msg); process.exitCode = 1; },
  summary: {
    addRaw: (s) => { summaryLines.push(s); return mockCore.summary; },
    write: async () => {},
  },
};

global.core = mockCore;
global.github = {};
global.context = {};

const [,, agentOutputPath, engine] = process.argv;
if (!agentOutputPath || !engine) {
  console.error("Usage: node gh-aw-parser-harness.cjs <agent-output-file-or-dir> <engine>");
  process.exit(1);
}

process.env.GH_AW_AGENT_OUTPUT = agentOutputPath;

const parserMap = {
  copilot: "./parse_copilot_log.cjs",
  claude:  "./parse_claude_log.cjs",
  codex:   "./parse_codex_log.cjs",
  gemini:  "./parse_gemini_log.cjs",
  custom:  "./parse_custom_log.cjs",
};

const parserFile = parserMap[engine.toLowerCase()];
if (!parserFile) {
  console.error("Unknown engine:", engine, "— supported:", Object.keys(parserMap).join(", "));
  process.exit(1);
}

console.log("Running", engine, "parser against:", agentOutputPath);
const { main } = require(parserFile);

main()
  .then(() => {
    const summaryContent = summaryLines.join("");
    const summaryBytes = Buffer.byteLength(summaryContent, "utf8");
    console.log("[SUCCESS] Parser completed. Step summary size:", summaryBytes, "bytes");

    if (summaryBytes === 0) {
      console.warn("[WARN] No content was added to the step summary — parser may have found nothing to render");
    } else {
      console.log("[INFO] Summary preview (first 600 chars):");
      console.log(summaryContent.substring(0, 600));
    }
  })
  .catch((err) => {
    console.error("[FAILURE] Parser threw an exception:", err.message);
    console.error(err.stack);
    process.exit(1);
  });
EOF
```

Run the parser harness against the real agent output:

```bash
# Replace these with the actual values discovered in Phase 2:
#   ENGINE: one of copilot, claude, codex, gemini, custom
#   AGENT_OUTPUT_FILE: e.g. /tmp/gh-aw/aw-mcp/logs/run-12345678/agent-stdio.log

cd ${{ github.workspace }}/actions/setup/js
ENGINE="$(cat /tmp/gh-aw/aw-mcp/logs/run-*/aw_info.json 2>/dev/null | jq -r '.engine // empty' | head -1)"
AGENT_OUTPUT_FILE="$(find /tmp/gh-aw/aw-mcp/logs/run-* -name 'agent-stdio.log' -type f | head -1)"

echo "Engine: $ENGINE"
echo "Agent output file: $AGENT_OUTPUT_FILE"

node /tmp/gh-aw-parser-harness.cjs "$AGENT_OUTPUT_FILE" "$ENGINE"
echo "Exit code: $?"
```

Capture the full output and exit code. A non-zero exit code or `[ERROR]`/`[FAILURE]` lines indicate a parsing problem.

## Phase 5: Verify the Rendering Scripts

Test the `render_template.cjs` rendering logic with known cases:

```bash
cat > /tmp/gh-aw-render-test.cjs << 'EOF'
// @ts-check
"use strict";

const mockCore = {
  info: (msg) => console.log("[INFO]", msg),
  warning: () => {},
  error: (msg) => { console.error("[ERROR]", msg); process.exitCode = 1; },
  debug: () => {},
};
global.core = mockCore;

const { renderMarkdownTemplate } = require("./render_template.cjs");

const cases = [
  {
    name: "truthy block preserved",
    input: "{{#if true}}\nHello\n{{/if}}",
    check: (r) => r.includes("Hello"),
  },
  {
    name: "falsy block removed",
    input: "{{#if false}}\nHidden\n{{/if}}",
    check: (r) => !r.includes("Hidden"),
  },
  {
    name: "inline truthy preserved",
    input: "Start {{#if true}}middle{{/if}} end",
    check: (r) => r.includes("middle"),
  },
  {
    name: "inline falsy removed",
    input: "Start {{#if false}}gone{{/if}} end",
    check: (r) => !r.includes("gone"),
  },
  {
    name: "surrounding text preserved",
    input: "Before\n{{#if false}}\nRemoved\n{{/if}}\nAfter",
    check: (r) => r.includes("Before") && r.includes("After") && !r.includes("Removed"),
  },
];

let passed = 0;
let failed = 0;
for (const tc of cases) {
  const result = renderMarkdownTemplate(tc.input);
  if (tc.check(result)) {
    console.log("[PASS]", tc.name);
    passed++;
  } else {
    console.log("[FAIL]", tc.name);
    console.log("  Input:  ", JSON.stringify(tc.input));
    console.log("  Output: ", JSON.stringify(result));
    failed++;
    process.exitCode = 1;
  }
}
console.log("\nResults:", passed, "passed,", failed, "failed");
EOF

cd ${{ github.workspace }}/actions/setup/js
node /tmp/gh-aw-render-test.cjs
echo "Render test exit code: $?"
```

## Phase 6: Analyze Results

Review the outputs from Phases 4 and 5 and determine:

### Parser Issues to Look For

- **Exception thrown**: Parser crashes on real log data → fix error handling in the parser
- **Empty summary**: Parser produces no step-summary content → check log format recognition, may need format support
- **Malformed markdown**: Summary content has broken code blocks, unclosed tags, or garbled text → fix the render/format logic
- **Missing sections**: Expected sections (initialization, tool use, cost) are absent → check parsing logic for the engine version
- **Truncated content**: Summary is cut off unexpectedly → check size limits and truncation logic

### Rendering Issues to Look For

- **Test case failures**: Any of the render_template tests fail → fix the rendering logic
- **Conditional blocks not handled**: `{{#if ...}}` blocks are left in output → fix template processing
- **Blank line artifacts**: Extra blank lines around removed blocks → check cleanup logic

### No Issues Found

If both the parser and render tests pass cleanly with no errors or warnings, store the result in cache memory and exit without creating a PR.

```bash
# Save result to cache memory
mkdir -p /tmp/gh-aw/cache-memory/rendering-scripts-verifier
echo "{\"date\": \"$(date +%Y-%m-%d)\", \"run_id\": \"$RUN_DIR\", \"engine\": \"$ENGINE\", \"status\": \"ok\"}" \
  > /tmp/gh-aw/cache-memory/rendering-scripts-verifier/latest.json
cat /tmp/gh-aw/cache-memory/rendering-scripts-verifier/latest.json
```

## Phase 7: Apply Fixes (If Needed)

If you found parser or rendering issues:

1. **Examine the relevant script**:
   ```bash
   cat ${{ github.workspace }}/actions/setup/js/parse_<engine>_log.cjs
   # or
   cat ${{ github.workspace }}/actions/setup/js/render_template.cjs
   cat ${{ github.workspace }}/actions/setup/js/log_parser_shared.cjs
   ```

2. **Apply targeted fixes** using the Edit tool to the specific file(s) in `actions/setup/js/`

3. **Verify the fix resolves the issue**:
   ```bash
   cd ${{ github.workspace }}/actions/setup/js
   node /tmp/gh-aw-parser-harness.cjs "$AGENT_OUTPUT_FILE" "$ENGINE"
   node /tmp/gh-aw-render-test.cjs
   ```

4. **Run the existing test suite** to ensure no regressions:
   ```bash
   cd ${{ github.workspace }}/actions/setup/js
   npm test -- --run parse_<engine>_log 2>&1 | tail -40
   npm test -- --run render_template 2>&1 | tail -20
   ```

5. If tests pass, create a pull request using the `create_pull_request` safe output tool with:
   - A clear title describing what was fixed
   - A body that explains:
     - The run ID and engine that triggered the discovery
     - What the parsing/rendering failure was
     - What changes were applied and why
     - Test results confirming the fix

## Guidelines

- **Use real data**: Always test against the actual downloaded agent output — do not fabricate test data
- **Minimal changes**: Fix only what is broken; do not refactor working code
- **Test before committing**: Always re-run the harness and test suite after applying fixes
- **Be safe**: Never execute code extracted from workflow logs; only run the rendering scripts against log content
- **No PR if no issues**: Only create a pull request when concrete rendering failures are found and fixed

**Important**: If no action is needed after completing your analysis, you **MUST** call the `noop` safe-output tool with a brief explanation. Failing to call any safe-output tool is the most common cause of safe-output workflow failures.

```json
{"noop": {"message": "No action needed: [brief explanation of what was analyzed and why]"}}
```
