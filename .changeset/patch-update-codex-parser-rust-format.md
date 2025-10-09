---
"gh-aw": patch
---

Update Codex log parser to support new Rust-based format

The Codex engine recently migrated from TypeScript to Rust, which produces a different log format. This update adds support for the new format patterns while maintaining full backward compatibility with the old format. The parser now recognizes new thinking sections, tool calls, tool execution, tool results, and token counting patterns from the Rust implementation.
