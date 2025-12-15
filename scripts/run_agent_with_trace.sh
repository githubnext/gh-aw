#!/usr/bin/env bash
set -euo pipefail

# Agent execution wrapper with checkpoint capture
# This script wraps agent execution and records checkpoints for replay

TRACE_DIR="${TRACE_DIR:-trace}"
REPLAY_RUN_ID="${REPLAY_RUN_ID:-}"
START_CHECKPOINT="${START_CHECKPOINT:-}"
TOOL_MODE="${TOOL_MODE:-cached}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}" >&2
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}" >&2
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}" >&2
}

log_error() {
    echo -e "${RED}❌ $1${NC}" >&2
}

# Initialize trace directory structure
init_trace_dir() {
    log_info "Initializing trace directory: $TRACE_DIR"
    mkdir -p "$TRACE_DIR"/{diffs,tools,summaries}
    
    # Initialize manifest.json
    cat > "$TRACE_DIR/manifest.json" <<EOF
{
  "run_id": "${GITHUB_RUN_ID:-local}",
  "workflow": "${GITHUB_WORKFLOW:-local-test}",
  "engine": "${ENGINE:-copilot}",
  "repo_sha": "$(git rev-parse HEAD 2>/dev/null || echo 'unknown')",
  "created_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "trace_version": "v1"
}
EOF
    
    # Initialize empty checkpoints.jsonl
    : > "$TRACE_DIR/checkpoints.jsonl"
    
    log_success "Trace directory initialized"
}

# Record a checkpoint
record_checkpoint() {
    local kind="$1"
    local name="$2"
    local checkpoint_id="$3"
    local extra_metadata="${4:-{}}"
    
    local repo_sha
    repo_sha="$(git rev-parse HEAD 2>/dev/null || echo 'unknown')"
    
    local checkpoint
    checkpoint=$(cat <<EOF
{
  "id": "$checkpoint_id",
  "ts": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "kind": "$kind",
  "name": "$name",
  "repo_sha": "$repo_sha",
  "metadata": $extra_metadata
}
EOF
)
    
    echo "$checkpoint" >> "$TRACE_DIR/checkpoints.jsonl"
    log_info "Recorded checkpoint: $checkpoint_id ($kind: $name)"
}

# Check if we're in replay mode
is_replay_mode() {
    [[ -n "$REPLAY_RUN_ID" ]]
}

