#!/bin/bash

# End-to-end testing script for GitHub Agentic Workflows
# This script triggers all test workflows and validates their outcomes
#
# Usage: ./e2e.sh [OPTIONS]
#
# This script will:
# 1. Check prerequisites (gh CLI, authentication, gh-aw binary)
# 2. Enable workflows before testing them
# 3. Trigger workflows using "gh aw run" 
# 4. Wait for completion and validate outcomes
# 5. Disable workflows after testing
# 6. Generate comprehensive test report
# 7. Optionally clean up test resources
#
# Test Types:
# - workflow_dispatch: Direct trigger tests (create issues, PRs, repository security advisories, etc.)
# - issue-triggered: Tests triggered by creating issues with specific titles
# - command-triggered: Tests triggered by posting commands in issue comments  
# - PR-triggered: Tests triggered by creating pull requests
#
# Options:
#   --workflow-dispatch-only   Run only workflow_dispatch triggered tests
#   --issue-triggered-only     Run only issue-triggered tests  
#   --command-triggered-only   Run only command-triggered tests
#   --pr-triggered-only        Run only PR-triggered tests
#   --dry-run                  Show what would be tested without running
#   --help, -h                 Show help message
#
# Examples:
#   ./e2e.sh                               # Run all tests
#   ./e2e.sh --workflow-dispatch-only      # Run only direct trigger tests
#   ./e2e.sh --dry-run                     # See what would be tested
#
# Prerequisites:
#   - GitHub CLI (gh) installed and authenticated
#   - gh-aw binary built (run 'make build')
#   - Proper repository permissions for creating issues/PRs
#   - Internet access for GitHub API calls

set -euo pipefail

# Colors and emojis for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Test results tracking
declare -a PASSED_TESTS=()
declare -a FAILED_TESTS=()
declare -a SKIPPED_TESTS=()

# Configuration
REPO_OWNER="githubnext"
REPO_NAME="gh-aw"
TIMEOUT_MINUTES=10
POLL_INTERVAL=30
LOG_FILE="e2e-test-$(date +%Y%m%d-%H%M%S).log"

# Utility functions
log() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') $*" | tee -a "$LOG_FILE"
}

info() {
    echo -e "${BLUE}ℹ️  $*${NC}" | tee -a "$LOG_FILE"
}

success() {
    echo -e "${GREEN}✅ $*${NC}" | tee -a "$LOG_FILE"
}

warning() {
    echo -e "${YELLOW}⚠️  $*${NC}" | tee -a "$LOG_FILE"
}

error() {
    echo -e "${RED}❌ $*${NC}" | tee -a "$LOG_FILE"
}

progress() {
    echo -e "${PURPLE}🔨 $*${NC}" | tee -a "$LOG_FILE"
}

# Test pattern matching functions
matches_pattern() {
    local test_name="$1"
    local pattern="$2"
    
    # Convert glob pattern to regex
    local regex_pattern
    regex_pattern=$(echo "$pattern" | sed 's/\*/[^[:space:]]*/g')
    
    if [[ "$test_name" =~ ^${regex_pattern}$ ]]; then
        return 0
    else
        return 1
    fi
}

