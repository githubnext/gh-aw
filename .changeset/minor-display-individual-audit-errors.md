---
"gh-aw": minor
---

Display individual errors and warnings in audit command output

The `audit` command now displays individual errors and warnings using the same format as compiler errors (`file:line type: message`), making it much easier to identify and locate issues in workflow runs. Previously, the audit report only showed error counts without any details about what went wrong or where. Now each error and warning is displayed with its location and message in an IDE-parseable format.
