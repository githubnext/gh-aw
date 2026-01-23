---
"gh-aw": patch
---

Update `smoke-claude` workflow to import the shared `go-make` workflow and
expose `safeinputs-go` and `safeinputs-make` tools for running Go and Make
commands used by CI and local testing. This is an internal tooling update and
does not change public APIs.

The workflow now validates the `safeinputs-make` tool by running `make build`.
