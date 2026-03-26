---
"gh-aw": minor
---

Added `gh aw audit report` subcommand for cross-run security audit reports. Features include:
- `--workflow` flag to filter by workflow name or filename
- `--last` flag to control how many recent runs to analyze (default: 20, max: 50)
- Parallel artifact downloads using the existing concurrent download infrastructure
- Cross-run aggregation with `CrossRunFirewallReport` type containing executive summary, domain inventory, and per-run breakdown
- Markdown output by default (suitable for security reviews, `$GITHUB_STEP_SUMMARY`), with JSON (`--json`) and pretty console (`--format pretty`) output options
