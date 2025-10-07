set -e

# Checkout PR branch for ready_for_review event
PR_BRANCH="${{ github.event.pull_request.head.ref }}"

echo "Checking out PR branch: $PR_BRANCH"
git fetch origin "$PR_BRANCH"
git checkout "$PR_BRANCH"
