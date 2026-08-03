#!/usr/bin/env bash
# Installs a pinned release of the Squad CLI (https://github.com/bradygaster/squad)
# and scaffolds the .squad/ team state + .github/agents/squad.agent.md so the
# copilot engine can be launched with `--agent squad`.
#
# Safe to invoke directly as the pre-agent setup step, or sourced from a
# custom setup script when a workflow needs additional bootstrapping.
set -euo pipefail

SQUAD_CLI_VERSION="${SQUAD_CLI_VERSION:-0.11.0}"

echo "Installing @bradygaster/squad-cli@${SQUAD_CLI_VERSION}..."
npm install -g "@bradygaster/squad-cli@${SQUAD_CLI_VERSION}"

echo "Squad CLI version: $(squad --version 2>/dev/null || echo unknown)"

# `squad init` is idempotent (safe to run multiple times), so re-running it on
# a repository that already has a `.squad/` directory is a no-op refresh
# rather than an error.
echo "Initializing Squad team state (idempotent)..."
squad init --preset default
