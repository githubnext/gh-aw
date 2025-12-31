---
"gh-aw": patch
---

Refactor docker image download inline script to an external shell script at `actions/setup/sh/download_docker_images.sh`.

The generated workflow now calls `bash /tmp/gh-aw/actions/download_docker_images.sh` with image arguments instead of embedding the pull-and-retry function inline. No behavior changes.

