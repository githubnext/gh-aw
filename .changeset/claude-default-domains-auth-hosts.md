---
"gh-aw": patch
---

Add `claude.ai` and `platform.claude.com` to the Claude engine default network domains. Recent Claude Code CLI versions contact these hosts during startup for authentication and sign-in/feature configuration; when the firewall blocked them the CLI exited before emitting any structured log entry, surfacing as `ERR_CONFIG: Claude execution failed: no structured log entries were produced`.
