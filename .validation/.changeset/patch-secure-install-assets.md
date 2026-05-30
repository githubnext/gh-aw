---
"gh-aw": patch
---

Harden the Copilot CLI installer to download release binaries directly and verify SHA256 checksums before running, and avoid shell-interpreted `exec.exec` calls in `upload_assets.cjs` by using the array argument form.
