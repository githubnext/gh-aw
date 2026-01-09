#!/bin/bash
#
# Audit smoke-copilot workflow runs and validate gateway logs in artifacts
#
# This script:
# 1. Finds recent smoke-copilot workflow runs
# 2. Downloads artifacts from the most recent run
# 3. Validates that gateway logs exist in the MCP logs
# 4. Analyzes potential errors in the logs

set -euo pipefail

REPO_OWNER="githubnext"
REPO_NAME="gh-aw"
WORKFLOW_FILE="smoke-copilot.md"

echo "🔍 Finding recent smoke-copilot workflow runs..."

# Get the most recent smoke-copilot run
# We'll try to use the gh CLI first, but this might require authentication
if command -v gh &> /dev/null; then
    echo "✓ GitHub CLI found, querying recent runs..."
    
    # List recent runs for the smoke-copilot workflow
    RECENT_RUNS=$(gh run list \
        --repo "${REPO_OWNER}/${REPO_NAME}" \
        --workflow="${WORKFLOW_FILE}" \
        --limit=5 \
        --json databaseId,status,conclusion,createdAt,displayTitle \
        2>&1 || echo "[]")
    
    if [ "$RECENT_RUNS" != "[]" ] && [ -n "$RECENT_RUNS" ]; then
        echo "Recent smoke-copilot runs:"
        echo "$RECENT_RUNS" | jq -r '.[] | "\(.databaseId) - \(.status)/\(.conclusion) - \(.createdAt)"'
        
        # Get the most recent run ID
        RUN_ID=$(echo "$RECENT_RUNS" | jq -r '.[0].databaseId')
        
        if [ -n "$RUN_ID" ] && [ "$RUN_ID" != "null" ]; then
            echo ""
            echo "📦 Auditing run ID: $RUN_ID"
            echo ""
            
            # Use gh-aw audit command to download and analyze
            ./gh-aw audit "$RUN_ID" --output .github/aw/logs -v
            
            # Check for gateway logs in the downloaded artifacts
            echo ""
            echo "🔍 Checking for gateway logs in artifacts..."
            
            RUN_DIR=".github/aw/logs/run-${RUN_ID}"
            
            if [ -d "$RUN_DIR" ]; then
                # Look for MCP logs directory
                MCP_LOGS_DIR="$RUN_DIR/mcp-logs"
                
                if [ -d "$MCP_LOGS_DIR" ]; then
                    echo "✓ Found MCP logs directory: $MCP_LOGS_DIR"
                    
                    # List all files in the MCP logs directory
                    echo ""
                    echo "Files in MCP logs directory:"
                    find "$MCP_LOGS_DIR" -type f -exec ls -lh {} \;
                    
                    # Check for gateway logs specifically
                    GATEWAY_LOGS=$(find "$MCP_LOGS_DIR" -name "*gateway*" -o -name "*mcp-gateway*" 2>/dev/null || true)
                    
                    if [ -n "$GATEWAY_LOGS" ]; then
                        echo ""
                        echo "✓ Found gateway log files:"
                        echo "$GATEWAY_LOGS"
                        
                        # Analyze each gateway log file
                        while IFS= read -r log_file; do
                            if [ -f "$log_file" ]; then
                                echo ""
                                echo "📄 Analyzing: $log_file"
                                echo "─────────────────────────────────────"
                                
                                # Check file size
                                SIZE=$(wc -c < "$log_file")
                                echo "File size: $SIZE bytes"
                                
                                # Look for errors or warnings
                                if grep -qi "error\|fatal\|fail\|panic" "$log_file"; then
                                    echo "⚠️  Found potential errors in log:"
                                    grep -i "error\|fatal\|fail\|panic" "$log_file" | head -20
                                else
                                    echo "✓ No obvious errors found"
                                fi
                                
                                # Show first 50 lines of the log
                                echo ""
                                echo "First 50 lines of log:"
                                head -50 "$log_file"
                            fi
                        done <<< "$GATEWAY_LOGS"
                    else
                        echo "⚠️  No gateway log files found in MCP logs directory"
                        echo "Files found:"
                        ls -la "$MCP_LOGS_DIR" || true
                    fi
                else
                    echo "⚠️  MCP logs directory not found: $MCP_LOGS_DIR"
                    echo "Available directories in run directory:"
                    ls -la "$RUN_DIR" || true
                fi
            else
                echo "⚠️  Run directory not found: $RUN_DIR"
            fi
            
            exit 0
        fi
    fi
    
    echo "⚠️  Could not retrieve workflow runs via gh CLI"
    echo "Output: $RECENT_RUNS"
fi

echo ""
echo "❌ Unable to audit smoke-copilot runs"
echo "This script requires:"
echo "  1. GitHub CLI (gh) to be installed and authenticated"
echo "  2. Access to the ${REPO_OWNER}/${REPO_NAME} repository"
echo ""
echo "Alternative: Provide a run ID directly:"
echo "  ./gh-aw audit <run-id> --output .github/aw/logs -v"

exit 1
