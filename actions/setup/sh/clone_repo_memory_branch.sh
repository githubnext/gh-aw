#!/usr/bin/env bash
# Clone repo-memory branch script
# Clones a repo-memory branch or creates an orphan branch if it doesn't exist
#
# Required environment variables:
#   GH_TOKEN: GitHub token for authentication
#   BRANCH_NAME: Name of the branch to clone
#   TARGET_REPO: Repository to clone from (e.g., owner/repo)
#   MEMORY_DIR: Directory to clone into
#   CREATE_ORPHAN: Whether to create orphan branch if it doesn't exist (true/false)

set -e

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

# Try to clone the branch (don't fail if it doesn't exist)
set +e
git clone --depth 1 --single-branch --branch "$BRANCH_NAME" "https://x-access-token:${GH_TOKEN}@github.com/${TARGET_REPO}.git" "$MEMORY_DIR" 2>/dev/null
CLONE_EXIT_CODE=$?
set -e

if [ $CLONE_EXIT_CODE -ne 0 ]; then
  # Clone failed - branch doesn't exist
  if [ "$CREATE_ORPHAN" = "true" ]; then
    echo "Branch $BRANCH_NAME does not exist, creating orphan branch"
    mkdir -p "$MEMORY_DIR"
    cd "$MEMORY_DIR"
    git init
    git checkout --orphan "$BRANCH_NAME"
    git config user.name "github-actions[bot]"
    git config user.email "github-actions[bot]@users.noreply.github.com"
    git remote add origin "https://x-access-token:${GH_TOKEN}@github.com/${TARGET_REPO}.git"
  else
    echo "Branch $BRANCH_NAME does not exist and create-orphan is false, skipping"
    mkdir -p "$MEMORY_DIR"
  fi
else
  # Clone succeeded
  echo "Successfully cloned $BRANCH_NAME branch"
  cd "$MEMORY_DIR"
  git config user.name "github-actions[bot]"
  git config user.email "github-actions[bot]@users.noreply.github.com"
fi

# Ensure memory directory exists
mkdir -p "$MEMORY_DIR"
echo "Repo memory directory ready at $MEMORY_DIR"
