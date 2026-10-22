---
"gh-aw": patch
---

Codex workflow-declared sub-agents are now written as native TOML role files in `$CODEX_HOME/agents` with their resolved models, and an unresolvable sub-agent model stops the run before Codex starts. Sub-agent aliases resolve only among models whose `/reflect` `supported_endpoints` include `/responses`, and a model without `/responses` fails before spawn with the new `model.resolution` cause `unsupported_endpoint`.
