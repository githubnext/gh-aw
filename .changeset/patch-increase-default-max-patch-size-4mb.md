---
"gh-aw": patch
---
Increase default `max-patch-size` from 1024 KB (1 MB) to 4096 KB (4 MB) for `create-pull-request` and `push-to-pull-request-branch` safe outputs. The previous 1 MB default was too small for workflows that generate larger files automatically (e.g. lock files, generated code). The new 4 MB default reduces the number of unexpected patch-size failures without requiring manual configuration changes.
