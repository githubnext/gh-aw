#!/bin/bash
set +o histexpand

# Safe Outputs Specification Conformance Checker
# This script implements automated checks for the Safe Outputs specification
# Specification: docs/src/content/docs/reference/safe-outputs-specification.md
# Version: 1.16.0 (2026-04-06)

set -euo pipefail

# Color codes for output
RED='\033[0;31m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Counters
CRITICAL_FAILURES=0
HIGH_FAILURES=0
MEDIUM_FAILURES=0
LOW_FAILURES=0

# Logging functions
log_critical() {
    echo -e "${RED}[CRITICAL]${NC} $1"
    ((CRITICAL_FAILURES += 1))
}

log_high() {
    echo -e "${RED}[HIGH]${NC} $1"
    ((HIGH_FAILURES += 1))
}

log_medium() {
    echo -e "${YELLOW}[MEDIUM]${NC} $1"
    ((MEDIUM_FAILURES += 1))
}

log_low() {
    echo -e "${BLUE}[LOW]${NC} $1"
    ((LOW_FAILURES += 1))
}

log_pass() {
    echo -e "${GREEN}[PASS]${NC} $1"
}

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

# Change to repo root
cd "$(dirname "$0")/.."

echo "=================================================="
echo "Safe Outputs Specification Conformance Checker"
echo "=================================================="
echo ""

# SEC-001: Privilege Separation Enforcement
echo "Running SEC-001: Privilege Separation Enforcement..."
check_privilege_separation() {
    local failed=0
    
    # Find all compiled workflow files
    find .github/workflows -name "*.lock.yml" | while read -r workflow; do
        # Check if agent job has write permissions
        if grep -A 50 "^jobs:" "$workflow" | grep -A 20 "^\s*agent:" | grep -qE "issues:\s*write|pull-requests:\s*write|contents:\s*write"; then
            log_critical "SEC-001: Agent job in $workflow has write permissions"
            failed=1
        fi
    done
    
    if [ $failed -eq 0 ]; then
        log_pass "SEC-001: All agent jobs properly lack write permissions"
    fi
}
check_privilege_separation

