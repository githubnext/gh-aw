package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/githubnext/gh-aw/internal/mapper"
	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/goccy/go-yaml"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

// httpURLLoader implements URLLoader for HTTP(S) URLs
type httpURLLoader struct {
	client *http.Client
}

// Load implements URLLoader interface for HTTP URLs
func (h *httpURLLoader) Load(url string) (any, error) {
	resp, err := h.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch URL %s: HTTP %d", url, resp.StatusCode)
	}

	var result interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode JSON from %s: %w", url, err)
	}

	return result, nil
}

// validateWorkflowSchema validates the generated YAML content against the GitHub Actions workflow schema
func (c *Compiler) validateWorkflowSchema(yamlContent string) error {
	return c.validateWorkflowSchemaWithFile(yamlContent, "")
}

// validateWorkflowSchemaWithFile validates the generated YAML content against the GitHub Actions workflow schema
// and optionally provides enhanced error reporting with source file locations
func (c *Compiler) validateWorkflowSchemaWithFile(yamlContent, sourceFile string) error {
	// Convert YAML to JSON for validation
	var workflowData interface{}
	if err := yaml.Unmarshal([]byte(yamlContent), &workflowData); err != nil {
		return fmt.Errorf("failed to parse generated YAML: %w", err)
	}

	// Convert to JSON
	jsonData, err := json.Marshal(workflowData)
	if err != nil {
		return fmt.Errorf("failed to convert YAML to JSON: %w", err)
	}

	// Load GitHub Actions workflow schema from SchemaStore
	schemaURL := "https://raw.githubusercontent.com/SchemaStore/schemastore/master/src/schemas/json/github-workflow.json"

	// Create compiler with HTTP loader
	loader := jsonschema.NewCompiler()
	httpLoader := &httpURLLoader{
		client: &http.Client{Timeout: 30 * time.Second},
	}

	// Configure the compiler to use HTTP loader for https and http schemes
	schemeLoader := jsonschema.SchemeURLLoader{
		"https": httpLoader,
		"http":  httpLoader,
	}
	loader.UseLoader(schemeLoader)

	schema, err := loader.Compile(schemaURL)
	if err != nil {
		return fmt.Errorf("failed to load GitHub Actions schema from %s: %w", schemaURL, err)
	}

	// Validate the JSON data against the schema
	var jsonObj interface{}
	if err := json.Unmarshal(jsonData, &jsonObj); err != nil {
		return fmt.Errorf("failed to unmarshal JSON for validation: %w", err)
	}

	if err := schema.Validate(jsonObj); err != nil {
		// Enhanced error reporting with source locations
		return c.formatSchemaValidationError(err, yamlContent, sourceFile)
	}

	return nil
}

// formatSchemaValidationError formats a JSON schema validation error with precise YAML source locations
func (c *Compiler) formatSchemaValidationError(validationErr error, yamlContent, sourceFile string) error {
	// Try to cast to ValidationError for detailed information
	var valErr *jsonschema.ValidationError
	if !errors.As(validationErr, &valErr) {
		// Fallback to generic error if we can't get detailed information
		return fmt.Errorf("workflow schema validation failed: %w", validationErr)
	}

	// Convert InstanceLocation ([]string) to JSON pointer format ("/path/to/field")
	instancePath := "/" + strings.Join(valErr.InstanceLocation, "/")

	// Extract error kind information
	errorMeta := c.extractErrorMeta(valErr)

	// Use mapper to get precise YAML source locations
	spans, err := mapper.MapErrorToSpans([]byte(yamlContent), instancePath, errorMeta)
	if err != nil || len(spans) == 0 {
		// Fallback to generic error if mapping fails
		if sourceFile != "" {
			return fmt.Errorf("workflow schema validation failed at %s: %w", instancePath, validationErr)
		}
		return fmt.Errorf("workflow schema validation failed: %w", validationErr)
	}

	// Use the highest confidence span for error reporting
	bestSpan := spans[0]

	// Create a formatted error with source location
	compilerErr := console.CompilerError{
		Position: console.ErrorPosition{
			File:   sourceFile,
			Line:   bestSpan.StartLine,
			Column: bestSpan.StartCol,
		},
		Type:    "error",
		Message: fmt.Sprintf("schema validation failed: %s", c.formatValidationMessage(valErr, errorMeta)),
		Hint:    c.generateValidationHint(valErr, errorMeta),
	}

	// Add context lines if we can read the source file
	if sourceFile != "" {
		compilerErr.Context = c.extractContextLines(yamlContent, bestSpan.StartLine)
	}

	return errors.New(console.FormatError(compilerErr))
}

