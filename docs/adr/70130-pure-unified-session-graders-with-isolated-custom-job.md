# ADR-70130: Pure Unified-Session Graders with an Isolated Custom Grader Job

**Date**: 2026-10-19
**Status**: Draft
**Deciders**: pelikhan, [TODO: verify additional reviewers]

---

### Context

Graders score agentic workflow runs. Before this change, they could read from the filesystem, be Bash or file-backed (`run`), override built-in implementations, and run user-provided code or steps inside the trusted `conclusion` job. They also consumed legacy trace payloads, which gave an incomplete and inconsistent view of the evidence. Running user JavaScript and arbitrary steps in `conclusion`, which handles privileged safe-output and reporting work, widens the trusted surface. It also makes grading results depend on whatever files happen to be on disk. Some trajectory analyses are also expensive, so running every built-in by default costs time on every run.

### Decision

We will make every grader a pure computation over the redacted, in-memory unified agent session. Built-in graders run in `conclusion` after session collection. Ten fast built-ins are enabled by default, and expensive trajectory analyses are opt-in. Custom `graders.<id>.script` JavaScript runs only in a separate downstream `graders` job (`needs: [activation, conclusion]`, `contents: read`). There, a Node worker gets only immutable, frozen `session`, `config`, and `helpers`. The worker runs under the Node permission model with an empty environment, a 256 MB heap cap, a 7 s timeout, and a `vm` context with `require`, `process`, `fetch`, `eval`, `Function`, and similar globals removed. The compiler rejects Bash/file graders, `run`, the reserved `operational-value` ID, built-in overrides, and user-provided conclusion steps. A new AST-based `grader-file-io` lint rule and a custom-job threat-detection gating rule enforce the contract statically. Built-in results ship with the `usage` artifact under `usage/graders/` (schema v2). Custom results go to a separate, source-free `graders` artifact.

### Alternatives Considered

#### Alternative 1: Keep custom graders in `conclusion` with tighter sandboxing

We could keep running custom JavaScript inside `conclusion` and only add a `vm`/permission sandbox there. That avoids an extra job and a second artifact. We rejected it because user-provided code would still run in the trusted job that holds safe-output privileges. Any sandbox escape would then compromise the most sensitive part of the pipeline. Job-level isolation adds a second boundary that does not depend on the sandbox alone.

#### Alternative 2: Keep file-backed and Bash graders alongside the new contract

We could support both the legacy file/Bash/`run` graders and the pure session contract to avoid a breaking change. We rejected it because the legacy forms need filesystem and process access by design. That rules out reproducible replay (`gh aw graders run`) from a single session file and leaves the trusted surface larger. The PR chooses a major-version break with a documented migration instead.

#### Alternative 3: Enable all built-in graders by default

Running every trajectory analysis by default would give the most coverage without configuration. We rejected it because the expensive analyses add cost to every workflow run. Only ten fast built-ins are on by default; the rest are opt-in.

### Consequences

#### Positive
- No user-provided scripts or steps run in the trusted `conclusion` job, which shrinks its attack surface.
- Graders are deterministic functions of the session. That allows faithful replay from unified-session JSONL or a JSON event array and consistent results across audit, logs, and experiments.
- Restrictions are enforced at several layers: the compiler rejects disallowed forms, AST lint checks file I/O, the job has restricted permissions, and the runtime has a Node permission model, `vm` sandbox, and memory/time limits.
- Shared statistics, correlated tool-call lookups, and event indexes let graders reuse precomputed views instead of re-parsing the session.

#### Negative
- This is a breaking change. Bash/file graders, `run`, `operational-value`, built-in overrides, and user conclusion steps are rejected. Grader artifacts move to schema v2 and a new layout, and all workflows must be recompiled (all 345 lock files were regenerated).
- Workflows with custom graders gain an extra `graders` job and artifact. That adds runner time, and audit, logs, cached-report refresh, and experiments must merge the two result sources.
- Custom graders cannot read files, call the network, or load modules. Analyses that relied on such access must be rewritten or dropped.
- Sandbox tuning (256 MB heap, 7 s timeout) may be too tight for some legitimate graders. [TODO: verify whether limits are configurable]

#### Neutral
- The JavaScript `grader-file-io` lint rule relies on the existing opt-in `--javascript-module` parser. Without it, the rule reports unchecked coverage instead of failing.
- Custom agent-dependent jobs, including both TruffleHog jobs, were hardened to require successful threat detection under a new default AW rule.
- Schemas, docs (graders specification, trace-graders, artifacts, audit, CLI), and skills were updated to describe the new contract.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
