package workflow

import (
	"fmt"
	"strings"
	"time"
)

// ValidationError represents an error that occurred during input validation
type ValidationError struct {
	Field      string
	Value      string
	Reason     string
	Suggestion string
	LearnMore  string // Optional documentation link
	Timestamp  time.Time
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	var b strings.Builder

	fmt.Fprintf(&b, "❌ Validation failed for '%s'", e.Field)

	if e.Value != "" {
		// Truncate long values
		truncatedValue := e.Value
		if len(truncatedValue) > 100 {
			truncatedValue = truncatedValue[:97] + "..."
		}
		fmt.Fprintf(&b, "\n\n📝 Value provided: %s", truncatedValue)
	}

	fmt.Fprintf(&b, "\n\n💡 What went wrong: %s", e.Reason)

	if e.Suggestion != "" {
		fmt.Fprintf(&b, "\n\n✅ How to fix: %s", e.Suggestion)
	}

	if e.LearnMore != "" {
		fmt.Fprintf(&b, "\n\n📚 Learn more: %s", e.LearnMore)
	}

	return b.String()
}

// NewValidationError creates a new validation error with context
func NewValidationError(field, value, reason, suggestion string) *ValidationError {
	return &ValidationError{
		Field:      field,
		Value:      value,
		Reason:     reason,
		Suggestion: suggestion,
		Timestamp:  time.Now(),
	}
}

// NewValidationErrorWithLearnMore creates a new validation error with documentation link
func NewValidationErrorWithLearnMore(field, value, reason, suggestion, learnMore string) *ValidationError {
	return &ValidationError{
		Field:      field,
		Value:      value,
		Reason:     reason,
		Suggestion: suggestion,
		LearnMore:  learnMore,
		Timestamp:  time.Now(),
	}
}

// OperationError represents an error that occurred during an operation
type OperationError struct {
	Operation  string
	EntityType string
	EntityID   string
	Cause      error
	Suggestion string
	LearnMore  string // Optional documentation link
	Timestamp  time.Time
}

// Error implements the error interface
func (e *OperationError) Error() string {
	var b strings.Builder

	fmt.Fprintf(&b, "❌ Failed to %s %s", e.Operation, e.EntityType)

	if e.EntityID != "" {
		fmt.Fprintf(&b, " #%s", e.EntityID)
	}

	if e.Cause != nil {
		fmt.Fprintf(&b, "\n\n⚠️  Underlying error: %v", e.Cause)
	}

	if e.Suggestion != "" {
		fmt.Fprintf(&b, "\n\n✅ How to fix: %s", e.Suggestion)
	} else {
		// Provide default suggestion
		fmt.Fprintf(&b, "\n\n✅ How to fix: Check that the %s exists and you have the necessary permissions.", e.EntityType)
	}

	if e.LearnMore != "" {
		fmt.Fprintf(&b, "\n\n📚 Learn more: %s", e.LearnMore)
	}

	return b.String()
}

// Unwrap returns the underlying error
func (e *OperationError) Unwrap() error {
	return e.Cause
}

// NewOperationError creates a new operation error with context
func NewOperationError(operation, entityType, entityID string, cause error, suggestion string) *OperationError {
	return &OperationError{
		Operation:  operation,
		EntityType: entityType,
		EntityID:   entityID,
		Cause:      cause,
		Suggestion: suggestion,
		Timestamp:  time.Now(),
	}
}

// NewOperationErrorWithLearnMore creates a new operation error with documentation link
func NewOperationErrorWithLearnMore(operation, entityType, entityID string, cause error, suggestion, learnMore string) *OperationError {
	return &OperationError{
		Operation:  operation,
		EntityType: entityType,
		EntityID:   entityID,
		Cause:      cause,
		Suggestion: suggestion,
		LearnMore:  learnMore,
		Timestamp:  time.Now(),
	}
}

// ConfigurationError represents an error in safe-outputs configuration
type ConfigurationError struct {
	ConfigKey  string
	Value      string
	Reason     string
	Suggestion string
	LearnMore  string // Optional documentation link
	Timestamp  time.Time
}

// Error implements the error interface
func (e *ConfigurationError) Error() string {
	var b strings.Builder

	fmt.Fprintf(&b, "⚙️  Configuration error in '%s'", e.ConfigKey)

	if e.Value != "" {
		// Truncate long values
		truncatedValue := e.Value
		if len(truncatedValue) > 100 {
			truncatedValue = truncatedValue[:97] + "..."
		}
		fmt.Fprintf(&b, "\n\n📝 Value provided: %s", truncatedValue)
	}

	fmt.Fprintf(&b, "\n\n💡 What went wrong: %s", e.Reason)

	if e.Suggestion != "" {
		fmt.Fprintf(&b, "\n\n✅ How to fix: %s", e.Suggestion)
	} else {
		// Provide default suggestion
		fmt.Fprintf(&b, "\n\n✅ How to fix: Check the safe-outputs configuration in your workflow frontmatter and ensure '%s' is correctly specified.", e.ConfigKey)
	}

	if e.LearnMore != "" {
		fmt.Fprintf(&b, "\n\n📚 Learn more: %s", e.LearnMore)
	}

	return b.String()
}

