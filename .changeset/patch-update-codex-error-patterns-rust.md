---
"gh-aw": patch
---

Update Codex error patterns to support new Rust-based format

The Codex engine migrated from TypeScript to Rust, which produces a different log format for error and warning messages. This update replaces the old TypeScript format error patterns (with bracketed timestamps) with new Rust format patterns (unbracketed timestamps with milliseconds and Z timezone suffix). No backward compatibility is maintained - only the new Rust format is supported.
