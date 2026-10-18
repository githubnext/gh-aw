---
"gh-aw": patch
---

Add three missing authored-AW threat checks to `gh aw lint`: unsafe `pull_request_target` role admission, shell expansions blocked in safe-output steps, and firewall URL filtering without SSL inspection. Share the existing shell detector with compiler validation and document the remaining semantic and runtime coverage boundaries.
