---
# Headroom — Context Compression Layer for AI Agents
# Reduces token usage by 60–95% by compressing large content before it reaches the LLM.
# Accuracy is preserved; originals are cached and retrievable on demand (CCR).
#
# Source:   https://github.com/chopratejas/headroom
# PyPI:     pip install headroom-ai
# Docker:   ghcr.io/chopratejas/headroom:latest
# Docs:     https://headroom-docs.vercel.app/docs
#
# This shared workflow:
#   1. Installs headroom-ai and starts the headroom proxy on port 8787 (pre-step)
#   2. Exposes the headroom MCP server so the agent can call compression tools directly
#
# MCP tools exposed to the agent:
#   - headroom_compress  — compress text/JSON/code content; returns a summary + cache hash
#   - headroom_retrieve  — fetch the original full content by hash when needed
#   - headroom_stats     — report token savings for the current session
#
# Usage:
#   imports:
#     - shared/mcp/headroom.md

pre-steps:
  - name: Install and start headroom proxy
    run: |
      pip install "headroom-ai[proxy,mcp]==0.26.0" --quiet
      headroom proxy --host 0.0.0.0 --port 8787 &
      # Wait up to 20 s for the proxy to become ready
      READY=false
      for i in $(seq 1 20); do
        if curl -sf http://localhost:8787/health > /dev/null 2>&1; then
          echo "✅ Headroom proxy ready"
          READY=true
          break
        fi
        sleep 1
      done
      if [ "$READY" != "true" ]; then
        echo "❌ Headroom proxy did not become ready within 20 seconds"
        exit 1
      fi

network:
  allowed:
    - python
    - local

mcp-servers:
  headroom:
    container: "ghcr.io/chopratejas/headroom:latest"
    args:
      - "--network"
      - "host"
    entrypoint: "headroom"
    entrypointArgs:
      - "mcp"
      - "serve"
      - "--proxy-url"
      - "http://localhost:8787"
    allowed:
      - headroom_compress
      - headroom_retrieve
      - headroom_stats
---

<!--
# Headroom Context Compression

Headroom is active.  The proxy is running at `http://localhost:8787` and the MCP server
exposes three tools to this agent:

| Tool                  | When to use                                                                 |
|-----------------------|-----------------------------------------------------------------------------|
| `headroom_compress`   | Before processing a large file, log dump, search result set, or JSON blob — compress it first and work from the summary. |
| `headroom_retrieve`   | When the summary is insufficient and you need the full original content.    |
| `headroom_stats`      | At the end of a run to log how many tokens were saved this session.         |

## Recommended usage pattern

```
1. Read a large content source (file, API response, log, …)
2. Call headroom_compress with the content
3. Work from the compressed summary
4. Call headroom_retrieve only if a specific detail is needed
5. Call headroom_stats at session end and include the savings in your report
```

Typical savings: 73% on GitHub issue triage, 92% on log analysis, 47% on codebase exploration.
-->
