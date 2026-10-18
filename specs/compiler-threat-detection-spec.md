---
title: GitHub Actions Compiler Threat Detection Specification
description: Normative requirements for compiler rules that prevent unsafe generated workflows
sidebar:
  order: 1001
---

# GitHub Actions Compiler Threat Detection Specification

**Version**: 1.0.56
**Status**: Candidate Recommendation  
**Latest Version**: https://github.com/github/gh-aw/blob/main/specs/compiler-threat-detection-spec.md  
**Editors**: GitHub Next (GitHub, Inc.)

## Abstract

This specification is the source of truth for compiler-side threat detection in GitHub Agentic Workflows (gh-aw). A conforming implementation detects unsafe generated GitHub Actions behavior before runtime and keeps rule definitions, implementations, tests, and daily review evidence synchronized.

## Status

The gh-aw maintainers revise this Candidate Recommendation using security-review and conformance evidence. Requirement keywords use [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

## 1. Scope and Conformance

This specification covers `pkg/workflow/`, related `pkg/parser/` and `actions/setup/` validation, compiler-generated safe-output publishing identity, the host-side MCP compilation boundary in `pkg/cli/`, and the daily optimizer. It excludes runtime detection-job internals, external scanner ecosystems, and non-compiler repositories.

A conforming implementation satisfies every MUST in Sections 2–8. Rules are specification-first, secure by default, mapped bidirectionally to implementation, and auditable through version history.

## 2. Spec-to-Implementation Sync

Each version maps to the minimum compatible binary. A version change MUST update this table and any lock-file compatibility change in the same pull request.

| Versions | Minimum gh-aw | Compatibility |
|---|---:|---|
| `1.0.56` | Unreleased | Adds explicit permission-scope/OIDC AST checks (CTR-001 subset) and authored AW source checks for disabled version integrity, conditional imports, and private-to-public opt-outs (CTR-018/020/031). These do not infer job authorization, imported policy, or missing source metadata. No new `CTR-*` rule or lock-file schema change. |
| `1.0.55` | Unreleased | Maps independent Actions/AW AST lint checks to observable subsets of CTR-002, CTR-006, CTR-017, CTR-021, and CTR-026. Existing compiler and runtime conformance requirements remain authoritative; no new `CTR-*` rule or lock-file schema change. |
| `1.0.54` | Unreleased | Specifies the host-side MCP compilation capability boundary: a dedicated read-only mode without startup write-tool opt-in, validation independent of stale outputs, unchanged filesystem/Docker state, cached-only remote imports and nested includes, and repository-root cache lookup. Preserves trusted CLI no-emit import population. Adds bounded TLA+ evidence; no new `CTR-*` rule or lock-file schema change. |
| `1.0.53` | Unreleased | Adds CTR-034: formalizes pull-request fork admission by repository pattern or explicit author association, retaining same-repository defaults and separating admission from permissions and secrets. Clarifies that same-repository bot checkout on `pull_request` and `pull_request_target` `ready_for_review` and `reopened` events is a runtime trust control governed by Security Architecture Specification RS-05a, not a new compiler threat rule. |
| `1.0.52` | Unreleased | Adds CTR-033: repository-list checkout requires literal `multi: true`; ordinary repository expressions retain single-checkout semantics with runtime cardinality validation in both jobs. Safe Outputs PTI-001–007 and its TLA+ model clarify a runtime patch/bundle validation-to-application integrity boundary without changing permissions or lock-file schema. |
| `1.0.51` | Unreleased | Extends CTR-008 to reject `pull_request_target` with `on.roles` containing `read` or `all` in strict mode and permit suppression of only the corresponding warning in non-strict mode via an explicit dangerous feature flag. |
| `1.0.50` | Unreleased | Extends CTR-032 with mandatory creating-run traceability from first publication, HTML markers for issue comments and PR descriptions, and Markdown anchor markers for PR review comments. Versioned activation markers and review-comment persistence coverage remain implementation obligations. |
| `1.0.49` | Unreleased | Adds CTR-032; safe-output publishing jobs supply step-level check-run identity, and runtime publishers replace forged execution-provenance markers with trusted metadata. |
| `1.0.48` | Unreleased | Audit-only; same-repository checkout for allow-listed App comments and reviews is a runtime trust control governed by Security Architecture Specification RS-05a, not a new compiler threat rule. |
| `1.0.47` | Unreleased | Audit-only; same-repository checkout for allow-listed App comments is a runtime trust control governed by Security Architecture Specification RS-05a, not a new compiler threat rule. |
| `1.0.46` | Unreleased | Extends CTR-017 OTLP sandbox exclusions to evals AWF invocations. No lock-file schema change. |
| `1.0.45` | Unreleased | Extends CTR-017 with host-only OTLP exporter configuration: regenerated agent and threat-detection AWF commands exclude collector endpoints and headers when AWF supports `--exclude-env`. No lock-file schema change. |
| `1.0.44` | Unreleased | Clarifies CTR-001: `id-token: write` is exempt from repository-write rejection, and its OIDC trust-policy reminder is informational rather than a warning. No lock-file compatibility change. |
| `1.0.43` | `v0.87.9` | Clarifies CTR-008 trusted `pull_request_target` checkouts: acknowledgment suppresses only the trigger warning, allowlisted checkouts require an exact literal repository/ref pair, and any configured `fetch` is rejected. |
| `1.0.42` | `v0.87.9` | Audit-only; same-repository bot PR checkout authorization is a runtime trust control governed by the security architecture specification, not a new compiler threat rule. |
| `1.0.41` | `v0.87.9` | Adds CTR-031; private-to-public flow opt-outs are rejected in strict mode and warned in non-strict mode because they can expose private data through public logs or destinations. |
| `1.0.40` | `v0.87.9` | Adds CTR-030; activation sparse checkouts use the PR base SHA for pull-request and review events before loading runtime imports or skills. |
| `1.0.38` | `v0.87.9` | Audit-only; safe-outputs workspace checkout discovery is a runtime trust boundary governed by the safe-outputs and checkout-behavior specifications, not a new CTR rule. |
| `1.0.37` | `v0.87.9` | Adds CTR-028; PR-triggered agent jobs restore agent configuration from the base branch before any step that installs agent content. |
| `1.0.36` | `v0.87.9` | Audit-only; Opengrep build-reproducibility findings (non-deterministic `npm`/`uv pip` installs, non-SHA-pinned Dockerfile image) are out of conformance scope per Section 1. |
| `1.0.35` | `v0.87.9` | Audit-only; #681/#678/#676/#675, #679, #674/#669/#668/#667, #663, #657, #652/#651, and #680 are not new threat classes. |
| `1.0.34` | `v0.87.9` | Adds CTR-027; allowlisted bot synchronization requires trusted same-repository provenance. |
| `1.0.33` | `v0.87.9` | Audit-only; no new CTR rule or lock-file schema change. |
| `1.0.32` | `v0.87.9` | Audit-only; no new CTR rule or lock-file schema change. |
| `1.0.31` | `v0.87.9` | Audit-only; no new CTR rule or lock-file schema change. |
| `1.0.30` | `v0.87.9` | CTR-001 status-function guard mapping update only. |
| `1.0.29` | `v0.87.9` | CTR-004 Playwright renderer mapping update only. |
| `1.0.27`–`1.0.28` | `v0.87.4` | CTR-004 enclave and CTR-006 guard-renderer mapping updates only. |
| `1.0.26` | `v0.87.4` | Adds CTR-026; generated job timeouts are literal positive integers. |
| `1.0.23`–`1.0.25` | `v0.87.1` | Adds CTR-025; mapping-only audit updates otherwise. |
| `1.0.22` | `v0.87.0` | Adds suppression manifest and optimizer safeguards. |
| `1.0.15`–`1.0.21` | `v0.72.1`–`v0.83.6` | Adds CTR-021–023 and editorial/mapping updates. |
| `1.0.8`–`1.0.14` | `v0.72.1` | Establishes CTR-016, CTR-018–020; earlier versions establish CTR-001–015. |

## 3. Threat Model Overview

Generated workflows run with elevated permissions and consume untrusted content (issues, PRs, comments, external tool output). The catalog in Section 5 groups the threats a conforming compiler MUST detect into five classes: unauthorized privilege or scope expansion, unsafe or bypassed sandboxing, injection (template, shell, or subprocess argument), unsafe output and supply-chain routes (including forged execution attribution), and compile-time drift between manifests, mappings, and the rules that reference them. Each class maps to one or more `CTR-*` rules with a stable trigger and compiler action.

Runtime trust boundaries inside the agent workspace are out of scope for this specification. In particular, the safe-outputs repository checkout discovery path (`actions/setup/js/find_repo_checkout.cjs`) resolves an `owner/repo` target from agent-writable workspace state; its normative controls — manifest precedence, workspace confinement, per-invocation `safe.directory` scoping, read-only discovery, remote host constraint, and deferred durable trust — are specified as Threat T7 and requirements RCR1–RCR7 in the Safe Outputs MCP Gateway Specification and as §3.5 of the Checkout Behavior Specification. A conforming compiler is not required to detect this class, because the compiler emits no generated-workflow construct that selects the checkout directory.

PR-branch checkout authorization in `actions/setup/js/checkout_pr_branch.cjs` is also a runtime trust control, not a compiler-detectable threat. The same-repository bot exception covers `pull_request` and `pull_request_target` `opened`, `synchronize`, `ready_for_review`, and `reopened` events with matching bot sender and actor plus equal positive runtime, PR head, and PR base repository IDs. For `issue_comment`, `pull_request_review_comment`, and `pull_request_review`, an explicitly `on.bots`-allow-listed App may bypass the collaborator check only when `sender.type` is `Bot`, the sender and matching comment or review author logins match exactly, the sender is canonically equivalent to the runtime actor (App slug or `[bot]` form), the runtime repository ID is positive, and the fetched PR head and base repository IDs both match it. Forks, unlisted bots, unverifiable or mismatched identities, and other actors retain the collaborator-permission check. The normative requirements are in the Security Architecture Specification, RS-05a; this control does not add a `CTR-*` rule.

Patch/bundle replacement between safe-output validation and Git application is likewise a runtime boundary, not a new compiler threat rule. Safe Outputs [PTI-001–007](../docs/src/content/docs/specs/safe-outputs-specification.md#patch-and-bundle-transport-integrity-requirements) require canonical capture, nonblocking regular-file checks, private snapshot consumption, full byte checks at application/publication boundaries, permanent integrity-error classification, registered local cleanup, and safe diagnostics. The implementation is `actions/setup/js/transport_integrity.cjs`, used by both PR handlers and their publication callbacks; evidence is mapped in the safe-output specification and [`SafeOutputTransport.tla`](workflow-security/SafeOutputTransport.tla). Artifact run/invocation provenance, schema validation, detector gating, and CTR-005 safe-output routing remain independent requirements: none proves that a mutable artifact still contains the bytes validated earlier. This amendment changes no generated workflow construct, GitHub App permission, suppression rule, or `CTR-*` identifier.

Sandbox bypass includes the provenance of the agent's own configuration. On pull-request triggers the workspace holds head-branch content that the PR author controls, so agent configuration folders and root instruction files are attacker-controlled inputs until the generated job replaces them with the base-branch snapshot captured before that content was checked out.

Activation also reads workflow imports and skills before the agent job runs. Restoring agent configuration later cannot make instructions already loaded during activation trustworthy; the activation sparse checkout must select a trusted ref independently.

Host-side OTLP exporter configuration is another secret-bearing trust boundary. Passing collector headers and endpoints through AWF's `--env-all` exposes collector credentials to sandboxed processes and can cause an agent CLI to export duplicate telemetry. CTR-017 covers compiler-generated exclusions for the agent, threat-detection, and evals sandboxes.

Safe-output body content is untrusted even when supplied to a trusted publisher. A body can contain forged reserved HTML comments, including nested, variant, or unterminated openers, or Markdown anchor markers that imitate execution provenance. Attribution can also be lost when a publisher omits markers from initial WIP bodies or uses HTML comments in PR review comments, where GitHub's API filters them out. CTR-032 requires the compiler to provide the publishing check-run identity and the runtime publisher to remove or neutralize supplied reserved markers before appending trusted execution metadata in a format retained by the destination API; missing or invalid identity fails closed.

Repository-selection expressions are runtime data, not an implicit request for repository-list expansion. Inferring multi mode from an expression or an allowlist changes authentication, visibility, path, and secondary-fetch behavior unexpectedly. CTR-033 requires explicit mode classification and generated single-selector validation. The multi-mode preflight and privileged-consumer boundary are specified by checkout behavior §3.1 and safe outputs RCR8–RCR15 and modeled in `specs/workflow-security/RuntimeRepositoryCheckout.tla`; the model does not prove parser, filesystem, or Git correctness.

An MCP caller can request host-side compilation of untrusted workflow content without being authorized to change compiler-managed files. Actor permissions and server write-tool capability are separate: an actor with repository write access cannot enable writing through a tool call. Errors, external scanner preparation, automatic fixes, update checks, and import-cache population can otherwise turn nominal validation into a write path. Section 5.2.1 specifies this boundary independently of the generated-workflow `CTR-*` catalog; [`MCPCompile.tla`](workflow-security/MCPCompile.tla) models the bounded policy and operation ordering, not a filesystem sandbox or a proof of compiler refinement.

## 4. Governance and Responsibilities

Maintainers own the rule catalog, its implementation mapping, and its test coverage; the daily optimizer (Section 6) reviews new candidate threats and reports coverage gaps. Every rule addition, deprecation, or mapping change MUST be reviewed against this specification in the same pull request that changes the implementation, per the Spec-to-Implementation Sync table in Section 2.

## 5. Rule Model and Requirements

Each rule has a stable `CTR-*` ID, threat class, trigger, compiler action, diagnostic, implementation mapping, and test.

### 5.1 Core Rule Catalog

- **CTR-001 Privilege Escalation**: Reject unauthorized generated-job write permissions. `id-token: write` MUST be exempt from repository-write rejection because it permits OIDC token issuance, not repository modification. Its audience and cloud-provider trust-policy reminder MUST be informational, MUST NOT increment the compiler warning count, and MUST NOT by itself fail warnings-as-errors validation. Missing required OIDC permissions and invalid `id-token: read` declarations MUST still be rejected.
- **CTR-002 Unpinned Action Integrity**: Reject unpinned action references in strict contexts.
- **CTR-003 Unsafe Tool Scope Expansion**: Reject or warn on policy-violating wildcard or overbroad tool scope.
- **CTR-004 Sandbox Bypass Configuration**: Reject generated configuration that disables required sandboxing.
- **CTR-005 Unsafe Output Route**: Reject direct write paths that bypass safe outputs.
- **CTR-006 Template Injection**: Reject user-controlled expressions directly embedded in shell commands.
- **CTR-007 Markdown Content Security**: Detect unsafe external markdown, including obfuscation, scripts, and social engineering.
- **CTR-008 Pull Request Target Safety**: For `pull_request_target`, strict mode MUST warn about elevated permissions and secret access unless `on.pull_request_target.acknowledge-risk` is `true`. This acknowledgment suppresses only that warning; it MUST NOT authorize a checkout or suppress checkout diagnostics. If `on.roles` is `all` or `read`, or a role list contains either value, the compiler MUST reject the workflow in strict mode regardless of checkout configuration, trigger-risk acknowledgment, or feature flags. In non-strict mode, this role combination MUST produce a security warning unless `features.dangerously-allow-roles-read-and-pull-request-target` is explicitly set to boolean `true`. This feature flag suppresses only the role-combination warning; it MUST NOT affect strict-mode rejection or other `pull_request_target` trigger and checkout diagnostics. An explicit `checkout: false` is accepted. Otherwise, every configured checkout MUST independently use either the supported base-repository/base-ref form or an exact, case-sensitive literal repository/ref pair in `on.pull_request_target.allowed-checkouts`. The allowlist path MUST reject expressions, wildcard refs, `refs/pull/...` refs, wiki checkouts, and omitted external refs. Neither the base-checkout nor allowlist path may accept any non-empty `fetch` value. An omitted checkout configuration is not equivalent to explicit `checkout: false`. In strict mode an unsafe checkout MUST be rejected; in non-strict mode it MUST produce a warning.
- **CTR-009 Shell Expansion in Safe-Outputs**: Reject dangerous shell expansion in safe-output scripts.
- **CTR-010 Expression Safety Allowlist**: Reject unauthorized or multiline GitHub Actions expressions.
- **CTR-011 Network Firewall Configuration**: Reject missing firewall prerequisites and strict-mode wildcard domains.
- **CTR-012 Safe-Outputs Wildcard Push Scope**: Warn for unconstrained wildcard PR-branch pushes.
- **CTR-013 Argument Injection via Package/Image Names**: Reject hyphen-prefixed package and image names before subprocess use.
- **CTR-014 Supply Chain Attack via Install Scripts**: Warn, or reject in strict mode, when Node install scripts are enabled.
- **CTR-015 Allowed Label Glob Scope**: Reject bare `*` safe-output allowed-label patterns.
- **CTR-016 Compile-Time Manifest Drift**: Reject new restricted secrets or action references absent from an existing manifest.
- **CTR-017 Secret Leakage via Environment Variables**: Warn, or reject in strict mode, for uncontrolled secret-expression placement. When OTLP is enabled and the effective AWF version supports `--exclude-env`, generated agent, threat-detection, and evals AWF invocations using `--env-all` MUST exclude `OTEL_EXPORTER_OTLP_ENDPOINT`, `OTEL_EXPORTER_OTLP_HEADERS`, and `GH_AW_OTLP_ENDPOINTS`. These values MUST remain available to host-side exporters. Both inline and external threat-detection paths and evals execution MUST preserve the exclusions. Workflows without OTLP MUST NOT gain these OTLP-specific exclusions unless independently required by another exclusion policy.
- **CTR-018 Version Integrity Bypass**: Warn, or reject in strict mode, for `check-for-updates: false`.
- **CTR-019 Cache-Memory Integrity Enforcement**: Require cache updates only after successful agent and threat-detection jobs.
- **CTR-020 Conditional Import Security**: Reject `imports` entries containing `if`.
- **CTR-021 Workflow Run Trigger Branch Scope**: Warn, or reject in strict mode, for unscoped `workflow_run`; always reject missing `workflows`.
- **CTR-022 Git Subprocess Argument Injection**: Reject unsafe remote ref/path arguments before invoking Git.
- **CTR-023 Bash Command Allowlist Illusion**: Reject explicit bash restrictions for engines that cannot enforce them.
- **CTR-025 Framework Self-Prompt Misattribution**: Strip only a leading framework `<system>` block before analysis.
- **CTR-026 Generated Job Timeout Expression Injection**: Reject non-positive or expression job timeout values.
- **CTR-027 Allowlisted Bot Synchronization Provenance**: Deny bot-driven PR synchronization when the actor differs from the PR author unless the bot is explicitly allowlisted and active, the PR is from the base repository, the PR author satisfies the configured roles, and the actor is not Dependabot.
- **CTR-028 Agent Configuration Restore Provenance**: For pull-request triggers, emit the base-branch restore of every engine's agent configuration folders and root instruction files after the PR checkout and before any generated step that installs agent content or executes the agent.
- **CTR-029 Repository Dispatch App Authorization**: On `repository_dispatch`, authorize a GitHub App that is explicitly listed in `on.bots` when the collaborator lookup conclusively reports it absent; deny non-allowlisted actors, lookup failures, and App actors on every other trigger.
- **CTR-030 Activation Checkout Provenance**: For `pull_request`, `pull_request_review`, and `pull_request_review_comment` events, emit an activation sparse checkout pinned to `github.event.pull_request.base.sha`, guarded by event name and pull-request payload presence. Match trigger names exactly so `pull_request_target` does not opt into this rule. Other events retain their existing checkout ref (including the resolved callee SHA for `workflow_call`); cross-repository `workflow_call` checkout remains same-repository-only when activation authentication can fall back to `GITHUB_TOKEN`.
- **CTR-031 Private-to-Public Flow Opt-Out**: Reject `tools.github.private-to-public-flows` in strict mode in both its blanket and server-list forms; in non-strict mode, emit a security warning because the opt-out can expose private data through public Actions logs or agent-controlled public destinations.
- **CTR-032 Safe-Output Execution Provenance Spoofing**: Generated built-in, custom, maintenance, and activation-time publishers MUST receive `job.check_run_id` only in the publishing step. Before publishing issue, pull request, comment, or discussion bodies, runtime publishers MUST replace supplied `gh-aw-execution` and `gh-aw-agentic-workflow` markers—including nested, variant, and unterminated comment forms and Markdown anchor forms—with trusted versioned execution metadata. Every issue comment, pull request description, and pull request review comment MUST remain traceable to the workflow run ID that created it, using the destination-specific marker contract in Section 5.1.1. Missing or invalid provenance metadata MUST fail closed.
- **CTR-033 Explicit Checkout Expansion**: Only literal `checkout.multi: true` MUST enable repository-list expansion. Non-boolean mode values MUST fail compilation. With multi omitted or false, selectors MUST retain ordinary single-checkout behavior without list-only restrictions; literal lists MUST fail compilation, and generated agent and safe-output checkout steps MUST validate expression values as exactly one `owner/repo` slug before `actions/checkout`. Empty, malformed, wildcard, and multiple values, including duplicate slugs, MUST fail closed. Expressions MUST be passed through environment data, not executable script text. With multi enabled, the compiler MUST require a trusted scoped allowlist and literal non-root prefix and reject LFS, enabled submodules, `current`, `wiki`, and implicitly scoped Apps as specified by checkout behavior §3.1. Entries MUST NOT merge across modes, and neither mode MUST widen safe-output write authorization.
- **CTR-034 Pull Request Fork Admission**: For `pull_request`, the compiler MUST always admit same-repository PRs. If `on.pull_request.forks` is omitted or an empty pattern list, fork PRs MUST be denied. String/list repository patterns MUST admit matching forks, and `*` MUST admit all forks. An `author-association` list MUST admit a fork PR when `github.event.pull_request.author_association` is a member of that explicit list; no association is implicit, and an empty list MUST admit no forks. The compiler MUST validate association values against GitHub's uppercase author-association enum: `OWNER`, `MEMBER`, `COLLABORATOR`, `CONTRIBUTOR`, `FIRST_TIMER`, `FIRST_TIME_CONTRIBUTOR`, `MANNEQUIN`, and `NONE`. The filter MUST NOT change GitHub Actions permissions, secret availability, or other trust controls, and MUST NOT filter non-`pull_request` events.

### 5.1.1 Creating-Run Marker Contract (CTR-032)

The compiler MUST supply trusted runtime identity to every publisher of issue comments, pull request descriptions, and pull request review comments. The persisted body MUST contain a machine-readable marker with the creating run's positive integer `GITHUB_RUN_ID` as `run_id` and its repository-qualified Actions URL as `run_url`. Cross-repository outputs MUST identify the originating workflow run, not a run in the destination repository. Run identity MUST come from the trusted publishing environment, never from agent-supplied body text.

| Destination | Required marker syntax |
|---|---|
| Issue comment or pull request description | HTML comment: `<!-- gh-aw-execution: <versioned execution metadata JSON> -->` |
| Pull request review comment, including a reply | Markdown anchor/reference definition: `[gh-aw-execution]: # "<percent-encoded versioned execution metadata JSON>"` |

In the table, angle-bracketed text denotes the serialized metadata payload, not literal marker content. The Markdown marker MUST encode the same versioned metadata as the HTML marker. It MUST occupy its own line, and its payload MUST be percent-encoded so quotes, newlines, or delimiters cannot alter the reference definition. Consumers MUST decode and validate the payload before using it. Publishers MUST NOT rely on HTML comments for review-comment provenance because GitHub's API filters them out.

Markers MUST be present on the first API publication, including activation-time WIP resources, fallback issue comments, individually created or buffered review comments, and review-comment replies. A marker on an enclosing review body does not satisfy the requirement for each individual review comment. Custom, disabled, or empty visible footers MUST NOT suppress the marker. Publishers MUST remove or neutralize forged reserved markers in both HTML and Markdown forms before appending trusted metadata; a visible generated-by link alone is not a substitute.

Conformance tests MUST simulate destination API filtering and inspect the stored or returned body, not only the outgoing request. Each covered item MUST retain a decodable marker whose `run_id` and `run_url` match the creating run, including cross-repository, footer-disabled, initial-WIP, reply, buffered-review, and forged-marker cases.

### 5.2 Compiler Response Requirements

The compiler MUST produce deterministic, actionable diagnostics and either reject insecure generation or apply a safe rewrite for every active rule in Section 5.1.

### 5.2.1 Host-Side MCP Compilation Boundary

These requirements apply to the MCP `compile` invocation, not arbitrary CLI flag combinations or server-startup bookkeeping.

| Requirement | Required behavior |
|---|---|
| **MCP-COMPILE-001 Startup capability** | Only server startup with `--allow-write-tools` MAY enable writing tools. Tool arguments or actor permissions MUST NOT enable that capability. Without it, `compile` MUST remain available as a read-only tool, advertise `ReadOnlyHint: true`, and bypass write-actor authorization; `add`, `update`, and `fix` MUST remain disabled. With it, those tools and write-enabled `compile` MUST retain configured actor validation and its repository write-permission floor. |
| **MCP-COMPILE-002 Read-only invocation** | Without startup write capability, `compile` MUST automatically select the dedicated read-only compiler mode and pass `--no-emit --no-check-update`. It MUST reject `fix`, `dry_run`, and external scanner options (`zizmor`, `poutine`, `actionlint`, `runner-guard`, `syft`, `grype`, `grant`, `yamllint`) before starting the compile subprocess or scanner preparation. `dry_run` MUST NOT substitute for `--no-emit`, because development compilation emits files. |
| **MCP-COMPILE-003 Filesystem preservation** | Read-only compilation MUST NOT create, modify, or delete compiler-managed files on success or failure. This includes workflow sources, `.lock.yml` outputs, `.invalid.yml` diagnostics, action-pin caches, `.gitattributes`, import-cache entries, and temporary remote-include files outside the repository. It MUST skip writing post-processing and diagnostic emission rather than writing and restoring files, including when outputs already exist. Container validation MUST NOT pull Docker images; it MUST use local inspection and report a missing required local image without modifying the image store. |
| **MCP-COMPILE-004 Cached imports** | Read-only compilation MUST reuse available remote-import cache entries without rewriting them. An uncached remote import MUST produce an actionable error before download, cache insertion, or temporary-file fallback. This MUST cover frontmatter imports, body-level includes, engine-aware and manifest-aware expansion, and nested includes. Shared cache initialization MUST prefer the Git repository root over the process working directory, including when launched from a repository subdirectory; outside Git it MUST retain the working-directory fallback (`.` if that directory cannot be obtained). Import expansion and hashing MUST share that cache. |
| **MCP-COMPILE-005 Validation preservation** | Read-only compilation MUST preserve validation of the newly generated YAML and return compilation errors without emitting diagnostic files; `--no-emit` MUST NOT convert invalid input into a successful result. No-emit validation MUST NOT consume stale on-disk lock YAML as the generated output. Tests MUST cover valid and invalid inputs, existing and absent outputs (including malformed stale locks), explicit-file and directory compilation, and cached imports with a subdirectory launch. |
| **MCP-COMPILE-006 Write-enabled compatibility** | Explicitly write-enabled compilation MUST retain normal output emission after authorization. The stronger MCP read-only mode MUST remain separate from ordinary CLI no-emit compilation, so trusted `validate`, maintenance, and label-discovery callers can populate imports. Read-only admission MUST NOT weaken authorization for write-enabled compilation or the other writing tools. Neither mode grants additional GitHub permissions. |

This contract concerns compilation-controlled filesystem and Docker image-store effects, not arbitrary external programs, concurrent filesystem changes, or all process-level temporary files. In particular, MCP server-startup manifest baselines remain independent bookkeeping. Read-only mode does not prohibit network reads to resolve branch/tag refs. The bounded model abstracts file operations as mutation events; real-binary filesystem snapshots, isolated temporary directories, and fake-Docker tests supply the concrete evidence.

### 5.3 Candidacy and Lifecycle

When a threat is found, maintainers MUST add its mapping and test if covered, or implement detection, tests, and documentation if not. Experimental threats MUST NOT fail production compilation. Candidates need a trigger, action, stable diagnostic, test, deployment evidence, and security-maintainer review before becoming normative.

### 5.4 Deprecation Policy

Removing a rule dependency MUST deprecate—not delete—the catalog and mapping rows in the same change set. The catalog records the version and reason; mapped tests become `[DEPRECATED]`; the mapping implementation cell is cleared; and the changelog records the retirement. `TestFormal_DeprecationPolicy_SpecArtifactsConform` enforces this policy.

## 6. Daily Optimizer Protocol

The daily optimizer MUST review recent compiler changes, related validation paths, open/recent security findings, and this catalog. For each candidate threat, it MUST determine coverage, update the mapping/tests if covered, or implement, test, and document remediation if uncovered. Its output MUST be either a pull request or an explicit noop report.

### 6.1 Suppressions

`threat-detection-suppress` entries MUST provide a rule and non-empty reason; `expires` is optional ISO 8601. Active entries MUST retain rule, reason, and expiry in the lock-file manifest. Expired entries do not suppress a rule.

The optimizer SHOULD resolve false positives affecting non-strict rejection controls within 10 business days. It MUST report older suppressions as `SLA_BREACH` with rule, reason, age, owner, and expiry, and MUST create a follow-up action after 20 business days. The mechanically-verified norms for suppression handling are cataloged as `T-CTR-024`–`T-CTR-029` in `specs/compiler-threat-detection-compliance/README.md` (Section 6.4 False-Positive Handling Norms).

### 6.2 Failure Safeguards

| Failure | Required behavior |
|---|---|
| API unavailable | Retry with bounded exponential backoff (10 seconds to 5 minutes, three attempts); then emit `OPTIMIZER_DEGRADED` with endpoint, error, and UTC time. Do not create artifacts from incomplete data. |
| Timeout | Emit `OPTIMIZER_TIMEOUT` with completed step and unevaluated rules; discard partial artifacts. Use an explicit timeout and request same-day retry. |
| Rate limit | Apply `RATE_LIMIT_RETRY_CONFIG`; after exhaustion emit `OPTIMIZER_RATE_LIMITED` with endpoint and retry metadata. Count neither completion nor noop; retry next window. |
| Missed schedule | Emit `OPTIMIZER_MISSED_CRON` with scheduled and detected times and lookback; do not count completion and create a follow-up action. |

The mechanically-verified norms for these safeguards are cataloged as `T-CTR-030`–`T-CTR-038`, `T-CTR-040` in `specs/compiler-threat-detection-compliance/README.md` (Section 6.6 Optimizer Failure Safeguards).

## 7. Implementation Mapping

Every active rule MUST map to implementation and test coverage. References are patterns and MUST be verified against concrete paths whenever changed.

### 7.1 Baseline Rule Mapping

| Rule ID | Primary Implementation Areas | Test Coverage Targets |
|---------|------------------------------|-----------------------|
| CTR-001 Privilege Escalation | `pkg/workflow/*permissions*validation*.go`, `permissions_compiler_validator.go`, `compiler_builtin_job_augmentation.go` | `*permissions*_test.go`, `compiler_custom_jobs_test.go`, `idtoken_write_info_test.go`, `compiler_threat_detection_formal_test.go`, `pkg/cli/compile_development_test.go` |
| CTR-002 Unpinned Action Integrity | `pkg/workflow/*action*.go`, `tools_validation*.go`, strict-mode validation; AST subset: `pkg/workflow/lint/integrity.go` | `*action*_test.go`, `*tools*_test.go`, `pkg/workflow/lint/integrity_test.go` |
| CTR-003 Unsafe Tool Scope Expansion | `pkg/workflow/*action*.go`, `tools_validation*.go`, strict-mode validation | `*action*_test.go`, `*tools*_test.go` |
| CTR-004 Sandbox Bypass Configuration | sandbox validation, `enclaves.go`, `enclave_github_proxy.go` | sandbox, enclave, and proxy tests |
| CTR-005 Unsafe Output Route | safe-output compiler/validation; setup safe-output and manifest helpers | safe-output and setup helper tests |
| CTR-006 Template Injection | template/heredoc validation, `mcp_renderer_guard.go`; AST subset: `pkg/workflow/lint/template_injection.go` | template-injection and MCP tests; `pkg/workflow/lint/template_injection_test.go` |
| CTR-007 Markdown Content Security | `markdown_security_scanner.go`, `pull_request_target_validation.go` | corresponding workflow and sanitizer tests |
| CTR-008 Pull Request Target Safety | `pkg/workflow/pull_request_target_validation.go` | `pkg/workflow/pull_request_target_validation_test.go`; `specs/workflow-security/PullRequestTargetCheckout.tla` |
| CTR-009 Shell Expansion in Safe-Outputs | safe-output shell/push validation, expression validation, firewall validation | corresponding workflow tests |
| CTR-010 Expression Safety Allowlist | safe-output shell/push validation, expression validation, firewall validation | corresponding workflow tests |
| CTR-011 Network Firewall Configuration | safe-output shell/push validation, expression validation, firewall validation | corresponding workflow tests |
| CTR-012 Safe-Outputs Wildcard Push Scope | safe-output shell/push validation, expression validation, firewall validation | corresponding workflow tests |
| CTR-013 Argument Injection via Package/Image Names | name, install-script, and allowed-label validation | `argument_injection_test.go`, corresponding validation tests |
| CTR-014 Supply Chain Attack via Install Scripts | name, install-script, and allowed-label validation | `argument_injection_test.go`, corresponding validation tests |
| CTR-015 Allowed Label Glob Scope | name, install-script, and allowed-label validation | `argument_injection_test.go`, corresponding validation tests |
| CTR-016 Compile-Time Manifest Drift | safe-update, strict env/update, cache, and expression builder | corresponding enforcement, secrets, update, and cache tests |
| CTR-017 Secret Leakage via Environment Variables | safe-update, strict env/update, cache, and expression builder; `pkg/workflow/awf_command_builder.go`, `threat_detection_external.go`, `evals_steps.go`; AST subset: `pkg/workflow/lint/secrets.go`, `pkg/workflow/taint/` | corresponding enforcement, secrets, update, and cache tests; `TestOTLPEnvExcludedFromAWFSandboxes` in `awf_env_test.go`, `TestEvalsOTLPEnvExcludedFromSandbox` in `evals_steps_test.go`, `TestCompiledOTLPEnvExcludedFromAgentDetectionAndEvals` in `threat_detection_isolation_test.go`; `pkg/workflow/taint/taint_test.go` |
| CTR-018 Version Integrity Bypass | safe-update, strict env/update, cache, and expression builder | corresponding enforcement, secrets, update, and cache tests |
| CTR-019 Cache-Memory Integrity Enforcement | safe-update, strict env/update, cache, and expression builder | corresponding enforcement, secrets, update, and cache tests |
| CTR-020 Conditional Import Security | `pkg/parser/import_bfs.go` | `pkg/parser/import_bfs_test.go` |
| CTR-021 Workflow Run Trigger Branch Scope | `agent_validation.go`, `agentic_engine.go`, `pkg/gitutil/gitutil.go`; AST subset: `pkg/workflow/lint/workflow_run_scope.go` | workflow-run, bash-allowlist, gitutil, and download tests; `pkg/workflow/lint/workflow_run_scope_test.go` |
| CTR-022 Git Subprocess Argument Injection | `agent_validation.go`, `agentic_engine.go`, `pkg/gitutil/gitutil.go` | workflow-run, bash-allowlist, gitutil, and download tests |
| CTR-023 Bash Command Allowlist Illusion | `agent_validation.go`, `agentic_engine.go`, `pkg/gitutil/gitutil.go` | workflow-run, bash-allowlist, gitutil, and download tests |
| CTR-025 Framework Self-Prompt Misattribution | `actions/setup/js/setup_threat_detection.cjs` | `setup_threat_detection.test.cjs` |
| CTR-026 Generated Job Timeout Expression Injection | custom-job properties and timeout resolution; AST subset: `pkg/workflow/lint/integrity.go` | custom-job and timeout tests; `pkg/workflow/lint/integrity_test.go` |
| CTR-027 Allowlisted Bot Synchronization Provenance | `actions/setup/js/check_membership.cjs`, `actions/setup/js/check_permissions_utils.cjs` | `actions/setup/js/check_membership.test.cjs`, `actions/setup/js/check_permissions_utils.test.cjs` |
| CTR-028 Agent Configuration Restore Provenance | `pkg/workflow/pr.go`, `actions/setup/sh/save_base_github_folders.sh`, `actions/setup/sh/restore_base_github_folders.sh` | `pkg/workflow/compiler_pre_agent_steps_test.go`, `actions/setup/sh/restore_base_github_folders_test.sh` |
| CTR-029 Repository Dispatch App Authorization | `actions/setup/js/check_membership.cjs`, `actions/setup/js/check_permissions_utils.cjs` | `actions/setup/js/check_membership.test.cjs` |
| CTR-030 Activation Checkout Provenance | `pkg/workflow/compiler_activation_job.go` | `pkg/workflow/compiler_activation_job_test.go` |
| CTR-031 Private-to-Public Flow Opt-Out | `pkg/workflow/strict_mode_private_to_public_flows_validation.go`, `pkg/workflow/compiler_validators.go` | `pkg/workflow/private_to_public_flows_test.go` |
| CTR-032 Safe-Output Execution Provenance Spoofing | `pkg/workflow/compiler_safe_outputs_steps.go`, custom and maintenance safe-output job builders, `pkg/workflow/compiler_steering_issue.go`, `actions/setup/js/execution_provenance.cjs`, issue/PR publishers, `actions/setup/js/create_pr_review_comment.cjs`, `actions/setup/js/pr_review_buffer.cjs`, `actions/setup/js/reply_to_pr_review_comment.cjs` | `pkg/workflow/safe_outputs_job_identity_test.go`, `pkg/workflow/compiler_steering_issue_test.go`, `actions/setup/js/execution_provenance.test.cjs`, WIP and review-comment handler tests; destination-filtering coverage for Section 5.1.1 remains required |
| CTR-033 Explicit Checkout Expansion | `pkg/workflow/checkout_config_parser.go`, `pkg/workflow/checkout_manager.go`, `pkg/workflow/checkout_step_generator.go`, `actions/setup/js/checkout_dynamic.cjs` | `pkg/workflow/checkout_multi_test.go`, checkout dynamic/policy/private/safe-output tests, `actions/setup/js/checkout_dynamic.test.cjs`, `specs/workflow-security/RuntimeRepositoryCheckout.tla` |
| CTR-034 Pull Request Fork Admission | `pkg/workflow/filters.go`, `pkg/parser/schemas/main_workflow_schema.json` | `pkg/workflow/filters_forks_test.go`, `pkg/parser/schema_forks_test.go`, `specs/workflow-security/PullRequestForkAdmission.tla` |

### 7.2 Mapping Audit History

Dated mapping-audit records are maintained in `specs/compiler-threat-detection-changelog.md`. A daily optimizer run MUST append its audit entry to that changelog, not to this specification.

### 7.3 Host-Side MCP Compilation Mapping

The `MCP-COMPILE-*` requirements are separate from the `CTR-*` / `T-CTR-*` generated-workflow catalogs.

| Requirements | Implementation | Concrete evidence and bounded invariants |
|---|---|---|
| MCP-COMPILE-001–002 | `pkg/cli/mcp_server.go`, `mcp_write_tools.go`, `mcp_tools_readonly.go` | `pkg/cli/mcp_write_tools_test.go`; `MCPCompile.tla`: `StartupCapability`, `ActorAuthorization`, `WritingOptionsGated`, `ReadOnlyMode`, `NoExternalScanners`, `DeniedBeforeExecution`. |
| MCP-COMPILE-003, MCP-COMPILE-005 | `pkg/cli/compile_pipeline.go`, `compile_validation.go`, `pkg/workflow/compiler.go`, `docker_validation.go` | `TestMCPServer_ReadOnlyCompilePreservesFiles` in `pkg/cli/mcp_server_compile_test.go` snapshots inventory, file bytes, modes, and modification times; `pkg/cli/compile_validation_no_emit_test.go`, `pkg/workflow/docker_validation_readonly_test.go`, `compiler_template_injection_both_paths_test.go`; model: `ReadOnlyIsolation`, `NoImagePulls`, `ValidationPreserved`, `ReadOnlyAvailability`. |
| MCP-COMPILE-004 | `pkg/parser/import_cache.go`, `remote_workflow_spec.go`, `include_expander.go`, `include_processor.go`, `pkg/workflow/compiler_mutators.go`, `compiler_yaml.go` | `pkg/parser/import_cache_test.go`, `include_readonly_test.go`, `pkg/workflow/compiler_shared_cache_test.go`, `TestCompilerSharedImportCacheRoot` in `pkg/workflow/compiler_mutators_test.go`, cached/uncached and subdirectory-launch cases in the real-binary MCP test; model: `NoRemoteDownloads`, `CacheRootConforms`, `ReadOnlyAvailability`. |
| MCP-COMPILE-006 | `pkg/cli/mcp_write_tools.go`, `mcp_tools_readonly.go`, compiler option and import-cache wiring | `pkg/cli/mcp_write_tools_test.go`, `TestMCPServer_CompileTool` in `pkg/cli/mcp_server_compile_test.go`, `pkg/workflow/compiler_shared_cache_test.go`, `pkg/parser/include_readonly_test.go`; model: `ActorAuthorization`, `WriteEnabledEmission`, `TrustedNoEmitAvailability`. |

`specs/workflow-security/check.py` MUST exhaust the secure MCP configuration and verify each configured synthetic negative control and reachability witness. These checks MUST NOT substitute for the concrete filesystem-preservation tests or be reported as confirmed product vulnerabilities.

### 7.4 Actions AST Lint Coverage

The in-process registry in `pkg/workflow/lint` supplements compiler validation by checking existing generated Actions ASTs, with an optional companion AW document. `gh aw lint` runs all registered checks by default. These checks do not replace the catalog's existing strict-mode validation, safe rewrites, runtime controls, or formal conformance evidence. They report only the subset observable in their input, using value-free diagnostics; a clean result MUST NOT be represented as proof of complete `CTR-*` conformance.

| Threat rule | AST lint rule | Observable invariant | Implementation and evidence |
|---|---|---|---|
| CTR-001 | `permission-scope` | Explicit declarations avoid blanket `write-all` grants and invalid OIDC `id-token: read`. Scoped writes and OIDC issuance are not automatically violations; authorization remains outside this check. | `pkg/workflow/lint/permission_scope.go`, `permission_scope_test.go` |
| CTR-002 | `action-pin` | Remote actions and reusable workflows use immutable commit references; Docker actions use immutable digests. Local references are not remote supply-chain pins. | `pkg/workflow/lint/integrity.go`, `integrity_test.go` |
| CTR-006 | `template-injection` | Known attacker-controlled expression strings are not interpolated directly into executable `run` text. Passing values through environment data is not direct template injection. | `pkg/workflow/lint/template_injection.go`, `template_injection_test.go` |
| CTR-017 | `secret-leak` | Symbolic secrets do not reach modeled disclosure sinks through expression, environment, output, shell, or file flows. Coverage warnings identify unmodeled paths. | `pkg/workflow/lint/secrets.go`, `pkg/workflow/taint/`, `pkg/workflow/taint/taint_test.go` |
| CTR-018 | `version-check` | An authored AW document does not explicitly disable version integrity checking. | `pkg/workflow/lint/source_policy.go`, `source_policy_test.go` |
| CTR-020 | `conditional-import` | Authored AW configuration import entries do not contain `if`; runtime prompt conditionals are unaffected. | `pkg/workflow/lint/source_policy.go`, `source_policy_test.go` |
| CTR-021 | `workflow-run-scope` | `workflow_run` selects workflows and has branch scope, with strict/non-strict severity policy. | `pkg/workflow/lint/workflow_run_scope.go`, `workflow_run_scope_test.go` |
| CTR-026 | `job-timeout` | A specified job timeout is a positive integer literal, not a runtime expression. Native Actions omission is not an invented requirement to specify a timeout. | `pkg/workflow/lint/integrity.go`, `integrity_test.go` |
| CTR-031 | `private-to-public` | Authored AW GitHub tool configuration does not opt out of private-to-public flow protection, in blanket or server-list form. | `pkg/workflow/lint/source_policy.go`, `source_policy_test.go` |

`pkg/workflow/lint/repository_workflows_test.go` applies the registered checks to repository lock files and companion AW documents. `pkg/cli/lint_workflow_test.go` exercises default execution and preserves diagnostics from independent rules even when another rule fails.

Source-only checks require a companion AW document and skip native Actions without that evidence. They inspect authored frontmatter only; they do not recursively load imports or reconstruct compiler/CLI policy overrides. Version-check and private-to-public findings follow authored strict policy; conditional imports are always errors. Blanket write grants follow the same authored strict policy, while invalid OIDC read grants are always errors.

Further AST candidates need additional policy or semantic evidence before becoming default rules. Full CTR-001 authorization requires job-purpose policy to distinguish agent privileges from intended safe-output writes, beyond the explicit declarations checked here. CTR-004/005/011/019 require trusted command identification and proof of sandbox, output-route, firewall, or success-gating semantics. CTR-008/028/030/033/034 require checkout or trigger provenance and guarded-expression reasoning, not textual substring matching. CTR-003/010/012-016/023 depend on source configuration or manifests that need not survive in Actions YAML. CTR-007/009/013/014/022/025/027/029/032 include content, shell/subprocess, or runtime obligations that an Actions AST alone cannot prove. In particular, CTR-032 marker persistence requires destination API evidence; host-side `MCP-COMPILE-*` filesystem and capability boundaries are not generated-workflow properties.

## 8. Compliance Testing

Each active rule MUST have at least one deterministic test that covers its primary trigger and stable diagnostic. Rule additions MUST add tests in the same change set; deprecated-rule tests become `[DEPRECATED]`.

### 8.1 Test ID Catalog

| Test ID | Rule | Detection Trigger | Expected Compiler Action | Stable Diagnostic ID |
|---------|------|--------------------|---------------------------|----------------------|
| **T-CTR-001** | CTR-001 Privilege Escalation | Unauthorized generated-job write permissions or an `id-token` declaration | Reject unauthorized repository writes, missing required OIDC permissions, and `id-token: read`; allow `id-token: write` with an informational reminder and no additional warning count. | `CTR-001` |
| **T-CTR-002** | CTR-002 Unpinned Action Integrity | Reject unpinned action references in strict contexts | Reject unpinned action references in strict contexts. | `CTR-002` |
| **T-CTR-003** | CTR-003 Unsafe Tool Scope Expansion | Reject or warn on policy-violating wildcard or overbroad tool scope | Reject or warn on policy-violating wildcard or overbroad tool scope. | `CTR-003` |
| **T-CTR-004** | CTR-004 Sandbox Bypass Configuration | Reject generated configuration that disables required sandboxing | Reject generated configuration that disables required sandboxing. | `CTR-004` |
| **T-CTR-005** | CTR-005 Unsafe Output Route | Reject direct write paths that bypass safe outputs | Reject direct write paths that bypass safe outputs. | `CTR-005` |
| **T-CTR-006** | CTR-006 Template Injection | Reject user-controlled expressions directly embedded in shell commands | Reject user-controlled expressions directly embedded in shell commands. | `CTR-006` |
| **T-CTR-007** | CTR-007 Markdown Content Security | Detect unsafe external markdown, including obfuscation, scripts, and social engineering | Detect unsafe external markdown, including obfuscation, scripts, and social engineering. | `CTR-007` |
| **T-CTR-008** | CTR-008 Pull Request Target Safety | A `pull_request_target` workflow acknowledges trigger risk, configures base or external checkouts, sets `fetch`, or combines the trigger with `on.roles` containing `read` or `all` | Reject the unsafe role combination in strict mode regardless of acknowledgment, checkout, or feature flag; in non-strict mode warn unless `features.dangerously-allow-roles-read-and-pull-request-target: true`, which suppresses only that warning. Suppress only the strict trigger warning when acknowledged; accept only explicit disablement, trusted base checkouts, or exact allowlisted literal repository/ref pairs without `fetch`; otherwise reject in strict mode and warn in non-strict mode. | `CTR-008` |
| **T-CTR-009** | CTR-009 Shell Expansion in Safe-Outputs | Reject dangerous shell expansion in safe-output scripts | Reject dangerous shell expansion in safe-output scripts. | `CTR-009` |
| **T-CTR-010** | CTR-010 Expression Safety Allowlist | Reject unauthorized or multiline GitHub Actions expressions | Reject unauthorized or multiline GitHub Actions expressions. | `CTR-010` |
| **T-CTR-011** | CTR-011 Network Firewall Configuration | Reject missing firewall prerequisites and strict-mode wildcard domains | Reject missing firewall prerequisites and strict-mode wildcard domains. | `CTR-011` |
| **T-CTR-012** | CTR-012 Safe-Outputs Wildcard Push Scope | Warn for unconstrained wildcard PR-branch pushes | Warn for unconstrained wildcard PR-branch pushes. | `CTR-012` |
| **T-CTR-013** | CTR-013 Argument Injection via Package/Image Names | Reject hyphen-prefixed package and image names before subprocess use | Reject hyphen-prefixed package and image names before subprocess use. | `CTR-013` |
| **T-CTR-014** | CTR-014 Supply Chain Attack via Install Scripts | Warn, or reject in strict mode, when Node install scripts are enabled | Warn, or reject in strict mode, when Node install scripts are enabled. | `CTR-014` |
| **T-CTR-015** | CTR-015 Allowed Label Glob Scope | Reject bare `*` safe-output allowed-label patterns | Reject bare `*` safe-output allowed-label patterns. | `CTR-015` |
| **T-CTR-016** | CTR-016 Compile-Time Manifest Drift | Reject new restricted secrets or action references absent from an existing manifest | Reject new restricted secrets or action references absent from an existing manifest. | `CTR-016` |
| **T-CTR-017** | CTR-017 Secret Leakage via Environment Variables | Uncontrolled secret-expression placement, or OTLP-enabled agent/threat-detection/evals AWF generation with `--exclude-env` support | Warn, or reject in strict mode, for uncontrolled secret-expression placement; exclude all three host-side OTLP variables from agent, inline/external detection, and evals sandboxes while retaining them for host exporters; preserve behavior without OTLP. | `CTR-017` |
| **T-CTR-018** | CTR-018 Version Integrity Bypass | Warn, or reject in strict mode, for `check-for-updates: false` | Warn, or reject in strict mode, for `check-for-updates: false`. | `CTR-018` |
| **T-CTR-019** | CTR-019 Cache-Memory Integrity Enforcement | Require cache updates only after successful agent and threat-detection jobs | Require cache updates only after successful agent and threat-detection jobs. | `CTR-019` |
| **T-CTR-020** | CTR-020 Conditional Import Security | Reject `imports` entries containing `if` | Reject `imports` entries containing `if`. | `CTR-020` |
| **T-CTR-021** | CTR-021 Workflow Run Trigger Branch Scope | Warn, or reject in strict mode, for unscoped `workflow_run`; always reject missing `workflows` | Warn, or reject in strict mode, for unscoped `workflow_run`; always reject missing `workflows`. | `CTR-021` |
| **T-CTR-022** | CTR-022 Git Subprocess Argument Injection | Reject unsafe remote ref/path arguments before invoking Git | Reject unsafe remote ref/path arguments before invoking Git. | `CTR-022` |
| **T-CTR-023** | CTR-023 Bash Command Allowlist Illusion | Reject explicit bash restrictions for engines that cannot enforce them | Reject explicit bash restrictions for engines that cannot enforce them. | `CTR-023` |
| **T-CTR-039** | CTR-025 Framework Self-Prompt Misattribution | Strip only a leading framework `<system>` block before analysis | Strip only a leading framework `<system>` block before analysis. | `CTR-025` |
| **T-CTR-041** | CTR-026 Generated Job Timeout Expression Injection | Reject non-positive or expression job timeout values | Reject non-positive or expression job timeout values. | `CTR-026` |
| **T-CTR-042** | CTR-027 Allowlisted Bot Synchronization Provenance | An allowlisted bot synchronizes a PR authored by another actor | Authorize only when the bot, repository provenance, and PR author satisfy all trust requirements; otherwise deny with `confused_deputy` or `bot_not_active`. | `CTR-027` |
| **T-CTR-043** | CTR-028 Agent Configuration Restore Provenance | A pull-request-triggered workflow generates an agent job that installs agent content through `steps:` or `pre-agent-steps:` | Emit the base-branch restore of the engine's agent configuration folders and root instruction files after the PR checkout and before any agent-content install step and the agent execution step. | `CTR-028` |
| **T-CTR-044** | CTR-029 Repository Dispatch App Authorization | An allowlisted GitHub App sends `repository_dispatch` and the collaborator lookup reports it absent | Authorize only the allowlisted App on `repository_dispatch`; deny non-allowlisted actors, lookup failures, and all other triggers. | `CTR-029` |
| **T-CTR-045** | CTR-030 Activation Checkout Provenance | A workflow with pull-request and/or review triggers generates an activation sparse checkout | Pin supported PR events to the PR base SHA; preserve the existing fallback ref for other events and the same-repo guard when activation auth may use `GITHUB_TOKEN`. | `CTR-030` |
| **T-CTR-046** | CTR-031 Private-to-Public Flow Opt-Out | A workflow configures `tools.github.private-to-public-flows` as `allow` or a server-ID list | Reject both forms in strict mode; emit a security warning in non-strict mode. | `CTR-031` |
| **T-CTR-047** | CTR-032 Safe-Output Execution Provenance Spoofing | A built-in, custom, maintenance, or activation-time publisher creates an issue comment, PR description, or PR review comment/reply, including forged markers, buffered comments, cross-repository targets, and custom/disabled footers | Pass the runtime check-run ID only to the publishing step; replace supplied reserved HTML and Markdown markers with trusted metadata; fail closed on invalid or missing identity; verify that the first persisted body retains the creating `run_id` and `run_url` after destination API filtering, using HTML comments except Markdown anchors for each PR review comment. | `CTR-032` |
| **T-CTR-048** | CTR-033 Explicit Checkout Expansion | A checkout selector is an expression or list, with multi omitted, false, true, or non-boolean; imports mix modes | Enable list controls only for literal true; validate single expression results before both jobs' checkout without list restrictions; reject literal lists without opt-in, invalid mode types, invalid single results, and missing multi policy; preserve per-mode merge identity and independent write authorization. | `CTR-033` |
| **T-CTR-049** | CTR-034 Pull Request Fork Admission | A `pull_request` event uses omitted, pattern, wildcard, or author-association fork configuration | Admit same-repository PRs; default to denying forks; admit a fork only when a configured repository pattern or explicit author-association matches; preserve non-PR triggers and do not change permissions or secret availability. | `CTR-034` |

The core tests exercise their catalog trigger and assert the expected rejection, warning, rewrite, or runtime-safe output.

### 8.2 Optimizer Protocol Test ID Catalog

| Test IDs | Rules |
|---|---|
| T-CTR-024–038, T-CTR-040 | Suppression and optimizer protocol requirements in Section 6 |

Optimizer tests cover suppression validation/auditing/SLA/expiration; degraded API, timeout, and rate-limit handling; and missed schedules. The concrete norm-to-test mapping is maintained in `specs/compiler-threat-detection-compliance/README.md` (Section 6.4 and Section 6.6 norm tables) and enforced against `pkg/workflow/compiler_threat_optimizer_protocol_test.go` by `TestFormal_ComplianceReadmeNormTestNamesStaySynced`.

### 8.3 Deprecated Test ID Retirement

A test ID that is deprecated under Section 5.4 MUST remain listed in Section 8.1, marked `[DEPRECATED]`, but MUST leave the required conformance gate: `requiredTestIDs()` excludes any row marked `[DEPRECATED]`, so a deprecated rule's test is retained for audit history without being required to pass.

## 9. References

- RFC 2119: Key words for use in RFCs to Indicate Requirement Levels
- GitHub Actions syntax and permissions documentation
- gh-aw security architecture and safe-output specifications

## 10. Change Log

The version history for this specification is maintained in `specs/compiler-threat-detection-changelog.md`. A version change MUST add an entry there in the same pull request that updates Section 2.
