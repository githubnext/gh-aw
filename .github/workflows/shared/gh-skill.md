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
#         github-token: ${{ secrets.MY_TOKEN }}  # optional: defaults to GITHUB_TOKEN
#         upstream: false            # optional: pass false to skip --upstream (default: true)
#         skills:
#           - github/awesome-copilot/documentation-writer
#           - github/awesome-copilot/code-review@v1.2.0
#
# Skill format:
#   - owner/repo                     — install all skills from the repository
#   - owner/repo/skill-name          — install a specific skill (latest, with --upstream)
#   - owner/repo/skill-name@ref      — install a pinned version (passes --pin ref)
#
# Engine mapping (from $GH_AW_ENGINE_ID, set by the compiler):
#   copilot  → github-copilot
#   claude   → claude-code
#   codex    → codex
#   gemini   → gemini-cli
#   opencode → opencode

import-schema:
  skills:
    type: array
    items:
      type: string
    required: true
    description: >
      List of skills to install. Each entry is in the format owner/repo,
      owner/repo/skill-name, or owner/repo/skill-name@ref.
      Examples: "github/awesome-copilot", "github/awesome-copilot/documentation-writer",
      "github/awesome-copilot/code-review@v1.2.0"

  github-token:
    type: string
    required: false
    description: >
      GitHub token used to authenticate skill downloads. Pass via a secret
      expression, e.g. ${{ secrets.GH_TOKEN }}. Defaults to the built-in
      GITHUB_TOKEN when omitted (sufficient for public skill repositories).

  upstream:
    type: boolean
    required: false
    description: >
      When true (default), passes --upstream to gh skill install so skills are
      updated from their upstream source. Set to false to skip the upstream
      update and use the locally cached version.

pre-agent-steps:
  - name: Install agent skills
    env:
      GH_TOKEN: ${{ github.aw.import-inputs.github-token || secrets.GITHUB_TOKEN }}
      GH_AW_SKILLS: ${{ github.aw.import-inputs.skills }}
      GH_AW_SKILL_UPSTREAM: ${{ github.aw.import-inputs.upstream }}
    run: |
      set -euo pipefail
      echo "::group::gh skill prerequisites"
      gh --version
      echo "::endgroup::"
      skills_json="${GH_AW_SKILLS}"
      count=$(echo "$skills_json" | jq 'length')
      if [ "$count" = "0" ]; then
        echo "::error::shared/gh-skill.md import provided no skills. Add skills: <list> in the with: block."
        exit 1
      fi
      # GH_AW_ENGINE_ID is injected by the compiler as a job-level env var
      case "${GH_AW_ENGINE_ID:-copilot}" in
        claude)   agent="claude-code" ;;
        codex)    agent="codex" ;;
        gemini)   agent="gemini-cli" ;;
        opencode) agent="opencode" ;;
        *)        agent="github-copilot" ;;
      esac
      # --upstream is enabled by default; set upstream: false in the with: block to disable
      upstream_flag="--upstream"
      if [ "${GH_AW_SKILL_UPSTREAM:-true}" = "false" ]; then
        upstream_flag=""
      fi
      printf "::notice::Installing %d skill(s) for agent: %s\n" "$count" "$agent"
      while IFS= read -r skill_entry; do
        repo=$(echo "$skill_entry" | cut -d'/' -f1,2)
        skill_part=$(echo "$skill_entry" | cut -d'/' -f3-)
        if [ -n "$skill_part" ]; then
          # Split skill_name and optional @ref pin
          if [[ "$skill_part" == *@* ]]; then
            skill_name="${skill_part%%@*}"
            pin_ref="${skill_part#*@}"
            echo "Installing skill: $repo $skill_name (pinned to $pin_ref)"
            gh skill install "$repo" "$skill_name" --agent "$agent" --pin "$pin_ref" --allow-hidden-dirs --force $upstream_flag
          else
            echo "Installing skill: $repo $skill_part"
            gh skill install "$repo" "$skill_part" --agent "$agent" --allow-hidden-dirs --force $upstream_flag
          fi
        else
          echo "Installing all skills from: $repo"
          gh skill install "$repo" --agent "$agent" --allow-hidden-dirs --force $upstream_flag
        fi
      done < <(echo "$skills_json" | jq -r '.[]')
---
