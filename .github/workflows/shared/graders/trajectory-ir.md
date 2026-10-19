## Canonical Trajectory IR

Every trajectory grader (built-in in
`actions/setup/js/trajectory_graders.cjs`) is a projection over one shared,
canonical intermediate representation (IR) of a completed agent run, built
**once** per grader run rather than re-parsed per grader. Do not write a
bespoke parser per grader — build (or reuse) this IR first, then compute the
grader's formula against it.

### Why an IR

`gh-aw` already ships deterministic built-in graders (`tool-success-rate`,
`retries`, `loops`, `trajectory-efficiency`, `execution-step-count`,
`execution-duration`, `working-set-rebuild-factor`, `context-growth`,
`artifact-production` — see
[Graders reference](https://githubnext.github.io/gh-aw/experimental/trace-graders/)).
Those cover step count, retries, generic loop counts, duration, tool-success
rate, and trajectory efficiency. The graders in this directory intentionally
do **not** duplicate those signals; they exist to answer different questions
(near-miss safety, exploration vs. exploitation, recurrence structure,
provenance, termination timing, planning conformance, reference-trajectory
fidelity). Build the IR once and project every new grader from it so adding a
26th grader never means writing a 26th trace parser.

### Source data

Build the IR only from the in-memory unified session event array supplied
in conclusion. Use `createSessionLookups(session)` from
`actions/setup/js/grader_session.cjs` to share event indexes, correlated tool
calls, statistics, and the trajectory projection. Graders must not read
files, inspect environment variables, execute scripts, or make network calls.

The collector supplies tool, usage, `workflow.info`, `safe_output.result`,
and optional `trajectory.ir` events. Explicit objectives, state transitions,
references, and provenance must already be recorded in those events; never
invent missing evidence or load it from workflow files.

### IR schema

```jsonc
{
  "runId": "string",
  "events": [
    // one entry per atomic occurrence, in execution order
    { "index": 0, "kind": "tool_call|observation|state_change|message|safe_output", "timestamp": "ISO8601", "ref": "id of actions[]/observations[]/... entry" }
  ],
  "states": [
    // canonical, coarse-grained snapshots of "where the agent is"
    // canonicalize so that semantically identical states collapse to the same id
    // (e.g. same file + same tool + same normalized argument shape)
    { "id": "string", "label": "string", "firstEventIndex": 0 }
  ],
  "actions": [
    { "id": "string", "eventIndex": 0, "type": "string", "target": "string|null", "validAtIssueTime": true, "consequential": true }
  ],
  "toolCalls": [
    { "id": "string", "eventIndex": 0, "name": "string", "arguments": {}, "success": true, "durationMs": 0, "outputRef": "observations[].id|null" }
  ],
  "observations": [
    { "id": "string", "eventIndex": 0, "sourceToolCallId": "string|null", "consumedByActionIds": ["string"] }
  ],
  "resources": [
    { "id": "string", "kind": "file|url|issue|pr|artifact", "uri": "string" }
  ],
  "provenanceEdges": [
    // directed edges: evidence/tool/environment -> derived action/output
    { "from": "observations[].id|toolCalls[].id", "to": "actions[].id|safe_output id", "relation": "informed|required-for|derived-from" }
  ],
  "objectives": [
    // declared or inferred completion/evidence conditions
    { "id": "string", "description": "string", "satisfiedAtEventIndex": null, "dependsOn": ["objective-id"] }
  ],
  "reference": null
  // optional: reference trajectory / patch for benchmark-mode graders (#20-#24).
  // Populate only when the workflow importing the grader supplies one; leave
  // null otherwise and let the grader report "not-applicable". Shape:
  // {
  //   "toolCalls": [{ "name": "string", "arguments": {} }], // tool-wise-score
  //   "states": ["states[].id"],                            // trajectory-ndtw
  //   "patch": { "files": [{ "path": "string" }] }          // code-search-recall / code-read-precision / code-edit-precision
  // }
}
```

### Build procedure (once per run)

1. Validate and index the supplied unified session once.
2. Reuse the latest agent-phase `trajectory.ir` event when present.
   Otherwise, project the recorded agent events and correlated tool calls.
3. Preserve explicit states, action effects, observation dependencies,
   objectives, and reference evidence. Do not infer dependencies from event
   order, consequential actions from name substrings, or objectives from
   unavailable repository content.
4. Reuse this in-memory projection for every configured built-in trajectory
   grader. Do not persist or reload an intermediate IR file.

Trajectory analyses are opt-in. A grader whose required IR fields are
missing reports `unavailable` rather than fabricating a score.

For `code-read-precision`, record file content reads as actions with `type: "read"`
and file inspections as `type: "inspect"`, `target` set to the file path and
`consequential: false`. Searches and listings use distinct action types and do not
count as content reads. The grader compares distinct target paths with the
reference patch files; the IR has no symbol records, so precision is file-level.

For `code-edit-precision`, record file edits as actions with `type: "edit"`,
`target` set to the edited file path, and `consequential: true`. The grader
compares distinct edited paths with the reference patch files; the IR has no
line or hunk records, so precision is file-level.

### Output contract every grader in this directory follows

Each built-in grader returns either a finite number or an object with a finite
numeric `value`. Return `value: null` when required trace data is unavailable
or the grader is not applicable:

```jsonc
{
  "value": 0.0,
  "details": "short grounded summary citing IR event indices or ids"
}
```

The grader ID, name, unit, and `higher_is_better` or `lower_is_better`
direction come from built-in metadata and data-only configuration overrides.
An object result may additionally include `passed`, `severity`, `details`,
`diagnostics`, or `message`. Use
`{ value: null, passed: null, message: "not applicable: ..." }`
rather than fabricating a score when the trace does not contain the inputs the
grader needs (e.g. no reference trajectory for benchmark-mode graders or no
declared objectives for `objective-coverage`).
