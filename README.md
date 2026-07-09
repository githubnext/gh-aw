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

<sup>Community members whose issues were resolved — updated automatically.</sup>

[@ahmadabdalla (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aahmadabdalla)
[@AkshatRaj00 (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AAkshatRaj00)
[@alanpeabody (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aalanpeabody)
[@alcastaneda (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aalcastaneda)
[@AlexDeMichieli (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AAlexDeMichieli)
[@alondahari (9)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aalondahari)
[@anthonymastreanvae (9)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aanthonymastreanvae)
[@aoxiangtianyu-go (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aaoxiangtianyu-go)
[@apenab (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aapenab)
[@app/github-actions (10)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aapp/github-actions)
[@arthurfvives (7)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aarthurfvives)
[@Artur- (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AArtur-)
[@askpaisa (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aaskpaisa)
[@astefan (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aastefan)
[@b2pacific (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ab2pacific)
[@babaakihiro (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ababaakihiro)
[@bartul (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Abartul)
[@bbonafed (10)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Abbonafed)
[@benissimo (6)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Abenissimo)
[@benvillalobos (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Abenvillalobos)
[@bmerkle (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Abmerkle)
[@boydj (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aboydj)
[@Bra1nFartz (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3ABra1nFartz)
[@bryanchen-d (13)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Abryanchen-d)
[@Calidus (4)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3ACalidus)
[@CatsMiaow (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3ACatsMiaow)
[@chrizbo (4)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Achrizbo)
[@CiscoRob (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3ACiscoRob)
[@cknight (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Acknight)
[@clementbolin (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aclementbolin)
[@cogni-ai-ee (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Acogni-ai-ee)
[@consulthys (3)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aconsulthys)
[@corygehr (17)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Acorygehr)
[@Daidanny008 (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3ADaidanny008)
[@Dan-Albrecht (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3ADan-Albrecht)
[@danielmeppiel (3)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Adanielmeppiel)
[@danquirk (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Adanquirk)
[@darwin-gonzales (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Adarwin-gonzales)
[@DeagleGross (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3ADeagleGross)
[@devantler (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Adevantler)
[@deyaaeldeen (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Adeyaaeldeen)
[@dfrysinger (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Adfrysinger)
[@dgolombek (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Adgolombek)
[@dholmes (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Adholmes)
[@drehelis (3)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Adrehelis)
[@dsibilio (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Adsibilio)
[@dsyme (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Adsyme)
[@duncankmckinnon (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aduncankmckinnon)
[@edburns (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aedburns)
[@edgeq (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aedgeq)
[@ericstj (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aericstj)
[@Evangelink (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AEvangelink)
[@fbecar22 (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Afbecar22)
[@flatiron32 (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aflatiron32)
[@GandrotulaRajesh (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AGandrotulaRajesh)
[@github-antoine-brechon (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Agithub-antoine-brechon)
[@GKersten (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AGKersten)
[@h-no (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ah-no)
[@h3y6e (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ah3y6e)
[@haavamoa (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ahaavamoa)
[@heiskr (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aheiskr)
[@hermanho (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ahermanho)
[@hpsin (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ahpsin)
[@IEvangelist (12)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AIEvangelist)
[@ivancea (3)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aivancea)
[@jamesadevine (4)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ajamesadevine)
[@JamesNK (4)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AJamesNK)
[@JanKrivanek (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AJanKrivanek)
[@jaroslawgajewski (9)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ajaroslawgajewski)
[@JasonYeMSFT (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AJasonYeMSFT)
[@jbaruch (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ajbaruch)
[@jcooklin (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ajcooklin)
[@jeffhandley (8)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ajeffhandley)
[@jitran (3)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ajitran)
[@joesturge (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ajoesturge)
[@johnpreed (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ajohnpreed)
[@jonathanpeppers (3)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ajonathanpeppers)
[@jsoref (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ajsoref)
[@jsquire (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ajsquire)
[@jtracey93 (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ajtracey93)
[@kaovilai (4)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Akaovilai)
[@karl-petter-sj (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Akarl-petter-sj)
[@katriendg (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Akatriendg)
[@kkruel8100 (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Akkruel8100)
[@labudis (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Alabudis)
[@ladamski (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aladamski)
[@lindeberg (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Alindeberg)
[@lpcox (5)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Alpcox)
[@lupinthe14th (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Alupinthe14th)
[@m-titov (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Am-titov)
[@maikelvdh (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amaikelvdh)
[@mason-tim (5)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amason-tim)
[@mattcosta7 (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amattcosta7)
[@MatthewBunker (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AMatthewBunker)
[@MatthewLabasan-NBCU (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AMatthewLabasan-NBCU)
[@MattSkala (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AMattSkala)
[@MauroDruwel (4)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AMauroDruwel)
[@maxknv (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amaxknv)
[@mdashrraf (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amdashrraf)
[@michen00 (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amichen00)
[@microsasa (3)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amicrosasa)
[@mnkiefer (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amnkiefer)
[@mrfelton (4)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amrfelton)
[@mrjf (7)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amrjf)
[@nestele (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Anestele)
[@neta-vega (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aneta-vega)
[@NicolasRannou (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3ANicolasRannou)
[@NikolajBjorner (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3ANikolajBjorner)
[@norrietaylor (8)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Anorrietaylor)
[@octatone (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aoctatone)
[@PaulAylward2 (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3APaulAylward2)
[@petercort (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Apetercort)
[@pethers (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Apethers)
[@pgaskin (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Apgaskin)
[@pholleran (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Apholleran)
[@polmichel (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Apolmichel)
[@PureWeen (3)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3APureWeen)
[@rabo-unumed (3)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Arabo-unumed)
[@radiantspace (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aradiantspace)
[@reggie-k (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Areggie-k)
[@rhardouin (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Arhardouin)
[@romainh-betclic (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aromainh-betclic)
[@rspurgeon (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Arspurgeon)
[@Rubyj (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3ARubyj)
[@ryckmansm (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aryckmansm)
[@samuelkahessay (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Asamuelkahessay)
[@sbodapati-gfm (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Asbodapati-gfm)
[@seangibeault (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aseangibeault)
[@sg650 (11)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Asg650)
[@shiran-gutsy (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ashiran-gutsy)
[@shubhamtanwar23 (3)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ashubhamtanwar23)
[@stefankrzyz (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Astefankrzyz)
[@straub (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Astraub)
[@strawgate (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Astrawgate)
[@susmahad (3)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Asusmahad)
[@szabta89 (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aszabta89)
[@tadelesh (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Atadelesh)
[@theletterf (14)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Atheletterf)
[@tinytelly (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Atinytelly)
[@tore-unumed (10)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Atore-unumed)
[@trask (4)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Atrask)
[@tsm-harmoney (5)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Atsm-harmoney)
[@tvu4-wowcorp (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Atvu4-wowcorp)
[@tylersmalley (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Atylersmalley)
[@UncleBats (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AUncleBats)
[@v1v (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Av1v)
[@verkyyi (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Averkyyi)
[@vishalagrawal-jisr (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Avishalagrawal-jisr)
[@wizardofosmium (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Awizardofosmium)
[@wtgodbe (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Awtgodbe)
[@Yoyokrazy (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AYoyokrazy)
[@yskopets (44)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ayskopets)
[@zarenner (5)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Azarenner)
[@zkoppert (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Azkoppert)



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
