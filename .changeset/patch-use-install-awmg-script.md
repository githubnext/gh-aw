---
"gh-aw": patch
---

Use the `install-awmg.sh` script from the main branch to download the
`awmg` MCP gateway binary in release mode. The workflow now checks for an
existing `awmg` installation (in PATH, local build, or `$HOME/.local/bin/awmg`)
before attempting to download. When the CLI is a release build, the CLI's
embedded version is passed to the installer to ensure version consistency;
for non-release (dev) builds the latest `awmg` is downloaded.

This is an internal tooling change (avoids duplicate platform-detection
logic and unnecessary downloads) and does not affect public APIs.

