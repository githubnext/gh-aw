<!--

Hello fellow agent! Welcome to GitHub Agentic Workflows = Actions + Agent + Safety. 
Here are some pointers to get you started in using this tool.

- Create a new workflow: https://raw.githubusercontent.com/github/gh-aw/main/create.md
- Install: https://raw.githubusercontent.com/github/gh-aw/main/install.md
- Reference: https://raw.githubusercontent.com/github/gh-aw/main/.github/aw/github-agentic-workflows.md

-->

# GitHub Agentic Workflows

Write agentic workflows in natural language markdown, and run them in GitHub Actions.

## Contents

- [Quick Start](#quick-start)
- [Overview](#overview)
- [Guardrails](#guardrails)
- [Documentation](#documentation)
- [Contributing](#contributing)
- [🌍 Community Contributions](#-community-contributions)
- [Share Feedback](#share-feedback)
- [Peli's Agent Factory](#pelis-agent-factory)
- [Related Projects](#related-projects)

## Quick Start

Ready to get your first agentic workflow running? Follow our step-by-step [Quick Start Guide](https://github.github.com/gh-aw/setup/quick-start/) to install the extension, add a sample workflow, and see it in action.

## Overview

Learn about the concepts behind agentic workflows, explore available workflow types, and understand how AI can automate your repository tasks. See [How It Works](https://github.github.com/gh-aw/introduction/how-they-work/).

## Guardrails

Guardrails, safety and security are foundational to GitHub Agentic Workflows. Workflows run with read-only permissions by default, with write operations only allowed through sanitized `safe-outputs`. The system implements multiple layers of protection including sandboxed execution, input sanitization, network isolation, supply chain security (SHA-pinned dependencies), tool allow-listing, and compile-time validation. Access can be gated to team members only, with human approval gates for critical operations, ensuring AI agents operate safely within controlled boundaries. See the [Security Architecture](https://github.github.com/gh-aw/introduction/architecture/) for comprehensive details on threat modeling, implementation guidelines, and best practices.

Using agentic workflows in your repository requires careful attention to security considerations and careful human supervision, and even then things can still go wrong. Use it with caution, and at your own risk.

## Documentation

For complete documentation, examples, and guides, see the [Documentation](https://github.github.com/gh-aw/). If you are an agent, download the [llms.txt](https://github.github.com/gh-aw/llms.txt).

## Contributing

For development setup and contribution guidelines, see [CONTRIBUTING.md](CONTRIBUTING.md).

## 🌍 Community Contributions

Thank you to the community members whose issue reports were resolved in this project!
This list is updated automatically and reflects all attributed contributions.

| Issue | Title | Author | Resolved By | Attribution |
|-------|-------|--------|-------------|-------------|
| [#22138](https://github.com/github/gh-aw/issues/22138) | Release community attribution silently misses valid fixes when resolution flows through follow-up issues | @samuelkahessay | — | direct issue |
| [#22017](https://github.com/github/gh-aw/issues/22017) | Add VEX auto-generator workflow for dismissed Dependabot alerts | @carlincherry | — | direct issue |
| [#21990](https://github.com/github/gh-aw/issues/21990) | `create-pull-request` signed commits fail when branch does not yet exist on remote | @bbonafed | — | direct issue |
| [#21978](https://github.com/github/gh-aw/issues/21978) | gh aw new safe-outputs are not always valid | @kbreit-insight | — | direct issue |
| [#21957](https://github.com/github/gh-aw/issues/21957) | safe_outputs job: agent_output.json not found (nested artifact path) | @Infinnerty | — | direct issue |
| [#21955](https://github.com/github/gh-aw/issues/21955) | GitHub App auth exempts public repos from automatic min-integrity protection | @samuelkahessay | — | direct issue |
| [#21863](https://github.com/github/gh-aw/issues/21863) | `add-comment` safe output declared in frontmatter but missing from compiled handler config | @chrizbo | — | direct issue |
| [#21834](https://github.com/github/gh-aw/issues/21834) | Possible regression bug - safe-outputs fails on uploading artifacts | @molson504x | — | direct issue |
| [#21816](https://github.com/github/gh-aw/issues/21816) | slash_command activation fails for bot comments that append metadata after a newline | @jaroslawgajewski | — | direct issue |
| [#21686](https://github.com/github/gh-aw/issues/21686) | agentic-wiki-writer template uses invalid 'protected-files' property in create-pull-request | @insop | — | direct issue |
| [#21630](https://github.com/github/gh-aw/issues/21630) | Support sparse-checkout in compiled workflows for large monorepos | @Mossaka | — | direct issue |
| [#21615](https://github.com/github/gh-aw/issues/21615) | How does this work on GH ARC? | @molson504x | — | direct issue |
| [#21562](https://github.com/github/gh-aw/issues/21562) | Enterprise blocker: create-pull-request safe output fails with org-level required_signatures ruleset | @mason-tim | — | direct issue |
| [#21542](https://github.com/github/gh-aw/issues/21542) | push-to-pull-request-branch safe-output fails with "Cannot generate incremental patch" due to shallow checkout | @NicoAvanzDev | — | direct issue |
| [#21501](https://github.com/github/gh-aw/issues/21501) | workflow_dispatch targeted issue binding ignored — agent never reads bound issue | @samuelkahessay | — | direct issue |
| [#21334](https://github.com/github/gh-aw/issues/21334) | update_project safe output: add content_repo for cross-repo project item resolution | @johnpreed | — | direct issue |
| [#21313](https://github.com/github/gh-aw/issues/21313) | checkout: false still emits 'Configure Git credentials' steps that fail without .git | @pholleran | — | direct issue |
| [#21304](https://github.com/github/gh-aw/issues/21304) | Built safe-outputs prompt says to use safeoutputs for all GitHub operations | @samuelkahessay | — | direct issue |
| [#21260](https://github.com/github/gh-aw/issues/21260) | Bug: Grumpy Code Wants GH_AW_GITHUB_TOKEN | @veverkap | — | direct issue |
| [#21257](https://github.com/github/gh-aw/issues/21257) | Feature Request: Modify PR before creation | @veverkap | — | direct issue |
| [#21243](https://github.com/github/gh-aw/issues/21243) | Support `github-app:` auth and Claude Code plugin registration for `dependencies:` (APM) | @holwerda | — | direct issue |
| [#21207](https://github.com/github/gh-aw/issues/21207) | create-pull-request safe output does not add reviewers configured in workflow | @alondahari | — | direct issue |
| [#21205](https://github.com/github/gh-aw/issues/21205) | `call-workflow` is not wired into the consolidated `safe_outputs` handler-manager path | @johnwilliams-12 | — | direct issue |
| [#21203](https://github.com/github/gh-aw/issues/21203) | Allow conditional trigger filtering without failing workflow runs | @MattSkala | — | direct issue |
| [#21135](https://github.com/github/gh-aw/issues/21135) | sandbox.mcp.payloadSizeThreshold is ignored during frontmatter extraction | @strawgate | — | direct issue |
| [#21098](https://github.com/github/gh-aw/issues/21098) | check_membership.cjs error branch short-circuits before bot allowlist fallback | @microsasa | — | direct issue |
| [#21074](https://github.com/github/gh-aw/issues/21074) | HTTP safe-outputs server does not register generated `call-workflow` tools | @johnwilliams-12 | — | direct issue |
| [#21071](https://github.com/github/gh-aw/issues/21071) | `call-workflow` generated caller jobs omit required `permissions:` for reusable workflows | @johnwilliams-12 | — | direct issue |
| [#21062](https://github.com/github/gh-aw/issues/21062) | `call-workflow` fan-out jobs do not forward declared `workflow_call.inputs` beyond payload | @johnwilliams-12 | — | direct issue |
| [#21028](https://github.com/github/gh-aw/issues/21028) | Feature: support explicit custom key for close-older matching | @strawgate | — | direct issue |
| [#20953](https://github.com/github/gh-aw/issues/20953) | Warning about push-to-pull-request-branch should not be shown in public repos | @dsyme | — | direct issue |
| [#20952](https://github.com/github/gh-aw/issues/20952) | Build/test failures on main | @dsyme | — | direct issue |
| [#20950](https://github.com/github/gh-aw/issues/20950) | Redaction still too strong | @dsyme | — | direct issue |
| [#20910](https://github.com/github/gh-aw/issues/20910) | workflow_call safe_outputs can download unprefixed agent artifact name | @strawgate | — | direct issue |
| [#20885](https://github.com/github/gh-aw/issues/20885) | Safe-outputs MCP transport silently closes on idle during long agent runs | @benvillalobos | — | direct issue |
| [#20868](https://github.com/github/gh-aw/issues/20868) | `gh aw upgrade` does not correct drift between `uses:` comment version and `with: version:` | @grahame-white | — | direct issue |
| [#20851](https://github.com/github/gh-aw/issues/20851) | The agent cannot close PRs even though the frontmatter explicitly configures it | @microsasa | — | direct issue |
| [#20821](https://github.com/github/gh-aw/issues/20821) | GitHub App token fallback uses full slug instead of repo name in workflow_call relays | @johnwilliams-12 | — | direct issue |
| [#20813](https://github.com/github/gh-aw/issues/20813) | Workflow-level `GH_HOST` leaks into Copilot CLI install step | @jaroslawgajewski | — | direct issue |
| [#20811](https://github.com/github/gh-aw/issues/20811) | Compiler does not emit `GITHUB_HOST` in MCP server env for GHES targets | @jaroslawgajewski | — | direct issue |
| [#20801](https://github.com/github/gh-aw/issues/20801) | Feature Request: `skip-if-no-match` / `skip-if-match` support for cross-repo queries | @bbonafed | — | direct issue |
| [#20787](https://github.com/github/gh-aw/issues/20787) | E2E failures | @dsyme | — | direct issue |
| [#20781](https://github.com/github/gh-aw/issues/20781) | Issue #20664 still unresolved in v0.58.0: target-repo unsupported in submit-pull-request-review | @alexsiilvaa | — | direct issue |
| [#20780](https://github.com/github/gh-aw/issues/20780) | create-pull-request: allow disabling branch name sanitization (lowercase + salt suffix) | @tore-unumed | — | direct issue |
| [#20779](https://github.com/github/gh-aw/issues/20779) | `dispatch-workflow` uses caller's `GITHUB_REF` for cross-repo dispatch instead of target repo's default branch | @johnwilliams-12 | — | direct issue |
| [#20719](https://github.com/github/gh-aw/issues/20719) | Bug: Workflow validator error – Exceeded max expression length in daily-test-improver.lock.yml | @grahame-white | — | direct issue |
| [#20697](https://github.com/github/gh-aw/issues/20697) | Bug: Activation checkout does not preserve callee workflow ref in caller-hosted relays | @johnwilliams-12 | — | direct issue |
| [#20694](https://github.com/github/gh-aw/issues/20694) | Bug: `dispatch_workflow` ignores `target-repo` and dispatches to `context.repo` in cross-repo relays | @johnwilliams-12 | — | direct issue |
| [#20679](https://github.com/github/gh-aw/issues/20679) | `gh aw logs` requests unsupported `path` JSON field from `gh run list` | @rabo-unumed | — | direct issue |
| [#20664](https://github.com/github/gh-aw/issues/20664) | submit_pull_request_review lacks target-repo support and fails in cross-repo workflows | @alexsiilvaa | — | direct issue |
| [#20663](https://github.com/github/gh-aw/issues/20663) | The `dependencies:` documentation undersells APM and lacks guidance for users | @danielmeppiel | — | direct issue |
| [#20658](https://github.com/github/gh-aw/issues/20658) | Bug: `Checkout actions folder` emitted without `repository:` or `ref:` — `Setup Scripts` fails in cross-repo relay | @johnwilliams-12 | — | direct issue |
| [#20657](https://github.com/github/gh-aw/issues/20657) | Activation Upload Artifact Conflict | @fr4nc1sc0-r4m0n | — | direct issue |
| [#20646](https://github.com/github/gh-aw/issues/20646) | invalid html anchor used in error message: CONTRIBUTING.md#prerequisites | @bmerkle | — | direct issue |
| [#20629](https://github.com/github/gh-aw/issues/20629) | compile --actionlint reports zero errors but exits nonzero (false negative or integration bug) | @grahame-white | — | direct issue |
| [#20597](https://github.com/github/gh-aw/issues/20597) | When PR creation is not created due to a fallback agent still claims the PR was created | @tspascoal | — | direct issue |
| [#20592](https://github.com/github/gh-aw/issues/20592) | Option to skip API secret prompt for `add-wizard` | @mcantrell | — | direct issue |
| [#20586](https://github.com/github/gh-aw/issues/20586) | pre_activation role check fails for workflow_run events (should use workflow-based trust) | @microsasa | — | direct issue |
| [#20578](https://github.com/github/gh-aw/issues/20578) | Add warnings about push-to-pull-request-branch | @dsyme | — | direct issue |
| [#20567](https://github.com/github/gh-aw/issues/20567) | Cross-repo activation checkout still broken for event-driven relay workflows after #20301 | @johnwilliams-12 | — | direct issue |
| [#20540](https://github.com/github/gh-aw/issues/20540) | push_to_pull_request_branch: git fetch still fails after clean_git_credentials.sh (v0.53.3) | @NicoAvanzDev | — | direct issue |
| [#20528](https://github.com/github/gh-aw/issues/20528) | push-to-pull-request-branch defaults to max: 0 instead of documented default max: 1 | @NicoAvanzDev | — | direct issue |
| [#20515](https://github.com/github/gh-aw/issues/20515) | `allowed-files` is an allowlist, not an "additionally allow" list — undocumented and counterintuitive | @arezero | — | direct issue |
| [#20513](https://github.com/github/gh-aw/issues/20513) | `protected_path_prefixes` overrides `allowed-files` — no way to allow `.github/` files via frontmatter | @arezero | — | direct issue |
| [#20512](https://github.com/github/gh-aw/issues/20512) | `reply_to_pull_request_review_comment` tool generated in `tools.json` but missing from `config.json` | @arezero | — | direct issue |
| [#20511](https://github.com/github/gh-aw/issues/20511) | `clean_git_credentials.sh` breaks `push_to_pull_request_branch` | @arezero | — | direct issue |
| [#20510](https://github.com/github/gh-aw/issues/20510) | `bots:` allowlist does not override `pre_activation` team membership check | @arezero | — | direct issue |
| [#20483](https://github.com/github/gh-aw/issues/20483) | Repeated tarball download timeouts for external repos consuming gh-aw actions | @DimaBir | — | direct issue |
| [#20457](https://github.com/github/gh-aw/issues/20457) | Codex is able to use web search even when tool is not provided | @eaftan | — | direct issue |
| [#20420](https://github.com/github/gh-aw/issues/20420) | Failed create_pull_request or push_to_pull_request_branch due to merge conflict should create better fallback issue | @dsyme | — | direct issue |
| [#20416](https://github.com/github/gh-aw/issues/20416) | GitHub Agentic Workflow Engine Enhancement Proposal | @CiscoRob | — | direct issue |
| [#20411](https://github.com/github/gh-aw/issues/20411) | Feature Request: `call-workflow` safe output for `workflow_call` chaining | @mvdbos | — | direct issue |
| [#20394](https://github.com/github/gh-aw/issues/20394) | Support configuring a different repository for failure issues | @heiskr | — | direct issue |
| [#20380](https://github.com/github/gh-aw/issues/20380) | feat: Move APM dependency resolution to activation job via pack/unpack | @danielmeppiel | — | direct issue |
| [#20378](https://github.com/github/gh-aw/issues/20378) | `environment:` frontmatter field not propagated to `safe_outputs` job — breaks environment-level secrets | @bbonafed | — | direct issue |
| [#20359](https://github.com/github/gh-aw/issues/20359) | safe-outputs.create-pull-request.draft: false is ignored when agent specifies draft: true | @UncleBats | — | direct issue |
| [#20335](https://github.com/github/gh-aw/issues/20335) | copilot-requests property | @mark-hingston | — | direct issue |
| [#20322](https://github.com/github/gh-aw/issues/20322) | Commits made by AI do not have signature | @chepa92 | — | direct issue |
| [#20308](https://github.com/github/gh-aw/issues/20308) | `max-patch-size` under `tools.repo-memory` rejected by compiler but documented as valid | @G1Vh | — | direct issue |
| [#20299](https://github.com/github/gh-aw/issues/20299) | Bug: `gh aw upgrade` generates lock files with previous version after upgrade | @grahame-white | — | direct issue |
| [#20259](https://github.com/github/gh-aw/issues/20259) | safe-outputs: create_pull_request_review_comment does not treat pull_request_target as PR context | @strawgate | — | direct issue |
| [#20249](https://github.com/github/gh-aw/issues/20249) | Feature Request: Cross-repo `workflow_call` validation and docs | @mvdbos | — | direct issue |
| [#20243](https://github.com/github/gh-aw/issues/20243) | Improve the activation summary | @dsyme | — | direct issue |
| [#20241](https://github.com/github/gh-aw/issues/20241) | Staged mode support needs better docs | @dsyme | — | direct issue |
| [#20222](https://github.com/github/gh-aw/issues/20222) | repo-assist: __GH_AW_WIKI_NOTE__ placeholder not substituted when Wiki is disabled | @ericchansen | — | direct issue |
| [#20187](https://github.com/github/gh-aw/issues/20187) | Job-level concurrency group ignores workflow inputs | @JanKrivanek | — | direct issue |
| [#20168](https://github.com/github/gh-aw/issues/20168) | safe-outputs: target="triggering" rejects pull_request_target PR context | @strawgate | — | direct issue |
| [#20125](https://github.com/github/gh-aw/issues/20125) | safe_outputs: created_issue_* outputs missing because emitter is never called | @strawgate | — | direct issue |
| [#20108](https://github.com/github/gh-aw/issues/20108) | Error: Cannot find module '/opt/gh-aw/actions/campaign_discovery.cjs' | @dsyme | — | direct issue |
| [#20103](https://github.com/github/gh-aw/issues/20103) | Change to protected file not correctly using a fallback issue | @dsyme | — | direct issue |
| [#20035](https://github.com/github/gh-aw/issues/20035) | safe-outputs: handler failures computed in failureCount but never escalated to core.setFailed() | @samuelkahessay | — | direct issue |
| [#20033](https://github.com/github/gh-aw/issues/20033) | Agent sandbox git identity missing: first commit fails, then agent self-configures | @strawgate | — | direct issue |
| [#20031](https://github.com/github/gh-aw/issues/20031) | `dispatch-workflow` validation is compile-order dependent | @samuelkahessay | — | direct issue |
| [#20030](https://github.com/github/gh-aw/issues/20030) | `on.bots` matching is exact-string only and fails for `<slug>` vs `<slug>[bot]` GitHub App identities | @samuelkahessay | — | direct issue |
| [#19976](https://github.com/github/gh-aw/issues/19976) | repo-memory fails when memory exceeds allowed size | @dsyme | — | direct issue |
| [#19839](https://github.com/github/gh-aw/issues/19839) | `gh aw add` cannot be used from an agentic workflow to roll out shared workflows cross-repo | @corymhall | — | direct issue |
| [#19836](https://github.com/github/gh-aw/issues/19836) | [research] Overview of docs improver agents | @mnkiefer | — | direct issue |
| [#19810](https://github.com/github/gh-aw/issues/19810) | Step summary truncates agent output at 500 chars with no visible warning | @danielmeppiel | — | direct issue |
| [#19773](https://github.com/github/gh-aw/issues/19773) | Bug: workflow_dispatch item_number not wired into expression extraction for label trigger shorthand | @deyaaeldeen | — | direct issue |
| [#19770](https://github.com/github/gh-aw/issues/19770) | Bug: Label trigger shorthand does not produce label filter condition in compiled workflow | @deyaaeldeen | — | direct issue |
| [#19765](https://github.com/github/gh-aw/issues/19765) | `assign-to-agent` fails with GitHub App tokens — Copilot assignment API requires a PAT | @mason-tim | — | direct issue |
| [#19732](https://github.com/github/gh-aw/issues/19732) | GitHub App token is repo scoped | @jaroslawgajewski | — | direct issue |
| [#19708](https://github.com/github/gh-aw/issues/19708) | gh aw add-wizard for scheduled workflow should offer choice of frequencies | @dsyme | — | direct issue |
| [#19640](https://github.com/github/gh-aw/issues/19640) | Bug: `gh aw upgrade` wraps `uses` value in quotes, including the inline comment | @srgibbs99 | — | direct issue |
| [#19631](https://github.com/github/gh-aw/issues/19631) | `gh aw upgrade` Reformats `copilot-setup-steps` | @straub | — | direct issue |
| [#19622](https://github.com/github/gh-aw/issues/19622) | Bug: `gh aw upgrade` and `gh aw compile` produce different lock files — toggle endlessly | @srgibbs99 | — | direct issue |
| [#19605](https://github.com/github/gh-aw/issues/19605) | `handle_create_pr_error`: unhandled exceptions on API calls crash conclusion job | @samuelkahessay | — | direct issue |
| [#19547](https://github.com/github/gh-aw/issues/19547) | [Question] Can I not use a PAT for Copilot? | @hrishikeshathalye | — | direct issue |
| [#19500](https://github.com/github/gh-aw/issues/19500) | Bug: gh-aw compile incorrectly prepends repository name to #runtime-import paths in .github repositories | @MatthewLabasan-NBCU | — | direct issue |
| [#19476](https://github.com/github/gh-aw/issues/19476) | push_repo_memory.cjs has no retry/backoff, fails on concurrent pushes | @samuelkahessay | — | direct issue |
| [#19475](https://github.com/github/gh-aw/issues/19475) | get_current_branch.cjs leaks stderr when not in a git repository | @samuelkahessay | — | direct issue |
| [#19474](https://github.com/github/gh-aw/issues/19474) | Unconditional agent-output artifact download causes ENOENT noise on pre-agent failures | @samuelkahessay | — | direct issue |
| [#19473](https://github.com/github/gh-aw/issues/19473) | Copilot engine fallback model path uses --model CLI flag instead of COPILOT_MODEL env var | @samuelkahessay | — | direct issue |
| [#19468](https://github.com/github/gh-aw/issues/19468) | Allowed expressions should allow simple defaults | @dsyme | — | direct issue |
| [#19465](https://github.com/github/gh-aw/issues/19465) | "GitHub Actions is not permitted to create or approve pull requests." | @dsyme | — | direct issue |
| [#19451](https://github.com/github/gh-aw/issues/19451) | Bug: `gh aw upgrade` does not set a sha for `setup-cli` in `copilot-setup-steps.yml` | @rspurgeon | — | direct issue |
| [#19441](https://github.com/github/gh-aw/issues/19441) | The Setup CLI Action Ignores Pinned Version | @harrisoncramer | — | direct issue |
| [#19421](https://github.com/github/gh-aw/issues/19421) | feat: allow configuring the token used for pre-activation reactions | @swimmesberger | — | direct issue |
| [#19370](https://github.com/github/gh-aw/issues/19370) | Cross-repo create-pull-request fails: GITHUB_TOKEN not available for dynamic checkout | @tore-unumed | — | direct issue |
| [#19347](https://github.com/github/gh-aw/issues/19347) | Bug: Cross-repo `update-issue` safe-outputs broken | @chrizbo | — | direct issue |
| [#19219](https://github.com/github/gh-aw/issues/19219) | Cross-repo push-to-pull-request-branch doesn't have access to correct repo contents | @dsyme | — | direct issue |
| [#19172](https://github.com/github/gh-aw/issues/19172) | close-older-issues closes issues from different calling workflows | @strawgate | — | direct issue |
| [#19158](https://github.com/github/gh-aw/issues/19158) | `gh aw checks --json` collapses optional third-party failures into top-level state | @samuelkahessay | — | direct issue |
| [#19120](https://github.com/github/gh-aw/issues/19120) | github.event_name should be an allowed expression | @dsyme | — | direct issue |
| [#19104](https://github.com/github/gh-aw/issues/19104) | Continue to work to remove the dead code | @dsyme | — | direct issue |
| [#19067](https://github.com/github/gh-aw/issues/19067) | Duplicate HANDLER_MAP in JS code - safe_output_unified_handler_manager.cjs is dead code | @dsyme | — | direct issue |
| [#19024](https://github.com/github/gh-aw/issues/19024) | Malformed #aw_* references in body text pass through without validation | @samuelkahessay | — | direct issue |
| [#19023](https://github.com/github/gh-aw/issues/19023) | Mixed-trigger workflows collapse workflow_dispatch runs into degenerate concurrency group | @samuelkahessay | — | direct issue |
| [#19020](https://github.com/github/gh-aw/issues/19020) | Auto-merge gating has no way to ignore non-required third-party deployment statuses | @samuelkahessay | — | direct issue |
| [#19018](https://github.com/github/gh-aw/issues/19018) | EACCES on /tmp/gh-aw/mcp-logs — no ownership repair between workflow runs | @samuelkahessay | — | direct issue |
| [#19017](https://github.com/github/gh-aw/issues/19017) | Permanently deferred safe-output items do not fail the workflow | @samuelkahessay | — | direct issue |
| [#18945](https://github.com/github/gh-aw/issues/18945) | submit_pull_request_review: REQUEST_CHANGES/APPROVE fails on own PR despite override check | @strawgate | — | direct issue |
| [#18900](https://github.com/github/gh-aw/issues/18900) | Replace format-patch/git-am pipeline with tree diff + GraphQL commit API | @strawgate | — | direct issue |
| [#18875](https://github.com/github/gh-aw/issues/18875) | `gh aw trial` fails with 404 — missing `.github/` prefix in workflow path resolution | @maxbeizer | — | direct issue |
| [#18854](https://github.com/github/gh-aw/issues/18854) | Main failing | @dsyme | — | direct issue |
| [#18825](https://github.com/github/gh-aw/issues/18825) | Fix checkout frontmatter: emit token (not github-token) for actions/checkout | @Corb3nik | — | direct issue |
| [#18781](https://github.com/github/gh-aw/issues/18781) | Add ModelsLab Engine for Multi-Modal AI Generation Support | @adhikjoshi | — | direct issue |
| [#18763](https://github.com/github/gh-aw/issues/18763) | Your Docs Provide an Unsafe Expression | @harrisoncramer | — | direct issue |
| [#18751](https://github.com/github/gh-aw/issues/18751) | Bug Report: `safeoutputs` MCP server crashes with `context is not defined` on `create_pull_request` | @srgibbs99 | — | direct issue |
| [#18745](https://github.com/github/gh-aw/issues/18745) | Bug: \| block scalar description in safe-inputs breaks generated Python script | @srgibbs99 | — | direct issue |
| [#18744](https://github.com/github/gh-aw/issues/18744) | feat: add target config to resolve-pull-request-review-thread | @strawgate | — | direct issue |
| [#18723](https://github.com/github/gh-aw/issues/18723) | Feature Request: Option to suppress "Generated by..." text | @beardofedu | — | direct issue |
| [#18714](https://github.com/github/gh-aw/issues/18714) | MCP gateway /close teardown fails with invalid API key (gateway-api-key output quoted) | @aaronspindler | — | direct issue |
| [#18712](https://github.com/github/gh-aw/issues/18712) | Copilot CLI does not recognize HTTP-based custom MCP server tools despite successful gateway connection | @lupinthe14th | — | direct issue |
| [#18711](https://github.com/github/gh-aw/issues/18711) | runtime-import fails for .github/workflows/* paths (resolved as workflows/*) | @DrPye | — | direct issue |
| [#18703](https://github.com/github/gh-aw/issues/18703) | safeoutputs-push_to_pull_request_branch fails on fetch | @AlexanderWert | — | direct issue |
| [#18574](https://github.com/github/gh-aw/issues/18574) | In private repos, events triggered in comments PRs are not able to access the PR branch | @dsyme | — | direct issue |
| [#18565](https://github.com/github/gh-aw/issues/18565) | Commits via `git` are unverified; switch to GraphQL for commits | @strawgate | — | direct issue |
| [#18563](https://github.com/github/gh-aw/issues/18563) | Check-out from Fork does not work with workflow_call | @strawgate | — | direct issue |
| [#18557](https://github.com/github/gh-aw/issues/18557) | `network: { allowed: [] }` still allows infrastructure domains — same behavior as `network: {}` | @benvillalobos | — | direct issue |
| [#18556](https://github.com/github/gh-aw/issues/18556) | Workflows fail with 'Copilot is not a user' error on agent-created PRs | @KGoovaer | — | direct issue |
| [#18547](https://github.com/github/gh-aw/issues/18547) | safe_outputs checkout fails for pull_request_review events | @strawgate | — | direct issue |
| [#18545](https://github.com/github/gh-aw/issues/18545) | bug: duplicate env vars when import and main workflow reference the same repository variable | @strawgate | — | direct issue |
| [#18542](https://github.com/github/gh-aw/issues/18542) | Support for external secret managers | @bbonafed | — | direct issue |
| [#18535](https://github.com/github/gh-aw/issues/18535) | Instructions for issue created when pull request creation failed should be better | @dsyme | — | direct issue |
| [#18501](https://github.com/github/gh-aw/issues/18501) | create_pull_request fails with large commits | @strawgate | — | direct issue |
| [#18488](https://github.com/github/gh-aw/issues/18488) | Add issue type add/remove safe output | @Krzysztof-Cieslak | — | direct issue |
| [#18485](https://github.com/github/gh-aw/issues/18485) | gh aw add-wizard: If the workflow has an engine declaration, and the user chooses a different engine | @dsyme | — | direct issue |
| [#18483](https://github.com/github/gh-aw/issues/18483) | gh add-wizard: Failed to commit files in a repo | @dsyme | — | direct issue |
| [#18482](https://github.com/github/gh-aw/issues/18482) | `gh add-wizard` - if the user doesn't have write access, don't ask them to configure secrets | @dsyme | — | direct issue |
| [#18481](https://github.com/github/gh-aw/issues/18481) | Using gh-aw in forks of repositories | @dsyme | — | direct issue |
| [#18480](https://github.com/github/gh-aw/issues/18480) | gh-aw not working in cloud enterprise environments | @JoshGreenslade | — | direct issue |
| [#18468](https://github.com/github/gh-aw/issues/18468) | Add broken redirects for these patterns pages | @samus-aran | — | direct issue |
| [#18465](https://github.com/github/gh-aw/issues/18465) | safe_output_handler_manager ignores allowed-domains, redacts URLs from allowlisted domains | @theletterf | — | direct issue |
| [#18421](https://github.com/github/gh-aw/issues/18421) | gh aw update fails | @dsolteszopyn | — | direct issue |
| [#18386](https://github.com/github/gh-aw/issues/18386) | Copilot workflow steps cannot access Azure/Azure DevOps APIs after azure/login@v2 | @praveenkuttappan | — | direct issue |
| [#18385](https://github.com/github/gh-aw/issues/18385) | Squid config error on self-hosted ARC Runners | @dhrapson | — | direct issue |
| [#18379](https://github.com/github/gh-aw/issues/18379) | Feature request to support GitHub app-based authentication for copilot requests | @praveenkuttappan | — | direct issue |
| [#18373](https://github.com/github/gh-aw/issues/18373) | `gh aw compile` consistent actions/setup sha generation | @rspurgeon | — | direct issue |
| [#18362](https://github.com/github/gh-aw/issues/18362) | Safe Output custom token source | @strawgate | — | direct issue |
| [#18356](https://github.com/github/gh-aw/issues/18356) | workflows run errors if used as required in repository ruleset | @jaroslawgajewski | — | direct issue |
| [#18340](https://github.com/github/gh-aw/issues/18340) | shell(dotnet) tool denied despite being in allowed tools — requires 'env dotnet' workaround | @ViktorHofer | — | direct issue |
| [#18329](https://github.com/github/gh-aw/issues/18329) | How to create PRs in multiple repos from a single workflow? | @tore-unumed | — | direct issue |
| [#18311](https://github.com/github/gh-aw/issues/18311) | gh aw compile does not add pull-requests: write to safe_outputs job when add-comment is configured | @ViktorHofer | — | direct issue |
| [#18295](https://github.com/github/gh-aw/issues/18295) | MCP tool calling loop issues | @adam-cobb | — | direct issue |
| [#18263](https://github.com/github/gh-aw/issues/18263) | Add support for the android-arm64 architecture | @BrandonLewis | — | direct issue |
| [#18226](https://github.com/github/gh-aw/issues/18226) | fix: imported safe-output fragments override explicit threat-detection: false | @strawgate | — | direct issue |
| [#18211](https://github.com/github/gh-aw/issues/18211) | Add some notion of embedded resource/file that gets installed with a workflow | @dsyme | — | direct issue |
| [#18200](https://github.com/github/gh-aw/issues/18200) | `hide-older-comments` on `add-comment` safe output finds no matching comments despite correct `workflow_id` marker | @Nikhil-Anand-DSG | — | direct issue |
| [#18196](https://github.com/github/gh-aw/issues/18196) | Editor Link is invalid | @jeremiah-snee-openx | — | direct issue |
| [#18162](https://github.com/github/gh-aw/issues/18162) | Update status field in Github Project | @qwert666 | — | direct issue |
| [#18123](https://github.com/github/gh-aw/issues/18123) | Mermaid flowchart node multiline text is not rendered correctly in the documentation | @tspascoal | — | direct issue |
| [#18121](https://github.com/github/gh-aw/issues/18121) | Add explicit CI state classification command for gh-aw PR triage | @davidahmann | — | direct issue |
| [#18115](https://github.com/github/gh-aw/issues/18115) | GitHub MCP `issue_read` tool unavailable when app token is scoped to multiple repositories | @benvillalobos | — | direct issue |
| [#18109](https://github.com/github/gh-aw/issues/18109) | `allowed-repos` not accepted inline for `assign-to-user` and `remove-labels` safe outputs (schema gap) | @benvillalobos | — | direct issue |
| [#18107](https://github.com/github/gh-aw/issues/18107) | safe-outputs create_pull_request fails for cross-repo checkouts: uses GITHUB_SHA from workflow repo as merge base | @tore-unumed | — | direct issue |
| [#18103](https://github.com/github/gh-aw/issues/18103) | Compiler drops 'blocked' constraints from safe-outputs configs inconsistently | @benvillalobos | — | direct issue |
| [#18101](https://github.com/github/gh-aw/issues/18101) | Bug: Per-engine job concurrency blocks workflow_dispatch issue workflows from running in parallel | @benvillalobos | — | direct issue |
| [#18018](https://github.com/github/gh-aw/issues/18018) | Lots of failures on push-to-pull-request-branch | @dsyme | — | direct issue |
| [#17995](https://github.com/github/gh-aw/issues/17995) | Confusing error message: `max-turns not supported` example contradicts the error | @benvillalobos | — | direct issue |
| [#17982](https://github.com/github/gh-aw/issues/17982) | Fix support for custom named COPILOT_GITHUB_TOKEN secret | @racedale | — | direct issue |
| [#17978](https://github.com/github/gh-aw/issues/17978) | gh-aw: GitHub App token narrowing omits Dependabot alerts permission for GitHub MCP | @Dan-Co | — | direct issue |
| [#17975](https://github.com/github/gh-aw/issues/17975) | Code Simplification agent silently fails to create PRs when the repo stores line endings as CRLF | @AmoebaChant | — | direct issue |
| [#17962](https://github.com/github/gh-aw/issues/17962) | Full self-hosted runner support | @dhrapson | — | direct issue |
| [#17943](https://github.com/github/gh-aw/issues/17943) | Bug: `engine.agent` propagates to threat detection job, causing "No such agent" failure | @benvillalobos | — | direct issue |
| [#17839](https://github.com/github/gh-aw/issues/17839) | Retry downloads automatically | @strawgate | — | direct issue |
| [#17828](https://github.com/github/gh-aw/issues/17828) | Feature request: add flag to disable activation/fallback comments | @strawgate | — | direct issue |
| [#17763](https://github.com/github/gh-aw/issues/17763) | Unable to use ci-coach | @askpt | — | direct issue |
| [#17522](https://github.com/github/gh-aw/issues/17522) | update-pull-request should honor footer: false | @strawgate | — | direct issue |
| [#17521](https://github.com/github/gh-aw/issues/17521) | Add safe-output fail-fast mode for code push operations | @strawgate | — | direct issue |
| [#17299](https://github.com/github/gh-aw/issues/17299) | [bug] base-branch in assign-to-agent uses customInstructions text instead of GraphQL baseRef field | @steliosfran | — | direct issue |
| [#17298](https://github.com/github/gh-aw/issues/17298) | HTML in `update-issue` body gets escaped/mangled | @srgibbs99 | — | direct issue |
| [#17289](https://github.com/github/gh-aw/issues/17289) | create-pull-request safe output fails with "No changes to commit" when workspace is a cross-repo checkout | @tore-unumed | — | direct issue |
| [#17243](https://github.com/github/gh-aw/issues/17243) | cache-memory: GH_AW_WORKFLOW_ID_SANITIZED not defined in update_cache_memory job | @joperezr | — | direct issue |
| [#17151](https://github.com/github/gh-aw/issues/17151) | Stabilize frontmatter hash across LF/CRLF newline conventions | @davidahmann | — | direct issue |
| [#17058](https://github.com/github/gh-aw/issues/17058) | Daily Documentation Updater fails to run | @dsolteszopyn | — | direct issue |
| [#17046](https://github.com/github/gh-aw/issues/17046) | [enhancement] Add base-branch support to assign-to-agent safe output for cross-repo PR creation | @steliosfran | — | direct issue |
| [#16896](https://github.com/github/gh-aw/issues/16896) | Customize Checkout Depth | @strawgate | — | direct issue |
| [#16673](https://github.com/github/gh-aw/issues/16673) | add-comment doesn't actually require `pull_requests: write` | @strawgate | — | direct issue |
| [#16664](https://github.com/github/gh-aw/issues/16664) | Support repository-local `mcp.json` | @strawgate | — | direct issue |
| [#16642](https://github.com/github/gh-aw/issues/16642) | `add-reviewer` safe-output handler not loaded at runtime — message skipped with warning | @pmalarme | — | direct issue |
| [#16625](https://github.com/github/gh-aw/issues/16625) | Feature request: `blocked` pattern matching for `add-labels` safe output | @benvillalobos | — | direct issue |
| [#16587](https://github.com/github/gh-aw/issues/16587) | submit_pull_request_review safe output: review context lost during finalization — review never submitted | @ppusateri | — | direct issue |
| [#16555](https://github.com/github/gh-aw/issues/16555) | Feature Request: Add support for status checks as integration points for ci-doctor | @rmarinho | — | direct issue |
| [#16511](https://github.com/github/gh-aw/issues/16511) | Add `inline-prompt` option to compile workflows without runtime-import macros | @strawgate | — | direct issue |
| [#16467](https://github.com/github/gh-aw/issues/16467) | AWF chroot: COPILOT_GITHUB_TOKEN not passed to Copilot CLI despite --env-all | @jaroslawgajewski | — | direct issue |
| [#16457](https://github.com/github/gh-aw/issues/16457) | assign-to-user / unassign-from-user safe outputs are ignored | @eran-medan | — | direct issue |
| [#16370](https://github.com/github/gh-aw/issues/16370) | Nested remote imports resolve against hardcoded .github/workflows/ instead of parent workflowspec base path | @strawgate | — | direct issue |
| [#16360](https://github.com/github/gh-aw/issues/16360) | Add lock schema compatibility gate for compiled .lock.yml files | @davidahmann | — | direct issue |
| [#16331](https://github.com/github/gh-aw/issues/16331) | push-to-pull-request-branch safe output unconditionally requests issues: write | @timdittler | — | direct issue |
| [#16314](https://github.com/github/gh-aw/issues/16314) | The workflow compiler always adds discussions permission into generated jobs | @jaroslawgajewski | — | direct issue |
| [#16312](https://github.com/github/gh-aw/issues/16312) | compiled agentic workflows require modification when in a enterprise cloud environment | @JoshGreenslade | — | direct issue |
| [#16280](https://github.com/github/gh-aw/issues/16280) | Allow `assign-to-agent` safe output to select the repo that the PR should be created in | @steliosfran | — | direct issue |
| [#16236](https://github.com/github/gh-aw/issues/16236) | Playwright MCP tools not available in GitHub Agentic Workflows (initialize: EOF during MCP init) | @Phonesis | — | direct issue |
| [#16163](https://github.com/github/gh-aw/issues/16163) | Cannot create PR modifying .github/workflows/* due to disallowed workflows:write permission | @elika56 | — | direct issue |
| [#16150](https://github.com/github/gh-aw/issues/16150) | `push_repo_memory.cjs` script has hardcoded github.com reference | @jaroslawgajewski | — | direct issue |
| [#16145](https://github.com/github/gh-aw/issues/16145) | Add Mistral Vibe as coding agent | @dsyme | — | direct issue |
| [#16117](https://github.com/github/gh-aw/issues/16117) | create-pull-request: reviewers config not compiled into handler config | @timdittler | — | direct issue |
| [#16107](https://github.com/github/gh-aw/issues/16107) | App token for safe-outputs doesn't work | @timdittler | — | direct issue |
| [#16036](https://github.com/github/gh-aw/issues/16036) | Add fuzzy scheduling for running on weekdays | @strawgate | — | direct issue |
| [#16005](https://github.com/github/gh-aw/issues/16005) | ARM64 container images not available for gh-aw firewall/MCP gateway | @mstrathman | — | direct issue |
| [#15982](https://github.com/github/gh-aw/issues/15982) | Nested local path imports in remote workflows should resolve | @strawgate | — | direct issue |
| [#15976](https://github.com/github/gh-aw/issues/15976) | add-comment tool enforces rules during safe-outputs instead of during call | @strawgate | — | direct issue |
| [#15836](https://github.com/github/gh-aw/issues/15836) | Create a comment with a link to create a pull request | @strawgate | — | direct issue |
| [#15595](https://github.com/github/gh-aw/issues/15595) | safe-outputs create-discussion does not apply configured labels | @rspurgeon | — | direct issue |
| [#15583](https://github.com/github/gh-aw/issues/15583) | Consider dedicated setting for PR Review Footer w/o Body | @strawgate | — | direct issue |
| [#15576](https://github.com/github/gh-aw/issues/15576) | Add safe outputs for replying to and resolving pull request review comments | @strawgate | — | direct issue |
| [#15510](https://github.com/github/gh-aw/issues/15510) | Create a workflow that bills the Codex subscription instead of API key | @whoschek | — | direct issue |
| [#11190](https://github.com/github/gh-aw/issues/11190) | How to install yarn? | @rafael-unloan | — | direct issue |

## Share Feedback

We welcome your feedback on GitHub Agentic Workflows! 

- [Community Feedback Discussions](https://github.com/orgs/community/discussions/186451)
- [GitHub Next Discord](https://gh.io/next-discord)

## Peli's Agent Factory

See the [Peli's Agent Factory](https://github.github.com/gh-aw/blog/2026-01-12-welcome-to-pelis-agent-factory/) for a guided tour through many uses of agentic workflows.

## Related Projects

GitHub Agentic Workflows is supported by companion projects that provide additional security and integration capabilities:

- **[Agent Workflow Firewall (AWF)](https://github.com/github/gh-aw-firewall)** - Network egress control for AI agents, providing domain-based access controls and activity logging for secure workflow execution
- **[MCP Gateway](https://github.com/github/gh-aw-mcpg)** - Routes Model Context Protocol (MCP) server calls through a unified HTTP gateway for centralized access management
- **[gh-aw-actions](https://github.com/github/gh-aw-actions)** - Shared library of custom GitHub Actions used by compiled workflows, providing functionality such as MCP server file management
