#!/bin/bash
# Run agentic engine in an iterative loop
# On each iteration, appends the output from the previous iteration to the original prompt
# This implements a ralph-loop style iterative refinement pattern

set -e
set -o pipefail

# ============================================================================
# Configuration and Validation
# ============================================================================

# Required environment variables
ITERATIONS="${GH_AW_ITERATIONS:-1}"
PROMPT_FILE="${GH_AW_PROMPT:-/tmp/gh-aw/aw-prompts/prompt.txt}"
LOGS_DIR="${GH_AW_LOGS_DIR:-/tmp/gh-aw/logs}"
LOG_FILE="${GH_AW_LOG_FILE:-$LOGS_DIR/agent-stdio.log}"

# Validate required inputs
if [ ! -f "$PROMPT_FILE" ]; then
    echo "❌ Error: Prompt file not found at $PROMPT_FILE"
    exit 1
fi

# Validate iterations is a positive integer
if ! [[ "$ITERATIONS" =~ ^[0-9]+$ ]] || [ "$ITERATIONS" -lt 1 ]; then
    echo "❌ Error: ITERATIONS must be a positive integer (got: $ITERATIONS)"
    exit 1
fi

# Skip loop if iterations is 1 (single run)
if [ "$ITERATIONS" -eq 1 ]; then
    echo "ℹ️  Iterations set to 1, running single execution (no loop)"
    # Execute the command passed as arguments directly
    exec "$@"
fi

echo "🔄 Starting iterative agent loop with $ITERATIONS iterations"
echo "📝 Original prompt file: $PROMPT_FILE"
echo "📁 Logs directory: $LOGS_DIR"
echo "📄 Log file: $LOG_FILE"

# ============================================================================
# Setup
# ============================================================================

# Create directories if they don't exist
mkdir -p "$LOGS_DIR"
mkdir -p "$(dirname "$PROMPT_FILE")"

# Save the original prompt
ORIGINAL_PROMPT_FILE="$LOGS_DIR/original-prompt.txt"
cp "$PROMPT_FILE" "$ORIGINAL_PROMPT_FILE"
echo "💾 Saved original prompt to: $ORIGINAL_PROMPT_FILE"

# Store iteration results
ITERATION_RESULTS_DIR="$LOGS_DIR/iterations"
mkdir -p "$ITERATION_RESULTS_DIR"

# Combined log for all iterations
COMBINED_LOG="$LOGS_DIR/combined-iterations.log"
> "$COMBINED_LOG"

echo ""
echo "================================"
echo "Iteration Loop Configuration"
echo "================================"
echo "Iterations: $ITERATIONS"
echo "Original Prompt: $(wc -l < "$ORIGINAL_PROMPT_FILE") lines, $(wc -c < "$ORIGINAL_PROMPT_FILE") bytes"
echo "Command: $*"
echo "================================"
echo ""

# ============================================================================
# Iteration Loop
# ============================================================================

for ((i=1; i<=ITERATIONS; i++)); do
    echo ""
    echo "========================================"
    echo "Iteration $i of $ITERATIONS"
    echo "========================================"
    echo ""
    
    # Set iteration-specific log file
    ITERATION_LOG="$ITERATION_RESULTS_DIR/iteration-$i.log"
    ITERATION_OUTPUT="$ITERATION_RESULTS_DIR/iteration-$i-output.txt"
    
    echo "📝 Iteration $i log: $ITERATION_LOG"
    echo "📤 Iteration $i output: $ITERATION_OUTPUT"
    
    # Log iteration start time
    START_TIME=$(date +%s)
    echo "⏱️  Start time: $(date -u +"%Y-%m-%d %H:%M:%S UTC")"
    
    # Execute the agent command
    echo ""
    echo "🚀 Executing iteration $i..."
    echo "   Command: $*"
    echo ""
    
    # Run the command and capture output
    # Use a subshell to prevent early exit on command failure
    EXIT_CODE=0
    if ! "$@" 2>&1 | tee "$ITERATION_LOG"; then
        EXIT_CODE=$?
        echo ""
        echo "❌ Iteration $i failed with exit code $EXIT_CODE"
    else
        echo ""
        echo "✅ Iteration $i completed successfully"
    fi
    
    # Log iteration end time and duration
    END_TIME=$(date +%s)
    DURATION=$((END_TIME - START_TIME))
    echo "⏱️  End time: $(date -u +"%Y-%m-%d %H:%M:%S UTC")"
    echo "⏱️  Duration: ${DURATION}s"
    
    # Extract meaningful output from the iteration log
    # This is a simplified extraction - you may want to customize based on engine output format
    echo ""
    echo "📊 Extracting output from iteration $i..."
    
    # Copy the full log as output for now
    # Future enhancement: parse engine-specific output format
    cp "$ITERATION_LOG" "$ITERATION_OUTPUT"
    
    OUTPUT_SIZE=$(wc -c < "$ITERATION_OUTPUT")
    echo "   Output size: $OUTPUT_SIZE bytes"
    
    # Append iteration log to combined log
    {
        echo ""
        echo "=========================================="
        echo "Iteration $i (Duration: ${DURATION}s)"
        echo "=========================================="
        echo ""
        cat "$ITERATION_LOG"
    } >> "$COMBINED_LOG"
    
    # Exit if command failed
    if [ $EXIT_CODE -ne 0 ]; then
        echo ""
        echo "❌ Stopping iteration loop due to failure in iteration $i"
        echo "📁 Partial results saved in: $ITERATION_RESULTS_DIR"
        exit $EXIT_CODE
    fi
    
    # If not the last iteration, append output to prompt for next iteration
    if [ $i -lt "$ITERATIONS" ]; then
        echo ""
        echo "🔄 Preparing prompt for next iteration..."
        
        # Create the augmented prompt for next iteration
        AUGMENTED_PROMPT_FILE="$ITERATION_RESULTS_DIR/prompt-iteration-$((i+1)).txt"
        
        {
            # Start with original prompt
            cat "$ORIGINAL_PROMPT_FILE"
            echo ""
            echo ""
            echo "---"
            echo ""
            echo "# Previous Iteration Results"
            echo ""
            echo "The following is the output from iteration $i. Please review and build upon this work:"
            echo ""
            echo '```'
            cat "$ITERATION_OUTPUT"
            echo '```'
        } > "$AUGMENTED_PROMPT_FILE"
        
        # Replace the prompt file with the augmented version
        cp "$AUGMENTED_PROMPT_FILE" "$PROMPT_FILE"
        
        AUGMENTED_SIZE=$(wc -c < "$PROMPT_FILE")
        echo "   Augmented prompt size: $AUGMENTED_SIZE bytes (original: $(wc -c < "$ORIGINAL_PROMPT_FILE") bytes)"
        echo "   Saved to: $PROMPT_FILE"
    fi
    
    echo ""
done

# ============================================================================
# Completion
# ============================================================================

echo ""
echo "========================================"
echo "✅ All iterations completed successfully"
echo "========================================"
echo ""
echo "📊 Summary:"
echo "   Total iterations: $ITERATIONS"
echo "   Results directory: $ITERATION_RESULTS_DIR"
echo "   Combined log: $COMBINED_LOG"
echo ""

# The final output is already in the log file from the last iteration
# Copy it to the main log location if needed
if [ -f "$ITERATION_LOG" ] && [ "$ITERATION_LOG" != "$LOG_FILE" ]; then
    echo "📄 Copying final iteration log to: $LOG_FILE"
    cp "$ITERATION_LOG" "$LOG_FILE"
fi

echo "✨ Iteration loop completed successfully"
exit 0
