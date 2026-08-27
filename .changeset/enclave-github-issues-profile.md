---
"gh-aw": minor
---

Add the opt-in `enclaves[].agent.github.cli: issues-read-v1` compiler profile
for isolated GitHub Issues REST access through an AWF-owned enclave proxy.
Update the default dependencies to AWF `v0.28.9` and mcpg `v0.4.12`. AWF
`v0.28.9` adds protected enclave agent startup and entrypoint diagnostics while
mcpg `v0.4.12` provides the Issues profile and private-alias TLS SAN support.
