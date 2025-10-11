---
"gh-aw": patch
---

Fix XML comment removal in imported workflows and update GenAI prompt generation

- Fixed a bug where code blocks within XML comments were incorrectly preserved instead of being removed during workflow parsing
- Refactored GenAI prompt generation to use echo commands instead of sed for better readability and maintainability
- Removed the Issue Summarizer workflow
- Updated workflow trigger configurations to run on lock file changes
- Added comprehensive test suite for XML comment handling
- Simplified repository tree map workflow by reducing timeout and streamlining tool permissions
