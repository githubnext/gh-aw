---
"gh-aw": patch
---

Replace the npm-based GitHub Copilot CLI installation with the
official installer script and add support for mounting the installed
binary into AWF runs.

This removes the Node.js npm dependency for AWF mode and documents
the new `--mount /usr/local/bin/copilot:/usr/local/bin/copilot:ro`
usage for workflows that run Copilot inside AWF.

