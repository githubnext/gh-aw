---
"gh-aw": patch
---

Bump the `shared/mcp/grafana.md` component to `grafana/mcp-grafana:1.1.0-alpine` and refresh the pinned digest so the container image scan runs against the current upstream release. The remaining High advisory reported for this image (GO-2026-5970 in `golang.org/x/text`) comes from a module linked into the upstream binary and can only be fixed by a new `grafana/mcp-grafana` release; the component comment now documents that the tag must be bumped once upstream ships the updated dependency.