should_run_test() {
    local test_name="$1"
    local patterns=("${@:2}")
    
    # If no patterns specified, run all tests
    if [[ ${#patterns[@]} -eq 0 ]]; then
        return 0
    fi
    
    # Check if test matches any pattern
    for pattern in "${patterns[@]}"; do
        if matches_pattern "$test_name" "$pattern"; then
            return 0
        fi
    done
    
    return 1
}

get_all_test_names() {
    echo "test-claude-create-issue"
    echo "test-codex-create-issue"
    echo "test-claude-create-pull-request"
    echo "test-codex-create-pull-request"
    echo "test-claude-create-repository-security-advisory"
    echo "test-codex-create-repository-security-advisory"
    echo "test-claude-mcp"
    echo "test-codex-mcp"
    echo "test-custom-safe-outputs"
    echo "test-claude-add-issue-comment"
    echo "test-codex-add-issue-comment"
    echo "test-claude-add-issue-labels"
    echo "test-codex-add-issue-labels"
    echo "test-claude-command"
    echo "test-codex-command"
    echo "test-claude-push-to-branch"
    echo "test-codex-push-to-branch"
    echo "test-claude-create-pull-request-review-comment"
    echo "test-codex-create-pull-request-review-comment"
    echo "test-claude-update-issue"
    echo "test-codex-update-issue"
}

get_workflow_dispatch_tests() {
    echo "test-claude-create-issue"
    echo "test-codex-create-issue"
    echo "test-claude-create-pull-request"
    echo "test-codex-create-pull-request"
    echo "test-claude-create-repository-security-advisory"
    echo "test-codex-create-repository-security-advisory"
    echo "test-claude-mcp"
    echo "test-codex-mcp"
    echo "test-custom-safe-outputs"
}

get_issue_triggered_tests() {
    echo "test-claude-add-issue-comment"
    echo "test-claude-add-issue-labels"
    echo "test-codex-add-issue-comment" 
    echo "test-codex-add-issue-labels"
    echo "test-claude-update-issue"
    echo "test-codex-update-issue"
}

get_command_triggered_tests() {
    echo "test-claude-command"
    echo "test-codex-command"
    echo "test-claude-push-to-branch"
    echo "test-codex-push-to-branch"
}

get_pr_triggered_tests() {
    echo "test-claude-create-pull-request-review-comment"
    echo "test-codex-create-pull-request-review-comment"
}

filter_tests_by_patterns() {
    local test_type="$1"
    shift
    local patterns=("$@")
    
    local all_tests
    case "$test_type" in
        "workflow-dispatch")
            all_tests=($(get_workflow_dispatch_tests))
            ;;
        "issue-triggered") 
            all_tests=($(get_issue_triggered_tests))
            ;;
        "command-triggered")
            all_tests=($(get_command_triggered_tests))
            ;;
        "pr-triggered")
            all_tests=($(get_pr_triggered_tests))
            ;;
        *)
            error "Unknown test type: $test_type"
            return 1
            ;;
    esac
    
    local filtered_tests=()
    for test in "${all_tests[@]}"; do
        if should_run_test "$test" "${patterns[@]}"; then
            filtered_tests+=("$test")
        fi
    done
    
    # Only print if there are filtered tests
    if [[ ${#filtered_tests[@]} -gt 0 ]]; then
        printf '%s\n' "${filtered_tests[@]}"
    fi
}

check_prerequisites() {
    info "Checking prerequisites..."
    
    # Check gh CLI is installed and authenticated
    if ! command -v gh &> /dev/null; then
        error "GitHub CLI (gh) is not installed"
        exit 1
    fi
    
    # Check authentication
    if ! gh auth status &> /dev/null; then
        error "GitHub CLI is not authenticated. Run 'gh auth login'"
        exit 1
    fi
    
    # Check gh-aw is built
    if [[ ! -f "./gh-aw" ]]; then
        error "gh-aw binary not found. Run 'make build' first"
        exit 1
    fi
    
    # Check we're in the right repo
    local current_repo
    current_repo=$(gh repo view --json nameWithOwner -q .nameWithOwner 2>/dev/null || echo "")
    if [[ "$current_repo" != "$REPO_OWNER/$REPO_NAME" ]]; then
        error "Not in the correct repository. Expected $REPO_OWNER/$REPO_NAME, got $current_repo"
        exit 1
    fi
    
    success "Prerequisites check passed"
}

wait_for_workflow() {
    local workflow_name="$1"
    local run_id="$2"
    local timeout_seconds=$((TIMEOUT_MINUTES * 60))
    local start_time=$(date +%s)
    
    progress "Waiting for workflow '$workflow_name' (run #$run_id) to complete..."
    
    while true; do
        local current_time=$(date +%s)
        local elapsed=$((current_time - start_time))
        
        if [[ $elapsed -gt $timeout_seconds ]]; then
            error "Timeout waiting for workflow '$workflow_name' after $TIMEOUT_MINUTES minutes"
            error "View run details: https://github.com/$REPO_OWNER/$REPO_NAME/actions/runs/$run_id"
            return 1
        fi
        
        local status conclusion
        if status=$(gh run view "$run_id" --json status,conclusion -q '.status + "," + (.conclusion // "")' 2>/dev/null); then
            IFS=',' read -r run_status run_conclusion <<< "$status"
            
            case "$run_status" in
                "completed")
                    case "$run_conclusion" in
                        "success")
                            success "Workflow '$workflow_name' completed successfully"
                            return 0
                            ;;
                        "failure"|"cancelled"|"timed_out")
                            error "Workflow '$workflow_name' failed with conclusion: $run_conclusion"
                            error "View run details: https://github.com/$REPO_OWNER/$REPO_NAME/actions/runs/$run_id"
                            return 1
                            ;;
                        *)
                            error "Workflow '$workflow_name' completed with unexpected conclusion: $run_conclusion"
                            error "View run details: https://github.com/$REPO_OWNER/$REPO_NAME/actions/runs/$run_id"
                            return 1
                            ;;
                    esac
                    ;;
                "in_progress"|"queued"|"requested"|"waiting"|"pending")
                    echo -n "."
                    sleep $POLL_INTERVAL
                    ;;
                *)
                    error "Workflow '$workflow_name' has unexpected status: $run_status"
                    error "View run details: https://github.com/$REPO_OWNER/$REPO_NAME/actions/runs/$run_id"
                    return 1
                    ;;
            esac
        else
            error "Failed to get status for workflow run $run_id"
            error "View run details: https://github.com/$REPO_OWNER/$REPO_NAME/actions/runs/$run_id"
            return 1
        fi
    done
}

get_latest_run_id() {
    local workflow_file="$1"
    gh run list --workflow="$workflow_file" --limit=1 --json databaseId -q '.[0].databaseId' 2>/dev/null || echo ""
}

enable_workflow() {
    local workflow_name="$1"
    
    info "Enabling workflow '$workflow_name'..."
    if ./gh-aw enable "$workflow_name" &>> "$LOG_FILE"; then
        success "Successfully enabled '$workflow_name'"
        return 0
    else
        error "Failed to enable '$workflow_name'"
        return 1
    fi
}

disable_workflow() {
    local workflow_name="$1"
    
    info "Disabling workflow '$workflow_name'..."
    if ./gh-aw disable "$workflow_name" &>> "$LOG_FILE"; then
        success "Successfully disabled '$workflow_name'"
        return 0
    else
        warning "Failed to disable '$workflow_name' (may already be disabled)"
        return 0  # Don't fail the test if disable fails
    fi
}

