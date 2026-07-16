<!--

Hello fellow agent! Welcome to GitHub Agentic Workflows = Actions + Agent + Safety. 
Here are some pointers to get you started in using this tool.

- Create a new workflow: https://raw.githubusercontent.com/github/gh-aw/main/create.md
- Install: https://raw.githubusercontent.com/github/gh-aw/main/install.md
- Package a workflow repository: https://raw.githubusercontent.com/github/gh-aw/main/package.md
- Reference: https://raw.githubusercontent.com/github/gh-aw/main/.github/aw/github-agentic-workflows.md

-->

<details>
<summary>Configure this project with agentic workflows</summary>

Hey agent! If this repository hasn't been configured with agentic workflows yet, here is how to set it up.

1. Install the `gh aw` CLI extension using the install script (no GitHub token required):

   ```bash
   curl -sL https://raw.githubusercontent.com/github/gh-aw/main/install-gh-aw.sh | bash
   ```

2. Run the `init` command to configure the repository:

   ```bash
   gh aw init
   ```

3. Add a **daily repo status** agentic workflow using the `add` command:

   ```bash
   gh aw add --create-pull-request
   ```

   The workflow should:
   - Run on a daily schedule
   - Summarize the repository status (open issues, recent PRs, CI health)
   - Use the [Quick Start engine selector](https://github.com/github/gh-aw/blob/main/docs/src/content/docs/setup/quick-start.mdx) to choose your runtime:
     - For `copilot`, grant `copilot-requests: write`
     - For `claude`, set the `ANTHROPIC_API_KEY` repository secret
     - For `codex`, set the `OPENAI_API_KEY` repository secret
     - For `gemini`, set the `GEMINI_API_KEY` repository secret

</details>

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
- [Workshop](#workshop)

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

[@aaronspindler (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aaaronspindler)
[@abillingsley (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aabillingsley)
[@adam-cobb (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aadam-cobb)
[@adamhenson (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aadamhenson)
[@adhikjoshi (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aadhikjoshi)
[@ahmadabdalla (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aahmadabdalla)
[@ajfeldman6 (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aajfeldman6)
[@AkshatRaj00 (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AAkshatRaj00)
[@alanpeabody (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aalanpeabody)
[@alcastaneda (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aalcastaneda)
[@AlexanderWert (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AAlexanderWert)
[@AlexDeMichieli (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AAlexDeMichieli)
[@alexsiilvaa (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aalexsiilvaa)
[@alondahari (11)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aalondahari)
[@AmoebaChant (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AAmoebaChant)
[@anthonymastreanvae (10)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aanthonymastreanvae)
[@aoxiangtianyu-go (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aaoxiangtianyu-go)
[@apenab (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aapenab)
[@arezero (6)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aarezero)
[@arthurfvives (8)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aarthurfvives)
[@Artur- (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AArtur-)
[@askpaisa (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aaskpaisa)
[@askpt (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aaskpt)
[@astefan (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aastefan)
[@b2pacific (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ab2pacific)
[@babaakihiro (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ababaakihiro)
[@bartul (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Abartul)
[@bbonafed (23)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Abbonafed)
[@beardofedu (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Abeardofedu)
[@benissimo (6)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Abenissimo)
[@benvillalobos (12)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Abenvillalobos)
[@bmerkle (3)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Abmerkle)
[@boydj (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aboydj)
[@Bra1nFartz (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3ABra1nFartz)
[@BrandonLewis (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3ABrandonLewis)
[@bryanchen-d (14)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Abryanchen-d)
[@bryanknox (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Abryanknox)
[@Calidus (4)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3ACalidus)
[@camposbrunocampos (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Acamposbrunocampos)
[@carlincherry (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Acarlincherry)
[@CatsMiaow (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3ACatsMiaow)
[@chepa92 (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Achepa92)
[@chrisfregly (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Achrisfregly)
[@chrizbo (7)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Achrizbo)
[@CiscoRob (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3ACiscoRob)
[@cknight (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Acknight)
[@clementbolin (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aclementbolin)
[@cogni-ai-ee (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Acogni-ai-ee)
[@consulthys (3)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aconsulthys)
[@Corb3nik (3)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3ACorb3nik)
[@corygehr (20)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Acorygehr)
[@corymhall (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Acorymhall)
[@dagecko (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Adagecko)
[@Daidanny008 (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3ADaidanny008)
[@Dan-Albrecht (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3ADan-Albrecht)
[@Dan-Co (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3ADan-Co)
[@danielmeppiel (6)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Adanielmeppiel)
[@danquirk (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Adanquirk)
[@darwin-gonzales (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Adarwin-gonzales)
[@davidahmann (3)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Adavidahmann)
[@dbudym-cs (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Adbudym-cs)
[@DeagleGross (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3ADeagleGross)
[@devantler (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Adevantler)
[@deyaaeldeen (10)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Adeyaaeldeen)
[@dfrysinger (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Adfrysinger)
[@dgolombek (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Adgolombek)
[@dholmes (3)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Adholmes)
[@dhrapson (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Adhrapson)
[@DimaBir (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3ADimaBir)
[@dkurepa (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Adkurepa)
[@DogeAmazed (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3ADogeAmazed)
[@doughgle (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Adoughgle)
[@drehelis (4)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Adrehelis)
[@DrPye (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3ADrPye)
[@dsibilio (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Adsibilio)
[@dsolteszopyn (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Adsolteszopyn)
[@dsyme (32)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Adsyme)
[@duncankmckinnon (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aduncankmckinnon)
[@eaftan (3)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aeaftan)
[@edburns (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aedburns)
[@edgeq (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aedgeq)
[@elika56 (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aelika56)
[@eran-medan (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aeran-medan)
[@ericchansen (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aericchansen)
[@ericstj (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aericstj)
[@Esomoire-consultancy-Company (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AEsomoire-consultancy-Company)
[@Evangelink (3)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AEvangelink)
[@fbecar22 (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Afbecar22)
[@ferryhinardi (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aferryhinardi)
[@flatiron32 (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aflatiron32)
[@fr4nc1sc0-r4m0n (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Afr4nc1sc0-r4m0n)
[@G1Vh (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AG1Vh)
[@GandrotulaRajesh (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AGandrotulaRajesh)
[@github-actions (10)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Agithub-actions)
[@github-antoine-brechon (4)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Agithub-antoine-brechon)
[@GKersten (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AGKersten)
[@glitch-ux (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aglitch-ux)
[@grahame-white (9)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Agrahame-white)
[@h-no (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ah-no)
[@h3y6e (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ah3y6e)
[@haavamoa (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ahaavamoa)
[@harrisoncramer (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aharrisoncramer)
[@heaversm (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aheaversm)
[@heiskr (3)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aheiskr)
[@hermanho (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ahermanho)
[@holwerda (3)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aholwerda)
[@hpsin (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ahpsin)
[@hrishikeshathalye (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ahrishikeshathalye)
[@ianreay (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aianreay)
[@IEvangelist (13)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AIEvangelist)
[@Infinnerty (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AInfinnerty)
[@insop (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ainsop)
[@ivancea (3)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aivancea)
[@j-srodka (6)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aj-srodka)
[@jamesadevine (4)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ajamesadevine)
[@JamesNK (4)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AJamesNK)
[@JanKrivanek (3)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AJanKrivanek)
[@jaroslawgajewski (26)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ajaroslawgajewski)
[@JasonYeMSFT (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AJasonYeMSFT)
[@jbaruch (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ajbaruch)
[@jcooklin (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ajcooklin)
[@jeffhandley (9)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ajeffhandley)
[@jeremiah-snee-openx (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ajeremiah-snee-openx)
[@jfomhover (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ajfomhover)
[@jitran (3)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ajitran)
[@joesturge (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ajoesturge)
[@johnpreed (4)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ajohnpreed)
[@johnwilliams-12 (11)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ajohnwilliams-12)
[@jonathanpeppers (3)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ajonathanpeppers)
[@joperezr (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ajoperezr)
[@JoshGreenslade (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AJoshGreenslade)
[@jsalmassy (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ajsalmassy)
[@jsoref (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ajsoref)
[@jsquire (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ajsquire)
[@jtracey93 (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ajtracey93)
[@kaovilai (4)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Akaovilai)
[@karl-petter-sj (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Akarl-petter-sj)
[@katriendg (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Akatriendg)
[@kbreit-insight (5)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Akbreit-insight)
[@KGoovaer (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AKGoovaer)
[@kkruel8100 (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Akkruel8100)
[@Krzysztof-Cieslak (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AKrzysztof-Cieslak)
[@kthompson (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Akthompson)
[@labudis (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Alabudis)
[@ladamski (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aladamski)
[@lindeberg (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Alindeberg)
[@look (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Alook)
[@lpcox (6)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Alpcox)
[@lukeed (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Alukeed)
[@lupinthe14th (3)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Alupinthe14th)
[@m-titov (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Am-titov)
[@maikelvdh (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amaikelvdh)
[@mark-hingston (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amark-hingston)
[@mason-tim (7)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amason-tim)
[@matiloti (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amatiloti)
[@mattcosta7 (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amattcosta7)
[@MatthewBunker (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AMatthewBunker)
[@MatthewLabasan-NBCU (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AMatthewLabasan-NBCU)
[@MattSkala (3)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AMattSkala)
[@MauroDruwel (4)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AMauroDruwel)
[@maxbeizer (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amaxbeizer)
[@maxknv (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amaxknv)
[@mcantrell (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amcantrell)
[@mdashrraf (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amdashrraf)
[@MH0386 (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AMH0386)
[@mhavelock (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amhavelock)
[@michen00 (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amichen00)
[@microsasa (9)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amicrosasa)
[@mlinksva (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amlinksva)
[@mnkiefer (4)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amnkiefer)
[@molson504x (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amolson504x)
[@Mossaka (3)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AMossaka)
[@mrfelton (4)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amrfelton)
[@mrjf (7)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amrjf)
[@mstrathman (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amstrathman)
[@mvdbos (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amvdbos)
[@nestele (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Anestele)
[@neta-vega (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aneta-vega)
[@NicoAvanzDev (3)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3ANicoAvanzDev)
[@NicolasRannou (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3ANicolasRannou)
[@Nikhil-Anand-DSG (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3ANikhil-Anand-DSG)
[@NikolajBjorner (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3ANikolajBjorner)
[@norrietaylor (8)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Anorrietaylor)
[@octatone (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aoctatone)
[@PaulAylward2 (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3APaulAylward2)
[@petercort (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Apetercort)
[@pethers (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Apethers)
[@pgaskin (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Apgaskin)
[@pholleran (4)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Apholleran)
[@Phonesis (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3APhonesis)
[@Pierrci (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3APierrci)
[@plengauer (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aplengauer)
[@pmalarme (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Apmalarme)
[@polmichel (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Apolmichel)
[@ppusateri (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Appusateri)
[@praveenkuttappan (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Apraveenkuttappan)
[@PureWeen (5)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3APureWeen)
[@qwert666 (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aqwert666)
[@rabo-unumed (4)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Arabo-unumed)
[@racedale (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aracedale)
[@radiantspace (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aradiantspace)
[@rafael-unloan (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Arafael-unloan)
[@reggie-k (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Areggie-k)
[@rhardouin (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Arhardouin)
[@rmarinho (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Armarinho)
[@romainh-betclic (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aromainh-betclic)
[@rspurgeon (4)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Arspurgeon)
[@Rubyj (3)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3ARubyj)
[@ruokun-niu (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aruokun-niu)
[@ryckmansm (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aryckmansm)
[@salekseev (3)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Asalekseev)
[@samuelkahessay (29)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Asamuelkahessay)
[@samus-aran (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Asamus-aran)
[@sbodapati-gfm (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Asbodapati-gfm)
[@seangibeault (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aseangibeault)
[@sg650 (12)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Asg650)
[@shiran-gutsy (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ashiran-gutsy)
[@shubhamtanwar23 (3)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ashubhamtanwar23)
[@siyo-rms (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Asiyo-rms)
[@srgibbs99 (6)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Asrgibbs99)
[@stacktick (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Astacktick)
[@stefankrzyz (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Astefankrzyz)
[@steliosfran (3)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Asteliosfran)
[@straub (4)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Astraub)
[@strawgate (44)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Astrawgate)
[@susmahad (3)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Asusmahad)
[@swimmesberger (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aswimmesberger)
[@szabta89 (3)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aszabta89)
[@tadelesh (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Atadelesh)
[@Tarekchehahde (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3ATarekchehahde)
[@theletterf (16)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Atheletterf)
[@thi-feonir (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Athi-feonir)
[@timdittler (3)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Atimdittler)
[@tinytelly (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Atinytelly)
[@tomasmed (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Atomasmed)
[@tore-unumed (16)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Atore-unumed)
[@trask (4)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Atrask)
[@tsm-harmoney (5)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Atsm-harmoney)
[@tspascoal (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Atspascoal)
[@tvu4-wowcorp (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Atvu4-wowcorp)
[@tylersmalley (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Atylersmalley)
[@UncleBats (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AUncleBats)
[@v1v (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Av1v)
[@verkyyi (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Averkyyi)
[@veverkap (3)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aveverkap)
[@ViktorHofer (3)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AViktorHofer)
[@virenpepper (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Avirenpepper)
[@vishalagrawal-jisr (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Avishalagrawal-jisr)
[@whoschek (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Awhoschek)
[@wizardofosmium (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Awizardofosmium)
[@wtgodbe (4)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Awtgodbe)
[@yaananth (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ayaananth)
[@Yoyokrazy (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AYoyokrazy)
[@yskopets (51)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ayskopets)
[@zarenner (5)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Azarenner)
[@zkoppert (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Azkoppert)


### ⚠️ Attribution Candidates Need Review

The following community issues were closed during this period but could not be automatically linked to a specific merged PR. Please verify whether they should be credited:

- **@austenstone** for [[bug] Intake Triage agent job: MODULE_NOT_FOUND for merge_awf_model_multipliers.cjs (5 failures in 8d, v0.77.5)](https://github.com/github/gh-aw/issues/42874) — closed 2026-07-04, no confirmed PR linkage found
- **@martinbriley** for [Request that the sentiment functionality emit events for sentiment analysis](https://github.com/github/gh-aw/issues/42842) — closed 2026-07-05, no confirmed PR linkage found
- **@theletterf** for [docs-pr-ai-menu: transient GitHub API 502 causes permanent workflow failure (no retries)](https://github.com/github/gh-aw/issues/39904) — closed 2026-06-18, no confirmed PR linkage found

_Note: 2 additional automation-related issues (#41256, #41255) were closed but excluded from this list._
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

## Workshop

> [!TIP]
> **Ready to learn GitHub Agentic Workflows hands-on?** The [**gh-aw-workshop**](https://github.com/githubnext/gh-aw-workshop) is a self-contained, step-by-step workshop repository designed to teach you how to build, run, and customize agentic workflows from scratch.
>
> Fork the workshop repository and follow along at your own pace.
