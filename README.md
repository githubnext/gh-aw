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

<details>
<summary>Thank you to the community members whose issue reports were resolved in this project! This list is updated automatically and reflects all attributed contributions.</summary>

### @aaronspindler

- [MCP gateway /close teardown fails with invalid API key (gateway-api-key output quoted)](https://github.com/github/gh-aw/issues/18714) _(direct issue)_

### @adam-cobb

- [MCP tool calling loop issues](https://github.com/github/gh-aw/issues/18295) _(direct issue)_

### @adhikjoshi

- [Add ModelsLab Engine for Multi-Modal AI Generation Support](https://github.com/github/gh-aw/issues/18781) _(direct issue)_

### @AlexanderWert

- [safeoutputs-push_to_pull_request_branch fails on fetch](https://github.com/github/gh-aw/issues/18703) _(direct issue)_

### @alexsiilvaa

- [Issue #20664 still unresolved in v0.58.0: target-repo unsupported in submit-pull-request-review](https://github.com/github/gh-aw/issues/20781) _(direct issue)_
- [submit_pull_request_review lacks target-repo support and fails in cross-repo workflows](https://github.com/github/gh-aw/issues/20664) _(direct issue)_

### @alondahari

- [create-pull-request safe output does not add reviewers configured in workflow](https://github.com/github/gh-aw/issues/21207) _(direct issue)_

### @AmoebaChant

- [Code Simplification agent silently fails to create PRs when the repo stores line endings as CRLF](https://github.com/github/gh-aw/issues/17975) _(direct issue)_

### @arezero

- [`allowed-files` is an allowlist, not an "additionally allow" list — undocumented and counterintuitive](https://github.com/github/gh-aw/issues/20515) _(direct issue)_
- [`protected_path_prefixes` overrides `allowed-files` — no way to allow `.github/` files via frontmatter](https://github.com/github/gh-aw/issues/20513) _(direct issue)_
- [`reply_to_pull_request_review_comment` tool generated in `tools.json` but missing from `config.json`](https://github.com/github/gh-aw/issues/20512) _(direct issue)_
- [`clean_git_credentials.sh` breaks `push_to_pull_request_branch`](https://github.com/github/gh-aw/issues/20511) _(direct issue)_
- [`bots:` allowlist does not override `pre_activation` team membership check](https://github.com/github/gh-aw/issues/20510) _(direct issue)_

### @askpt

- [Unable to use ci-coach](https://github.com/github/gh-aw/issues/17763) _(direct issue)_

### @bbonafed

- [`create-pull-request` signed commits fail when branch does not yet exist on remote](https://github.com/github/gh-aw/issues/21990) _(direct issue)_
- [Feature Request: `skip-if-no-match` / `skip-if-match` support for cross-repo queries](https://github.com/github/gh-aw/issues/20801) _(direct issue)_
- [`environment:` frontmatter field not propagated to `safe_outputs` job — breaks environment-level secrets](https://github.com/github/gh-aw/issues/20378) _(direct issue)_
- [Support for external secret managers](https://github.com/github/gh-aw/issues/18542) _(direct issue)_

### @beardofedu

- [Feature Request: Option to suppress "Generated by..." text](https://github.com/github/gh-aw/issues/18723) _(direct issue)_

### @benvillalobos

- [Safe-outputs MCP transport silently closes on idle during long agent runs](https://github.com/github/gh-aw/issues/20885) _(direct issue)_
- [`network: { allowed: [] }` still allows infrastructure domains — same behavior as `network: {}`](https://github.com/github/gh-aw/issues/18557) _(direct issue)_
- [GitHub MCP `issue_read` tool unavailable when app token is scoped to multiple repositories](https://github.com/github/gh-aw/issues/18115) _(direct issue)_
- [`allowed-repos` not accepted inline for `assign-to-user` and `remove-labels` safe outputs (schema gap)](https://github.com/github/gh-aw/issues/18109) _(direct issue)_
- [Compiler drops 'blocked' constraints from safe-outputs configs inconsistently](https://github.com/github/gh-aw/issues/18103) _(direct issue)_
- [Bug: Per-engine job concurrency blocks workflow_dispatch issue workflows from running in parallel](https://github.com/github/gh-aw/issues/18101) _(direct issue)_
- [Confusing error message: `max-turns not supported` example contradicts the error](https://github.com/github/gh-aw/issues/17995) _(direct issue)_
- [Bug: `engine.agent` propagates to threat detection job, causing "No such agent" failure](https://github.com/github/gh-aw/issues/17943) _(direct issue)_
- [Feature request: `blocked` pattern matching for `add-labels` safe output](https://github.com/github/gh-aw/issues/16625) _(direct issue)_

### @bmerkle

- [invalid html anchor used in error message: CONTRIBUTING.md#prerequisites](https://github.com/github/gh-aw/issues/20646) _(direct issue)_

### @BrandonLewis

- [Add support for the android-arm64 architecture](https://github.com/github/gh-aw/issues/18263) _(direct issue)_

### @carlincherry

- [Add VEX auto-generator workflow for dismissed Dependabot alerts](https://github.com/github/gh-aw/issues/22017) _(direct issue)_

### @chepa92

- [Commits made by AI do not have signature](https://github.com/github/gh-aw/issues/20322) _(direct issue)_

### @chrizbo

- [`add-comment` safe output declared in frontmatter but missing from compiled handler config](https://github.com/github/gh-aw/issues/21863) _(direct issue)_
- [Bug: Cross-repo `update-issue` safe-outputs broken](https://github.com/github/gh-aw/issues/19347) _(direct issue)_

### @CiscoRob

- [GitHub Agentic Workflow Engine Enhancement Proposal](https://github.com/github/gh-aw/issues/20416) _(direct issue)_

### @Corb3nik

- [Fix checkout frontmatter: emit token (not github-token) for actions/checkout](https://github.com/github/gh-aw/issues/18825) _(direct issue)_

### @corymhall

- [`gh aw add` cannot be used from an agentic workflow to roll out shared workflows cross-repo](https://github.com/github/gh-aw/issues/19839) _(direct issue)_

### @Dan-Co

- [gh-aw: GitHub App token narrowing omits Dependabot alerts permission for GitHub MCP](https://github.com/github/gh-aw/issues/17978) _(direct issue)_

### @danielmeppiel

- [The `dependencies:` documentation undersells APM and lacks guidance for users](https://github.com/github/gh-aw/issues/20663) _(direct issue)_
- [feat: Move APM dependency resolution to activation job via pack/unpack](https://github.com/github/gh-aw/issues/20380) _(direct issue)_
- [Step summary truncates agent output at 500 chars with no visible warning](https://github.com/github/gh-aw/issues/19810) _(direct issue)_

### @davidahmann

- [Add explicit CI state classification command for gh-aw PR triage](https://github.com/github/gh-aw/issues/18121) _(direct issue)_
- [Stabilize frontmatter hash across LF/CRLF newline conventions](https://github.com/github/gh-aw/issues/17151) _(direct issue)_
- [Add lock schema compatibility gate for compiled .lock.yml files](https://github.com/github/gh-aw/issues/16360) _(direct issue)_

### @deyaaeldeen

- [Bug: workflow_dispatch item_number not wired into expression extraction for label trigger shorthand](https://github.com/github/gh-aw/issues/19773) _(direct issue)_
- [Bug: Label trigger shorthand does not produce label filter condition in compiled workflow](https://github.com/github/gh-aw/issues/19770) _(direct issue)_

### @dhrapson

- [Squid config error on self-hosted ARC Runners](https://github.com/github/gh-aw/issues/18385) _(direct issue)_
- [Full self-hosted runner support](https://github.com/github/gh-aw/issues/17962) _(direct issue)_

### @DimaBir

- [Repeated tarball download timeouts for external repos consuming gh-aw actions](https://github.com/github/gh-aw/issues/20483) _(direct issue)_

### @DrPye

- [runtime-import fails for .github/workflows/* paths (resolved as workflows/*)](https://github.com/github/gh-aw/issues/18711) _(direct issue)_

### @dsolteszopyn

- [gh aw update fails](https://github.com/github/gh-aw/issues/18421) _(direct issue)_
- [Daily Documentation Updater fails to run](https://github.com/github/gh-aw/issues/17058) _(direct issue)_

### @dsyme

- [Warning about push-to-pull-request-branch should not be shown in public repos](https://github.com/github/gh-aw/issues/20953) _(direct issue)_
- [Build/test failures on main](https://github.com/github/gh-aw/issues/20952) _(direct issue)_
- [Redaction still too strong](https://github.com/github/gh-aw/issues/20950) _(direct issue)_
- [E2E failures](https://github.com/github/gh-aw/issues/20787) _(direct issue)_
- [Add warnings about push-to-pull-request-branch](https://github.com/github/gh-aw/issues/20578) _(direct issue)_
- [Failed create_pull_request or push_to_pull_request_branch due to merge conflict should create better fallback issue](https://github.com/github/gh-aw/issues/20420) _(direct issue)_
- [Improve the activation summary](https://github.com/github/gh-aw/issues/20243) _(direct issue)_
- [Staged mode support needs better docs](https://github.com/github/gh-aw/issues/20241) _(direct issue)_
- [Error: Cannot find module '/opt/gh-aw/actions/campaign_discovery.cjs'](https://github.com/github/gh-aw/issues/20108) _(direct issue)_
- [Change to protected file not correctly using a fallback issue](https://github.com/github/gh-aw/issues/20103) _(direct issue)_
- [repo-memory fails when memory exceeds allowed size](https://github.com/github/gh-aw/issues/19976) _(direct issue)_
- [gh aw add-wizard for scheduled workflow should offer choice of frequencies](https://github.com/github/gh-aw/issues/19708) _(direct issue)_
- [Allowed expressions should allow simple defaults](https://github.com/github/gh-aw/issues/19468) _(direct issue)_
- ["GitHub Actions is not permitted to create or approve pull requests."](https://github.com/github/gh-aw/issues/19465) _(direct issue)_
- [Cross-repo push-to-pull-request-branch doesn't have access to correct repo contents](https://github.com/github/gh-aw/issues/19219) _(direct issue)_
- [github.event_name should be an allowed expression](https://github.com/github/gh-aw/issues/19120) _(direct issue)_
- [Continue to work to remove the dead code](https://github.com/github/gh-aw/issues/19104) _(direct issue)_
- [Duplicate HANDLER_MAP in JS code - safe_output_unified_handler_manager.cjs is dead code](https://github.com/github/gh-aw/issues/19067) _(direct issue)_
- [Main failing](https://github.com/github/gh-aw/issues/18854) _(direct issue)_
- [In private repos, events triggered in comments PRs are not able to access the PR branch](https://github.com/github/gh-aw/issues/18574) _(direct issue)_
- [Instructions for issue created when pull request creation failed should be better](https://github.com/github/gh-aw/issues/18535) _(direct issue)_
- [gh aw add-wizard: If the workflow has an engine declaration, and the user chooses a different engine](https://github.com/github/gh-aw/issues/18485) _(direct issue)_
- [gh add-wizard: Failed to commit files in a repo](https://github.com/github/gh-aw/issues/18483) _(direct issue)_
- [`gh add-wizard` - if the user doesn't have write access, don't ask them to configure secrets](https://github.com/github/gh-aw/issues/18482) _(direct issue)_
- [Using gh-aw in forks of repositories](https://github.com/github/gh-aw/issues/18481) _(direct issue)_
- [Add some notion of embedded resource/file that gets installed with a workflow](https://github.com/github/gh-aw/issues/18211) _(direct issue)_
- [Lots of failures on push-to-pull-request-branch](https://github.com/github/gh-aw/issues/18018) _(direct issue)_
- [Add Mistral Vibe as coding agent](https://github.com/github/gh-aw/issues/16145) _(direct issue)_

### @eaftan

- [Codex is able to use web search even when tool is not provided](https://github.com/github/gh-aw/issues/20457) _(direct issue)_

### @elika56

- [Cannot create PR modifying .github/workflows/* due to disallowed workflows:write permission](https://github.com/github/gh-aw/issues/16163) _(direct issue)_

### @eran-medan

- [assign-to-user / unassign-from-user safe outputs are ignored](https://github.com/github/gh-aw/issues/16457) _(direct issue)_

### @ericchansen

- [repo-assist: __GH_AW_WIKI_NOTE__ placeholder not substituted when Wiki is disabled](https://github.com/github/gh-aw/issues/20222) _(direct issue)_

### @fr4nc1sc0-r4m0n

- [Activation Upload Artifact Conflict](https://github.com/github/gh-aw/issues/20657) _(direct issue)_

### @G1Vh

- [`max-patch-size` under `tools.repo-memory` rejected by compiler but documented as valid](https://github.com/github/gh-aw/issues/20308) _(direct issue)_

### @grahame-white

- [`gh aw upgrade` does not correct drift between `uses:` comment version and `with: version:`](https://github.com/github/gh-aw/issues/20868) _(direct issue)_
- [Bug: Workflow validator error – Exceeded max expression length in daily-test-improver.lock.yml](https://github.com/github/gh-aw/issues/20719) _(direct issue)_
- [compile --actionlint reports zero errors but exits nonzero (false negative or integration bug)](https://github.com/github/gh-aw/issues/20629) _(direct issue)_
- [Bug: `gh aw upgrade` generates lock files with previous version after upgrade](https://github.com/github/gh-aw/issues/20299) _(direct issue)_

### @harrisoncramer

- [The Setup CLI Action Ignores Pinned Version](https://github.com/github/gh-aw/issues/19441) _(direct issue)_
- [Your Docs Provide an Unsafe Expression](https://github.com/github/gh-aw/issues/18763) _(direct issue)_

### @heiskr

- [Support configuring a different repository for failure issues](https://github.com/github/gh-aw/issues/20394) _(direct issue)_

### @holwerda

- [Support `github-app:` auth and Claude Code plugin registration for `dependencies:` (APM)](https://github.com/github/gh-aw/issues/21243) _(direct issue)_

### @hrishikeshathalye

- [[Question] Can I not use a PAT for Copilot?](https://github.com/github/gh-aw/issues/19547) _(direct issue)_

### @Infinnerty

- [safe_outputs job: agent_output.json not found (nested artifact path)](https://github.com/github/gh-aw/issues/21957) _(direct issue)_

### @insop

- [agentic-wiki-writer template uses invalid 'protected-files' property in create-pull-request](https://github.com/github/gh-aw/issues/21686) _(direct issue)_

### @JanKrivanek

- [Job-level concurrency group ignores workflow inputs](https://github.com/github/gh-aw/issues/20187) _(direct issue)_

### @jaroslawgajewski

- [slash_command activation fails for bot comments that append metadata after a newline](https://github.com/github/gh-aw/issues/21816) _(direct issue)_
- [Workflow-level `GH_HOST` leaks into Copilot CLI install step](https://github.com/github/gh-aw/issues/20813) _(direct issue)_
- [Compiler does not emit `GITHUB_HOST` in MCP server env for GHES targets](https://github.com/github/gh-aw/issues/20811) _(direct issue)_
- [GitHub App token is repo scoped](https://github.com/github/gh-aw/issues/19732) _(direct issue)_
- [workflows run errors if used as required in repository ruleset](https://github.com/github/gh-aw/issues/18356) _(direct issue)_
- [AWF chroot: COPILOT_GITHUB_TOKEN not passed to Copilot CLI despite --env-all](https://github.com/github/gh-aw/issues/16467) _(direct issue)_
- [The workflow compiler always adds discussions permission into generated jobs](https://github.com/github/gh-aw/issues/16314) _(direct issue)_
- [`push_repo_memory.cjs` script has hardcoded github.com reference](https://github.com/github/gh-aw/issues/16150) _(direct issue)_

### @jeremiah-snee-openx

- [Editor Link is invalid](https://github.com/github/gh-aw/issues/18196) _(direct issue)_

### @johnpreed

- [update_project safe output: add content_repo for cross-repo project item resolution](https://github.com/github/gh-aw/issues/21334) _(direct issue)_

### @johnwilliams-12

- [`call-workflow` is not wired into the consolidated `safe_outputs` handler-manager path](https://github.com/github/gh-aw/issues/21205) _(direct issue)_
- [HTTP safe-outputs server does not register generated `call-workflow` tools](https://github.com/github/gh-aw/issues/21074) _(direct issue)_
- [`call-workflow` generated caller jobs omit required `permissions:` for reusable workflows](https://github.com/github/gh-aw/issues/21071) _(direct issue)_
- [`call-workflow` fan-out jobs do not forward declared `workflow_call.inputs` beyond payload](https://github.com/github/gh-aw/issues/21062) _(direct issue)_
- [GitHub App token fallback uses full slug instead of repo name in workflow_call relays](https://github.com/github/gh-aw/issues/20821) _(direct issue)_
- [`dispatch-workflow` uses caller's `GITHUB_REF` for cross-repo dispatch instead of target repo's default branch](https://github.com/github/gh-aw/issues/20779) _(direct issue)_
- [Bug: Activation checkout does not preserve callee workflow ref in caller-hosted relays](https://github.com/github/gh-aw/issues/20697) _(direct issue)_
- [Bug: `dispatch_workflow` ignores `target-repo` and dispatches to `context.repo` in cross-repo relays](https://github.com/github/gh-aw/issues/20694) _(direct issue)_
- [Bug: `Checkout actions folder` emitted without `repository:` or `ref:` — `Setup Scripts` fails in cross-repo relay](https://github.com/github/gh-aw/issues/20658) _(direct issue)_
- [Cross-repo activation checkout still broken for event-driven relay workflows after #20301](https://github.com/github/gh-aw/issues/20567) _(direct issue)_

### @joperezr

- [cache-memory: GH_AW_WORKFLOW_ID_SANITIZED not defined in update_cache_memory job](https://github.com/github/gh-aw/issues/17243) _(direct issue)_

### @JoshGreenslade

- [gh-aw not working in cloud enterprise environments](https://github.com/github/gh-aw/issues/18480) _(direct issue)_
- [compiled agentic workflows require modification when in a enterprise cloud environment](https://github.com/github/gh-aw/issues/16312) _(direct issue)_

### @kbreit-insight

- [gh aw new safe-outputs are not always valid](https://github.com/github/gh-aw/issues/21978) _(direct issue)_

### @KGoovaer

- [Workflows fail with 'Copilot is not a user' error on agent-created PRs](https://github.com/github/gh-aw/issues/18556) _(direct issue)_

### @Krzysztof-Cieslak

- [Add issue type add/remove safe output](https://github.com/github/gh-aw/issues/18488) _(direct issue)_

### @lupinthe14th

- [Copilot CLI does not recognize HTTP-based custom MCP server tools despite successful gateway connection](https://github.com/github/gh-aw/issues/18712) _(direct issue)_

### @mark-hingston

- [copilot-requests property](https://github.com/github/gh-aw/issues/20335) _(direct issue)_

### @mason-tim

- [Enterprise blocker: create-pull-request safe output fails with org-level required_signatures ruleset](https://github.com/github/gh-aw/issues/21562) _(direct issue)_
- [`assign-to-agent` fails with GitHub App tokens — Copilot assignment API requires a PAT](https://github.com/github/gh-aw/issues/19765) _(direct issue)_

### @MatthewLabasan-NBCU

- [Bug: gh-aw compile incorrectly prepends repository name to #runtime-import paths in .github repositories](https://github.com/github/gh-aw/issues/19500) _(direct issue)_

### @MattSkala

- [Allow conditional trigger filtering without failing workflow runs](https://github.com/github/gh-aw/issues/21203) _(direct issue)_

### @maxbeizer

- [`gh aw trial` fails with 404 — missing `.github/` prefix in workflow path resolution](https://github.com/github/gh-aw/issues/18875) _(direct issue)_

### @mcantrell

- [Option to skip API secret prompt for `add-wizard`](https://github.com/github/gh-aw/issues/20592) _(direct issue)_

### @microsasa

- [check_membership.cjs error branch short-circuits before bot allowlist fallback](https://github.com/github/gh-aw/issues/21098) _(direct issue)_
- [The agent cannot close PRs even though the frontmatter explicitly configures it](https://github.com/github/gh-aw/issues/20851) _(direct issue)_
- [pre_activation role check fails for workflow_run events (should use workflow-based trust)](https://github.com/github/gh-aw/issues/20586) _(direct issue)_

### @mnkiefer

- [[research] Overview of docs improver agents](https://github.com/github/gh-aw/issues/19836) _(direct issue)_

### @molson504x

- [Possible regression bug - safe-outputs fails on uploading artifacts](https://github.com/github/gh-aw/issues/21834) _(direct issue)_
- [How does this work on GH ARC?](https://github.com/github/gh-aw/issues/21615) _(direct issue)_

### @Mossaka

- [Support sparse-checkout in compiled workflows for large monorepos](https://github.com/github/gh-aw/issues/21630) _(direct issue)_

### @mstrathman

- [ARM64 container images not available for gh-aw firewall/MCP gateway](https://github.com/github/gh-aw/issues/16005) _(direct issue)_

### @mvdbos

- [Feature Request: `call-workflow` safe output for `workflow_call` chaining](https://github.com/github/gh-aw/issues/20411) _(direct issue)_
- [Feature Request: Cross-repo `workflow_call` validation and docs](https://github.com/github/gh-aw/issues/20249) _(direct issue)_

### @NicoAvanzDev

- [push-to-pull-request-branch safe-output fails with "Cannot generate incremental patch" due to shallow checkout](https://github.com/github/gh-aw/issues/21542) _(direct issue)_
- [push_to_pull_request_branch: git fetch still fails after clean_git_credentials.sh (v0.53.3)](https://github.com/github/gh-aw/issues/20540) _(direct issue)_
- [push-to-pull-request-branch defaults to max: 0 instead of documented default max: 1](https://github.com/github/gh-aw/issues/20528) _(direct issue)_

### @Nikhil-Anand-DSG

- [`hide-older-comments` on `add-comment` safe output finds no matching comments despite correct `workflow_id` marker](https://github.com/github/gh-aw/issues/18200) _(direct issue)_

### @pholleran

- [checkout: false still emits 'Configure Git credentials' steps that fail without .git](https://github.com/github/gh-aw/issues/21313) _(direct issue)_

### @Phonesis

- [Playwright MCP tools not available in GitHub Agentic Workflows (initialize: EOF during MCP init)](https://github.com/github/gh-aw/issues/16236) _(direct issue)_

### @pmalarme

- [`add-reviewer` safe-output handler not loaded at runtime — message skipped with warning](https://github.com/github/gh-aw/issues/16642) _(direct issue)_

### @ppusateri

- [submit_pull_request_review safe output: review context lost during finalization — review never submitted](https://github.com/github/gh-aw/issues/16587) _(direct issue)_

### @praveenkuttappan

- [Copilot workflow steps cannot access Azure/Azure DevOps APIs after azure/login@v2](https://github.com/github/gh-aw/issues/18386) _(direct issue)_
- [Feature request to support GitHub app-based authentication for copilot requests](https://github.com/github/gh-aw/issues/18379) _(direct issue)_

### @qwert666

- [Update status field in Github Project](https://github.com/github/gh-aw/issues/18162) _(direct issue)_

### @rabo-unumed

- [`gh aw logs` requests unsupported `path` JSON field from `gh run list`](https://github.com/github/gh-aw/issues/20679) _(direct issue)_

### @racedale

- [Fix support for custom named COPILOT_GITHUB_TOKEN secret](https://github.com/github/gh-aw/issues/17982) _(direct issue)_

### @rafael-unloan

- [How to install yarn?](https://github.com/github/gh-aw/issues/11190) _(direct issue)_

### @rmarinho

- [Feature Request: Add support for status checks as integration points for ci-doctor](https://github.com/github/gh-aw/issues/16555) _(direct issue)_

### @rspurgeon

- [Bug: `gh aw upgrade` does not set a sha for `setup-cli` in `copilot-setup-steps.yml`](https://github.com/github/gh-aw/issues/19451) _(direct issue)_
- [`gh aw compile` consistent actions/setup sha generation](https://github.com/github/gh-aw/issues/18373) _(direct issue)_
- [safe-outputs create-discussion does not apply configured labels](https://github.com/github/gh-aw/issues/15595) _(direct issue)_

### @samuelkahessay

- [Release community attribution silently misses valid fixes when resolution flows through follow-up issues](https://github.com/github/gh-aw/issues/22138) _(direct issue)_
- [GitHub App auth exempts public repos from automatic min-integrity protection](https://github.com/github/gh-aw/issues/21955) _(direct issue)_
- [workflow_dispatch targeted issue binding ignored — agent never reads bound issue](https://github.com/github/gh-aw/issues/21501) _(direct issue)_
- [Built safe-outputs prompt says to use safeoutputs for all GitHub operations](https://github.com/github/gh-aw/issues/21304) _(direct issue)_
- [safe-outputs: handler failures computed in failureCount but never escalated to core.setFailed()](https://github.com/github/gh-aw/issues/20035) _(direct issue)_
- [`dispatch-workflow` validation is compile-order dependent](https://github.com/github/gh-aw/issues/20031) _(direct issue)_
- [`on.bots` matching is exact-string only and fails for `<slug>` vs `<slug>[bot]` GitHub App identities](https://github.com/github/gh-aw/issues/20030) _(direct issue)_
- [`handle_create_pr_error`: unhandled exceptions on API calls crash conclusion job](https://github.com/github/gh-aw/issues/19605) _(direct issue)_
- [push_repo_memory.cjs has no retry/backoff, fails on concurrent pushes](https://github.com/github/gh-aw/issues/19476) _(direct issue)_
- [get_current_branch.cjs leaks stderr when not in a git repository](https://github.com/github/gh-aw/issues/19475) _(direct issue)_
- [Unconditional agent-output artifact download causes ENOENT noise on pre-agent failures](https://github.com/github/gh-aw/issues/19474) _(direct issue)_
- [Copilot engine fallback model path uses --model CLI flag instead of COPILOT_MODEL env var](https://github.com/github/gh-aw/issues/19473) _(direct issue)_
- [`gh aw checks --json` collapses optional third-party failures into top-level state](https://github.com/github/gh-aw/issues/19158) _(direct issue)_
- [Malformed #aw_* references in body text pass through without validation](https://github.com/github/gh-aw/issues/19024) _(direct issue)_
- [Mixed-trigger workflows collapse workflow_dispatch runs into degenerate concurrency group](https://github.com/github/gh-aw/issues/19023) _(direct issue)_
- [Auto-merge gating has no way to ignore non-required third-party deployment statuses](https://github.com/github/gh-aw/issues/19020) _(direct issue)_
- [EACCES on /tmp/gh-aw/mcp-logs — no ownership repair between workflow runs](https://github.com/github/gh-aw/issues/19018) _(direct issue)_
- [Permanently deferred safe-output items do not fail the workflow](https://github.com/github/gh-aw/issues/19017) _(direct issue)_

### @samus-aran

- [Add broken redirects for these patterns pages](https://github.com/github/gh-aw/issues/18468) _(direct issue)_

### @srgibbs99

- [Bug: `gh aw upgrade` wraps `uses` value in quotes, including the inline comment](https://github.com/github/gh-aw/issues/19640) _(direct issue)_
- [Bug: `gh aw upgrade` and `gh aw compile` produce different lock files — toggle endlessly](https://github.com/github/gh-aw/issues/19622) _(direct issue)_
- [Bug Report: `safeoutputs` MCP server crashes with `context is not defined` on `create_pull_request`](https://github.com/github/gh-aw/issues/18751) _(direct issue)_
- [Bug: \| block scalar description in safe-inputs breaks generated Python script](https://github.com/github/gh-aw/issues/18745) _(direct issue)_
- [HTML in `update-issue` body gets escaped/mangled](https://github.com/github/gh-aw/issues/17298) _(direct issue)_

### @steliosfran

- [[bug] base-branch in assign-to-agent uses customInstructions text instead of GraphQL baseRef field](https://github.com/github/gh-aw/issues/17299) _(direct issue)_
- [[enhancement] Add base-branch support to assign-to-agent safe output for cross-repo PR creation](https://github.com/github/gh-aw/issues/17046) _(direct issue)_
- [Allow `assign-to-agent` safe output to select the repo that the PR should be created in](https://github.com/github/gh-aw/issues/16280) _(direct issue)_

### @straub

- [`gh aw upgrade` Reformats `copilot-setup-steps`](https://github.com/github/gh-aw/issues/19631) _(direct issue)_

### @strawgate

- [sandbox.mcp.payloadSizeThreshold is ignored during frontmatter extraction](https://github.com/github/gh-aw/issues/21135) _(direct issue)_
- [Feature: support explicit custom key for close-older matching](https://github.com/github/gh-aw/issues/21028) _(direct issue)_
- [workflow_call safe_outputs can download unprefixed agent artifact name](https://github.com/github/gh-aw/issues/20910) _(direct issue)_
- [safe-outputs: create_pull_request_review_comment does not treat pull_request_target as PR context](https://github.com/github/gh-aw/issues/20259) _(direct issue)_
- [safe-outputs: target="triggering" rejects pull_request_target PR context](https://github.com/github/gh-aw/issues/20168) _(direct issue)_
- [safe_outputs: created_issue_* outputs missing because emitter is never called](https://github.com/github/gh-aw/issues/20125) _(direct issue)_
- [Agent sandbox git identity missing: first commit fails, then agent self-configures](https://github.com/github/gh-aw/issues/20033) _(direct issue)_
- [close-older-issues closes issues from different calling workflows](https://github.com/github/gh-aw/issues/19172) _(direct issue)_
- [submit_pull_request_review: REQUEST_CHANGES/APPROVE fails on own PR despite override check](https://github.com/github/gh-aw/issues/18945) _(direct issue)_
- [Replace format-patch/git-am pipeline with tree diff + GraphQL commit API](https://github.com/github/gh-aw/issues/18900) _(direct issue)_
- [feat: add target config to resolve-pull-request-review-thread](https://github.com/github/gh-aw/issues/18744) _(direct issue)_
- [Commits via `git` are unverified; switch to GraphQL for commits](https://github.com/github/gh-aw/issues/18565) _(direct issue)_
- [Check-out from Fork does not work with workflow_call](https://github.com/github/gh-aw/issues/18563) _(direct issue)_
- [safe_outputs checkout fails for pull_request_review events](https://github.com/github/gh-aw/issues/18547) _(direct issue)_
- [bug: duplicate env vars when import and main workflow reference the same repository variable](https://github.com/github/gh-aw/issues/18545) _(direct issue)_
- [create_pull_request fails with large commits](https://github.com/github/gh-aw/issues/18501) _(direct issue)_
- [Safe Output custom token source](https://github.com/github/gh-aw/issues/18362) _(direct issue)_
- [fix: imported safe-output fragments override explicit threat-detection: false](https://github.com/github/gh-aw/issues/18226) _(direct issue)_
- [Retry downloads automatically](https://github.com/github/gh-aw/issues/17839) _(direct issue)_
- [Feature request: add flag to disable activation/fallback comments](https://github.com/github/gh-aw/issues/17828) _(direct issue)_
- [update-pull-request should honor footer: false](https://github.com/github/gh-aw/issues/17522) _(direct issue)_
- [Add safe-output fail-fast mode for code push operations](https://github.com/github/gh-aw/issues/17521) _(direct issue)_
- [Customize Checkout Depth](https://github.com/github/gh-aw/issues/16896) _(direct issue)_
- [add-comment doesn't actually require `pull_requests: write`](https://github.com/github/gh-aw/issues/16673) _(direct issue)_
- [Support repository-local `mcp.json`](https://github.com/github/gh-aw/issues/16664) _(direct issue)_
- [Add `inline-prompt` option to compile workflows without runtime-import macros](https://github.com/github/gh-aw/issues/16511) _(direct issue)_
- [Nested remote imports resolve against hardcoded .github/workflows/ instead of parent workflowspec base path](https://github.com/github/gh-aw/issues/16370) _(direct issue)_
- [Add fuzzy scheduling for running on weekdays](https://github.com/github/gh-aw/issues/16036) _(direct issue)_
- [Nested local path imports in remote workflows should resolve](https://github.com/github/gh-aw/issues/15982) _(direct issue)_
- [add-comment tool enforces rules during safe-outputs instead of during call](https://github.com/github/gh-aw/issues/15976) _(direct issue)_
- [Create a comment with a link to create a pull request](https://github.com/github/gh-aw/issues/15836) _(direct issue)_
- [Consider dedicated setting for PR Review Footer w/o Body](https://github.com/github/gh-aw/issues/15583) _(direct issue)_
- [Add safe outputs for replying to and resolving pull request review comments](https://github.com/github/gh-aw/issues/15576) _(direct issue)_

### @swimmesberger

- [feat: allow configuring the token used for pre-activation reactions](https://github.com/github/gh-aw/issues/19421) _(direct issue)_

### @theletterf

- [safe_output_handler_manager ignores allowed-domains, redacts URLs from allowlisted domains](https://github.com/github/gh-aw/issues/18465) _(direct issue)_

### @timdittler

- [push-to-pull-request-branch safe output unconditionally requests issues: write](https://github.com/github/gh-aw/issues/16331) _(direct issue)_
- [create-pull-request: reviewers config not compiled into handler config](https://github.com/github/gh-aw/issues/16117) _(direct issue)_
- [App token for safe-outputs doesn't work](https://github.com/github/gh-aw/issues/16107) _(direct issue)_

### @tore-unumed

- [create-pull-request: allow disabling branch name sanitization (lowercase + salt suffix)](https://github.com/github/gh-aw/issues/20780) _(direct issue)_
- [Cross-repo create-pull-request fails: GITHUB_TOKEN not available for dynamic checkout](https://github.com/github/gh-aw/issues/19370) _(direct issue)_
- [How to create PRs in multiple repos from a single workflow?](https://github.com/github/gh-aw/issues/18329) _(direct issue)_
- [safe-outputs create_pull_request fails for cross-repo checkouts: uses GITHUB_SHA from workflow repo as merge base](https://github.com/github/gh-aw/issues/18107) _(direct issue)_
- [create-pull-request safe output fails with "No changes to commit" when workspace is a cross-repo checkout](https://github.com/github/gh-aw/issues/17289) _(direct issue)_

### @tspascoal

- [When PR creation is not created due to a fallback agent still claims the PR was created](https://github.com/github/gh-aw/issues/20597) _(direct issue)_
- [Mermaid flowchart node multiline text is not rendered correctly in the documentation](https://github.com/github/gh-aw/issues/18123) _(direct issue)_

### @UncleBats

- [safe-outputs.create-pull-request.draft: false is ignored when agent specifies draft: true](https://github.com/github/gh-aw/issues/20359) _(direct issue)_

### @veverkap

- [Bug: Grumpy Code Wants GH_AW_GITHUB_TOKEN](https://github.com/github/gh-aw/issues/21260) _(direct issue)_
- [Feature Request: Modify PR before creation](https://github.com/github/gh-aw/issues/21257) _(direct issue)_

### @ViktorHofer

- [shell(dotnet) tool denied despite being in allowed tools — requires 'env dotnet' workaround](https://github.com/github/gh-aw/issues/18340) _(direct issue)_
- [gh aw compile does not add pull-requests: write to safe_outputs job when add-comment is configured](https://github.com/github/gh-aw/issues/18311) _(direct issue)_

### @whoschek

- [Create a workflow that bills the Codex subscription instead of API key](https://github.com/github/gh-aw/issues/15510) _(direct issue)_

</details>
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
