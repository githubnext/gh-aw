package workflow

import "github.com/github/gh-aw/pkg/logger"

// CreateParseOptions defines common preprocessing options for create-entity parsers.
type CreateParseOptions struct {
	BoolFields    []string
	IntFields     []string
	HandleExpires bool
}

// parseCreateEntityConfig parses create-* config scaffolding shared by issue/discussion/PR handlers.
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
