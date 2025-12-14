package campaign

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/githubnext/gh-aw/pkg/parser"
	"github.com/goccy/go-yaml"
	"github.com/xeipuuv/gojsonschema"
)

//go:embed schemas/campaign_spec_schema.json
var campaignSpecSchemaFS embed.FS

// Cached compiled schema to avoid reloading on every validation
var (
	campaignSchemaOnce sync.Once
	campaignSchema     gojsonschema.JSONLoader
	schemaLoadError    error
)

// getCampaignSchema returns the cached campaign schema loader, loading it once
func getCampaignSchema() (gojsonschema.JSONLoader, error) {
	campaignSchemaOnce.Do(func() {
		// Load schema from embed.FS
		schemaData, err := campaignSpecSchemaFS.ReadFile("schemas/campaign_spec_schema.json")
		if err != nil {
			schemaLoadError = fmt.Errorf("failed to load campaign spec schema: %w", err)
			return
		}

		// Create schema loader once
		campaignSchema = gojsonschema.NewBytesLoader(schemaData)
	})

	return campaignSchema, schemaLoadError
}

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
	// Get cached schema loader
	schemaLoader, err := getCampaignSchema()
	if err != nil {
		return []string{err.Error()}
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

	// Create document loader
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
	// Read the campaign spec file content first, then extract frontmatter
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read campaign spec file: %w", err)
	}

	data, err := parser.ExtractFrontmatterFromContent(string(content))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	if len(data.Frontmatter) == 0 {
		return nil, nil, fmt.Errorf("no frontmatter found in campaign spec file")
	}

	// Convert frontmatter map into YAML, then unmarshal into CampaignSpec using
	// YAML tags so kebab-case keys (e.g. tracker-label) map correctly.
	yamlBytes, err := yaml.Marshal(data.Frontmatter)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal frontmatter to YAML: %w", err)
	}

	var spec CampaignSpec
	if err := yaml.Unmarshal(yamlBytes, &spec); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal spec from YAML: %w", err)
	}

	problems := ValidateSpec(&spec)
	return &spec, problems, nil
}

// ValidateWorkflowsExist checks that all workflows referenced in a campaign spec
// actually exist in the .github/workflows directory.
// Returns a list of problems for workflows that don't exist.
func ValidateWorkflowsExist(spec *CampaignSpec, workflowsDir string) []string {
	var problems []string

	for _, workflowID := range spec.Workflows {
		// Check for both .md and .lock.yml versions
		mdPath := filepath.Join(workflowsDir, workflowID+".md")
		lockPath := filepath.Join(workflowsDir, workflowID+".lock.yml")

		mdExists := false
		lockExists := false

		if _, err := os.Stat(mdPath); err == nil {
			mdExists = true
		}
		if _, err := os.Stat(lockPath); err == nil {
			lockExists = true
		}

		if !mdExists && !lockExists {
			problems = append(problems, fmt.Sprintf("workflow '%s' not found (expected %s.md or %s.lock.yml)", workflowID, workflowID, workflowID))
		} else if !mdExists {
			problems = append(problems, fmt.Sprintf("workflow '%s' has lock file but missing source .md file", workflowID))
		}
	}

	return problems
}
