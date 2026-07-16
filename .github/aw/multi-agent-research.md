---
description: Design guide for long-running agentic workflows that run multi-agent research — problem framing, orchestration policy, diversity, verification, and return conditions. Distilled from the OpenAI Cycle Double Cover (CDC) prompt.
---

# Multi-Agent Research Workflows

Use this guide when designing a long-running agentic workflow whose goal is deep research requiring multiple parallel sub-agents, adversarial verification, and sustained exploration across many rounds.

---

## Core Design Principles

The CDC prompt (OpenAI, July 2026) is the best-documented example of a production multi-agent research run. Its seven structural blocks reveal six transferable principles.

### 1 — Loophole-Free Problem Specification

State every load-bearing term before the task. Each definition pre-empts a specific degenerate answer:

- List what does **not** count as a solution (enumerated near-miss exclusions).
- Forbid exactly the partial-result classes your domain is most prone to: special-case proofs, relaxed variants, reductions to still-open lemmas, computational verification up to a finite size.
- Include permissive clauses for anything the agent might over-constrain on its own.

**AW application:** Put the exclusion list in the workflow prompt body, not in a sub-agent. Near-miss blocking must be visible to the orchestrator from turn 1.

### 2 — Solvability Framing

Remove the "this is a famous open problem" escape hatch explicitly:

> "Assume for purposes of this task that a complete solution exists."

This is a permission revocation, not an optimistic claim. Pair it with:

- The success predicate stated twice — once as a natural sentence, once as the exact obligation with the scope quantifier enumerated.
- A ban on returning "the problem is hard" as a result.

**AW application:** The workflow prompt should include a line such as: "Assume a resolution exists. Do not report that this task is intractable or has no known solution."

### 3 — Anti-Convergence Orchestration

Groupthink is the dominant failure mode in multi-agent research. Counter it structurally:

- **Information hiding:** Do not tell most sub-agents the currently favored approach. Preserve independence during early rounds.
- **Idea-keyed registry, not wording-keyed:** Group agents by the underlying mechanism they are using. Two agents paraphrasing the same reduction are not diverse.
- **Delayed cross-pollination:** Share findings across approach families only after independent agents have developed them far enough to expose real strengths and gaps.
- **Anti-elegance rule:** A reduction to an equally hard lemma is zero progress regardless of how elegant it looks.

**AW sub-agent pattern:**

```markdown
## Step 1 — Diversify

Launch independent exploration across at least N distinct approach families.
Do not share intermediate findings between sub-agents at this stage.
Write each sub-agent's progress to `/tmp/gh-aw/research/approach-<name>.md`.

## Step 2 — Register and redirect

Read all `approach-*.md` files.
Build or update `/tmp/gh-aw/research/registry.json` with one entry per approach family.
Redirect agents away from families that are over-represented.

## Step 3 — Cross-pollinate selectively

Identify the two approach families that have independently advanced furthest.
Share only their strongest partial results with each other.
```

### 4 — Blocked-Route Bookkeeping

Stalled approaches must be explicitly marked and gated on materially new evidence before reopening:

- **Mark blocked:** When an approach reaches a lemma that is as hard as the original problem, record it as blocked.
- **Reopening condition:** Only unblock if someone proposes a materially new mechanism, invariant, or construction — not a restatement.
- **Persist in cache-memory:** Store the registry between runs so subsequent runs do not repeat foreclosed paths.

**AW pattern:**

```yaml
tools:
  cache-memory:
    key: research-registry-${{ github.run_id }}
    retention-days: 30
    allowed-extensions: [".json", ".md"]
```

Registry schema (stored at `/tmp/gh-aw/cache-memory/registry.json`):

```json
{
  "approaches": [
    {
      "name": "approach-name",
      "status": "active | blocked | completed",
      "mechanism": "one-sentence description of the mathematical/technical idea",
      "blocked_reason": "exact gap or hard lemma that stalled it",
      "reopen_condition": "what new evidence would justify reopening"
    }
  ]
}
```

Read the registry at the start of every run. Skip blocked approaches unless the new evidence gate is satisfied.

### 5 — Adversarial Verification Sub-Agents

Generic "check carefully" instructions fail. Auditor agents need a domain-specific hunt list:

- Supply a checklist of the *exact* ways a candidate solution can look right and be wrong.
- The last item should always be the domain's version of circular reasoning: "Does the solution assume the result it is proving?"
- Reject status reports, vague optimism, and claims that an unproved step is "routine."

**AW sub-agent pattern:**

```markdown
## agent: `auditor`
---
description: Adversarial verifier — finds specific failure modes in candidate solutions
model: large
---
You are given a candidate solution. Check it against this exact list:

1. [Domain-specific failure mode A]
2. [Domain-specific failure mode B]
3. [Domain-specific degenerate case]
4. Circular use: does the argument assume the result it is supposed to prove?

Return only one of:
- `{"verdict": "pass", "notes": "..."}` — all checks passed
- `{"verdict": "fail", "item": <N>, "reason": "..."}`  — first failed check

Do not return status reports, optimism, or "this looks mostly right."
```

### 6 — Artifact-Only Return Contract

