---
name: "cross-squad"
description: "Coordinating work across multiple Squad instances — discovery, delegation, and disambiguation when the user says 'squad' (the product) vs casual English 'group of agents'."
domain: "orchestration"
confidence: "medium"
source: "manual"
triggers:
  - "spawn N squads"
  - "spawn a squad"
  - "another squad"
  - "two squads of"
  - "second squad"
  - "fan out to squads"
  - "delegate to a squad"
  - "set up squads for"
  - "create a squad to review"
  - "ask the other squad"
tools:
  - name: "squad-discover"
    description: "List known squads and their capabilities"
    when: "When you need to find which squad can handle a task"
  - name: "squad-delegate"
    description: "Create work in another squad's repository"
    when: "When a task belongs to another squad's domain"
---

## Context

> **Read this FIRST any time the user says "squad" as a thing to spawn, delegate to, address, or fan out to** — e.g., *"spawn two squads of designers and devs"*, *"ask the other squad"*, *"delegate to a squad"*. In Squad-PRODUCT vocabulary, "squad" is a **peer** (an independent installation with its own `.squad/`, `team.md`, MCP server, and agents) — NOT a generic English synonym for "team" or "group". Do not fan out raw `task` agents inside your own coordinator context when the user means "another squad". Use the discovery and communication patterns below (and the companion `cross-squad-communication` skill for the actual protocols).

When an organization runs multiple Squad instances (e.g., platform-squad, frontend-squad, data-squad), those squads need to discover each other, share context, and hand off work across repository boundaries. This skill teaches agents how to coordinate across squads without creating tight coupling.

> **Companion skill — for protocol details:** `cross-squad-communication/SKILL.md` covers the four communication patterns (synchronous CLI, read-only knowledge query, git-based async, and GitHub-issue-based delegation) once a peer squad is discovered via the registry below. This skill answers "who?" — the companion answers "how?".

Cross-squad orchestration applies when:
- A task requires capabilities owned by another squad
- An architectural decision affects multiple squads
- A feature spans multiple repositories with different squads
- A squad needs to request infrastructure, tooling, or support from another squad

## Disambiguation: "squad" vs ad-hoc agents

When the user uses the word **"squad" / "squads"** or asks to **"spawn a team"**, the coordinator MUST treat it as a literal reference to a Squad install (a `.squad/` directory with its own roster, casting, and coordinator) — NOT as a casual synonym for "a group of sub-agents".

### Default behaviour (apply unless explicitly told otherwise)

| User says | Coordinator does |
|---|---|
| *"spawn two squads of X and Y"* / *"set up squads for X, Y, Z"* | Bootstrap N **real** Squad installs — separate folder + `git init` + `squad init` per squad — then use the cross-squad patterns below (`.squad/manifest.json`, `squad registry add`, `squad delegate`) and the protocols in the `cross-squad-communication` skill |
| *"ask the other squad about X"* / *"delegate to the data squad"* | Discover the peer via `squad registry list` (or by reading a known `.squad/manifest.json`), then use `cross-squad-communication` Pattern 0 / 1 / 2 / 3 — never re-implement the protocol with `task` |
| *"spawn a few agents to do X"* / *"have some agents review X"* / *"in parallel, get sub-agents to..."* | Ad-hoc `task` fan-out is fine — no `.squad/` bootstrap needed. This is the only path where raw `task` is appropriate when the user mentioned a multi-agent activity |

### Ambiguous? `ask_user`, never silently downgrade

If the request **could** be either interpretation AND bootstrapping real squads is non-trivial (more than one or two `squad init` runs), you MUST use the `ask_user` tool with a 2-choice prompt before proceeding:

```
question: "Should I create separate Squad installs or just dispatch ad-hoc agents?"
choices:
  - "Real squads — separate .squad/ per squad (heavier, persistent, can be re-engaged later)"
  - "Ad-hoc agents — one-shot `task` dispatch (lighter, ephemeral, no .squad/ created)"
```

The cost of asking is one `ask_user`. The cost of getting it wrong is the user has to redo the work. **Never silently pick the cheaper option just because it feels disproportionate for the task size — surface the trade-off and let the user pick.**

### Anti-patterns (every one of these is a real failure mode observed in production)

