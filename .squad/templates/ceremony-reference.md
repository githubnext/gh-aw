# Ceremony Reference

On-demand reference for ceremony configuration, facilitator spawn, and execution rules.

## Config Format

Ceremonies are declared in `.squad/ceremonies.md`. Each ceremony is a section with a table of fields:

```markdown
## {CeremonyName}

| Field | Value |
|-------|-------|
| **Trigger** | auto \| manual |
| **When** | before \| after |
| **Condition** | {when auto-triggered: natural language condition} |
| **Facilitator** | lead \| {specific-agent} |
| **Participants** | all-relevant \| all-involved \| {comma-separated names} |
| **Time budget** | focused \| extended |
| **Enabled** | ✅ yes \| ❌ no |

**Agenda:**
1. {Step 1}
2. {Step 2}
...
```

### Field Definitions

| Field | Values | Meaning |
|-------|--------|---------|
| Trigger | `auto` | Fires automatically when Condition matches |
| Trigger | `manual` | Only when user says "run {ceremony}" |
| When | `before` | Runs before work batch spawns |
| When | `after` | Runs after work batch completes |
| Condition | free text | Evaluated against current task context |
| Facilitator | agent name | Who runs the meeting |
| Participants | selector | Who attends |
| Time budget | `focused` | Keep it short — key decisions only |
| Time budget | `extended` | Thorough discussion — all angles |
| Enabled | boolean | Skip disabled ceremonies entirely |

## Facilitator Spawn Template

When a ceremony triggers, spawn the facilitator (sync) with this prompt structure:

```
You are {FacilitatorName}, facilitating the "{CeremonyName}" ceremony.

PARTICIPANTS: {participant list}
TRIGGER CONDITION: {what triggered this ceremony}
AGENDA:
{numbered agenda items from config}

RULES:
- Follow the agenda in order.
- For each agenda item, spawn relevant participants as sub-tasks to gather their input.
- Synthesize participant input into clear decisions and action items.
- Keep to the time budget: {focused|extended}.
- Output a structured summary at the end.

TASK CONTEXT:
{description of the work that triggered this ceremony}
```

## Execution Rules

1. **Before ceremonies** fire AFTER routing decisions but BEFORE agent spawn. The ceremony summary is included in all subsequent work-batch spawn prompts.
2. **After ceremonies** fire when ALL agents in the batch have completed (success or failure).
3. **Manual ceremonies** fire only on explicit user request ("run retro", "do a design review").
4. **Cooldown:** After a ceremony completes, skip auto-trigger checks for the immediately following step. This prevents ceremony loops.
5. **Participant resolution:**
   - `all-relevant` → agents routed to the current task
   - `all-involved` → agents that participated in the completed batch
   - Named agents → spawn only those specific agents
6. **Scribe integration:** Spawn Scribe (background) at ceremony start to record decisions and action items.
7. **Output format:**
   ```
   📋 {CeremonyName} completed — facilitated by {Facilitator}.
   Decisions: {count} | Action items: {count}.
   ```
8. **Failure handling:** If the facilitator fails or times out, log a warning and proceed with work. Ceremonies must never block the pipeline indefinitely.
