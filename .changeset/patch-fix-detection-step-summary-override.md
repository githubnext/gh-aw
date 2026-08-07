---
"gh-aw": patch
---

Fix detection job failing with `config_error exit=2` when `$GITHUB_STEP_SUMMARY` is inaccessible inside the AWF chroot. The inline detection engine execution step now overrides `GITHUB_STEP_SUMMARY` at the step level to a writable path (`/tmp/gh-aw/threat-detection/step-summary.md`). A new `detection_step_summary` step captures the file content into `$GITHUB_OUTPUT` without printing it to the runner log.
