package campaign

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

//go:embed schemas/project_update_schema.json
var projectUpdateSchemaFS embed.FS

var (
	compiledProjectUpdateSchemaOnce sync.Once
	compiledProjectUpdateSchema     *jsonschema.Schema
	projectUpdateSchemaCompileError error
)

func getCompiledProjectUpdateSchema() (*jsonschema.Schema, error) {
	compiledProjectUpdateSchemaOnce.Do(func() {
		schemaData, err := projectUpdateSchemaFS.ReadFile("schemas/project_update_schema.json")
		if err != nil {
			projectUpdateSchemaCompileError = fmt.Errorf("failed to load project update schema: %w", err)
			return
		}

		var schemaDoc any
		if err := json.Unmarshal(schemaData, &schemaDoc); err != nil {
			projectUpdateSchemaCompileError = fmt.Errorf("failed to parse project update schema: %w", err)
			return
		}

		compiler := jsonschema.NewCompiler()
		schemaURL := "project-update.json"
		if err := compiler.AddResource(schemaURL, schemaDoc); err != nil {
			projectUpdateSchemaCompileError = fmt.Errorf("failed to add project update schema resource: %w", err)
			return
		}

		schema, err := compiler.Compile(schemaURL)
		if err != nil {
			projectUpdateSchemaCompileError = fmt.Errorf("failed to compile project update schema: %w", err)
			return
		}

		compiledProjectUpdateSchema = schema
	})

	return compiledProjectUpdateSchema, projectUpdateSchemaCompileError
}

// ValidateProjectUpdatePayload enforces the governed update-project contract.
//
// This validator is intentionally deterministic:
// - JSON Schema enforces payload shape.
// - Semantic checks enforce equality constraints that JSON Schema cannot.
//
// payload is expected to be a YAML/JSON-decoded structure (map[string]any, etc.).
func ValidateProjectUpdatePayload(payload any, expectedProjectURL string, expectedCampaignID string) []string {
	schema, err := getCompiledProjectUpdateSchema()
	if err != nil {
		return []string{err.Error()}
	}

	normalized := normalizeYAMLValue(payload)
	if err := schema.Validate(normalized); err != nil {
		return formatValidationErrors(err)
	}

	var problems []string
	root, ok := normalized.(map[string]any)
	if !ok {
		return []string{"root: payload must be an object"}
	}

	if expectedProjectURL != "" {
		if project, _ := root["project"].(string); project != expectedProjectURL {
			problems = append(problems, fmt.Sprintf("project: must equal %q", expectedProjectURL))
		}
	}

	if expectedCampaignID != "" {
		if campaignID, _ := root["campaign_id"].(string); campaignID != expectedCampaignID {
			problems = append(problems, fmt.Sprintf("campaign_id: must equal %q", expectedCampaignID))
		}
	}

	fields, _ := root["fields"].(map[string]any)
	if expectedCampaignID != "" {
		if backfillCampaignID, ok := fields["campaign_id"].(string); ok && backfillCampaignID != expectedCampaignID {
			problems = append(problems, fmt.Sprintf("fields.campaign_id: must equal %q", expectedCampaignID))
		}
	}

	// Deterministic normalization: for status-only updates, never allow other writes.
	if len(fields) > 1 {
		// Schema already enforces the full-backfill set, but we keep a stable message for checklist enforcement.
		requiredKeys := []string{"campaign_id", "worker_workflow", "repository", "priority", "size", "start_date", "end_date"}
		var missing []string
		for _, key := range requiredKeys {
			if _, ok := fields[key]; !ok {
				missing = append(missing, key)
			}
		}
		if len(missing) > 0 {
			problems = append(problems, fmt.Sprintf("fields: missing required backfill keys: %s", strings.Join(missing, ", ")))
		}
	}

	return problems
}

// normalizeYAMLValue converts map[any]any (from YAML decoding) into JSON-friendly structures.
func normalizeYAMLValue(value any) any {
	switch v := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, child := range v {
			out[key] = normalizeYAMLValue(child)
		}
		return out
	case map[any]any:
		out := make(map[string]any, len(v))
		for key, child := range v {
			keyStr, ok := key.(string)
			if !ok {
				keyStr = fmt.Sprintf("%v", key)
			}
			out[keyStr] = normalizeYAMLValue(child)
		}
		return out
	case []any:
		out := make([]any, 0, len(v))
		for _, child := range v {
			out = append(out, normalizeYAMLValue(child))
		}
		return out
	default:
		return value
	}
}
