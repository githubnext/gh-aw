#!/bin/bash
# ralph-loop.sh - Autonomous agent loop that runs Copilot repeatedly until all PRD items are complete
#
# This script implements the Ralph pattern from https://github.com/snarktank/ralph
# Each iteration is a fresh Copilot agent with clean context
# Memory persists via git history, progress.txt, and prd.json
#
# Usage: ralph-loop.sh [max_iterations]
#   max_iterations: Maximum number of iterations (default: 10)
#
# Environment variables:
#   GH_AW_PROMPT: Path to the prompt file (required)
#   GH_AW_MCP_CONFIG: Path to MCP config file (required)
#   RALPH_MAX_ITERATIONS: Override max iterations from command line
#   COPILOT_GITHUB_TOKEN: GitHub token for Copilot (required)

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
MAX_ITERATIONS=${RALPH_MAX_ITERATIONS:-${1:-10}}
ITERATION=0
PRD_FILE="prd.json"
PROGRESS_FILE="progress.txt"

echo -e "${BLUE}=== Ralph Loop Starting ===${NC}"
echo "Max iterations: $MAX_ITERATIONS"
echo "PRD file: $PRD_FILE"
echo "Progress file: $PROGRESS_FILE"
echo ""

# Function to check if all stories are complete
all_stories_complete() {
    # Check if prd.json exists
    if [ ! -f "$PRD_FILE" ]; then
        echo -e "${RED}Error: prd.json not found${NC}"
        return 1
    fi

    # Count total stories and completed stories
    local total=$(jq '.userStories | length' "$PRD_FILE")
    local completed=$(jq '[.userStories[] | select(.passes == true)] | length' "$PRD_FILE")

    echo "Progress: $completed/$total stories complete"

    if [ "$completed" -eq "$total" ] && [ "$total" -gt 0 ]; then
        return 0
    else
        return 1
    fi
}

# Function to get the next incomplete story
get_next_story() {
    jq -r '.userStories[] | select(.passes == false) | .id' "$PRD_FILE" | head -1
}

# Function to update story status
update_story_status() {
    local story_id=$1
    local status=$2
    
    # Create a temporary file with updated status
    jq --arg id "$story_id" --argjson passes "$status" \
        '(.userStories[] | select(.id == $id) | .passes) |= $passes' \
        "$PRD_FILE" > "${PRD_FILE}.tmp"
    
    mv "${PRD_FILE}.tmp" "$PRD_FILE"
}

# Function to append to progress file
log_progress() {
    local message=$1
    echo "[$(date -u +"%Y-%m-%dT%H:%M:%SZ")] Iteration $ITERATION: $message" >> "$PROGRESS_FILE"
}

# Main loop
while [ $ITERATION -lt $MAX_ITERATIONS ]; do
    ITERATION=$((ITERATION + 1))
    
    echo -e "${BLUE}=== Iteration $ITERATION/$MAX_ITERATIONS ===${NC}"
    
    # Check if all stories are complete
    if all_stories_complete; then
        echo -e "${GREEN}✓ All stories complete!${NC}"
        echo "<promise>COMPLETE</promise>"
        log_progress "All stories complete"
        exit 0
    fi
    
    # Get next story to work on
    NEXT_STORY=$(get_next_story)
    if [ -z "$NEXT_STORY" ]; then
        echo -e "${YELLOW}Warning: No incomplete stories found${NC}"
        log_progress "No incomplete stories found"
        break
    fi
    
    echo -e "${BLUE}Working on story: $NEXT_STORY${NC}"
    log_progress "Starting story: $NEXT_STORY"
    
    # Generate dynamic prompt for this iteration
    ITERATION_PROMPT="/tmp/gh-aw/ralph/iteration-prompt.txt"
    cat > "$ITERATION_PROMPT" <<EOF
You are working on story: $NEXT_STORY

Current PRD status:
$(cat "$PRD_FILE" | jq -r '.userStories[] | "- [\(.passes | if . then "x" else " " end)] \(.id): \(.title)"')

Previous learnings:
$(tail -20 "$PROGRESS_FILE" 2>/dev/null || echo "No previous learnings")

Instructions:
1. Implement the story: $NEXT_STORY
2. Run tests and quality checks (typecheck, lint, tests)
3. Commit your changes if checks pass
4. Update AGENTS.md with any learnings or patterns discovered
5. Report success or failure

Original prompt:
$(cat "$GH_AW_PROMPT" 2>/dev/null || echo "No prompt file found")
EOF
    
    # Run Copilot with the dynamic prompt
    echo -e "${BLUE}Executing Copilot...${NC}"
    
    # Save current prompt and replace with iteration prompt
    cp "$GH_AW_PROMPT" "${GH_AW_PROMPT}.backup"
    cp "$ITERATION_PROMPT" "$GH_AW_PROMPT"
    
    # Execute Copilot (this will be the actual execution in the workflow)
    # For now, we'll simulate with a placeholder that the actual workflow will replace
    STORY_SUCCESS=false
    
    # In the actual workflow, Copilot execution happens here
    # The workflow will determine success based on Copilot's output
    # For this script, we'll check if there were new commits
    
    BEFORE_COMMITS=$(git rev-list --count HEAD 2>/dev/null || echo "0")
    
    # Placeholder for Copilot execution
    # The actual workflow will invoke copilot CLI here
    echo "Copilot execution would happen here..."
    
    AFTER_COMMITS=$(git rev-list --count HEAD 2>/dev/null || echo "0")
    
    # Check if commits were made (indicating progress)
    if [ "$AFTER_COMMITS" -gt "$BEFORE_COMMITS" ]; then
        STORY_SUCCESS=true
        echo -e "${GREEN}✓ Story completed: $NEXT_STORY${NC}"
        log_progress "Story completed: $NEXT_STORY"
        update_story_status "$NEXT_STORY" true
    else
        echo -e "${YELLOW}⚠ Story incomplete: $NEXT_STORY${NC}"
        log_progress "Story incomplete: $NEXT_STORY (no commits)"
        # Keep passes as false, will retry in next iteration if we have iterations left
    fi
    
    # Restore original prompt
    cp "${GH_AW_PROMPT}.backup" "$GH_AW_PROMPT"
    
    echo ""
done

# Loop completed
echo -e "${BLUE}=== Ralph Loop Completed ===${NC}"
if all_stories_complete; then
    echo -e "${GREEN}✓ All stories complete!${NC}"
    echo "<promise>COMPLETE</promise>"
    log_progress "All stories complete after $ITERATION iterations"
    exit 0
else
    echo -e "${YELLOW}⚠ Max iterations reached, some stories incomplete${NC}"
    log_progress "Max iterations reached, not all stories complete"
    exit 1
fi
