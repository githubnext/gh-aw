---
"gh-aw": patch
---

Add three missing authored-AW threat checks to `gh aw lint`: unsafe `pull_request_target` role admission, shell expansions blocked in safe-output steps, and firewall URL filtering without SSL inspection. The firewall rule inspects accepted `sandbox.agent.args` flags rather than schema-rejected `network.firewall` frontmatter. Share the existing shell detector with compiler validation and document the remaining semantic and runtime coverage boundaries.

Reference known compiler threat IDs in rule listings and text/JSON diagnostics, and move the detailed rule catalog into its own reference page. Check URL-filter SSL prerequisites in authored `sandbox.agent.args`, which compiles only in non-strict mode. Report schema-invalid legacy `network.firewall` blocks as requiring migration, never recommend `network.firewall.ssl-bump` as a fix, and verify accepted remedies through source-to-compiler regression tests. Imported settings and runtime enforcement remain outside this source check.
