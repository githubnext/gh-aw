---
"gh-aw": patch
---
Remove Serena local mode, delete the unpinned start_serena_server.sh script, and add a codemod that automatically migrates `tools.serena.mode: local` to `tools.serena.mode: docker` so workflows keep compiling with the secure container-only setup.
