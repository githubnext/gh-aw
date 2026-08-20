# ADR-54341: Filter Runner-Guard RGS-012 False Positives for Copilot Local Allow-Tool Declarations

**Date**: 2026-08-20
**Status**: Draft
**Deciders**: pelikhan (via Copilot SWE agent)

---

### Context

Runner-Guard enforces rule RGS-012, which flags workflow lines that reference `curl` with a local-host URL as potential secret exfiltration attempts. The Copilot CLI workflow compiler generates YAML comment headers (e.g., `# --allow-tool shell(curl http://localhost:*)`) and `--allow-tool` argument lines documenting the tools the agent is permitted to use. These declarations are never executed by the shell — they are documentation artifacts — but Runner-Guard cannot distinguish them from executable `curl` commands. This causes systematic false-positive RGS-012 findings that block legitimate Copilot-enabled workflows from passing the Runner-Guard gate. The fix must suppress only the false positives while preserving all genuine exfiltration signals.

### Decision

We will add a targeted post-processing filter, `filterCopilotLocalAllowToolFindings`, in the Runner-Guard output pipeline. The filter suppresses RGS-012 findings that point at lines within a Copilot CLI execution step whose `--allow-tool` declarations reference only loopback hosts (`localhost`, `127.0.0.1`, `::1`, `host.docker.internal`). It preserves findings on all other lines, on steps that mix local and non-local targets, and on any other rule.

### Alternatives Considered

#### Alternative 1: Patch the Runner-Guard RGS-012 rule to understand Copilot allow-tool syntax

Runner-Guard is a separate tool maintained outside this repository. Modifying its rule logic would require upstream changes and cross-team coordination. A local post-processing filter can be shipped independently without blocking on upstream changes, making it the lower-friction path to resolution.

#### Alternative 2: Add a blanket suppression for all RGS-012 findings on local-curl targets

Suppressing every local-curl RGS-012 finding would hide genuine exfiltration attempts such as `curl http://localhost:4321/collect -d "secret=$SECRET_TOKEN"` inside executable `run:` blocks. The PR's safety requirement explicitly preserves those findings; a blanket suppression is therefore ruled out.

### Consequences

#### Positive
- Eliminates systematic false-positive noise for Copilot CLI workflows that include local `--allow-tool` declarations.
- Preserves security signal: executable `curl` commands to local endpoints with payloads remain flagged by RGS-012.
- Steps that mix local and non-local allow-tool targets are intentionally not suppressed, keeping the safety boundary strict.

#### Negative
- The filter depends on the Copilot CLI's generated comment format (`# Copilot CLI tool arguments`) and step name (`Execute GitHub Copilot CLI`). If either changes, the filter silently stops suppressing false positives — it will not introduce new false negatives, but the benefit is lost until the filter is updated.
- Any new local-endpoint alias beyond the four currently recognized (`localhost`, `127.0.0.1`, `::1`, `host.docker.internal`) must be explicitly added to `isLocalCurlAllowToolHost`.

#### Neutral
- This is the third targeted filter added to the Runner-Guard post-processing pipeline, following the pattern of `filterRunnerGuardIgnoredFindings` and `filterGvisorInstallFindings`.
- The implementation is isolated in a new file (`runner_guard_copilot_allow_tool.go`), keeping `runner_guard.go` uncluttered.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
