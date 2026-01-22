---
"gh-aw": patch
---

Add retry logic to the GitHub Copilot CLI installer by moving the
installation steps into a dedicated shell script and invoking it from
workflows. This prevents intermittent download failures during setup.

The change includes creating `actions/setup/sh/install_copilot_cli.sh`,
updating `pkg/workflow/copilot_srt.go` to call the script, and
recompiling workflows that now reference the retry-enabled installer.

