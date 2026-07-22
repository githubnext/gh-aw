# Git AI Integration Opportunities

**Date:** 2026-07-22  
**Git AI version reviewed:** [v1.6.15](https://github.com/git-ai-project/git-ai/releases/tag/v1.6.15)  
**Status:** Exploratory analysis

## Executive Summary

[Git AI](https://github.com/git-ai-project/git-ai) is an Apache-2.0 Git
extension that records line-level AI authorship in `refs/notes/ai`. Its agent
coverage overlaps strongly with the engines supported by GitHub Agentic
Workflows (gh-aw), and its open note format could add code provenance,
acceptance-rate, and model-level reporting to agent-created pull requests.

The strongest opportunity is an opt-in integration that preserves Git AI
metadata through gh-aw's existing bundle-based safe-output path. A simple
installation step is not sufficient: Git AI records checkpoints in the agent
job, while gh-aw transports commits to a separate `safe_outputs` job and can
re-create commits when signing them. Git notes must therefore be transported
and retargeted alongside the commits they describe.

Recommended next step: build a narrow proof of concept for one engine and one
`create-pull-request` workflow. Keep telemetry disabled, use a pinned and
verified binary, transport only the open-standard authorship note, and verify
the final remote commit with `git ai blame` and `git ai stats --json`.

## What Git AI Provides

Git AI installs native hooks into supported coding agents. Those hooks create
checkpoints as tools edit files. After a commit, its background process turns
the checkpoints into an authorship log attached as a Git note.

The [Git AI Standard v3.0.0](https://github.com/git-ai-project/git-ai/blob/v1.6.15/specs/git_ai_standard_v3.0.0.md)
defines the portable data model:

- Notes live under `refs/notes/ai`, separate from the default Git notes ref.
- Each note maps files and line ranges to an AI session and trace.
- JSON metadata identifies the schema, base commit, agent, and model.
- Prompts are stored locally rather than copied into the repository in the
  default open source configuration.

The CLI exposes useful integration surfaces:

| Command | Potential gh-aw use |
|---|---|
| `git ai install-hooks` | Configure supported engines before execution |
| `git ai checkpoint` | Record an explicit agent edit checkpoint |
| `git ai stats --json` | Produce structured attribution and acceptance metrics |
| `git ai blame` | Explain the provenance of individual lines |
| `git ai fetch-notes` | Retrieve attribution from a remote |
| `git ai ci github run` | Preserve notes across GitHub squash and rebase merges |
| `git ai await` | Drain background processing before artifacts are uploaded |

Git AI v1.6.15 has presets for Claude, Codex, Gemini, GitHub Copilot,
OpenCode, and Pi. These align with gh-aw engines, but compatibility must be
tested against the exact CLI versions and filesystem layout used by each
engine.

## Fit with the gh-aw Architecture

### Natural integration points

gh-aw already has the lifecycle hooks needed for an experiment:

1. `pre-agent-steps` can install and configure Git AI immediately before the
   engine starts.
2. The engine can use its native Git AI hooks while editing and committing.
3. `post-steps` can wait for the daemon, validate the note, and emit metrics.
4. The unified agent artifact can carry attribution data to downstream jobs.
5. The `safe_outputs` job can attach and push the note after it determines the
   final commit SHA.

The bundle transport is a better starting point than the legacy patch
transport. gh-aw already packages the agent's commits into a Git bundle,
preserving topology, authorship, and messages. Git AI also provides CI logic
for preserving notes after GitHub performs squash or rebase merges.

### Critical gap: attribution crosses a job boundary

Code-writing safe outputs do not push directly from the agent job. As
documented in the
[pull-request safe-output flow](../docs/src/content/docs/reference/safe-outputs-pull-requests.md#how-it-works),
the agent's commits are uploaded as an artifact and a separate,
permission-controlled job applies and pushes them.

This creates four issues:

1. `refs/notes/ai` is not automatically included when a bundle contains only
   the agent branch.
2. Git AI's local SQLite data and working logs are ephemeral unless explicitly
   transported.
3. gh-aw's signed-commit push can produce a different final commit SHA, which
   strands a note attached to the original SHA.
4. Pushing a branch does not necessarily push `refs/notes/ai`; the notes ref
   needs its own authenticated, conflict-aware update.

An integration must treat the final remote commit as the unit of identity. It
should not claim provenance success merely because a note existed in the
agent job.

### Sandbox and lifecycle constraints

Custom steps run outside gh-aw's firewall sandbox, while the engine runs
inside it. Installing Git AI on the runner does not guarantee that:

- the binary and control socket are available in the engine container;
- the engine reads the modified hook configuration;
- its background process survives for the whole engine invocation;
- multi-repository workflows resolve checkpoints to the correct checkout; or
- attribution is finalized before the unified artifact upload.

These are proof-of-concept questions, not assumptions to encode in the
compiler.

## Integration Options

### Option 1: Read existing notes

Add a shared reporting pattern that fetches `refs/notes/ai` and runs
`git ai stats --json`, or parses the open standard directly.

**Value**

- Adds attribution summaries to reports and audits.
- Works for repositories that already use Git AI.
- Does not modify engine execution.

**Limitations**

- Does not attribute code generated by the current workflow.
- Full-history and notes fetching conflict with the default shallow checkout.
- A native parser would create schema-version maintenance work.

**Assessment:** Low-risk and useful as a standalone example, but not a full
integration.

### Option 2: Attribute the agent run

Install Git AI before engine startup, disable outbound telemetry, install the
matching engine hooks, and wait for attribution finalization after execution.

**Value**

- Captures the model and session that produced each accepted line.
- Reuses Git AI's supported-agent adapters rather than adding engine-specific
  tracking to gh-aw.

**Limitations**

- Requires binary, hook configuration, daemon state, and checkout access to be
  shared correctly with the sandbox.
- Hooks can drift as engine CLIs evolve.
- This produces local attribution but does not solve safe-output transport.

**Assessment:** Necessary for end-to-end attribution, but should remain
experimental until the sandbox contract is proven.

### Option 3: Preserve attribution through safe outputs

Extend bundle-based `create-pull-request` and
`push-to-pull-request-branch` processing with an attribution sidecar:

1. Finalize and validate the agent-job authorship data.
2. Upload only the data needed to reconstruct the v3 authorship note.
3. In `safe_outputs`, map the original commit to the final signed commit.
4. Attach the note to the final SHA.
5. Push `refs/notes/ai` with explicit credentials and conflict handling.
6. Verify the note from the remote before reporting success.

**Value**

- Makes provenance survive gh-aw's security boundary.
- Keeps write credentials out of the agent job.
- Supports signed commits without silently losing attribution.

**Limitations**

- Adds a new authenticated write to safe outputs.
- Concurrent note updates require fetch/merge/retry behavior.
- Patch fallback, cross-repository targets, forks, multiple commits, and
  partial success all need defined semantics.

**Assessment:** Highest-value native integration and the main technical
investigation.

### Option 4: Preserve notes after GitHub merges

Offer guidance or a reusable workflow based on `git ai ci github run` for
`pull_request` `closed` and `synchronize` events.

**Value**

- Handles squash and rebase merges that replace commit SHAs.
- Completes the provenance lifecycle after a gh-aw-created PR is merged.

**Limitations**

- Requires `contents: write`.
- Is repository-wide infrastructure rather than a property of one agentic
  workflow.
- Installation and action dependencies must be pinned rather than copied from
  the upstream curl-based example.

**Assessment:** Complementary follow-up, not part of the first engine pilot.

## Security and Privacy

Git AI is a substantial executable in the trusted portion of the workflow. An
integration should meet these requirements:

- **Opt in only.** Do not install or run Git AI in existing workflows by
  default.
- **Pin and verify.** Download a versioned release asset, verify its SHA-256
  digest, and consider GitHub artifact attestation verification. Do not execute
  a mutable remote installer.
- **Disable telemetry.** The
  [Git AI privacy documentation](https://github.com/git-ai-project/git-ai/blob/v1.6.15/data-privacy.md)
  says error and exception telemetry is on by default in OSS mode. An
  integration should set `telemetry_oss` to `off` before startup.
- **Keep prompts out of artifacts.** Transport the minimum authorship data,
  not the local prompt database or complete agent sessions.
- **Document repository-visible metadata.** Git notes expose agent, model,
  accepted-rate, line attribution, and the steering Git identity to everyone
  with repository access.
- **Preserve the safe-output trust boundary.** The agent job must not receive
  credentials for pushing notes.
- **Fail closed on mismatch.** A malformed note, unsupported schema, missing
  commit mapping, or ambiguous target repository should omit the note and
  surface an explicit failure rather than attach incorrect provenance.
- **Avoid untrusted execution.** Do not install or run tooling from a PR head
  in privileged `pull_request_target` contexts.

The Git AI CLI is Apache-2.0 licensed. Its hosted dashboards and team features
are separate services and are not required for the proposed OSS integration.

## Recommended Proof of Concept

Limit the first experiment to:

- one Ubuntu runner architecture;
- one engine with native hook support;
- one repository checkout at the workspace root;
- `create-pull-request` with bundle transport;
- one agent-created commit;
- OSS mode with telemetry disabled; and
- no cloud login or prompt upload.

### Success criteria

1. A known AI edit receives a v3 authorship entry in the agent job.
2. Human-authored fixture lines remain unattributed or human-attributed.
3. The final pushed commit, including a signed replacement commit, has the
   correct `refs/notes/ai` note.
4. A fresh clone can fetch the note and reproduce `git ai blame` and
   `git ai stats --json` results.
5. No prompt database, secret, token, or control socket is uploaded.
6. Telemetry produces no network request.
7. The workflow remains correct when the safe-output job is retried.
8. A missing or failed Git AI installation leaves the code change safe and
   reports attribution as unavailable, never as successful.

### Test matrix before broader support

| Dimension | Cases |
|---|---|
| Engines | Copilot, Claude, Codex, Gemini, OpenCode, Pi |
| Transport | Bundle, patch fallback |
| Commit handling | Original commit, signed replacement, multiple commits |
| Repository | Same repo, cross repo, fork PR, multiple checkouts |
| Checkout | Shallow, full history, sparse |
| Outcome | No changes, partial hook capture, daemon timeout, note conflict |
| Merge | Merge commit, squash, rebase |

## Proposed Direction

1. Start with a repository-local experimental workflow rather than a new
   frontmatter field.
2. Validate engine hooks and sandbox visibility without pushing notes.
3. Design an explicit versioned attribution sidecar between `agent` and
   `safe_outputs`.
4. Add final-SHA retargeting and conflict-safe notes push to the safe-output
   runtime.
5. Surface structured attribution in step summaries and audit artifacts.
6. Only after successful multi-engine tests, consider an opt-in frontmatter
   surface such as `attribution: git-ai`.
7. Treat post-merge reconciliation as a separate reusable workflow.

This order prevents a convenient setup option from shipping before gh-aw can
preserve the attribution it claims to collect.

## Open Questions

- Can each engine load hooks installed immediately before startup, or do some
  cache configuration earlier?
- Which Git AI files are the minimum safe sidecar when the commit is created
  before artifact upload?
- Can Git AI retarget a validated authorship log to a signed replacement
  commit through a supported API, or is upstream work needed?
- How should concurrent updates to `refs/notes/ai` be merged and retried?
- Should attribution failure block a code push when the feature is explicitly
  required?
- How should excluded or protected files removed by safe outputs be removed
  from the attribution log?
- What retention and disclosure policy should apply to model and steering
  identity metadata?
- Can Git AI metrics complement gh-aw OTEL token metrics without duplicating
  or contradicting them?
- Should the integration depend on the CLI or only on the open note standard?

## References

### Git AI

- [Repository and capability overview](https://github.com/git-ai-project/git-ai/tree/v1.6.15)
- [v1.6.15 release](https://github.com/git-ai-project/git-ai/releases/tag/v1.6.15)
- [Git AI Standard v3.0.0](https://github.com/git-ai-project/git-ai/blob/v1.6.15/specs/git_ai_standard_v3.0.0.md)
- [Data privacy](https://github.com/git-ai-project/git-ai/blob/v1.6.15/data-privacy.md)
- [GitHub CI implementation](https://github.com/git-ai-project/git-ai/blob/v1.6.15/src/ci/github.rs)
- [CI command implementation](https://github.com/git-ai-project/git-ai/blob/v1.6.15/src/commands/ci_handlers.rs)
- [Agent hook installers](https://github.com/git-ai-project/git-ai/tree/v1.6.15/src/mdm/agents)
- [License](https://github.com/git-ai-project/git-ai/blob/v1.6.15/LICENSE)

### gh-aw

- [Custom step lifecycle](../docs/src/content/docs/reference/steps-jobs.md)
- [Repository checkout and credential model](../docs/src/content/docs/reference/checkout.md)
- [Pull-request safe-output transport](../docs/src/content/docs/reference/safe-outputs-pull-requests.md)
- [`generate_git_patch.cjs`](../actions/setup/js/generate_git_patch.cjs)
- [`create_pull_request.cjs`](../actions/setup/js/create_pull_request.cjs)
- [`push_signed_commits.cjs`](../actions/setup/js/push_signed_commits.cjs)

