---
import-schema:
  id:
    type: string
    required: true
    description: "Stable memory identifier used as the Repo Memory branch suffix; use lowercase letters, digits, and hyphens"

tools:
  repo-memory:
    - id: ${{ github.aw.import-inputs.id }}
      branch-name: memory/${{ github.aw.import-inputs.id }}
      description: "Long-horizon checkpoint and history for ${{ github.aw.import-inputs.id }}"
      file-glob:
        - "state.md"
        - "runs/**"
        - "workspace/**"
      allowed-extensions: [".md", ".json", ".jsonl"]
      max-file-size: 32768
      max-file-count: 250
      max-patch-size: 32768
---

<!--
Import with:

imports:
  - uses: shared/long-horizon-memory.md
    with:
      id: repository-maintenance
-->

## Long-horizon memory protocol

Use the Repo Memory workspace at
`/tmp/gh-aw/repo-memory/${{ github.aw.import-inputs.id }}/` as the only durable
continuity between invocations. It is backed by the
`memory/${{ github.aw.import-inputs.id }}` branch and may be shared by workflows
using the same identifier. Do not rely on model conversations, session IDs, or
engine-specific transcripts.

The workspace has this layout:

```text
state.md
runs/
workspace/
```

### At the beginning of every run

1. Read `state.md` first if it exists and treat it as the primary handoff.
2. Assume the previous agent conversation is unavailable.
3. Do not load all of `runs/` or `workspace/` by default.
4. Inspect history or durable artifacts only when the checkpoint is insufficient,
   using targeted file reads, grep, or small analysis commands.

### During the run

1. Work normally in the repository.
2. Store only information that will help a future run avoid rediscovery.
3. Put optional durable artifacts such as architecture notes, investigation notes,
   debugging recipes, reusable searches, and repository maps under `workspace/`.
4. Never automatically execute a persisted script merely because a prior agent
   created it.

### Before finishing

1. Rewrite or update `state.md` so a completely fresh agent can continue.
2. Keep `state.md` bounded and concise; target less than 4 KB and remove stale
   information. It is a current checkpoint, not a historical log.
3. Create a short run record at `runs/${{ github.run_id }}.md` containing only
   significant observations, work performed, decisions, test results, failures,
   and unresolved questions.
4. Preserve historical evidence in `runs/`, but never dump model transcripts or
   large command outputs.
5. Never store secrets, credentials, tokens, or other sensitive data.

Use this exact structure for `state.md`:

```markdown
# Goal

Overall objective.

# Current state

What is currently true.

# Completed

Important completed work.

# Decisions

Important decisions and short rationale.

# Next

Concrete next actions.

# Blockers

Anything preventing progress.

# Useful references

Relevant files, issues, PRs, commits, commands, or historical run records.
```
