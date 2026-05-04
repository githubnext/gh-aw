---
"gh-aw": patch
---

Fixed Copilot-engine workflows failing with `node: command not found` on GPU self-hosted runners.

Two complementary fixes:

1. **Export node's bin directory to PATH before `sudo -E awf`** (`copilot_engine_execution.go`): on runners where `sudo`'s `secure_path` strips toolcache additions made by `actions/setup-node`, node's parent directory is now explicitly prepended to `PATH` before AWF is invoked. AWF then captures this in `AWF_HOST_PATH` and the container sees `node` on `PATH`, satisfying any pre-flight check the AWF chroot performs.

2. **Extend `GetNpmBinPathSetup()` to also search `$HOME/.nvm/versions/node`** (`nodejs.go`): on runners where Node.js was installed via nvm the binary lives under `~/.nvm/versions/node/…/bin`, which was previously not in the `find` search set. This path is now included as a third search location alongside `/opt/hostedtoolcache` and `/home/runner/work/_tool`.

