---
"githubnext/gh-aw": patch
---

Fix copilot log parser to show tool success status instead of question marks

The copilot engine workflow log parser now correctly displays tool execution status with ✅ (success) or ❌ (failure) icons instead of showing ❓ (unknown) for all tool calls. This fix creates synthetic tool_result entries in the parseDebugLogFormat function, allowing the parser to properly match tools with their results and determine their execution status.
