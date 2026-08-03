---
# Squad Bootstrap — installs and initializes Squad (https://github.com/bradygaster/squad)
# in the activation job only, then republishes the generated team state to the agent job.
#
# The Squad CLI is never installed or executed in the agent job — only the files it
# produces (`.squad/` team state and `.github/agents/squad.agent.md`) are restored there.
#
# Usage:
#   imports:
#     - shared/squad.md
#
# Optional custom credentials for `squad init` (only needed when Squad must reach other
# organizations or private repositories beyond the current one):
#   vars.SQUAD_GITHUB_APP_ID / secrets.SQUAD_GITHUB_APP_PRIVATE_KEY / vars.SQUAD_GITHUB_APP_OWNER
#     — mints a GitHub App installation token for `squad init`
#   secrets.SQUAD_GITHUB_TOKEN
#     — used if the App id is not set
# Auth precedence: GitHub App installation token > SQUAD_GITHUB_TOKEN > the workflow's
# own default token (`github.token`).
#
# Optional custom Squad CLI version (defaults to 0.11.0):
#   vars.SQUAD_CLI_VERSION
engine:
  id: copilot
  agent: squad
  copilot-sdk: true

jobs:
  activation:
    pre-steps:
      - name: Setup Node.js for Squad
        uses: actions/setup-node@v7.0.0
        with:
          node-version: "22"

      - name: Mint Squad GitHub App token
        id: squad-app-token
        if: ${{ vars.SQUAD_GITHUB_APP_ID != '' }}
        uses: actions/create-github-app-token@v2
        with:
          app-id: ${{ vars.SQUAD_GITHUB_APP_ID }}
          private-key: ${{ secrets.SQUAD_GITHUB_APP_PRIVATE_KEY }}
          owner: ${{ vars.SQUAD_GITHUB_APP_OWNER }}

      - name: Install Squad CLI
        env:
          SQUAD_CLI_VERSION: ${{ vars.SQUAD_CLI_VERSION }}
        run: |
          set -euo pipefail
          npm install -g "@bradygaster/squad-cli@${SQUAD_CLI_VERSION:-0.11.0}"

      - name: Initialize Squad team
        env:
          GH_TOKEN: ${{ steps.squad-app-token.outputs.token || secrets.SQUAD_GITHUB_TOKEN || github.token }}
        run: |
          set -euo pipefail
          squad init --preset default

      - name: Upload Squad state artifact
        if: success()
        uses: actions/upload-artifact@v7.0.1
        with:
          name: squad-state
          include-hidden-files: true
          path: |
            .squad
            .github/agents/squad.agent.md
          if-no-files-found: ignore
          retention-days: 1

steps:
  - name: Restore Squad state from activation artifact
    continue-on-error: true
    uses: actions/download-artifact@v8.0.1
    with:
      name: squad-state
      path: ${{ github.workspace }}
---

<!--

## Squad Bootstrap Component

This shared component moves the entire Squad (https://github.com/bradygaster/squad)
install/init lifecycle out of the agent job:

1. **`jobs.activation.pre-steps`** — the repository is already checked out by the
   activation job itself, so this only installs the pinned `@bradygaster/squad-cli`
   npm release, optionally mints a GitHub App installation token (or uses a supplied
   PAT) so `squad init` can see other organizations or private repositories, runs
   `squad init --preset default` (idempotent), and uploads the resulting `.squad/`
   team state plus `.github/agents/squad.agent.md` as a dedicated `squad-state`
   artifact — all inside the activation job, alongside the rest of the
   prompt/skills/sub-agent packaging.
2. **`steps:`** (agent job) — downloads the `squad-state` artifact and restores it into
   the checked-out workspace. The Squad CLI itself is never installed here; only the
   files it produced are copied in.

-->

## Working with Squad

Squad's team state (`.squad/`) and its Copilot custom agent
(`.github/agents/squad.agent.md`) were already initialized during activation and
restored into this checkout before you started — do not install Squad or run
`squad init` yourself.

- Verify `.squad/team.md` exists before delegating work to the team. If it is
  missing, the activation-job bootstrap step failed — call `noop` and explain
  why instead of proceeding.
- Coordinate work through the Squad team already defined in `.squad/` rather
  than proposing a brand-new team from scratch.
