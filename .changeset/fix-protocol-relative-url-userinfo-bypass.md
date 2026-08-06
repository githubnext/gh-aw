---
"gh-aw": patch
---

Fixed a domain-allowlist bypass in the content sanitizer for protocol-relative URLs containing userinfo. `//github.com@evil.com/x` was read as the allowlisted host `github.com` (the host pattern stops at `@`) and passed through unredacted, even though a browser on an HTTPS page resolves it to `https://github.com@evil.com/x` and connects to `evil.com`. In a markdown image this was an exfiltration channel via GitHub's camo proxy. `sanitizeUrlDomains` now strips userinfo from protocol-relative URLs (including chained `a@b@host` forms and `host:port@` forms) before the allowlist check, matching the existing handling for explicit `scheme://` URLs.