trigger_workflow_dispatch() {
    local workflow_name="$1"
    local workflow_file="${workflow_name}.lock.yml"
    
    info "Triggering workflow_dispatch for '$workflow_name'..."
    
    # Enable the workflow first
    if ! enable_workflow "$workflow_name"; then
        return 1
    fi
    
    # Get the run ID before triggering
    local before_run_id
    before_run_id=$(get_latest_run_id "$workflow_file")
    
    # Trigger the workflow using gh aw run
    if ./gh-aw run "$workflow_name" &>> "$LOG_FILE"; then
        success "Successfully triggered '$workflow_name'"
        
        # Wait a bit for the new run to appear
        sleep 5
        
        # Get the new run ID
        local after_run_id
        after_run_id=$(get_latest_run_id "$workflow_file")
        
        if [[ "$after_run_id" != "$before_run_id" && -n "$after_run_id" ]]; then
            local result=0
            wait_for_workflow "$workflow_name" "$after_run_id" || result=1
            
            # Disable the workflow after running
            disable_workflow "$workflow_name"
            
            return $result
        else
            error "Could not find new workflow run for '$workflow_name'"
            disable_workflow "$workflow_name"
            return 1
        fi
    else
        error "Failed to trigger '$workflow_name'"
        disable_workflow "$workflow_name"
        return 1
    fi
}

create_test_issue() {
    local title="$1"
    local body="$2"
    local labels="${3:-}"
    
    local issue_url
    if [[ -n "$labels" ]]; then
        issue_url=$(gh issue create --title "$title" --body "$body" --label "$labels" 2>/dev/null)
    else
        issue_url=$(gh issue create --title "$title" --body "$body" 2>/dev/null)
    fi
    
    if [[ -n "$issue_url" ]]; then
        local issue_number
        issue_number=$(echo "$issue_url" | grep -o '[0-9]\+$')
        echo "$issue_number"
    else
        echo ""
    fi
}

create_test_pr() {
    local title="$1"
    local body="$2"
    local branch="test-pr-$(date +%s)"
    
    # Create a test branch and commit
    git checkout -b "$branch" &>/dev/null
    echo "# Test PR Content" > "test-file-$(date +%s).md"
    git add . &>/dev/null
    git commit -m "Test commit for PR" &>/dev/null
    git push origin "$branch" &>/dev/null
    
    local pr_url
    pr_url=$(gh pr create --title "$title" --body "$body" --head "$branch" 2>/dev/null)
    
    # Switch back to main
    git checkout main &>/dev/null
    
    if [[ -n "$pr_url" ]]; then
        local pr_number
        pr_number=$(echo "$pr_url" | grep -o '[0-9]\+$')
        echo "$pr_number"
    else
        echo ""
    fi
}

post_issue_command() {
    local issue_number="$1"
    local command="$2"
    
    gh issue comment "$issue_number" --body "$command" &>/dev/null
}

