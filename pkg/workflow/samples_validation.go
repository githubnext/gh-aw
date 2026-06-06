package workflow

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// sampleSidecarFields lists fields recognized inside a `samples` entry
// that are NOT passed to the MCP tool's `tools/call` arguments. They are stripped
// from the sample before schema validation and consumed by the replay driver
// (e.g. to pre-stage a branch + patch on disk).
var sampleSidecarFields = map[string]map[string]bool{
	"create_pull_request": {
		"patch": true,
	},
	"push_to_pull_request_branch": {
		"patch": true,
	},
}

// compiledToolSchemas caches the per-tool jsonschema.Schema parsed from the
// embedded safe_outputs_tools.json. Compiled lazily on first use.
var (
	compiledToolSchemasOnce sync.Once
	compiledToolSchemas     map[string]*jsonschema.Schema
	compiledToolSchemasErr  error
)

func getCompiledToolSchemas() (map[string]*jsonschema.Schema, error) {
	compiledToolSchemasOnce.Do(func() {
		var tools []struct {
			Name        string          `json:"name"`
			InputSchema json.RawMessage `json:"inputSchema"`
		}
		if err := json.Unmarshal([]byte(safeOutputsToolsJSONContent), &tools); err != nil {
			compiledToolSchemasErr = fmt.Errorf("failed to parse safe_outputs_tools.json for samples validation: %w", err)
			return
		}
		out := make(map[string]*jsonschema.Schema, len(tools))
		for _, t := range tools {
			if len(t.InputSchema) == 0 {
				continue
			}
			var schemaDoc any
			if err := json.Unmarshal(t.InputSchema, &schemaDoc); err != nil {
				compiledToolSchemasErr = fmt.Errorf("failed to parse inputSchema for tool %q: %w", t.Name, err)
				return
			}
			compiler := jsonschema.NewCompiler()
			schemaURL := fmt.Sprintf("inmem://safe-outputs-tools/%s.json", t.Name)
			if err := compiler.AddResource(schemaURL, schemaDoc); err != nil {
				compiledToolSchemasErr = fmt.Errorf("failed to add schema resource for tool %q: %w", t.Name, err)
				return
			}
			schema, err := compiler.Compile(schemaURL)
			if err != nil {
				compiledToolSchemasErr = fmt.Errorf("failed to compile inputSchema for tool %q: %w", t.Name, err)
				return
			}
			out[t.Name] = schema
		}
		compiledToolSchemas = out
	})
	return compiledToolSchemas, compiledToolSchemasErr
}

// validateSafeOutputsSamples validates every `samples` entry on every
// enabled safe-output handler against the corresponding MCP tool's inputSchema.
// Sample sidecar fields (e.g. `patch`) are stripped before validation. Returns
// the first error encountered; iteration order is deterministic (sorted by
// struct field name) so error messages are stable.
func validateSafeOutputsSamples(config *SafeOutputsConfig) error {
	if config == nil {
		return nil
	}

	fieldNames := make([]string, 0, len(safeOutputFieldMapping))
	for fieldName := range safeOutputFieldMapping {
		fieldNames = append(fieldNames, fieldName)
	}
	sort.Strings(fieldNames)

	for _, fieldName := range fieldNames {
		toolName := safeOutputFieldMapping[fieldName]
		base := extractBaseSafeOutputConfig(config, fieldName)
		if base == nil || len(base.Samples) == 0 {
			continue
		}
		if err := validateSamplesForTool(toolName, base.Samples); err != nil {
			return err
		}
	}
	return nil
}

// extractBaseSafeOutputConfig returns the embedded BaseSafeOutputConfig of the
// non-nil safe-output config at SafeOutputsConfig.<fieldName>, or nil if the
// field is unset or the struct does not embed BaseSafeOutputConfig.
func extractBaseSafeOutputConfig(config *SafeOutputsConfig, fieldName string) *BaseSafeOutputConfig {
	field, ok := safeOutputPointerFieldValue(config, fieldName)
	if !ok || field.IsNil() {
		return nil
	}
	elem := field.Elem()
	if elem.Kind() != reflect.Struct {
		return nil
	}
	baseField := elem.FieldByName("BaseSafeOutputConfig")
	if !baseField.IsValid() || !baseField.CanAddr() {
		return nil
	}
	if base, ok := baseField.Addr().Interface().(*BaseSafeOutputConfig); ok {
		return base
	}
	return nil
}

// validateSamplesForTool validates each sample against the named MCP tool's
// inputSchema after stripping recognized sidecar fields.
func validateSamplesForTool(toolName string, samples []map[string]any) error {
	schemas, err := getCompiledToolSchemas()
	if err != nil {
		return err
	}
	schema, found := schemas[toolName]
	if !found {
		return fmt.Errorf("samples: no MCP tool schema found for %q (yaml key %q). Available tools come from pkg/workflow/js/safe_outputs_tools.json", toolName, toolDisplayKey(toolName))
	}
	displayKey := toolDisplayKey(toolName)
	sidecars := sampleSidecarFields[toolName]
	for i, sample := range samples {
		stripped := stripSidecarFields(sample, sidecars)
		if err := schema.Validate(stripped); err != nil {
			return fmt.Errorf("safe-outputs.%s.samples[%d]: %w", displayKey, i, err)
		}
	}
	return nil
}

// stripSidecarFields returns a shallow copy of sample with sidecar keys removed.
// The original map is never modified, even when no sidecars are configured —
// callers may mutate the returned map without affecting the caller's input.
func stripSidecarFields(sample map[string]any, sidecars map[string]bool) map[string]any {
	out := make(map[string]any, len(sample))
	for k, v := range sample {
		if sidecars[k] {
			continue
		}
		out[k] = v
	}
	return out
}

// toolDisplayKey converts a snake_case MCP tool name into the hyphenated YAML
// frontmatter key (e.g. "create_pull_request" -> "create-pull-request").
func toolDisplayKey(toolName string) string {
	return strings.ReplaceAll(toolName, "_", "-")
}
