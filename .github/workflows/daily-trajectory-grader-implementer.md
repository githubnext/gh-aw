---
private: true
emoji: "🧮"
name: Daily Trajectory Grader Implementer
description: >
  Implements exactly one grader per run from the ranked graders
  catalog (shared/graders/README.md) as a new, self-contained
  shared agentic workflow component, and opens a draft PR with the addition.
on:
  schedule: daily
  workflow_dispatch:
  skip-if-match: 'is:open is:pr in:title "[trajectory-grader]"'

tracker-id: daily-trajectory-grader-implementer

permissions:
  contents: read
  actions: read
  issues: read
  pull-requests: read

engine:
  id: copilot
strict: true
timeout-minutes: 25

tools:
  bash:
    - "find .github/workflows/shared/graders -maxdepth 1 -type f -name \"*.md\" | sort"
    - "cat .github/workflows/shared/graders/*.md"
    - "cat .github/workflows/shared/aw-logs-24h-fetch-prompt.md"
    - "find .github/workflows/shared -maxdepth 1 -type f -name \"*.md\" | sort"
  edit:

safe-outputs:
  create-pull-request:
    steer: true
    draft: true
    expires: 14d
    title-prefix: "[trajectory-grader] "
    labels: [automation, observability, graders, trajectory-graders]

    allowed-files:
      - ".github/workflows/shared/graders/*.md"
  noop:

features:
  gh-aw-detection: true
---

# Daily Trajectory Grader Implementer

You are a deterministic-grader engineer. Your mission is to ship exactly one
new grader per run from the ranked catalog in
`.github/workflows/shared/graders/README.md`, as a new shared
agentic workflow component file, without touching any other part of the
repository.

## Step 1 — Read the catalog and the IR contract

1. Read `.github/workflows/shared/graders/README.md` in full.
2. Read `.github/workflows/shared/graders/trajectory-ir.md` in
   full — every grader you write MUST be expressed as a projection over
   this canonical Trajectory IR, not as a bespoke ad-hoc parser.
3. List the existing grader files with
   `find .github/workflows/shared/graders -maxdepth 1 -type f -name "*.md" | sort`.
   `README.md` and `trajectory-ir.md` are not graders.

## Step 2 — Select the next grader

Walk the catalog **tier-first, then rank-within-tier**: all of Tier 1
top-to-bottom, then all of Tier 2, then all of Tier 3, exactly as the
three tables are ordered in the README (ranks are not contiguous within a
tier — do not simply sort by rank number 1 through 25). Select the first
grader ID that:

- does **not** already have a `shared/graders/<id>.md` file, and
- is marked `Not started` in the catalog table.

If every grader already has a file and the table shows all 25 as
`Implemented`, call `noop` with reason "all 25 catalog graders implemented"
and stop. Do not re-implement or "improve" an existing grader file in this
workflow — one net-new grader per run only.

## Step 3 — Implement the grader as a shared component

Create exactly one new file:
`.github/workflows/shared/graders/<selected-id>.md`

This is a plain markdown shared component (no frontmatter — follow the
style of `shared/aw-logs-24h-fetch-prompt.md`: a short `##` heading, prose,
and inline code where useful). It must be self-contained and importable by
any workflow via:

```yaml
imports:
  - shared/graders/trajectory-ir.md
  - shared/graders/<selected-id>.md
```

The file must include, in this order:

1. **`## <Grader Title>`** heading naming the grader ID.
2. **What it measures** — one paragraph, matching the one-line summary
   already given for this grader in the README's "What each grader
   answers" section; expand it with the concrete failure mode it catches
   and why it is distinct from the existing built-in graders
   (`tool-success-rate`, `retries`, `loops`, `trajectory-efficiency`,
   `execution-step-count`, `execution-duration`,
   `working-set-rebuild-factor`, `context-growth`, `artifact-production`).
   Never re-derive step count, retries, generic loop counts, duration,
   tool-success rate, or generic trajectory efficiency — those already
   exist.
3. **Required IR fields** — which fields of the Trajectory IR
   (`events`, `states`, `actions`, `toolCalls`, `observations`,
   `resources`, `provenanceEdges`, `objectives`, `reference`) this grader
   reads, and what to do (`applicable: false` with a reason) when a
   required field is empty or absent for this trace.
4. **Computation** — an exact, deterministic algorithm, written either as
   numbered prose steps or a short fenced code block (JavaScript or Python,
   illustrative — it is executed by the LLM reasoning over the IR data, not
   run as a script by the harness). Prefer a closed-form formula when the
   grader has one (e.g. state-revisit-probability-rep is literally
   `(visited - distinct) / visited`). For recurrence-family graders
   (`recurrence-*`), define the recurrence matrix explicitly: two IR
   events `i` and `j` are "recurrent" iff their canonical `states[]` id is
   equal (or, if using tool calls only, iff `toolCalls[i].name` and a
   normalized argument signature match `toolCalls[j]`); derive
   determinism/laminarity/trapping-time/recurrence-rate from that matrix
   using the standard RQA definitions (diagonal-line density, vertical-line
   density, mean vertical-line length, and overall recurrence density,
   respectively).
5. **Output** — restate the shared output contract from
   `trajectory-ir.md` (`id`, `value`, `unit`, `direction`, `evidence`,
   `applicable`, `notApplicableReason`) with this grader's concrete
   `id`, `unit`, and `direction` filled in.
6. **Worked micro-example** — a tiny (3-6 event) illustrative IR excerpt
   and the resulting computed value, so an importing workflow can
   self-check its implementation.

Keep the file focused and grounded: no invented statistics, no references
to tools or data this grader does not need, no network calls, no `require`,
`import`, `fetch`, or `eval` (this is a prompt fragment, not executable
code, but keep it consistent with the constraints custom inline graders
must satisfy per the
[Graders Specification](https://githubnext.github.io/gh-aw/specs/graders-specification/)).

## Step 4 — Update the catalog

Edit `.github/workflows/shared/graders/README.md`:

- Flip the selected grader's **Status** cell from `Not started` to
  `Implemented`.
- Do not change any other row, ranking, or wording.

## Step 5 — Output contract

Emit exactly one `create_pull_request` safe output touching only:

- `.github/workflows/shared/graders/<selected-id>.md` (new file)
- `.github/workflows/shared/graders/README.md` (status flip)

Title: `[trajectory-grader] Implement <selected-id>`

Body must include:

- Which grader was implemented and its rank/tier.
- Why it is distinct from existing built-in graders.
- The required IR fields it depends on.
- A link back to `shared/graders/README.md` for the full catalog and remaining count (e.g. "6 of 25 implemented").

Do not modify any file outside the two listed above. Do not edit any
`.lock.yml` file. If you cannot produce a complete, self-contained grader
file meeting all six required sections above, call `noop` with a clear
reason instead of emitting a partial `create_pull_request`.
