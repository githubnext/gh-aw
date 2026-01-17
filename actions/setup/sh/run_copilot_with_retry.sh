#!/usr/bin/env bash
#
# run_copilot_with_retry.sh - Execute Copilot CLI with automatic retry on finish_reason errors
#
# This script wraps the Copilot CLI execution and automatically retries if the command
# fails within 5 seconds with a "missing finish_reason" error. This handles transient
# API errors from the Copilot backend where the response is missing the required
# finish_reason field.
#
# Usage:
#   run_copilot_with_retry.sh <copilot_command_and_args...>
#
# Environment Variables:
#   COPILOT_RETRY_MAX_ATTEMPTS - Maximum number of retry attempts (default: 3)
#   COPILOT_RETRY_DELAY - Delay in seconds between retries (default: 2)
#
# Exit Codes:
#   0 - Success
#   1 - Failed after all retries
#   2 - Invalid arguments

set -euo pipefail

# Configuration
MAX_ATTEMPTS="${COPILOT_RETRY_MAX_ATTEMPTS:-3}"
RETRY_DELAY="${COPILOT_RETRY_DELAY:-2}"
QUICK_FAIL_THRESHOLD=5  # If failure occurs within 5 seconds, check for finish_reason error

# Colors for output
RED='\033[0;31m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

# Validate arguments
if [ $# -eq 0 ]; then
    echo -e "${RED}Error: No command provided${NC}" >&2
    echo "Usage: $0 <copilot_command_and_args...>" >&2
    exit 2
fi

# Function to check if error output contains finish_reason error
check_finish_reason_error() {
    local stderr_log="$1"
    if [ -f "$stderr_log" ]; then
        if grep -q "missing finish_reason" "$stderr_log"; then
            return 0
        fi
    fi
    return 1
}

# Main execution loop
attempt=1
success=false

while [ $attempt -le $MAX_ATTEMPTS ]; do
    echo -e "${GREEN}[Attempt $attempt/$MAX_ATTEMPTS]${NC} Executing: $*" >&2
    
    # Create temporary files for stdout and stderr capture
    stdout_log=$(mktemp)
    stderr_log=$(mktemp)
    trap "rm -f $stdout_log $stderr_log" EXIT
    
    # Record start time
    start_time=$(date +%s)
    
    # Execute the command, capturing both stdout and stderr while displaying in real-time
    # Use process substitution to tee both streams and capture stderr for error checking
    set +e
    "$@" > >(tee "$stdout_log") 2> >(tee "$stderr_log" >&2)
    exit_code=$?
    set -e
    
    # Wait for background processes to complete
    wait
    
    # Record end time and calculate duration
    end_time=$(date +%s)
    duration=$((end_time - start_time))
    
    # Check if command succeeded
    if [ $exit_code -eq 0 ]; then
        echo -e "${GREEN}✓ Command succeeded on attempt $attempt${NC}" >&2
        success=true
        break
    fi
    
    # Check if this was a quick failure (within threshold)
    if [ $duration -le $QUICK_FAIL_THRESHOLD ]; then
        echo -e "${YELLOW}⚠ Command failed quickly (${duration}s) - checking for finish_reason error${NC}" >&2
        
        # Check if the error is the finish_reason issue
        if check_finish_reason_error "$stderr_log"; then
            echo -e "${YELLOW}✗ Detected 'missing finish_reason' error${NC}" >&2
            
            if [ $attempt -lt $MAX_ATTEMPTS ]; then
                echo -e "${YELLOW}⟳ Retrying in ${RETRY_DELAY}s...${NC}" >&2
                sleep $RETRY_DELAY
                attempt=$((attempt + 1))
                rm -f "$stdout_log" "$stderr_log"
                continue
            else
                echo -e "${RED}✗ Max retry attempts reached${NC}" >&2
                exit 1
            fi
        else
            echo -e "${RED}✗ Quick failure but not a finish_reason error - not retrying${NC}" >&2
            exit $exit_code
        fi
    else
        # Failure after threshold - don't retry
        echo -e "${RED}✗ Command failed after ${duration}s - not a quick failure, not retrying${NC}" >&2
        exit $exit_code
    fi
done

if [ "$success" = true ]; then
    exit 0
else
    exit 1
fi
