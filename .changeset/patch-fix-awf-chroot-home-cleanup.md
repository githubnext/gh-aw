---
"gh-aw": patch
---

Fix EACCES errors when cleaning up root-owned AWF chroot home directories. When AWF ran with `--enable-host-access` on a shared GitHub-hosted runner, it created `/tmp/awf-*-chroot-home` directories whose files were owned by root. The Copilot CLI cleanup path (`rimrafSync`) ran as the non-root runner user and failed with `EACCES`, causing the "engine terminated unexpectedly" failure. This change adds `sudo rm -rf /tmp/awf-*-chroot-home` cleanup to `post.js`, `clean.sh`, and `install_copilot_cli.sh`, mirroring the existing pattern used for `/tmp/gh-aw`.