# SEC-002: Validation Before API Calls
echo "Running SEC-002: Validation Before API Calls..."
check_validation_ordering() {
    local failed=0
    
    for handler in actions/setup/js/*.cjs; do
        # Skip test files
        [[ "$handler" =~ test ]] && continue
        [[ "$handler" =~ parse ]] && continue
        [[ "$handler" =~ buffer ]] && continue
        
        # Check if handler has API calls
        if grep -q "octokit\." "$handler"; then
            # Check if validation appears before API calls
            if ! awk '/octokit\./{api_line=NR} /validate|sanitize|enforceLimit/{if(NR<api_line || api_line==0) valid=1} END{exit !valid}' "$handler" 2>/dev/null; then
                log_critical "SEC-002: $handler may have API calls before validation"
                failed=1
            fi
        fi
    done
    
    if [ $failed -eq 0 ]; then
        log_pass "SEC-002: All handlers validate before API calls"
    fi
}
check_validation_ordering

# SEC-003: Max Limit Enforcement
echo "Running SEC-003: Max Limit Enforcement..."
check_max_limits() {
    local failed=0
    
    for handler in actions/setup/js/*.cjs; do
        # Skip test and utility files
        [[ "$handler" =~ (test|parse|buffer|factory) ]] && continue
        
        # Only check files that perform GitHub API operations
        if ! grep -q "octokit\." "$handler"; then
            continue
        fi
        
        # Check if handler enforces max limits using any recognized pattern
        if ! grep -qE "\.length.*>.*\.max|enforceMaxLimit|checkLimit|max.*exceeded|enforceArrayLimit|tryEnforceArrayLimit|limit_enforcement_helpers" "$handler"; then
            log_medium "SEC-003: $handler may not enforce max limits"
            failed=1
        fi
    done
    
    if [ $failed -eq 0 ]; then
        log_pass "SEC-003: All handlers enforce max limits"
    fi
}
check_max_limits

# SEC-004: Content Sanitization Required
echo "Running SEC-004: Content Sanitization Required..."
check_sanitization() {
    local failed=0
    
    for handler in actions/setup/js/*.cjs; do
        # Skip test and utility files
        [[ "$handler" =~ (test|parse|buffer) ]] && continue

        # Skip files with a documented SEC-004 exemption annotation
        if grep -q "@safe-outputs-exempt[[:space:]]\\+SEC-004" "$handler"; then
            continue
        fi
        
        # Check if handler has body/content fields
        if grep -q "\"body\"\|body:" "$handler"; then
            # Check for sanitization
            if ! grep -q "sanitize\|stripHTML\|escapeMarkdown\|cleanContent" "$handler"; then
                log_medium "SEC-004: $handler has body field but no sanitization"
                failed=1
            fi
        fi
    done
    
    if [ $failed -eq 0 ]; then
        log_pass "SEC-004: All handlers properly sanitize content"
    fi
}
check_sanitization

# SEC-005: Cross-Repository Validation
echo "Running SEC-005: Cross-Repository Validation..."
check_cross_repo() {
    local failed=0
    
    for handler in actions/setup/js/*.cjs; do
        # Skip test files
        [[ "$handler" =~ test ]] && continue
        
        # Skip files with a documented SEC-005 exemption annotation
        if grep -q "@safe-outputs-exempt.*SEC-005" "$handler"; then
            continue
        fi
        
        # Check if handler supports target-repo
        if grep -q "target.*[Rr]epo\|targetRepo" "$handler"; then
            # Check for allowlist validation
            if ! grep -q "allowed.*[Rr]epos\|validateTargetRepo\|checkAllowedRepo" "$handler"; then
                log_high "SEC-005: $handler supports target-repo but lacks allowlist check"
                failed=1
            fi
        fi
    done
    
    if [ $failed -eq 0 ]; then
        log_pass "SEC-005: All cross-repo handlers validate allowlists"
    fi
}
check_cross_repo

# USE-001: Error Code Standardization
echo "Running USE-001: Error Code Standardization..."
check_error_codes() {
    local failed=0
    
    for handler in actions/setup/js/*.cjs; do
        # Skip test files and non-safe-output modules
        [[ "$handler" =~ test ]] && continue
        [[ "$handler" =~ (apm_unpack|run_apm_unpack|observability|generate_observability) ]] && continue
        
        # Only check handlers that interact with GitHub via octokit or record safe output operations
        if ! grep -qE "octokit\.|safe_output|safeOutput|NDJSON" "$handler"; then
            continue
        fi
        
        # Check if handler throws errors
        if grep -q "throw.*Error\|core\.setFailed" "$handler"; then
            # Check for standardized error codes
            if ! grep -qE "E[0-9]{3}|ERROR_|ERR_" "$handler"; then
                log_low "USE-001: $handler may not use standardized error codes"
                failed=1
            fi
        fi
    done
    
    if [ $failed -eq 0 ]; then
        log_pass "USE-001: All handlers use standardized error codes"
    fi
}
check_error_codes

# USE-002: Footer Attribution Required
echo "Running USE-002: Footer Attribution Required..."
check_footers() {
    local failed=0
    
    # Check handlers that create issues/PRs/discussions
    for handler in actions/setup/js/{create_issue,create_pull_request,create_discussion,add_comment}.cjs; do
        [ ! -f "$handler" ] && continue
        
        # Check if handler adds footers
        if ! grep -q "footer\|addFooter\|attribution\|AI generated" "$handler"; then
            log_low "USE-002: $handler may not add footer attribution"
            failed=1
        fi
    done
    
    if [ $failed -eq 0 ]; then
        log_pass "USE-002: All handlers add footer attribution when configured"
    fi
}
check_footers

# USE-003: Staged Mode Preview Format
echo "Running USE-003: Staged Mode Preview Format..."
check_staged_mode() {
    local failed=0
    
    for handler in actions/setup/js/*.cjs; do
        # Skip test files and non-safe-output modules
        [[ "$handler" =~ test ]] && continue
        [[ "$handler" =~ (apm_unpack|run_apm_unpack|observability|generate_observability) ]] && continue
        
        # Only check handlers that explicitly reference the safe outputs staged mode env var
        if grep -q "GH_AW_SAFE_OUTPUTS_STAGED\|logStagedPreviewInfo\|generateStagedPreview" "$handler"; then
            # Check for emoji in preview
            if ! grep -q "🎭\|Staged Mode.*Preview\|logStagedPreviewInfo\|generateStagedPreview" "$handler"; then
                log_low "USE-003: $handler has staged mode but missing 🎭 emoji"
                failed=1
            fi
        fi
    done
    
    if [ $failed -eq 0 ]; then
        log_pass "USE-003: All handlers use correct staged mode format"
    fi
}
check_staged_mode

# REQ-001: RFC 2119 Keyword Usage
echo "Running REQ-001: RFC 2119 Keyword Usage..."
check_rfc2119() {
    local spec_file="docs/src/content/docs/reference/safe-outputs-specification.md"
    local failed=0
    
    # Check key sections have RFC 2119 keywords
    for section in "Security Architecture" "Configuration Semantics" "Execution Guarantees"; do
        if ! grep -A 200 "## .*$section" "$spec_file" 2>/dev/null | grep -q "MUST\|SHALL\|SHOULD\|MAY"; then
            log_medium "REQ-001: Section '$section' may lack RFC 2119 keywords"
            failed=1
        fi
    done
    
    if [ $failed -eq 0 ]; then
        log_pass "REQ-001: Normative sections use RFC 2119 keywords"
    fi
}
check_rfc2119

# REQ-002: Safe Output Type Completeness
echo "Running REQ-002: Safe Output Type Completeness..."
check_type_completeness() {
    local spec_file="docs/src/content/docs/reference/safe-outputs-specification.md"
    local failed=0
    
    # Extract type names
    grep "^#### Type:" "$spec_file" 2>/dev/null | sed 's/^#### Type: //' | head -10 | while read -r type_name; do
        sections_found=0
        
        # Check for required sections
        for section in "MCP Tool Schema" "Operational Semantics" "Configuration Parameters" "Security Requirements" "Required Permissions"; do
            if grep -A 200 "^#### Type: $type_name" "$spec_file" 2>/dev/null | grep -q "**$section**"; then
                ((sections_found += 1))
            fi
        done
        
        if [ $sections_found -lt 5 ]; then
            log_medium "REQ-002: Type '$type_name' has only $sections_found/5 required sections"
            failed=1
        fi
    done
    
    if [ $failed -eq 0 ]; then
        log_pass "REQ-002: All safe output types have complete documentation"
    fi
}
check_type_completeness

# REQ-003: Verification Method Specification
echo "Running REQ-003: Verification Method Specification..."
check_verification_methods() {
    local spec_file="docs/src/content/docs/reference/safe-outputs-specification.md"
    local failed=0
    
    # Check key requirements have verification methods
    # Accept both bold (**Verification:**) and italic (*Verification*:) formats
    for req in "AR1" "AR2" "AR3" "SP1" "SP2" "SP3"; do
        if ! grep -A 30 "\*\*Requirement $req:\|\*\*Property $req:" "$spec_file" 2>/dev/null | grep -qE "\*\*Verification\*\*:|\*Verification\*:|\*\*Formal Definition\*\*:|\*Formal Definition\*:"; then
            log_low "REQ-003: Requirement $req may lack verification method"
            failed=1
        fi
    done
    
    if [ $failed -eq 0 ]; then
        log_pass "REQ-003: All requirements have verification methods"
    fi
}
check_verification_methods

# IMP-001: Handler Registration Completeness
echo "Running IMP-001: Handler Registration Completeness..."
check_handler_registration() {
    local failed=0
    
    # Check if standard handlers exist
    for type in create_issue add_comment close_issue update_issue add_labels remove_labels; do
        handler_file="actions/setup/js/${type}.cjs"
        if [ ! -f "$handler_file" ]; then
            log_high "IMP-001: Missing handler file $handler_file"
            failed=1
        fi
    done
    
    if [ $failed -eq 0 ]; then
        log_pass "IMP-001: All standard handlers are registered"
    fi
}
check_handler_registration

# IMP-002: Permission Computation Accuracy
echo "Running IMP-002: Permission Computation Accuracy..."
check_permission_computation() {
    # Check if permission computation file exists and is well-formed
    if [ -f "pkg/workflow/safe_outputs_permissions.go" ]; then
        # Basic check that it defines ComputePermissionsForSafeOutputs
        if grep -q "ComputePermissionsForSafeOutputs" "pkg/workflow/safe_outputs_permissions.go"; then
            log_pass "IMP-002: Permission computation function exists"
        else
            log_high "IMP-002: Permission computation function not found"
        fi
    else
        log_high "IMP-002: Permission computation file missing"
    fi
}
check_permission_computation

# IMP-003: Schema Validation Consistency
echo "Running IMP-003: Schema Validation Consistency..."
check_schema_consistency() {
    local failed=0
    
    # Check if safe outputs config generation file exists with schema functions
    if [ -f "pkg/workflow/safe_outputs_config_generation.go" ]; then
        # Check for schema generation functions (custom job tool definition generation)
        if ! grep -q "generateCustomJobToolDefinition" "pkg/workflow/safe_outputs_config_generation.go"; then
            log_medium "IMP-003: Dynamic schema generation function missing"
            failed=1
        fi
    else
        log_medium "IMP-003: Safe outputs config generation file missing"
        failed=1
    fi
    
    # Check if static schemas file exists (embedded JSON)
    if [ -f "pkg/workflow/js/safe_outputs_tools.json" ]; then
        # Verify it contains MCP tool definitions with inputSchema
        if ! grep -q '"inputSchema"' "pkg/workflow/js/safe_outputs_tools.json"; then
            log_medium "IMP-003: Static schema definitions missing inputSchema"
            failed=1
        fi
    else
        log_medium "IMP-003: Static safe outputs tools schema missing"
        failed=1
    fi
    
    # Check if safe_outputs_config.go has documentation about schema architecture
    if [ -f "pkg/workflow/safe_outputs_config.go" ]; then
        if ! grep -q "Schema Generation Architecture" "pkg/workflow/safe_outputs_config.go"; then
            log_medium "IMP-003: Schema architecture documentation missing"
            failed=1
        fi
    fi
    
    if [ $failed -eq 0 ]; then
        log_pass "IMP-003: Schema generation is implemented"
    fi
}
check_schema_consistency

# MCE-001: Tool Description Constraint Disclosure (Section 8.3 MCE2)
echo "Running MCE-001: Tool Description Constraint Disclosure..."
check_mce_constraint_disclosure() {
    local tools_json="pkg/workflow/js/safe_outputs_tools.json"
    local failed=0
    
    if [ ! -f "$tools_json" ]; then
        log_high "MCE-001: Tool definitions file missing: $tools_json"
        return
    fi
    
    # Per spec Section 8.3 MCE2: add_comment MUST surface its constraint limits in description
    # Required: 65536 char limit, 10 mentions, 50 links (checks the combined tool JSON file)
    if ! grep -iE "65536" "$tools_json" > /dev/null 2>&1; then
        log_medium "MCE-001: add_comment tool description may be missing 65536 character limit"
        failed=1
    fi
    if ! grep -iE "10 mention" "$tools_json" > /dev/null 2>&1; then
        log_medium "MCE-001: add_comment tool description may be missing 10 mention limit"
        failed=1
    fi
    if ! grep -iE "50 link" "$tools_json" > /dev/null 2>&1; then
        log_medium "MCE-001: add_comment tool description may be missing 50 link limit"
        failed=1
    fi
    
    # Verify add_comment description contains CONSTRAINTS or IMPORTANT keyword
    if ! grep -A 5 '"add_comment"' "$tools_json" | grep -qE "CONSTRAINTS|IMPORTANT.*constraint|validation constraint"; then
        log_medium "MCE-001: add_comment tool description missing required CONSTRAINTS/IMPORTANT disclosure"
        failed=1
    fi
    
    if [ $failed -eq 0 ]; then
        log_pass "MCE-001: Tool descriptions properly disclose enforcement constraints"
    fi
}
check_mce_constraint_disclosure

# MCE-002: Dual Enforcement Pattern (Section 8.3 MCE4)
echo "Running MCE-002: Dual Enforcement Pattern..."
check_mce_dual_enforcement() {
    local failed=0
    local helpers_file="actions/setup/js/comment_limit_helpers.cjs"
    
    # Per spec Section 8.3 MCE4: constraints must be enforced at both MCP invocation
    # time and safe output processing time
    
    # Check constraint helper module exists
    if [ ! -f "$helpers_file" ]; then
        log_high "MCE-002: Constraint helper module missing: $helpers_file"
        failed=1
        return
    fi
    
    # Check that both the MCP gateway handler (records operations) and the safe output 
    # processor (add_comment.cjs - executes API calls) import/use the constraint helpers.
    # Per spec MCE4: dual enforcement must exist at both invocation and processing time.
    local gateway_handler="actions/setup/js/safe_outputs_handlers.cjs"
    local add_comment_handler="actions/setup/js/add_comment.cjs"
    
    for handler in "$gateway_handler" "$add_comment_handler"; do
        if [ ! -f "$handler" ]; then
            log_medium "MCE-002: Expected handler file missing: $handler"
            failed=1
            continue
        fi
        if ! grep -q "comment_limit_helpers\|enforceCommentLimits" "$handler"; then
            log_medium "MCE-002: $handler does not enforce comment constraints (dual enforcement pattern)"
            failed=1
        fi
    done
    
    if [ $failed -eq 0 ]; then
        log_pass "MCE-002: Dual enforcement pattern implemented in both gateway and processor"
    fi
}
check_mce_dual_enforcement

# CI-001: Cache Memory Integrity Scripts Exist (Section 11 CI6, CI10)
echo "Running CI-001: Cache Memory Integrity Scripts Exist..."
check_cache_memory_scripts() {
    local failed=0
    
    # Per spec Section 11 CI6 and CI10: setup and commit scripts must exist
    local setup_script="actions/setup/sh/setup_cache_memory_git.sh"
    local commit_script="actions/setup/sh/commit_cache_memory_git.sh"
    
    for script in "$setup_script" "$commit_script"; do
        if [ ! -f "$script" ]; then
            log_high "CI-001: Required cache memory integrity script missing: $script"
            failed=1
        fi
    done
    
    if [ $failed -eq 0 ]; then
        log_pass "CI-001: Cache memory integrity scripts exist"
    fi
}
check_cache_memory_scripts

# CI-002: Cache Memory Integrity Branch Support (Section 11.2, CI7, CI8)
echo "Running CI-002: Cache Memory Integrity Branch Support..."
check_cache_integrity_branches() {
    local setup_script="actions/setup/sh/setup_cache_memory_git.sh"
    local commit_script="actions/setup/sh/commit_cache_memory_git.sh"
    local failed=0
    
    if [ ! -f "$setup_script" ]; then
        log_medium "CI-002: Setup script missing — skipping integrity branch check"
        return
    fi
    
    # Per spec Section 11.2: all four integrity levels must be supported (merged > approved > unapproved > none)
    for level in merged approved unapproved none; do
        if ! grep -q "\"$level\"\|'$level'" "$setup_script"; then
            log_high "CI-002: Integrity level '$level' not found in setup script"
            failed=1
        fi
    done
    
    # Per spec CI8: merge-down from higher-integrity branches must be implemented
    if ! grep -q "merge\|git merge" "$setup_script"; then
        log_high "CI-002: Setup script missing merge-down implementation (CI8)"
        failed=1
    fi
    
    # Per spec CI9: merge failure must abort and exit with non-zero status
    if ! grep -qE "merge.*abort|abort.*merge|exit.*\$|exit [0-9]" "$setup_script"; then
        log_high "CI-002: Setup script missing merge failure abort/exit handling (CI9)"
        failed=1
    fi
    
    # Per spec CI11: commit script must invoke git gc --auto for compaction
    if [ -f "$commit_script" ]; then
        if ! grep -q "git gc" "$commit_script"; then
            log_medium "CI-002: Commit script missing 'git gc --auto' (CI11 repository compaction)"
            failed=1
        fi
    fi
    
    # Per spec CI12: commit script must handle missing .git gracefully
    if [ -f "$commit_script" ]; then
        if ! grep -q '\.git\|no.*git\|skip.*git' "$commit_script"; then
            log_medium "CI-002: Commit script may not handle missing .git directory (CI12)"
            failed=1
        fi
    fi
    
    if [ $failed -eq 0 ]; then
        log_pass "CI-002: Cache memory integrity branching properly implemented"
    fi
}
check_cache_integrity_branches

# MCE-003: Constraint Limit Consistency (Section 8.3 MCE5)
echo "Running MCE-003: Constraint Limit Consistency..."
check_mce_constraint_consistency() {
    local tools_json="pkg/workflow/js/safe_outputs_tools.json"
    local helpers_file="actions/setup/js/comment_limit_helpers.cjs"
    local failed=0

    if [ ! -f "$helpers_file" ]; then
        log_high "MCE-003: Constraint helper module missing: $helpers_file"
        return
    fi
    if [ ! -f "$tools_json" ]; then
        log_high "MCE-003: Tool definitions file missing: $tools_json"
        return
    fi

    # Per spec Section 8.3 MCE5: limits in tool descriptions MUST match enforcement code.
    # Extract the declared limits from comment_limit_helpers.cjs and verify they appear
    # verbatim in the tools JSON, ensuring both layers agree on the constraint values.

    # Check body length limit (65536) appears in both files
    if ! grep -q "65536" "$helpers_file"; then
        log_high "MCE-003: MAX_COMMENT_LENGTH (65536) not found in $helpers_file"
        failed=1
    fi
    if ! grep -q "65536" "$tools_json"; then
        log_high "MCE-003: 65536 character limit not found in $tools_json"
        failed=1
    fi

    # Check mention limit (10) appears in both files
    if ! grep -qE "MAX_MENTIONS\s*=\s*10|max.*mention.*10|10.*mention" "$helpers_file"; then
        log_high "MCE-003: MAX_MENTIONS (10) not declared in $helpers_file"
        failed=1
    fi
    if ! grep -qiE "10 mention|mention.*10" "$tools_json"; then
        log_high "MCE-003: 10-mention limit not found in $tools_json"
        failed=1
    fi

    # Check link limit (50) appears in both files
    if ! grep -qE "MAX_LINKS\s*=\s*50|max.*link.*50|50.*link" "$helpers_file"; then
        log_high "MCE-003: MAX_LINKS (50) not declared in $helpers_file"
        failed=1
    fi
    if ! grep -qiE "50 link|link.*50" "$tools_json"; then
        log_high "MCE-003: 50-link limit not found in $tools_json"
        failed=1
    fi

    if [ $failed -eq 0 ]; then
        log_pass "MCE-003: Constraint limits are consistent between tool descriptions and enforcement code"
    fi
}
check_mce_constraint_consistency

# CI-003: Policy Hash and Nopolicy Sentinel (Section 11.4 CI3, CI4)
echo "Running CI-003: Policy Hash and Nopolicy Sentinel..."
check_policy_hash_implementation() {
    local cache_integrity_file="pkg/workflow/cache_integrity.go"
    local failed=0

    # Per spec Section 11.4 CI3: policy hash MUST be SHA-256, first 8 chars of lowercase hex
    # Per spec Section 11.4 CI4: workflows without policy MUST use "nopolicy" sentinel
    if [ ! -f "$cache_integrity_file" ]; then
        log_high "CI-003: Cache integrity implementation missing: $cache_integrity_file"
        return
    fi

    # Check SHA-256 is used for policy hash computation (CI3)
    if ! grep -q "sha256\|crypto/sha256" "$cache_integrity_file"; then
        log_high "CI-003: SHA-256 not used for policy hash computation (CI3)"
        failed=1
    fi

    # Check nopolicy sentinel constant exists (CI4)
    if ! grep -q "nopolicy\|noPolicySentinel" "$cache_integrity_file"; then
        log_high "CI-003: 'nopolicy' sentinel not found in cache integrity implementation (CI4)"
        failed=1
    fi

    # Check that the 8-character prefix is taken (CI3: first 8 chars of lowercase hex)
    if ! grep -qE "\[:8\]|first.*8|8.*char|hex\[:8\]" "$cache_integrity_file"; then
        log_medium "CI-003: 8-character hash prefix truncation not evident in $cache_integrity_file (CI3)"
        failed=1
    fi

    if [ $failed -eq 0 ]; then
        log_pass "CI-003: Policy hash uses SHA-256 with nopolicy sentinel as required"
    fi
}
check_policy_hash_implementation

# CI-004: .git Directory Exclusion from Cache Memory Validation (Section 11.5 CI5)
echo "Running CI-004: .git Directory Exclusion from Validation..."
check_git_dir_exclusion() {
    local setup_script="actions/setup/sh/setup_cache_memory_git.sh"
    local failed=0

    # Per spec Section 11.5 CI5: file validation steps MUST skip the .git directory.
    # The .git directory contains binary/extension-less files not managed by the agent.

    if [ ! -f "$setup_script" ]; then
        log_medium "CI-004: Setup script missing — skipping .git exclusion check"
        return
    fi

    # Check that .git is referenced in the context of exclusion or skip logic
    if ! grep -qE "\.git|git_dir|skip.*\.git|exclude.*git|prune.*git" "$setup_script"; then
        log_medium "CI-004: Setup script does not reference .git exclusion (CI5)"
        failed=1
    fi

    # Check compiled workflow lock files: cache-memory file validation should skip .git
    if find .github/workflows -name "*.lock.yml" | xargs grep -l "cache-memory\|GH_AW_CACHE_MEMORY" 2>/dev/null | \
        xargs grep -l "validate\|allowed.*ext\|file.*check" 2>/dev/null | \
        xargs grep -qv "\.git\|skip.*git" 2>/dev/null; then
        log_low "CI-004: Some cache-memory workflow lock files may not exclude .git in validation (CI5)"
        # Not failing here — informational only as implementation details vary
    fi

    if [ $failed -eq 0 ]; then
        log_pass "CI-004: .git directory exclusion from validation is present"
    fi
}
check_git_dir_exclusion

# CI-005: Integrity-Scoped Cache Keys (Section 11.3 CI1)
echo "Running CI-005: Integrity-Scoped Cache Keys..."
check_integrity_scoped_keys() {
    local failed=0

    # Per spec Section 11.3 CI1: All cache-memory keys MUST include the integrity level
    # and policy hash as prefixes in the format: memory-{integrityLevel}-{policyHash}-...

    local cache_lock_files
    cache_lock_files=$(find .github/workflows -name "*.lock.yml" -exec grep -l "GH_AW_CACHE_MEMORY\|cache-memory" {} \; 2>/dev/null)

    if [ -z "$cache_lock_files" ]; then
        log_pass "CI-005: No cache-memory workflows found — nothing to check"
        return
    fi

    while IFS= read -r workflow; do
        # Extract cache keys from the workflow
        # Valid format: memory-{integrityLevel}-{policyHash}-...
        # integrityLevel must be one of: merged, approved, unapproved, none
        # policyHash must be 8-char hex or the sentinel "nopolicy"
        while IFS= read -r key_line; do
            key=$(echo "$key_line" | sed 's/.*key:\s*//')
            if [[ "$key" =~ ^memory- ]]; then
                if ! echo "$key" | grep -qE "^memory-(merged|approved|unapproved|none)-(nopolicy|[0-9a-f]{8})-"; then
                    log_high "CI-005: Cache key in $workflow does not follow integrity-scoped format (CI1): $key"
                    failed=1
                fi
            fi
        done < <(grep "key: memory-" "$workflow" 2>/dev/null)
    done <<< "$cache_lock_files"

    if [ $failed -eq 0 ]; then
        log_pass "CI-005: All cache-memory keys use the integrity-scoped format (CI1)"
    fi
}
check_integrity_scoped_keys

