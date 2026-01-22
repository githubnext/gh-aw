---
"gh-aw": patch
---

Add runtime mount manager and compiler integration that contributes host toolcache
and runtime cache folders (Node, Python, Go, Ruby, Java, Dotnet, Bun, Deno, UV,
Elixir, Haskell) into sandboxed agent containers so runtime binaries and caches
are available inside the agent environment. Compiler auto-adds mounts when
runtimes are detected; smoke tests validate runtime commands inside containers.

*** End Patch
