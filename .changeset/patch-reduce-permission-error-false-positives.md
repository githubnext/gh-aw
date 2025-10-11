---
"gh-aw": patch
---

Mark permission-related error patterns as warnings to reduce false positives

Permission-related error patterns were being classified as fatal errors, causing workflow runs to fail unnecessarily when encountering informational messages about permissions, authentication, or authorization. This change introduces a `Severity` field to the `ErrorPattern` struct that allows explicit override of the automatic level detection logic, enabling fine-grained control over which patterns should be treated as errors versus warnings.

Updated 26 permission and authentication-related patterns across the Codex and Copilot engines to be classified as warnings instead of errors, improving workflow reliability while maintaining visibility of permission issues for troubleshooting.
