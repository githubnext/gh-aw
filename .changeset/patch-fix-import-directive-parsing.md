---
"gh-aw": patch
---

Fix import directive parsing for new {{#import}} syntax

Fixed a bug in `processIncludesWithWorkflowSpec` where the new `{{#import}}` syntax was incorrectly parsed using manual regex group extraction, causing malformed workflowspec paths. The function now uses the `ParseImportDirective` helper that correctly handles both legacy `@include` and new `{{#import}}` syntax. Also added safety checks for empty file paths and comprehensive unit tests.
