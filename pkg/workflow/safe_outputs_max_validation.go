package workflow

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

var safeOutputsMaxValidationLog = newValidationLogger("safe_outputs_max")

// safeOutputMaxEntry caches precomputed reflect field indices for a single SafeOutputsConfig
// pointer field. Using Field(idx) instead of FieldByName(name) replaces an O(n) struct scan
// with an O(1) array index, eliminating the dominant reflection cost in validateSafeOutputsMax.
type safeOutputMaxEntry struct {
	fieldIdx   int    // index of the pointer field in SafeOutputsConfig
	baseCfgIdx int    // index of BaseSafeOutputConfig within the pointed-to type; -1 if absent
	toolName   string // display name (value from safeOutputFieldMapping, e.g. "add_comment")
}

// maxFieldIdxInBase is the index of the Max field within BaseSafeOutputConfig.
// Pre-computed once at init to avoid a repeated FieldByName("Max") call in the hot loop.
var maxFieldIdxInBase = func() int {
	t := reflect.TypeFor[BaseSafeOutputConfig]()
	sf, ok := t.FieldByName("Max")
	if !ok || len(sf.Index) != 1 {
		panic("safe_outputs_max_validation: Max field not found or has unexpected nesting in BaseSafeOutputConfig")
	}
	return sf.Index[0]
}()

// safeOutputMaxEntries is the pre-sorted slice of field descriptors used by validateSafeOutputsMax.
// Sorted by toolName for deterministic error reporting. Computed once at init.
var safeOutputMaxEntries = func() []safeOutputMaxEntry {
	configType := reflect.TypeFor[SafeOutputsConfig]()
	entries := make([]safeOutputMaxEntry, 0, len(safeOutputFieldMapping))
	for fieldName, toolName := range safeOutputFieldMapping {
		sf, ok := configType.FieldByName(fieldName)
		if !ok {
			// safeOutputFieldMapping references a field that does not exist in SafeOutputsConfig.
			// This is a programming error — panic at init so it is caught immediately.
			panic(fmt.Sprintf("safe_outputs_max_validation: field %q from safeOutputFieldMapping not found in SafeOutputsConfig", fieldName))
		}
		if len(sf.Index) != 1 {
			// Unexpectedly promoted / nested field — same programming-error treatment.
			panic(fmt.Sprintf("safe_outputs_max_validation: field %q has unexpected nesting depth %d in SafeOutputsConfig (expected 1)", fieldName, len(sf.Index)))
		}
		fieldIdx := sf.Index[0]

		// Dereference the pointer type to reach the concrete struct type.
		elemType := sf.Type
		if elemType.Kind() == reflect.Ptr {
			elemType = elemType.Elem()
		}

		// Locate BaseSafeOutputConfig within the element type.
		// Fields that don't embed it (e.g. DispatchRepositoryConfig) get baseCfgIdx = -1
		// and are skipped in the hot loop.
		baseCfgIdx := -1
		if baseSF, ok := elemType.FieldByName("BaseSafeOutputConfig"); ok && len(baseSF.Index) == 1 {
			baseCfgIdx = baseSF.Index[0]
		}

		entries = append(entries, safeOutputMaxEntry{
			fieldIdx:   fieldIdx,
			baseCfgIdx: baseCfgIdx,
			toolName:   toolName,
		})
	}
	// Sort by toolName for deterministic error reporting.
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].toolName < entries[j].toolName
	})
	return entries
}()

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

	// Use precomputed field indices (O(1) Field access) instead of FieldByName (O(n) scan).
	// safeOutputMaxEntries is sorted by toolName at init for deterministic error reporting.
	for _, entry := range safeOutputMaxEntries {
		if entry.baseCfgIdx < 0 {
			// This field's type does not embed BaseSafeOutputConfig; skip max validation.
			continue
		}

		field := val.Field(entry.fieldIdx)
		if field.IsNil() {
			continue
		}

		elem := field.Elem()
		baseCfgField := elem.Field(entry.baseCfgIdx)

		maxField := baseCfgField.Field(maxFieldIdxInBase)
		if maxField.IsNil() {
			continue
		}

		maxPtr, ok := maxField.Interface().(*string)
		if !ok || maxPtr == nil || isExpression(*maxPtr) {
			continue
		}

		n, err := strconv.Atoi(*maxPtr)
		if err != nil {
			continue
		}

		if isInvalidMaxValue(n) {
			toolDisplayName := strings.ReplaceAll(entry.toolName, "_", "-")
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
			if tool == nil || tool.Max == nil || isExpression(*tool.Max) {
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
