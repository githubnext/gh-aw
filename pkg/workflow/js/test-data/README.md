# Copilot Log Test Data

This directory contains sample logs from actual Copilot CLI workflow runs for testing and documentation purposes.

## Files

### Raw Logs (Not in Git)
These files are excluded from git via `.gitignore` due to their size. They are stored locally in `sample-logs/` directory.

#### sample-logs/copilot-raw-logs-run-18296543175.log
- **Source**: https://github.com/githubnext/gh-aw/actions/runs/18296543175/job/52095988377
- **Workflow**: Tidy workflow (automatic code formatting and linting)
- **Format**: Raw Copilot CLI debug logs with timestamps and embedded JSON API responses
- **Size**: ~152KB, 4429 lines
- **Download**: Get from workflow artifacts or run `gh aw audit 18296543175`

### Transformed Output (In Git)

#### copilot-transformed-run-18296543175.md
- **Description**: Output from applying `parse_copilot_log.cjs` to workflow run 18296543175 (tidy workflow)
- **Result**: Successfully parsed debug logs showing code formatting and linting workflow
- **Location**: Committed to git for review (small file, ~1.4KB)

#### copilot-transformed-run-18296916269.md
- **Description**: Output from applying `parse_copilot_log.cjs` to workflow run 18296916269 (MCP imports research)
- **Result**: Successfully parsed debug logs showing research workflow with web search and file editing
- **Location**: Committed to git for review (small file, ~6.9KB)
- **Workflow**: Research task involving MCP server configuration and imports

## Format Analysis

The current `parse_copilot_log.cjs` implementation expects structured JSON logs similar to Claude:
```json
[
  {"type": "system", "subtype": "init", ...},
  {"type": "assistant", "message": {...}},
  {"type": "user", "message": {...}}
]
```

However, the actual Copilot CLI produces debug logs with this structure:
```
2025-10-06T22:48:23.111Z [INFO] Starting Copilot CLI: 0.0.335
2025-10-06T22:48:27.031Z [DEBUG] AnthropicTokenLimitErrorTruncator will truncate...
2025-10-06T22:49:18.969Z [DEBUG] response (Request-ID ...):
2025-10-06T22:49:18.969Z [DEBUG] data:
2025-10-06T22:49:18.969Z [DEBUG] {
  "choices": [
    {
      "finish_reason": "tool_calls",
      "message": {
        "content": "...",
        "role": "assistant"
      }
    }
  ],
  "usage": {...}
}
```

## Next Steps

The parser needs to be updated to handle the actual Copilot CLI debug log format, or Copilot CLI needs to output structured JSON logs similar to Claude.

### Option 1: Update Parser to Handle Debug Logs
- Parse timestamped log lines
- Extract JSON responses from `[DEBUG] data:` blocks
- Reconstruct conversation flow from API responses

### Option 2: Update Copilot CLI Output
- Have Copilot CLI output structured JSON logs
- Match the format used by Claude for consistency
- Include system init, assistant messages, and user responses in a structured array

### Option 3: Dual Format Support
- Support both formats in the parser
- Auto-detect which format is being used
- Handle both structured JSON and debug logs
