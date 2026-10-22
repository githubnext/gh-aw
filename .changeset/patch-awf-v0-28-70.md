---
"gh-aw": patch
---

Bump the default gh-aw-firewall from v0.28.62 to v0.28.70, including compromised eMudhra CA trust mitigation, missing credential-file placeholder fixes, and enclave execution policy and lifecycle updates.

Eligible, verified public-only Cloud Hypervisor enclaves now retain bounded host-only diagnostics automatically; these logs are not automatically uploaded. Private, mixed, and dynamic scopes retain their existing behavior.

Follow-up integrations: provenance-authenticated AWF bootstrap downloads (the gh-aw installer still verifies checksums only), optional protected command-output capture, and an explicit public-enclave diagnostics opt-out. Existing feature-version gates and generated configuration remain compatible; upstream adds no schema properties or validation constraints in this release range.
