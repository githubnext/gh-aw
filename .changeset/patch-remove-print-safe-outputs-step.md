---
"githubnext/gh-aw": patch
---

Remove "Print Safe Outputs" step from generated lock files

The "Print Safe Outputs" step has been removed from all generated GitHub Actions workflow lock files. This step was previously displaying safe outputs in the GitHub Actions step summary but is no longer needed. The "Upload Safe Outputs" step remains intact and continues to upload safe outputs as artifacts for downstream jobs to consume.
