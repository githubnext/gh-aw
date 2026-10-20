---
"gh-aw": patch
---

Fix false Copilot SDK shell permission denials for commands containing comments while preserving literal hashes and checking every executable command against the workflow allowlist.

Keep ambiguous nested shell quoting fail-closed and terminate comments only at line feeds so quote confusion and carriage returns cannot hide disallowed commands.

Reject unsupported unquoted parenthesis contexts, including extended globs, so literal pattern hashes cannot conceal following commands.

Check commands after background (`&`) and stderr-pipe (`|&`) operators while preserving redirection ampersands, and retain Bash-specific word boundaries across escaped newlines and non-Bash whitespace.
