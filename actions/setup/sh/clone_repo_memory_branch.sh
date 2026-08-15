#!/usr/bin/env bash
set +o histexpand

# Clone repo-memory branch script
# Clones a repo-memory branch or creates an orphan branch if it doesn't exist
#
# Required environment variables:
#   GH_TOKEN: GitHub token for authentication
#   BRANCH_NAME: Name of the branch to clone
#   TARGET_REPO: Repository to clone from (e.g., owner/repo)
#   MEMORY_DIR: Directory to clone into
#   CREATE_ORPHAN: Whether to create orphan branch if it doesn't exist (true/false)
#   GITHUB_SERVER_URL: GitHub server URL (e.g., https://github.com or https://ghe.company.com)

set -e

scrub_git_config_entries() {
  local key_prefix="$1"
  while IFS= read -r key_name; do
    [ -n "$key_name" ] || continue
    git config --unset-all "$key_name" >/dev/null 2>&1 || true
  done < <(
    git config --local --name-only --list 2>/dev/null \
      | grep -E -i "^${key_prefix}\\." \
      | sort -u
  )
}

has_symlinked_git_metadata() {
  local repo_root="$1"
  local path
  for path in "$repo_root/.git" "$repo_root/.git/config" "$repo_root/.git/info" "$repo_root/.git/hooks"; do
    if [ -L "$path" ]; then
      return 0
    fi
  done
  return 1
}

harden_repo_memory_git_state() {
  local repo_root="$1"
  local origin_url="$2"

  if [ ! -d "$repo_root/.git" ] && [ ! -L "$repo_root/.git" ]; then
    return 0
  fi

  if has_symlinked_git_metadata "$repo_root"; then
    echo "WARNING: Detected symlinked repo-memory git metadata; reinitializing git metadata"
    rm -rf "$repo_root/.git"
    git init -q
    git checkout --orphan "$BRANCH_NAME"
    git remote remove origin >/dev/null 2>&1 || true
    git remote add origin "$origin_url"
  fi

  if [ -d .git/hooks ]; then
    find .git/hooks -type f ! -name '*.sample' -delete
  fi
  mkdir -p .git/info
  rm -f .git/info/exclude .git/info/attributes .git/info/grafts .git/info/sparse-checkout

  git config --unset-all core.attributesFile >/dev/null 2>&1 || true
  git config --unset-all core.fsmonitor >/dev/null 2>&1 || true
  git config --unset-all core.sshCommand >/dev/null 2>&1 || true
  git config --unset-all core.hooksPath >/dev/null 2>&1 || true
  scrub_git_config_entries include
  scrub_git_config_entries includeif
  scrub_git_config_entries credential
  scrub_git_config_entries alias
  scrub_git_config_entries filter
  scrub_git_config_entries merge

  git config user.name "github-actions[bot]"
  git config user.email "github-actions[bot]@users.noreply.github.com"
  git config core.hooksPath /dev/null
  git config core.fsmonitor false
}

# Validate required environment variables
if [ -z "$GH_TOKEN" ]; then
  echo "ERROR: GH_TOKEN environment variable is required"
  exit 1
fi

if [ -z "$BRANCH_NAME" ]; then
  echo "ERROR: BRANCH_NAME environment variable is required"
  exit 1
fi

if [ -z "$TARGET_REPO" ]; then
  echo "ERROR: TARGET_REPO environment variable is required"
  exit 1
fi

if [ -z "$MEMORY_DIR" ]; then
  echo "ERROR: MEMORY_DIR environment variable is required"
  exit 1
fi

if [ -z "$CREATE_ORPHAN" ]; then
  echo "ERROR: CREATE_ORPHAN environment variable is required"
  exit 1
fi

# Default to github.com if not set
if [ -z "$GITHUB_SERVER_URL" ]; then
  GITHUB_SERVER_URL="https://github.com"
fi

# Extract host from server URL (remove https:// or http:// prefix)
SERVER_HOST="${GITHUB_SERVER_URL#https://}"
SERVER_HOST="${SERVER_HOST#http://}"
ORIGIN_URL="https://x-access-token:${GH_TOKEN}@${SERVER_HOST}/${TARGET_REPO}.git"

# Try to clone the branch (don't fail if it doesn't exist)
set +e
git clone --depth 1 --single-branch --branch "$BRANCH_NAME" "$ORIGIN_URL" "$MEMORY_DIR" 2>/dev/null
CLONE_EXIT_CODE=$?
set -e

if [ $CLONE_EXIT_CODE -ne 0 ]; then
  # Clone failed - branch doesn't exist
  if [ "$CREATE_ORPHAN" = "true" ]; then
    echo "Branch $BRANCH_NAME does not exist, creating orphan branch"
    mkdir -p "$MEMORY_DIR"
    cd "$MEMORY_DIR"
    if has_symlinked_git_metadata "$MEMORY_DIR"; then
      echo "WARNING: Detected symlinked repo-memory git metadata; reinitializing git metadata"
      rm -rf "$MEMORY_DIR/.git"
    fi
    git init
    git checkout --orphan "$BRANCH_NAME"
    git remote add origin "$ORIGIN_URL"
    harden_repo_memory_git_state "$MEMORY_DIR" "$ORIGIN_URL"
  else
    echo "Branch $BRANCH_NAME does not exist and create-orphan is false, skipping"
    mkdir -p "$MEMORY_DIR"
  fi
else
  # Clone succeeded
  echo "Successfully cloned $BRANCH_NAME branch"
  cd "$MEMORY_DIR"
  harden_repo_memory_git_state "$MEMORY_DIR" "$ORIGIN_URL"
fi

# Ensure memory directory exists
mkdir -p "$MEMORY_DIR"
echo "Repo memory directory ready at $MEMORY_DIR"
