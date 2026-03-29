---
"gh-aw": patch
---

Fix a shell-escaping bypass in workflow engine command generation by always escaping arguments and validating agent file paths, preventing shell injection from crafted agent import paths.
