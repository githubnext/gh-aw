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
// Callback lifecycle:
//   - preUnmarshal is invoked first with the raw config map; if it returns false, parsing is aborted.
//   - onError is invoked when YAML unmarshaling fails and returns the fallback config behavior.
//   - postUnmarshal is invoked after successful unmarshaling and receives expiresDisabled,
//     which is true when expires was explicitly set to false.
func parseCreateEntityConfig[T any](
	outputMap map[string]any,
	configKey string,
	opts CreateParseOptions,
	log *logger.Logger,
	onError func(error) *T,
	preUnmarshal func(map[string]any) bool,
	postUnmarshal func(map[string]any, *T, bool),
) *T {
	if _, exists := outputMap[configKey]; !exists {
		return nil
	}

	configData, _ := outputMap[configKey].(map[string]any)
	if preUnmarshal != nil && !preUnmarshal(configData) {
		return nil
	}

	expiresDisabled := false
	if opts.HandleExpires {
		expiresDisabled = preprocessExpiresField(configData, log)
	}

	for _, field := range opts.BoolFields {
		if err := preprocessBoolFieldAsString(configData, field, log); err != nil {
			log.Printf("Invalid %s value: %v", field, err)
			return nil
		}
	}

	for _, field := range opts.IntFields {
		if err := preprocessIntFieldAsString(configData, field, log); err != nil {
			log.Printf("Invalid %s value: %v", field, err)
			return nil
		}
	}

	config := parseConfigScaffold(outputMap, configKey, log, onError)
	if config == nil {
		return nil
	}

	if postUnmarshal != nil {
		postUnmarshal(configData, config, expiresDisabled)
	}

	return config
}
