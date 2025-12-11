package campaign

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/parser"
	"github.com/xeipuuv/gojsonschema"
)

//go:embed schemas/campaign_spec_schema.json
var campaignSpecSchemaFS embed.FS

// ValidateSpec performs lightweight semantic validation of a
// single CampaignSpec and returns a slice of human-readable problems.
//
// It uses JSON schema validation first, then adds additional semantic checks.
func ValidateSpec(spec *CampaignSpec) []string {
	var problems []string

	// First, validate against JSON schema
	schemaProblems := ValidateSpecWithSchema(spec)
	problems = append(problems, schemaProblems...)

	// Additional semantic validation beyond schema
	trimmedID := strings.TrimSpace(spec.ID)
	if trimmedID == "" {
		problems = append(problems, "id is required and must be non-empty")
	} else {
		// Enforce a simple, URL-safe pattern for IDs
		for _, ch := range trimmedID {
			if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-' {
				continue
			}
			problems = append(problems, "id must use only lowercase letters, digits, and hyphens ("+trimmedID+")")
			break
		}
	}

	if strings.TrimSpace(spec.Name) == "" {
		problems = append(problems, "name should be provided (falls back to id, but explicit names are recommended)")
	}

	if len(spec.Workflows) == 0 {
		problems = append(problems, "workflows should list at least one workflow implementing this campaign")
	}

	if strings.TrimSpace(spec.TrackerLabel) == "" {
		problems = append(problems, "tracker-label should be set to link issues and PRs to this campaign")
	} else if !strings.Contains(spec.TrackerLabel, ":") {
		problems = append(problems, "tracker-label should follow a namespaced pattern (for example: campaign:security-q1-2025)")
	}

	// Normalize and validate version/state when present.
	if strings.TrimSpace(spec.Version) == "" {
		// Default version for v1 specs when omitted.
		spec.Version = "v1"
	}

	if spec.State != "" {
		switch spec.State {
		case "planned", "active", "paused", "completed", "archived":
			// valid
		default:
			problems = append(problems, "state must be one of: planned, active, paused, completed, archived")
		}
	}

	return problems
}

// ValidateSpecWithSchema validates a CampaignSpec against the JSON schema.
// Returns a list of validation error messages, or an empty list if valid.
func ValidateSpecWithSchema(spec *CampaignSpec) []string {
	// Read embedded schema
	schemaData, err := campaignSpecSchemaFS.ReadFile("schemas/campaign_spec_schema.json")
	if err != nil {
		return []string{fmt.Sprintf("failed to load campaign spec schema: %v", err)}
	}

	// Convert spec to JSON for validation, excluding runtime fields.
	// Create a copy without ConfigPath (which is set at runtime, not in YAML).
	//
	// JSON property names intentionally mirror the kebab-case YAML keys so the
	// JSON Schema can validate both YAML and JSON representations consistently.
	type CampaignSpecForValidation struct {
		ID                 string                  `json:"id"`
		Name               string                  `json:"name"`
		Description        string                  `json:"description,omitempty"`
		Version            string                  `json:"version,omitempty"`
		Workflows          []string                `json:"workflows,omitempty"`
		MemoryPaths        []string                `json:"memory-paths,omitempty"`
		MetricsGlob        string                  `json:"metrics-glob,omitempty"`
		Owners             []string                `json:"owners,omitempty"`
		ExecutiveSponsors  []string                `json:"executive-sponsors,omitempty"`
		RiskLevel          string                  `json:"risk-level,omitempty"`
		TrackerLabel       string                  `json:"tracker-label,omitempty"`
		State              string                  `json:"state,omitempty"`
		Tags               []string                `json:"tags,omitempty"`
		AllowedSafeOutputs []string                `json:"allowed-safe-outputs,omitempty"`
		ApprovalPolicy     *CampaignApprovalPolicy `json:"approval-policy,omitempty"`
	}

	validationSpec := CampaignSpecForValidation{
		ID:                 spec.ID,
		Name:               spec.Name,
		Description:        spec.Description,
		Version:            spec.Version,
		Workflows:          spec.Workflows,
		MemoryPaths:        spec.MemoryPaths,
		MetricsGlob:        spec.MetricsGlob,
		Owners:             spec.Owners,
		ExecutiveSponsors:  spec.ExecutiveSponsors,
		RiskLevel:          spec.RiskLevel,
		TrackerLabel:       spec.TrackerLabel,
		State:              spec.State,
		Tags:               spec.Tags,
		AllowedSafeOutputs: spec.AllowedSafeOutputs,
		ApprovalPolicy:     spec.ApprovalPolicy,
	}

	specJSON, err := json.Marshal(validationSpec)
	if err != nil {
		return []string{fmt.Sprintf("failed to marshal spec to JSON: %v", err)}
	}

	// Create schema and document loaders
	schemaLoader := gojsonschema.NewBytesLoader(schemaData)
	documentLoader := gojsonschema.NewBytesLoader(specJSON)

	// Validate
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return []string{fmt.Sprintf("schema validation error: %v", err)}
	}

	if result.Valid() {
		return nil
	}

	// Collect validation errors
	var problems []string
	for _, err := range result.Errors() {
		// Format error message similar to how workflow validation does it
		field := err.Field()
		if field == "(root)" {
			field = "root"
		}
		problems = append(problems, fmt.Sprintf("%s: %s", field, err.Description()))
	}

	return problems
}

// ValidateSpecFromFile validates a campaign spec file by loading and validating it.
// This is useful for validation commands that operate on files directly.
func ValidateSpecFromFile(filePath string) (*CampaignSpec, []string, error) {
	data, err := parser.ExtractFrontmatterFromContent(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	if len(data.Frontmatter) == 0 {
		return nil, nil, fmt.Errorf("no frontmatter found in campaign spec file")
	}

	// Convert frontmatter to CampaignSpec
	specJSON, err := json.Marshal(data.Frontmatter)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal frontmatter: %w", err)
	}

	var spec CampaignSpec
	if err := json.Unmarshal(specJSON, &spec); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal spec: %w", err)
	}

	problems := ValidateSpec(&spec)
	return &spec, problems, nil
}
