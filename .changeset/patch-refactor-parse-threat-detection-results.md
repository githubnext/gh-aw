---
"gh-aw": patch
---

Refactor the threat detection result parsing step by moving the inline JavaScript into a dedicated
CommonJS module `actions/setup/js/parse_threat_detection_results.cjs` and update the compiler to require it.
Updated tests to use the require-based pattern and recompiled workflow lock files.
