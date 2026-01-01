#!/bin/bash

# Monitor adoption of low-adoption schema fields in agentic workflows
# This script tracks 5 fields with ≤1 workflow adoption:
# - run-name
# - runtimes
# - runs-on
# - post-steps
# - bots

set -euo pipefail

# Configuration
WORKFLOWS_DIR=".github/workflows"
METRICS_FILE=".github/adoption-metrics.json"
THRESHOLD_PERCENT=3

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Fields to monitor
FIELDS=("run-name" "runtimes" "runs-on" "post-steps" "bots")

# Function to count workflows using a specific field
count_field_usage() {
    local field="$1"
    local count=0
    
    # Search for field in frontmatter (YAML between --- markers)
    # Use grep -l to list files, then count them
    count=$(grep -l "^${field}:" "${WORKFLOWS_DIR}"/*.md 2>/dev/null | wc -l | tr -d ' ')
    
    echo "${count}"
}

# Function to get total workflow count
get_total_workflows() {
    find "${WORKFLOWS_DIR}" -name "*.md" -type f | wc -l | tr -d ' '
}

# Function to calculate percentage
calculate_percentage() {
    local count=$1
    local total=$2
    
    if [ "$total" -eq 0 ]; then
        echo "0.0"
    else
        # Use awk for more reliable floating point arithmetic
        awk "BEGIN {printf \"%.1f\", ($count * 100) / $total}"
    fi
}

# Function to get status indicator
get_status() {
    local percent=$1
    local threshold=$2
    
    # Use awk for float comparison
    if [ "$(awk "BEGIN {print ($percent < $threshold)}")" -eq 1 ]; then
        echo "⚠️  BELOW THRESHOLD"
    else
        echo "✅ ABOVE THRESHOLD"
    fi
}

# Main monitoring function
monitor_adoption() {
    echo -e "${BLUE}=== Schema Field Adoption Monitor ===${NC}"
    echo -e "Date: $(date -u +"%Y-%m-%d %H:%M:%S UTC")"
    echo -e "Threshold: ${THRESHOLD_PERCENT}%\n"
    
    # Get total workflow count
    local total_workflows
    total_workflows=$(get_total_workflows)
    
    echo -e "${BLUE}Total workflows:${NC} ${total_workflows}\n"
    
    # Create JSON structure
    local timestamp
    timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    local json_data="{"
    json_data+="\"timestamp\":\"${timestamp}\","
    json_data+="\"total_workflows\":${total_workflows},"
    json_data+="\"threshold_percent\":${THRESHOLD_PERCENT},"
    json_data+="\"fields\":{"
    
    # Monitor each field
    echo -e "${BLUE}Field Adoption Report:${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    printf "%-15s %-10s %-12s %-20s\n" "Field" "Count" "Percentage" "Status"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    local first=true
    local below_threshold_count=0
    
    for field in "${FIELDS[@]}"; do
        local count
        count=$(count_field_usage "$field")
        
        local percent
        percent=$(calculate_percentage "$count" "$total_workflows")
        
        local status
        status=$(get_status "$percent" "$THRESHOLD_PERCENT")
        
        # Determine color based on threshold
        local color="${GREEN}"
        if [ "$(awk "BEGIN {print ($percent < $THRESHOLD_PERCENT)}")" -eq 1 ]; then
            color="${RED}"
            below_threshold_count=$((below_threshold_count + 1))
        fi
        
        printf "${color}%-15s %-10s %-12s %-20s${NC}\n" "$field" "$count" "${percent}%" "$status"
        
        # Add to JSON
        if [ "$first" = false ]; then
            json_data+=","
        fi
        first=false
        json_data+="\"${field}\":{\"count\":${count},\"percentage\":${percent}}"
    done
    
    json_data+="}}"
    
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    # Summary
    echo ""
    if [ "$below_threshold_count" -gt 0 ]; then
        echo -e "${RED}⚠️  ${below_threshold_count} field(s) below ${THRESHOLD_PERCENT}% threshold${NC}"
        echo -e "${YELLOW}Consider deprecation discussion for fields with sustained low adoption${NC}"
    else
        echo -e "${GREEN}✅ All fields above ${THRESHOLD_PERCENT}% threshold${NC}"
    fi
    
    # Update metrics file
    if [ -f "$METRICS_FILE" ]; then
        # Append to existing metrics
        local temp_file
        temp_file=$(mktemp)
        jq ". += [${json_data}]" "$METRICS_FILE" > "$temp_file" && mv "$temp_file" "$METRICS_FILE"
        echo -e "\n${GREEN}✓${NC} Updated metrics file: ${METRICS_FILE}"
    else
        # Create new metrics file
        echo "[${json_data}]" | jq '.' > "$METRICS_FILE"
        echo -e "\n${GREEN}✓${NC} Created metrics file: ${METRICS_FILE}"
    fi
    
    # Show trend if historical data exists
    show_trend
    
    # Return exit code based on threshold
    if [ "$below_threshold_count" -gt 0 ]; then
        return 1
    else
        return 0
    fi
}

# Function to show trend from historical data
show_trend() {
    if [ ! -f "$METRICS_FILE" ]; then
        return
    fi
    
    local records_count
    records_count=$(jq 'length' "$METRICS_FILE")
    
    if [ "$records_count" -lt 2 ]; then
        return
    fi
    
    echo -e "\n${BLUE}Trend Analysis (comparing with previous measurement):${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    printf "%-15s %-12s %-12s %-15s\n" "Field" "Previous" "Current" "Change"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    for field in "${FIELDS[@]}"; do
        local current_count
        current_count=$(jq -r ".[-1].fields.\"${field}\".count" "$METRICS_FILE")
        
        local previous_count
        previous_count=$(jq -r ".[-2].fields.\"${field}\".count" "$METRICS_FILE")
        
        local change=$((current_count - previous_count))
        local change_str="${change}"
        
        if [ "$change" -gt 0 ]; then
            change_str="${GREEN}+${change}${NC}"
        elif [ "$change" -lt 0 ]; then
            change_str="${RED}${change}${NC}"
        else
            change_str="${YELLOW}±${change}${NC}"
        fi
        
        printf "%-15s %-12s %-12s ${change_str}\n" "$field" "$previous_count" "$current_count"
    done
    
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
}

# Parse command line arguments
OUTPUT_FORMAT="terminal"
while [[ $# -gt 0 ]]; do
    case $1 in
        --json)
            OUTPUT_FORMAT="json"
            shift
            ;;
        --help|-h)
            echo "Usage: $0 [--json] [--help]"
            echo ""
            echo "Monitor adoption of low-adoption schema fields in agentic workflows"
            echo ""
            echo "Options:"
            echo "  --json    Output results in JSON format"
            echo "  --help    Show this help message"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Run monitor
if [ "$OUTPUT_FORMAT" = "json" ]; then
    # JSON output only
    if [ -f "$METRICS_FILE" ]; then
        jq '.[-1]' "$METRICS_FILE"
    else
        monitor_adoption > /dev/null 2>&1
        jq '.[-1]' "$METRICS_FILE"
    fi
else
    # Terminal output
    monitor_adoption
fi