// extractErrorMeta converts a jsonschema ValidationError to mapper ErrorMeta
func (c *Compiler) extractErrorMeta(valErr *jsonschema.ValidationError) mapper.ErrorMeta {
	// Extract the primary error kind
	errorKind := "unknown"
	property := ""

	if valErr.ErrorKind != nil {
		// Get the keyword path to determine error type
		keywordPath := valErr.ErrorKind.KeywordPath()
		if len(keywordPath) > 0 {
			errorKind = keywordPath[len(keywordPath)-1] // Last keyword is usually the most specific
		}
	}

	// Try to extract property name from instance location for certain error types
	if len(valErr.InstanceLocation) > 0 {
		property = valErr.InstanceLocation[len(valErr.InstanceLocation)-1]
	}

	return mapper.ErrorMeta{
		Kind:     errorKind,
		Property: property,
	}
}

// formatValidationMessage creates a human-readable validation error message
func (c *Compiler) formatValidationMessage(valErr *jsonschema.ValidationError, meta mapper.ErrorMeta) string {
	baseMsg := valErr.Error()

	// Try to make the message more user-friendly based on error kind
	switch meta.Kind {
	case "type":
		return fmt.Sprintf("type mismatch at '%s'", strings.Join(valErr.InstanceLocation, "."))
	case "required":
		return fmt.Sprintf("missing required property '%s'", meta.Property)
	case "additionalProperties":
		return fmt.Sprintf("unexpected property '%s'", meta.Property)
	case "enum":
		return fmt.Sprintf("value not allowed at '%s'", strings.Join(valErr.InstanceLocation, "."))
	default:
		// Extract meaningful part from the original error message
		if idx := strings.Index(baseMsg, ":"); idx > 0 && idx < len(baseMsg)-1 {
			return strings.TrimSpace(baseMsg[idx+1:])
		}
		return baseMsg
	}
}

// generateValidationHint provides helpful hints for common validation errors
func (c *Compiler) generateValidationHint(valErr *jsonschema.ValidationError, meta mapper.ErrorMeta) string {
	switch meta.Kind {
	case "type":
		return "Check the data type - ensure strings are quoted, numbers are unquoted, etc."
	case "required":
		return fmt.Sprintf("Add the required property '%s' to this object", meta.Property)
	case "additionalProperties":
		return fmt.Sprintf("Remove the property '%s' or check for typos in property names", meta.Property)
	case "enum":
		return "Use one of the allowed values defined in the schema"
	default:
		return "Check the GitHub Actions workflow schema documentation"
	}
}

// extractContextLines extracts context lines around the error position
func (c *Compiler) extractContextLines(yamlContent string, errorLine int) []string {
	lines := strings.Split(yamlContent, "\n")

	// Extract 3 lines of context (1 before, error line, 1 after)
	start := errorLine - 2 // -1 for 0-based indexing, -1 for 1 line before
	if start < 0 {
		start = 0
	}

	end := errorLine + 1 // -1 for 0-based indexing, +1 for 1 line after
	if end >= len(lines) {
		end = len(lines) - 1
	}

	var context []string
	for i := start; i <= end; i++ {
		if i >= 0 && i < len(lines) {
			context = append(context, lines[i])
		}
	}

	return context
}
