---
"gh-aw": patch
---

Update `github.com/spf13/cobra` dependency from v1.10.1 to v1.10.2.

This patch updates the Cobra dependency to v1.10.2 which migrates its
internal YAML dependency from the deprecated `gopkg.in/yaml.v3` package to
`go.yaml.in/yaml/v3`. The change is internal to the dependency and is
transparent to consumers of `gh-aw`.

No code changes were required in this repository. Run the usual validation
steps: `make test`, `make lint`, and `make agent-finish`.