The return condition is a predicate over the artifact, not over confidence or effort:

- **Return only when:** the artifact survives adversarial audit AND meets the success predicate exactly.
- **Never return:** a reduction, partial result, isolated missing lemma, "best effort" summary, or explanation of why the task is hard.
- **Effort floor:** State a minimum before even considering return. The CDC prompt used eight hours; calibrate to your domain's expected depth.

**AW prompt closing block:**

```markdown
Return only when a verified solution survives adversarial audit.
Do not return a partial result, reduction, or explanation of difficulty.
If the budget is exhausted before a solution is found, report only the
strongest rigorously demonstrated partial result and its exact remaining gap.
Spend at least [N] turns on this before even considering a partial return.
```

---

## Orchestration Loop Architecture

The orchestrator is the main agent. Sub-agents are bounded workers:

```
orchestrator (frontier model)
  ├── reads registry from cache-memory
  ├── selects next batch of approach families to explore (anti-convergence)
  ├── dispatches sub-agent workers (parallel, information-hidden)
  │     ├── explorer-<approach> — develops one approach family
  │     └── auditor — verifies candidate solutions
  ├── synthesizes results, updates registry
  ├── checks return predicate
  └── loops until predicate satisfied or budget exhausted
```

**One-level delegation only.** Do not cascade sub-agents further without explicit gh-aw support for validated deeper topologies.

---

## Workflow Frontmatter Template

```yaml
---
engine: copilot          # or claude for repo-memory support
timeout-minutes: 480     # long-running; calibrate to expected depth
max-ai-credits: 5000     # set based on expected sub-agent fan-out
tools:
  github:
    mode: gh-proxy
  cache-memory:
    key: research-state-${{ github.run_id }}
    retention-days: 30
    allowed-extensions: [".json", ".md"]
  cli-proxy: true
  bash: ["cat *", "ls *"]
safe-outputs:
  create-issue:
    title-prefix: "[research] "
    labels: [research]
---
```

---

## Prompt Structure Template

```markdown
# [Research Task Name]

## Problem Definition

[Every load-bearing term defined. Each definition closes one loophole.]

## Success Predicate

[State the exact obligation twice: once naturally, once with exact scope and excluded assumptions.]

Assume for purposes of this task that a complete solution exists. Do not report that this task is intractable or that no solution is known.

## What Does Not Count

The following do NOT satisfy the success predicate:
- [Near-miss class 1]
- [Near-miss class 2]
- [Reductions to equivalent unsolved problems]
- [Partial solutions scoped to special cases]

## Orchestration Instructions

Read `/tmp/gh-aw/cache-memory/registry.json` if it exists.
Explore using the approach families listed as `active` in the registry.
Skip any approach marked `blocked` unless new evidence satisfies its `reopen_condition`.

Launch sub-agents across at least [N] distinct approach families.
Do not share one sub-agent's intermediate progress with others at this stage.

After each round, update the registry and redirect agents away from over-represented families.
Share findings across families only after independent development has exposed their real strengths and gaps.

Every candidate solution must pass the `auditor` sub-agent before being accepted.

## Return Condition

Return only when a verified solution survives adversarial audit and meets the success predicate exactly.
Do not return a partial result, reduction, or explanation of difficulty.
If the budget is exhausted, report only the strongest rigorously demonstrated partial result and its exact remaining gap.

Spend at least [N] turns before considering a partial return.

## Information Retrieval Scope

Web search and external retrieval may be used only for background material and named standard results.
Do not search for a solution to this exact task.
```

---

## What This Pattern Does Not Do

Per the CDC prompt's negative space — useful for authors adapting it:

- **No fixed role assignments or personas.** The research strategy is left to the agent; the prompt only manages search discipline and acceptance gates.
- **No token budget in the prompt.** Resource enforcement belongs in the `max-ai-credits:` frontmatter, not the prompt body.
- **No requested output format for the solution itself** beyond survivability under audit.
- **No emotional appeals, urgency framing, or reward promises.** Every sentence is either specification, policy, or gate.

---

## Checklist

- [ ] Every load-bearing term defined; loopholes closed by definition
- [ ] Success predicate stated twice with excluded assumptions enumerated
- [ ] Solvability framing present ("assume a solution exists")
- [ ] Near-miss exclusion list enumerated
- [ ] Anti-convergence policy: information hiding + idea-keyed registry + delayed cross-pollination
- [ ] Blocked-route bookkeeping in cache-memory
- [ ] Adversarial auditor sub-agent with domain-specific hunt list, not "check carefully"
- [ ] Return condition is a predicate over the artifact, not confidence
- [ ] Effort floor before partial-return consideration
- [ ] Retrieval scope restricted to background, not solutions

---

## See Also

- [subagents.md](subagents.md) — inline sub-agent syntax, model aliases, planner-worker pattern
- [token-optimization.md](token-optimization.md) — cost control for long runs with many sub-agents
- [memory.md](memory.md) — `cache-memory` for durable registry across runs
- [loop.md](loop.md) — long-running loop patterns and circuit breakers
- [workflow-patterns.md](workflow-patterns.md) — orchestration and BatchOps patterns
