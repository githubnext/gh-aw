---
title: Workflow Linters
description: Built-in Actions/AW lint rules, compiler threat references, analysis boundaries, and performance diagnostics.
---

`gh aw lint` checks authored AW Markdown and parsed Actions YAML without recompilation. See the [CLI reference](/gh-aw/setup/cli/#lint) for command options and file discovery.

## Built-in rules

The compiler threat IDs reference the [Compiler Threat Detection Specification](https://github.com/github/gh-aw/blob/main/specs/compiler-threat-detection-spec.md). Each mapping covers only the observable checks described here, not full conformance to the threat rule. Extension rules can omit a threat ID when no mapping is known.

| Built-in rule | Threat ID | ASTs | Checks |
|---|---|---|---|
| `action-pin` | `CTR-002` | AW, Actions | Remote actions and reusable workflows use full commit SHAs; Docker actions use digests. Local references are exempt. |
| `job-timeout` | `CTR-026` | AW, Actions | An explicit workflow/job timeout is a positive integer literal, not an expression. Omission is allowed. |
| `template-injection` | `CTR-006` | AW, Actions | Known attacker-controlled expressions are not interpolated into executable `run` text. Use environment bindings and quoted shell expansions instead. |
| `workflow-run-scope` | `CTR-021` | AW, Actions | `workflow_run` selects workflows and has a positive branch scope, not just wildcard, negated-only, or empty branch filters. |
| `secret-leak` | `CTR-017` | AW, Actions | Symbolic secrets do not reach modeled disclosure sinks. AW-only checks report missing generated execution coverage. |
| `permission-scope` | `CTR-001` | AW, Actions | Explicit permissions avoid blanket `write-all` grants; OIDC does not use invalid `id-token: read`. Scoped writes and `id-token: write` are not automatically violations. |
| `version-check` | `CTR-018` | AW | Authored source does not disable `check-for-updates`. |
| `conditional-import` | `CTR-020` | AW | Configuration import entries do not contain `if`, even if false or null. Runtime prompt conditionals are unaffected. |
| `private-to-public` | `CTR-031` | AW | GitHub tool configuration does not opt out of private-to-public flow protections, globally or for selected servers. |
| `install-scripts` | `CTR-014` | AW | Authored Node configuration does not enable dependency install hooks. Other runtimes and arbitrary shell commands are not inferred. |
| `allowed-labels` | `CTR-015` | AW | Issue, discussion, discussion-update, and PR creation label allowlists do not contain a bare `*`. Narrow patterns and literal required-label names are unaffected. |
| `network-scope` | `CTR-011` | AW | Network allowances do not contain a bare `*`, including alongside `defaults`. Domain wildcards such as `*.example.com` are allowed; omission does not imply missing firewall protection. |
| `push-target` | `CTR-012` | AW | A literal wildcard PR-branch push target has an authored title or label constraint. Respects deprecated aliases and defers when unresolved imports may supply constraints. |
| `tool-scope` | `CTR-003` | AW | Shell refusal does not conflict with CLI/gh proxies; `min-integrity: none` declares shell intent; cache memory avoids repository-wide scope. Missing prerequisites defer when imports may supply them. |
| `sandbox-config` | `CTR-004` | AW | The agent sandbox is not explicitly disabled and internal agent/MCP command, environment, argument, image, or version overrides are not enabled. Empty overrides are ignored. |
| `image-argument` | `CTR-013` | AW, Actions | Job/service and authored MCP container image operands do not start with `-`. Does not inspect registries or infer dynamically computed images. |
| `checkout-mode` | `CTR-033` | AW | Repository-list expansion is explicitly boolean; literal lists require multi mode, which requires a literal non-root prefix and scoped allowlist without LFS, submodules, current identity, or wiki expansion. Does not prove runtime selector trust or repository visibility. |
| `fork-association` | `CTR-034` | AW | Explicit fork admission uses GitHub's uppercase author-association enum. Empty lists and repository wildcard patterns are valid; generated admission guards are not reconstructed. |
| `pull-request-target-roles` | `CTR-008` | AW | `pull_request_target` does not admit explicit `read` or `all` roles. Checkout disablement and risk acknowledgment do not bypass this policy. Does not infer checkout safety or generated authorization. |
| `safe-output-shell-expansion` | `CTR-009` | AW | Authored `safe-outputs.steps[*].run` scripts avoid the shell expansion patterns blocked by the compiler/harness: parameter transformation, indirect expansion, command substitution, and backticks. Arithmetic and ordinary variables remain allowed. |
| `firewall-ssl-bump` | `CTR-011` | AW | Nonempty literal `--allow-urls` in `sandbox.agent.args` requires `--ssl-bump` in the same list. Unsupported legacy `network.firewall` blocks require migration, even with `ssl-bump: true`. Does not prove runtime HTTPS enforcement. |

## Severity and source policy

Action pinning, workflow-run branch scope, blanket write permissions, disabled version checking, private-to-public opt-outs, enabled Node install hooks, and bare network wildcard allowances are errors for an AW document with strict mode enabled (the AW default), and warnings for explicit non-strict AW. Native Actions policy-sensitive findings are warnings; AW-only rules skip native Actions without an AW document. Conditional imports, bare label-allowlist wildcards, invalid OIDC read grants, missing or malformed `workflow_run.workflows`, invalid explicit timeouts, and direct template injection are errors. Unconstrained wildcard push targets remain warnings even in strict mode. The linter does not reconstruct CLI/repository strict overrides, load imported configurations, infer missing permissions, or determine whether each repository-write grant is authorized.

Tool-scope and sandbox-config use the authored strict-policy error/warning distinction, including explicit non-strict sandbox opt-outs and internal customizations. Image-argument, checkout-mode, and fork-association findings are always errors. Checkout and tool checks defer absent prerequisites that unresolved imports may supply; they still report explicit contradictions. Image checks cover image operands, not inferred package-manager arguments.

Pull-request-target-roles rejects unsafe role admission in strict mode and warns in non-strict mode. Only the non-strict warning can be suppressed with boolean `features.dangerously-allow-roles-read-and-pull-request-target: true`; this does not suppress other checks. Safe-output-shell-expansion and firewall-ssl-bump are errors in either mode. The firewall rule inspects authored AWF arguments; `sandbox.agent.args` is an internal customization rejected in strict mode. The shell rule shares the compiler's lexical policy, including matches in quoted text or comments, and reports script-relative lines without script contents. It does not apply to ordinary Actions steps or infer safe-output jobs from their names.

Firewall argument checks inspect separate `--allow-urls`, URL operands and `--allow-urls=...` tokens before an option terminator (`--`). A literal `--ssl-bump` or `--ssl-bump=true` satisfies the lint prerequisite unless followed by `--ssl-bump=false`; the last SSL setting wins. The rule does not interpret arbitrary command strings, resolve imports, or validate runtime AWF option semantics. Internal `sandbox.agent.args` overrides require `strict: false`; adding SSL inspection does not make them valid in strict mode. A non-strict example is:

```aw wrap
strict: false
sandbox:
  agent:
    args: [--ssl-bump, --allow-urls, "https://example.com/path"]
```

The compiler rejects `network.firewall` regardless of its SSL setting. Remove that legacy block and use `network.allowed` for domain policy; do not add `network.firewall.ssl-bump` as a remedy.

## Secret-flow analysis

The `secret-leak` rule tracks symbolic secret references, `github.token`, and known app-token action outputs through environment scopes, shell assignments, step outputs, and dependent job outputs. It detects disclosures to logs, step summaries, workflow outputs, request URLs/bodies, known publication inputs, cache keys, and artifacts containing modeled secret-bearing files. Authentication headers, token inputs, and the reusable workflow `secrets:` channel are not disclosure sinks. Boolean comparisons and predicates filter secret values; formatting, JSON serialization, encoding, and `add-mask` registration do not sanitize subsequent uses.

In Bash/sh, recognized `test`/`[` string and file predicates and the `:`, `true`, and `false` builtins do not disclose their arguments. Presence-only alternate expansions such as `${TOKEN:+present}` do not contain the token, but secret-bearing replacements remain tainted. Nested command disclosures, parameter-expansion error messages, and `set -x` output are still checked. Numeric comparisons, malformed tests, and potentially word-splitting predicates retain coverage warnings because their error diagnostics can expose operands.

Unknown file reads produce secret-flow coverage warnings when they could select a modeled secret-bearing file or involve a secret-bearing filename. Unknown output targets warn when the output or filename is tainted; unrelated public file operations do not introduce secret-flow warnings.

For `.lock.yml` inputs, a companion `.md` adds authored-policy and prompt checks to the existing execution graph. Native `.yml`/`.yaml` inputs never acquire an assumed AW companion. Explicit `.md` inputs inspect the AW AST independently: the prompt, top-level `steps`/`post-steps`, and authored `jobs`/`safe-jobs`. No compiled graph is fabricated. Secret analysis retains a coverage warning for missing imports, generated steps, cross-job flows, and compiler defaults, visible with `--debug`; scripts without an observable shell remain unknown.

Analysis is conservative within Bash/sh control flow and can report infeasible paths. It does not execute analyzed code or inspect external actions, sourced shell scripts, Python/PowerShell programs, dynamically constructed shell code, or arbitrary filesystem operations. JavaScript inspection requires the explicit configuration below. Imported runtime prompt files are not recursively loaded. Authentication arguments are treated as intentional credential use; endpoint trust is not inferred. A clean result is not proof that a workflow cannot leak secrets.

## Diagnostics

Text findings include the linter rule ID and its known compiler threat ID, for example `[safe-output-shell-expansion] (CTR-009)`. `--list-rules` includes the same threat ID after the rule description. JSON diagnostics contain the file, rule ID, optional `threat_rule_id`, severity, AST path, optional script-relative line, message, and symbolic source locations, never resolved secrets or script contents. The threat ID is omitted for unmapped rules. Errors produce a nonzero exit status; coverage warnings alone do not.

## Optional native JavaScript analysis

This analysis is experimental and opt-in. `--javascript-module` enables JavaScript secret-flow analysis using Microsoft's Go TypeScript compiler through the upstream-exported, **unstable** `@typescript/native-preview/unstable/sync` and AST APIs. The exact supported package version is `7.0.0-dev.20260707.2`. Node.js 22.12 or newer and its platform-native compiler executable must already be installed; the CLI does not download dependencies or assume its own developer `node_modules` directory exists.

For a gh-aw source checkout, install the optional pinned dependency and supply absolute paths:

```bash wrap
npm ci --prefix /path/to/gh-aw/pkg/workflow/taint/javascript --ignore-scripts
gh aw lint --rules secret-leak --json \
  --javascript-module /path/to/gh-aw/pkg/workflow/taint/javascript \
  --javascript-root /path/to/repository
```

The checker follows literal `node`/`nodejs` file and `--eval` invocations, `--print` expressions, `shell: node {0}`, `actions/github-script` scripts, and available relative CommonJS/ES module imports. It tracks symbolic environment/argument/stdin sources, aliases, object members, local function calls, selected stdin callbacks, JSON/string transformations, and known file writes/reads. Object rest preserves member shapes: credential-bearing environment members do not taint an unrelated public timeout, while a secret-derived timeout remains tainted. Logging, summaries, outputs, and non-authentication request fields are disclosure sinks. Comparison/presence results and type-proven public length/index metadata remove values; arbitrary object properties and numeric coercions do not. Genuine escaped, single-record stdout `add-mask` registrations are recognized without sanitizing later uses or captured stdout. Dynamic reads remain explicit gaps rather than invented all-environment return values.

`--javascript-root` bounds source reads to the supplied repository, including symlink confinement. Repeated `--javascript-runtime-path 'runner-directory=repository-relative-directory'` flags can map generated runtime asset paths to **operator-verified matching source**. For example, only when the workflow uses the exact checkout's gh-aw setup assets, pass `--javascript-runtime-path '${RUNNER_TEMP}/gh-aw/actions=actions/setup/js'`. This mapping loads and analyzes source; it does not declare any helper safe. Unknown interpreter paths are not identified by basename.

Missing or incompatible compiler installations, unresolved sources/calls/modules, cycles, unsupported syntax, loop convergence, and traversal/budget limits retain incomplete-analysis warnings, visible with `--debug`. Unknown boundaries do not invent transformed stdout or confirmed leaks. Without `--javascript-module`, Bash analysis remains active and records unavailable JavaScript analysis at secret-bearing Node command boundaries; additional inline/action JavaScript inspection is opt-in. Configured analysis can substantially expand diagnostics and report infeasible source-flow paths; findings require review and are not a vulnerability count. An empty errors-only report does not imply complete coverage or whole-program JavaScript safety.

## Linter performance

`DEBUG=workflow:lint:perf` emits per-rule start/completion markers, elapsed nanoseconds, diagnostic counts, and failure status to stderr. Separate markers measure diagnostic sorting/deduplication and total runner time. These markers contain no AST contents, source symbols, or error text and do not change JSON diagnostics or enable coverage warnings:

```bash wrap
DEBUG=workflow:lint:perf DEBUG_COLORS=0 gh aw lint --rules= --json \
  > lint.json 2> lint-performance.log
```

Go execution traces include `workflow:lint`, `workflow:lint:rule:<id>`, and `workflow:lint:diagnostics` regions. Native JavaScript compiler time is included in the calling rule's wall time; the external compiler's CPU work is not part of a Go CPU profile.
