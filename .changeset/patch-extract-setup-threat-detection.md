---
"gh-aw": patch
---

Extract the "Setup threat detection" inline script into a reusable
`actions/setup/js/setup_threat_detection.cjs` module and update the
workflow compiler to require the module instead of embedding the
full script. Tests were updated to assert the require pattern.