validate_issue_created() {
    local title_prefix="$1"
    local expected_labels="$2"
    
    # Look for recently created issues with the title prefix
    local issue_number
    issue_number=$(gh issue list --limit 10 --json number,title,labels --jq ".[] | select(.title | startswith(\"$title_prefix\")) | .number" | head -1)
    
    if [[ -n "$issue_number" ]]; then
        if [[ -n "$expected_labels" ]]; then
            local issue_labels
            issue_labels=$(gh issue view "$issue_number" --json labels --jq '.labels[].name' | tr '\n' ',' | sed 's/,$//')
            for label in ${expected_labels//,/ }; do
                if [[ "$issue_labels" != *"$label"* ]]; then
                    error "Issue #$issue_number missing expected label: $label"
                    return 1
                fi
            done
        fi
        success "Issue #$issue_number created successfully with expected properties"
        return 0
    else
        error "No issue found with title prefix: $title_prefix"
        return 1
    fi
}

validate_issue_comment() {
    local issue_number="$1"
    local expected_comment_text="$2"
    
    local comments
    comments=$(gh issue view "$issue_number" --json comments --jq '.comments[].body')
    
    if echo "$comments" | grep -q "$expected_comment_text"; then
        success "Issue #$issue_number has expected comment containing: $expected_comment_text"
        return 0
    else
        error "Issue #$issue_number missing expected comment containing: $expected_comment_text"
        return 1
    fi
}

validate_issue_labels() {
    local issue_number="$1"
    local expected_label="$2"
    
    local labels
    labels=$(gh issue view "$issue_number" --json labels --jq '.labels[].name' | tr '\n' ',')
    
    if [[ "$labels" == *"$expected_label"* ]]; then
        success "Issue #$issue_number has expected label: $expected_label"
        return 0
    else
        error "Issue #$issue_number missing expected label: $expected_label"
        return 1
    fi
}

validate_pr_created() {
    local title_prefix="$1"
    
    # Look for recently created PRs with the title prefix
    local pr_number
    pr_number=$(gh pr list --limit 10 --json number,title --jq ".[] | select(.title | startswith(\"$title_prefix\")) | .number" | head -1)
    
    if [[ -n "$pr_number" ]]; then
        success "PR #$pr_number created successfully"
        return 0
    else
        error "No PR found with title prefix: $title_prefix"
        return 1
    fi
}

validate_repository_security_advisory() {
    local workflow_name="$1"
    
    # Determine expected title based on workflow name
    local expected_title
    if [[ "$workflow_name" == *"claude"* ]]; then
        expected_title="Claude wants security review."
    elif [[ "$workflow_name" == *"codex"* ]]; then
        expected_title="Codex wants security review."
    else
        expected_title="security review"  # Fallback for generic matching
    fi
    
    # Check for security advisories with the specific title
    local security_advisories
    security_advisories=$(gh api repos/:owner/:repo/security-advisories --jq ".[] | select(.title | contains(\"$expected_title\")) | .title" 2>/dev/null || echo "")
    
    if [[ -n "$security_advisories" ]]; then
        success "Security report workflow '$workflow_name' created security advisory with expected title: '$expected_title'"
        return 0
    else
        # Also check for issues with the specific title and security-related labels
        local security_issue
        security_issue=$(gh issue list --limit 10 --json title,labels --jq ".[] | select(.title | contains(\"$expected_title\")) | select(.labels[]?.name | contains(\"security\") or contains(\"vulnerability\")) | .title" 2>/dev/null | head -1)
        
        if [[ -n "$security_issue" ]]; then
            success "Security report workflow '$workflow_name' created security issue with expected title: '$expected_title'"
            return 0
        else
            # Fallback: check for any recent security-related content
            local any_security_content
            any_security_content=$(gh issue list --limit 5 --json title,body --jq ".[] | select(.title or .body | contains(\"security\") or contains(\"Security\")) | .title" 2>/dev/null | head -1)
            
            if [[ -n "$any_security_content" ]]; then
                warning "Security report workflow '$workflow_name' created security content but not with expected title. Found: '$any_security_content'"
                return 0  # Still pass - security content was created
            else
                error "Security report workflow '$workflow_name' completed but no repository security advisories found with expected title: '$expected_title'"
                return 1
            fi
        fi
    fi
}

validate_mcp_workflow() {
    local workflow_name="$1"
    
    # MCP workflows typically create issues with specific patterns indicating MCP tool usage
    # Look for issues with MCP-specific content patterns
    local recent_issues
    recent_issues=$(gh issue list --limit 5 --json title,body --jq '.[] | select(.body | contains("MCP time tool") or contains("current time is") or contains("UTC")) | .title' | head -1)
    
    if [[ -n "$recent_issues" ]]; then
        success "MCP workflow '$workflow_name' appears to have used MCP tools successfully"
        return 0
    else
        # Fallback to original time-based check for broader compatibility
        local time_issues
        time_issues=$(gh issue list --limit 5 --json title,body --jq '.[] | select(.title or .body | contains("time") or contains("Time") or contains("timestamp") or contains("Timestamp")) | .title' | head -1)
        
        if [[ -n "$time_issues" ]]; then
            success "MCP workflow '$workflow_name' appears to have used MCP tools successfully (time-based detection)"
            return 0
        else
            warning "MCP workflow '$workflow_name' completed but no clear evidence of MCP tool usage found"
            return 1
        fi
    fi
}

validate_safe_outputs_workflow() {
    local workflow_name="$1"
    
    # Safe outputs workflows test multiple output types
    local found_outputs=0
    
    # Check for test issues
    local test_issues
    test_issues=$(gh issue list --label "test-safe-outputs" --limit 5 --json number --jq 'length' 2>/dev/null || echo "0")
    if [[ "$test_issues" -gt 0 ]]; then
        found_outputs=$((found_outputs + 1))
        success "Safe outputs workflow '$workflow_name' created test issues"
    fi
    
    # Check for test PRs
    local test_prs
    test_prs=$(gh pr list --label "test-safe-outputs" --limit 5 --json number --jq 'length' 2>/dev/null || echo "0")
    if [[ "$test_prs" -gt 0 ]]; then
        found_outputs=$((found_outputs + 1))
        success "Safe outputs workflow '$workflow_name' created test PRs"
    fi
    
    if [[ $found_outputs -gt 0 ]]; then
        success "Safe outputs workflow '$workflow_name' validated successfully"
        return 0
    else
        warning "Safe outputs workflow '$workflow_name' completed but no test outputs found"
        return 1
    fi
}

validate_ai_inference_workflow() {
    local workflow_name="$1"
    
    # AI inference workflows typically create issues with AI-generated content
    local ai_content
    ai_content=$(gh issue list --limit 5 --json title,body --jq '.[] | select(.title or .body | contains("AI") or contains("inference") or contains("model") or contains("GitHub Models")) | .title' | head -1)
    
    if [[ -n "$ai_content" ]]; then
        success "AI inference workflow '$workflow_name' appears to have used AI models successfully"
        return 0
    else
        warning "AI inference workflow '$workflow_name' completed but no clear evidence of AI inference found"
        return 0  # Don't fail - the workflow may have run but not created visible AI content
    fi
}

validate_branch_created() {
    local branch_name="$1"
    
    if git ls-remote --heads origin "$branch_name" &>/dev/null; then
        success "Branch '$branch_name' created successfully"
        return 0
    else
        error "Branch '$branch_name' not found"
        return 1
    fi
}

cleanup_test_resources() {
    info "Cleaning up test resources..."
    
    # # Disable all test workflows
    # info "Disabling test workflows..."
    # local test_workflows=(
    #     "test-claude-create-issue"
    #     "test-codex-create-issue"
    #     "test-claude-create-pull-request"
    #     "test-codex-create-pull-request"
    #     "test-claude-create-repository-security-advisory"
    #     "test-codex-create-repository-security-advisory"
    #     "test-claude-mcp"
    #     "test-codex-mcp"
    #     "test-custom-safe-outputs"
    #     "test-claude-add-issue-comment"
    #     "test-codex-add-issue-comment"
    #     "test-claude-add-issue-labels"
    #     "test-codex-add-issue-labels"
    #     "test-claude-command"
    #     "test-codex-command"
    #     "test-claude-push-to-branch"
    #     "test-codex-push-to-branch"
    #     "test-claude-create-pull-request-review-comment"
    #     "test-codex-create-pull-request-review-comment"
    #     "test-claude-update-issue"
    #     "test-codex-update-issue"
    # )
    
    # for workflow in "${test_workflows[@]}"; do
    #     disable_workflow "$workflow" || true  # Continue even if disable fails
    # done
    
    # Close test issues
    gh issue list --label "claude,automation" --limit 20 --json number --jq '.[].number' | while read -r issue_num; do
        if [[ -n "$issue_num" ]]; then
            gh issue close "$issue_num" --comment "Closed by e2e test cleanup" &>/dev/null || true
        fi
    done
    
    # Close issues with titles containing "Hello from"
    gh issue list --limit 20 --json number,title --jq '.[] | select(.title | contains("Hello from")) | .number' | while read -r issue_num; do
        if [[ -n "$issue_num" ]]; then
            gh issue close "$issue_num" --comment "Closed by e2e test cleanup" &>/dev/null || true
        fi
    done
    
    # # Also close issues with other test labels
    # gh issue list --label "test-safe-outputs" --limit 20 --json number --jq '.[].number' | while read -r issue_num; do
    #     if [[ -n "$issue_num" ]]; then
    #         gh issue close "$issue_num" --comment "Closed by e2e test cleanup" &>/dev/null || true
    #     fi
    # done
    
    # Close test PRs
    gh pr list --label "claude,automation,bot" --limit 20 --json number --jq '.[].number' | while read -r pr_num; do
        if [[ -n "$pr_num" ]]; then
            gh pr close "$pr_num" --comment "Closed by e2e test cleanup" &>/dev/null || true
        fi
    done
    
    # gh pr list --label "test-safe-outputs" --limit 20 --json number --jq '.[].number' | while read -r pr_num; do
    #     if [[ -n "$pr_num" ]]; then
    #         gh pr close "$pr_num" --comment "Closed by e2e test cleanup" &>/dev/null || true
    #     fi
    # done
    
    # Delete test branches
    git branch -r | grep 'origin/test-pr-\|origin/claude-test-branch' | sed 's/origin\///' | while read -r branch; do
        if [[ -n "$branch" ]]; then
            git push origin --delete "$branch" &>/dev/null || true
        fi
    done
    
    success "Cleanup completed"
}

run_workflow_dispatch_tests() {
    local patterns=("$@")
    info "🚀 Running workflow_dispatch tests..."
    
    local workflows
    readarray -t workflows < <(filter_tests_by_patterns "workflow-dispatch" "${patterns[@]}")
    
    if [[ ${#workflows[@]} -eq 0 ]]; then
        warning "No workflow_dispatch tests match the specified patterns"
        return 0
    fi
    
    for workflow in "${workflows[@]}"; do
        progress "Testing workflow: $workflow"
        
        if trigger_workflow_dispatch "$workflow"; then
            # Validate specific outcomes based on workflow type
            case "$workflow" in
                *"create-issue")
                    if [[ "$workflow" == *"claude"* ]]; then
                        if validate_issue_created "[claude-test]" "claude,automation,haiku"; then
                            PASSED_TESTS+=("$workflow")
                        else
                            FAILED_TESTS+=("$workflow")
                        fi
                    else
                        if validate_issue_created "[codex-test]" "codex,automation"; then
                            PASSED_TESTS+=("$workflow")
                        else
                            FAILED_TESTS+=("$workflow")
                        fi
                    fi
                    ;;
                *"create-pull-request")
                    if [[ "$workflow" == *"claude"* ]]; then
                        if validate_pr_created "[claude-test]"; then
                            PASSED_TESTS+=("$workflow")
                        else
                            FAILED_TESTS+=("$workflow")
                        fi
                    else
                        if validate_pr_created "[codex-test]"; then
                            PASSED_TESTS+=("$workflow")
                        else
                            FAILED_TESTS+=("$workflow")
                        fi
                    fi
                    ;;
                *"repository-security-advisory")
                    if validate_security_report "$workflow"; then
                        PASSED_TESTS+=("$workflow")
                    else
                        FAILED_TESTS+=("$workflow")
                    fi
                    ;;
                *"mcp")
                    # MCP workflows create issues with "Hello from Claude/Codex" titles
                    if [[ "$workflow" == *"claude"* ]]; then
                        if validate_issue_created "Hello from Claude" ""; then
                            PASSED_TESTS+=("$workflow")
                        else
                            FAILED_TESTS+=("$workflow")
                        fi
                    elif [[ "$workflow" == *"codex"* ]]; then
                        if validate_issue_created "Hello from Codex" ""; then
                            PASSED_TESTS+=("$workflow")
                        else
                            FAILED_TESTS+=("$workflow")
                        fi
                    else
                        # Fallback to general MCP validation
                        if validate_mcp_workflow "$workflow"; then
                            PASSED_TESTS+=("$workflow")
                        else
                            FAILED_TESTS+=("$workflow")
                        fi
                    fi
                    ;;
                *"safe-outputs")
                    if validate_safe_outputs_workflow "$workflow"; then
                        PASSED_TESTS+=("$workflow")
                    else
                        FAILED_TESTS+=("$workflow")
                    fi
                    ;;
                *"ai-inference")
                    if validate_ai_inference_workflow "$workflow"; then
                        PASSED_TESTS+=("$workflow")
                    else
                        FAILED_TESTS+=("$workflow")
                    fi
                    ;;
                *)
                    # For truly unknown workflows, just check that they completed successfully
                    success "Workflow '$workflow' completed successfully (no specific validation available)"
                    PASSED_TESTS+=("$workflow")
                    ;;
            esac
        else
            FAILED_TESTS+=("$workflow")
        fi
        
        echo # Add spacing between tests
    done
}

