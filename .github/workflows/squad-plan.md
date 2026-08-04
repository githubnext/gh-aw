---
private: true
emoji: "🧑‍🤝‍🧑"
name: Squad Plan
description: Uses Squad to plan an issue from the /squad-plan slash command and create Copilot-ready sub-issues
on:
  slash_command:
    strategy: centralized
    name: squad-plan
    events: [issue_comment]
permissions:
  contents: read
  issues: read
  pull-requests: read
  copilot-requests: write
network:
  allowed:
    - defaults
imports:
  - shared/squad.md
tools:
  github:
    mode: gh-proxy
    toolsets: [default]
safe-outputs:
  create-issue:
    title-prefix: "[squad-plan] "
    labels: [cookie]
    assignees: [copilot]
    group: true
    max: 8
pre-agent-steps:
  - name: Check Squad files
    run: |
      set -euo pipefail
      GH_AW_SAFE_OUTPUTS="${GH_AW_SAFE_OUTPUTS:-${RUNNER_TEMP:-/tmp}/gh-aw/safeoutputs/outputs.jsonl}"
      mkdir -p "$(dirname "$GH_AW_SAFE_OUTPUTS")"

      missing=()
      for path in .squad/team.md .github/agents/squad.agent.md; do
        if [ ! -f "$path" ]; then
          missing+=("$path")
        fi
      done

      if [ "${#missing[@]}" -gt 0 ]; then
        message="Squad files are unavailable: ${missing[*]}. The activation-job bootstrap step likely failed."
        printf '{"type":"noop","message":"%s"}\n' "$message" >> "$GH_AW_SAFE_OUTPUTS"
        echo "$message"
      fi
---

# Squad Plan

Use the Squad (https://github.com/bradygaster/squad) team to review the issue
where `/squad-plan` was invoked, produce an implementation plan, and create
small Copilot-ready sub-issues.

## Task

1. Confirm Squad files are available before delegating work to the team:
   `.squad/team.md` and `.github/agents/squad.agent.md` should exist. If
   either file is missing, call `noop` with a short explanation instead of
   proceeding.
2. Review the triggering issue (#${{ github.event.issue.number }}) and the
   slash-command comment for any additional guidance.
3. Work with the Squad team to produce a concise implementation plan for the
   issue, including scope, sequencing, dependencies, and validation criteria.
4. Create at most 8 small, independently actionable sub-issues from the plan.
   With issue grouping enabled, create only the sub-issues; the safe-output
   runtime will group them under a parent tracking issue automatically.
5. Each sub-issue must be suitable for assignment to GitHub Copilot coding
   agent one by one, carry the configured `cookie` label, and include:
   - a clear objective
   - relevant issue context
   - concrete implementation guidance
   - acceptance criteria
   - any known ordering or dependency notes

## Safe Outputs

- Use `create_issue` for each planned sub-issue.
- Do not create a separate parent issue and do not use `parent` or
  `temporary_id`; grouping is automatic.
- If the issue is already fully planned or no useful sub-issues are needed,
  call `noop` with a short explanation.
- If Squad cannot produce a usable plan, call `noop` instead of filing
  incomplete issues.
