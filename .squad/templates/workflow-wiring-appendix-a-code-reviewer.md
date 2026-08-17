# Appendix A: Wiring a Code Reviewer — Complete Walkthrough

> End-to-end example of adding a code reviewer to your squad and wiring their gate so it actually gets enforced. This walkthrough addresses a common failure: a reviewer is on the roster but never reviews a single PR because the gate wasn't wired.

## The Problem This Solves

Adding a reviewer to `team.md` gives them an identity. It does NOT:
- Tell the coordinator to route PRs to them
- Prevent PRs from being merged without their approval
- Prevent issues from being closed before review happens

**What goes wrong without enforcement:** A reviewer can be on the roster as "Reviewer" from day one. Their charter says they review PRs. The routing table says "PR code review → {ReviewerName}." But PRs get merged and issues get closed without them ever being spawned. Why?

Because the routing table says WHO handles what — it's for incoming requests ("review PR #42"). It does NOT say "after every agent completes work, route their output to {ReviewerName}." The coordinator routes work TO agents, but nothing tells it to route COMPLETED work to a reviewer. The "After Agent Work" flow in `squad.agent.md` says: collect results → present → spawn Scribe. No review step.

**The fix has three layers:**

| Layer | What it does | Where it lives |
|-------|-------------|----------------|
| Identity | Reviewer exists and knows how to review | `team.md` roster + `charter.md` |
| Routing | User can explicitly request "review this" | `routing.md` routing table |
| **Enforcement** | Coordinator MUST route every PR to reviewer before merge | `routing.md` Rules section + `issue-lifecycle.md` post-work steps |

Most squads get layers 1 and 2 right. Layer 3 — enforcement — is what's usually missing.

## Step-by-Step Walkthrough

### Step 1: Create the reviewer's identity

Create `.squad/agents/{name}/charter.md`:

```markdown
# {Name} — Code Reviewer

## Identity
- **Name:** {Name}
- **Role:** Code Reviewer
- **Expertise:** Code quality, correctness, test coverage, security, patterns
- **Style:** Thorough, fair, specific. Provides actionable feedback.

## What I Own
- Reviewing PRs for code quality, correctness, and test coverage
- Identifying bugs, security issues, and design problems
- Providing specific, actionable feedback (not vague suggestions)

## How I Review
1. Read the PR diff completely
2. Check: does it do what the issue asked for?
3. Check: are there tests? Do they cover the important cases?
4. Check: are there bugs, edge cases, or security issues?
5. Check: does it follow project patterns and conventions?
6. Verdict: APPROVE or REJECT with specific feedback

## Boundaries
**I handle:** Code review, PR review, quality gates
**I don't handle:** Implementation, design, research, documentation

## On REJECT
I provide specific feedback: what's wrong, why, and what to do instead.
The original author fixes their work. I re-review after fixes.
```

Create `.squad/agents/{name}/history.md` seeded with project context.

### Step 2: Add to team.md roster

```markdown
| 👑 {Name} | Code Reviewer | `.squad/agents/{name}/charter.md` | ✅ Active |
```

### Step 3: Add routing table entry

In `routing.md` → routing table:

```markdown
| PR code review | 👑 {Name} | — | "Review PR #42", code quality, finding reports |
```

**⚠️ This is necessary but NOT sufficient.** This only handles explicit review requests. It does NOT enforce automatic review of every PR.

### Step 4: Add enforcement rule (THIS IS THE CRITICAL STEP)

In `routing.md` → `## Rules` section, add a numbered rule:

```markdown
N. **{Name} PR Gate** — every PR created by any agent MUST be reviewed by {Name}
   before merge. The coordinator spawns {Name} (sync) with the PR diff after
   the author pushes and creates the PR. On REJECT, the original author addresses
   feedback. On APPROVE, the coordinator merges via `gh pr merge`. No PR merges
   without {Name}'s approval.
```

**Why this works when the routing table alone didn't:** The routing table is for matching incoming work to agents. Rules are behavioral constraints the coordinator must follow AFTER work completes. The rule says "after a PR exists, you MUST do X before proceeding." The routing table says "if someone asks for a review, route to X."

### Step 5: Wire into issue-lifecycle.md

In `.squad/templates/issue-lifecycle.md`, the "Coordinator Post-Work Steps" section should reference your reviewer by name:

```markdown
4. **Route to reviewer.** Spawn {Name} (sync) with the PR diff for code review.
```

This is the operational detail — the step-by-step instructions the coordinator follows after an agent completes issue work. The routing rule (Step 4) is the mandate; the lifecycle template is the procedure.

### Step 6: Add to casting registry

Update `.squad/casting/registry.json` with the new entry.

### Step 7: Verify

Ask yourself these questions:

- [ ] If a clean session coordinator reads `routing.md` Rules, will it know to route PRs to this reviewer? → Check rule N exists.
- [ ] If an agent completes work and pushes a PR, does the coordinator's post-work flow include a review step? → Check `issue-lifecycle.md` step 4.
- [ ] Can the coordinator merge a PR without the reviewer's approval? → The rule should say "No PR merges without {Name}'s approval."
- [ ] Can the coordinator close an issue without a merged PR? → Check the issue closure rule exists.

If any answer is wrong, you have a gap.

## What Each File Controls (Summary)

| File | What it contributes to the reviewer gate |
|------|----------------------------------------|
| `charter.md` | WHO the reviewer is and HOW they review |
| `team.md` | That the reviewer EXISTS on the team |
| `routing.md` routing table | That explicit review requests go to this reviewer |
| `routing.md` Rules section | That the coordinator MUST route EVERY PR to this reviewer (enforcement) |
| `issue-lifecycle.md` | The step-by-step procedure for the post-work review flow |
| `casting/registry.json` | Persistent name tracking |

**Remove any one of these and the gate has a hole.** The most commonly missed piece is the Rules section entry (Step 4).
