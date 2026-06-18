#!/bin/sh

#
# configure_git_credentials.sh - Configure Git identity, safe directory, and remote authentication
#
# Sets up Git global configuration for use in GitHub Actions workflows and the gh-aw-node
# container. Always configures git identity and safe.directory trust. Configures the remote
# URL for authentication when credentials are provided; silently skips auth when any required
# credential variable (GITHUB_REPOSITORY, GITHUB_SERVER_URL, GITHUB_TOKEN) is absent.
#
# When a checkout manifest is present, every cross-repository checkout subdirectory it
# records is also trusted as a safe.directory so that safe-outputs handlers can run git
# inside those subdirectories without hitting "dubious ownership" errors.
#
# Required environment variables:
#   GITHUB_WORKSPACE     - Workspace directory path (for safe.directory)
#
# Optional environment variables:
#   RUNNER_TEMP              - Runner temp dir; used to locate the checkout manifest
#   GH_AW_CHECKOUT_MANIFEST  - Explicit path to the checkout manifest (overrides default)
#
# Optional environment variables for remote authentication:
#   GITHUB_REPOSITORY    - Repository slug (e.g., "org/repo")
#   GITHUB_SERVER_URL    - GitHub server URL (with or without https:// prefix)
#   GITHUB_TOKEN         - Authentication token; falls back to GIT_TOKEN
#
# Exit codes:
#   0 - Success
#   1 - Error

set -eu

# Configure git identity
git config --global user.email "github-actions[bot]@users.noreply.github.com"
git config --global user.name "github-actions[bot]"
git config --global am.keepcr true

# Trust the workspace directory to avoid "dubious ownership" errors
# when the repository is owned by a different user (e.g., in mounted containers)
if [ -n "${GITHUB_WORKSPACE:-}" ]; then
  git config --global --add safe.directory "${GITHUB_WORKSPACE}"
fi

# Trust cross-repository checkout directories recorded in the checkout manifest.
# Cross-repo checkouts live in subdirectories of the workspace (e.g.
# "${GITHUB_WORKSPACE}/github"), each a separate git repository whose top-level is
# not GITHUB_WORKSPACE. The safe-outputs handlers run git inside these
# subdirectories, so without trusting them git aborts with "dubious ownership"
# (surfacing as errors such as "Failed to pin branch").
MANIFEST_PATH="${GH_AW_CHECKOUT_MANIFEST:-}"
if [ -z "${MANIFEST_PATH}" ] && [ -n "${RUNNER_TEMP:-}" ]; then
  MANIFEST_PATH="${RUNNER_TEMP}/gh-aw/safeoutputs/checkout-manifest.json"
fi
if [ -n "${GITHUB_WORKSPACE:-}" ] && [ -n "${MANIFEST_PATH}" ] && [ -f "${MANIFEST_PATH}" ] && command -v node >/dev/null 2>&1; then
  GH_AW_MANIFEST_PATH="${MANIFEST_PATH}" node -e '
    const fs = require("fs");
    const path = require("path");
    const ws = process.env.GITHUB_WORKSPACE || "";
    try {
      const manifest = JSON.parse(fs.readFileSync(process.env.GH_AW_MANIFEST_PATH, "utf8"));
      if (manifest && typeof manifest === "object") {
        const seen = new Set();
        for (const entry of Object.values(manifest)) {
          if (!entry || typeof entry !== "object") continue;
          const p = typeof entry.path === "string" ? entry.path : "";
          if (!p) continue;
          if (/[\r\n\0]/.test(p)) continue;
          // Only trust paths that resolve to a location inside the workspace,
          // guarding against path traversal in a malformed/hostile manifest.
          const resolved = path.resolve(ws, p);
          if (/[\r\n\0]/.test(resolved)) continue;
          const rel = path.relative(ws, resolved);
          if (rel === "" || rel.startsWith("..") || path.isAbsolute(rel)) continue;
          if (seen.has(resolved)) continue;
          seen.add(resolved);
          process.stdout.write(resolved + "\n");
        }
      }
    } catch (_e) {
      /* ignore missing or malformed manifest */
    }
  ' 2>/dev/null | while IFS= read -r dir; do
    [ -n "${dir}" ] && git config --global --add safe.directory "${dir}"
  done
fi

# Configure remote URL authentication when all required credentials are present.
# Silently skips when any variable is absent (e.g., inside the safeoutputs container
# where GITHUB_SERVER_URL is intentionally not exposed).
REPO="${GITHUB_REPOSITORY:-}"
URL="${GITHUB_SERVER_URL:-}"
TOKEN="${GITHUB_TOKEN:-${GIT_TOKEN:-}}"

if [ -n "${REPO}" ] && [ -n "${URL}" ] && [ -n "${TOKEN}" ]; then
  URL_STRIPPED="${URL#https://}"
  git remote set-url origin "https://x-access-token:${TOKEN}@${URL_STRIPPED}/${REPO}.git"
fi

echo "Git configured with standard GitHub Actions identity" >&2
