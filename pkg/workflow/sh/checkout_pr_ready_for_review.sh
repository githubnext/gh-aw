set -e

# Checkout PR branch for pull_request events or comment events on PRs
if [ "${{ github.event_name }}" = "pull_request" ]; then
  # For pull_request events, use the head ref directly
  PR_BRANCH="${{ github.event.pull_request.head.ref }}"
  echo "Checking out PR branch: $PR_BRANCH"
  git fetch origin "$PR_BRANCH"
  git checkout "$PR_BRANCH"
else
  # For comment events on PRs, determine PR number and use gh pr checkout
  if [ "${{ github.event_name }}" = "issue_comment" ]; then
    PR_NUMBER="${{ github.event.issue.number }}"
  elif [ "${{ github.event_name }}" = "pull_request_review_comment" ]; then
    PR_NUMBER="${{ github.event.pull_request.number }}"
  elif [ "${{ github.event_name }}" = "pull_request_review" ]; then
    PR_NUMBER="${{ github.event.pull_request.number }}"
  fi
  
  if [ -n "$PR_NUMBER" ]; then
    echo "Fetching PR #$PR_NUMBER..."
    gh pr checkout "$PR_NUMBER"
  fi
fi
