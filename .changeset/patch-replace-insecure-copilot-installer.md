---
"gh-aw": patch
---

Replace insecure 'curl | sudo bash' Copilot installer usage with the official `install.sh` downloaded to a temporary file, executed, and removed. Tests updated to assert secure installer usage. Fixes githubnext/gh-aw#6674

---

This changeset was generated for PR #6691.