# CI-006: Restore Key Cascade (Section 11.3 CI2)
echo "Running CI-006: Restore Key Cascade..."
check_restore_key_cascade() {
    local failed=0

    # Per spec Section 11.3 CI2: Restore keys MUST use the same integrity-scoped prefix
    # so that a partial key match never crosses integrity level boundaries.
    # The restore-keys pattern must not contain a run_id (to allow matching prior runs).

    local cache_lock_files
    cache_lock_files=$(find .github/workflows -name "*.lock.yml" -exec grep -l "GH_AW_CACHE_MEMORY\|cache-memory" {} \; 2>/dev/null)

    if [ -z "$cache_lock_files" ]; then
        log_pass "CI-006: No cache-memory workflows found — nothing to check"
        return
    fi

    while IFS= read -r workflow; do
        # Check each restore-key entry that starts with "memory-"
        while IFS= read -r restore_key; do
            key=$(echo "$restore_key" | sed 's/^\s*//')
            if [[ "$key" =~ ^memory- ]]; then
                # restore-keys should include the integrity level and policy hash prefix
                # but must NOT include the run_id (to match prior runs of same workflow)
                if ! echo "$key" | grep -qE "^memory-(merged|approved|unapproved|none)-(nopolicy|[0-9a-f]{8})-"; then
                    # Allow the legacy fallback "memory-" entry documented in spec
                    if [ "$key" != "memory-" ]; then
                        log_high "CI-006: Restore key in $workflow does not use integrity-scoped prefix (CI2): $key"
                        failed=1
                    fi
                fi
                # Restore keys must NOT include github.run_id (they are prefix-only)
                if echo "$key" | grep -q "run_id"; then
                    log_medium "CI-006: Restore key in $workflow includes run_id — should be prefix-only for cascade (CI2): $key"
                    failed=1
                fi
            fi
        done < <(awk '/restore-keys:/,/^[^|]/' "$workflow" 2>/dev/null | grep "memory-")
    done <<< "$cache_lock_files"

    if [ $failed -eq 0 ]; then
        log_pass "CI-006: All cache-memory restore keys use the integrity-scoped prefix (CI2)"
    fi
}
check_restore_key_cascade

