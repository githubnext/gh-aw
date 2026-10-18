---
"gh-aw": major
---

Add opt-in per-rule performance markers and Go trace regions, with registry-complete benchmarks and AST workload budgets for runtime and allocation growth.

Rules are defined only against parsed AW or Actions ASTs, with explicit target metadata. The CLI accepts AW `.md` and native Actions `.yml`/`.yaml` as well as existing `.lock.yml` files without compilation. AW-only execution checks inspect explicit authored nodes and report missing generated-flow coverage instead of fabricating an execution graph. Directory scans avoid paired-file duplication, and Markdown is never passed to actionlint.

Add an extensible Actions/AW AST lint-rule framework and reusable source/sink/filter taint analysis. Run all registered AST checks alongside actionlint by default with `gh aw lint`, select specific checks with `--rules secret-leak`, list rules with `--list-rules`, and export value-free diagnostics with `--rules secret-leak --json`. Apply the rules to repository workflows in regression tests and move the Gemini inventory API key from its request URL to an authentication header.

Add threat-spec-mapped AST rules for action pinning, direct template injection, workflow-run scope, and literal job timeouts. Preserve independent diagnostics when another rule fails, and document the boundary between AST evidence and full compiler/runtime conformance.

Add permission-scope checks for blanket write grants and invalid OIDC reads, plus companion-AW source rules for disabled version checks, conditional imports, and private-to-public flow opt-outs. Respect authored strict policy without inferring job authorization or missing source configuration.

Convert further compiler policies into default source linters: enabled Node install hooks, bare safe-output label allowlists, bare network allowances, and unconstrained wildcard PR push targets. Preserve valid domain/label patterns, legacy push constraint aliases, literal required-label names, and import uncertainty; do not infer missing firewall configuration or effective runtime policy.

Add AST-only checks for shell/proxy tool-policy contradictions and cache isolation, sandbox bypasses/internal overrides, option-prefixed container image operands, explicit checkout expansion prerequisites, and fork author-association values. Keep runtime authorization, engine capabilities, selector trust, and provenance outside these observable subsets.

Model value-free shell predicates, no-op builtins, and presence-only alternate expansions without spurious secret-flow warnings. Scope file-coverage warnings to possible modeled secret flows rather than unrelated public file operations. Preserve disclosures through tracing, nested commands, and parameter-expansion errors; include script-relative locations for unmodeled command warnings.

Add opt-in JavaScript taint analysis using the Go TypeScript compiler through its pinned, upstream-exported native-preview AST client. Track symbolic environment, stdin, arguments, objects, local modules, files, and disclosure sinks without executing analyzed programs. Require explicit compiler/source configuration, report unsupported flows as coverage gaps, and do not invent secret-bearing output at unresolved boundaries.

Hide secret-leak coverage warnings in normal CLI text/JSON output and restore them with `lint --debug`; keep the warnings in AST results and preserve error findings, analysis failures, and other rules' warnings. Preserve member provenance when copying JavaScript object-rest properties so credential-bearing environment members do not falsely taint public timeout values. Require positive/negative functional contracts for every registered linter and supported AST.

**Breaking change:** `gh aw lint` now runs built-in AST rules by default, so workflows that previously passed actionlint-only linting may report AST findings and return a nonzero exit status. Review and fix those findings, or run `actionlint` directly on the generated `.lock.yml` files to retain actionlint-only behavior. `gh aw lint --rules=...` selects AST rules and skips actionlint; it is not an actionlint-only mode.
