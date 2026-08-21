---
"gh-aw": patch
---

Fixed the safe-outputs MCP server exposing a duplicate, unwired generic tool alongside the real, workflow-named tool for renamed dynamic safe outputs. When `call_workflow` (or `dispatch_workflow`/`dispatch_repository`) targeted a workflow, the server registered both the properly-typed tool (e.g. `agent_sandbox_stack`) and a generic `call_workflow` tool whose handler wrote a malformed record and reported false success, because the dedup check only compared exact tool names. Dynamic tool synthesis now also recognises tools by their family metadata (`_call_workflow_name`, `_workflow_name`, `_dispatch_repository_tool`) and no longer synthesizes tools for handler-only or global config keys such as `create_report_incomplete_issue`, `mentions`, and `max_bot_mentions`.
