---
title: Debugging Workflows
description: How to run, debug, and investigate agentic workflow failures using the Copilot CLI, gh aw audit, and log analysis.
sidebar:
  order: 250
---

This guide shows you how to debug agentic workflow failures on **github.com** using the Copilot CLI, the `gh aw` debugging commands, and manual investigation techniques.

All agent entry points use the same [debug.md bootstrap and inventory](https://github.com/github/gh-aw/blob/main/debug.md)
and [shared debugging strategy](https://github.com/github/gh-aw/blob/main/.github/aw/debug-agentic-workflow.md).
Start with existing evidence; diagnosis does not authorize dispatch. The shared
strategy owns local execution, MCP startup review, compilation, and live-test
authorization. For enterprise host configuration, retain the
[GHE-specific guide](/gh-aw/troubleshooting/debug-ghe/).

> [!TIP]
> The fastest path to a fix is to let an AI agent debug it for you. Launch the Copilot CLI, invoke the `agentic-workflows` skill, and paste the failing run URL.

## Debugging with the Copilot CLI

The Copilot CLI is the recommended first step: it audits logs, traces failures, and suggests fixes interactively.

Launch `copilot`, invoke **agentic-workflows** to enable `gh aw audit`, `gh aw logs`, and related debugging tools. Then paste the failing run URL:

```text
Debug this workflow run: https://github.com/OWNER/REPO/actions/runs/RUN_ID
```

Copilot downloads the logs, identifies the root cause (missing tools, permission errors, network blocks), and suggests fixes or opens a PR. Follow-up questions like "What domains were blocked?" or "Why did the MCP server fail?" work too.

### Alternative entry points

- **Copilot Chat on GitHub.com** (requires [agentic authoring setup](/gh-aw/setup/working-with-workflows/#configuring-your-repository-for-agentic-authoring)): `agentic-workflows debug <run-url>`
- **Any coding agent**: paste this prompt to install `gh aw` and run the standalone analysis:

  ```text
  Debug this workflow run using https://raw.githubusercontent.com/github/gh-aw/main/debug.md

  The failed workflow run is at https://github.com/OWNER/REPO/actions/runs/RUN_ID
  ```

## Debugging with CLI Commands

### Auditing a Specific Run

`gh aw audit` breaks down a single run — failure analysis and root cause, behavior fingerprint (network/tool/cost profile), tool usage, MCP server status, firewall analysis, token/cost metrics, and safe-outputs. Accepts a run ID, run URL, job URL (extracts first failing step), or step URL.

```bash
gh aw audit https://github.com/OWNER/REPO/actions/runs/12345678 --no-baseline --timeout 2m
gh aw audit https://github.com/OWNER/REPO/actions/runs/123/job/456#step:7:1
gh aw audit 12345678 --repo OWNER/REPO --parse         # parse to markdown
gh aw audit 12345678 12345679 --repo OWNER/REPO        # compare two runs
gh aw audit 12345678 12345679 --repo OWNER/REPO --group --json  # recurrence findings
```

For security and performance trends across multiple runs, use `gh aw logs --format`:

```bash
gh aw logs my-workflow --format markdown --count 10
gh aw logs my-workflow --format markdown --last 5 --json
```

See [Audit Commands](/gh-aw/reference/audit/) for complete flag documentation.

### Analyzing Workflow Logs

`gh aw logs` downloads and analyzes logs across multiple runs (tool usage, network patterns, errors). Results are cached for 10–100× speedup on later runs.

```bash
gh aw logs my-workflow
gh aw logs my-workflow -c 10 --start-date -1w   # filter by count and date
gh aw logs my-workflow --firewall               # include firewall analysis
gh aw logs my-workflow --safe-output            # include safe-output details
gh aw logs my-workflow --json                   # JSON for scripting
```

Logs reports automatically group failed, timed-out, cancelled, and startup-failure runs into **failure investigation clusters** using failed jobs/steps, MCP failures, missing tools/data, and blocked domains already available in the downloaded metadata. Clusters describe shared symptoms, not proven root causes; they can overlap, so do not sum their counts to compute a failure rate. Intentional failures and different conclusions/failure kinds remain separate.

Start with `[failure_investigation]`: unique failure/evidence counts and a deduplicated initial plan of up to five runs. Selection prioritizes specific failure boundaries, then maximizes new cluster coverage and affected-run reach, preferring richer/newer diagnostics when tied. Generic failure-handler `aftermath` ranks last; intentional failures are excluded from recommendations. Covered runs measure shared-symptom reach, not confirmed common causes. The report exposes remaining cluster counts so the initial plan is not mistaken for an exhaustive investigation.

The default `[failure_clusters]` output is a compact overview with representative IDs. Use `-v` for text membership or project `failure_investigation` from `--json` before reading the full run list. JSON and summary files retain full `failure_clusters`, per-run `failure_cluster_ids`, typed `signal.job`/`signal.step`, repository/workflow/engine/model facets, and first/last-seen timestamps. Up to three representatives per cluster prefer distinct workflow/engine/model contexts; each includes its context and cached `audit_path`/`logs_path`. Read cached evidence before invoking another audit. Stable cluster IDs identify the same symptom across queries; membership reflects only the returned runs.

No extra downloads or model calls are required. When detailed evidence is unavailable, a `metadata_only` cluster groups runs within the same workflow and repository without asserting a shared cause. Cached reports retain extracted signals; older cache records without signals use this metadata-only grouping.

Each cluster's `statistics` exposes its `failed_population`, observed `share`, and Wilson 95% bounds (`wilson_lower_95`/`wilson_upper_95`). The denominator includes unique returned failures, including metadata-only runs, with intentional failures in a separate population. These are symptom shares, not workflow failure rates. Bounds assume independent binomial observations; correlated retries, incomplete collection, and missing diagnostics limit interpretation. Do not use them as causal confidence, a significance test, or a forecast. Audit selection caches identical symptom profiles and uses marginal affected-run reach within each signal role to avoid rewarding already-covered runs.

### Checking Workflow Health

`gh aw health` reports workflow status across all workflows in a repository.
It walks backward independently for each active AW until it has five completed, first-attempt runs for the current revision, encounters a revision boundary, or exhausts the maximum horizon (`--days 7`, `30`, or `90`; default `7`). Rates, trends, durations, and token averages describe the recent current-revision sample, not the entire period. Sparse workflows and workflows with no recent runs remain visible as insufficient evidence.

Revision identity uses `frontmatter_hash`, `body_hash`, and compiler version from `gh-aw-metadata` at each run's immutable commit, not repository SHA changes alone. A branch-head metadata check detects pushed revisions with no runs yet. JSON `workflows[].revisions` (or `revisions` for one workflow) separates branch/event cohorts and includes current/previous hashes, counts, and run IDs. `recovery_observed` requires five successful first-attempt runs after a changed revision whose sampled predecessor failed and finished before those runs began. This is observed recovery, not proof that a particular commit fixed the cause. Missing hashes and triggers whose head SHA does not identify the executing workflow, including `workflow_run` and `pull_request`, remain `revision_unknown`.

`failure_clusters` and `failure_investigation` describe current-revision failures using the same implementation as `logs`; historical failures remain in revision comparisons, not the current audit plan. Health reuses fresh summaries from `.github/aw/logs` (override with `--logs-dir PATH`) without downloading artifacts. Missing or stale summaries produce `metadata_only` clusters. Cluster shares are not workflow failure rates.

Health checks the target host's core API quota, reuses immutable revision lookups, and stops rather than sleeping or retrying on primary or secondary rate limits. `collection` records quota, request counts, and partial results. Collection is bounded to 1,000 data requests and 20 pages per workflow; quota, request, or pagination limits are explicit. Incomplete evidence never counts as healthy. Use `logs` for a wider historical population and `audit` for targeted comparisons.

### Inspecting MCP Configuration

If you suspect MCP server issues, first review declarations, startup commands,
destinations, and credential references under the shared strategy. Commands that
start or connect to a server require its MCP startup gate; use cached schemas
when isolated test bindings are unavailable. Then inspect the configuration:

```bash
gh aw mcp list                              # list workflows with MCP servers
gh aw mcp inspect my-workflow               # inspect a workflow
gh aw mcp inspect my-workflow --inspector   # web-based inspector
```

## Common Errors

### "Authentication failed"

The Copilot token is missing, expired, or lacks the required permissions. Confirm you have an active Copilot subscription, that the token has **Copilot Requests** permission (fine-grained PATs), and that `gh auth status` reports it valid. See [Authentication Reference](/gh-aw/reference/auth/).

### "Tool not found" or Missing Tool Calls

The workflow references a tool that isn't configured or the MCP server failed to connect. Verify the `tools:` section in frontmatter, check the MCP server version, then run `gh aw mcp inspect my-workflow` and `gh aw audit <run-id>` to compare available vs. requested tools.

### Shell Commands Missing Inside the Agent Sandbox

`command not found` for `jq` or `safeoutputs` is different from an unavailable MCP tool. The setup message `Successfully copied ... safe-outputs files` confirms that server scripts were copied to the runner. It does not confirm that the CLI wrapper was mounted or that its interpreter and directory are available on the agent's `PATH`.

For intermittent failures, compare one successful and one failing run from the same workflow and deployment. Retain both run URLs, the gh-aw and engine versions, the exact `sandbox:` block, and the generated lock-file image, mounts, and `PATH` configuration. Compare setup, MCP startup, CLI mounting, and agent launch logs. Different self-hosted runner images, custom sandbox images, missing mounts, or shell startup files that reset `PATH` are possible causes, not proof of a provisioning race.

Check `command -v safeoutputs` in the **agent's shell** when permitted, not just in a runner setup step. If the CLI is unavailable, use direct safe-output MCP tools only when the engine exposes them. CLI payloads do not require a separate serializer: pass one single-quoted JSON object, encoding newlines as `\n`, double quotes as `\"`, backslashes as `\\`, and apostrophes as `\u0027`. Unicode can remain literal.

```bash
safeoutputs add_comment '{"item_number":42,"body":"### Result\n\nIt\u0027s ready.\nQuoted text: \"hello\".\nLiteral path: C:\\temp\\file.txt"}'
```

Stdin is also supported: `safeoutputs <tool_name> .` reads a complete JSON object from stdin. Prefer inline JSON without auxiliary tools, pipelines, temporary files, or shell expansion. Stdin support does not grant permission for additional commands or redirections, and quoting cannot guarantee acceptance by the engine's independent shell guard. Do not repeat a denied command or bypass the denial with an unapproved workaround.

Report unavailable functionality with [`report_missing_tool`](/gh-aw/reference/safe-outputs/) when configured and reachable; reserve `report_incomplete` for failed write attempts. If no supported safe-output transport is available, the agent must report the missing command in its final response without claiming that a diagnostic or write was emitted.

Collect diagnostics with the repository's [debug.md guidance](https://github.com/github/gh-aw/blob/main/debug.md). Redact secrets from logs and configuration; do not dump the full environment or make a real safe-output call merely to test availability.

### Network / Firewall Blocks

```text
DENIED CONNECT registry.npmjs.org:443
```

The agent reached a domain outside the firewall allow-list. Add it explicitly, or use an ecosystem shorthand:

```aw
network:
  allowed:
    - defaults
    - node        # npm, yarn, pnpm registries
    - python      # PyPI, conda registries
    - registry.npmjs.org
```

See [Network Configuration](/gh-aw/guides/network-configuration/) for common domain configurations.

### Safe-Outputs Not Creating Issues / Comments

The safe-outputs job failed, the agent didn't produce the expected output, or permissions are missing. Inspect the safe-outputs section in `gh aw audit <run-id>` and review the [Safe Outputs Reference](/gh-aw/reference/safe-outputs/).

### Silent Long-running Commands After a Result

If the harness log says `post-result watchdog terminating idle process`, the agent already emitted a terminal safe output and then the child process went quiet. The watchdog is dormant until a terminal safe output such as `noop`, a comment, label, push, or pull request creation is observed; diagnostic outputs such as `report_missing_tool`, `report_missing_data`, and `report_incomplete` do not arm it. After arming, stdout or stderr activity resets the clock, so quiet builds, tests, or monorepo scans can be terminated even while they are still doing useful CPU or I/O work.

Increase `engine.harness.watchdog-timeout` for literal frontmatter values, or set `GH_AW_HARNESS_WATCHDOG_TIMEOUT_MS` when you need a millisecond environment-variable override. See [Harness Settings and Runtime Tuning Variables](/gh-aw/reference/environment-variables/#harness-settings-and-runtime-tuning-variables).

### Compilation Errors

The frontmatter has schema validation errors or unsupported fields. Use `--verbose` to diagnose, `gh aw fix --write` to auto-correct, and `--validate` to check without writing the lock file. See [Error Reference](/gh-aw/troubleshooting/errors/) for specific messages.

```bash
gh aw compile my-workflow --verbose
gh aw fix --write
gh aw compile --validate
```

## Advanced Debugging

### Enable Debug Logging

`DEBUG` enables detailed internal logging for any `gh aw` command. Output goes to `stderr` — capture it with `2>&1 | tee debug.log`.

```bash
DEBUG=* gh aw compile my-workflow              # all logs
DEBUG=cli:* gh aw audit 12345678 --repo OWNER/REPO  # CLI-specific
DEBUG=workflow:*,cli:* gh aw compile           # multiple packages
```

### Enable GitHub API Request Logging

`gh aw` uses go-gh's native REST/GraphQL clients for many GitHub API calls. Setting `GH_DEBUG=api` prints verbose request/response details for calls made through those go-gh clients, with no extra flags needed:

```bash
GH_DEBUG=api gh aw compile my-workflow
```

### Enable GitHub Actions Debug Logging

`ACTIONS_STEP_DEBUG=true` enables verbose step-level logging in the Actions UI.
Changing repository secrets or rerunning a workflow is not part of read-only
diagnosis; use the shared strategy's authorization and live gates before either.

### Inspecting Firewall Logs

Workflow run artifacts include `sandbox/firewall/logs/access.log`. Each line shows allowed (`TCP_TUNNEL`) or blocked (`DENIED`) traffic:

```text
TCP_TUNNEL/200 api.github.com:443
DENIED CONNECT blocked-domain.com:443
```

Or use the CLI: `gh aw logs my-workflow --firewall`, or `gh aw audit <run-id>` for combined firewall analysis.

### Inspecting Artifacts

Workflow runs produce several artifacts useful for debugging:

| Artifact | Location | Contents |
|----------|----------|----------|
| `prompt.txt` | `/tmp/gh-aw/aw-prompts/` | Full prompt sent to the AI agent |
| `agent_output.json` | `/tmp/gh-aw/safeoutputs/` | Structured safe-output data |
| `agent-stdio.log` | `/tmp/gh-aw/` | Raw agent stdin/stdout log |
| `firewall-logs/` | `/tmp/gh-aw/firewall-logs/` | Network access logs |

Download from the Actions run page or `gh run download <run-id> --repo OWNER/REPO`.

### Recompiling for a Quick Fix

After editing the `.md` source, use the shared strategy's review and development
compilation gates:

```bash
gh aw compile my-workflow --dry-run
```

This emits a diagnostic lock, not a live run or a normal production replacement.
Preserve evidence and user changes using the strategy's snapshot/restore rules.
Publication and live testing follow its feature-branch and exact-revision gates;
do not infer permission to push or rerun from a successful compile.
