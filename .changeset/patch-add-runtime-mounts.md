---
"gh-aw": patch
---

Add runtime mount manager that contributes host toolcache and runtime cache
folders into sandboxed agent containers so runtime binaries and caches are
available inside the agent environment. The compiler now auto-adds mounts when
runtimes are detected; smoke tests validate runtime commands inside containers.

When workflows require runtimes (Node, Python, Go, Ruby, Java, Dotnet, Bun,
Deno, UV, Elixir, Haskell), the compiler contributes read-only toolcache mounts
and read-write cache directories (e.g., `/opt/hostedtoolcache/node:ro`,
`/home/runner/.npm:rw`) to ensure runtime binaries and caches are available
inside sandboxed agent containers.

