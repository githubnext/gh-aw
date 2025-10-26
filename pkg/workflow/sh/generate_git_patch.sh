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
      # Get the default branch name
      DEFAULT_BRANCH="${{ github.event.repository.default_branch }}"
      echo "Default branch: $DEFAULT_BRANCH"
      # Fetch the default branch to ensure it's available locally
      git fetch origin $DEFAULT_BRANCH
      # Find merge base between default branch and current branch
      BASE_REF=$(git merge-base origin/$DEFAULT_BRANCH $BRANCH_NAME)
      echo "Using merge-base as base: $BASE_REF"
    fi
    
    # Diagnostic logging before patch generation
    echo "========================================="
    echo "PRE-PATCH GENERATION DIAGNOSTICS"
    echo "========================================="
    echo "Recent commits (last 5):"
    git log --oneline -5 || echo "Failed to show recent commits"
    echo ""
    echo "Git status:"
    git status || echo "Failed to show git status"
    echo ""
    echo "Patch size preview (stat):"
    git diff --stat "$BASE_REF".."$BRANCH_NAME" || echo "Failed to show diff stat"
    echo ""
    echo "Exact git format-patch command to be executed:"
    echo "git format-patch \"$BASE_REF\"..\"$BRANCH_NAME\" --stdout > /tmp/gh-aw/aw.patch"
    echo "========================================="
    echo ""
    
    # Generate patch from the determined base to the branch
    git format-patch "$BASE_REF".."$BRANCH_NAME" --stdout > /tmp/gh-aw/aw.patch || echo "Failed to generate patch from branch" > /tmp/gh-aw/aw.patch
    
    # Diagnostic logging after patch generation
    echo ""
    echo "========================================="
    echo "POST-PATCH GENERATION DIAGNOSTICS"
    echo "========================================="
    if [ -f /tmp/gh-aw/aw.patch ]; then
      PATCH_SIZE=$(wc -c < /tmp/gh-aw/aw.patch)
      PATCH_LINES=$(wc -l < /tmp/gh-aw/aw.patch)
      PATCH_SIZE_KB=$((PATCH_SIZE / 1024))
      echo "Patch file size: $PATCH_SIZE bytes ($PATCH_SIZE_KB KB)"
      echo "Patch file lines: $PATCH_LINES"
      echo ""
      echo "Commits included in patch:"
      git log --oneline "$BASE_REF".."$BRANCH_NAME" || echo "Failed to list commits in patch"
      echo ""
      echo "Number of commits in patch:"
      git rev-list --count "$BASE_REF".."$BRANCH_NAME" || echo "Failed to count commits"
    else
      echo "Patch file was not created"
    fi
    echo "========================================="
    echo ""
    
    echo "Patch file created from branch: $BRANCH_NAME (base: $BASE_REF)"
  else
    echo "Branch $BRANCH_NAME does not exist, no patch"
  fi
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
