package workflow

import "github.com/github/gh-aw/pkg/logger"

// CreateParseOptions defines common preprocessing options for create-entity parsers.
//
// BoolFields and IntFields list config field names that should be normalized through
// templatable preprocessing before YAML unmarshaling.
// HandleExpires enables shared expires normalization via preprocessExpiresField.
type CreateParseOptions struct {
	BoolFields    []string
	IntFields     []string
	HandleExpires bool
}

// parseCreateEntityConfig parses create-* config scaffolding shared by issue/discussion/PR handlers.
//
// Parameters:
//   - outputMap: full safe-output map from frontmatter parsing.
//   - configKey: create-* key to parse (for example "create-issue").
//   - opts: shared preprocessing configuration for bool/int/expires fields.
//   - debugLog: logger used for preprocessing and parse diagnostics.
//   - onError: required error handler invoked on unmarshal failures.
//
// Callback lifecycle:
//   - preUnmarshal is optional (may be nil). When provided, it is invoked first with the raw
//     config map. The map may be nil when configKey exists but is not a map; if preUnmarshal
//     returns false, parsing is aborted.
//   - onError is invoked when YAML unmarshaling fails and returns the fallback config behavior.
//   - postUnmarshal is optional (may be nil). When provided, it is invoked after successful
//     unmarshaling and receives expiresDisabled (true when expires was explicitly set to false).
func parseCreateEntityConfig[T any](
	outputMap map[string]any,
	configKey string,
	opts CreateParseOptions,
	debugLog *logger.Logger,
	onError func(error) *T,
	preUnmarshal func(map[string]any) bool,
	postUnmarshal func(map[string]any, *T, bool),
) *T {
	if _, exists := outputMap[configKey]; !exists {
		return nil
	}

	configDataAny := outputMap[configKey]
	configData, isMap := configDataAny.(map[string]any)
	if !isMap {
		configData = nil
	}
	if preUnmarshal != nil && !preUnmarshal(configData) {
		return nil
	}

	expiresDisabled := false
	if opts.HandleExpires {
		expiresDisabled = preprocessExpiresField(configData, debugLog)
	}

	for _, field := range opts.BoolFields {
		if err := preprocessBoolFieldAsString(configData, field, debugLog); err != nil {
			debugLog.Printf("Invalid %s value: %v", field, err)
			return nil
		}
	}

	for _, field := range opts.IntFields {
		if err := preprocessIntFieldAsString(configData, field, debugLog); err != nil {
			debugLog.Printf("Invalid %s value: %v", field, err)
			return nil
		}
	}

	config := parseConfigScaffold(outputMap, configKey, debugLog, onError)
	if config == nil {
		return nil
	}

	if postUnmarshal != nil {
		postUnmarshal(configData, config, expiresDisabled)
	}

	return config
}
