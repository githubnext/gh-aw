package workflow

import "github.com/github/gh-aw/pkg/logger"

var safeOutputsBuilderLog = logger.New("workflow:safe_outputs_builder")

// handlerConfigBuilder provides a fluent API for building handler configurations
type handlerConfigBuilder struct {
	config map[string]any
}

// newHandlerConfigBuilder creates a new handler config builder
func newHandlerConfigBuilder() *handlerConfigBuilder {
	return &handlerConfigBuilder{
		config: map[string]any{},
	}
}

// AddIfPositive adds an integer field only if the value is greater than 0
func (b *handlerConfigBuilder) AddIfPositive(key string, value int) *handlerConfigBuilder {
	if value > 0 {
		b.config[key] = value
	}
	return b
}

// AddIfNotEmpty adds a string field only if the value is not empty
func (b *handlerConfigBuilder) AddIfNotEmpty(key string, value string) *handlerConfigBuilder {
	if value != "" {
		b.config[key] = value
	}
	return b
}

// AddStringSlice adds a string slice field only if the slice is not empty
func (b *handlerConfigBuilder) AddStringSlice(key string, value []string) *handlerConfigBuilder {
	if len(value) > 0 {
		b.config[key] = value
	}
	return b
}

// AddTemplatableStringSlice adds a string slice field that may contain a GitHub Actions
// expression.  When the slice has exactly one element and that element is a GitHub Actions
// expression (as produced by preprocessStringArrayFieldAsTemplatable or
// ParseStringArrayOrExprFromConfig), the expression string is stored as a plain JSON string
// rather than a JSON array.  This allows GitHub Actions to evaluate the expression at
// runtime when the config.json file is written via heredoc expansion.
//
// For all other non-empty slices the field is stored as a JSON array, matching the
// behaviour of AddStringSlice.
func (b *handlerConfigBuilder) AddTemplatableStringSlice(key string, value []string) *handlerConfigBuilder {
	if len(value) == 0 {
		return b
	}
	// A single-element expression slice is the canonical representation produced by
	// preprocessing – store as a string so GitHub Actions evaluates it at runtime.
	if len(value) == 1 && isExpression(value[0]) {
		b.config[key] = value[0]
		return b
	}
	b.config[key] = value
	return b
}

// AddBoolPtr adds a boolean pointer field only if the pointer is not nil
func (b *handlerConfigBuilder) AddBoolPtr(key string, value *bool) *handlerConfigBuilder {
	if value != nil {
		b.config[key] = *value
	}
	return b
}

// AddBoolPtrOrDefault adds a boolean field, using default if pointer is nil
func (b *handlerConfigBuilder) AddBoolPtrOrDefault(key string, value *bool, defaultValue bool) *handlerConfigBuilder {
	if value != nil {
		b.config[key] = *value
	} else {
		b.config[key] = defaultValue
	}
	return b
}

// AddStringPtr adds a string pointer field only if the pointer is not nil
func (b *handlerConfigBuilder) AddStringPtr(key string, value *string) *handlerConfigBuilder {
	if value != nil {
		b.config[key] = *value
	}
	return b
}

// AddDefault adds a field with a default value unconditionally
func (b *handlerConfigBuilder) AddDefault(key string, value any) *handlerConfigBuilder {
	b.config[key] = value
	return b
}

// AddIfTrue adds a boolean field only if the value is true
func (b *handlerConfigBuilder) AddIfTrue(key string, value bool) *handlerConfigBuilder {
	if value {
		b.config[key] = true
	}
	return b
}

// Build returns the built configuration map
func (b *handlerConfigBuilder) Build() map[string]any {
	return b.config
}

// handlerBuilder is a function that builds a handler config from SafeOutputsConfig
type handlerBuilder func(*SafeOutputsConfig) map[string]any

// getEffectiveFooterForTemplatable returns the effective footer as a templatable string.
// If the local string footer is set, use it; otherwise convert the global bool footer.
// Returns nil if neither is set (default to true in JavaScript).
func getEffectiveFooterForTemplatable(localFooter *string, globalFooter *bool) *string {
	if localFooter != nil {
		safeOutputsBuilderLog.Printf("Footer: using local override %q", *localFooter)
		return localFooter
	}
	if globalFooter != nil {
		var s string
		if *globalFooter {
			s = "true"
		} else {
			s = "false"
		}
		safeOutputsBuilderLog.Printf("Footer: derived %q from global bool", s)
		return &s
	}
	safeOutputsBuilderLog.Print("Footer: not configured, deferring to JS default")
	return nil
}

// getEffectiveFooterString returns the effective footer string value for a config.
// If the local string footer is set, use it; otherwise convert the global bool footer.
// Returns nil if neither is set (default to "always" in JavaScript).
func getEffectiveFooterString(localFooter *string, globalFooter *bool) *string {
	if localFooter != nil {
		safeOutputsBuilderLog.Printf("FooterString: using local override %q", *localFooter)
		return localFooter
	}
	if globalFooter != nil {
		var s string
		if *globalFooter {
			s = "always"
		} else {
			s = "none"
		}
		safeOutputsBuilderLog.Printf("FooterString: derived %q from global bool", s)
		return &s
	}
	safeOutputsBuilderLog.Print("FooterString: not configured, deferring to JS default")
	return nil
}