- **Calling `task` sub-agents "squad-alpha" / "squad-beta"** and treating them as squads. Naming something a squad doesn't make it one — a squad has its own `.squad/`, `team.md`, MCP server, and coordinator. If those aren't there, it's not a squad.
- **Matching a prior session's ad-hoc pattern without re-checking current intent.** If you see existing `reviews/squad-alpha/` folders from a previous run, that's a hint, NOT a contract — the user may have meant something different this time. Re-evaluate from scratch.
- **Silently choosing the cheaper interpretation because "bootstrapping two real squads for a 30-line app feels disproportionate".** That's a judgment call for the USER to make, not the coordinator. Use `ask_user`.
- **Loading the `cross-squad` skill, reading it, then doing `task` fan-out anyway** because the eager-execution / parallel-fan-out doctrine pulled you back. The disambiguation rule on this page OVERRIDES the generic fan-out doctrine when "squad" was the trigger.

## Patterns

### Discovery via Manifest
Each squad publishes a `.squad/manifest.json` declaring its name, capabilities, and contact information. Squads discover each other through two mechanisms:

1. **`.squad/squad-registry.json`** — **discovery-only.** Peer squads are findable via `squad discover` and addressable via `squad delegate`, but their skills/decisions/wisdom are NOT loaded into your coordinator. Manage with `squad registry add/list/remove`.
2. **`.squad/upstream.json`** — **discovery + inheritance.** Squads listed here are also discoverable, AND your coordinator inherits their skills/decisions/wisdom/routing at session start. Manage with `squad upstream add/list/remove/sync`.

Both forms read the peer's manifest via the same code path. The `path` field is the **repository root** (e.g. `../friend-repo`), and Squad appends `.squad/manifest.json` internally. Pointing at the `.squad/` directory works too — Squad accepts both forms (`readManifest` strips a trailing `.squad` if present).

```json
{
  "name": "platform-squad",
  "version": "1.0.0",
  "description": "Platform infrastructure team",
  "capabilities": ["kubernetes", "helm", "monitoring", "ci-cd"],
  "contact": {
    "repo": "org/platform",
    "labels": ["squad:platform"]
  },
  "accepts": ["issues", "prs"],
  "skills": ["helm-developer", "operator-developer", "pipeline-engineer"]
}
```

### Context Sharing
When delegating work, share only what the target squad needs:
- **Capability list**: What this squad can do (from manifest)
- **Relevant decisions**: Only decisions that affect the target squad
- **Handoff context**: A concise description of why this work is being delegated

Do NOT share:
- Internal team state (casting history, session logs)
- Full decision archives (send only relevant excerpts)
- Authentication credentials or secrets

### Work Handoff Protocol
1. **Check manifest**: Verify the target squad accepts the work type (issues, PRs)
2. **Create issue**: Use `gh issue create` in the target repo with:
   - Title: `[cross-squad] <description>`
   - Label: `squad:cross-squad` (or the squad's configured label)
   - Body: Context, acceptance criteria, and link back to originating issue
3. **Track**: Record the cross-squad issue URL in the originating squad's orchestration log
4. **Poll**: Periodically check if the delegated issue is closed/completed

### Feedback Loop
Track delegated work completion:
- Poll target issue status via `gh issue view`
- Update originating issue with status changes
- Close the feedback loop when delegated work merges

## Examples

### Registering a peer squad (no inheritance)
```bash
# Friend's repo is checked out at ../friend-platform/
squad registry add platform-squad ../friend-platform

# Verify
squad registry list
squad discover
```

### Discovering squads
```bash
# List all squads discoverable from registry + upstreams
squad discover

# Output:
#   platform-squad  →  org/platform  (kubernetes, helm, monitoring)
#   frontend-squad  →  org/frontend  (react, nextjs, storybook)
#   data-squad      →  org/data      (spark, airflow, dbt)
```

### Delegating work
```bash
# Delegate a task to the platform squad
squad delegate platform-squad "Add Prometheus metrics endpoint for the auth service"

# Creates issue in org/platform with cross-squad label and context
```

### Manifest in squad.config.ts
```typescript
export default defineSquad({
  manifest: {
    name: 'platform-squad',
    capabilities: ['kubernetes', 'helm'],
    contact: { repo: 'org/platform', labels: ['squad:platform'] },
    accepts: ['issues', 'prs'],
    skills: ['helm-developer', 'operator-developer'],
  },
});
```

## Anti-Patterns
- **Direct file writes across repos** — Never modify another squad's `.squad/` directory. Use issues and PRs as the communication protocol.
- **Tight coupling** — Don't depend on another squad's internal structure. Use the manifest as the public API contract.
- **Unbounded delegation** — Always include acceptance criteria and a timeout. Don't create open-ended requests.
- **Skipping discovery** — Don't hardcode squad locations. Use manifests and the discovery protocol.
- **Sharing secrets** — Never include credentials, tokens, or internal URLs in cross-squad issues.
- **Circular delegation** — Track delegation chains. If squad A delegates to B which delegates back to A, something is wrong.
