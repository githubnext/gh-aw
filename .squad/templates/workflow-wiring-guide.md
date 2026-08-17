# Squad Workflow Wiring Guide

> How to wire up new team members, reviewer gates, and custom workflows so they actually get enforced by the coordinator — even in a clean session with no prior memory.

## Why This Guide Exists

The Squad framework (`squad.agent.md`) provides generic orchestration primitives. **It does not prescribe a specific workflow.** Your project's workflow — whether that's "all code goes through PRs and reviews" or "just commit to main" — must be wired into project-level configuration files.

If a workflow rule exists only in someone's memory, in a chat transcript, or in `decisions.md` but NOT in a configuration file the coordinator reads at decision time — **it will not be followed in a clean session.**

### Why Existing Patterns Aren't Enough

The Squad framework already has concepts for routing tables, reviewer roles, and ceremonies. But having these concepts does NOT mean they work automatically:

- **Adding a reviewer to the roster ≠ enforcing reviews.** A reviewer can be on the roster with "Reviewer" as their role and never review a single PR — because no RULE in `routing.md` tells the coordinator to route PRs to them. The roster says WHO exists. Rules say WHAT they enforce.

- **Capturing a decision ≠ enforcing it.** `decisions.md` may contain "every change must go through a PR" and "only {ReviewerName} closes PRs." These can get buried in a large file that the coordinator reads for context but doesn't treat as enforcement rules. A decision is a historical record. A routing rule is an enforceable constraint.

- **Describing a lifecycle ≠ wiring it.** `squad.agent.md` describes issue→branch→PR→review→merge. But if the After Agent Work section (the flow the coordinator actually follows after every agent completes) has no push/PR/review step, the lifecycle is described conceptually but never connected to the coordinator's actual decision flow.

**The pattern that works:** A numbered rule in `routing.md` → Rules section. The coordinator reads this section, treats each rule as a constraint, and follows them. If your workflow isn't a numbered rule, it's a suggestion.

---

## Configuration Surface Area

The coordinator reads these files to decide how to behave. If your workflow isn't encoded in one of these, it doesn't exist.

| File | What It Controls | Read When |
|------|-----------------|-----------|
| `routing.md` | WHO handles what, behavioral RULES, reviewer GATES | Every session start, before every routing decision |
| `ceremonies.md` | Auto-triggered ceremonies (before/after work batches) | Before spawning work batches, after completion |
| `templates/issue-lifecycle.md` | Git workflow: push, PR, review, merge, issue closure | When spawning agents for issue-linked work |
| Agent `charter.md` | Per-agent identity, boundaries, behavior | Inlined into every spawn prompt |
| `team.md` | Roster, member capabilities | Session start |
| `decisions.md` | Captured decisions and directives | Read by agents at spawn time |

### How They Interact

```
User request arrives
  → Coordinator reads routing.md (WHO handles this?)
  → Coordinator checks ceremonies.md (any auto-triggered "before" ceremony?)
  → Coordinator reads agent charter.md (inline into spawn prompt)
  → If issue-linked: coordinator reads issue-lifecycle.md (add ISSUE CONTEXT to spawn prompt)
  → Agent works
  → Coordinator follows After Agent Work flow
  → Coordinator checks ceremonies.md (any auto-triggered "after" ceremony?)
  → Coordinator checks routing.md Rules section (any post-work rules to enforce?)
```

**The critical insight:** `routing.md` Rules section and `ceremonies.md` are the two enforcement mechanisms. If a rule isn't in one of these, the coordinator has no way to know about it.

---

## How to Wire Up a New Team Member

### Step 1: Create the member (files)

```
.squad/agents/{name}/
  charter.md    ← Identity, role, boundaries, what they own
  history.md    ← Seeded with project context from team.md
```

### Step 2: Add to roster (`team.md`)

Add a row to the `## Members` table:
```
| {emoji} {Name} | {Role} | `.squad/agents/{name}/charter.md` | ✅ Active |
```

### Step 3: Add routing entry (`routing.md`)

Add a row to the routing table:
```
| {Work Type} | {emoji} {Name} | {Output Location} | {Examples} |
```

### Step 4: Add issue routing (if applicable)

Add to the Issue Routing table in `routing.md`:
```
| squad:{name} | {Description of work} | {emoji} {Name} |
```

### Step 5: Add to casting registry

Update `.squad/casting/registry.json` with the new entry.

### Step 6: Wire any gates (if this member is a reviewer/gate)