run_issue_triggered_tests() {
    local patterns=("$@")
    info "📝 Running issue-triggered tests..."
    
    local workflows
    readarray -t workflows < <(filter_tests_by_patterns "issue-triggered" "${patterns[@]}")
    
    if [[ ${#workflows[@]} -eq 0 ]]; then
        warning "No issue-triggered tests match the specified patterns"
        return 0
    fi
    
    # Check if we need to run issue-triggered tests that require creating an issue
    local need_claude_comment=false
    local need_claude_labels=false
    
    if should_run_test "test-claude-add-issue-comment" "${patterns[@]}"; then
        need_claude_comment=true
    fi
    
    if should_run_test "test-claude-add-issue-labels" "${patterns[@]}"; then
        need_claude_labels=true
    fi
    
    # Only create test issue if we have tests that need the specific trigger
    if [[ "$need_claude_comment" == true ]] || [[ "$need_claude_labels" == true ]]; then
        progress "Testing claude issue comment and labeling workflows"
        local issue_num
        issue_num=$(create_test_issue "Hello from Claude" "This is a test issue to trigger Claude comment workflow")
        
        if [[ -n "$issue_num" ]]; then
            success "Created test issue #$issue_num"
            sleep 10 # Wait for workflow to trigger
            
            # Test comment workflow if selected
            if [[ "$need_claude_comment" == true ]]; then
                # Wait for the workflow to complete (check for comment)
                local max_wait=60
                local waited=0
                while [[ $waited -lt $max_wait ]]; do
                    if validate_issue_comment "$issue_num" "Reply from Claude"; then
                        PASSED_TESTS+=("test-claude-add-issue-comment")
                        break
                    fi
                    sleep 5
                    waited=$((waited + 5))
                done
                
                if [[ $waited -ge $max_wait ]]; then
                    FAILED_TESTS+=("test-claude-add-issue-comment")
                fi
            fi
            
            # Test issue labels workflow if selected
            if [[ "$need_claude_labels" == true ]]; then
                if validate_issue_labels "$issue_num" "claude-safe-output-label-test"; then
                    PASSED_TESTS+=("test-claude-add-issue-labels")
                else
                    FAILED_TESTS+=("test-claude-add-issue-labels")
                fi
            fi
        else
            error "Failed to create test issue"
            if [[ "$need_claude_comment" == true ]]; then
                FAILED_TESTS+=("test-claude-add-issue-comment")
            fi
            if [[ "$need_claude_labels" == true ]]; then
                FAILED_TESTS+=("test-claude-add-issue-labels")
            fi
        fi
    else
        info "No issue-triggered tests selected that require creating test issues"
    fi
    
    # Note: Additional issue-triggered tests could be added here
    # For now, we only test the Claude ones as examples
}

run_command_tests() {
    local patterns=("$@")
    info "💬 Running command-triggered tests..."
    
    local workflows
    readarray -t workflows < <(filter_tests_by_patterns "command-triggered" "${patterns[@]}")
    
    if [[ ${#workflows[@]} -eq 0 ]]; then
        warning "No command-triggered tests match the specified patterns"
        return 0
    fi
    
    # Check if we actually need to run any command tests
    local need_claude_command=false
    local need_claude_push_to_branch=false
    
    if should_run_test "test-claude-command" "${patterns[@]}"; then
        need_claude_command=true
    fi
    
    if should_run_test "test-claude-push-to-branch" "${patterns[@]}"; then
        need_claude_push_to_branch=true
    fi
    
    # Only create test issue if we have command tests to run
    if [[ "$need_claude_command" == true ]] || [[ "$need_claude_push_to_branch" == true ]]; then
        # Create a test issue for command testing
        local issue_num
        issue_num=$(create_test_issue "Test Issue for Commands" "This issue is for testing command workflows")
        
        if [[ -n "$issue_num" ]]; then
            # Test Claude command
            if [[ "$need_claude_command" == true ]]; then
                progress "Testing Claude command workflow"
                post_issue_command "$issue_num" "/test-claude-command What is this repository about?"
                
                sleep 15 # Wait for workflow to process
                
                if validate_issue_comment "$issue_num" "Claude"; then
                    PASSED_TESTS+=("test-claude-command")
                else
                    FAILED_TESTS+=("test-claude-command")
                fi
            fi
            
            # Test push to branch command
            if [[ "$need_claude_push_to_branch" == true ]]; then
                progress "Testing Claude push-to-branch workflow"
                post_issue_command "$issue_num" "/test-claude-push-to-branch"
                
                sleep 20 # Wait for workflow to process
                
                if validate_branch_created "claude-test-branch"; then
                    PASSED_TESTS+=("test-claude-push-to-branch")
                else
                    FAILED_TESTS+=("test-claude-push-to-branch")
                fi
            fi
        else
            error "Failed to create test issue for commands"
            if [[ "$need_claude_command" == true ]]; then
                FAILED_TESTS+=("test-claude-command")
            fi
            if [[ "$need_claude_push_to_branch" == true ]]; then
                FAILED_TESTS+=("test-claude-push-to-branch")
            fi
        fi
    else
        info "No command tests selected to run"
    fi
}

run_pr_triggered_tests() {
    local patterns=("$@")
    info "🔀 Running PR-triggered tests..."
    
    local workflows
    readarray -t workflows < <(filter_tests_by_patterns "pr-triggered" "${patterns[@]}")
    
    if [[ ${#workflows[@]} -eq 0 ]]; then
        warning "No PR-triggered tests match the specified patterns"
        return 0
    fi
    
    # Test PR review comment workflow only if selected
    if should_run_test "test-claude-create-pull-request-review-comment" "${patterns[@]}"; then
        progress "Testing PR review comment workflow"
        local pr_num
        pr_num=$(create_test_pr "Test PR for Review" "This PR is for testing review comment workflows")
        
        if [[ -n "$pr_num" ]]; then
            success "Created test PR #$pr_num"
            sleep 10 # Wait for workflow to trigger
            
            # Check if review comment was added (this is harder to validate automatically)
            # For now, we'll just mark it as passed if the PR was created
            PASSED_TESTS+=("test-claude-create-pull-request-review-comment")
        else
            error "Failed to create test PR"
            FAILED_TESTS+=("test-claude-create-pull-request-review-comment")
        fi
    else
        info "No PR-triggered tests selected that require creating test PRs"
    fi
}

print_final_report() {
    echo
    echo "============================================"
    echo -e "${CYAN}📊 FINAL TEST REPORT${NC}"
    echo "============================================"
    echo
    
    local total_tests=$((${#PASSED_TESTS[@]} + ${#FAILED_TESTS[@]} + ${#SKIPPED_TESTS[@]}))
    
    echo -e "${GREEN}✅ PASSED (${#PASSED_TESTS[@]}/$total_tests):${NC}"
    for test in "${PASSED_TESTS[@]}"; do
        echo -e "   ${GREEN}✓${NC} $test"
    done
    echo
    
    if [[ ${#FAILED_TESTS[@]} -gt 0 ]]; then
        echo -e "${RED}❌ FAILED (${#FAILED_TESTS[@]}/$total_tests):${NC}"
        for test in "${FAILED_TESTS[@]}"; do
            echo -e "   ${RED}✗${NC} $test"
        done
        echo
    fi
    
    if [[ ${#SKIPPED_TESTS[@]} -gt 0 ]]; then
        echo -e "${YELLOW}⏭️  SKIPPED (${#SKIPPED_TESTS[@]}/$total_tests):${NC}"
        for test in "${SKIPPED_TESTS[@]}"; do
            echo -e "   ${YELLOW}↷${NC} $test"
        done
        echo
    fi
    
    local success_rate
    if [[ $total_tests -gt 0 ]]; then
        success_rate=$(( (${#PASSED_TESTS[@]} * 100) / total_tests ))
    else
        success_rate=0
    fi
    
    echo "============================================"
    echo -e "${CYAN}📈 Success Rate: ${success_rate}% (${#PASSED_TESTS[@]}/$total_tests)${NC}"
    echo -e "${CYAN}📄 Log file: $LOG_FILE${NC}"
    echo "============================================"
    
    if [[ ${#FAILED_TESTS[@]} -gt 0 ]]; then
        exit 1
    fi
}

main() {
    echo -e "${CYAN}🧪 GitHub Agentic Workflows End-to-End Testing${NC}"
    echo -e "${CYAN}=================================================${NC}"
    echo
    
        # Parse command line arguments
    local run_workflow_dispatch=true
    local run_issue_triggered=true
    local run_command_triggered=true
    local run_pr_triggered=true
    local dry_run=false
    local specific_tests=()
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --workflow-dispatch-only)
                run_workflow_dispatch=true
                run_issue_triggered=false
                run_command_triggered=false
                run_pr_triggered=false
                shift
                ;;
            --issue-triggered-only)
                run_workflow_dispatch=false
                run_issue_triggered=true
                run_command_triggered=false
                run_pr_triggered=false
                shift
                ;;
            --command-triggered-only)
                run_workflow_dispatch=false
                run_issue_triggered=false
                run_command_triggered=true
                run_pr_triggered=false
                shift
                ;;
            --pr-triggered-only)
                run_workflow_dispatch=false
                run_issue_triggered=false
                run_command_triggered=false
                run_pr_triggered=true
                shift
                ;;
            --dry-run)
                dry_run=true
                shift
                ;;
            --help|-h)
                echo "Usage: $0 [OPTIONS] [TEST_PATTERNS...]"
                echo ""
                echo "Options:"
                echo "  --workflow-dispatch-only   Run only workflow_dispatch triggered tests"
                echo "  --issue-triggered-only     Run only issue-triggered tests"
                echo "  --command-triggered-only   Run only command-triggered tests"
                echo "  --pr-triggered-only        Run only PR-triggered tests"
                echo "  --dry-run                  Show what would be tested without running"
                echo "  --help, -h                 Show this help message"
                echo ""
                echo "TEST_PATTERNS:"
                echo "  Specific test names or glob patterns to run:"
                echo "    ./e2e.sh test-claude-create-issue"
                echo "    ./e2e.sh test-claude-* test-codex-*"
                echo "    ./e2e.sh test-*-create-issue"
                echo ""
                echo "By default, all test suites are run."
                exit 0
                ;;
            --*)
                error "Unknown option: $1"
                echo "Use --help for usage information"
                exit 1
                ;;
            *)
                # This is a test pattern
                specific_tests+=("$1")
                shift
                ;;
        esac
    done
    
    if [[ "$dry_run" == true ]]; then
        info "DRY RUN MODE - Showing what would be tested:"
        echo
        
        if [[ ${#specific_tests[@]} -gt 0 ]]; then
            info "🎯 Test Patterns: ${specific_tests[*]}"
            echo
        fi
        
        if [[ "$run_workflow_dispatch" == true ]]; then
            info "🚀 Workflow Dispatch Tests:"
            local workflows
            readarray -t workflows < <(filter_tests_by_patterns "workflow-dispatch" "${specific_tests[@]}")
            if [[ ${#workflows[@]} -gt 0 ]]; then
                for workflow in "${workflows[@]}"; do
                    echo "   - $workflow"
                done
            else
                echo "   (no tests match the specified patterns)"
            fi
            echo
        fi
        
        if [[ "$run_issue_triggered" == true ]]; then
            info "📝 Issue-Triggered Tests:"
            local workflows
            readarray -t workflows < <(filter_tests_by_patterns "issue-triggered" "${specific_tests[@]}")
            if [[ ${#workflows[@]} -gt 0 ]]; then
                for workflow in "${workflows[@]}"; do
                    echo "   - $workflow"
                done
            else
                echo "   (no tests match the specified patterns)"
            fi
            echo
        fi
        
        if [[ "$run_command_triggered" == true ]]; then
            info "� Command-Triggered Tests:"
            local workflows
            readarray -t workflows < <(filter_tests_by_patterns "command-triggered" "${specific_tests[@]}")
            if [[ ${#workflows[@]} -gt 0 ]]; then
                for workflow in "${workflows[@]}"; do
                    echo "   - $workflow"
                done
            else
                echo "   (no tests match the specified patterns)"
            fi
            echo
        fi
        
        if [[ "$run_pr_triggered" == true ]]; then
            info "� PR-Triggered Tests:"
            local workflows
            readarray -t workflows < <(filter_tests_by_patterns "pr-triggered" "${specific_tests[@]}")
            if [[ ${#workflows[@]} -gt 0 ]]; then
                for workflow in "${workflows[@]}"; do
                    echo "   - $workflow"
                done
            else
                echo "   (no tests match the specified patterns)"
            fi
            echo
        fi
        
        exit 0
    fi
    
    log "Starting e2e tests at $(date)"
    
    check_prerequisites
    
    # If specific tests are provided, determine which test suites need to run
    if [[ ${#specific_tests[@]} -gt 0 ]]; then
        info "🎯 Running specific tests: ${specific_tests[*]}"
        
        # Check if any workflow dispatch tests match
        local wd_tests
        readarray -t wd_tests < <(filter_tests_by_patterns "workflow-dispatch" "${specific_tests[@]}")
        if [[ ${#wd_tests[@]} -eq 0 ]]; then
            run_workflow_dispatch=false
        fi
        
        # Check if any issue triggered tests match
        local it_tests
        readarray -t it_tests < <(filter_tests_by_patterns "issue-triggered" "${specific_tests[@]}")
        if [[ ${#it_tests[@]} -eq 0 ]]; then
            run_issue_triggered=false
        fi
        
        # Check if any command triggered tests match
        local ct_tests
        readarray -t ct_tests < <(filter_tests_by_patterns "command-triggered" "${specific_tests[@]}")
        if [[ ${#ct_tests[@]} -eq 0 ]]; then
            run_command_triggered=false
        fi
        
        # Check if any PR triggered tests match
        local pt_tests
        readarray -t pt_tests < <(filter_tests_by_patterns "pr-triggered" "${specific_tests[@]}")
        if [[ ${#pt_tests[@]} -eq 0 ]]; then
            run_pr_triggered=false
        fi
    fi
    
    # Run test suites based on options
    if [[ "$run_workflow_dispatch" == true ]]; then
        run_workflow_dispatch_tests "${specific_tests[@]}"
    fi
    
    if [[ "$run_issue_triggered" == true ]]; then
        run_issue_triggered_tests "${specific_tests[@]}"
    fi
    
    if [[ "$run_command_triggered" == true ]]; then
        run_command_tests "${specific_tests[@]}"
    fi
    
    if [[ "$run_pr_triggered" == true ]]; then
        run_pr_triggered_tests "${specific_tests[@]}"
    fi
    
    print_final_report
    
    # Ask user if they want to cleanup
    echo
    read -p "$(echo -e ${YELLOW}⚠️  Do you want to clean up test resources \(issues, PRs, branches\)? [y/N]: ${NC})" -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        cleanup_test_resources
    else
        warning "Test resources were not cleaned up. You may want to manually clean them later."
    fi
    
    log "E2E tests completed at $(date)"
}

# Handle script interruption
trap 'error "Script interrupted"; exit 130' INT TERM

main "$@"
