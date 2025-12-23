#!/bin/bash
# Analyze GitHub Actions versions across all workflow files
# This script identifies version sprawl and suggests standardization

set -e

echo "=== GitHub Actions Version Analysis ==="
echo ""

# Find all workflow files (both .md and compiled .lock.yml)
MD_FILES=$(find .github/workflows -name "*.md" -type f | sort)
YML_FILES=$(find .github/workflows -name "*.lock.yml" -o -name "*.yml" | grep -v ".lock.yml" | sort)

# Function to analyze actions in files
analyze_action_versions() {
    local file_pattern=$1
    local file_list=$2
    
    echo "## Analyzing $file_pattern files..."
    echo ""
    
    # Get all unique action references
    for action in "actions/checkout" "actions/setup-node" "actions/setup-python" "actions/setup-go" "actions/upload-artifact" "actions/download-artifact"; do
        echo "### $action"
        
        # Find all unique versions
        versions=$(echo "$file_list" | xargs grep -h "${action}@" 2>/dev/null | \
                   grep -oP "${action}@\S+" | \
                   sort -u | \
                   sed 's/^/  - /')
        
        if [ -n "$versions" ]; then
            echo "$versions"
            
            # Count occurrences
            count=$(echo "$versions" | wc -l)
            echo "  Total unique versions: $count"
        else
            echo "  Not used"
        fi
        echo ""
    done
}

# Analyze .md workflow files
if [ -n "$MD_FILES" ]; then
    analyze_action_versions "Markdown (.md)" "$MD_FILES"
fi

# Analyze .yml workflow files  
if [ -n "$YML_FILES" ]; then
    analyze_action_versions "YAML (.yml)" "$YML_FILES"
fi

# Summary statistics
echo "## Summary Statistics"
echo ""

total_md=$(echo "$MD_FILES" | wc -l)
total_yml=$(echo "$YML_FILES" | wc -l)

echo "- Total Markdown workflows: $total_md"
echo "- Total YAML workflows: $total_yml"
echo ""

# Check for version sprawl
checkout_versions=$(find .github/workflows \( -name "*.md" -o -name "*.yml" -o -name "*.yaml" \) -type f -exec grep -h "actions/checkout@" {} \; 2>/dev/null | \
                    grep -oP 'actions/checkout@\S+' | \
                    sort -u | \
                    wc -l)

echo "- Unique actions/checkout versions: $checkout_versions"

if [ "$checkout_versions" -gt 1 ]; then
    echo ""
    echo "⚠️  WARNING: Multiple versions of actions/checkout detected!"
    echo "   Recommendation: Standardize to actions/checkout@v5 (latest stable)"
fi

echo ""
echo "=== Analysis Complete ==="
