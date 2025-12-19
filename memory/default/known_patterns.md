## Known Patterns (2025-12-19)

- Prompt quality canon emerging: Copilot PR prompt analysis (6939) shows higher merge rates for longer prompts with file references; clustering remains active, so prompt engineering guidance is converging.
- Type safety push: Typist report (6936) flags heavy `map[string]any` use and recommends typed frontmatter/step structs, extending prior hygiene audits.
- Documentation friction loop: Noob test (6941) highlights Playwright download blocks in restricted runners and unclear PAT permissions/jargon, feeding repeat onboarding fixes.
- Firewall data freshness gap: Daily firewall report (6943) relied on cached Dec 16 data because gh auth/MCP/visualization deps were missing, signaling recurring telemetry availability issues.
- Workflow error noise continues: Daily audit (6910) still shows false-positive error spikes (e.g., Smoke Claude) and MCP URL misconfigs even on successful runs.
