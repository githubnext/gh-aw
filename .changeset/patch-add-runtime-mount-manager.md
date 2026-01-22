---
"gh-aw": patch
---

Add a runtime mount manager that contributes host toolcache and runtime cache
folders (Node, Python, Go, Ruby, Java, Dotnet, Bun, Deno, UV, Elixir,
Haskell) into sandboxed agent containers so runtime binaries and caches are
available inside the agent environment. The compiler now auto-adds mounts when
runtimes are detected; smoke tests validate runtime commands inside containers.

