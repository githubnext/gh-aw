set -e

# Checkout PR branch for pull_request events
PR_BRANCH="${{ github.event.pull_request.head.ref }}"

echo "Checking out PR branch: $PR_BRANCH"
git fetch origin "$PR_BRANCH"
git checkout "$PR_BRANCH"
