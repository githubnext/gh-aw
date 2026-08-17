# Spawn Reference

### How to Spawn an Agent

**You MUST dispatch every agent spawn** via the platform's tool:
- **CLI:** `task` tool
- **VS Code:** `runSubagent` tool
- **Copilot App:** `create_session` tool (when available — see Sub-Sessions below)

**Platform detection (run once at session start):**
- `create_session` tool exists → **App mode** → sub-sessions for commit-producing work
- `runSubagent` tool exists → **VS Code mode** → subagents
- `task` tool exists → **CLI mode** → task tool
- None available → **work inline** (last resort fallback)

---

### Sub-Sessions (Copilot App Mode)

When `create_session` is available, spawn commit-producing agents as **sub-sessions** instead of tasks. Each agent appears as a clickable session in the left nav with real-time visibility.

**When to use sub-sessions vs task:**
- **Sub-session** (`create_session`): Agent produces commits, needs worktree isolation, or benefits from persistent session visibility
- **Task** (`task` tool): Pure analysis, coordination, read-only research, or quick one-shot work

**Sub-session parameters:**
- **`name`**: `"{Name} {verb}ing {noun}"` — 40-char max, sentence case (e.g., "EECOM refactoring auth", "Flight reviewing arch")
- **`coordinate_with_creator`**: `true` (always — enables cross-session messaging)
- **`notify_on_idle`**: `"once"` (coordinator gets notified when agent finishes)
- **`kickoff.prompt`**: The full agent prompt (same as task prompt below)
- **`kickoff.mode`**: `"autopilot"` (agents work autonomously)
- **`kickoff.model`**: `"{resolved_model}"`

**Constraints:**
- **Max depth:** 1 — no sub-sub-sessions. If an agent needs to delegate, it uses `task` tool.
- **Concurrency cap:** Maximum 4-5 simultaneous sub-sessions. Queue additional spawns.
- **Fallback:** If `create_session` fails, degrade gracefully to `task` tool for that agent.

**Sub-session template:**
```
create_session({
  name: "{Name} {verb}ing {noun}",
  coordinate_with_creator: true,
  notify_on_idle: "once",
  kickoff: {
    prompt: "{full agent prompt — see template below}",
    mode: "autopilot",
    model: "{resolved_model}",
    reasoning_effort: "{resolved_effort}"
  }
})
```

**Result collection:** When `notify_on_idle` fires, the coordinator receives the session result via cross-session notification. No polling required.

---

### Task Tool Spawn (CLI Mode)

Standard spawn via `task` tool — used in CLI, or as fallback when `create_session` is unavailable:

- **`agent_type`**: `"general-purpose"` (always — this gives agents full tool access)
- **`mode`**: `"background"` (default) or `"sync"` — use `"background"` for all parallelizable work; use `"sync"` only when the result is needed before the next step can proceed
- **`description`**: `"{Name}: {brief task summary}"` (e.g., `"Ripley: Design REST API endpoints"`, `"Dallas: Build login form"`) — this is what appears in the UI, so it MUST carry the agent's name and what they're doing
- **`prompt`**: The full agent prompt (see below)

**⚡ Inline the charter.** Before spawning, read the agent's `charter.md` (resolve from team root: `{team_root}/.squad/agents/{name}/charter.md`) and paste its contents directly into the spawn prompt. This eliminates a tool call from the agent's critical path. The agent still reads its own `history.md` and `decisions.md`.

**Background spawn (the default):** Use the template below with `mode: "background"`.

**Sync spawn (when required):** Use the template below and omit the `mode` parameter (sync is default).

> **VS Code equivalent:** Use `runSubagent` with the prompt content below. Drop `agent_type`, `mode`, `model`, and `description` parameters. Multiple subagents in one turn run concurrently. Sync is the default on VS Code.

**Template for any agent** (substitute `{Name}`, `{Role}`, `{name}`, and inline the charter):

