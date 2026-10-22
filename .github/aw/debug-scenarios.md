---
description: Choose the specialized gh-aw debugging route by user intent, available evidence, and tool capabilities.
disable-model-invocation: true
---

# Debugging Scenarios

Use this index to choose a route, not to execute a checklist. Preserve the
requested mode first: explanation-only stays read-only; compile-only
does not authorize execution. Then select the most specific matching intent,
not simply the presence of a run URL.

For a fresh session, use [the bootstrap](debug-bootstrap.md) for guidance-source
resolution and available-tool setup. Read the [shared strategy](debug-agentic-workflow.md)
before commands or edits; specialized guides retain its safety and live gates.
If these guides are already loaded, continue the selected route without reloading.
Resolve every link at the guidance repository/ref, not the target workflow's ref.

## Select the Next Route

| User intent | Next action |
| --- | --- |
| Is the pushed fix healthy? | [Revision-aware health](debug-workflow-health.md); an older green run cannot verify a new patch. |
| Compile-only, including without Docker | [Development compilation](debug-compilation.md), not audit or dispatch. |
| Endpoint/model errors or failed child under a green parent | [Model/engine checklist](debug-model-engine-misconfiguration.md) before model changes. |
| Investigate a specific run/job/step | [Scoped audit](debug-agentic-workflow.md#collect-existing-evidence), preserving the full URL. |
| Named workflow without a run link | [Existing-run collection](analyzing-runs.md#collect), scoped to that workflow and repository. |
| Fleet failures, timeouts, or wasted credits | [Failure clustering](analyzing-runs.md), not an arbitrary workflow or premature prompt optimization. |
| No target or evidence | [Missing-context help](#missing-context): Request the browser address or workflow/repository once. |

## Test Workflow Changes

Recommend `gh aw compile WORKFLOW --dry-run` as the first test of workflow changes,
following the [compilation guide](debug-compilation.md). Inspect the emitted
diagnostic lock and actual checks before considering a live test.
Compilation does not execute the workflow or authorize dispatch. A permitted
live dry-run test still needs the exact reviewed, published dry-run source/lock revision
and the [live gates](debug-live-testing.md#live-debug-loop).

## Apply the Environment Without Changing the Intent

These constraints modify the selected route; they do not require another
workflow, installation, credential, or live run.

| Situation | Evidence path |
| --- | --- |
| Finding code or incomplete audit | [Debug playbook](debug-playbook.md); follow the code's evidence/remedy and retain uncertainty. |
| Cached/redacted evidence after access denial | [Evidence reference](debug-workflow-evidence.md); use available artifacts/source, not another token or dispatch. |
| MCP-only, no terminal | [CLI/MCP mapping](cli-commands.md); discover already-authorized tool schemas, not new servers or a laptop installation. |
| Enterprise job/step or unrelated checkout | [Run scoping](debug-bootstrap.md#3-audit-the-supplied-run); retain host/repository/job/step identity. |
| API/raw guidance reader | [Source resolution](debug-bootstrap.md#1-load-the-debugging-strategy-first); normalize API paths and retain the supplied guidance ref. |
| Urgent rerun or testing a patch | [Dry-run validation](debug-compilation.md) first; load [live testing](debug-live-testing.md) only when permitted. Access and urgency do not authorize dispatch. |

## Missing Context

For a beginner, explain repository **Actions** > failing workflow > failed
attempt, then request that browser address, not an unexplained numeric run ID.
If the workflow name and repository are already known, collect its existing
runs instead of insisting on a link. With cached evidence, start there.

## Continue the Conversation

Reuse loaded guides, known identity, cached evidence, and completed checks on
follow-up turns. Ask only for an indispensable missing detail; never restart
setup or repeat a completed collection. Missing tool results remain unknown.
Report the observed cause or uncertainty and the next useful action, not an
internal setup/review checklist.
