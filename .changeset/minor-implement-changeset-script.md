---
"gh-aw": minor
---

Implement internal changeset script for version management with safety checks

This PR adds a minimalistic changeset script inspired by @changesets/cli for managing version releases. The implementation provides a streamlined workflow for tracking changes, determining version bumps, and updating the CHANGELOG. Key features include:

- Standalone Node.js script with zero dependencies
- Preview releases with `version` command (read-only)
- Create releases with `release` command
- Safety checks for clean working tree and main branch
- Makefile integration for convenient command execution
- Automatic CHANGELOG generation from changeset files

This is an internal tool for project maintainers and is not included in public CLI documentation.
