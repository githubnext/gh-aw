# Diagnostic logging: Show recent commits before patch generation
echo "=== Diagnostic: Recent commits (last 5) ==="
git log --oneline -5 || echo "Failed to show git log"

# Check current git status
echo ""
echo "=== Diagnostic: Current git status ==="
git status

# Extract branch name from JSONL output
BRANCH_NAME=""
if [ -f "$GH_AW_SAFE_OUTPUTS" ]; then
  echo ""
  echo "Checking for branch name in JSONL output..."
  while IFS= read -r line; do
    if [ -n "$line" ]; then
      # Extract branch from create-pull-request line using simple grep and sed
      # Note: types use underscores (normalized by safe-outputs MCP server)
      if echo "$line" | grep -q '"type"[[:space:]]*:[[:space:]]*"create_pull_request"'; then
        echo "Found create_pull_request line: $line"
        # Extract branch value using sed
        BRANCH_NAME="$(echo "$line" | sed -n 's/.*"branch"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')"
        if [ -n "$BRANCH_NAME" ]; then
          echo "Extracted branch name from create_pull_request: $BRANCH_NAME"
          break
        fi
      # Extract branch from push_to_pull_request_branch line using simple grep and sed
      # Note: types use underscores (normalized by safe-outputs MCP server)
      elif echo "$line" | grep -q '"type"[[:space:]]*:[[:space:]]*"push_to_pull_request_branch"'; then
        echo "Found push_to_pull_request_branch line: $line"
        # Extract branch value using sed
        BRANCH_NAME="$(echo "$line" | sed -n 's/.*"branch"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')"
        if [ -n "$BRANCH_NAME" ]; then
          echo "Extracted branch name from push_to_pull_request_branch: $BRANCH_NAME"
          break
        fi
      fi
    fi
  done < "$GH_AW_SAFE_OUTPUTS"
fi

# If no branch or branch doesn't exist, no patch
if [ -z "$BRANCH_NAME" ]; then
  echo "No branch found, no patch generation"
fi

# If we have a branch name, check if that branch exists and get its diff
if [ -n "$BRANCH_NAME" ]; then
  echo "Looking for branch: $BRANCH_NAME"
  # Check if the branch exists
  if git show-ref --verify --quiet refs/heads/$BRANCH_NAME; then
    echo "Branch $BRANCH_NAME exists, generating patch from branch changes"
    
    # Check if origin/$BRANCH_NAME exists to use as base
    if git show-ref --verify --quiet refs/remotes/origin/$BRANCH_NAME; then
      echo "Using origin/$BRANCH_NAME as base for patch generation"
      BASE_REF="origin/$BRANCH_NAME"
    else
      echo "origin/$BRANCH_NAME does not exist, using merge-base with default branch"
      # Use the default branch name from environment variable
      echo "Default branch: $DEFAULT_BRANCH"
      # Fetch the default branch to ensure it's available locally
      git fetch origin $DEFAULT_BRANCH
      # Find merge base between default branch and current branch
      BASE_REF="$(git merge-base origin/$DEFAULT_BRANCH $BRANCH_NAME)"
      echo "Using merge-base as base: $BASE_REF"
    fi
    
    # Diagnostic logging: Show diff stats before generating patch
    echo ""
    echo "=== Diagnostic: Diff stats for patch generation ==="
    echo "Command: git diff --stat $BASE_REF..$BRANCH_NAME"
    git diff --stat "$BASE_REF".."$BRANCH_NAME" || echo "Failed to show diff stats"
    
    # Diagnostic logging: Count commits to be included
    echo ""
    echo "=== Diagnostic: Commits to be included in patch ==="
    COMMIT_COUNT="$(git rev-list --count "$BASE_REF".."$BRANCH_NAME" 2>/dev/null || echo "0")"
    echo "Number of commits: $COMMIT_COUNT"
    if [ "$COMMIT_COUNT" -gt 0 ]; then
      echo "Commit SHAs:"
      git log --oneline "$BASE_REF".."$BRANCH_NAME" || echo "Failed to list commits"
    fi
    
    # Diagnostic logging: Show the exact command being used
    echo ""
    echo "=== Diagnostic: Generating patch ==="
    echo "Command: git format-patch $BASE_REF..$BRANCH_NAME --stdout > /tmp/gh-aw/aw.patch"
    
    # Generate patch from the determined base to the branch
    git format-patch "$BASE_REF".."$BRANCH_NAME" --stdout > /tmp/gh-aw/aw.patch || echo "Failed to generate patch from branch" > /tmp/gh-aw/aw.patch
    echo "Patch file created from branch: $BRANCH_NAME (base: $BASE_REF)"
  else
    echo "Branch $BRANCH_NAME does not exist, no patch"
  fi
fi

# Show patch info if it exists
if [ -f /tmp/gh-aw/aw.patch ]; then
  echo ""
  echo "=== Diagnostic: Patch file information ==="
  ls -lh /tmp/gh-aw/aw.patch
  
  # Get patch file size in KB
  PATCH_SIZE="$(du -k /tmp/gh-aw/aw.patch | cut -f1)"
  echo "Patch file size: ${PATCH_SIZE} KB"
  
  # Count lines in patch
  PATCH_LINES="$(wc -l < /tmp/gh-aw/aw.patch)"
  echo "Patch file lines: $PATCH_LINES"
  
  # Extract and count commits from patch file (each commit starts with "From <sha>")
  PATCH_COMMITS="$(grep -c "^From [0-9a-f]\{40\}" /tmp/gh-aw/aw.patch 2>/dev/null || echo "0")"
  echo "Commits included in patch: $PATCH_COMMITS"
  
  # List commit SHAs in the patch
  if [ "$PATCH_COMMITS" -gt 0 ]; then
    echo "Commit SHAs in patch:"
    grep "^From [0-9a-f]\{40\}" /tmp/gh-aw/aw.patch | sed 's/^From \([0-9a-f]\{40\}\).*/  \1/' || echo "Failed to extract commit SHAs"
  fi
  
  # Show the first 50 lines of the patch for review
  {
    echo '## Git Patch'
    echo ''
    echo '```diff'
    head -500 /tmp/gh-aw/aw.patch || echo "Could not display patch contents"
    echo '...'
    echo '```'
    echo ''
  } >> "$GITHUB_STEP_SUMMARY"
fi
