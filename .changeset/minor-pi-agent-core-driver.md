"gh-aw": minor

Add pi engine driver mode with @earendil-works/pi-agent-core sample driver

Adds `engine.driver` — a new engine configuration field that lets the pi engine (and
potentially other engines) run a custom Node.js driver script instead of the built-in CLI:

```yaml
engine:
  id: pi
  driver: pi_agent_core_driver.cjs          # built-in driver
  # or a workspace-relative path:
  driver: .github/drivers/pi_agent_core_driver_sample_node.cjs
```

When `driver` is set on the pi engine, gh-aw launches the driver directly with Node.js
instead of the pi CLI. The driver is expected to output JSONL compatible with the existing
`parse_pi_log.cjs` log parser, so all downstream tooling (step summaries, token usage
tracking) works unchanged.

New files:

- `actions/setup/js/pi_agent_core_driver.cjs` — built-in driver that runs a pi agent
  session using `@earendil-works/pi-agent-core`.  Reads `GH_AW_PROMPT`,
  `GH_AW_PI_MODEL`, and `PI_CODING_AGENT_DIR` from the environment; supports both AWF
  gateway (firewall) mode via `models.json` and direct API access.
- `.github/drivers/pi_agent_core_driver_sample_node.cjs` — minimal sample driver that
  users can copy to their own `.github/drivers/` directory and customise.

Go changes:

- `EngineConfig.Driver` field added (generic, not copilot-specific).
- `engine.driver` frontmatter key is parsed and validated with the same safety rules as
  `engine.copilot-sdk-driver` (no absolute paths, no `..`, no shell metacharacters; only
  `.js`/`.cjs`/`.mjs` extensions or a bare name are accepted).
- JSON schema updated to document `engine.driver`.
