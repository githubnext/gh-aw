---
"gh-aw": patch
---

Detect and report detection job failures in the conclusion job. Adds support for a
`safe-outputs.messages.detection-failure` message, updates `notify_comment` and
message rendering, and includes tests covering detection failure scenarios.
