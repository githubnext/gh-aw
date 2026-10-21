# ADR-70697: Mirror the AWF Firewall Allowlist into Copilot CLI URL Approvals and Derive Ledger Tool Inventories from Typed Definitions

**Date**: 2026-10-21
**Status**: Draft
**Deciders**: pelikhan, Copilot coding agent (draft — [TODO: verify])

---

### Context

Two problems showed up in smoke runs (#70654, #70635, #70627). First, the Copilot CLI checks URL access separately from shell tool permissions. Workflows that granted `--allow-tool shell` but no `--allow-url` approvals had network-bearing shell commands (such as `curl` or `pip install`) denied in noninteractive mode. This happened even when the AWF firewall already allowed those destinations. Second, the ledger safe-output tools (such as `ledger_append` and the notes tools) were exposed at runtime by the safe-outputs MCP server. They were missing from the compiled `<safe-output-tools>` prompt inventory and the MCP manifest, so agents wrongly reported them as unavailable. Ledger tool definitions were built inline as untyped `map[string]any` values inside `generateDynamicTools`. That made it hard to reuse them in other compile-time surfaces.

### Decision

We will generate one explicit Copilot CLI `--allow-url <domain>` argument for each destination in the effective AWF firewall allowlist (`buildCopilotAllowedDomains`). This applies only when the firewall is enabled and the engine is not in SDK mode, and the wildcard `*` is never passed through. The firewall stays the single source of truth for network policy, and Copilot's URL approval layer is kept in sync with it. Copilot is never told to allow all URLs. We will also move ledger tool definitions into a typed `dynamicLedgerTool` struct, returned by `generateDynamicLedgerTools`. Runtime `tools.json` generation, the prompt tool inventory (`buildSafeOutputsSections`) and the MCP manifest (`collectMCPServersForManifest`) will all derive ledger tool names from that one function through `collectLedgerSafeOutputToolNames`.

### Alternatives Considered

#### Alternative 1: Grant all URLs to Copilot (`--allow-all-urls` / `--allow-url '*'`)

This is the simplest fix for denied network commands, and the firewall would still enforce egress. It was rejected because it removes Copilot's own defense-in-depth URL layer. The agent would see every destination as approved, even ones the firewall later blocks. The code explicitly skips `*` for this reason.

#### Alternative 2: Separate, user-configured Copilot URL allowlist

Workflow authors could list Copilot URL approvals in frontmatter, separate from `network.allowed`. This gives fine-grained control. It was rejected because it duplicates network policy in two places that can drift apart, and it pushes extra configuration onto every workflow that already declares its network needs.

#### Alternative 3: Hard-code ledger tool names in the prompt and manifest builders

The prompt and manifest builders could list ledger tool names as static strings (as is still done for the `ledger` MCP server entry). This is a smaller change, but the names would drift from runtime tool generation as ledger types and note tools change. That drift is the bug this PR fixes.

### Consequences

#### Positive
- Network commands to firewall-approved destinations run noninteractively under Copilot CLI without widening URL access beyond the firewall.
- Network policy is defined once (the AWF allowlist) and reused for both enforcement and Copilot approvals.
- Prompt inventory, MCP manifest and runtime `tools.json` share one typed source for ledger tools, so agents no longer see "missing" tools that actually exist.

#### Negative
- Compiled lock files grow considerably: every Copilot workflow with the firewall enabled now has one `--allow-url` argument per domain. All 345 workflows needed regeneration.
- Copilot's URL approvals are now tied to the AWF domain format. If Copilot `--allow-url` semantics differ from AWF wildcard semantics (for example `*.pythonhosted.org`), approvals may be wider or narrower than intended. [TODO: verify]
- Workflows without the firewall, and SDK-mode workflows, still get no URL approvals, so behavior is different across modes.

#### Neutral
- Shell tool permissions, GitHub permissions and SDK permission handling are unchanged. Only the URL approval layer is added.
- The ledger prompt now tells agents to fall back to Python's read-only `sqlite3` module when the `sqlite3` executable is missing.
- Live confirmation that the change fixes the problem is still pending, because no workflows were dispatched during the PR.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
