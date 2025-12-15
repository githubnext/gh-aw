---
"gh-aw": patch
---

Move mention filtering from incoming text processing to the agent output collector.

This is an internal refactor and bugfix: sanitizers were modularized, mention
resolution was moved into the output collector, and a bug that prevented known
authors from being preserved in mentions was fixed. Tests were updated.

