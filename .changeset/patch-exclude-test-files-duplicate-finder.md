---
"gh-aw": patch
---

Update duplicate finder workflow to ignore test files

Updated the duplicate code detector agentic workflow to exclude test files from analysis. Test files often contain acceptable duplication (setup/teardown code, similar test patterns) that doesn't warrant refactoring. The workflow now filters out common test file patterns across Go, JavaScript, TypeScript, and Python, reducing false positives and improving signal-to-noise ratio.