// NewConfigurationError creates a new configuration error with context
func NewConfigurationError(configKey, value, reason, suggestion string) *ConfigurationError {
	return &ConfigurationError{
		ConfigKey:  configKey,
		Value:      value,
		Reason:     reason,
		Suggestion: suggestion,
		Timestamp:  time.Now(),
	}
}

// NewConfigurationErrorWithLearnMore creates a new configuration error with documentation link
func NewConfigurationErrorWithLearnMore(configKey, value, reason, suggestion, learnMore string) *ConfigurationError {
	return &ConfigurationError{
		ConfigKey:  configKey,
		Value:      value,
		Reason:     reason,
		Suggestion: suggestion,
		LearnMore:  learnMore,
		Timestamp:  time.Now(),
	}
}

// EnhanceError adds context to an existing error
func EnhanceError(err error, context, suggestion string) error {
	if err == nil {
		return nil
	}

	timestamp := time.Now().Format(time.RFC3339)

	var b strings.Builder
	fmt.Fprintf(&b, "[%s] %s", timestamp, context)
	fmt.Fprintf(&b, "\n\nOriginal error: %v", err)

	if suggestion != "" {
		fmt.Fprintf(&b, "\nSuggestion: %s", suggestion)
	}

	return fmt.Errorf("%s", b.String())
}

// WrapErrorWithContext wraps an error with additional context using fmt.Errorf %w
// This preserves error unwrapping while adding context
func WrapErrorWithContext(err error, context, suggestion string) error {
	if err == nil {
		return nil
	}

	timestamp := time.Now().Format(time.RFC3339)

	if suggestion != "" {
		return fmt.Errorf("[%s] %s (suggestion: %s): %w", timestamp, context, suggestion, err)
	}

	return fmt.Errorf("[%s] %s: %w", timestamp, context, err)
}

// ValidateRequired validates that a required field is not empty
func ValidateRequired(field, value string) error {
	if strings.TrimSpace(value) == "" {
		return NewValidationError(
			field,
			value,
			"field is required and cannot be empty",
			fmt.Sprintf("Provide a non-empty value for '%s'", field),
		)
	}
	return nil
}

// ValidateMaxLength validates that a field does not exceed maximum length
func ValidateMaxLength(field, value string, maxLength int) error {
	if len(value) > maxLength {
		return NewValidationError(
			field,
			value,
			fmt.Sprintf("field exceeds maximum length of %d characters (actual: %d)", maxLength, len(value)),
			fmt.Sprintf("Shorten '%s' to %d characters or less", field, maxLength),
		)
	}
	return nil
}

// ValidateMinLength validates that a field meets minimum length requirement
func ValidateMinLength(field, value string, minLength int) error {
	if len(value) < minLength {
		return NewValidationError(
			field,
			value,
			fmt.Sprintf("field is shorter than minimum length of %d characters (actual: %d)", minLength, len(value)),
			fmt.Sprintf("Ensure '%s' is at least %d characters long", field, minLength),
		)
	}
	return nil
}

// ValidateInList validates that a value is in an allowed list
func ValidateInList(field, value string, allowedValues []string) error {
	for _, allowed := range allowedValues {
		if value == allowed {
			return nil
		}
	}

	return NewValidationError(
		field,
		value,
		fmt.Sprintf("value is not in allowed list: %v", allowedValues),
		fmt.Sprintf("Choose one of the allowed values for '%s': %s", field, strings.Join(allowedValues, ", ")),
	)
}

// ValidatePositiveInt validates that a value is a positive integer
func ValidatePositiveInt(field string, value int) error {
	if value <= 0 {
		return NewValidationError(
			field,
			fmt.Sprintf("%d", value),
			"value must be a positive integer",
			fmt.Sprintf("Provide a positive integer value for '%s'", field),
		)
	}
	return nil
}

// ValidateNonNegativeInt validates that a value is a non-negative integer
func ValidateNonNegativeInt(field string, value int) error {
	if value < 0 {
		return NewValidationError(
			field,
			fmt.Sprintf("%d", value),
			"value must be a non-negative integer",
			fmt.Sprintf("Provide a non-negative integer value for '%s'", field),
		)
	}
	return nil
}
