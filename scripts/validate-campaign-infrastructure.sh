#!/bin/bash
# Campaign Infrastructure Validation Script
# Validates that all required infrastructure is in place for campaigns to run

set -o pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Counters
PASS=0
FAIL=0
WARN=0

# Helper functions
print_header() {
    echo -e "\n${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}\n"
}

print_section() {
    echo -e "\n${BLUE}--- $1 ---${NC}"
}

check_pass() {
    echo -e "  ${GREEN}✓${NC} $1"
    PASS=$((PASS + 1))
}

check_fail() {
    echo -e "  ${RED}✗${NC} $1"
    FAIL=$((FAIL + 1))
}

check_warn() {
    echo -e "  ${YELLOW}⚠${NC} $1"
    WARN=$((WARN + 1))
}

# Validation functions

validate_campaign_orchestrators() {
    print_section "Campaign Orchestrators"
    
    # Check Project 67 orchestrator
    if [ -f ".github/workflows/docs-quality-maintenance-project67.campaign.g.md" ]; then
        check_pass "docs-quality-maintenance-project67.campaign.g.md exists"
    else
        check_fail "docs-quality-maintenance-project67.campaign.g.md MISSING"
    fi
    
    # Check Project 64 orchestrator
    if [ -f ".github/workflows/go-file-size-reduction-project64.campaign.g.md" ]; then
        check_pass "go-file-size-reduction-project64.campaign.g.md exists"
    else
        check_fail "go-file-size-reduction-project64.campaign.g.md MISSING"
    fi
    
    # Check compiled lock files
    if [ -f ".github/workflows/docs-quality-maintenance-project67.campaign.g.lock.yml" ]; then
        check_pass "docs-quality-maintenance-project67.campaign.g.lock.yml compiled"
    else
        check_fail "docs-quality-maintenance-project67.campaign.g.lock.yml MISSING"
    fi
    
    if [ -f ".github/workflows/go-file-size-reduction-project64.campaign.g.lock.yml" ]; then
        check_pass "go-file-size-reduction-project64.campaign.g.lock.yml compiled"
    else
        check_fail "go-file-size-reduction-project64.campaign.g.lock.yml MISSING"
    fi
    
    # Check scheduling
    # Note: Asterisks are escaped (\*) to match literal asterisks in the cron pattern
    if grep -q 'cron: "0 18 \* \* \*"' ".github/workflows/docs-quality-maintenance-project67.campaign.g.lock.yml" 2>/dev/null; then
        check_pass "Project 67 orchestrator scheduled for 18:00 UTC daily"
    else
        check_fail "Project 67 orchestrator schedule NOT FOUND or incorrect"
    fi
    
    if grep -q 'cron: "0 18 \* \* \*"' ".github/workflows/go-file-size-reduction-project64.campaign.g.lock.yml" 2>/dev/null; then
        check_pass "Project 64 orchestrator scheduled for 18:00 UTC daily"
    else
        check_fail "Project 64 orchestrator schedule NOT FOUND or incorrect"
    fi
}

validate_worker_workflows() {
    print_section "Worker Workflows"
    
    # Project 67 workflows
    WORKFLOWS_67=("daily-doc-updater" "docs-noob-tester" "daily-multi-device-docs-tester" "unbloat-docs" "developer-docs-consolidator" "technical-doc-writer")
    
    echo "Project 67 workflows (6 required):"
    for workflow in "${WORKFLOWS_67[@]}"; do
        if [ -f ".github/workflows/${workflow}.md" ] && [ -f ".github/workflows/${workflow}.lock.yml" ]; then
            check_pass "${workflow} exists and compiled"
        elif [ -f ".github/workflows/${workflow}.md" ]; then
            check_warn "${workflow}.md exists but NOT compiled"
        else
            check_fail "${workflow} MISSING"
        fi
    done
    
    # Project 64 workflows
    echo ""
    echo "Project 64 workflows (1 required):"
    if [ -f ".github/workflows/daily-file-diet.md" ] && [ -f ".github/workflows/daily-file-diet.lock.yml" ]; then
        check_pass "daily-file-diet exists and compiled"
    elif [ -f ".github/workflows/daily-file-diet.md" ]; then
        check_warn "daily-file-diet.md exists but NOT compiled"
    else
        check_fail "daily-file-diet MISSING"
    fi
}

validate_memory_configuration() {
    print_section "Memory Path Configuration"
    
    # Check repo-memory configuration in orchestrators
    if grep -q "branch-name: memory/campaigns" ".github/workflows/docs-quality-maintenance-project67.campaign.g.md" 2>/dev/null; then
        check_pass "Project 67 configured for memory/campaigns branch"
    else
        check_fail "Project 67 memory configuration NOT FOUND"
    fi
    
    if grep -q "campaign-id: docs-quality-maintenance-project67" ".github/workflows/docs-quality-maintenance-project67.campaign.g.md" 2>/dev/null; then
        check_pass "Project 67 campaign-id configured"
    else
        check_fail "Project 67 campaign-id NOT FOUND"
    fi
    
    if grep -q "branch-name: memory/campaigns" ".github/workflows/go-file-size-reduction-project64.campaign.g.md" 2>/dev/null; then
        check_pass "Project 64 configured for memory/campaigns branch"
    else
        check_fail "Project 64 memory configuration NOT FOUND"
    fi
    
    if grep -q "campaign-id: go-file-size-reduction-project64" ".github/workflows/go-file-size-reduction-project64.campaign.g.md" 2>/dev/null; then
        check_pass "Project 64 campaign-id configured"
    else
        check_fail "Project 64 campaign-id NOT FOUND"
    fi
    
    # Check for memory branches (may not exist yet)
    echo ""
    echo "Memory branches (auto-created on first run):"
    if git ls-remote --heads origin memory/campaigns &>/dev/null; then
        check_pass "memory/campaigns branch exists"
    else
        check_warn "memory/campaigns branch does NOT exist yet (will be auto-created)"
    fi
    
    if git ls-remote --heads origin memory/meta-orchestrators &>/dev/null; then
        check_pass "memory/meta-orchestrators branch exists"
    else
        check_warn "memory/meta-orchestrators branch does NOT exist yet (will be auto-created)"
    fi
}