# MCE-004: Early Validation at MCP Invocation (Section 8.3 MCE1)
echo "Running MCE-004: Early Validation at MCP Invocation..."
check_mce_early_validation() {
    local gateway_handler="actions/setup/js/safe_outputs_handlers.cjs"
    local failed=0

    # Per spec Section 8.3 MCE1: MCP servers MUST enforce operational constraints during
    # tool invocation (Phase 4) rather than deferring all validation to safe output
    # processing (Phase 6). This provides immediate feedback to the LLM.

    if [ ! -f "$gateway_handler" ]; then
        log_high "MCE-004: MCP gateway handler missing: $gateway_handler"
        return
    fi

    # Check that constraint validation (enforceCommentLimits or equivalent) is called
    # before the operation is appended to safe outputs (appendSafeOutput)
    if ! grep -q "enforceCommentLimits\|enforceConstraints\|validateEarly" "$gateway_handler"; then
        log_high "MCE-004: MCP gateway handler does not call early constraint validation (MCE1)"
        failed=1
    fi

    # Verify the handler references MCE1 requirement in documentation
    if ! grep -q "MCE1\|Early Validation\|tool invocation" "$gateway_handler"; then
        log_medium "MCE-004: MCE1 early validation pattern not documented in $gateway_handler"
        failed=1
    fi

    # Ensure validation occurs before recording (appendSafeOutput / recordOperation)
    # by checking that enforceCommentLimits appears before appendSafeOutput in the file
    enforce_line=$(grep -n "enforceCommentLimits" "$gateway_handler" | head -1 | cut -d: -f1)
    append_line=$(grep -n "appendSafeOutput" "$gateway_handler" | head -1 | cut -d: -f1)
    if [ -n "$enforce_line" ] && [ -n "$append_line" ]; then
        if [ "$enforce_line" -gt "$append_line" ]; then
            log_critical "MCE-004: enforceCommentLimits appears AFTER appendSafeOutput — validation is not early (MCE1)"
            failed=1
        fi
    fi

    if [ $failed -eq 0 ]; then
        log_pass "MCE-004: MCP gateway enforces constraints early at tool invocation (MCE1)"
    fi
}
check_mce_early_validation

