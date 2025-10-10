---
"gh-aw": patch
---

Improve Copilot error detection to treat permission denied messages as warnings

Updated error pattern classification in the Copilot engine to correctly identify "Permission denied and could not request permission from user" messages as warnings instead of errors. This change improves error reporting accuracy and reduces false positives in workflow execution metrics.
