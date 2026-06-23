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

Ready to get your first agentic workflow running? Follow our step-by-step [Quick Start Guide](https://github.com/github/gh-aw/blob/main/docs/src/content/docs/setup/quick-start.mdx) to install the extension, add a sample workflow, and see it in action.

## Overview

Learn about the concepts behind agentic workflows, explore available workflow types, and understand how AI can automate your repository tasks. See [How It Works](https://github.com/github/gh-aw/blob/main/docs/src/content/docs/introduction/how-they-work.mdx).
Supports GitHub Copilot, Claude (Anthropic), Codex (OpenAI), and Gemini (Google) — pick whichever AI account you already have.

## Guardrails

Guardrails, safety and security are foundational to GitHub Agentic Workflows. Workflows run with read-only permissions by default, with write operations only allowed through sanitized `safe-outputs`. The system implements multiple layers of protection including sandboxed execution, input sanitization, network isolation, supply chain security (SHA-pinned dependencies), tool allow-listing, and compile-time validation. Access can be gated to team members only, with human approval gates for critical operations, ensuring AI agents operate safely within controlled boundaries. See the [Security Architecture](https://github.com/github/gh-aw/blob/main/docs/src/content/docs/introduction/architecture.mdx) for comprehensive details on threat modeling, implementation guidelines, and best practices.

Using agentic workflows in your repository requires careful attention to security considerations and careful human supervision, and even then things can still go wrong. Use it with caution, and at your own risk.

## Documentation

For complete documentation, examples, and guides, see the [Documentation](https://github.com/github/gh-aw/tree/main/docs). If you are an agent, see [llms.txt source](https://github.com/github/gh-aw/blob/main/docs/src/pages/llms.txt.ts) and [llms-full.txt source](https://github.com/github/gh-aw/blob/main/docs/src/pages/llms-full.txt.ts).

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

- @adamhenson: #25345 _(direct issue)_, #24282 _(direct issue)_
- @ahmadabdalla: #27473 _(direct issue)_
- @AkshatRaj00: #34302 _(direct issue)_
- @AlexDeMichieli: #26645 _(direct issue)_
- @anthonymastreanvae: #39882 _(direct issue)_, #39792 _(direct issue)_, #32481 _(direct issue)_, #32479 _(direct issue)_, #30897 _(direct issue)_, #30841 _(direct issue)_
- @aoxiangtianyu-go: #39479 _(direct issue)_
- @apenab: #25626 _(direct issue)_
- @app/github-actions: #38226 _(direct issue)_, #31288 _(direct issue)_, #30740 _(direct issue)_, #29561 _(direct issue)_, #29343 _(direct issue)_, #26257 _(direct issue)_, #26256 _(direct issue)_, #26255 _(direct issue)_, #26254 _(direct issue)_, #26253 _(direct issue)_
- @arthurfvives: #35682 _(direct issue)_, #35483 _(direct issue)_, #35157 _(direct issue)_, #30356 _(direct issue)_, #30088 _(direct issue)_, #26223 _(direct issue)_, #25993 _(direct issue)_, #25294 _(direct issue)_
- @Artur-: #39532 _(direct issue)_
- @askpaisa: #29240 _(direct issue)_
- @b2pacific: #28720 _(direct issue)_
- @bartul: #29499 _(direct issue)_
- @bbonafed: #29174 _(direct issue)_, #29173 _(direct issue)_, #29172 _(direct issue)_, #29171 _(direct issue)_, #27670 _(direct issue)_, #27472 _(direct issue)_, #26719 _(direct issue)_, #26045 _(direct issue)_, #26043 _(direct issue)_, #25646 _(direct issue)_, #25224 _(direct issue)_, #24949 _(direct issue)_, #24918 _(direct issue)_, #24896 _(direct issue)_, #24323 _(direct issue)_
- @benissimo: #39237 _(direct issue)_, #38716 _(direct issue)_, #36460 _(direct issue)_, #36234 _(direct issue)_, #36003 _(direct issue)_, #35561 _(direct issue)_
- @benvillalobos: #25717 _(direct issue)_
- @bmerkle: #31689 _(direct issue)_, #26621 _(direct issue)_
- @boydj: #33777 _(direct issue)_, #33605 _(direct issue)_
- @Bra1nFartz: #35192 _(direct issue)_
- @bryanchen-d: #35284 _(direct issue)_, #35283 _(direct issue)_, #35075 _(direct issue)_, #34109 _(direct issue)_, #34108 _(direct issue)_, #30866 _(direct issue)_, #30704 _(direct issue)_, #30695 _(direct issue)_, #30472 _(direct issue)_, #28774 _(direct issue)_, #26696 _(direct issue)_, #26487 _(direct issue)_, #25719 _(direct issue)_
- @bryanknox: #25351 _(direct issue)_
- @Calidus: #38210 _(direct issue)_, #33188 _(direct issue)_, #26923 _(direct issue)_
- @chrisfregly: #25349 _(direct issue)_, #23963 _(direct issue)_
- @chrizbo: #34980 _(direct issue)_, #32446 _(direct issue)_, #31399 _(direct issue)_, #28158 _(direct issue)_
- @CiscoRob: #35032 _(direct issue)_
- @cknight: #39837 _(direct issue)_
- @clementbolin: #28888 _(direct issue)_
- @cogni-ai-ee: #32803 _(direct issue)_, #32741 _(direct issue)_
- @consulthys: #39748 _(direct issue)_, #39670 _(direct issue)_, #39504 _(direct issue)_
- @corygehr: #38681 _(direct issue)_, #38214 _(direct issue)_, #38150 _(direct issue)_, #36969 _(direct issue)_, #36709 _(direct issue)_, #36696 _(direct issue)_, #35272 _(direct issue)_, #33622 _(direct issue)_, #33436 _(direct issue)_, #33367 _(direct issue)_, #33366 _(direct issue)_, #31577 _(direct issue)_, #27638 _(direct issue)_, #26539 _(direct issue)_, #26270 _(direct issue)_, #26268 _(direct issue)_, #25680 _(direct issue)_, #24355 _(direct issue)_, #23944 _(direct issue)_
- @dagecko: #24743 _(direct issue)_
- @Daidanny008: #27402 _(direct issue)_
- @Dan-Albrecht: #33892 _(direct issue)_
- @danielmeppiel: #35688 _(direct issue)_, #29076 _(direct issue)_, #28678 _(direct issue)_
- @danquirk: #30403 _(direct issue)_
- @DeagleGross: #35161 _(direct issue)_
- @devantler: #25768 _(direct issue)_, #25767 _(direct issue)_
- @deyaaeldeen: #28966 _(direct issue)_, #26486 _(direct issue)_, #25573 _(direct issue)_, #25359 _(direct issue)_
- @dfrysinger: #34886 _(direct issue)_, #34885 _(direct issue)_
- @dgolombek: #40775 _(direct issue)_
- @dholmes: #34949 _(direct issue)_, #29228 _(direct issue)_
- @dkurepa: #25511 _(direct issue)_
- @drehelis: #39149 _(direct issue)_, #25304 _(direct issue)_
- @dsibilio: #36196 _(direct issue)_
- @dsyme: #36244 _(direct issue)_, #23936 _(direct issue)_
- @duncankmckinnon: #25944 _(direct issue)_
- @edburns: #26920 _(direct issue)_
- @edgeq: #28315 _(direct issue)_, #28308 _(direct issue)_
- @ericstj: #30260 _(direct issue)_
- @fbecar22: #39514 _(direct issue)_
- @ferryhinardi: #24128 _(direct issue)_
- @flatiron32: #36473 _(direct issue)_
- @GandrotulaRajesh: #33981 _(direct issue)_
- @glitch-ux: #24403 _(direct issue)_
- @h3y6e: #38797 _(direct issue)_, #27794 _(direct issue)_
- @haavamoa: #30191 _(direct issue)_
- @heiskr: #39841 _(direct issue)_
- @hermanho: #32197 _(direct issue)_
- @hpsin: #35611 _(direct issue)_
- @IEvangelist: #34998 _(direct issue)_, #33285 _(direct issue)_, #33069 _(direct issue)_, #33068 _(direct issue)_, #33067 _(direct issue)_, #33060 _(direct issue)_, #33043 _(direct issue)_, #32536 _(direct issue)_, #32354 _(direct issue)_, #30848 _(direct issue)_, #26908 _(direct issue)_, #25467 _(direct issue)_
- @ivancea: #40345 _(direct issue)_
- @j-srodka: #25199 _(direct issue)_
- @jamesadevine: #37806 _(direct issue)_, #28957 _(direct issue)_, #26407 _(direct issue)_, #26406 _(direct issue)_
- @JamesNK: #29310 _(direct issue)_, #28867 _(direct issue)_, #28863 _(direct issue)_, #28704 _(direct issue)_
- @JanKrivanek: #25656 _(direct issue)_, #25439 _(direct issue)_
- @jaroslawgajewski: #34917 _(direct issue)_, #33644 _(direct issue)_, #33640 _(direct issue)_, #31678 _(direct issue)_, #31658 _(direct issue)_, #25593 _(direct issue)_, #24373 _(direct issue)_, #24372 _(direct issue)_, #24371 _(direct issue)_, #24259 _(direct issue)_, #24036 _(direct issue)_
- @JasonYeMSFT: #27424 _(direct issue)_
- @jbaruch: #30832 _(direct issue)_
- @jcooklin: #36785 _(direct issue)_
- @jeffhandley: #30232 _(direct issue)_, #30204 _(direct issue)_, #26799 _(direct issue)_, #26788 _(direct issue)_, #24384 _(direct issue)_
- @jfomhover: #25420 _(direct issue)_
- @jitran: #38969 _(direct issue)_, #37052 _(direct issue)_, #33649 _(direct issue)_
- @joesturge: #40063 _(direct issue)_
- @johnpreed: #25687 _(direct issue)_
- @jonathanpeppers: #38900 _(direct issue)_, #32893 _(direct issue)_, #30662 _(direct issue)_
- @jsoref: #38928 _(direct issue)_, #27230 _(direct issue)_
- @jtracey93: #26176 _(direct issue)_
- @kaovilai: #32596 _(direct issue)_, #32587 _(direct issue)_, #32482 _(direct issue)_, #32467 _(direct issue)_
- @karl-petter-sj: #36209 _(direct issue)_
- @katriendg: #38561 _(direct issue)_
- @kbreit-insight: #24930 _(direct issue)_, #23940 _(direct issue)_
- @kkruel8100: #30867 _(direct issue)_
- @kthompson: #25550 _(direct issue)_
- @labudis: #30846 _(direct issue)_
- @ladamski: #33641 _(direct issue)_
- @lindeberg: #34006 _(direct issue)_
- @lpcox: #35972 _(direct issue)_, #35937 _(direct issue)_, #30634 _(direct issue)_, #29353 _(direct issue)_, #29191 _(direct issue)_
- @lupinthe14th: #26542 _(direct issue)_, #26441 _(direct issue)_
- @mason-tim: #33084 _(direct issue)_, #33074 _(direct issue)_, #31489 _(direct issue)_, #30336 _(direct issue)_, #29301 _(direct issue)_
- @MatthewBunker: #39876 _(direct issue)_, #39649 _(direct issue)_
- @MatthewLabasan-NBCU: #26289 _(direct issue)_
- @MattSkala: #40354 _(direct issue)_, #24567 _(direct issue)_
- @MauroDruwel: #30178 _(direct issue)_, #30169 _(direct issue)_, #29379 _(direct issue)_, #29378 _(direct issue)_
- @mdashrraf: #28657 _(direct issue)_
- @michen00: #36857 _(direct issue)_, #31869 _(direct issue)_
- @microsasa: #38859 _(direct issue)_, #27715 _(direct issue)_
- @mrfelton: #39764 _(direct issue)_, #38782 _(direct issue)_, #38326 _(direct issue)_, #37366 _(direct issue)_
- @mrjf: #32271 _(direct issue)_, #32069 _(direct issue)_, #31600 _(direct issue)_, #29152 _(direct issue)_, #28955 _(direct issue)_, #28471 _(direct issue)_, #28197 _(direct issue)_
- @neta-vega: #26447 _(direct issue)_, #25895 _(direct issue)_
- @NicolasRannou: #31701 _(direct issue)_
- @NikolajBjorner: #35762 _(direct issue)_, #28812 _(direct issue)_
- @norrietaylor: #37067 _(direct issue)_, #36700 _(direct issue)_, #36510 _(direct issue)_, #33199 _(direct issue)_, #32312 _(direct issue)_, #32310 _(direct issue)_, #30733 _(direct issue)_, #30392 _(direct issue)_
- @octatone: #31918 _(direct issue)_
- @PaulAylward2: #34844 _(direct issue)_
- @petercort: #28281 _(direct issue)_
- @pethers: #28470 _(direct issue)_
- @pgaskin: #26156 _(direct issue)_
- @pholleran: #36324 _(direct issue)_, #25313 _(direct issue)_
- @polmichel: #34904 _(direct issue)_, #32991 _(direct issue)_
- @PureWeen: #28767 _(direct issue)_, #27655 _(direct issue)_
- @rabo-unumed: #31660 _(direct issue)_, #31578 _(direct issue)_, #31513 _(direct issue)_
- @radiantspace: #40607 _(direct issue)_, #38627 _(direct issue)_
- @rhardouin: #30840 _(direct issue)_, #30838 _(direct issue)_
- @romainh-betclic: #28143 _(direct issue)_
- @rspurgeon: #26475 _(direct issue)_
- @Rubyj: #31542 _(direct issue)_
- @ruokun-niu: #24961 _(direct issue)_
- @ryckmansm: #36883 _(direct issue)_, #31501 _(direct issue)_
- @salekseev: #25137 _(direct issue)_, #25122 _(direct issue)_, #24135 _(direct issue)_
- @samuelkahessay: #33016 _(direct issue)_, #33015 _(direct issue)_, #24756 _(direct issue)_, #24755 _(direct issue)_, #24754 _(direct issue)_
- @sbodapati-gfm: #29417 _(direct issue)_
- @seangibeault: #26910 _(direct issue)_, #24905 _(direct issue)_
- @sg650: #36485 _(direct issue)_, #36466 _(direct issue)_, #35294 _(direct issue)_, #33787 _(direct issue)_, #32044 _(direct issue)_, #31617 _(direct issue)_, #31616 _(direct issue)_, #29009 _(direct issue)_, #28612 _(direct issue)_
- @shiran-gutsy: #27641 _(direct issue)_
- @shubhamtanwar23: #39206 _(direct issue)_, #39205 _(direct issue)_, #39202 _(direct issue)_
- @stefankrzyz: #27260 _(direct issue)_
- @straub: #24569 _(direct issue)_
- @strawgate: #33597 _(direct issue)_, #24422 _(direct issue)_, #24199 _(direct issue)_
- @susmahad: #26276 _(direct issue)_, #25866 _(direct issue)_, #25710 _(direct issue)_
- @szabta89: #29064 _(direct issue)_, #29063 _(direct issue)_, #24037 _(direct issue)_
- @tadelesh: #26001 _(direct issue)_
- @theletterf: #39793 _(direct issue)_, #36423 _(direct issue)_, #34981 _(direct issue)_, #33963 _(direct issue)_, #32846 _(direct issue)_, #30964 _(direct issue)_, #30365 _(direct issue)_, #30327 _(direct issue)_, #28898 _(direct issue)_, #28895 _(direct issue)_, #28691 _(direct issue)_, #28672 _(direct issue)_, #28221 _(direct issue)_, #27566 _(direct issue)_, #25494 _(direct issue)_
- @tinytelly: #27282 _(direct issue)_
- @tore-unumed: #36885 _(direct issue)_, #35446 _(direct issue)_, #35159 _(direct issue)_, #33545 _(direct issue)_, #31909 _(direct issue)_, #31650 _(direct issue)_, #30550 _(direct issue)_, #30324 _(direct issue)_, #29312 _(direct issue)_, #28019 _(direct issue)_
- @trask: #31612 _(direct issue)_, #31241 _(direct issue)_, #31098 _(direct issue)_, #31097 _(direct issue)_
- @tsm-harmoney: #39544 _(direct issue)_, #39382 _(direct issue)_, #31695 _(direct issue)_, #27880 _(direct issue)_
- @tvu4-wowcorp: #34556 _(direct issue)_
- @tylersmalley: #35287 _(direct issue)_
- @v1v: #39372 _(direct issue)_
- @verkyyi: #27407 _(direct issue)_, #27259 _(direct issue)_
- @vishalagrawal-jisr: #36469 _(direct issue)_, #36242 _(direct issue)_
- @wizardofosmium: #36579 _(direct issue)_
- @wtgodbe: #32834 _(direct issue)_, #26057 _(direct issue)_, #25130 _(direct issue)_, #24921 _(direct issue)_
- @yaananth: #24125 _(direct issue)_
- @Yoyokrazy: #36548 _(direct issue)_, #36547 _(direct issue)_
- @yskopets: #39605 _(direct issue)_, #39595 _(direct issue)_, #39462 _(direct issue)_, #38924 _(direct issue)_, #38905 _(direct issue)_, #38901 _(direct issue)_, #38897 _(direct issue)_, #38890 _(direct issue)_, #37705 _(direct issue)_, #36678 _(direct issue)_, #34772 _(direct issue)_, #34256 _(direct issue)_, #34250 _(direct issue)_, #34166 _(direct issue)_, #34134 _(direct issue)_, #32022 _(direct issue)_, #31831 _(direct issue)_, #31086 _(direct issue)_, #31073 _(direct issue)_, #30872 _(direct issue)_, #30705 _(direct issue)_, #27935 _(direct issue)_, #27898 _(direct issue)_, #27881 _(direct issue)_, #27773 _(direct issue)_, #27757 _(direct issue)_, #26922 _(direct issue)_, #26569 _(direct issue)_, #26468 _(direct issue)_, #26358 _(direct issue)_, #26346 _(direct issue)_, #26345 _(direct issue)_, #26280 _(direct issue)_, #26279 _(direct issue)_, #26120 _(direct issue)_, #26101 _(direct issue)_, #26085 _(direct issue)_, #26080 _(direct issue)_, #26067 _(direct issue)_, #25959 _(direct issue)_, #25946 _(direct issue)_, #25833 _(direct issue)_, #25363 _(direct issue)_, #25362 _(direct issue)_, #25125 _(direct issue)_, #24897 _(direct issue)_, #24573 _(direct issue)_
- @zarenner: #40156 _(direct issue)_, #38979 _(direct issue)_, #35577 _(direct issue)_, #35576 _(direct issue)_, #35575 _(direct issue)_
- @zkoppert: #27741 _(direct issue)_

</details>

## Share Feedback

We welcome your feedback on GitHub Agentic Workflows! 

- [Community Feedback Discussions](https://github.com/orgs/community/discussions/186451)
- [GitHub Discussions](https://github.com/github/gh-aw/discussions)

## Peli's Agent Factory

See the [Peli's Agent Factory](https://github.com/github/gh-aw/blob/main/docs/src/content/docs/blog/2026-01-12-welcome-to-pelis-agent-factory.md) for a guided tour through many uses of agentic workflows.

## Related Projects

GitHub Agentic Workflows is supported by companion projects that provide additional security and integration capabilities:

- **[Agent Workflow Firewall (AWF)](https://github.com/github/gh-aw-firewall)** - Network egress control for AI agents, providing domain-based access controls and activity logging for secure workflow execution
- **[MCP Gateway](https://github.com/github/gh-aw-mcpg)** - Routes Model Context Protocol (MCP) server calls through a unified HTTP gateway for centralized access management
- **[gh-aw-actions](https://github.com/github/gh-aw-actions)** - Shared library of custom GitHub Actions used by compiled workflows, providing functionality such as MCP server file management