# Download trace from previous run
download_replay_trace() {
    log_info "Downloading trace from run: $REPLAY_RUN_ID"
    
    # Use GitHub CLI to download artifacts
    gh run download "$REPLAY_RUN_ID" -n trace -D replay_in 2>/dev/null || {
        log_error "Failed to download trace artifact from run $REPLAY_RUN_ID"
        return 1
    }
    
    # Copy prior trace into current trace folder
    cp -r replay_in/trace/* "$TRACE_DIR/" 2>/dev/null || {
        log_error "Failed to copy replay trace"
        return 1
    }
    
    log_success "Downloaded replay trace"
}

# Restore repository to checkpoint state
restore_repo_state() {
    local target_sha
    target_sha=$(python3 -c "import json; print(json.load(open('$TRACE_DIR/manifest.json'))['repo_sha'])" 2>/dev/null || echo "")
    
    if [[ -z "$target_sha" ]]; then
        log_error "Failed to extract repo SHA from manifest"
        return 1
    fi
    
    log_info "Restoring repo to SHA: $target_sha"
    git checkout "$target_sha" 2>/dev/null || {
        log_error "Failed to checkout SHA: $target_sha"
        return 1
    }
    
    log_success "Repository restored to checkpoint state"
}

# Check if we should skip to a specific checkpoint
should_skip_checkpoint() {
    local current_checkpoint="$1"
    
    if [[ -z "$START_CHECKPOINT" ]]; then
        return 1 # Don't skip, no start checkpoint specified
    fi
    
    # Extract numeric part for comparison (c003 -> 3)
    local current_num="${current_checkpoint#c}"
    local start_num="${START_CHECKPOINT#c}"
    
    # Remove leading zeros
    current_num=$((10#$current_num))
    start_num=$((10#$start_num))
    
    [[ $current_num -lt $start_num ]]
}

# Load cached tool response (replay mode)
load_cached_tool_response() {
    local checkpoint_id="$1"
    local response_file="$TRACE_DIR/tools/${checkpoint_id}.response.json"
    
    if [[ -f "$response_file" ]]; then
        cat "$response_file"
        return 0
    fi
    
    log_warning "No cached response found for checkpoint: $checkpoint_id"
    return 1
}

# Example: Wrap a tool call with checkpoint capture
wrap_tool_call() {
    local tool_name="$1"
    local operation="$2"
    shift 2
    local args=("$@")
    
    local checkpoint_id
    checkpoint_id=$(printf "c%03d" "$CHECKPOINT_COUNTER")
    CHECKPOINT_COUNTER=$((CHECKPOINT_COUNTER + 1))
    
    # Check if we should skip this checkpoint (replay mode)
    if is_replay_mode && should_skip_checkpoint "$checkpoint_id"; then
        log_info "Skipping checkpoint: $checkpoint_id (replay mode)"
        
        # Try to load cached response
        if [[ "$TOOL_MODE" == "cached" ]]; then
            if load_cached_tool_response "$checkpoint_id"; then
                record_checkpoint "tool_call" "${tool_name}.${operation}" "$checkpoint_id" '{"replayed":true}'
                return 0
            fi
        fi
    fi
    
    # Execute the actual tool call
    local start_time
    start_time=$(date +%s%N)
    
    local request_file="$TRACE_DIR/tools/${checkpoint_id}.request.json"
    local response_file="$TRACE_DIR/tools/${checkpoint_id}.response.json"
    
    # Record request
    cat > "$request_file" <<EOF
{
  "tool": "$tool_name",
  "operation": "$operation",
  "arguments": $(printf '%s\n' "${args[@]}" | jq -R . | jq -s .),
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
EOF
    
    # Execute tool (this is where you'd call the actual tool)
    local exit_code=0
    local output
    output=$(execute_tool "$tool_name" "$operation" "${args[@]}" 2>&1) || exit_code=$?
    
    local end_time
    end_time=$(date +%s%N)
    local duration=$((($end_time - $start_time) / 1000000)) # milliseconds
    
    # Record response
    cat > "$response_file" <<EOF
{
  "tool": "$tool_name",
  "operation": "$operation",
  "success": $([ $exit_code -eq 0 ] && echo "true" || echo "false"),
  "data": $(echo "$output" | jq -Rs .),
  "duration": "${duration}ms",
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
EOF
    
    # Record checkpoint
    record_checkpoint "tool_call" "${tool_name}.${operation}" "$checkpoint_id" "{\"duration\":\"${duration}ms\",\"success\":$([ $exit_code -eq 0 ] && echo "true" || echo "false")}"
    
    echo "$output"
    return $exit_code
}

# Placeholder for actual tool execution (to be implemented per engine)
execute_tool() {
    local tool_name="$1"
    local operation="$2"
    shift 2
    
    # This is where you'd integrate with MCP servers, GitHub CLI, etc.
    log_info "Executing tool: ${tool_name}.${operation}"
    echo '{"status": "ok", "message": "Tool execution placeholder"}'
}

# Main execution
main() {
    log_info "Starting agent execution with trace capture"
    
    # Global checkpoint counter
    export CHECKPOINT_COUNTER=0
    
    # Initialize trace directory
    init_trace_dir
    
    # Handle replay mode
    if is_replay_mode; then
        log_info "Replay mode enabled (run_id: $REPLAY_RUN_ID, start: $START_CHECKPOINT)"
        download_replay_trace || exit 1
        restore_repo_state || exit 1
    fi
    
    # This is where you'd call the actual agent
    # For now, we'll just demonstrate with a simple example
    log_info "Executing agent workflow..."
    
    # Example checkpoint captures:
    # - Tool call
    # wrap_tool_call "github" "search" "query=foo"
    # - Patch application would be captured similarly
    # - Eval would be captured similarly
    
    log_success "Agent execution completed"
    
    # Display checkpoint summary
    local checkpoint_count
    checkpoint_count=$(wc -l < "$TRACE_DIR/checkpoints.jsonl" | tr -d ' ')
    log_info "Captured $checkpoint_count checkpoints"
}

# Run main if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
