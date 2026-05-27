<!--

Hello fellow agent! Welcome to GitHub Agentic Workflows = Actions + Agent + Safety. 
Here are some pointers to get you started in using this tool.

- Create a new workflow: https://raw.githubusercontent.com/github/gh-aw/main/create.md
- Install: https://raw.githubusercontent.com/github/gh-aw/main/install.md
- Package a workflow repository: https://raw.githubusercontent.com/github/gh-aw/main/package.md
- Reference: https://raw.githubusercontent.com/github/gh-aw/main/.github/aw/github-agentic-workflows.md

-->

# GitHub Agentic Workflows

Write agentic workflows in natural language markdown, and run them in GitHub Actions.

> [!NOTE]
> **Releases 0.68.4 through 0.71.3 are being retired** due to a bug that impacts billing. If you are running one of these versions, please upgrade to the latest release as soon as possible.

## Contents

- [Quick Start](#quick-start)
- [Overview](#overview)
- [Guardrails](#guardrails)
- [Documentation](#documentation)
- [FAQ](#faq)
- [Contributing](#contributing)
- [Community Contributions](#-community-contributions)
- [Share Feedback](#share-feedback)
- [Peli's Agent Factory](#pelis-agent-factory)
- [Related Projects](#related-projects)

## Quick Start

Ready to get your first agentic workflow running? Follow our step-by-step [Quick Start Guide](https://github.github.com/gh-aw/setup/quick-start/) to install the extension, add a sample workflow, and see it in action.

## Overview

Learn about the concepts behind agentic workflows, explore available workflow types, and understand how AI can automate your repository tasks. See [How It Works](https://github.github.com/gh-aw/introduction/how-they-work/).
Supports GitHub Copilot, Claude (Anthropic), Codex (OpenAI), and Gemini (Google) — pick whichever AI account you already have.

## Guardrails

Guardrails, safety and security are foundational to GitHub Agentic Workflows. Workflows run with read-only permissions by default, with write operations only allowed through sanitized `safe-outputs`. The system implements multiple layers of protection including sandboxed execution, input sanitization, network isolation, supply chain security (SHA-pinned dependencies), tool allow-listing, and compile-time validation. Access can be gated to team members only, with human approval gates for critical operations, ensuring AI agents operate safely within controlled boundaries. See the [Security Architecture](https://github.github.com/gh-aw/introduction/architecture/) for comprehensive details on threat modeling, implementation guidelines, and best practices.

Using agentic workflows in your repository requires careful attention to security considerations and careful human supervision, and even then things can still go wrong. Use it with caution, and at your own risk.

## Documentation

For complete documentation, examples, and guides, see the [Documentation](https://github.github.com/gh-aw/). If you are an agent, download [llms.txt](https://github.github.com/gh-aw/llms.txt) or the full corpus [llms-full.txt](https://github.github.com/gh-aw/llms-full.txt).

If you are running a version between 0.68.4 and 0.71.3, upgrading is strongly recommended due to a bug that impacts billing.

## Contributing

For development setup and contribution guidelines, see [CONTRIBUTING.md](CONTRIBUTING.md).

### Custom Go linters

To build and test repository custom linters:

- `go test ./pkg/linters/<linter-name>/...`
- `go build ./cmd/linters`
- `make golint-custom`

`make golint-custom` builds `cmd/linters` and runs the custom analyzers against `./cmd/...` and `./pkg/...`.

## 🌍 Community Contributions

<details>
<summary>Thank you to the community members whose issue reports were resolved in this project! This list is updated automatically and reflects all attributed contributions.</summary>

- @abillingsley: #23736 _(direct issue)_
- @adamhenson: #25345 _(direct issue)_, #24282 _(direct issue)_
- @ahmadabdalla: #27473 _(direct issue)_
- @ajfeldman6: #23924 _(direct issue)_
- @AkshatRaj00: #34302 _(direct issue)_
- @AlexDeMichieli: #26645 _(direct issue)_
- @alexsiilvaa: #20781 _(direct issue)_, #20664 _(direct issue)_
- @alondahari: #21207 _(direct issue)_
- @anthonymastreanvae: #32481 _(direct issue)_, #32479 _(direct issue)_, #30897 _(direct issue)_, #30841 _(direct issue)_
- @apenab: #25626 _(direct issue)_
- @app/github-actions: #31288 _(direct issue)_, #30740 _(direct issue)_, #29561 _(direct issue)_, #29343 _(direct issue)_, #26257 _(direct issue)_, #26256 _(direct issue)_, #26255 _(direct issue)_, #26254 _(direct issue)_, #26253 _(direct issue)_
- @arezero: #20515 _(direct issue)_, #20514 _(direct issue)_, #20513 _(direct issue)_, #20512 _(direct issue)_, #20511 _(direct issue)_, #20510 _(direct issue)_
- @arthurfvives: #30356 _(direct issue)_, #30088 _(direct issue)_, #26223 _(direct issue)_, #25993 _(direct issue)_, #25294 _(direct issue)_
- @askpaisa: #29240 _(direct issue)_
- @b2pacific: #28720 _(direct issue)_
- @bartul: #29499 _(direct issue)_
- @bbonafed: #29174 _(direct issue)_, #29173 _(direct issue)_, #29172 _(direct issue)_, #29171 _(direct issue)_, #27670 _(direct issue)_, #27472 _(direct issue)_, #26719 _(direct issue)_, #26045 _(direct issue)_, #26043 _(direct issue)_, #25646 _(direct issue)_, #25224 _(direct issue)_, #24949 _(direct issue)_, #24918 _(direct issue)_, #24896 _(direct issue)_, #24323 _(direct issue)_, #23900 _(direct issue)_, #23724 _(direct issue)_, #23566 _(direct issue)_, #22564 _(direct issue)_, #21990 _(direct issue)_, #20801 _(direct issue)_
- @benvillalobos: #25717 _(direct issue)_, #20885 _(direct issue)_
- @bmerkle: #31689 _(direct issue)_, #26621 _(direct issue)_, #20646 _(direct issue)_
- @bryanchen-d: #34109 _(direct issue)_, #34108 _(direct issue)_, #30866 _(direct issue)_, #30704 _(direct issue)_, #30695 _(direct issue)_, #30472 _(direct issue)_, #28774 _(direct issue)_, #26696 _(direct issue)_, #26487 _(direct issue)_, #25719 _(direct issue)_, #23265 _(direct issue)_
- @bryanknox: #25351 _(direct issue)_
- @Calidus: #33188 _(direct issue)_, #26923 _(direct issue)_
- @camposbrunocampos: #23726 _(direct issue)_, #22897 _(direct issue)_
- @carlincherry: #22017 _(direct issue)_
- @chrisfregly: #25349 _(direct issue)_, #23963 _(direct issue)_
- @chrizbo: #34980 _(direct issue)_, #32446 _(direct issue)_, #31399 _(direct issue)_, #28158 _(direct issue)_, #22510 _(direct issue)_, #21863 _(direct issue)_
- @clementbolin: #28888 _(direct issue)_
- @cogni-ai-ee: #32803 _(direct issue)_, #32741 _(direct issue)_
- @Corb3nik: #21306 _(direct issue)_
- @corygehr: #33622 _(direct issue)_, #33436 _(direct issue)_, #33367 _(direct issue)_, #31577 _(direct issue)_, #27638 _(direct issue)_, #26539 _(direct issue)_, #26270 _(direct issue)_, #26268 _(direct issue)_, #25680 _(direct issue)_, #24355 _(direct issue)_, #23944 _(direct issue)_, #23753 _(direct issue)_
- @dagecko: #24743 _(direct issue)_
- @Daidanny008: #27402 _(direct issue)_
- @Dan-Co: #22707 _(direct issue)_
- @danielmeppiel: #29076 _(direct issue)_, #28678 _(direct issue)_, #20663 _(direct issue)_
- @danquirk: #30403 _(direct issue)_
- @dbudym-cs: #22913 _(direct issue)_
- @devantler: #25768 _(direct issue)_, #25767 _(direct issue)_
- @deyaaeldeen: #28966 _(direct issue)_, #26486 _(direct issue)_, #25573 _(direct issue)_, #25359 _(direct issue)_, #23198 _(direct issue)_, #23024 _(direct issue)_, #23020 _(direct issue)_, #22957 _(direct issue)_
- @dfrysinger: #34886 _(direct issue)_, #34885 _(direct issue)_
- @dholmes: #29228 _(direct issue)_, #23578 _(direct issue)_
- @DimaBir: #20483 _(direct issue)_
- @dkurepa: #25511 _(direct issue)_
- @DogeAmazed: #22703 _(direct issue)_
- @doughgle: #23655 _(direct issue)_
- @drehelis: #25304 _(direct issue)_
- @dsyme: #23936 _(direct issue)_, #22340 _(direct issue)_, #20953 _(direct issue)_, #20952 _(direct issue)_, #20950 _(direct issue)_, #20787 _(direct issue)_, #20578 _(direct issue)_, #20420 _(direct issue)_
- @duncankmckinnon: #25944 _(direct issue)_
- @eaftan: #23257 _(direct issue)_, #20457 _(direct issue)_
- @edburns: #26920 _(direct issue)_
- @edgeq: #28315 _(direct issue)_, #28308 _(direct issue)_
- @ericstj: #30260 _(direct issue)_, #23766 _(direct issue)_
- @ferryhinardi: #24128 _(direct issue)_
- @flatiron32: #22469 _(direct issue)_
- @fr4nc1sc0-r4m0n: #20657 _(direct issue)_
- @GandrotulaRajesh: #33981 _(direct issue)_
- @glitch-ux: #24403 _(direct issue)_
- @grahame-white: #23643 _(direct issue)_, #23093 _(direct issue)_, #23092 _(direct issue)_, #23088 _(direct issue)_, #23083 _(direct issue)_, #20868 _(direct issue)_, #20719 _(direct issue)_, #20629 _(direct issue)_
- @h3y6e: #27794 _(direct issue)_
- @haavamoa: #30191 _(direct issue)_
- @hermanho: #32197 _(direct issue)_
- @holwerda: #21243 _(direct issue)_
- @IEvangelist: #34998 _(direct issue)_, #33285 _(direct issue)_, #33069 _(direct issue)_, #33068 _(direct issue)_, #33060 _(direct issue)_, #33043 _(direct issue)_, #32536 _(direct issue)_, #32354 _(direct issue)_, #30848 _(direct issue)_, #26908 _(direct issue)_, #25467 _(direct issue)_
- @Infinnerty: #21957 _(direct issue)_
- @insop: #21686 _(direct issue)_
- @j-srodka: #25199 _(direct issue)_, #23485 _(direct issue)_, #23484 _(direct issue)_, #23483 _(direct issue)_, #23482 _(direct issue)_, #23461 _(direct issue)_
- @jamesadevine: #28957 _(direct issue)_, #26407 _(direct issue)_, #26406 _(direct issue)_
- @JamesNK: #29310 _(direct issue)_, #28867 _(direct issue)_, #28704 _(direct issue)_
- @JanKrivanek: #25656 _(direct issue)_, #25439 _(direct issue)_
- @jaroslawgajewski: #34917 _(direct issue)_, #33644 _(direct issue)_, #33640 _(direct issue)_, #31678 _(direct issue)_, #31658 _(direct issue)_, #25593 _(direct issue)_, #24373 _(direct issue)_, #24372 _(direct issue)_, #24371 _(direct issue)_, #24259 _(direct issue)_, #24036 _(direct issue)_, #23779 _(direct issue)_, #23558 _(direct issue)_, #22647 _(direct issue)_, #21816 _(direct issue)_, #20813 _(direct issue)_, #20811 _(direct issue)_
- @JasonYeMSFT: #27424 _(direct issue)_
- @jbaruch: #30832 _(direct issue)_
- @jeffhandley: #30232 _(direct issue)_, #30204 _(direct issue)_, #26799 _(direct issue)_, #26788 _(direct issue)_, #24384 _(direct issue)_
- @jfomhover: #25420 _(direct issue)_
- @jitran: #33649 _(direct issue)_
- @johnpreed: #25687 _(direct issue)_, #23777 _(direct issue)_, #23212 _(direct issue)_, #21334 _(direct issue)_
- @johnwilliams-12: #21205 _(direct issue)_, #21074 _(direct issue)_, #21071 _(direct issue)_, #21062 _(direct issue)_, #20821 _(direct issue)_, #20779 _(direct issue)_, #20697 _(direct issue)_, #20694 _(direct issue)_, #20658 _(direct issue)_, #20567 _(direct issue)_, #20508 _(direct issue)_
- @jonathanpeppers: #30662 _(direct issue)_
- @jsoref: #27230 _(direct issue)_
- @jtracey93: #26176 _(direct issue)_
- @kaovilai: #32596 _(direct issue)_, #32587 _(direct issue)_, #32482 _(direct issue)_, #32467 _(direct issue)_
- @kbreit-insight: #24930 _(direct issue)_, #23940 _(direct issue)_, #23725 _(direct issue)_, #22430 _(direct issue)_, #21978 _(direct issue)_
- @kkruel8100: #30867 _(direct issue)_
- @kthompson: #25550 _(direct issue)_
- @labudis: #30846 _(direct issue)_
- @ladamski: #33641 _(direct issue)_
- @lindeberg: #34006 _(direct issue)_
- @look: #23258 _(direct issue)_
- @lpcox: #30634 _(direct issue)_, #29353 _(direct issue)_, #29191 _(direct issue)_, #22281 _(direct issue)_
- @lukeed: #20552 _(direct issue)_
- @lupinthe14th: #26542 _(direct issue)_, #26441 _(direct issue)_
- @mason-tim: #33084 _(direct issue)_, #33074 _(direct issue)_, #31489 _(direct issue)_, #30336 _(direct issue)_, #29301 _(direct issue)_, #21562 _(direct issue)_
- @MatthewLabasan-NBCU: #26289 _(direct issue)_
- @MattSkala: #24567 _(direct issue)_, #21203 _(direct issue)_
- @MauroDruwel: #30178 _(direct issue)_, #30169 _(direct issue)_, #29379 _(direct issue)_, #29378 _(direct issue)_
- @mcantrell: #20592 _(direct issue)_
- @mdashrraf: #28657 _(direct issue)_
- @MH0386: #20997 _(direct issue)_
- @mhavelock: #22110 _(direct issue)_
- @michen00: #31869 _(direct issue)_
- @microsasa: #27715 _(direct issue)_, #21103 _(direct issue)_, #21098 _(direct issue)_, #20851 _(direct issue)_, #20833 _(direct issue)_, #20586 _(direct issue)_
- @mlinksva: #22533 _(direct issue)_
- @mnkiefer: #22409 _(direct issue)_
- @molson504x: #21834 _(direct issue)_, #21615 _(direct issue)_
- @Mossaka: #21644 _(direct issue)_, #21630 _(direct issue)_
- @mrjf: #32271 _(direct issue)_, #32069 _(direct issue)_, #31600 _(direct issue)_, #29152 _(direct issue)_, #28955 _(direct issue)_, #28471 _(direct issue)_, #28197 _(direct issue)_
- @neta-vega: #26447 _(direct issue)_, #25895 _(direct issue)_
- @NicoAvanzDev: #21542 _(direct issue)_, #20540 _(direct issue)_, #20528 _(direct issue)_
- @NicolasRannou: #31701 _(direct issue)_
- @NikolajBjorner: #28812 _(direct issue)_
- @norrietaylor: #33199 _(direct issue)_, #32312 _(direct issue)_, #32310 _(direct issue)_, #30733 _(direct issue)_, #30392 _(direct issue)_
- @octatone: #31918 _(direct issue)_
- @petercort: #28281 _(direct issue)_
- @pethers: #28470 _(direct issue)_
- @pgaskin: #26156 _(direct issue)_
- @pholleran: #25313 _(direct issue)_, #23572 _(direct issue)_, #21313 _(direct issue)_
- @polmichel: #32991 _(direct issue)_
- @PureWeen: #28767 _(direct issue)_, #27655 _(direct issue)_, #23769 _(direct issue)_, #23567 _(direct issue)_
- @rabo-unumed: #31660 _(direct issue)_, #31578 _(direct issue)_, #31513 _(direct issue)_, #20679 _(direct issue)_
- @rhardouin: #30840 _(direct issue)_, #30838 _(direct issue)_
- @romainh-betclic: #28143 _(direct issue)_
- @rspurgeon: #26475 _(direct issue)_
- @Rubyj: #31542 _(direct issue)_, #21432 _(direct issue)_
- @ruokun-niu: #24961 _(direct issue)_
- @ryckmansm: #31501 _(direct issue)_
- @salekseev: #25137 _(direct issue)_, #25122 _(direct issue)_, #24135 _(direct issue)_
- @samuelkahessay: #33016 _(direct issue)_, #33015 _(direct issue)_, #24756 _(direct issue)_, #24755 _(direct issue)_, #24754 _(direct issue)_, #22380 _(direct issue)_, #22364 _(direct issue)_, #22161 _(direct issue)_, #22138 _(direct issue)_, #21975 _(direct issue)_, #21955 _(direct issue)_, #21784 _(direct issue)_, #21501 _(direct issue)_, #21304 _(direct issue)_
- @sbodapati-gfm: #29417 _(direct issue)_
- @seangibeault: #26910 _(direct issue)_, #24905 _(direct issue)_
- @sg650: #33787 _(direct issue)_, #32044 _(direct issue)_, #31617 _(direct issue)_, #31616 _(direct issue)_, #29009 _(direct issue)_, #28612 _(direct issue)_
- @shiran-gutsy: #27641 _(direct issue)_
- @srgibbs99: #22939 _(direct issue)_
- @stacktick: #21361 _(direct issue)_
- @stefankrzyz: #27260 _(direct issue)_
- @straub: #24569 _(direct issue)_
- @strawgate: #33597 _(direct issue)_, #24422 _(direct issue)_, #24199 _(direct issue)_, #23935 _(direct issue)_, #23768 _(direct issue)_, #21157 _(direct issue)_, #21144 _(direct issue)_, #21135 _(direct issue)_, #21028 _(direct issue)_, #20910 _(direct issue)_
- @susmahad: #26276 _(direct issue)_, #25866 _(direct issue)_, #25710 _(direct issue)_
- @szabta89: #29064 _(direct issue)_, #29063 _(direct issue)_, #24037 _(direct issue)_
- @tadelesh: #26001 _(direct issue)_
- @theletterf: #33963 _(direct issue)_, #32846 _(direct issue)_, #30964 _(direct issue)_, #30365 _(direct issue)_, #30327 _(direct issue)_, #28898 _(direct issue)_, #28895 _(direct issue)_, #28691 _(direct issue)_, #28672 _(direct issue)_, #28221 _(direct issue)_, #27566 _(direct issue)_, #25494 _(direct issue)_
- @thi-feonir: #21426 _(direct issue)_
- @tinytelly: #27282 _(direct issue)_
- @tore-unumed: #33545 _(direct issue)_, #31909 _(direct issue)_, #31650 _(direct issue)_, #30550 _(direct issue)_, #30324 _(direct issue)_, #29312 _(direct issue)_, #28019 _(direct issue)_, #20780 _(direct issue)_
- @trask: #31612 _(direct issue)_, #31241 _(direct issue)_, #31098 _(direct issue)_, #31097 _(direct issue)_
- @tsm-harmoney: #31695 _(direct issue)_, #27880 _(direct issue)_
- @tspascoal: #20597 _(direct issue)_
- @tvu4-wowcorp: #34556 _(direct issue)_
- @verkyyi: #27407 _(direct issue)_, #27259 _(direct issue)_
- @veverkap: #22362 _(direct issue)_, #21260 _(direct issue)_, #21257 _(direct issue)_
- @virenpepper: #23765 _(direct issue)_
- @wtgodbe: #32834 _(direct issue)_, #26057 _(direct issue)_, #25130 _(direct issue)_, #24921 _(direct issue)_
- @yaananth: #24125 _(direct issue)_
- @yskopets: #34772 _(direct issue)_, #34256 _(direct issue)_, #34250 _(direct issue)_, #34166 _(direct issue)_, #34134 _(direct issue)_, #32022 _(direct issue)_, #31831 _(direct issue)_, #31086 _(direct issue)_, #31073 _(direct issue)_, #30872 _(direct issue)_, #30705 _(direct issue)_, #27935 _(direct issue)_, #27898 _(direct issue)_, #27881 _(direct issue)_, #27773 _(direct issue)_, #27757 _(direct issue)_, #26922 _(direct issue)_, #26569 _(direct issue)_, #26468 _(direct issue)_, #26358 _(direct issue)_, #26346 _(direct issue)_, #26345 _(direct issue)_, #26280 _(direct issue)_, #26279 _(direct issue)_, #26120 _(direct issue)_, #26101 _(direct issue)_, #26085 _(direct issue)_, #26080 _(direct issue)_, #26067 _(direct issue)_, #25959 _(direct issue)_, #25946 _(direct issue)_, #25833 _(direct issue)_, #25363 _(direct issue)_, #25362 _(direct issue)_, #25125 _(direct issue)_, #24897 _(direct issue)_, #24573 _(direct issue)_, #23914 _(direct issue)_
- @zkoppert: #27741 _(direct issue)_

### ⚠️ Attribution Candidates Need Review

The following community issues were closed during this period but could not be automatically linked to a specific merged PR. Please verify whether they should be credited:

- **@arthurfvives** for [Dry-Run Mode with imports (APM)](https://github.com/github/gh-aw/issues/32974) — closed 2026-05-22, no confirmed PR linkage found
- **@jobayer-4** for [Poor README](https://github.com/github/gh-aw/issues/32608) — closed 2026-05-16, no confirmed PR linkage found
- **@app/github-actions** for [[deep-report] Quick Start sample workflow defaults to Copilot — Claude-only users will fail their first run](https://github.com/github/gh-aw/issues/32423) — closed 2026-05-15, no confirmed PR linkage found
- **@app/github-actions** for [[aw-failures] Claude workflows hit max-turns burning every turn on Bash permission denials (Step Name Alignment, Design Decision Gate)](https://github.com/github/gh-aw/issues/31178) — closed 2026-05-14, no confirmed PR linkage found
- **@JamesNK** for [Hang and timeout while running workflow](https://github.com/github/gh-aw/issues/28868) — closed 2026-04-30, no confirmed PR linkage found

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