**This is the step most people miss.** If the new member should review or gate other members' work, you need to wire enforcement. See "How to Wire Up a Reviewer Gate" below.

---

## How to Wire Up a Reviewer Gate

A reviewer gate means: "Agent X must review Agent Y's output before it proceeds." The framework supports this but does NOT automatically enforce it. You must wire it.

### Option A: Routing Rule (recommended for simple gates)

Add to `routing.md` → `## Rules` section:

```markdown
N. **{GateName} Gate** — Every {output type} from {Author} MUST be reviewed by {ReviewerName} before {next step}. The coordinator routes {Author}'s output to {ReviewerName} (sync spawn), collects the verdict, and only proceeds if approved. On rejection, {Author} revises based on {ReviewerName}'s feedback.
```

**Example — reviewer for all PRs:**
```markdown
9. **{ReviewerName} PR Gate** — Every PR created by any agent MUST be reviewed by {ReviewerName} before merge. The coordinator spawns {ReviewerName} (sync) with the PR diff, collects APPROVE/REJECT verdict. On rejection, the original author addresses feedback.
```

**Example — design review gate:**
```markdown
10. **{DesignReviewer} Design Gate** — Every design doc produced by the architect MUST be reviewed by {DesignReviewer} before implementation begins. {DesignReviewer} always rejects the first draft on concept/approach. Implementation is BLOCKED until {DesignReviewer} approves.
```

**Why this works:** The coordinator reads the Rules section before and after every work batch. Rules are behavioral constraints the coordinator must follow.

### Option B: Ceremony (recommended for multi-participant gates)

Add to `ceremonies.md` using the Markdown table format the file uses:

```markdown
## Design Review

| Field | Value |
|-------|-------|
| **Trigger** | auto |
| **When** | before |
| **Condition** | task involves implementing a design doc |
| **Facilitator** | {DesignReviewer} |
| **Participants** | Architect, {DesignReviewer} |
| **Time budget** | focused |
| **Enabled** | ✅ yes |

**Agenda:**
1. Read the design doc
2. Challenge the premise and approach
3. Demand alternatives and evidence
4. Verdict: APPROVE or REJECT
```

**Why this works:** The coordinator checks ceremonies.md for `before` ceremonies whose condition matches the current task. If matched, the ceremony runs before work begins.

### Option A vs Option B

| Use Case | Use Routing Rule | Use Ceremony |
|----------|-----------------|--------------|
| Simple 1-on-1 review (reviewer → author) | ✅ | Overkill |
| Multi-participant alignment (3+ agents) | Too simple | ✅ |
| Needs structured facilitation | No | ✅ |
| Must run automatically before specific work | Either works | ✅ |
| One-line behavioral constraint | ✅ | Overkill |

---

## How to Wire Up an Issue Lifecycle (Git Workflow)

This is where you define what happens after an agent completes work on a GitHub issue. The framework references `.squad/templates/issue-lifecycle.md` but does NOT create it — you must create it yourself.

> **⚠️ This file is required if your project uses GitHub Issues Mode.** Without it, the coordinator has no post-work steps for push/PR/review and will treat agent commit as "done."

See `.squad/templates/issue-lifecycle.md` for the full template if your project already has one. If not, create it following the pattern below.

### Step 1: Create `templates/issue-lifecycle.md`

Create `.squad/templates/issue-lifecycle.md` with your project's git workflow. At minimum it should include:

- An ISSUE CONTEXT block template (for spawn prompts)
- Coordinator post-work steps (verify push → verify PR → route to reviewer → merge on approval)
- Issue closure rules (PR merge auto-close vs manual close)
- Worktree requirements (if applicable)

### Step 2: Add enforcement rules to `routing.md`

Add numbered rules to the `## Rules` section that reference the lifecycle:

```markdown
N.  **Issue lifecycle enforcement** — all issue-linked work follows the lifecycle
    in `.squad/templates/issue-lifecycle.md`. The coordinator adds the ISSUE CONTEXT
    block to spawn prompts and follows the post-work steps (verify push → verify PR
    → route to reviewer → merge on approval). Read `issue-lifecycle.md` before
    spawning any agent for issue work.

N+1. **{ReviewerName} PR Gate** — every PR created by any agent MUST be reviewed
    by {ReviewerName} before merge. The coordinator spawns {ReviewerName} (sync)
    with the PR diff. On REJECT, the original author addresses feedback. On APPROVE,
    the coordinator merges. No PR merges without {ReviewerName}'s approval.

N+2. **Issue closure restriction** — issues that produced files (code, docs, scripts,
    designs, tests) close ONLY via PR merge auto-close ("Closes #N" in PR body).
    Never use `gh issue close` for file-producing work. Exception: tracking/strategic
    issues and superseded issues may be closed with a comment.

N+3. **Worktree for all file-producing work** — every task that creates or modifies
    files (including documentation) requires a worktree. Exceptions: read-only queries,
    Scribe (.squad/ state), pure analysis producing no files.
```

