<!--

Hello fellow agent! Welcome to GitHub Agentic Workflows = Actions + Agent + Safety. 
Here are some pointers to get you started in using this tool.

- Create a new workflow: https://raw.githubusercontent.com/github/gh-aw/main/create.md
- Install: https://raw.githubusercontent.com/github/gh-aw/main/install.md
- Reference: https://raw.githubusercontent.com/github/gh-aw/main/.github/aw/github-agentic-workflows.md

-->

# GitHub Agentic Workflows

Write agentic workflows in natural language markdown, and run them in GitHub Actions.

> [!NOTE]
> The Team and the Agents are in Spring Break. Expect delays.

## Contents

- [Quick Start](#quick-start)
- [Overview](#overview)
- [Guardrails](#guardrails)
- [Documentation](#documentation)
- [Contributing](#contributing)
- [Community Contributions](#-community-contributions)
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

- #18714 _(direct issue)_

### @abillingsley

- #23736 _(direct issue)_

### @adam-cobb

- #18295 _(direct issue)_

### @adamhenson

- #24282 _(direct issue)_

### @adhikjoshi

- #18781 _(direct issue)_

### @ajfeldman6

- #23924 _(direct issue)_

### @AlexanderWert

- #18703 _(direct issue)_

### @alexsiilvaa

- #20781 _(direct issue)_
- #20664 _(direct issue)_

### @alondahari

- #21207 _(direct issue)_

### @AmoebaChant

- #17975 _(direct issue)_

### @arezero

- #20515 _(direct issue)_
- #20513 _(direct issue)_
- #20512 _(direct issue)_
- #20511 _(direct issue)_
- #20510 _(direct issue)_

### @askpt

- #17763 _(direct issue)_

### @bbonafed

- #24918 _(direct issue)_
- #24896 _(direct issue)_
- #24323 _(direct issue)_
- #23900 _(direct issue)_
- #23724 _(direct issue)_
- #23566 _(direct issue)_
- #22564 _(direct issue)_
- #21990 _(direct issue)_
- #20801 _(direct issue)_
- #20378 _(direct issue)_
- #18542 _(direct issue)_

### @beardofedu

- #18723 _(direct issue)_

### @benvillalobos

- #20885 _(direct issue)_
- [`network: { allowed: [] }` still allows infrastructure domains — same behavior as `network: {}`](https://github.com/github/gh-aw/issues/18557) _(direct issue)_
- #18115 _(direct issue)_
- #18109 _(direct issue)_
- #18103 _(direct issue)_
- #18101 _(direct issue)_
- #17995 _(direct issue)_
- #17943 _(direct issue)_
- #16625 _(direct issue)_

### @bmerkle

- #20646 _(direct issue)_

### @bryanchen-d

- #23265 _(direct issue)_

### @BrandonLewis

- #18263 _(direct issue)_

### @carlincherry

- #22017 _(direct issue)_

### @chepa92

- #20322 _(direct issue)_

### @chrisfregly

- #23963 _(direct issue)_

### @chrizbo

- #22510 _(direct issue)_
- #21863 _(direct issue)_
- #19347 _(direct issue)_

### @CiscoRob

- #20416 _(direct issue)_

### @Corb3nik

- #18825 _(direct issue)_

### @corygehr

- #24355 _(direct issue)_
- #23944 _(direct issue)_
- #23753 _(direct issue)_

### @corymhall

- #19839 _(direct issue)_

### @dagecko

- #24743 _(direct issue)_

### @Dan-Co

- #22707 _(direct issue)_
- #17978 _(direct issue)_

### @danielmeppiel

- #20663 _(direct issue)_
- #20380 _(direct issue)_
- #19810 _(direct issue)_

### @davidahmann

- #18121 _(direct issue)_
- #17151 _(direct issue)_
- #16360 _(direct issue)_

### @deyaaeldeen

- #23198 _(direct issue)_
- #23024 _(direct issue)_
- #23020 _(direct issue)_
- #22957 _(direct issue)_
- #19773 _(direct issue)_
- #19770 _(direct issue)_

### @dholmes

- #23578 _(direct issue)_

### @dhrapson

- #18385 _(direct issue)_
- #17962 _(direct issue)_

### @DimaBir

- #20483 _(direct issue)_

### @DogeAmazed

- #22703 _(direct issue)_

### @DrPye

- #18711 _(direct issue)_

### @dsolteszopyn

- #18421 _(direct issue)_
- #17058 _(direct issue)_

### @dsyme

- #23936 _(direct issue)_
- #22340 _(direct issue)_
- #20953 _(direct issue)_
- #20952 _(direct issue)_
- #20950 _(direct issue)_
- #20787 _(direct issue)_
- #20578 _(direct issue)_
- #20420 _(direct issue)_
- #20243 _(direct issue)_
- #20241 _(direct issue)_
- #20108 _(direct issue)_
- #20103 _(direct issue)_
- #19976 _(direct issue)_
- #19708 _(direct issue)_
- #19468 _(direct issue)_
- #19465 _(direct issue)_
- #19219 _(direct issue)_
- #19120 _(direct issue)_
- #19104 _(direct issue)_
- #19067 _(direct issue)_
- #18854 _(direct issue)_
- #18831 _(direct issue)_
- #18574 _(direct issue)_
- #18535 _(direct issue)_
- #18485 _(direct issue)_
- #18483 _(direct issue)_
- #18482 _(direct issue)_
- #18481 _(direct issue)_
- #18211 _(direct issue)_
- #18018 _(direct issue)_
- #16145 _(direct issue)_

### @eaftan

- #23257 _(direct issue)_
- #20457 _(direct issue)_
- #18412 _(direct issue)_

### @elika56

- #16163 _(direct issue)_

### @eran-medan

- #16457 _(direct issue)_

### @ericchansen

- #20222 _(direct issue)_

### @ericstj

- #23766 _(direct issue)_

### @Esomoire-consultancy-Company

- #20207 _(direct issue)_

### @ferryhinardi

- #24128 _(direct issue)_

### @flatiron32

- #22469 _(direct issue)_

### @fr4nc1sc0-r4m0n

- #20657 _(direct issue)_

### @G1Vh

- #20308 _(direct issue)_

### @glitch-ux

- #24403 _(direct issue)_

### @grahame-white

- #23643 _(direct issue)_
- #23093 _(direct issue)_
- #23092 _(direct issue)_
- #23088 _(direct issue)_
- #23083 _(direct issue)_
- #20868 _(direct issue)_
- #20719 _(direct issue)_
- #20629 _(direct issue)_
- #20299 _(direct issue)_

### @harrisoncramer

- #19441 _(direct issue)_
- #18763 _(direct issue)_

### @heaversm

- #18747 _(direct issue)_

### @heiskr

- #20394 _(direct issue)_

### @Henry-Shan

- #23461 _(direct issue)_

### @holwerda

- #21243 _(direct issue)_

### @hrishikeshathalye

- [[Question] Can I not use a PAT for Copilot?](https://github.com/github/gh-aw/issues/19547) _(direct issue)_

### @Infinnerty

- #21957 _(direct issue)_

### @insop

- #21686 _(direct issue)_

### @j-srodka

- #23485 _(direct issue)_
- #23484 _(direct issue)_
- #23483 _(direct issue)_
- #23482 _(direct issue)_

### @JanKrivanek

- #20187 _(direct issue)_

### @jaroslawgajewski

- #24373 _(direct issue)_
- #24372 _(direct issue)_
- #24371 _(direct issue)_
- #24259 _(direct issue)_
- #24036 _(direct issue)_
- #23779 _(direct issue)_
- #23558 _(direct issue)_
- #22647 _(direct issue)_
- #21816 _(direct issue)_
- #20813 _(direct issue)_
- #20811 _(direct issue)_
- #19732 _(direct issue)_
- #18356 _(direct issue)_
- #16467 _(direct issue)_
- #16314 _(direct issue)_
- #16150 _(direct issue)_

### @jeffhandley

- #24384 _(direct issue)_

### @jeremiah-snee-openx

- #18196 _(direct issue)_

### @johnpreed

- #23777 _(direct issue)_
- #23212 _(direct issue)_
- #21334 _(direct issue)_

### @johnwilliams-12

- #21205 _(direct issue)_
- #21074 _(direct issue)_
- #21071 _(direct issue)_
- #21062 _(direct issue)_
- #20821 _(direct issue)_
- #20779 _(direct issue)_
- #20697 _(direct issue)_
- #20694 _(direct issue)_
- #20658 _(direct issue)_
- #20567 _(direct issue)_
- #20508 _(direct issue)_

### @joperezr

- #17243 _(direct issue)_

### @JoshGreenslade

- #18480 _(direct issue)_
- #16312 _(direct issue)_

### @kbreit-insight

- #23940 _(direct issue)_
- #23725 _(direct issue)_
- #22430 _(direct issue)_
- #21978 _(direct issue)_

### @KGoovaer

- #18556 _(direct issue)_

### @Krzysztof-Cieslak

- #18488 _(direct issue)_

### @look

- #23258 _(direct issue)_

### @lpcox

- #22281 _(direct issue)_

### @lupinthe14th

- #18712 _(direct issue)_

### @mark-hingston

- #20335 _(direct issue)_

### @mason-tim

- #21562 _(direct issue)_
- #19765 _(direct issue)_

### @MatthewLabasan-NBCU

- #19500 _(direct issue)_

### @MattSkala

- #24567 _(direct issue)_
- #21203 _(direct issue)_

### @maxbeizer

- #18875 _(direct issue)_

### @mcantrell

- #20592 _(direct issue)_

### @mhavelock

- #22110 _(direct issue)_

### @microsasa

- #21098 _(direct issue)_
- #20851 _(direct issue)_
- #20833 _(direct issue)_
- #20586 _(direct issue)_

### @mnkiefer

- #22409 _(direct issue)_
- [[research] Overview of docs improver agents](https://github.com/github/gh-aw/issues/19836) _(direct issue)_

### @molson504x

- #21834 _(direct issue)_
- #21615 _(direct issue)_

### @Mossaka

- #21630 _(direct issue)_

### @mstrathman

- #16005 _(direct issue)_

### @mvdbos

- #20411 _(direct issue)_
- #20249 _(direct issue)_

### @NicoAvanzDev

- #21542 _(direct issue)_
- #20540 _(direct issue)_
- #20528 _(direct issue)_

### @Nikhil-Anand-DSG

- #18200 _(direct issue)_

### @pholleran

- #23572 _(direct issue)_
- #21313 _(direct issue)_

### @Phonesis

- #16236 _(direct issue)_

### @Pierrci

- #18587 _(direct issue)_

### @plengauer

- #18297 _(direct issue)_

### @pmalarme

- #16642 _(direct issue)_

### @ppusateri

- #16587 _(direct issue)_

### @praveenkuttappan

- #18386 _(direct issue)_
- #18379 _(direct issue)_

### @PureWeen

- #23567 _(direct issue)_

### @qwert666

- #18162 _(direct issue)_

### @rabo-unumed

- #20679 _(direct issue)_

### @racedale

- #17982 _(direct issue)_

### @rafael-unloan

- #11190 _(direct issue)_

### @rmarinho

- #16555 _(direct issue)_

### @rspurgeon

- #19451 _(direct issue)_
- #18373 _(direct issue)_
- #15595 _(direct issue)_

### @Rubyj

- #21432 _(direct issue)_
- #20283 _(direct issue)_

### @salekseev

- #24135 _(direct issue)_

### @samuelkahessay

- #24756 _(direct issue)_
- #24755 _(direct issue)_
- #24754 _(direct issue)_
- #22380 _(direct issue)_
- #22364 _(direct issue)_
- #22161 _(direct issue)_
- #22138 _(direct issue)_
- #21975 _(direct issue)_
- #21955 _(direct issue)_
- #21501 _(direct issue)_
- #21304 _(direct issue)_
- #20035 _(direct issue)_
- #20031 _(direct issue)_
- #20030 _(direct issue)_
- #19605 _(direct issue)_
- #19476 _(direct issue)_
- #19475 _(direct issue)_
- #19474 _(direct issue)_
- #19473 _(direct issue)_
- #19158 _(direct issue)_
- #19024 _(direct issue)_
- #19023 _(direct issue)_
- #19020 _(direct issue)_
- #19018 _(direct issue)_
- #19017 _(direct issue)_

### @samus-aran

- #18468 _(direct issue)_

### @seangibeault

- #24905 _(direct issue)_

### @srgibbs99

- #22939 _(direct issue)_
- #19640 _(direct issue)_
- #19622 _(direct issue)_
- #18751 _(direct issue)_
- #18745 _(direct issue)_
- #17298 _(direct issue)_

### @stacktick

- #21361 _(direct issue)_

### @steliosfran

- [[bug] base-branch in assign-to-agent uses customInstructions text instead of GraphQL baseRef field](https://github.com/github/gh-aw/issues/17299) _(direct issue)_
- [[enhancement] Add base-branch support to assign-to-agent safe output for cross-repo PR creation](https://github.com/github/gh-aw/issues/17046) _(direct issue)_
- #16280 _(direct issue)_

### @straub

- #24569 _(direct issue)_
- #19631 _(direct issue)_

### @strawgate

- #24422 _(direct issue)_
- #24199 _(direct issue)_
- #23935 _(direct issue)_
- #23768 _(direct issue)_
- #21157 _(direct issue)_
- #21144 _(direct issue)_
- #19982 _(direct issue)_
- #21135 _(direct issue)_
- #21028 _(direct issue)_
- #20910 _(direct issue)_
- #20259 _(direct issue)_
- #20168 _(direct issue)_
- #20125 _(direct issue)_
- #20033 _(direct issue)_
- #19172 _(direct issue)_
- #18945 _(direct issue)_
- #18900 _(direct issue)_
- #18744 _(direct issue)_
- #18565 _(direct issue)_
- #18563 _(direct issue)_
- #18547 _(direct issue)_
- #18545 _(direct issue)_
- #18501 _(direct issue)_
- #18362 _(direct issue)_
- #18226 _(direct issue)_
- #17839 _(direct issue)_
- #17828 _(direct issue)_
- #17522 _(direct issue)_
- #17521 _(direct issue)_
- #16896 _(direct issue)_
- #16673 _(direct issue)_
- #16664 _(direct issue)_
- #16511 _(direct issue)_
- #16370 _(direct issue)_
- #16036 _(direct issue)_
- #15982 _(direct issue)_
- #15976 _(direct issue)_
- #15836 _(direct issue)_
- #15583 _(direct issue)_
- #15576 _(direct issue)_

### @swimmesberger

- #19421 _(direct issue)_

### @szabta89

- #24037 _(direct issue)_

### @theletterf

- #18465 _(direct issue)_

### @timdittler

- #16331 _(direct issue)_
- #16117 _(direct issue)_
- #16107 _(direct issue)_

### @tomasmed

- #20157 _(direct issue)_

### @tore-unumed

- #20780 _(direct issue)_
- #19370 _(direct issue)_
- #18329 _(direct issue)_
- #18107 _(direct issue)_
- #17289 _(direct issue)_

### @tspascoal

- #20597 _(direct issue)_
- #18123 _(direct issue)_

### @UncleBats

- #20359 _(direct issue)_

### @veverkap

- #22362 _(direct issue)_
- #21260 _(direct issue)_
- #21257 _(direct issue)_

### @ViktorHofer

- #18340 _(direct issue)_
- #18311 _(direct issue)_

### @virenpepper

- #23765 _(direct issue)_

### @whoschek

- #15510 _(direct issue)_

### @wtgodbe

- #24921 _(direct issue)_

### @yaananth

- #24125 _(direct issue)_

### @yskopets

- #24897 _(direct issue)_
- #24573 _(direct issue)_
- #23914 _(direct issue)_

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
