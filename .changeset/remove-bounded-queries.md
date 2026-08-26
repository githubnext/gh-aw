---
"gh-aw": major
---

Remove `tools.github.bounded-queries` support with no legacy/back-compat handling.

Migration to `enclaves` requires a shape change:

```yaml
# Before
tools:
  github:
    bounded-queries:
      private-repos:
        - name: octo-org/private-service
          sensitivity: confidential
      runtime: sbx
      interpreter: claude

# After
enclaves:
  - script: null
    repos:
      - repo: octo-org/private-service
        sensitivity: confidential
```

Notes:
- `private-repos` moves to `enclaves[].repos`.
- You must define a script enclave entry (`script: null`).
- `interpreter` has no `enclaves` equivalent and cannot be carried over.
