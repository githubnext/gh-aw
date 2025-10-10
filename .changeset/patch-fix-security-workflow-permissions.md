---
"gh-aw": patch
---

Add security-events permission to security workflow

Fixed a permissions error in the security-fix-pr workflow that prevented it from accessing code scanning alerts. The workflow now includes the required `security-events: read` permission to successfully query GitHub's Code Scanning API for vulnerability analysis and automated fix generation.
