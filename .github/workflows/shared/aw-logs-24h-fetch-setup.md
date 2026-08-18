---
# Pre-fetch last 24 hours of agentic workflow logs for analysis
# Saves logs to /tmp/gh-aw/aw-mcp/logs/
#
# NOTE: --count defaults to 10 and is applied *in addition to* --start-date
# (it caps the number of matching runs returned, not just how far back to
# look). On a high-volume fleet the default silently truncates the "24h"
# window to only the most recent handful of runs, so an explicit high
# --count is required here to actually cover the full 24h window. 3000 is
# intentionally generous (the fleet has been observed running 100+ runs/hour,
# i.e. up to ~2400+/day) to keep headroom above expected peak daily volume;
# if the fleet ever exceeds this, the CLI reports a truncation/continuation
# signal that should be surfaced rather than silently dropped.

tools:
  agentic-workflows:
  cache-memory: true
  timeout: 300

steps:
  - name: Download logs from last 24 hours
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: ./gh-aw logs --start-date -1d --count 3000 -o /tmp/gh-aw/aw-mcp/logs
---
