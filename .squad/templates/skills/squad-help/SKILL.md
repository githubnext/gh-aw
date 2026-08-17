---
name: "squad-help"
description: "How to actually use Squad — Squad is a custom Copilot agent (invoked via the task tool with agent_type='Squad'), not a skill. This file explains the right invocation paths for setting up a team, listing squad commands, and initializing Squad in a new project."
allowedTools: []
confidence: high
domain: squad-onboarding
---

# Skill: squad-help

> **Quick reference.** If you're reading this because a user said "use squad" or "squad" or "set up a squad", you're in the right place — read on for the correct invocation paths.

---

## Squad is a custom agent, not a skill

The Squad framework registers a **custom Copilot CLI agent** at `.github/agents/squad.agent.md`. The agent is named **`Squad`** and its description is *"Your AI team. Describe what you're building, get a team of specialists that live in your repo."*

Copilot CLI agents and skills are different things:

| Thing | How to invoke | Example |
|---|---|---|
| **Skill** | `skill(name)` tool call or natural-language match | `skill(squad-commands)` |
| **Agent** | `task` tool with `agent_type=<name>` | `task(name="...", agent_type="Squad", prompt="...")` |
| **Slash command** | Built-in CLI keyword | `/agent`, `/skills`, `/mcp` |

Calling `skill(Squad)` will fail with *"Skill not found: Squad"* because Squad is the agent, not a skill. (`/squad` as a slash command also does not exist — only built-in CLI keywords like `/agent`, `/skills`, `/mcp` are slash commands. There's no way to map a skill name to a slash command without a Copilot CLI feature change.)

---

## How to actually use Squad

Pick the path that matches the user's intent:

### A) Invoke the Squad coordinator agent (most common)

The Squad coordinator orchestrates a team of specialists. It routes work to the right agent, scaffolds a team if none exists, and enforces handoffs.

```text
task(
  name="<short-task-name>",
  agent_type="Squad",
  prompt="<what you want the team to do>"
)
```

Use this when the user says things like:
- *"Use Squad to build X"*
- *"Set up an AI team for this project"*
- *"Have the Squad coordinator design Y"*
- *"Spawn Squad"* / *"Squad, help me with ..."*

### B) See what Squad commands exist

The `squad-commands` skill is a categorized catalog of common Squad operations. The coordinator presents it as an interactive menu.

Trigger by natural-language match: `"squad commands"`, `"what can squad do"`, `"show me squad options"`, `"slash commands"`, `"what commands are available"`.

Use this when the user says things like:
- *"What can Squad do?"*
- *"Show me the squad commands"*
- *"squad help"*

### C) Initialize Squad in a fresh project

`squad init` is a **shell command**, not a tool call. The user runs it in their terminal in a project that has no `.squad/` directory yet.

```bash
squad init
```

Do **not** try to invoke this from inside an existing Copilot session — `.squad/` is already initialized if you're reading this file.

---

## What NOT to do

- ❌ Do not call `skill(Squad)`, `skill(squad)`, or `skill(squad-coordinator)` — Squad is not a skill.
- ❌ Do not type `/squad` expecting a slash command — slash commands are CLI keywords, not skill names. Use `/agent` (browse) or invoke the `Squad` agent via the `task` tool.
- ❌ Do not call `task(agent_type="Squad", …)` for tiny tasks the current agent can handle directly. Squad is for work that needs orchestration; trivial edits do not.

---

## How this skill was discovered

This skill ships from the Squad SDK templates and is wired into `MANIFEST_SKILL_NAMES`. It lives at `.copilot/skills/squad-help/SKILL.md` so the Copilot CLI's `/skills` loader picks it up alongside the other bundled Squad skills.

If you removed this skill on purpose, the model will fall back to its own reasoning and may make the lookup mistakes described above.

---

## See also

- `.github/agents/squad.agent.md` — the actual Squad coordinator agent
- `.copilot/skills/squad-commands/SKILL.md` — the command catalog
- `.copilot/skills/squad-conventions/SKILL.md` — conventions for working on the Squad codebase itself
- `.copilot/skills/squad-version-check/SKILL.md` — version-stamping mechanics
