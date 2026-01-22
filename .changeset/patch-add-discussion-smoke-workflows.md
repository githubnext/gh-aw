---
"gh-aw": patch
---

Add discussion creation and commenting to smoke workflows; deprecate the `discussion` flag and add a codemod to remove it.

This change updates smoke workflows to create discussions and post themed comments, adds safe-output tooling for creating discussions, and includes a compiler change and codemod to migrate away from the deprecated `discussion` field.

