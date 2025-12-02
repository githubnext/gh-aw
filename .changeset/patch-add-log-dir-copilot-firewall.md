---
"gh-aw": patch
---

Add the missing `--log-dir` argument to the Copilot sandbox (firewall) mode so logs are written to the expected
location (`/tmp/gh-aw/.agent/logs/`) for parsing and analysis.

Files changed:
- `pkg/workflow/copilot_engine.go` (added `--log-dir`)
- `pkg/workflow/firewall_args_test.go` (test added to verify presence)
