---
"gh-aw": patch
---

Add workflow installation instructions to safe output footers with enterprise support

Updates footers in JavaScript safe outputs to include workflow installation instructions and adds support for GitHub Enterprise deployments using `github.server_url`. The URL building logic has been moved from JavaScript to Go for better maintainability.
