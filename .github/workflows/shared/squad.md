---
# Squad Bootstrap — lazily installs and initializes Squad (https://github.com/bradygaster/squad)
# in the activation job when needed, then republishes the team state to the agent job.
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
#   env.SQUAD_CLI_VERSION
#   vars.SQUAD_CLI_VERSION
# Environment variable precedence: SQUAD_CLI_VERSION > vars.SQUAD_CLI_VERSION > 0.11.0.
#
# Optional remote Squad repository:
#   imports:
#     - uses: shared/squad.md
#       with:
#         squad-repository: owner/repo
#     — checks out the requested Squad repository on demand and overrides any local
#       or previously persisted Squad installation for this run.
#
# No `copilot-sdk` sidecar is needed here: Copilot's `--agent squad` flag runs
# through the standard CLI harness, and the Squad CLI itself only ever runs in
# the activation job.
import-schema:
  squad-repository:
    type: string
    required: false
    default: ""
    description: "Optional owner/repo of a Squad repository to install on demand instead of the local or persisted Squad."
engine:
  id: copilot
  agent: squad
env:
  SQUAD_TEAM_ROOT: /tmp/gh-aw/repo-memory/squad-state
ambient-folders:
  - .squad
  - .github/agents
  - .gh-aw-squad-remote
tools:
  repo-memory:
    - id: squad-state
      branch-name: memory/squad-state
      file-glob:
        - .squad/**
        - .github/agents/**
        - .gh-aw-squad-remote/**
        - .mcp.json
      allowed-extensions: [".json", ".md"]
      format-json: true
      description: Squad team state persisted between workflow runs.
jobs:
  activation:
    steps:
      - name: Restore persisted Squad state
        env:
          GH_TOKEN: ${{ github.token }}
          GITHUB_SERVER_URL: ${{ github.server_url }}
          SQUAD_MEMORY_BRANCH: memory/squad-state
          SQUAD_REMOTE_REPOSITORY: ${{ github.aw.import-inputs.squad-repository }}
          SQUAD_REPO_MEMORY_DIR: /tmp/gh-aw/repo-memory/squad-state
          TARGET_REPO: ${{ github.repository }}
        run: |
          set -euo pipefail

          rm -rf .gh-aw-squad-remote

          remote_repo="${SQUAD_REMOTE_REPOSITORY:-}"

          if [ -n "$remote_repo" ]; then
            rm -rf .squad .mcp.json
            rm -f .github/agents/squad.agent.md
            exit 0
          fi

          repo_url="${GITHUB_SERVER_URL%/}/${TARGET_REPO}.git"
          auth_url="${repo_url/https:\/\//https://x-access-token:${GH_TOKEN}@}"

          rm -rf "$SQUAD_REPO_MEMORY_DIR"
          mkdir -p "$(dirname "$SQUAD_REPO_MEMORY_DIR")"
          if ! git clone --quiet --branch "$SQUAD_MEMORY_BRANCH" --single-branch "$auth_url" "$SQUAD_REPO_MEMORY_DIR"; then
            mkdir -p "$SQUAD_REPO_MEMORY_DIR"
            exit 0
          fi

          if [ -d "$SQUAD_REPO_MEMORY_DIR/.squad" ]; then
            rm -rf .squad
            cp -a "$SQUAD_REPO_MEMORY_DIR/.squad" .squad
          fi
          if [ -d "$SQUAD_REPO_MEMORY_DIR/.github/agents" ]; then
            mkdir -p .github
            rm -rf .github/agents
            cp -a "$SQUAD_REPO_MEMORY_DIR/.github/agents" .github/agents
          fi
          if [ -f "$SQUAD_REPO_MEMORY_DIR/.mcp.json" ]; then
            cp -a "$SQUAD_REPO_MEMORY_DIR/.mcp.json" .mcp.json
          fi
          if [ -d "$SQUAD_REPO_MEMORY_DIR/.gh-aw-squad-remote" ]; then
            cp -a "$SQUAD_REPO_MEMORY_DIR/.gh-aw-squad-remote" .gh-aw-squad-remote
          fi
      - name: Detect existing Squad installation
        id: squad-installation
        env:
          SQUAD_REMOTE_REPOSITORY: ${{ github.aw.import-inputs.squad-repository }}
        run: |
          remote_repo="${SQUAD_REMOTE_REPOSITORY:-}"
          echo "remote_repo=$remote_repo" >> "$GITHUB_OUTPUT"
          if [ -n "$remote_repo" ]; then
            echo "installed=false" >> "$GITHUB_OUTPUT"
            exit 0
          fi
          if [ -f .squad/team.md ] && [ -f .github/agents/squad.agent.md ]; then
            echo "installed=true" >> "$GITHUB_OUTPUT"
          else
            echo "installed=false" >> "$GITHUB_OUTPUT"
          fi
      - name: Mint Squad GitHub App token
        id: squad-app-token
        if: ${{ steps.squad-installation.outputs.installed != 'true' && vars.SQUAD_GITHUB_APP_ID != '' }}
        uses: actions/create-github-app-token@v3.2.0
        with:
          app-id: ${{ vars.SQUAD_GITHUB_APP_ID }}
          private-key: ${{ secrets.SQUAD_GITHUB_APP_PRIVATE_KEY }}
          owner: ${{ vars.SQUAD_GITHUB_APP_OWNER }}
      - name: Checkout requested Squad repository
        if: ${{ steps.squad-installation.outputs.remote_repo != '' }}
        uses: actions/checkout@v7.0.1
        with:
          repository: ${{ steps.squad-installation.outputs.remote_repo }}
          path: .gh-aw-squad-remote
          token: ${{ steps.squad-app-token.outputs.token || secrets.SQUAD_GITHUB_TOKEN || github.token }}
          persist-credentials: false
      - name: Initialize Squad team
        if: ${{ steps.squad-installation.outputs.installed != 'true' }}
        env:
          SQUAD_CLI_VERSION_FROM_VARS: ${{ vars.SQUAD_CLI_VERSION }}
          GH_TOKEN: ${{ steps.squad-app-token.outputs.token || secrets.SQUAD_GITHUB_TOKEN || github.token }}
          SQUAD_REMOTE_REPOSITORY: ${{ steps.squad-installation.outputs.remote_repo }}
        run: |
          set -euo pipefail

          squad_cli_version="${SQUAD_CLI_VERSION:-${SQUAD_CLI_VERSION_FROM_VARS:-0.11.0}}"
          if [ -n "$SQUAD_REMOTE_REPOSITORY" ]; then
            npx --yes "@bradygaster/squad-cli@$squad_cli_version" init --mode remote .gh-aw-squad-remote
          else
            npx --yes "@bradygaster/squad-cli@$squad_cli_version" init --preset default
          fi
steps:
  - name: Stage Squad state in repo-memory
    env:
      SQUAD_REPO_MEMORY_DIR: /tmp/gh-aw/repo-memory/squad-state
    run: |
      set -euo pipefail

      mkdir -p "$SQUAD_REPO_MEMORY_DIR"

      rm -rf "$SQUAD_REPO_MEMORY_DIR/.squad"
      cp -a .squad "$SQUAD_REPO_MEMORY_DIR/.squad"

      mkdir -p "$SQUAD_REPO_MEMORY_DIR/.github"
      rm -rf "$SQUAD_REPO_MEMORY_DIR/.github/agents"
      cp -a .github/agents "$SQUAD_REPO_MEMORY_DIR/.github/agents"

      if [ -f .mcp.json ]; then
        cp -a .mcp.json "$SQUAD_REPO_MEMORY_DIR/.mcp.json"
      else
        rm -f "$SQUAD_REPO_MEMORY_DIR/.mcp.json"
      fi

      if [ -d .gh-aw-squad-remote ]; then
        rm -rf "$SQUAD_REPO_MEMORY_DIR/.gh-aw-squad-remote"
        cp -a .gh-aw-squad-remote "$SQUAD_REPO_MEMORY_DIR/.gh-aw-squad-remote"
      else
        rm -rf "$SQUAD_REPO_MEMORY_DIR/.gh-aw-squad-remote"
      fi

---

<!--

## Squad Bootstrap Component

This shared component moves the Squad (https://github.com/bradygaster/squad)
install/init lifecycle out of the agent job and performs it lazily:

1. **`jobs.activation.steps`** — the repository is already checked out by the
   activation job itself, so this first checks for the existing Squad team state
   and custom agent, restoring them first from `repo-memory` when available.
   When either is missing, it installs the pinned `@bradygaster/squad-cli` npm
   release, optionally mints a GitHub App installation token (or uses a supplied
   PAT) so `squad init` can see other organizations or private repositories, and
   runs `squad init --preset default`.
   If `squad-repository` is provided as an import input, activation checks out that
   repository and runs `squad init --mode remote .gh-aw-squad-remote` instead,
   overriding any local or persisted Squad for the run.
2. **`ambient-folders`** — bundles the resulting `.squad/` team state and
   `.github/agents/` files (plus the on-demand remote Squad checkout when used)
   into the standard activation artifact alongside the rest of the
   prompt/skills/sub-agent packaging, then restores them into the agent checkout.
   The Squad CLI itself is never installed in the agent job; only the files it
   produced are copied in.
3. **`tools.repo-memory`** — persists the active Squad state on
   `memory/squad-state`. Before the agent starts, the restored activation state is
   staged into `/tmp/gh-aw/repo-memory/squad-state/`; `SQUAD_TEAM_ROOT` points
   Squad at that repo-memory checkout so runtime state changes are saved by the
   generated repo-memory push job.

-->

## Working with Squad

Squad's team state (`.squad/`) and its Copilot custom agent
(`.github/agents/squad.agent.md`) were reused or initialized during activation,
staged into repo-memory, and restored into this checkout before you started — do
not install Squad or run `squad init` yourself.

- Verify `.squad/team.md` exists before delegating work to the team. If it is
  missing, the activation-job bootstrap step failed — call `noop` and explain
  why instead of proceeding.
- Coordinate work through the Squad team already defined in `.squad/` rather
  than proposing a brand-new team from scratch.
- Treat `/tmp/gh-aw/repo-memory/squad-state/` as the durable Squad team root.
  Do not overwrite `SQUAD_TEAM_ROOT`; Squad runtime state changes written there
  are persisted after the run.
