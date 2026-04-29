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
#         engine: copilot            # optional: copilot (default), claude, codex, gemini, opencode
#         token: ${{ secrets.MY_TOKEN }}  # optional: defaults to GITHUB_TOKEN
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

  token:
    type: string
    required: false
    description: >
      GitHub token used to authenticate skill downloads. Pass via a secret
      expression, e.g. ${{ secrets.GH_TOKEN }}. Defaults to the built-in
      GITHUB_TOKEN when omitted (sufficient for public skill repositories).

  engine:
    type: string
    required: false
    description: >
      The gh-aw engine name. Determines which agent host receives the skills.
      Accepted values: copilot (default), claude, codex, gemini, opencode.
      Maps to the corresponding gh skill --agent value:
        copilot  → github-copilot
        claude   → claude-code
        codex    → codex
        gemini   → gemini-cli
        opencode → opencode

pre-agent-steps:
  - name: Install agent skills
    env:
      GH_TOKEN: ${{ github.aw.import-inputs.token || secrets.GITHUB_TOKEN }}
      GH_AW_SKILLS: ${{ github.aw.import-inputs.skills }}
      GH_AW_SKILL_ENGINE: ${{ github.aw.import-inputs.engine }}
    run: |
      set -euo pipefail
      skills_json="${GH_AW_SKILLS}"
      count=$(echo "$skills_json" | jq 'length')
      if [ "$count" = "0" ]; then
        echo "::error::shared/gh-skill.md import provided no skills. Add skills: <list> in the with: block."
        exit 1
      fi
      case "${GH_AW_SKILL_ENGINE:-copilot}" in
        claude)   agent="claude-code" ;;
        codex)    agent="codex" ;;
        gemini)   agent="gemini-cli" ;;
        opencode) agent="opencode" ;;
        *)        agent="github-copilot" ;;
      esac
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

### Usage

```yaml
engine: copilot
imports:
  - uses: shared/gh-skill.md
    with:
      skills:
        - github/awesome-copilot/documentation-writer
        - github/awesome-copilot/code-review@v1.2.0
```

For other engines, set `engine:` in the `with:` block to match your workflow
engine so skills are installed for the correct agent host:

```yaml
engine: claude
imports:
  - uses: shared/gh-skill.md
    with:
      engine: claude
      skills:
        - github/awesome-copilot/documentation-writer
```

### Inputs

| Input | Required | Description |
|-------|----------|-------------|
| `skills` | ✅ | List of skills to install (see formats below) |
| `engine` | No | gh-aw engine name — determines the `--agent` target (default: `copilot`) |
| `token` | No | GitHub token for downloading skills (default: built-in `GITHUB_TOKEN`) |

### Skill format

Each entry in `skills:` is one of:

| Format | Example | Effect |
|--------|---------|--------|
| `owner/repo` | `github/awesome-copilot` | Installs all skills from the repo |
| `owner/repo/skill-name` | `github/awesome-copilot/documentation-writer` | Installs a specific skill (latest) |
| `owner/repo/skill-name@version` | `github/awesome-copilot/code-review@v1.2.0` | Installs a pinned version |

### Engine → agent mapping

| `engine` input | `gh skill --agent` value |
|---------------|--------------------------|
| `copilot` (default) | `github-copilot` |
| `claude` | `claude-code` |
| `codex` | `codex` |
| `gemini` | `gemini-cli` |
| `opencode` | `opencode` |

### Authentication

Uses the token provided via the `token` input, falling back to the built-in
`GITHUB_TOKEN`. For private skill repositories, pass a token with read access:

```yaml
imports:
  - uses: shared/gh-skill.md
    with:
      token: ${{ secrets.MY_SKILLS_TOKEN }}
      skills:
        - my-org/private-skills/my-skill
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
