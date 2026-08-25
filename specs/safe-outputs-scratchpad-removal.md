# Safe Outputs Scratchpad Removal Checklist

The deprecated [`scratchpad/safe-outputs-specification.md`](../scratchpad/safe-outputs-specification.md) must be removed on or before **2026-09-21**. The canonical specification is [`docs/src/content/docs/specs/safe-outputs-specification.md`](../docs/src/content/docs/specs/safe-outputs-specification.md).

## Removal checklist

- [ ] Before 2026-09-21, replace references to the deprecated scratchpad specification in doc-site navigation, workflow files, and internal links with the canonical specification.
- [ ] Owner: SPDD daily rotation. Before 2026-09-21, verify `grep -r "scratchpad/safe-outputs-specification.md" docs/ .github/` returns zero matches outside this removal notice before deleting the scratchpad file.
- [ ] On or before 2026-09-21, delete `scratchpad/safe-outputs-specification.md`.
- [ ] Verify that no remaining repository references resolve to the deleted scratchpad path.
