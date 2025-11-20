---
"gh-aw": patch
---

Fix aw.patch generation logic to handle local commits

Fixed a bug where patch generation only captured commits from explicitly named branches. When an LLM makes commits directly to the currently checked out branch during action execution, those commits are now properly captured in the patch file. Added HEAD-based patch generation as a fallback strategy and extensive logging throughout the patch generation process.
