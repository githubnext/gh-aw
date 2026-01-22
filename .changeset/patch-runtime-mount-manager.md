---
"gh-aw": patch
---

Add a runtime mount manager that contributes well-known host toolcache
and cache folders into sandboxed agent containers so runtime binaries
and caches (Node, Python, Go, Ruby, Java, Dotnet, Bun, Deno, UV,
Elixir, Haskell) are available inside the agent environment.

This change centralizes runtime mount definitions, hooks them into the
compiler so mounts are automatically added when runtimes are detected,
and adds smoke-test validation to ensure runtime commands (for example
`npm ls`) work inside containers.

