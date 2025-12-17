---
"gh-aw": patch
---

Use the official Copilot CLI `install.sh` script instead of piping a
downloaded script directly into `sudo bash`. The new pattern downloads the
installer to a temporary file, executes it, and removes the temporary file to
reduce supply-chain risk. Tests were updated to assert the secure install
pattern. Fixes githubnext/gh-aw#6674

