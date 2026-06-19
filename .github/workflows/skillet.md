---
private: true
emoji: "🍳"
name: "Skillet"
description: Reviews pull requests by mapping any slash command to a matching repository skill under .github/skills
on:
  slash_command:
    strategy: centralized
    name: "*"
    events: [pull_request_comment, pull_request_review_comment]
permissions:
  contents: read
  pull-requests: read
  issues: read
  copilot-requests: write
engine:
  id: copilot
imports:
  - uses: shared/pr-review-base.md
    with:
      min-integrity: approved
  - shared/otlp.md
tools:
  github:
    mode: gh-proxy
    toolsets: [pull_requests, repos]
safe-outputs:
  messages:
    footer: "> 🍳 *Reviewed by [{workflow_name}]({run_url}) with `/${{ needs.pre_activation.outputs.skill_name }}`*{ai_credits_suffix}{history_link}"
    run-started: "🍳 [{workflow_name}]({run_url}) is loading `/${{ needs.pre_activation.outputs.skill_name }}` for this {event_type}..."
    run-success: "🍳 [{workflow_name}]({run_url}) completed the skill-guided review."
    run-failure: "⚠️ [{workflow_name}]({run_url}) {status} during the skill-guided review."
if: needs.pre_activation.outputs.activated == 'true' && needs.pre_activation.outputs.should_run == 'true'
timeout-minutes: 15
jobs:
  pre-activation:
    pre-steps:
      - name: Checkout skills directory
        uses: actions/checkout
        with:
          sparse-checkout: |
            .github/skills
          persist-credentials: false
    steps:
      - name: Match requested skill
        id: match_skill
        shell: bash
        run: |
          set -euo pipefail
          python <<'PY'
          import json
          import os
          import pathlib
          import re

          event_path = pathlib.Path(os.environ["GITHUB_EVENT_PATH"])
          workspace = pathlib.Path(os.environ["GITHUB_WORKSPACE"])
          event = json.loads(event_path.read_text())

          body = (
              event.get("comment", {}).get("body")
              or event.get("review", {}).get("body")
              or event.get("pull_request", {}).get("body")
              or event.get("issue", {}).get("body")
              or ""
          ).strip()

          match = re.match(r"^/([A-Za-z0-9][A-Za-z0-9._-]*)(?:\s+(.*))?$", body, re.S)
          command = match.group(1) if match else ""
          request_text = (match.group(2) or "").strip() if match else ""

          skills_dir = workspace / ".github" / "skills"
          available_skills = sorted(path.parent.name for path in skills_dir.glob("*/SKILL.md"))
          matched_skill_path = skills_dir / command / "SKILL.md"
          is_pr_context = bool(event.get("pull_request") or event.get("issue", {}).get("pull_request"))
          should_run = is_pr_context and command in available_skills

          if not is_pr_context:
              skip_reason = "Skillet only reviews pull requests."
          elif not command:
              skip_reason = "No slash command was found at the start of the comment."
          elif command not in available_skills:
              skip_reason = f"No repository skill matched /{command}."
          else:
              skip_reason = ""

          with open(os.environ["GITHUB_OUTPUT"], "a", encoding="utf-8") as output:
              output.write(f"should_run={'true' if should_run else 'false'}\n")
              output.write(f"skill_name={command}\n")
              output.write(f"skill_path={matched_skill_path if should_run else ''}\n")
              output.write(f"available_skills={','.join(available_skills)}\n")
              output.write("request_text<<__GHAW_REQUEST__\n")
              output.write(request_text + "\n")
              output.write("__GHAW_REQUEST__\n")
              output.write("skip_reason<<__GHAW_SKIP__\n")
              output.write(skip_reason + "\n")
              output.write("__GHAW_SKIP__\n")

          with open(os.environ["GITHUB_STEP_SUMMARY"], "a", encoding="utf-8") as summary:
              summary.write("### Skillet pre-activation\n")
              summary.write(f"- Command: `/{command or '<none>'}`\n")
              summary.write(f"- Pull request context: `{'yes' if is_pr_context else 'no'}`\n")
              summary.write(f"- Skill match: `{'yes' if should_run else 'no'}`\n")
              if skip_reason:
                  summary.write(f"- Skip reason: {skip_reason}\n")
          PY
    outputs:
      should_run: ${{ steps.match_skill.outputs.should_run }}
      skill_name: ${{ steps.match_skill.outputs.skill_name }}
      skill_path: ${{ steps.match_skill.outputs.skill_path }}
      available_skills: ${{ steps.match_skill.outputs.available_skills }}
      request_text: ${{ steps.match_skill.outputs.request_text }}
      skip_reason: ${{ steps.match_skill.outputs.skip_reason }}
---

# Skillet 🍳

You are a pull request reviewer that applies exactly one repository skill selected from the triggering slash command.

## Current Context

- **Repository**: ${{ github.repository }}
- **Pull Request**: #${{ github.event.issue.number || github.event.pull_request.number }}
- **Triggered by**: @${{ github.actor }}
- **Matched skill**: `/${{ needs.pre_activation.outputs.skill_name }}`
- **Skill file**: `${{ needs.pre_activation.outputs.skill_path }}`
- **Request text**: "${{ needs.pre_activation.outputs.request_text }}"
- **Original comment**: "${{ steps.sanitized.outputs.text }}"

## Required Flow

1. Read only the matched skill file at `${{ needs.pre_activation.outputs.skill_path }}` and apply its guidance directly.
2. Treat the request text after the slash command as the user’s specific review instruction. If it is empty, default to reviewing the PR with the matched skill’s standard guidance.
3. Fetch the pull request diff, changed files, and existing review comments with the GitHub pull request tools.
4. Review changed lines only and prioritize correctness, security, and maintainability risks.
5. Use `create-pull-request-review-comment` for line-specific findings and `submit-pull-request-review` exactly once for the overall verdict.
6. When there are no actionable issues, call `noop`. If you approve the PR, also call `create_check_run` with a short success summary.

## Review Guidelines

- Keep the review tightly scoped to what the matched skill is relevant for.
- Do not load unrelated skills.
- Keep visible review text brief and use `<details>` blocks for longer rationale or examples.
- Avoid repeating existing unresolved review comments unless you are materially adding new information.

{{#runtime-import shared/noop-reminder.md}}