# MCE-005: Actionable Error Responses (Section 8.3 MCE3)
echo "Running MCE-005: Actionable Error Responses..."
check_mce_actionable_errors() {
    local helpers_file="actions/setup/js/comment_limit_helpers.cjs"
    local gateway_handler="actions/setup/js/safe_outputs_handlers.cjs"
    local failed=0

    # Per spec Section 8.3 MCE3: When constraints are violated, MCP servers MUST return
    # error responses that:
    #   1. Identify the violated constraint with specific name and limit
    #   2. Report the actual value that triggered the violation
    #   3. Provide remediation guidance on how to correct the issue
    #   4. Use standard error codes (E006-E008 for add_comment limits)

    if [ ! -f "$helpers_file" ]; then
        log_high "MCE-005: Constraint helper module missing: $helpers_file"
        return
    fi

    # Check that error messages identify the constraint name and limit (requirement 1)
    if ! grep -qE "E006.*length|E007.*mention|E008.*link" "$helpers_file"; then
        log_medium "MCE-005: Error messages do not clearly identify constraint name in $helpers_file (MCE3 req 1)"
        failed=1
    fi

    # Check that error messages report the actual violating value (requirement 2)
    # Look for patterns like "got ${body.length}" or "contains ${mentions}" 
    if ! grep -qE "got \\\${|contains \\\${|\\.length\}" "$helpers_file"; then
        log_medium "MCE-005: Error messages do not report the actual violating value in $helpers_file (MCE3 req 2)"
        failed=1
    fi

    # Check that error responses include remediation guidance (requirement 3)
    # Either via a structured data.guidance field or guidance-oriented language in errors
    if ! grep -qE "guidance|reduce|Reduce|consider|Consider|fewer|lower" "$helpers_file"; then
        log_low "MCE-005: Error responses in $helpers_file may lack remediation guidance (MCE3 req 3)"
        failed=1
    fi

    # Check that standard error codes are used (requirement 4)
    if ! grep -qE "E006|E007|E008" "$helpers_file"; then
        log_medium "MCE-005: Standard error codes E006-E008 not used in $helpers_file (MCE3 req 4)"
        failed=1
    fi

    # Check gateway handler re-throws with standard -32602 JSON-RPC error code
    if [ -f "$gateway_handler" ]; then
        if ! grep -q "\-32602" "$gateway_handler"; then
            log_medium "MCE-005: MCP gateway handler does not return -32602 JSON-RPC error code for constraint violations (MCE3)"
            failed=1
        fi
    fi

    if [ $failed -eq 0 ]; then
        log_pass "MCE-005: Error responses include constraint identification, actual values, and standard codes (MCE3)"
    fi
}
check_mce_actionable_errors

# Summary
echo ""
echo "=================================================="
echo "Conformance Check Summary"
echo "=================================================="
echo -e "${RED}Critical Failures:${NC} $CRITICAL_FAILURES"
echo -e "${RED}High Failures:${NC} $HIGH_FAILURES"
echo -e "${YELLOW}Medium Failures:${NC} $MEDIUM_FAILURES"
echo -e "${BLUE}Low Failures:${NC} $LOW_FAILURES"
echo ""

# Exit code based on failures
if [ $CRITICAL_FAILURES -gt 0 ]; then
    echo -e "${RED}FAIL:${NC} Critical conformance issues found"
    exit 2
elif [ $HIGH_FAILURES -gt 0 ]; then
    echo -e "${RED}FAIL:${NC} High priority conformance issues found"
    exit 1
elif [ $MEDIUM_FAILURES -gt 0 ]; then
    echo -e "${YELLOW}WARN:${NC} Medium priority conformance issues found"
    exit 0
else
    echo -e "${GREEN}PASS:${NC} All checks passed"
    exit 0
fi
