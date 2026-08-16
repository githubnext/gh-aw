---
"gh-aw": minor
---

Simplify GitHub access configuration into a single homogeneous `tools.github.mode` enum and a public `tools.mcp-mode` selector.

- `tools.github.mode` is now a homogeneous enum: `cli` (pre-authenticated `gh` CLI protected by the host policy proxy — recommended default), `mcp-local` (local Docker GitHub MCP server), and `mcp-remote` (hosted GitHub MCP service). The author-facing `gh-proxy` terminology is retired in favor of `cli`.
- **Default semantics (compile decision):** to preserve existing behavior, an omitted `tools.github.mode` continues to resolve to `mcp-local` (the historical GitHub MCP default) — omitting the mode never silently changes an existing workflow. `cli` is the recommended homogeneous value: it is auto-derived for engines without MCP support (e.g. Pi), used by new-workflow scaffolds, and selectable explicitly. The resolver additionally resolves to `cli` when `features.integrity-reactions` or legacy `features.cli-proxy: true` is set.
- The duplicate `tools.github.type` field and the legacy `features.cli-proxy` flag are deprecated. The top-level `tools.cli-proxy` boolean is renamed to `tools.mcp-mode: cli` (with `tools.mcp-mode: default` for normal MCP behavior).
- Engines without MCP support (including Pi) now automatically derive CLI GitHub access and CLI MCP-mode; authors no longer configure `tools.github.mode` / `tools.mcp-mode` for them, and the redundant Pi-specific requirement checks are removed.
- **Validation:** MCP-only fields (`toolsets`/`allowed`/`version`/`args`) are rejected with an explicit `tools.github.mode: cli`; `features.integrity-reactions` requires CLI access and rejects an explicit MCP mode; `cloud-hypervisor` rejects an explicit CLI mode; `tools.cli-proxy` and `tools.mcp-mode` cannot both be set. CLI-policy fields (`allowed-repos`, `min-integrity`, `github-token`, `github-app`) remain meaningful in all modes.
- A single typed resolver (`resolveGitHubAccessProfile`) now produces one effective GitHub access profile used consistently for prompt selection, MCP registration, host policy-proxy startup, AWF flags, `GH_TOKEN` exclusion, runtime compatibility, and integrity, replacing the previous split `isGitHubCLIModeEnabled` / `isCliProxyNeeded` semantics.
- `gh aw fix` codemods migrate legacy frontmatter: `mode: gh-proxy` → `mode: cli`; `mode`/`type: local` → `mode: mcp-local`; `mode`/`type: remote` → `mode: mcp-remote`; `features.cli-proxy: true` → `tools.github.mode: cli`; `tools.cli-proxy: true` → `tools.mcp-mode: cli`.
