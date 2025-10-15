## Changeset Format Reference

Based on https://github.com/changesets/changesets/blob/main/docs/adding-a-changeset.md

### Basic Format

```markdown
---
"gh-aw": patch
---

Fixed a bug in the component rendering logic
```

### Version Bump Types
- **patch**: Bug fixes, documentation updates, refactoring, non-breaking additions, new shared workflows (0.0.X)
- **minor**: Breaking changes in the cli (0.X.0)
- **major**: Major breaking changes. Very unlikely to be used often (X.0.0). You should be very careful when using this, it's probably a **minor**.

### Changeset File Structure
- Create file in `.changeset/` directory with descriptive kebab-case name
- Format: `<type>-<short-description>.md` (e.g., `minor-add-new-feature.md`)
- Use quotes around package names in YAML frontmatter
- Brief summary should be from PR title or first line of description
