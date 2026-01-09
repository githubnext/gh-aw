#!/bin/bash
# delete-old-copilot-branches.sh - Find and delete old copilot/* branches with closed/merged PRs
#
# This script identifies copilot/* branches that:
# - Have a closed or merged PR
# - Last commit is at least 1 day old
#
# The script outputs git commands to delete these branches.
#
# Usage:
#   ./scripts/delete-old-copilot-branches.sh
#
# Requirements:
#   - GitHub CLI (gh) installed and authenticated
#   - git installed
#   - Run from the repository root directory
#
# Environment Variables:
#   GITHUB_TOKEN - Optional. GitHub token for authentication (useful in CI/CD)
#
# Exit codes:
#   0 - Success
#   1 - Error (missing dependencies, git errors, etc.)

set -euo pipefail

# Show usage if --help or -h is passed
if [[ "${1:-}" == "--help" ]] || [[ "${1:-}" == "-h" ]]; then
    sed -n '2,/^$/p' "$0" | sed 's/^# \?//'
    exit 0
fi

# Colors for output
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

# Check for required dependencies
if ! command -v gh &> /dev/null; then
    echo -e "${RED}Error: GitHub CLI (gh) is not installed${NC}" >&2
    echo "Install it from: https://cli.github.com/" >&2
    exit 1
fi

if ! command -v git &> /dev/null; then
    echo -e "${RED}Error: git is not installed${NC}" >&2
    exit 1
fi

# Ensure we're authenticated with GitHub CLI
# In CI environments, GITHUB_TOKEN may be set
if [ -n "${GITHUB_TOKEN:-}" ]; then
    export GH_TOKEN="$GITHUB_TOKEN"
fi

if ! gh auth status &> /dev/null; then
    echo -e "${RED}Error: Not authenticated with GitHub CLI${NC}" >&2
    echo "Run: gh auth login" >&2
    echo "Or set GITHUB_TOKEN environment variable" >&2
    exit 1
fi

echo "Finding copilot/* branches with closed/merged PRs (last commit 1+ day old)..."
echo ""

# Get current time in seconds since epoch
current_time=$(date +%s)
one_day_ago=$((current_time - 86400))

# Track branches to delete
branches_to_delete=()

# Fetch latest remote information
echo "Fetching remote branches..."
git fetch origin --prune &> /dev/null || {
    echo -e "${RED}Error: Failed to fetch from remote${NC}" >&2
    exit 1
}

# Get all remote copilot/* branches
remote_branches=$(git branch -r | grep "origin/copilot/" | sed 's|origin/||' | grep -v HEAD || true)

if [ -z "$remote_branches" ]; then
    echo -e "${YELLOW}No copilot/* branches found${NC}"
    exit 0
fi

echo -e "${BLUE}Found $(echo "$remote_branches" | wc -l) copilot/* branch(es)${NC}"
echo ""

# Process each branch
while IFS= read -r branch; do
    # Skip empty lines
    [ -z "$branch" ] && continue
    
    # Remove any leading/trailing whitespace
    branch=$(echo "$branch" | xargs)
    
    echo -e "${BLUE}Checking: ${NC}$branch"
    
    # Get the last commit date for this branch
    commit_date=$(git log -1 --format=%ct "origin/$branch" 2>/dev/null || echo "0")
    
    if [ "$commit_date" = "0" ]; then
        echo -e "  ${YELLOW}⚠️  Could not determine commit date, skipping${NC}"
        echo ""
        continue
    fi
    
    # Check if commit is at least 1 day old
    if [ "$commit_date" -ge "$one_day_ago" ]; then
        commit_age_hours=$(( (current_time - commit_date) / 3600 ))
        echo -e "  ${YELLOW}⏱️  Last commit is only ${commit_age_hours}h old (< 24h), skipping${NC}"
        echo ""
        continue
    fi
    
    # Calculate age in days for display
    commit_age_days=$(( (current_time - commit_date) / 86400 ))
    echo -e "  ${GREEN}✓ Last commit is ${commit_age_days} day(s) old${NC}"
    
    # Check PR status using GitHub CLI
    # Search for PRs from this branch
    pr_status=$(gh pr list --head "$branch" --state all --json number,state,headRefName --jq '.[0] | select(.headRefName == "'"$branch"'") | .state' 2>/dev/null || echo "")
    
    if [ -z "$pr_status" ]; then
        echo -e "  ${YELLOW}⚠️  No PR found for this branch, skipping${NC}"
        echo ""
        continue
    fi
    
    # Check if PR is closed or merged
    if [ "$pr_status" = "CLOSED" ] || [ "$pr_status" = "MERGED" ]; then
        echo -e "  ${GREEN}✓ PR is ${pr_status}${NC}"
        branches_to_delete+=("$branch")
        echo -e "  ${GREEN}→ Will be deleted${NC}"
    else
        echo -e "  ${YELLOW}⚠️  PR is ${pr_status}, skipping${NC}"
    fi
    
    echo ""
done <<< "$remote_branches"

# Output deletion commands
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

if [ ${#branches_to_delete[@]} -eq 0 ]; then
    echo -e "${GREEN}No branches to delete${NC}"
    echo ""
    echo "All copilot/* branches either:"
    echo "  - Have open PRs"
    echo "  - Have recent commits (< 24h old)"
    echo "  - Have no associated PR"
else
    echo -e "${GREEN}Found ${#branches_to_delete[@]} branch(es) to delete:${NC}"
    echo ""
    
    echo "# Commands to delete branches:"
    echo ""
    
    for branch in "${branches_to_delete[@]}"; do
        echo "git push origin --delete $branch"
    done
    
    echo ""
    echo -e "${YELLOW}Note: Review the commands above before executing${NC}"
    echo "You can run them individually or pipe this output through bash"
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

exit 0
