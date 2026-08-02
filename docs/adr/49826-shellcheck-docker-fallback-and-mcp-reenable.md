# ADR-49826: Shellcheck Docker Fallback for Systems Without Native Binary; Re-enable in MCP

**Date**: 2026-08-02
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The shellcheck integration in the compile pipeline (introduced in ADR-49762) relied exclusively on a native shellcheck binary being present in PATH. On platforms that lack a native package (notably Windows), shellcheck was silently skipped, leaving run: step linting inoperative for those users. Separately, MCP compilation explicitly disabled shellcheck via `--no-shellcheck` on the assumption that shellcheck output would not reach the LLM — but this also prevented any shell script validation from occurring in that context.

The goal is to make shellcheck linting reliable across all supported platforms and re-enable it in MCP compilation, without requiring users to install shellcheck natively.

### Decision

We will add a lazy Docker container fallback for shellcheck (`koalaman/shellcheck:v0.10.0`, SHA-pinned) that is invoked only when the native binary is absent and Docker is available. The Docker path pipes scripts via stdin (no volume mount required). In parallel, we will remove `--no-shellcheck` from MCP compilation so that shellcheck runs during MCP compilation via whichever path (binary or Docker) is available.

The fallback precedence is: native binary → Docker container → silent skip. Warnings and errors in `--strict`/`--validate` mode fire only when neither path is available.

### Alternatives Considered

#### Alternative 1: Require Native Binary Installation

Continue requiring users to install shellcheck natively on every platform. Document the requirement more prominently, and surface a clear error on unsupported platforms.

Rejected because it creates a hard barrier for Windows users and automated MCP environments where installing system packages is not feasible. The user experience degrades from "linting is silently skipped" to "an error is shown", with no practical linting benefit on those platforms.

#### Alternative 2: Keep Shellcheck Disabled in MCP

Continue passing `--no-shellcheck` during MCP compilation, accepting that LLM-generated workflows are never linted for shell script quality.

Rejected because the original rationale (shellcheck output not reaching the LLM via JSON response) was addressed by the Docker fallback design: shellcheck now has a reliable execution path in any environment that has Docker running, which covers the MCP use case.

#### Alternative 3: Require Docker Pull at Startup

Pull the shellcheck Docker image eagerly at compile startup rather than lazily on first use.

Rejected in favor of the lazy pull: Docker is only contacted when there are actual bash/sh run: steps to lint. An eager pull would add latency to all compilations, including those with no shell scripts.

### Consequences

#### Positive
- Shellcheck linting is now available on all platforms that have Docker, including Windows.
- MCP-compiled workflows receive shell script validation, improving quality of AI-generated workflow steps.
- The Docker image is SHA-pinned, preventing unexpected behavior from upstream image changes.
- No temporary files or volume mounts are required; scripts are piped via stdin.

#### Negative
- Docker must be running for the fallback to work; users without Docker on non-PATH platforms still get silent skips.
- The first use of the fallback triggers a Docker image pull, adding latency that is invisible to the user.
- Adding Docker as an implicit runtime dependency for shellcheck increases the surface area of the tool's external dependencies.

#### Neutral
- MCP shellcheck output goes to stderr (not the JSON response body), so the LLM does not see individual findings — only whether compilation succeeded or failed.
- The existing `--no-shellcheck` flag remains available for users who want to opt out of linting entirely.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
