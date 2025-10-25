# Check current git status
echo "Current git status:"
git status

# Extract branch name from JSONL output
BRANCH_NAME=""
if [ -f "$GH_AW_SAFE_OUTPUTS" ]; then
  echo "Checking for branch name in JSONL output..."
  while IFS= read -r line; do
    if [ -n "$line" ]; then
      # Extract branch from create-pull-request line using simple grep and sed
      # Note: types use underscores (normalized by safe-outputs MCP server)
      if echo "$line" | grep -q '"type"[[:space:]]*:[[:space:]]*"create_pull_request"'; then
        echo "Found create_pull_request line: $line"
        # Extract branch value using sed
        BRANCH_NAME=$(echo "$line" | sed -n 's/.*"branch"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')
        if [ -n "$BRANCH_NAME" ]; then
          echo "Extracted branch name from create_pull_request: $BRANCH_NAME"
          break
        fi
      # Extract branch from push_to_pull_request_branch line using simple grep and sed
      # Note: types use underscores (normalized by safe-outputs MCP server)
      elif echo "$line" | grep -q '"type"[[:space:]]*:[[:space:]]*"push_to_pull_request_branch"'; then
        echo "Found push_to_pull_request_branch line: $line"
        # Extract branch value using sed
        BRANCH_NAME=$(echo "$line" | sed -n 's/.*"branch"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')
        if [ -n "$BRANCH_NAME" ]; then
          echo "Extracted branch name from push_to_pull_request_branch: $BRANCH_NAME"
          break
        fi
      fi
    fi
  done < "$GH_AW_SAFE_OUTPUTS"
fi

# Determine which branch to use for patch generation
TARGET_BRANCH=""

if [ -n "$BRANCH_NAME" ]; then
  echo "Branch name from safe-outputs: $BRANCH_NAME"
  # Check if the branch exists locally
  if git show-ref --verify --quiet refs/heads/$BRANCH_NAME; then
    echo "Branch $BRANCH_NAME exists locally"
    TARGET_BRANCH="$BRANCH_NAME"
  else
    echo "Branch $BRANCH_NAME does not exist locally, falling back to current HEAD"
    # Get the current branch name
    CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
    if [ "$CURRENT_BRANCH" != "HEAD" ]; then
      echo "Using current branch: $CURRENT_BRANCH"
      TARGET_BRANCH="$CURRENT_BRANCH"
    else
      echo "Warning: Detached HEAD state, using HEAD directly"
      TARGET_BRANCH="HEAD"
    fi
  fi
else
  echo "No branch name found in safe-outputs, using current branch"
  # Get the current branch name
  CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
  if [ "$CURRENT_BRANCH" != "HEAD" ]; then
    echo "Using current branch: $CURRENT_BRANCH"
    TARGET_BRANCH="$CURRENT_BRANCH"
  else
    echo "Warning: Detached HEAD state, using HEAD directly"
    TARGET_BRANCH="HEAD"
  fi
fi

# Generate patch if we have a target branch
if [ -n "$TARGET_BRANCH" ]; then
  echo "Generating patch for: $TARGET_BRANCH"
  
  # Determine the base ref for patch generation
  if [ "$TARGET_BRANCH" = "HEAD" ]; then
    # For detached HEAD, use merge-base with default branch
    DEFAULT_BRANCH="${{ github.event.repository.default_branch }}"
    echo "Default branch: $DEFAULT_BRANCH"
    git fetch origin $DEFAULT_BRANCH
    BASE_REF=$(git merge-base origin/$DEFAULT_BRANCH HEAD)
    echo "Using merge-base as base: $BASE_REF"
  elif git show-ref --verify --quiet refs/remotes/origin/$TARGET_BRANCH; then
    echo "Using origin/$TARGET_BRANCH as base for patch generation"
    BASE_REF="origin/$TARGET_BRANCH"
  else
    echo "origin/$TARGET_BRANCH does not exist, using merge-base with default branch"
    # Get the default branch name
    DEFAULT_BRANCH="${{ github.event.repository.default_branch }}"
    echo "Default branch: $DEFAULT_BRANCH"
    # Fetch the default branch to ensure it's available locally
    git fetch origin $DEFAULT_BRANCH
    # Find merge base between default branch and current branch
    BASE_REF=$(git merge-base origin/$DEFAULT_BRANCH $TARGET_BRANCH)
    echo "Using merge-base as base: $BASE_REF"
  fi
  
  # Generate patch from the determined base to the target
  git format-patch "$BASE_REF".."$TARGET_BRANCH" --stdout > /tmp/gh-aw/aw.patch || echo "Failed to generate patch from branch" > /tmp/gh-aw/aw.patch
  echo "Patch file created from: $TARGET_BRANCH (base: $BASE_REF)"
else
  echo "No target branch determined, no patch generation"
fi

# Show patch info if it exists
if [ -f /tmp/gh-aw/aw.patch ]; then
  ls -la /tmp/gh-aw/aw.patch
  # Show the first 50 lines of the patch for review
  echo '## Git Patch' >> $GITHUB_STEP_SUMMARY
  echo '' >> $GITHUB_STEP_SUMMARY
  echo '```diff' >> $GITHUB_STEP_SUMMARY
  head -500 /tmp/gh-aw/aw.patch >> $GITHUB_STEP_SUMMARY || echo "Could not display patch contents" >> $GITHUB_STEP_SUMMARY
  echo '...' >> $GITHUB_STEP_SUMMARY
  echo '```' >> $GITHUB_STEP_SUMMARY
  echo '' >> $GITHUB_STEP_SUMMARY
fi
