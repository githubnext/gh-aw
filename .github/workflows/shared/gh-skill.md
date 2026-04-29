---
# gh skill - Install Agent Skills
# Installs agent skills from GitHub repositories using `gh skill install`
# before the AI agent starts.
#
# Requirements:
#   GitHub CLI v2.90.0 or later (for `gh skill` support)
#
# Documentation: https://cli.github.com/manual/gh_skill_install
#
# Usage:
#   imports:
#     - uses: shared/gh-skill.md
#       with:
#         skills:
#           - github/awesome-copilot/documentation-writer
#           - github/awesome-copilot/code-review
#
# Skill format:
#   - owner/repo                     — install all skills from the repository
#   - owner/repo/skill-name          — install a specific skill (latest)
#   - owner/repo/skill-name@version  — install a pinned version

import-schema:
  skills:
    type: array
    items:
      type: string
    required: true
    description: >
      List of skills to install. Each entry is in the format owner/repo,
      owner/repo/skill-name, or owner/repo/skill-name@version.
      Examples: "github/awesome-copilot", "github/awesome-copilot/documentation-writer",
      "github/awesome-copilot/code-review@v1.2.0"

pre-agent-steps:
  - name: Install agent skills
    env:
      GH_TOKEN: ${{ secrets.GH_AW_PLUGINS_TOKEN || secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}
      GH_AW_SKILLS: ${{ github.aw.import-inputs.skills }}
    run: |
      set -euo pipefail
      skills_json="${GH_AW_SKILLS}"
      count=$(echo "$skills_json" | jq 'length')
      if [ "$count" = "0" ]; then
        echo "::error::shared/gh-skill.md import provided no skills. Add skills: <list> in the with: block."
        exit 1
      fi
      agent="${AI_AGENT:-github-copilot}"
      printf "::notice::Installing %d skill(s) for agent: %s\n" "$count" "$agent"
      while IFS= read -r skill_entry; do
        repo=$(echo "$skill_entry" | cut -d'/' -f1,2)
        skill_part=$(echo "$skill_entry" | cut -d'/' -f3-)
        if [ -n "$skill_part" ]; then
          echo "Installing skill: $repo $skill_part"
          gh skill install "$repo" "$skill_part" --agent "$agent"
        else
          echo "Installing all skills from: $repo"
          gh skill install "$repo" --agent "$agent"
        fi
      done < <(echo "$skills_json" | jq -r '.[]')
---

<!--
## Agent Skills

This shared workflow installs agent skills before the AI agent runs, using the
[`gh skill install`](https://cli.github.com/manual/gh_skill_install) command
(available in GitHub CLI v2.90.0+).

### How it works

Each skill in the `skills:` list is installed into the repository's
`.github/skills/` directory via `gh skill install`, making the skills available
to the AI agent during its session.

The target agent host is determined from the `AI_AGENT` environment variable if
set (for example, Claude Code automatically sets `AI_AGENT=claude-code` for its
subprocesses), otherwise defaults to `github-copilot`. Set `AI_AGENT` in your
workflow's `agent.env` block to override:

```yaml
agent:
  env:
    AI_AGENT: claude-code
```

### Usage

```yaml
imports:
  - uses: shared/gh-skill.md
    with:
      skills:
        - github/awesome-copilot/documentation-writer
        - github/awesome-copilot/code-review@v1.2.0
```

### Skill format

Each entry in `skills:` is one of:

| Format | Example | Effect |
|--------|---------|--------|
| `owner/repo` | `github/awesome-copilot` | Installs all skills from the repo |
| `owner/repo/skill-name` | `github/awesome-copilot/documentation-writer` | Installs a specific skill (latest) |
| `owner/repo/skill-name@version` | `github/awesome-copilot/code-review@v1.2.0` | Installs a pinned version |

### Authentication

Uses a token from the standard precedence chain:
`GH_AW_PLUGINS_TOKEN` → `GH_AW_GITHUB_TOKEN` → `GITHUB_TOKEN`.

For private skill repositories, configure `GH_AW_PLUGINS_TOKEN` or
`GH_AW_GITHUB_TOKEN` with appropriate access.

### Agent targeting

The `AI_AGENT` environment variable controls which agent host receives the
installed skills:

- `AI_AGENT=github-copilot` — GitHub Copilot (default when unset)
- `AI_AGENT=claude-code` — Claude Code
- `AI_AGENT=cursor` — Cursor
- `AI_AGENT=codex` — OpenAI Codex
- `AI_AGENT=gemini-cli` — Gemini CLI

Set `AI_AGENT` in `agent.env` to match the engine used by the workflow:

```yaml
engine: claude
agent:
  env:
    AI_AGENT: claude-code
imports:
  - uses: shared/gh-skill.md
    with:
      skills:
        - github/awesome-copilot/documentation-writer
```

### Network access

`gh skill install` downloads skill files from GitHub. No additional network
configuration is needed on standard GitHub-hosted runners. If your workflow
uses a strict network firewall, add `github.com` to the allowed domains:

```yaml
network:
  allowed:
    - github.com
```
-->
