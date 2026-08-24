## state-revisit-probability-rep

**What it measures** — `state-revisit-probability-rep` measures the fraction
of visited canonical states that are redundant revisits of any state already
seen earlier in the run: `(visited states - distinct states) / visited states`.
It catches wasted exploration where the agent keeps returning to known
behavioral states anywhere in the trajectory. This is distinct from built-in
`loops`, which only detects adjacent or contiguous repeated sequences, and
from `trajectory-efficiency`, which is a task-success-relative usefulness
ratio; REP is a pure structural redundancy signal over canonical state visits,
independent of adjacency, outcome, step count, retries, duration, tool success,
working-set rebuilds, context growth, or artifact production.

**Required IR fields** — Read `states[]` for canonical state IDs and `events[]`
to recover chronological visit order. Treat each state entry as one visit: if
entries are objects, use `state.id` as the canonical ID and sort by
`state.firstEventIndex` against `events[].index`; if entries are strings, use
the array order as the visit order. Report `applicable: false` when `states[]`
is absent, empty, or has fewer than two entries, or when object entries do not
contain usable canonical IDs.

**Computation**

1. Build `orderedStateIds`, the chronological list of canonical state IDs.
2. Let `visited = orderedStateIds.length`.
3. If `visited < 2`, emit `applicable: false` with
   `notApplicableReason: "fewer than two state visits"`.
4. Let `distinct = new Set(orderedStateIds).size`.
5. Compute `value = (visited - distinct) / visited`.
6. Evidence should include `visited`, `distinct`, and a short sample of the
   repeated state IDs with their event indices when available.

**Output** — Append one object to
`/tmp/gh-aw/agent/graders/custom_grader_results.json` using the shared output
contract:

```json
{
  "id": "state-revisit-probability-rep",
  "value": 0.0,
  "unit": "ratio",
  "direction": "lower-is-better",
  "evidence": ["visited=4 distinct=3 revisits=1; repeated states: inspect:README.md at event 3"],
  "applicable": true,
  "notApplicableReason": null
}
```

Use a numeric ratio in `[0, 1)`: `0` means no state was revisited; values closer
to `1` mean most visits were redundant revisits.

**Worked micro-example**

```json
{
  "events": [
    { "index": 0, "kind": "tool_call", "timestamp": "2026-01-01T00:00:00Z", "ref": "tc0" },
    { "index": 1, "kind": "tool_call", "timestamp": "2026-01-01T00:00:01Z", "ref": "tc1" },
    { "index": 2, "kind": "tool_call", "timestamp": "2026-01-01T00:00:02Z", "ref": "tc2" },
    { "index": 3, "kind": "tool_call", "timestamp": "2026-01-01T00:00:03Z", "ref": "tc3" }
  ],
  "states": [
    { "id": "inspect:README.md", "label": "Read README", "firstEventIndex": 0 },
    { "id": "inspect:pkg/workflow", "label": "Search workflow package", "firstEventIndex": 1 },
    { "id": "edit:grader-catalog", "label": "Edit grader catalog", "firstEventIndex": 2 },
    { "id": "inspect:README.md", "label": "Read README again", "firstEventIndex": 3 }
  ]
}
```

`visited = 4`, `distinct = 3`, so
`state-revisit-probability-rep = (4 - 3) / 4 = 0.25`.