validate_metrics_infrastructure() {
    print_section "Metrics Infrastructure"
    
    # Check metrics collector workflow
    if [ -f ".github/workflows/metrics-collector.md" ]; then
        check_pass "metrics-collector.md exists"
    else
        check_fail "metrics-collector.md MISSING"
    fi
    
    if [ -f ".github/workflows/metrics-collector.lock.yml" ]; then
        check_pass "metrics-collector.lock.yml compiled"
    else
        check_fail "metrics-collector.lock.yml MISSING"
    fi
    
    # Check scheduling
    if grep -q "schedule:" ".github/workflows/metrics-collector.lock.yml" 2>/dev/null; then
        check_pass "Metrics collector has schedule configured"
    else
        check_fail "Metrics collector schedule NOT FOUND"
    fi
    
    # Check repo-memory configuration
    if grep -q "branch-name: memory/meta-orchestrators" ".github/workflows/metrics-collector.md" 2>/dev/null; then
        check_pass "Metrics collector configured for memory/meta-orchestrators"
    else
        check_fail "Metrics collector memory configuration INCORRECT"
    fi
    
    # Check for expected metrics paths
    if grep -q 'file-glob: "memory/meta-orchestrators/metrics/\*\*"' ".github/workflows/metrics-collector.md" 2>/dev/null; then
        check_pass "Metrics collector file-glob configured"
    else
        check_warn "Metrics collector file-glob may need verification"
    fi
}

validate_project_access() {
    print_section "Project Board Access"
    
    # Check if gh is available
    if command -v gh &> /dev/null; then
        check_pass "GitHub CLI (gh) is available"
        
        # Check if we can access projects (requires auth)
        if [ -n "$GH_TOKEN" ]; then
            check_pass "GH_TOKEN environment variable is set"
            
            # Try to access Project 67
            if gh api graphql -f query='query { organization(login: "githubnext") { projectV2(number: 67) { title } } }' &>/dev/null; then
                check_pass "Project 67 is accessible"
            else
                check_fail "Project 67 is NOT accessible (check token permissions)"
            fi
            
            # Try to access Project 64
            if gh api graphql -f query='query { organization(login: "githubnext") { projectV2(number: 64) { title } } }' &>/dev/null; then
                check_pass "Project 64 is accessible"
            else
                check_fail "Project 64 is NOT accessible (check token permissions)"
            fi
        else
            check_warn "GH_TOKEN not set - cannot test project access"
            check_warn "In workflows, ensure GH_AW_PROJECT_GITHUB_TOKEN secret is configured"
        fi
    else
        check_warn "GitHub CLI (gh) not available - cannot test project access"
    fi
}

validate_safe_outputs() {
    print_section "Safe Output Configuration"
    
    # Check Project 67 safe outputs
    if grep -q "update-project:" ".github/workflows/docs-quality-maintenance-project67.campaign.g.md" 2>/dev/null; then
        check_pass "Project 67 has update-project safe-output configured"
        
        if grep -q "github-token:.*GH_AW_PROJECT_GITHUB_TOKEN" ".github/workflows/docs-quality-maintenance-project67.campaign.g.md" 2>/dev/null; then
            check_pass "Project 67 references GH_AW_PROJECT_GITHUB_TOKEN secret"
        else
            check_fail "Project 67 safe-output token configuration INCORRECT"
        fi
    else
        check_fail "Project 67 update-project safe-output NOT FOUND"
    fi
    
    # Check Project 64 safe outputs
    if grep -q "update-project:" ".github/workflows/go-file-size-reduction-project64.campaign.g.md" 2>/dev/null; then
        check_pass "Project 64 has update-project safe-output configured"
        
        if grep -q "github-token:.*GH_AW_PROJECT_GITHUB_TOKEN" ".github/workflows/go-file-size-reduction-project64.campaign.g.md" 2>/dev/null; then
            check_pass "Project 64 references GH_AW_PROJECT_GITHUB_TOKEN secret"
        else
            check_fail "Project 64 safe-output token configuration INCORRECT"
        fi
    else
        check_fail "Project 64 update-project safe-output NOT FOUND"
    fi
}

# Main validation
main() {
    print_header "Campaign Infrastructure Validation"
    echo "Validating infrastructure for:"
    echo "  - Campaign: docs-quality-maintenance-project67 (Project 67)"
    echo "  - Campaign: go-file-size-reduction-project64 (Project 64)"
    
    validate_campaign_orchestrators
    validate_worker_workflows
    validate_memory_configuration
    validate_metrics_infrastructure
    validate_project_access
    validate_safe_outputs
    
    # Summary
    print_header "Validation Summary"
    echo -e "${GREEN}Passed:${NC}  $PASS"
    echo -e "${YELLOW}Warnings:${NC} $WARN"
    echo -e "${RED}Failed:${NC}  $FAIL"
    echo ""
    
    if [ $FAIL -eq 0 ]; then
        echo -e "${GREEN}✓ All critical validations passed!${NC}"
        if [ $WARN -gt 0 ]; then
            echo -e "${YELLOW}⚠ Some warnings present - review above${NC}"
        fi
        exit 0
    else
        echo -e "${RED}✗ Some validations failed - review above${NC}"
        exit 1
    fi
}

# Run main
main
