# GitHub Agentic Workflows

<details>
<summary>Agent quick links</summary>

Hello fellow agent! Welcome to GitHub Agentic Workflows = Actions + Agent + Safety.
Here are some pointers to get you started in using this tool.

- Create a new workflow: https://raw.githubusercontent.com/github/gh-aw/main/create.md
- Install: https://raw.githubusercontent.com/github/gh-aw/main/install.md
- Package a workflow repository: https://raw.githubusercontent.com/github/gh-aw/main/package.md
- Reference: https://raw.githubusercontent.com/github/gh-aw/main/.github/aw/github-agentic-workflows.md

</details>

GitHub Agentic Workflows (`gh-aw`) lets developers define AI-powered repository automation in Markdown with YAML frontmatter and run AI agents securely through GitHub Actions. The `gh-aw` GitHub CLI extension compiles each agentic workflow into a standard GitHub Actions workflow.

Use conventional GitHub Actions for deterministic builds, tests, linting, deployments, and reproducible scripts. Add an agentic workflow when a task needs reasoning or interpretation, such as issue triage, pull-request review, CI failure investigation, documentation maintenance, dependency analysis, or repository reporting. GitHub Agentic Workflows complements existing CI/CD; it does not replace it.

Built-in AI engines include GitHub Copilot, Claude Code, OpenAI Codex, Google Gemini, and Pi. Agent jobs are read-only and sandboxed by default, and configured GitHub writes are normally applied through validated `safe-outputs` jobs with scoped permissions.

> [!NOTE]
> **Releases 0.68.4 through 0.71.3 are being retired** due to a bug that impacts billing. If you are running one of these versions, please upgrade to the latest release as soon as possible.

## Contents

