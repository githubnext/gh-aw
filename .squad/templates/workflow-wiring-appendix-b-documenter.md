# Appendix B: Wiring a Documenter/Librarian — Complete Walkthrough

> End-to-end example of adding a documenter role that ensures significant changes are documented. This is a FOLLOW-UP TRIGGER pattern — not a gate (which blocks), but an automatic downstream task that fires after work completes.

## The Problem This Solves

Your project has agents building features, fixing bugs, and writing tools. But nobody documents what was built, how to use it, or what changed. Documentation happens only when someone explicitly asks — and by then, the context is lost.

A documenter/librarian role solves this by automatically evaluating whether completed work needs documentation and producing it if so.

## Gate vs Follow-Up Trigger

| Pattern | Blocks work? | When it runs | Example |
|---------|-------------|-------------|---------|
| **Gate** (Appendix A) | Yes — work cannot proceed without approval | Before merge | Code reviewer must approve PR |
| **Follow-up trigger** | No — work proceeds, documentation happens in parallel | After merge | Documenter evaluates if docs are needed |

A documenter is typically a follow-up trigger, not a gate. You don't want documentation review to block a hotfix from merging. But you DO want documentation to happen automatically after significant changes.

## Step-by-Step Walkthrough

### Step 1: Create the documenter's identity

Create `.squad/agents/{name}/charter.md`:

```markdown
# {Name} — Documenter

## Identity
- **Name:** {Name}
- **Role:** Documenter / Librarian
- **Expertise:** Documentation, guides, READMEs, changelogs, knowledge management
- **Style:** Clear, thorough, user-focused. Makes complex things understandable.

## What I Own
- Evaluating whether completed work needs documentation
- Writing/updating READMEs, guides, and runbooks
- Maintaining a docs index so nothing gets lost
- Summarizing design decisions and architectural changes

## How I Work
1. Read the PR diff or agent output
2. Assess: does this change user-facing behavior? Add a new feature? Change configuration?
3. If yes: write or update the relevant documentation
4. If no: report "no docs needed" with brief justification

## Boundaries
**I handle:** Documentation, guides, READMEs, summaries, knowledge management
**I don't handle:** Code implementation, code review, research, operations
```

Create `.squad/agents/{name}/history.md` seeded with project context.

### Step 2: Add to team.md roster

```markdown
| 📝 {Name} | Documenter | `.squad/agents/{name}/charter.md` | ✅ Active |
```

### Step 3: Add routing table entry

In `routing.md` → routing table:

```markdown
| Documentation, reports, summaries | 📝 {Name} | `docs/` | "Write docs for X", "Summarize this", guides, READMEs |
```

### Step 4: Add follow-up trigger rule

In `routing.md` → `## Rules` section, add a numbered rule:

```markdown
N. **Documentation follow-up** — after any PR is merged that adds or modifies
   user-facing features, scripts, tools, or configuration, the coordinator
   spawns {Name} (background) to evaluate whether documentation is needed.
   {Name} reads the merged PR diff and either writes/updates docs or reports
   "no docs needed." This is a follow-up, not a gate — it does not block
   the merge.
```

**Why a rule and not a ceremony:** Ceremonies are structured multi-participant meetings. This is a single-agent follow-up task. A routing rule is simpler and more appropriate.

**Why background, not sync:** Documentation doesn't block other work. The documenter runs in parallel with whatever comes next.

### Step 5: Wire into the coordinator's post-merge flow

This is the trickiest part. The coordinator's After Agent Work flow doesn't currently have a "post-merge" hook. You wire this through the issue-lifecycle template.

In `.squad/templates/issue-lifecycle.md`, after the merge step, add:

```markdown
8. **Documentation follow-up.** After merge, check routing.md Rules for
   documentation follow-up rule. If present, spawn the documenter (background)
   with the merged PR diff to evaluate whether docs are needed.
```

Alternatively, you can wire this as an `after` ceremony in `ceremonies.md`:

```yaml
- name: "Documentation Check"
  when: "after"
  condition: "PR merged that adds features, scripts, tools, or config changes"
  facilitator: "{DocumenterName}"
  participants: ["{DocumenterName}"]
  output: "Docs written/updated, or 'no docs needed' with justification"
```

### Step 6: Worktree for doc changes

If the documenter produces files, they need a worktree — docs are files too. The coordinator should:
1. Create a worktree for the doc update (e.g., `squad/{issue}-docs`)
2. The documenter commits and pushes
3. A PR is created for the docs
4. The docs PR goes through the normal review flow (including the code reviewer if you have one)

This means doc changes also get reviewed. The documenter is not exempt from the review gate.

### Step 7: Add to casting registry

Update `.squad/casting/registry.json` with the new entry.

### Step 8: Verify

- [ ] After a feature PR merges, does the coordinator spawn the documenter? → Check the routing rule exists.
- [ ] Does the documenter get a worktree for their work? → Check the worktree rule covers docs.
- [ ] Do doc changes go through the review gate? → They should — docs are files, files need PRs, PRs need review.
- [ ] Is the follow-up non-blocking? → The documenter should be background, not sync.

## What Each File Controls (Summary)

| File | What it contributes |
|------|-------------------|
| `charter.md` | WHO the documenter is and HOW they evaluate |
| `team.md` | That the documenter EXISTS |
| `routing.md` routing table | That explicit doc requests go to this member |
| `routing.md` Rules section | That the coordinator MUST spawn docs evaluation after merges (enforcement) |
| `issue-lifecycle.md` or `ceremonies.md` | The procedural hook: when exactly the follow-up fires |
| `casting/registry.json` | Persistent name tracking |

**The most commonly missed piece:** The Rules section entry (Step 4). Without it, the documenter only runs when someone explicitly says "write docs for X." The whole point is that it runs automatically.
