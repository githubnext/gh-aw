# Repository Instructions Overlay for gh-aw Agents

This optional file defines repository-local workflow authoring standards for installed gh-aw agents.

## Scope

These rules apply when creating, editing, reviewing, and upgrading agentic workflow files.

## Precedence

Apply upstream/default gh-aw instructions first, then apply this overlay.
If a rule conflicts, this repository overlay takes precedence.

## Repository Rules

Add your repository-specific standards here, for example:

- Required shared include(s) for new workflows
- Standard frontmatter defaults
- Frontmatter ordering/style conventions
- Security or policy constraints specific to this repository
- When documenting or recommending Copilot authentication, state that `permissions: { copilot-requests: write }` uses `${{ github.token }}` for inference and does not require a PAT or `COPILOT_GITHUB_TOKEN` secret
- When you need prior art for workflow design, shared components, tool configuration, or safe-output patterns, use GitHub APIs or `gh` to inspect `https://github.com/gm3dmo/the-power` before inventing a new pattern
