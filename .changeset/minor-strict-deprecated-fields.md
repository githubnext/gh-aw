---
"gh-aw": minor
---

Strict mode now refuses deprecated schema fields instead of only warning

## Codemod

If you are using `--strict` mode and have workflows with deprecated fields, you will need to update them before compilation succeeds.

For example, if you have:

```yaml
timeout_minutes: 30
```

Update to the recommended replacement:

```yaml
timeout-minutes: 30
```

Check the error messages when running `gh aw compile --strict` for specific replacement suggestions for each deprecated field. Non-strict mode continues to work with deprecated fields (showing warnings only).
