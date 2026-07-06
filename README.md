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

[<img src="https://github.com/ahmadabdalla.png?size=24" width="24" height="24" alt="@ahmadabdalla">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aahmadabdalla)
[<img src="https://github.com/AkshatRaj00.png?size=24" width="24" height="24" alt="@AkshatRaj00">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AAkshatRaj00)
[<img src="https://github.com/alcastaneda.png?size=24" width="24" height="24" alt="@alcastaneda">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aalcastaneda)
[<img src="https://github.com/AlexDeMichieli.png?size=24" width="24" height="24" alt="@AlexDeMichieli">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AAlexDeMichieli)
[<img src="https://github.com/alondahari.png?size=24" width="24" height="24" alt="@alondahari">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aalondahari)
[<img src="https://github.com/anthonymastreanvae.png?size=24" width="24" height="24" alt="@anthonymastreanvae">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aanthonymastreanvae)
[<img src="https://github.com/aoxiangtianyu-go.png?size=24" width="24" height="24" alt="@aoxiangtianyu-go">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aaoxiangtianyu-go)
[<img src="https://github.com/apenab.png?size=24" width="24" height="24" alt="@apenab">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aapenab)
[<img src="https://github.com/app/github-actions.png?size=24" width="24" height="24" alt="@app/github-actions">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aapp/github-actions)
[<img src="https://github.com/arthurfvives.png?size=24" width="24" height="24" alt="@arthurfvives">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aarthurfvives)
[<img src="https://github.com/Artur-.png?size=24" width="24" height="24" alt="@Artur-">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AArtur-)
[<img src="https://github.com/askpaisa.png?size=24" width="24" height="24" alt="@askpaisa">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aaskpaisa)
[<img src="https://github.com/astefan.png?size=24" width="24" height="24" alt="@astefan">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aastefan)
[<img src="https://github.com/b2pacific.png?size=24" width="24" height="24" alt="@b2pacific">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ab2pacific)
[<img src="https://github.com/bartul.png?size=24" width="24" height="24" alt="@bartul">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Abartul)
[<img src="https://github.com/bbonafed.png?size=24" width="24" height="24" alt="@bbonafed">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Abbonafed)
[<img src="https://github.com/benissimo.png?size=24" width="24" height="24" alt="@benissimo">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Abenissimo)
[<img src="https://github.com/benvillalobos.png?size=24" width="24" height="24" alt="@benvillalobos">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Abenvillalobos)
[<img src="https://github.com/bmerkle.png?size=24" width="24" height="24" alt="@bmerkle">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Abmerkle)
[<img src="https://github.com/boydj.png?size=24" width="24" height="24" alt="@boydj">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aboydj)
[<img src="https://github.com/Bra1nFartz.png?size=24" width="24" height="24" alt="@Bra1nFartz">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3ABra1nFartz)
[<img src="https://github.com/bryanchen-d.png?size=24" width="24" height="24" alt="@bryanchen-d">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Abryanchen-d)
[<img src="https://github.com/Calidus.png?size=24" width="24" height="24" alt="@Calidus">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3ACalidus)
[<img src="https://github.com/CatsMiaow.png?size=24" width="24" height="24" alt="@CatsMiaow">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3ACatsMiaow)
[<img src="https://github.com/chrizbo.png?size=24" width="24" height="24" alt="@chrizbo">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Achrizbo)
[<img src="https://github.com/CiscoRob.png?size=24" width="24" height="24" alt="@CiscoRob">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3ACiscoRob)
[<img src="https://github.com/cknight.png?size=24" width="24" height="24" alt="@cknight">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Acknight)
[<img src="https://github.com/clementbolin.png?size=24" width="24" height="24" alt="@clementbolin">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aclementbolin)
[<img src="https://github.com/cogni-ai-ee.png?size=24" width="24" height="24" alt="@cogni-ai-ee">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Acogni-ai-ee)
[<img src="https://github.com/consulthys.png?size=24" width="24" height="24" alt="@consulthys">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aconsulthys)
[<img src="https://github.com/corygehr.png?size=24" width="24" height="24" alt="@corygehr">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Acorygehr)
[<img src="https://github.com/Daidanny008.png?size=24" width="24" height="24" alt="@Daidanny008">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3ADaidanny008)
[<img src="https://github.com/Dan-Albrecht.png?size=24" width="24" height="24" alt="@Dan-Albrecht">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3ADan-Albrecht)
[<img src="https://github.com/danielmeppiel.png?size=24" width="24" height="24" alt="@danielmeppiel">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Adanielmeppiel)
[<img src="https://github.com/danquirk.png?size=24" width="24" height="24" alt="@danquirk">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Adanquirk)
[<img src="https://github.com/darwin-gonzales.png?size=24" width="24" height="24" alt="@darwin-gonzales">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Adarwin-gonzales)
[<img src="https://github.com/DeagleGross.png?size=24" width="24" height="24" alt="@DeagleGross">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3ADeagleGross)
[<img src="https://github.com/devantler.png?size=24" width="24" height="24" alt="@devantler">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Adevantler)
[<img src="https://github.com/deyaaeldeen.png?size=24" width="24" height="24" alt="@deyaaeldeen">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Adeyaaeldeen)
[<img src="https://github.com/dfrysinger.png?size=24" width="24" height="24" alt="@dfrysinger">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Adfrysinger)
[<img src="https://github.com/dgolombek.png?size=24" width="24" height="24" alt="@dgolombek">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Adgolombek)
[<img src="https://github.com/dholmes.png?size=24" width="24" height="24" alt="@dholmes">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Adholmes)
[<img src="https://github.com/dkurepa.png?size=24" width="24" height="24" alt="@dkurepa">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Adkurepa)
[<img src="https://github.com/drehelis.png?size=24" width="24" height="24" alt="@drehelis">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Adrehelis)
[<img src="https://github.com/dsibilio.png?size=24" width="24" height="24" alt="@dsibilio">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Adsibilio)
[<img src="https://github.com/dsyme.png?size=24" width="24" height="24" alt="@dsyme">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Adsyme)
[<img src="https://github.com/duncankmckinnon.png?size=24" width="24" height="24" alt="@duncankmckinnon">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aduncankmckinnon)
[<img src="https://github.com/edburns.png?size=24" width="24" height="24" alt="@edburns">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aedburns)
[<img src="https://github.com/edgeq.png?size=24" width="24" height="24" alt="@edgeq">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aedgeq)
[<img src="https://github.com/ericstj.png?size=24" width="24" height="24" alt="@ericstj">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aericstj)
[<img src="https://github.com/Evangelink.png?size=24" width="24" height="24" alt="@Evangelink">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AEvangelink)
[<img src="https://github.com/fbecar22.png?size=24" width="24" height="24" alt="@fbecar22">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Afbecar22)
[<img src="https://github.com/flatiron32.png?size=24" width="24" height="24" alt="@flatiron32">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aflatiron32)
[<img src="https://github.com/GandrotulaRajesh.png?size=24" width="24" height="24" alt="@GandrotulaRajesh">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AGandrotulaRajesh)
[<img src="https://github.com/h-no.png?size=24" width="24" height="24" alt="@h-no">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ah-no)
[<img src="https://github.com/h3y6e.png?size=24" width="24" height="24" alt="@h3y6e">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ah3y6e)
[<img src="https://github.com/haavamoa.png?size=24" width="24" height="24" alt="@haavamoa">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ahaavamoa)
[<img src="https://github.com/heiskr.png?size=24" width="24" height="24" alt="@heiskr">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aheiskr)
[<img src="https://github.com/hermanho.png?size=24" width="24" height="24" alt="@hermanho">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ahermanho)
[<img src="https://github.com/hpsin.png?size=24" width="24" height="24" alt="@hpsin">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ahpsin)
[<img src="https://github.com/IEvangelist.png?size=24" width="24" height="24" alt="@IEvangelist">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AIEvangelist)
[<img src="https://github.com/ivancea.png?size=24" width="24" height="24" alt="@ivancea">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aivancea)
[<img src="https://github.com/jamesadevine.png?size=24" width="24" height="24" alt="@jamesadevine">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ajamesadevine)
[<img src="https://github.com/JamesNK.png?size=24" width="24" height="24" alt="@JamesNK">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AJamesNK)
[<img src="https://github.com/JanKrivanek.png?size=24" width="24" height="24" alt="@JanKrivanek">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AJanKrivanek)
[<img src="https://github.com/jaroslawgajewski.png?size=24" width="24" height="24" alt="@jaroslawgajewski">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ajaroslawgajewski)
[<img src="https://github.com/JasonYeMSFT.png?size=24" width="24" height="24" alt="@JasonYeMSFT">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AJasonYeMSFT)
[<img src="https://github.com/jbaruch.png?size=24" width="24" height="24" alt="@jbaruch">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ajbaruch)
[<img src="https://github.com/jcooklin.png?size=24" width="24" height="24" alt="@jcooklin">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ajcooklin)
[<img src="https://github.com/jeffhandley.png?size=24" width="24" height="24" alt="@jeffhandley">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ajeffhandley)
[<img src="https://github.com/jfomhover.png?size=24" width="24" height="24" alt="@jfomhover">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ajfomhover)
[<img src="https://github.com/jitran.png?size=24" width="24" height="24" alt="@jitran">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ajitran)
[<img src="https://github.com/joesturge.png?size=24" width="24" height="24" alt="@joesturge">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ajoesturge)
[<img src="https://github.com/johnpreed.png?size=24" width="24" height="24" alt="@johnpreed">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ajohnpreed)
[<img src="https://github.com/jonathanpeppers.png?size=24" width="24" height="24" alt="@jonathanpeppers">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ajonathanpeppers)
[<img src="https://github.com/jsoref.png?size=24" width="24" height="24" alt="@jsoref">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ajsoref)
[<img src="https://github.com/jsquire.png?size=24" width="24" height="24" alt="@jsquire">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ajsquire)
[<img src="https://github.com/jtracey93.png?size=24" width="24" height="24" alt="@jtracey93">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ajtracey93)
[<img src="https://github.com/kaovilai.png?size=24" width="24" height="24" alt="@kaovilai">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Akaovilai)
[<img src="https://github.com/karl-petter-sj.png?size=24" width="24" height="24" alt="@karl-petter-sj">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Akarl-petter-sj)
[<img src="https://github.com/katriendg.png?size=24" width="24" height="24" alt="@katriendg">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Akatriendg)
[<img src="https://github.com/kkruel8100.png?size=24" width="24" height="24" alt="@kkruel8100">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Akkruel8100)
[<img src="https://github.com/kthompson.png?size=24" width="24" height="24" alt="@kthompson">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Akthompson)
[<img src="https://github.com/labudis.png?size=24" width="24" height="24" alt="@labudis">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Alabudis)
[<img src="https://github.com/ladamski.png?size=24" width="24" height="24" alt="@ladamski">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aladamski)
[<img src="https://github.com/lindeberg.png?size=24" width="24" height="24" alt="@lindeberg">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Alindeberg)
[<img src="https://github.com/lpcox.png?size=24" width="24" height="24" alt="@lpcox">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Alpcox)
[<img src="https://github.com/lupinthe14th.png?size=24" width="24" height="24" alt="@lupinthe14th">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Alupinthe14th)
[<img src="https://github.com/m-titov.png?size=24" width="24" height="24" alt="@m-titov">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Am-titov)
[<img src="https://github.com/maikelvdh.png?size=24" width="24" height="24" alt="@maikelvdh">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amaikelvdh)
[<img src="https://github.com/mason-tim.png?size=24" width="24" height="24" alt="@mason-tim">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amason-tim)
[<img src="https://github.com/mattcosta7.png?size=24" width="24" height="24" alt="@mattcosta7">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amattcosta7)
[<img src="https://github.com/MatthewBunker.png?size=24" width="24" height="24" alt="@MatthewBunker">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AMatthewBunker)
[<img src="https://github.com/MatthewLabasan-NBCU.png?size=24" width="24" height="24" alt="@MatthewLabasan-NBCU">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AMatthewLabasan-NBCU)
[<img src="https://github.com/MattSkala.png?size=24" width="24" height="24" alt="@MattSkala">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AMattSkala)
[<img src="https://github.com/MauroDruwel.png?size=24" width="24" height="24" alt="@MauroDruwel">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AMauroDruwel)
[<img src="https://github.com/maxknv.png?size=24" width="24" height="24" alt="@maxknv">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amaxknv)
[<img src="https://github.com/mdashrraf.png?size=24" width="24" height="24" alt="@mdashrraf">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amdashrraf)
[<img src="https://github.com/michen00.png?size=24" width="24" height="24" alt="@michen00">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amichen00)
[<img src="https://github.com/microsasa.png?size=24" width="24" height="24" alt="@microsasa">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amicrosasa)
[<img src="https://github.com/mnkiefer.png?size=24" width="24" height="24" alt="@mnkiefer">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amnkiefer)
[<img src="https://github.com/mrfelton.png?size=24" width="24" height="24" alt="@mrfelton">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amrfelton)
[<img src="https://github.com/mrjf.png?size=24" width="24" height="24" alt="@mrjf">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amrjf)
[<img src="https://github.com/nestele.png?size=24" width="24" height="24" alt="@nestele">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Anestele)
[<img src="https://github.com/neta-vega.png?size=24" width="24" height="24" alt="@neta-vega">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aneta-vega)
[<img src="https://github.com/NicolasRannou.png?size=24" width="24" height="24" alt="@NicolasRannou">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3ANicolasRannou)
[<img src="https://github.com/NikolajBjorner.png?size=24" width="24" height="24" alt="@NikolajBjorner">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3ANikolajBjorner)
[<img src="https://github.com/norrietaylor.png?size=24" width="24" height="24" alt="@norrietaylor">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Anorrietaylor)
[<img src="https://github.com/octatone.png?size=24" width="24" height="24" alt="@octatone">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aoctatone)
[<img src="https://github.com/PaulAylward2.png?size=24" width="24" height="24" alt="@PaulAylward2">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3APaulAylward2)
[<img src="https://github.com/petercort.png?size=24" width="24" height="24" alt="@petercort">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Apetercort)
[<img src="https://github.com/pethers.png?size=24" width="24" height="24" alt="@pethers">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Apethers)
[<img src="https://github.com/pgaskin.png?size=24" width="24" height="24" alt="@pgaskin">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Apgaskin)
[<img src="https://github.com/pholleran.png?size=24" width="24" height="24" alt="@pholleran">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Apholleran)
[<img src="https://github.com/polmichel.png?size=24" width="24" height="24" alt="@polmichel">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Apolmichel)
[<img src="https://github.com/PureWeen.png?size=24" width="24" height="24" alt="@PureWeen">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3APureWeen)
[<img src="https://github.com/rabo-unumed.png?size=24" width="24" height="24" alt="@rabo-unumed">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Arabo-unumed)
[<img src="https://github.com/radiantspace.png?size=24" width="24" height="24" alt="@radiantspace">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aradiantspace)
[<img src="https://github.com/reggie-k.png?size=24" width="24" height="24" alt="@reggie-k">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Areggie-k)
[<img src="https://github.com/rhardouin.png?size=24" width="24" height="24" alt="@rhardouin">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Arhardouin)
[<img src="https://github.com/romainh-betclic.png?size=24" width="24" height="24" alt="@romainh-betclic">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aromainh-betclic)
[<img src="https://github.com/rspurgeon.png?size=24" width="24" height="24" alt="@rspurgeon">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Arspurgeon)
[<img src="https://github.com/Rubyj.png?size=24" width="24" height="24" alt="@Rubyj">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3ARubyj)
[<img src="https://github.com/ryckmansm.png?size=24" width="24" height="24" alt="@ryckmansm">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aryckmansm)
[<img src="https://github.com/samuelkahessay.png?size=24" width="24" height="24" alt="@samuelkahessay">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Asamuelkahessay)
[<img src="https://github.com/sbodapati-gfm.png?size=24" width="24" height="24" alt="@sbodapati-gfm">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Asbodapati-gfm)
[<img src="https://github.com/seangibeault.png?size=24" width="24" height="24" alt="@seangibeault">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aseangibeault)
[<img src="https://github.com/sg650.png?size=24" width="24" height="24" alt="@sg650">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Asg650)
[<img src="https://github.com/shiran-gutsy.png?size=24" width="24" height="24" alt="@shiran-gutsy">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ashiran-gutsy)
[<img src="https://github.com/shubhamtanwar23.png?size=24" width="24" height="24" alt="@shubhamtanwar23">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ashubhamtanwar23)
[<img src="https://github.com/stefankrzyz.png?size=24" width="24" height="24" alt="@stefankrzyz">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Astefankrzyz)
[<img src="https://github.com/strawgate.png?size=24" width="24" height="24" alt="@strawgate">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Astrawgate)
[<img src="https://github.com/susmahad.png?size=24" width="24" height="24" alt="@susmahad">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Asusmahad)
[<img src="https://github.com/szabta89.png?size=24" width="24" height="24" alt="@szabta89">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aszabta89)
[<img src="https://github.com/tadelesh.png?size=24" width="24" height="24" alt="@tadelesh">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Atadelesh)
[<img src="https://github.com/theletterf.png?size=24" width="24" height="24" alt="@theletterf">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Atheletterf)
[<img src="https://github.com/tinytelly.png?size=24" width="24" height="24" alt="@tinytelly">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Atinytelly)
[<img src="https://github.com/tore-unumed.png?size=24" width="24" height="24" alt="@tore-unumed">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Atore-unumed)
[<img src="https://github.com/trask.png?size=24" width="24" height="24" alt="@trask">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Atrask)
[<img src="https://github.com/tsm-harmoney.png?size=24" width="24" height="24" alt="@tsm-harmoney">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Atsm-harmoney)
[<img src="https://github.com/tvu4-wowcorp.png?size=24" width="24" height="24" alt="@tvu4-wowcorp">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Atvu4-wowcorp)
[<img src="https://github.com/tylersmalley.png?size=24" width="24" height="24" alt="@tylersmalley">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Atylersmalley)
[<img src="https://github.com/v1v.png?size=24" width="24" height="24" alt="@v1v">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Av1v)
[<img src="https://github.com/verkyyi.png?size=24" width="24" height="24" alt="@verkyyi">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Averkyyi)
[<img src="https://github.com/vishalagrawal-jisr.png?size=24" width="24" height="24" alt="@vishalagrawal-jisr">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Avishalagrawal-jisr)
[<img src="https://github.com/wizardofosmium.png?size=24" width="24" height="24" alt="@wizardofosmium">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Awizardofosmium)
[<img src="https://github.com/wtgodbe.png?size=24" width="24" height="24" alt="@wtgodbe">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Awtgodbe)
[<img src="https://github.com/Yoyokrazy.png?size=24" width="24" height="24" alt="@Yoyokrazy">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AYoyokrazy)
[<img src="https://github.com/yskopets.png?size=24" width="24" height="24" alt="@yskopets">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ayskopets)
[<img src="https://github.com/zarenner.png?size=24" width="24" height="24" alt="@zarenner">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Azarenner)
[<img src="https://github.com/zkoppert.png?size=24" width="24" height="24" alt="@zkoppert">](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Azkoppert)

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
