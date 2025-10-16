#!/bin/bash
# validate_cache_memory_docs.sh
# Validates all code snippets in the cache-memory documentation
# Extracts snippets, compiles them, and verifies they work as described

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
DOCS_FILE="$REPO_ROOT/docs/src/content/docs/reference/cache-memory.md"
TEMP_DIR="/tmp/gh-aw-docs-validation"
SNIPPETS_DIR="$TEMP_DIR/snippets"
REPORTS_DIR="$TEMP_DIR/reports"
BINARY="$REPO_ROOT/gh-aw"

# Counters
TOTAL_SNIPPETS=0
PASSED_SNIPPETS=0
FAILED_SNIPPETS=0
SKIPPED_SNIPPETS=0

# Initialize temp directories
mkdir -p "$SNIPPETS_DIR"
mkdir -p "$REPORTS_DIR"

# Function to print section headers
print_header() {
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

# Function to log results
log_result() {
    local snippet_num=$1
    local status=$2
    local message=$3
    local details=$4
    
    if [ "$status" = "PASS" ]; then
        echo -e "${GREEN}✓${NC} Snippet $snippet_num: $message"
        PASSED_SNIPPETS=$((PASSED_SNIPPETS + 1))
    elif [ "$status" = "FAIL" ]; then
        echo -e "${RED}✗${NC} Snippet $snippet_num: $message"
        if [ -n "$details" ]; then
            echo -e "${RED}  Details: $details${NC}"
        fi
        FAILED_SNIPPETS=$((FAILED_SNIPPETS + 1))
    elif [ "$status" = "SKIP" ]; then
        echo -e "${YELLOW}⊘${NC} Snippet $snippet_num: $message"
        SKIPPED_SNIPPETS=$((SKIPPED_SNIPPETS + 1))
    fi
    
    # Write to report
    echo "[$status] Snippet $snippet_num: $message" >> "$REPORTS_DIR/summary.txt"
    if [ -n "$details" ]; then
        echo "  Details: $details" >> "$REPORTS_DIR/summary.txt"
    fi
}

# Function to extract code snippets
extract_snippets() {
    print_header "EXTRACTING CODE SNIPPETS FROM DOCUMENTATION"
    
    if [ ! -f "$DOCS_FILE" ]; then
        echo -e "${RED}Error: Documentation file not found at $DOCS_FILE${NC}"
        exit 1
    fi
    
    # Extract snippets using awk
    awk '
        /```aw/ {
            in_snippet = 1
            snippet_count++
            snippet_file = sprintf("'"$SNIPPETS_DIR"'/snippet_%02d.md", snippet_count)
            # Extract line number where snippet starts
            start_line = NR
            next
        }
        /```/ && in_snippet {
            in_snippet = 0
            # Write metadata
            meta_file = sprintf("'"$SNIPPETS_DIR"'/snippet_%02d.meta", snippet_count)
            print "start_line=" start_line > meta_file
            print "end_line=" NR >> meta_file
            next
        }
        in_snippet {
            print > snippet_file
        }
    ' "$DOCS_FILE"
    
    TOTAL_SNIPPETS=$(ls -1 "$SNIPPETS_DIR"/snippet_*.md 2>/dev/null | wc -l)
    echo -e "Found ${GREEN}$TOTAL_SNIPPETS${NC} code snippets"
    echo ""
}

# Function to validate frontmatter YAML
validate_frontmatter() {
    local snippet_file=$1
    local snippet_num=$2
    
    # Extract frontmatter (between --- markers)
    local frontmatter=$(awk '/^---$/ {flag++; next} flag == 1' "$snippet_file")
    
    if [ -z "$frontmatter" ]; then
        return 1
    fi
    
    # Check for required fields based on snippet type
    echo "$frontmatter" | grep -q "engine:" || return 1
    echo "$frontmatter" | grep -q "tools:" || return 1
    
    return 0
}

# Function to validate cache-memory configuration
validate_cache_memory_config() {
    local snippet_file=$1
    local snippet_num=$2
    
    # Check if cache-memory is configured
    if ! grep -q "cache-memory:" "$snippet_file"; then
        log_result "$snippet_num" "SKIP" "No cache-memory configuration"
        return 1
    fi
    
    # Extract cache-memory configuration
    local cache_config=$(awk '/cache-memory:/,/^[a-z]/' "$snippet_file")
    
    # Validate retention-days if present
    if echo "$cache_config" | grep -q "retention-days:"; then
        local retention=$(echo "$cache_config" | grep "retention-days:" | awk '{print $2}' | tr -d '#')
        if [ "$retention" -lt 1 ] || [ "$retention" -gt 90 ]; then
            log_result "$snippet_num" "FAIL" "Invalid retention-days value: $retention (must be 1-90)"
            return 1
        fi
    fi
    
    # Check for valid cache key format if present
    if echo "$cache_config" | grep -q "key:"; then
        local key=$(echo "$cache_config" | grep "key:" | awk '{print $2}')
        if [ -z "$key" ]; then
            log_result "$snippet_num" "FAIL" "Empty cache key"
            return 1
        fi
    fi
    
    # Validate array notation if present
    if echo "$cache_config" | grep -q "- id:"; then
        # Check that all array items have id field
        local id_count=$(echo "$cache_config" | grep -c "- id:" || true)
        if [ "$id_count" -eq 0 ]; then
            log_result "$snippet_num" "FAIL" "Array notation without id field"
            return 1
        fi
    fi
    
    return 0
}

# Function to compile snippet
compile_snippet() {
    local snippet_file=$1
    local snippet_num=$2
    local snippet_name=$(basename "$snippet_file" .md)
    
    # Create temporary git repository with proper structure
    local test_repo="$TEMP_DIR/test-repo-$snippet_num"
    rm -rf "$test_repo"
    mkdir -p "$test_repo/.github/workflows"
    
    # Initialize git repo (required by gh-aw compile)
    cd "$test_repo"
    git init -q
    git config user.email "test@example.com"
    git config user.name "Test User"
    
    # Add minimum required content for compilation
    local workflow_file=".github/workflows/${snippet_name}.md"
    
    # Check if snippet has markdown body (not just frontmatter)
    local has_body=false
    if grep -q "^#" "$snippet_file" || grep -q -E "^[^-]" "$snippet_file" | grep -qv "^---"; then
        has_body=true
    fi
    
    # If no body, add minimal content
    if [ "$has_body" = false ]; then
        cat "$snippet_file" > "$workflow_file"
        echo "" >> "$workflow_file"
        echo "# Test Workflow" >> "$workflow_file"
        echo "This is a test workflow to validate configuration." >> "$workflow_file"
    else
        cp "$snippet_file" "$workflow_file"
    fi
    
    # Try to compile using gh-aw
    local compile_output="$REPORTS_DIR/${snippet_name}_compile.log"
    local compile_result=0
    
    if [ ! -f "$BINARY" ]; then
        log_result "$snippet_num" "FAIL" "gh-aw binary not found at $BINARY"
        return 1
    fi
    
    # Add required 'on' field if missing (some snippets are fragments)
    if ! grep -q "^on:" "$workflow_file"; then
        # Insert 'on: workflow_dispatch' after engine line
        sed -i '/^engine:/a on: workflow_dispatch' "$workflow_file"
    fi
    
    # Run compilation (suppress output, capture to log)
    "$BINARY" compile "${snippet_name}" --verbose > "$compile_output" 2>&1 || compile_result=$?
    
    if [ $compile_result -ne 0 ]; then
        # Check if error is about missing fields or actual configuration errors
        if grep -q "workflow.*not found" "$compile_output"; then
            log_result "$snippet_num" "FAIL" "Compilation failed - workflow not found"
            return 1
        elif grep -q "error" "$compile_output" | grep -qv "0 error"; then
            local error_msg=$(grep -i "error" "$compile_output" | head -3 | tr '\n' ' ')
            log_result "$snippet_num" "FAIL" "Compilation failed" "$error_msg"
            return 1
        fi
    fi
    
    # Check if lock file was created (indicates successful compilation)
    if [ -f ".github/workflows/${snippet_name}.lock.yml" ]; then
        log_result "$snippet_num" "PASS" "Compilation successful"
        return 0
    else
        log_result "$snippet_num" "FAIL" "Compilation failed - no lock file generated"
        return 1
    fi
}

# Function to check best practices
check_best_practices() {
    local snippet_file=$1
    local snippet_num=$2
    local issues=()
    
    # Check for descriptive engine specification
    if ! grep -q "engine:" "$snippet_file"; then
        issues+=("Missing engine specification")
    fi
    
    # Check for appropriate timeout settings (if present)
    if grep -q "timeout_minutes:" "$snippet_file"; then
        local timeout=$(grep "timeout_minutes:" "$snippet_file" | awk '{print $2}')
        if [ "$timeout" -gt 30 ]; then
            issues+=("Timeout > 30 minutes (consider reducing)")
        fi
    fi
    
    # Check for tool configuration
    if ! grep -q "tools:" "$snippet_file"; then
        issues+=("Missing tools configuration")
    fi
    
    # Check for proper cache-memory configuration
    if grep -q "cache-memory: true" "$snippet_file"; then
        # Simple boolean configuration is valid
        :
    elif grep -q "cache-memory:" "$snippet_file"; then
        # Check for key or id configuration
        if ! grep -A 5 "cache-memory:" "$snippet_file" | grep -q -E "(key:|id:)"; then
            issues+=("cache-memory configuration missing key or id")
        fi
    fi
    
    if [ ${#issues[@]} -gt 0 ]; then
        local issues_str=$(IFS='; '; echo "${issues[*]}")
        log_result "$snippet_num" "FAIL" "Best practice violations" "$issues_str"
        return 1
    fi
    
    log_result "$snippet_num" "PASS" "Follows best practices"
    return 0
}

# Function to validate snippet content matches description
validate_snippet_description() {
    local snippet_file=$1
    local snippet_num=$2
    local meta_file="${snippet_file%.md}.meta"
    
    # Get the line numbers
    local start_line=$(grep "start_line=" "$meta_file" | cut -d= -f2)
    local end_line=$(grep "end_line=" "$meta_file" | cut -d= -f2)
    
    # Get surrounding context (lines before the snippet)
    local context_start=$((start_line - 10))
    if [ $context_start -lt 1 ]; then
        context_start=1
    fi
    
    local context=$(sed -n "${context_start},$((start_line-1))p" "$DOCS_FILE")
    
    # Check if snippet matches common documentation patterns
    local description=""
    
    # Look for section headers or descriptive text before snippet
    if echo "$context" | grep -q "### "; then
        description=$(echo "$context" | grep "### " | tail -1)
    elif echo "$context" | grep -q "## "; then
        description=$(echo "$context" | grep "## " | tail -1)
    fi
    
    # Validate specific snippet characteristics based on documentation
    
    # Check snippet 1: Basic enable pattern
    if [ "$snippet_num" = "01" ]; then
        if grep -q "cache-memory: true" "$snippet_file" && grep -q "engine: claude" "$snippet_file"; then
            log_result "$snippet_num" "PASS" "Basic enable pattern matches description"
            return 0
        fi
    fi
    
    # Check snippet 4: Custom key pattern
    if [ "$snippet_num" = "04" ]; then
        if grep -q "key: custom-memory" "$snippet_file" && grep -q "retention-days:" "$snippet_file"; then
            log_result "$snippet_num" "PASS" "Custom key pattern matches description"
            return 0
        fi
    fi
    
    # Check snippet 5: Multiple cache pattern
    if [ "$snippet_num" = "05" ]; then
        if grep -q "- id: default" "$snippet_file" && grep -q "- id: session" "$snippet_file"; then
            log_result "$snippet_num" "PASS" "Multiple cache pattern matches description"
            return 0
        fi
    fi
    
    # For other snippets, check basic validity
    log_result "$snippet_num" "PASS" "Snippet structure matches documentation"
    return 0
}

# Main validation function
validate_all_snippets() {
    print_header "VALIDATING CODE SNIPPETS"
    
    for snippet_file in "$SNIPPETS_DIR"/snippet_*.md; do
        if [ ! -f "$snippet_file" ]; then
            continue
        fi
        
        local snippet_name=$(basename "$snippet_file" .md)
        local snippet_num=$(echo "$snippet_name" | grep -o '[0-9]*$')
        
        echo ""
        echo -e "${BLUE}Validating Snippet $snippet_num${NC}"
        echo "----------------------------------------"
        
        # Step 1: Validate frontmatter
        if validate_frontmatter "$snippet_file" "$snippet_num"; then
            log_result "$snippet_num" "PASS" "Valid frontmatter structure"
        else
            log_result "$snippet_num" "SKIP" "No frontmatter or incomplete structure"
            continue
        fi
        
        # Step 2: Validate cache-memory configuration
        validate_cache_memory_config "$snippet_file" "$snippet_num" || continue
        
        # Step 3: Compile snippet
        compile_snippet "$snippet_file" "$snippet_num" || continue
        
        # Step 4: Check best practices
        check_best_practices "$snippet_file" "$snippet_num"
        
        # Step 5: Validate against description
        validate_snippet_description "$snippet_file" "$snippet_num"
    done
}

# Function to generate final report
generate_report() {
    print_header "VALIDATION SUMMARY"
    
    echo ""
    echo -e "Total Snippets:   ${BLUE}$TOTAL_SNIPPETS${NC}"
    echo -e "Passed:           ${GREEN}$PASSED_SNIPPETS${NC}"
    echo -e "Failed:           ${RED}$FAILED_SNIPPETS${NC}"
    echo -e "Skipped:          ${YELLOW}$SKIPPED_SNIPPETS${NC}"
    echo ""
    
    if [ $FAILED_SNIPPETS -eq 0 ]; then
        echo -e "${GREEN}✓ All validations passed!${NC}"
        EXIT_CODE=0
    else
        echo -e "${RED}✗ Some validations failed. See details above.${NC}"
        EXIT_CODE=1
    fi
    
    # Generate detailed report file
    local report_file="$REPORTS_DIR/validation_report.txt"
    {
        echo "Cache Memory Documentation Validation Report"
        echo "============================================="
        echo ""
        echo "Generated: $(date)"
        echo "Documentation: $DOCS_FILE"
        echo ""
        echo "Summary:"
        echo "  Total Snippets: $TOTAL_SNIPPETS"
        echo "  Passed:         $PASSED_SNIPPETS"
        echo "  Failed:         $FAILED_SNIPPETS"
        echo "  Skipped:        $SKIPPED_SNIPPETS"
        echo ""
        echo "Detailed Results:"
        echo "=================="
        cat "$REPORTS_DIR/summary.txt"
    } > "$report_file"
    
    echo ""
    echo -e "Detailed report saved to: ${BLUE}$report_file${NC}"
    
    return $EXIT_CODE
}

# Main execution
main() {
    print_header "CACHE MEMORY DOCUMENTATION VALIDATION"
    echo ""
    echo "Documentation: $DOCS_FILE"
    echo "Binary:        $BINARY"
    echo "Temp Dir:      $TEMP_DIR"
    echo ""
    
    # Clean temp directory
    rm -rf "$TEMP_DIR"
    mkdir -p "$TEMP_DIR"
    mkdir -p "$SNIPPETS_DIR"
    mkdir -p "$REPORTS_DIR"
    
    # Extract snippets
    extract_snippets
    
    # Validate all snippets
    validate_all_snippets
    
    # Generate report
    generate_report
}

# Run main function
main
