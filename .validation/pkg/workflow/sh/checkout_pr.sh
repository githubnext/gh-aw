set -e

# Determine PR number based on event type
if [ "${{ github.event_name }}" = "issue_comment" ]; then
  PR_NUMBER="${{ github.event.issue.number }}"
elif [ "${{ github.event_name }}" = "pull_request_review_comment" ]; then
  PR_NUMBER="${{ github.event.pull_request.number }}"
elif [ "${{ github.event_name }}" = "pull_request_review" ]; then
  PR_NUMBER="${{ github.event.pull_request.number }}"
fi

echo "Fetching PR #$PR_NUMBER..."
gh pr checkout "$PR_NUMBER"
