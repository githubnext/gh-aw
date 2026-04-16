package workflow

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

var safeOutputsMaxValidationLog = newValidationLogger("safe_outputs_max")

// isInvalidMaxValue returns true if n is not a valid max field value.
// Valid values are positive integers (n > 0) or -1 (unlimited).
// Invalid values are 0 and negative integers except -1.
func isInvalidMaxValue(n int) bool {
	if n == -1 {
		return false // -1 = unlimited, explicitly allowed by spec
	}
	return n <= 0
}

// maxInvalidErrSuffix is the common suffix of max validation error messages.
const maxInvalidErrSuffix = "\n\nThe max field controls how many times this safe output can be triggered.\nProvide a positive integer (e.g., max: 1 or max: 5) or -1 for unlimited"

// validateSafeOutputsMax validates that all max fields in safe-outputs configs hold valid values.
// Valid values are positive integers (n > 0) or -1 (unlimited per spec).
// 0 and other negative values are rejected.
// GitHub Actions expressions (e.g. "${{ inputs.max }}") are not evaluable at compile time
// and are therefore skipped.
func validateSafeOutputsMax(config *SafeOutputsConfig) error {
	if config == nil {
		return nil
	}

	safeOutputsMaxValidationLog.Print("Validating safe-outputs max fields")

	val := reflect.ValueOf(config).Elem()

	// Iterate over sorted field names for deterministic error reporting.
	sortedFieldNames := make([]string, 0, len(safeOutputFieldMapping))
	for fieldName := range safeOutputFieldMapping {
		sortedFieldNames = append(sortedFieldNames, fieldName)
	}
	sort.Strings(sortedFieldNames)

	// Validate max on all named safe output fields that embed BaseSafeOutputConfig
	for _, fieldName := range sortedFieldNames {
		toolName := safeOutputFieldMapping[fieldName]
		field := val.FieldByName(fieldName)
		if !field.IsValid() || field.IsNil() {
			continue
		}

		elem := field.Elem()
		baseCfgField := elem.FieldByName("BaseSafeOutputConfig")
		if !baseCfgField.IsValid() {
			continue
		}

		maxField := baseCfgField.FieldByName("Max")
		if !maxField.IsValid() || maxField.IsNil() {
			continue
		}

		maxPtr, ok := maxField.Interface().(*string)
		if !ok || maxPtr == nil || isExpressionString(*maxPtr) {
			continue
		}

		n, err := strconv.Atoi(*maxPtr)
		if err != nil {
			continue
		}

		if isInvalidMaxValue(n) {
			toolDisplayName := strings.ReplaceAll(toolName, "_", "-")
			safeOutputsMaxValidationLog.Printf("Invalid max value %d for %s", n, toolDisplayName)
			return fmt.Errorf(
				"safe-outputs.%s: max must be a positive integer or -1 (unlimited), got %d%s",
				toolDisplayName, n, maxInvalidErrSuffix,
			)
		}
	}

	// Validate max on dispatch_repository tools (different structure: map of tools).
	// Use sorted tool names for deterministic error reporting.
	if config.DispatchRepository != nil {
		sortedToolNames := make([]string, 0, len(config.DispatchRepository.Tools))
		for toolName := range config.DispatchRepository.Tools {
			sortedToolNames = append(sortedToolNames, toolName)
		}
		sort.Strings(sortedToolNames)

		for _, toolName := range sortedToolNames {
			tool := config.DispatchRepository.Tools[toolName]
			if tool == nil || tool.Max == nil || isExpressionString(*tool.Max) {
				continue
			}

			n, err := strconv.Atoi(*tool.Max)
			if err != nil {
				continue
			}

			if isInvalidMaxValue(n) {
				safeOutputsMaxValidationLog.Printf("Invalid max value %d for dispatch_repository tool %s", n, toolName)
				return fmt.Errorf(
					"safe-outputs.dispatch_repository.%s: max must be a positive integer or -1 (unlimited), got %d%s",
					toolName, n, maxInvalidErrSuffix,
				)
			}
		}
	}

	safeOutputsMaxValidationLog.Print("Safe-outputs max fields validation passed")
	return nil
}