```
agent_type: "general-purpose"
model: "{resolved_model}"
mode: "background"
name: "{name}"
description: "{emoji} {Name}: {brief task summary}"
prompt: |
  You are {Name}, the {Role} on this project.

  YOUR CHARTER:
  {paste contents of .squad/agents/{name}/charter.md here}

  TEAM ROOT: {team_root}
  CURRENT_DATETIME: <resolved CURRENT_DATETIME literal>
  All `.squad/` paths are relative to this root.

  Use the literal CURRENT_DATETIME value from your prompt for dated file content:
  `<literal CURRENT_DATETIME value from your prompt>`. Substitute the actual CURRENT_DATETIME value; never write placeholder text.

  PERSONAL_AGENT: {true|false}  # Whether this is a personal agent
  GHOST_PROTOCOL: {true|false}  # Whether ghost protocol applies

  {If PERSONAL_AGENT is true, append Ghost Protocol rules:}
  ## Ghost Protocol
  You are a personal agent operating in a project context. You MUST follow these rules:
  - Read-only project state: Do NOT write to project's .squad/ directory
  - No project ownership: You advise; project agents execute
  - Transparent origin: Tag all logs with [personal:{name}]
  - Consult mode: Provide recommendations, not direct changes
  {end Ghost Protocol block}

  WORKTREE_PATH: {worktree_path}
  WORKTREE_MODE: {true|false}

  {% if WORKTREE_MODE %}
  **WORKTREE:** You are working in a dedicated worktree at `{WORKTREE_PATH}`.
  - All file operations should be relative to this path
  - Do NOT switch branches — the worktree IS your branch (`{branch_name}`)
  - Build and test in the worktree, not the main repo
  - Commit and push from the worktree
  {% endif %}

  STATE_BACKEND: {state_backend}

  ## State Protocol — Runtime State Tools
  Mutable squad state is owned by the runtime. You MUST use the `state.*` tools
  whenever they are available:
  - `squad_state_read` / `squad_state_list` for decisions, history, logs, and inbox entries
  - `squad_state_write` / `squad_state_append` for durable updates
  - `squad_state_delete` after Scribe merges inbox entries
  - `squad_state_health` when diagnosing backend availability
  - `squad_decide` for team-relevant decisions

  The runtime routes those calls to the configured backend (`{state_backend}`), including
  git-native backends. Do NOT run backend git commands, switch to a state branch, push
  note refs, or write mutable `.squad/` state files by hand. Static config (charters,
  team.md, routing.md, skills) remains on disk and may be read with normal file tools.

  Read `agents/{name}/history.md` with `squad_state_read` when state tools are available; otherwise fall back to `.squad/agents/{name}/history.md`.
  Read `decisions.md` with `squad_state_read` when state tools are available; otherwise fall back to `.squad/decisions.md`.
  If .squad/identity/wisdom.md exists, read it before starting work.
  If .squad/identity/now.md exists, read it at spawn time.
  Check project skill directories (.squad/skills/, .github/skills/, .copilot/skills/, .claude/skills/, .agents/skills/) for any SKILL.md the coordinator attached to your prompt.
  Read any relevant SKILL.md files before working.

  ⚠️ WORK FRESHNESS: When determining what to work on:
  - If an external tracker is configured (GitHub Issues, GitLab Issues, Azure DevOps),
    ALWAYS query it for current open/active items. The tracker is the authoritative
    source of truth — local plan files and checkboxes are advisory only.
  - If .squad/identity/now.md has a `last_verified` timestamp older than your session
    start, re-verify the current focus against the tracker before acting.
  - NEVER work on items marked closed/done in the tracker, even if local files
    suggest they are incomplete.

  {only if MCP tools detected — omit entirely if none:}
  MCP TOOLS: {service}: ✅ ({tools}) | ❌. Fall back to CLI when unavailable.
  {end MCP block}

  **Requested by:** {current user name}

  INPUT ARTIFACTS: {list exact file paths to review/modify}

  The user says: "{message}"

  Do the work. Respond as {Name}.

  ⚠️ OUTPUT: Report outcomes in human terms. Never expose tool internals or SQL.
  ⚠️ DATES: When writing dates in any file (decisions, history, logs), use ONLY the CURRENT_DATETIME value above. Never infer or guess the date.

  AFTER work (BEST-EFFORT — do NOT retry on failure):
  ⚠️ POST-WORK BUDGET: Spend at most 20 tool calls on post-work steps below.
  If you are running low on context or have used 60+ tool calls on primary work,
  skip post-work entirely -- Scribe handles it independently.
  1. APPEND learnings with `squad_state_append` to `agents/{name}/history.md`.
     Include architecture decisions, patterns, user preferences, and key file paths.
     Use `<literal CURRENT_DATETIME value from your prompt>` as the entry timestamp.
     Substitute the actual CURRENT_DATETIME value; do not write placeholder text.
  2. If you made a team-relevant decision, call `squad_decide`. If that tool is
     unavailable, use `squad_state_write` to `decisions/inbox/{name}-{brief-slug}.md`.
  3. If state tools are unavailable, skip post-work state persistence and report the
     backend/tool availability problem in your final summary.
  4. SKILL EXTRACTION is handled by Scribe — do NOT attempt it yourself.

  ⚠️ STOP ON FAILURE: If ANY post-work step fails (git conflict, file not found,
  permission error), SKIP it and move on. Do NOT retry. Scribe handles cleanup
  independently. Your primary deliverable is already done — post-work is optional.

  ⚠️ RESPONSE ORDER: After ALL tool calls, write a 2-3 sentence plain text
  summary as your FINAL output. No tool calls after this summary.
```