### Step 3: Verify your wiring

After creating both files, run the verification checklist (below) to confirm a clean session coordinator would follow the lifecycle.

---

## How to Wire Up a Custom Workflow Step

If you need something that isn't a reviewer gate or issue lifecycle — for example, "always run tests before pushing" or "docs must be reviewed by the author before merge" — here's where to put it:

### If it's a behavioral rule the coordinator should always follow:
→ Add to `routing.md` → `## Rules` section

### If it should trigger automatically before/after specific work:
→ Add to `ceremonies.md` as a `before` or `after` ceremony

### If it's something agents should do as part of their work:
→ Add to the agent's `charter.md` under a new section

### If it's something that applies only to issue-linked work:
→ Add to `templates/issue-lifecycle.md`

### If it's a team-wide constraint that should be visible to all agents:
→ Capture as a decision in `decisions.md` (via directive or decision inbox)

---

## Verification Checklist

After wiring any new member, gate, or workflow, verify:

- [ ] **Clean session test:** Start a new session (no memory). Give a task. Does the coordinator follow the new rule?
- [ ] **File completeness:** Is the rule/gate/workflow encoded in a file the coordinator reads? (routing.md, ceremonies.md, issue-lifecycle.md, charter.md)
- [ ] **No verbal-only rules:** Is there anything the coordinator should do that's only in chat history or your memory? If yes, it will be lost on session restart.
- [ ] **Gate enforcement:** If you added a reviewer gate, does the routing.md Rules section or ceremonies.md explicitly say the coordinator must route to the reviewer? "Having a reviewer on the roster" is not the same as "enforcing that they review."
- [ ] **Issue lifecycle:** If your project uses PRs, does `templates/issue-lifecycle.md` exist? Does routing.md reference it?

---

## Common Mistakes

1. **Adding a reviewer to the roster but not wiring a gate.** Having a reviewer on the team doesn't mean they review anything. You must add a rule in routing.md that says "route PRs to {ReviewerName}."

2. **Closing issues via `gh issue close` instead of PR merge.** If your project uses PRs, issue closure should happen via "Closes #N" in the PR body. Wire this in issue-lifecycle.md.

3. **Writing docs/scripts directly on main.** If your project requires branches for all changes, the worktree gate must apply to ALL file-producing work — including docs. Make this explicit in routing.md Rules.

4. **Assuming the coordinator remembers verbal instructions.** Each session starts fresh. If you told the coordinator "always use opus" in session 1, session 2 won't know unless it's in decisions.md or routing.md.

5. **Not creating `issue-lifecycle.md`.** The framework references it but doesn't create it. If your project uses GitHub Issues Mode, create this template.

6. **Capturing a decision but never encoding it as a rule.** `decisions.md` is a historical record. The coordinator reads it for context but doesn't treat entries as enforceable constraints. If a decision should be enforced, it must become a numbered rule in `routing.md` Rules section.

---

## Decisions Audit

Periodically scan `decisions.md` for directives that should be routing rules but aren't:

1. Search for phrases like "always", "never", "must", "every", "required"
2. For each match, ask: "Is this enforced by a numbered rule in routing.md?"
3. If no → either add a rule, or accept that it's advisory-only
4. If yes → verify the rule text matches the decision

This prevents `decisions.md` from becoming a graveyard of good intentions that the coordinator reads but doesn't act on.

---

## Appendices

For detailed end-to-end walkthroughs of specific wiring scenarios, see:

- **[Appendix A: Wiring a Code Reviewer](workflow-wiring-appendix-a-code-reviewer.md)** — Full walkthrough of adding a code reviewer member and wiring their gate so it actually gets enforced. Includes every file that needs modification with exact content.

- **[Appendix B: Wiring a Documenter/Librarian](workflow-wiring-appendix-b-documenter.md)** — Full walkthrough of adding a documenter role that ensures all significant changes are documented. Shows a follow-up trigger pattern rather than a gate pattern.
