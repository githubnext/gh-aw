---
"gh-aw": patch
---

Redirect `DOTNET_INSTALL_DIR` and `GOPATH` for ARC/DinD runners so setup actions use writable, daemon-visible install paths instead of `/usr/share/dotnet` or other unsuitable defaults.
