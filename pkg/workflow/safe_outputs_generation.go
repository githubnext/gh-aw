// Package workflow - safe outputs generation has been refactored into focused modules:
//   - safe_outputs_config_generation.go: main config generation and helpers
//   - safe_outputs_state.go: state inspection and validation (reflection-based)
//   - safe_outputs_runtime.go: runner/workspace configuration
//   - safe_outputs_tools_filtering.go: tool enumeration and filtering
//   - safe_outputs_dispatch.go: dispatch-workflow file mapping
package workflow
