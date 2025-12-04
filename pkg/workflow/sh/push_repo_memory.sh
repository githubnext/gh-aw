#!/bin/bash
set -e

# Push repo-memory changes to git branch
# Parameters (via environment variables):
#   MEMORY_DIR: Path to the repo-memory directory
#   TARGET_REPO: Target repository (owner/name)
#   BRANCH_NAME: Branch name to push to
#   MAX_FILE_SIZE: Maximum file size in bytes
#   MAX_FILE_COUNT: Maximum number of files per commit
#   FILE_GLOB_FILTER: Optional space-separated list of file patterns (e.g., "*.md *.txt")
#   GH_TOKEN: GitHub token for authentication

cd "$MEMORY_DIR"

# Configure git user as GitHub Actions bot
git config user.name "github-actions[bot]"
git config user.email "41898282+github-actions[bot]@users.noreply.github.com"

# Check if we have any changes to commit
if [ -n "$(git status --porcelain)" ]; then
  echo "Changes detected in repo memory, committing and pushing..."
  
  # Stage all changes
  git add .
  
  # Validate file patterns if filter is set
  if [ -n "$FILE_GLOB_FILTER" ]; then
    echo "Validating file patterns: $FILE_GLOB_FILTER"
    # Convert space-separated globs to regex alternation
    PATTERN=$(echo "$FILE_GLOB_FILTER" | sed 's/\*\./\\./g' | sed 's/\*/[^/]*/g' | sed 's/ /|/g')
    INVALID_FILES=$(git diff --cached --name-only | grep -v -E "^($PATTERN)$" || true)
    if [ -n "$INVALID_FILES" ]; then
      echo "Error: Files not matching allowed patterns detected:"
      echo "$INVALID_FILES"
      echo "Allowed patterns: $FILE_GLOB_FILTER"
      exit 1
    fi
  fi
  
  # Check file sizes
  echo "Checking file sizes (max: $MAX_FILE_SIZE bytes)..."
  TOO_LARGE=$(git diff --cached --name-only | while read -r file; do
    if [ -f "$file" ]; then
      SIZE=$(stat -f%z "$file" 2>/dev/null || stat -c%s "$file" 2>/dev/null)
      if [ "$SIZE" -gt "$MAX_FILE_SIZE" ]; then
        echo "$file ($SIZE bytes)"
      fi
    fi
  done)
  
  if [ -n "$TOO_LARGE" ]; then
    echo "Error: Files exceeding size limit detected:"
    echo "$TOO_LARGE"
    exit 1
  fi
  
  # Check file count
  echo "Checking file count (max: $MAX_FILE_COUNT files)..."
  FILE_COUNT=$(git diff --cached --name-only | wc -l | tr -d ' ')
  if [ "$FILE_COUNT" -gt "$MAX_FILE_COUNT" ]; then
    echo "Error: Too many files ($FILE_COUNT > $MAX_FILE_COUNT)"
    exit 1
  fi
  
  # Commit changes
  echo "Committing $FILE_COUNT file(s)..."
  git commit -m "Update repo memory from workflow run $GITHUB_RUN_ID"
  
  # Pull with merge strategy (ours wins on conflicts)
  echo "Pulling latest changes from $BRANCH_NAME..."
  git pull --no-rebase -X ours "https://x-access-token:${GH_TOKEN}@github.com/${TARGET_REPO}.git" "$BRANCH_NAME"
  
  # Push changes
  echo "Pushing changes to $BRANCH_NAME..."
  git push "https://x-access-token:${GH_TOKEN}@github.com/${TARGET_REPO}.git" HEAD:"$BRANCH_NAME"
  echo "Successfully pushed changes to $BRANCH_NAME branch"
else
  echo "No changes detected in repo memory"
fi
