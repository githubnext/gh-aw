---
"gh-aw": patch
---

The `shared/mcp/grafana.md` component now uses the Alpine-based `grafana/mcp-grafana:1.0.0-alpine` image instead of the untagged (Debian bookworm-slim) image. The Debian base layer shipped a large set of OS packages (perl-base, libc-bin, util-linux, ...) responsible for all of the container scan findings; the Alpine variant removes those packages and pins the component to an explicit release tag.