- [Quick Start](#quick-start)
- [How GitHub Agentic Workflows works](#how-github-agentic-workflows-works)
- [Security and permissions](#security-and-permissions)
- [Documentation](#documentation)
- [Contributing](#contributing)
- [Community Contributions](#-community-contributions)
- [Related Projects](#related-projects)
- [Workshop](#workshop)

## Quick Start

Install the GitHub CLI extension:

```bash
gh extension install github/gh-aw
```

Then follow the [GitHub Agentic Workflows quickstart](https://github.github.com/gh-aw/setup/quick-start/) to select an AI engine, add a sample workflow, and run it through GitHub Actions.

## How GitHub Agentic Workflows works

An agentic workflow has two parts: YAML frontmatter configures triggers, permissions, tools, and the AI engine; the Markdown body tells the AI agent what to accomplish. The `gh aw compile` command validates this source and generates the `.lock.yml` workflow that GitHub Actions executes. [Learn how GitHub Agentic Workflows works](https://github.github.com/gh-aw/introduction/how-they-work/).

## Security and permissions

Security, permissions, and controlled writes are core design concerns. The supported agent-job path defaults to read-only GitHub access and sandboxed execution. Safe outputs buffer configured writes, validate them, and apply them in separate jobs with scoped permissions. These controls are configurable, so workflow authors must review permissions, tools, network access, and generated files before deployment. [Learn how GitHub Agentic Workflows handles security and permissions](https://github.github.com/gh-aw/introduction/architecture/).

Using agentic workflows in your repository requires careful attention to security considerations and careful human supervision, and even then things can still go wrong. Use it with caution, and at your own risk.

## Documentation

Use the [GitHub Agentic Workflows documentation](https://github.github.com/gh-aw/) for these paths:

- [Create an agentic workflow](https://github.github.com/gh-aw/setup/creating-workflows/)
- [Choose and authenticate an AI engine](https://github.github.com/gh-aw/reference/engines/)
- [Browse GitHub Agentic Workflows examples by task](https://github.github.com/gh-aw/examples/)
- [Review the security architecture](https://github.github.com/gh-aw/introduction/architecture/)
- [Read the GitHub Agentic Workflows FAQ](https://github.github.com/gh-aw/reference/faq/)

For AI agents and retrieval tools, use the published [agent prompt index](https://github.github.com/gh-aw/llms.txt), [full prompt corpus](https://github.github.com/gh-aw/llms-full.txt), and [AI-readable project summary](https://github.github.com/gh-aw/ai/summary.json).

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

$(cat /tmp/gh-aw/agent/community-data/readme_community_section_tier012.md | tail -n +3)
[@jeremiah-snee-openx (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ajeremiah-snee-openx)
[@jfomhover (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ajfomhover)
[@jhamon (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ajhamon)
[@jitran (4)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ajitran)
[@joesturge (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ajoesturge)
[@johnpreed (5)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ajohnpreed)
[@johnwilliams-12 (11)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ajohnwilliams-12)
[@jonathanpeppers (4)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ajonathanpeppers)
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
[@kkruel8100 (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Akkruel8100)
[@Krzysztof-Cieslak (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AKrzysztof-Cieslak)
[@kthompson (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Akthompson)
[@kubaflo (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Akubaflo)
[@labudis (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Alabudis)
[@ladamski (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aladamski)
[@lecoursen (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Alecoursen)
[@lilseyi (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Alilseyi)
[@lindeberg (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Alindeberg)
[@loganrosen (4)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aloganrosen)
[@look (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Alook)
[@lpcox (7)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Alpcox)
[@ludoviclafole (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aludoviclafole)
[@lukeed (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Alukeed)
[@lupinthe14th (3)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Alupinthe14th)
[@m-titov (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Am-titov)
[@maikelvdh (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amaikelvdh)
[@mark-hingston (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amark-hingston)
[@mason-tim (8)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amason-tim)
[@matiloti (4)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amatiloti)
[@mattcosta7 (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amattcosta7)
[@MatthewBunker (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AMatthewBunker)
[@MatthewLabasan-NBCU (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AMatthewLabasan-NBCU)
[@MattSkala (4)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AMattSkala)
[@MauroDruwel (4)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AMauroDruwel)
[@maxbeizer (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amaxbeizer)
[@maxknv (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amaxknv)
[@mcantrell (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amcantrell)
[@mdashrraf (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amdashrraf)
[@MH0386 (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AMH0386)
[@mhavelock (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amhavelock)
[@michen00 (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amichen00)
[@microsasa (10)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amicrosasa)
[@mlinksva (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amlinksva)
[@mnkiefer (11)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amnkiefer)
[@molson504x (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amolson504x)
[@Mossaka (3)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AMossaka)
[@mrfelton (5)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amrfelton)
[@mrjf (7)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amrjf)
[@MrSanchez (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AMrSanchez)
[@mstrathman (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amstrathman)
[@mur6 (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amur6)
[@mvdbos (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Amvdbos)
[@nestele (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Anestele)
[@neta-vega (3)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aneta-vega)
[@NicoAvanzDev (3)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3ANicoAvanzDev)
[@NicolasRannou (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3ANicolasRannou)
[@nihal467 (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Anihal467)
[@Nikhil-Anand-DSG (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3ANikhil-Anand-DSG)
[@NikolajBjorner (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3ANikolajBjorner)
[@norrietaylor (8)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Anorrietaylor)
[@not-mksv (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Anot-mksv)
[@octatone (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aoctatone)
[@oscarvalenzuelab (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aoscarvalenzuelab)
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
[@PureWeen (10)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3APureWeen)
[@qwert666 (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aqwert666)
[@r-garcia-de-oliveira (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ar-garcia-de-oliveira)
[@rabo-unumed (4)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Arabo-unumed)
[@racedale (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aracedale)
[@radiantspace (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aradiantspace)
[@rafael-unloan (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Arafael-unloan)
[@rbstp (4)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Arbstp)
[@reggie-k (3)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Areggie-k)
[@remypanicker (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aremypanicker)
[@rhardouin (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Arhardouin)
[@rmarinho (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Armarinho)
[@romainh-betclic (3)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aromainh-betclic)
[@rspurgeon (4)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Arspurgeon)
[@Rubyj (3)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3ARubyj)
[@ruokun-niu (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aruokun-niu)
[@ryckmansm (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aryckmansm)
[@salekseev (3)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Asalekseev)
[@salmanmkc (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Asalmanmkc)
[@samuelkahessay (30)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Asamuelkahessay)
[@samus-aran (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Asamus-aran)
[@SanthoshNandha (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3ASanthoshNandha)
[@sbodapati-gfm (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Asbodapati-gfm)
[@seangibeault (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aseangibeault)
[@seesharprun (4)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aseesharprun)
[@sg650 (15)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Asg650)
[@shawnHartsell (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AshawnHartsell)
[@Shazwazza (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AShazwazza)
[@shiran-gutsy (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ashiran-gutsy)
[@shubhamtanwar23 (3)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ashubhamtanwar23)
[@siyo-rms (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Asiyo-rms)
[@srgibbs99 (6)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Asrgibbs99)
[@ssulei7 (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Assulei7)
[@stacktick (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Astacktick)
[@stefankrzyz (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Astefankrzyz)
[@steliosfran (3)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Asteliosfran)
[@straub (6)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Astraub)
[@strawgate (48)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Astrawgate)
[@susmahad (3)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Asusmahad)
[@swimmesberger (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aswimmesberger)
[@syarihu (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Asyarihu)
[@szabta89 (3)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aszabta89)
[@tadelesh (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Atadelesh)
[@Tarekchehahde (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3ATarekchehahde)
[@theletterf (22)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Atheletterf)
[@thi-feonir (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Athi-feonir)
[@timdittler (3)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Atimdittler)
[@tinytelly (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Atinytelly)
[@tobio (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Atobio)
[@tomasmed (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Atomasmed)
[@tore-unumed (16)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Atore-unumed)
[@trask (4)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Atrask)
[@tsm-harmoney (5)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Atsm-harmoney)
[@tspascoal (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Atspascoal)
[@tvu4-wowcorp (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Atvu4-wowcorp)
[@tylersmalley (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Atylersmalley)
[@UncleBats (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AUncleBats)
[@v1v (4)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Av1v)
[@verkyyi (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Averkyyi)
[@veverkap (3)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Aveverkap)
[@ViktorHofer (3)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AViktorHofer)
[@virenpepper (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Avirenpepper)
[@vishalagrawal-jisr (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Avishalagrawal-jisr)
[@whoschek (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Awhoschek)
[@wizardofosmium (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Awizardofosmium)
[@wtgodbe (4)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Awtgodbe)
[@xirzec (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Axirzec)
[@yaananth (1)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ayaananth)
[@Yoyokrazy (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3AYoyokrazy)
[@yskopets (53)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Ayskopets)
[@zarenner (5)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Azarenner)
[@zkoppert (2)](https://github.com/github/gh-aw/issues?q=is%3Aissue+is%3Aclosed+label%3Acommunity+author%3Azkoppert)



GitHub Agentic Workflows is supported by companion projects that provide additional security and integration capabilities:

- **[Agent Workflow Firewall (AWF)](https://github.com/github/gh-aw-firewall)** - Network egress control for AI agents, providing domain-based access controls and activity logging for secure workflow execution
- **[MCP Gateway](https://github.com/github/gh-aw-mcpg)** - Routes Model Context Protocol (MCP) server calls through a unified HTTP gateway for centralized access management
- **[gh-aw-actions](https://github.com/github/gh-aw-actions)** - Shared library of custom GitHub Actions used by compiled workflows, providing functionality such as MCP server file management

## Workshop

> [!TIP]
> **Ready to learn GitHub Agentic Workflows hands-on?** The [**Factory Tour Workshop**](https://github.com/githubnext/gh-aw-workshop) is a self-contained, step-by-step workshop repository designed to teach you how to build, run, and customize agentic workflows from scratch.
