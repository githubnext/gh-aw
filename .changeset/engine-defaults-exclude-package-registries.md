---
"gh-aw": patch
---

Remove package registries from engine default network domains. `registry.npmjs.org` (Copilot, Gemini, Pi) and `registry.npmjs.org`, `pypi.org`, `files.pythonhosted.org` (Claude) were merged into the firewall allow-list unconditionally, so an agent could reach npm and PyPI even when the workflow declared `network: {}` or `network: { allowed: [defaults, github] }`. This contradicted the documented behavior that package ecosystems require explicit opt-in.

Package registries now require an explicit `network.allowed` ecosystem entry (`node`, `python`, …) or a matching `runtimes:` declaration. Model/API transport domains are unchanged, and engine CLIs, SDKs and containerized `npx`/`uvx` MCP servers are unaffected because they are installed and launched outside the agent sandbox.

If a workflow relies on the agent itself reaching a registry (for example a `web-fetch` of `https://registry.npmjs.org/...`), add the matching ecosystem to `network.allowed` and recompile.
