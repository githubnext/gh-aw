---
description: Integration test for shared/gh-skill.md — verifies gh skill install runs correctly as a pre-agent step
on:
  workflow_dispatch:
permissions:
  contents: read
name: Smoke gh-skill
engine: copilot
imports:
  - uses: shared/gh-skill.md
    with:
      skills:
        - github/awesome-copilot/dependabot
timeout-minutes: 10
strict: true
---

# Integration Test: shared/gh-skill.md

Verify that the `shared/gh-skill.md` shared workflow correctly installed the `github/awesome-copilot/dependabot` skill before this agent started.

## Test Procedure

1. Check that the skill directory exists under `.github/skills/` or the agent's skill path (typically `~/.github/copilot/skills/` or similar).
2. Use the `bash` tool to list skill files installed by the pre-agent step, e.g.:
   ```
   find ~/.github -name "dependabot*" -o -name "SKILL.md" 2>/dev/null | head -20
   ```
3. Report whether the skill was found.

## Expected Output

- ✅ Skill installed: list the path(s) found
- ❌ Skill not found: report the error

Output a brief one-paragraph summary of whether the integration test passed or failed.
