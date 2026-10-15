---
"gh-aw": minor
---

Add `gh aw skill` to print the bundled agentic-workflows skill without creating repository files or requiring network access. Fall back to unauthenticated public raw URLs when instruction and skill downloads encounter HTTP 403 responses, including cross-organization SAML token restrictions.

Keep `gh aw init` working with SAML/403 tokens by fetching the public skill file list via a raw URL, with the embedded list available when both endpoints fail.
