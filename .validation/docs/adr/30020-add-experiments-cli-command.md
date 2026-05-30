# ADR-30020: Add `experiments` CLI Command as the Observability Interface for Git-Branch-Backed A/B Experiments

**Date**: 2026-05-03
**Status**: Draft
**Deciders**: pelikhan (PR author)

---

## Part 1 — Narrative (Human-Friendly)

### Context

ADR-29996 established `experiments/*` git branches as the durable storage backend for A/B experiment run state (`state.json`), and ADR-29985 extended that schema with per-run records (`runs` array). After these decisions, the only supported way to inspect accumulated experiment data was via raw `git show origin/experiments/<name>:state.json` or a direct GitHub API call — there was no user-facing CLI tool for this. As the number of experiments tracked by gh-aw grows, operators need a convenient way to see which experiments are running, how many runs each variant has collected, and what assignments recent runs received, without manually parsing JSON from git refs.

### Decision

We will add a hidden `gh aw experiments` command to the analysis group, with `list` and `analyze` subcommands that read `state.json` directly from `experiments/*` git branches. The command exposes two read modes: a local mode that calls `git show <ref>:state.json`, and a remote mode (`--repo`) that fetches content via the GitHub Contents API. Placing the command in the analysis group and marking it hidden keeps it consistent with the existing command structure while deliberately avoiding surfacing it in the default help output until it reaches a more polished state.

### Alternatives Considered

#### Alternative 1: Extend the Existing `audit` or `status` Command

The `audit` command already reads workflow run history and surfaces experiment variant assignments via ADR-29628's filter flags. Rather than introducing a new top-level command, experiment state could be surfaced as additional output sections or flags within `audit`. This was not chosen because `audit` queries GitHub Actions run history through the API; the new command reads cumulative state directly from the git branch, which is a fundamentally different data source. Mixing these would blur the command's responsibility and complicate its interface.

#### Alternative 2: No Dedicated Command — Use Raw `gh api` or `git show` Directly

Users could inspect experiment state by running `git show origin/experiments/my-workflow:state.json | jq .` or `gh api repos/{owner}/{repo}/contents/state.json?ref=experiments/my-workflow`. This requires no new Go code but demands knowledge of the branch naming convention, base64 decoding (GitHub API returns encoded content), and JSON parsing. As the feature matures and more workflows adopt experiments, this manual approach does not scale and puts an unreasonable burden on operators who just want a quick summary.

#### Alternative 3: Expose Experiment State Through the `gh aw status` Command

Routing experiment summary data through the existing `status` command would avoid a new entry point. This was not chosen because `status` aggregates workflow-level health (open issues, recent failures) and has no natural slot for per-experiment variant breakdowns without significantly expanding its scope.

### Consequences

#### Positive
- Operators can list all active experiments and inspect variant distribution without knowing the underlying storage format or branch naming convention.
- JSON output (`--json`) and remote target (`--repo`) flags follow the same conventions as every other gh-aw command, making the command scriptable and CI-friendly from day one.
- The `analyze` subcommand surfaces the last 10 run records from the `runs` array (ADR-29985), giving operators a concrete audit trail of recent variant assignments.

#### Negative
- The command is marked `Hidden: true`, so it will not appear in `gh aw --help`. Discoverability is effectively zero for users who are not already aware of it, which is intentional but means documentation or announcement is needed to drive adoption.
- The command only reads from `repo` storage mode; experiments using legacy `cache` storage (ADR-29996) are invisible to it, with no warning or indication that cache-mode experiments exist.
- Local mode depends on `git for-each-ref` and `git show`, which require the user to have the repository cloned and the remote fetched; it will return empty results in shallow clones that lack the `experiments/*` remote refs.

#### Neutral
- The `state.json` schema consumed by the command is defined by `pick_experiment.cjs` (runtime) and documented in ADR-29985; the command is a pure reader and does not modify state.
- The command's data types (`ExperimentState`, `ExperimentRunRecord`, `ExperimentVariantStats`) mirror the `state.json` schema directly; any future schema extension in ADR-29985 will require a corresponding update to these structs.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Command Structure

1. The `experiments` command **MUST** be registered in the `analysis` command group alongside `status`, `list`, `health`, `audit`, `logs`, and `checks`.
2. The `experiments` command **MUST** have `Hidden: true` set on the Cobra command until explicitly promoted to visible.
3. The `experiments` command **MUST** provide a `list` subcommand and an `analyze` subcommand.
4. Invoking `gh aw experiments` with no subcommand **MUST** behave identically to `gh aw experiments list`.
5. Both subcommands **MUST** support a `--json` / `-j` flag for machine-readable output and a `--repo` / `-r` flag to target a specific repository.

### Data Source and Branch Convention

1. Implementations **MUST** read experiment state exclusively from git branches matching the `experiments/*` prefix, as defined by ADR-29996.
2. In local mode (no `--repo` flag), implementations **MUST** resolve the experiment branch ref as `origin/experiments/<workflowID>`, falling back to the local branch `experiments/<workflowID>` if the remote ref does not exist.
3. In remote mode (`--repo` set), implementations **MUST** fetch `state.json` content via the GitHub Contents API (`repos/{owner}/{repo}/contents/state.json?ref=experiments/<workflowID>`).
4. Implementations **MUST** base64-decode the content returned by the GitHub API before parsing, stripping all whitespace from the encoded string prior to decoding.
5. When `state.json` is absent or unparseable, implementations **MUST NOT** return an error to the caller; they **MUST** treat the branch as having an empty experiment state (zero counts, no runs).

### `list` Subcommand

1. The `list` subcommand **MUST** enumerate all branches matching `experiments/*` and return one summary row per unique workflow ID.
2. Each summary row **MUST** include: workflow ID, full branch name, number of distinct experiments (keys in `state.counts`), total runs, and the date of the most recent run (`YYYY-MM-DD`).
3. Total runs **MUST** be derived from `len(state.runs)` when the `runs` array is non-empty; implementations **MUST** fall back to summing all variant counts from `state.counts` when `runs` is absent or empty.
4. In local mode, duplicate workflow IDs from both remote and local refs **MUST** be deduplicated; the remote ref (`origin/`) **SHOULD** be preferred.

### `analyze` Subcommand

1. The `analyze` subcommand **MUST** accept exactly one positional argument: the workflow ID (branch name without the `experiments/` prefix).
2. The `analyze` subcommand **MUST** display per-experiment variant counts with percentages and total runs for the specified workflow.
3. The `analyze` subcommand **MUST** display the most recent run records from `state.runs`, capped at 10 entries.
4. Implementations **MUST** return a user-visible error when the specified experiment branch does not exist; they **MUST NOT** silently return empty output.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Specifically: (a) the command is registered in the analysis group with `Hidden: true`, (b) local and remote read modes are both supported, (c) absent or invalid `state.json` is treated as empty state (not an error) for the `list` subcommand, (d) a missing branch is a user-visible error for the `analyze` subcommand, and (e) total-runs falls back to count-summation when `runs` is absent. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25293168791) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
