---
"gh-aw": patch
---

Improve visibility when safe output messages are not handled

Fixed an issue where safe output messages (like create_issue) were silently skipped when no handler was loaded, with only a debug log that isn't visible by default. Now these cases produce clear warnings to help users identify configuration issues.

Changes:
- Convert debug logging to warning when message handlers are missing
- Add detailed warning explaining the issue and suggesting fixes
- Track skipped messages separately in processing summary
- Add test coverage for missing handler scenario

This ensures users are notified when their safe output messages aren't being processed, making it easier to diagnose configuration issues.
