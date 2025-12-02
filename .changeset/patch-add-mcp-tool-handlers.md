---
"gh-aw": patch
---

Add support for loading MCP tool handlers from external files. This change
updates the JavaScript MCP server core and its schema to allow a tool's
`handler` field to point to a file whose default export (or module export)
is used as the handler function. Supports sync/async handlers, ES module
default exports, shell-script handlers, path traversal protection,
serialization fallback for circular structures, and extensive logging. Tests
for handler loading were added and workflows were recompiled.

