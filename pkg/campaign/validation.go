package campaign

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/parser"
	"github.com/goccy/go-yaml"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

var validationLog = logger.New("campaign:validation")

//go:embed schemas/campaign_spec_schema.json
var campaignSpecSchemaFS embed.FS

// Cached compiled schema to avoid recompiling on every validation
var (
	compiledSchemaOnce sync.Once
	compiledSchema     *jsonschema.Schema
	schemaCompileError error
)

// ValidateSpec performs lightweight semantic validation of a
// single CampaignSpec and returns a slice of human-readable problems.
//
// It uses JSON schema validation first, then adds additional semantic checks.
func ValidateSpec(spec *CampaignSpec) []string {
	validationLog.Printf("Validating campaign spec: id=%s", spec.ID)
	var problems []string

	// First, validate against JSON schema
	schemaProblems := ValidateSpecWithSchema(spec)
	problems = append(problems, schemaProblems...)
	if len(schemaProblems) > 0 {
		validationLog.Printf("Schema validation found %d problems for campaign '%s'", len(schemaProblems), spec.ID)
	}

	// Additional semantic validation beyond schema
	trimmedID := strings.TrimSpace(spec.ID)
	if trimmedID == "" {
		problems = append(problems, "id is required and must be non-empty - example: 'security-q1-2025'")
	} else {
		// Enforce a simple, URL-safe pattern for IDs
		for _, ch := range trimmedID {
			if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-' {
				continue
			}
			problems = append(problems, fmt.Sprintf("id must use only lowercase letters, digits, and hyphens - got '%s', try '%s'", trimmedID, suggestValidID(trimmedID)))
			break
		}
	}

	if strings.TrimSpace(spec.Name) == "" {
		problems = append(problems, "name should be provided (falls back to id, but explicit names are recommended) - example: 'Security Q1 2025'")
	}

	if len(spec.Workflows) == 0 {
		problems = append(problems, "workflows should list at least one workflow implementing this campaign - example: ['vulnerability-scanner', 'dependency-updater']")
	}

	// Validate tracker-label format if provided
	if spec.TrackerLabel != "" {
		trimmedLabel := strings.TrimSpace(spec.TrackerLabel)
		// Tracker labels should follow the pattern "campaign:campaign-id"
		if !strings.HasPrefix(trimmedLabel, "campaign:") {
			problems = append(problems, fmt.Sprintf("tracker-label should start with 'campaign:' prefix - got '%s', recommended: 'campaign:%s'", trimmedLabel, spec.ID))
		}
		// Check for invalid characters in labels (GitHub label restrictions)
		if strings.Contains(trimmedLabel, " ") {
			problems = append(problems, fmt.Sprintf("tracker-label cannot contain spaces - got '%s', try replacing spaces with hyphens", trimmedLabel))
		}
	}

	// Validate that campaigns with workflows or tracker-label have allowed-repos or allowed-orgs
	// This ensures discovery is properly scoped
	hasDiscovery := len(spec.Workflows) > 0 || spec.TrackerLabel != ""
	hasScope := len(spec.AllowedRepos) > 0 || len(spec.AllowedOrgs) > 0
	if hasDiscovery && !hasScope {
		problems = append(problems, "campaigns with workflows or tracker-label must specify allowed-repos or allowed-orgs for discovery scoping - configure at least one to define where the campaign can discover items")
	}

	// Validate allowed-repos format if provided (now optional - defaults to current repo)
	if len(spec.AllowedRepos) > 0 {
		// Validate each repository format
		for _, repo := range spec.AllowedRepos {
			trimmed := strings.TrimSpace(repo)
			if trimmed == "" {
				problems = append(problems, "allowed-repos must not contain empty entries - remove empty strings from the list")
				continue
			}
			// Validate owner/repo format
			parts := strings.Split(trimmed, "/")
			if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
				problems = append(problems, fmt.Sprintf("allowed-repos entry '%s' must be in 'owner/repo' format - example: 'github/docs' or 'myorg/myrepo'", trimmed))
			}
			// Warn about common mistakes
			if strings.Contains(trimmed, "*") {
				problems = append(problems, fmt.Sprintf("allowed-repos entry '%s' cannot contain wildcards - list each repository explicitly or use allowed-orgs for organization-wide scope", trimmed))
			}
		}
	}

	// Validate allowed-orgs if provided (optional)
	if len(spec.AllowedOrgs) > 0 {
		for _, org := range spec.AllowedOrgs {
			trimmed := strings.TrimSpace(org)
			if trimmed == "" {
				problems = append(problems, "allowed-orgs must not contain empty entries - remove empty strings from the list")
				continue
			}
			// Validate organization name format (no slashes, valid GitHub org name)
			if strings.Contains(trimmed, "/") {
				problems = append(problems, fmt.Sprintf("allowed-orgs entry '%s' must be an organization name only (not owner/repo format) - example: 'github' not 'github/docs'", trimmed))
			}
			if strings.Contains(trimmed, "*") {
				problems = append(problems, fmt.Sprintf("allowed-orgs entry '%s' cannot contain wildcards - use the organization name directly (e.g., 'myorg')", trimmed))
			}
		}
	}

	if strings.TrimSpace(spec.ProjectURL) == "" {
		problems = append(problems, "project-url is required (GitHub Project URL used as the campaign dashboard) - example: 'https://github.com/orgs/myorg/projects/1'")
	} else {
		parsed, err := url.Parse(strings.TrimSpace(spec.ProjectURL))
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			problems = append(problems, "project-url must be a valid absolute URL - example: 'https://github.com/orgs/myorg/projects/1' or 'https://github.com/users/username/projects/1'")
		} else if !strings.Contains(parsed.Path, "/projects/") {
			problems = append(problems, "project-url must point to a GitHub Project (URL path should include '/projects/') - you may have provided a repository URL instead")
		}
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

	if spec.Governance != nil {
		if spec.Governance.MaxNewItemsPerRun < 0 {
			problems = append(problems, "governance.max-new-items-per-run must be >= 0")
		}
		if spec.Governance.MaxDiscoveryItemsPerRun < 0 {
			problems = append(problems, "governance.max-discovery-items-per-run must be >= 0")
		}
		if spec.Governance.MaxDiscoveryPagesPerRun < 0 {
			problems = append(problems, "governance.max-discovery-pages-per-run must be >= 0")
		}
		if spec.Governance.MaxProjectUpdatesPerRun < 0 {
			problems = append(problems, "governance.max-project-updates-per-run must be >= 0")
		}
		if spec.Governance.MaxCommentsPerRun < 0 {
			problems = append(problems, "governance.max-comments-per-run must be >= 0")
		}
	}

	// Goals/KPIs: optional, but when provided they must be consistent and well-formed.
	problems = append(problems, validateObjectiveAndKPIs(spec)...)

	if len(problems) == 0 {
		validationLog.Printf("Campaign spec '%s' validation passed with no problems", spec.ID)
	} else {
		validationLog.Printf("Campaign spec '%s' validation completed with %d problems", spec.ID, len(problems))
	}

	return problems
}

func validateObjectiveAndKPIs(spec *CampaignSpec) []string {
	var problems []string

	objective := strings.TrimSpace(spec.Objective)
	if objective == "" && len(spec.KPIs) > 0 {
		problems = append(problems, "objective should be set when kpis are provided - describe what success looks like for this campaign")
	}
	if objective != "" && len(spec.KPIs) == 0 {
		problems = append(problems, "kpis should include at least one KPI when objective is provided - add measurable metrics (e.g., 'Pull requests merged: 0 → 100 over 30 days')")
	}
	if len(spec.KPIs) == 0 {
		return problems
	}

	primaryCount := 0
	for _, kpi := range spec.KPIs {
		name := strings.TrimSpace(kpi.Name)
		if name == "" {
			name = "(unnamed)"
		}
		if strings.TrimSpace(kpi.Priority) == "primary" {
			primaryCount++
		}
		if kpi.TimeWindowDays < 1 {
			problems = append(problems, fmt.Sprintf("kpi '%s': time-window-days must be >= 1 - specify the rolling time window in days (e.g., 30 for monthly)", name))
		}
		if dir := strings.TrimSpace(kpi.Direction); dir != "" {
			switch dir {
			case "increase", "decrease":
				// ok
			default:
				problems = append(problems, fmt.Sprintf("kpi '%s': direction must be one of: 'increase' or 'decrease' - got '%s'", name, dir))
			}
		}
		if src := strings.TrimSpace(kpi.Source); src != "" {
			switch src {
			case "ci", "pull_requests", "code_security", "custom":
				// ok
			default:
				problems = append(problems, fmt.Sprintf("kpi '%s': source must be one of: 'ci', 'pull_requests', 'code_security', or 'custom' - got '%s'", name, src))
			}
		}
	}

	// Semantic rule: exactly one primary KPI when there are multiple KPIs.
	// If there is only one KPI and priority is omitted, treat it as implicitly primary.
	if len(spec.KPIs) == 1 {
		if strings.TrimSpace(spec.KPIs[0].Priority) == "" {
			return problems
		}
	}
	if primaryCount == 0 {
		problems = append(problems, "kpis must include exactly one primary KPI (priority: primary) - mark your main success metric as primary")
	}
	if primaryCount > 1 {
		problems = append(problems, fmt.Sprintf("kpis must include exactly one primary KPI (found %d primary KPIs) - choose one main success metric and mark others as 'supporting'", primaryCount))
	}

	return problems
}

// suggestValidID converts an invalid campaign ID into a valid one by:
// - Converting to lowercase
// - Replacing invalid characters with hyphens
// - Collapsing multiple hyphens
// - Trimming leading/trailing hyphens
func suggestValidID(id string) string {
	// Convert to lowercase
	result := strings.ToLower(id)

	// Replace invalid characters with hyphens
	var builder strings.Builder
	for _, ch := range result {
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') {
			builder.WriteRune(ch)
		} else {
			builder.WriteRune('-')
		}
	}
	result = builder.String()

	// Collapse multiple hyphens into single hyphen
	for strings.Contains(result, "--") {
		result = strings.ReplaceAll(result, "--", "-")
	}

	// Trim leading/trailing hyphens
	result = strings.Trim(result, "-")

	return result
}

// getCompiledSchema returns the compiled campaign spec schema, compiling it once and caching
func getCompiledSchema() (*jsonschema.Schema, error) {
	compiledSchemaOnce.Do(func() {
		// Read embedded schema
		schemaData, err := campaignSpecSchemaFS.ReadFile("schemas/campaign_spec_schema.json")
		if err != nil {
			schemaCompileError = fmt.Errorf("failed to load campaign spec schema: %w", err)
			return
		}

		// Parse schema as JSON
		var schemaDoc any
		if err := json.Unmarshal(schemaData, &schemaDoc); err != nil {
			schemaCompileError = fmt.Errorf("failed to parse campaign spec schema: %w", err)
			return
		}

		// Create compiler and add schema resource
		compiler := jsonschema.NewCompiler()
		schemaURL := "campaign-spec.json"
		if err := compiler.AddResource(schemaURL, schemaDoc); err != nil {
			schemaCompileError = fmt.Errorf("failed to add schema resource: %w", err)
			return
		}

		// Compile schema once
		schema, err := compiler.Compile(schemaURL)
		if err != nil {
			schemaCompileError = fmt.Errorf("failed to compile schema: %w", err)
			return
		}

		compiledSchema = schema
	})

	return compiledSchema, schemaCompileError
}

// ValidateSpecWithSchema validates a CampaignSpec against the JSON schema.
// Returns a list of validation error messages, or an empty list if valid.
func ValidateSpecWithSchema(spec *CampaignSpec) []string {
	// Get the cached compiled schema
	schema, err := getCompiledSchema()
	if err != nil {
		return []string{err.Error()}
	}

	// Convert spec to JSON for validation, excluding runtime fields.
	// Create a copy without ConfigPath (which is set at runtime, not in YAML).
	//
	// JSON property names intentionally mirror the kebab-case YAML keys so the
	// JSON Schema can validate both YAML and JSON representations consistently.
	type CampaignGovernancePolicyForValidation struct {
		MaxNewItemsPerRun       int      `json:"max-new-items-per-run,omitempty"`
		MaxDiscoveryItemsPerRun int      `json:"max-discovery-items-per-run,omitempty"`
		MaxDiscoveryPagesPerRun int      `json:"max-discovery-pages-per-run,omitempty"`
		OptOutLabels            []string `json:"opt-out-labels,omitempty"`
		DoNotDowngradeDoneItems *bool    `json:"do-not-downgrade-done-items,omitempty"`
		MaxProjectUpdatesPerRun int      `json:"max-project-updates-per-run,omitempty"`
		MaxCommentsPerRun       int      `json:"max-comments-per-run,omitempty"`
	}

	type CampaignKPIForValidation struct {
		ID             string  `json:"id,omitempty"`
		Name           string  `json:"name"`
		Priority       string  `json:"priority,omitempty"`
		Unit           string  `json:"unit,omitempty"`
		Baseline       float64 `json:"baseline"`
		Target         float64 `json:"target"`
		TimeWindowDays int     `json:"time-window-days"`
		Direction      string  `json:"direction,omitempty"`
		Source         string  `json:"source,omitempty"`
	}

	type CampaignSpecForValidation struct {
		ID                 string                                 `json:"id"`
		Name               string                                 `json:"name"`
		Description        string                                 `json:"description,omitempty"`
		Objective          string                                 `json:"objective,omitempty"`
		KPIs               []CampaignKPIForValidation             `json:"kpis,omitempty"`
		ProjectURL         string                                 `json:"project-url,omitempty"`
		ProjectGitHubToken string                                 `json:"project-github-token,omitempty"`
		Version            string                                 `json:"version,omitempty"`
		Workflows          []string                               `json:"workflows,omitempty"`
		AllowedRepos       []string                               `json:"allowed-repos,omitempty"`
		AllowedOrgs        []string                               `json:"allowed-orgs,omitempty"`
		MemoryPaths        []string                               `json:"memory-paths,omitempty"`
		MetricsGlob        string                                 `json:"metrics-glob,omitempty"`
		CursorGlob         string                                 `json:"cursor-glob,omitempty"`
		Owners             []string                               `json:"owners,omitempty"`
		ExecutiveSponsors  []string                               `json:"executive-sponsors,omitempty"`
		RiskLevel          string                                 `json:"risk-level,omitempty"`
		State              string                                 `json:"state,omitempty"`
		Tags               []string                               `json:"tags,omitempty"`
		AllowedSafeOutputs []string                               `json:"allowed-safe-outputs,omitempty"`
		Governance         *CampaignGovernancePolicyForValidation `json:"governance,omitempty"`
		ApprovalPolicy     *CampaignApprovalPolicy                `json:"approval-policy,omitempty"`
	}

	validationSpec := CampaignSpecForValidation{
		ID:          spec.ID,
		Name:        spec.Name,
		Description: spec.Description,
		Objective:   strings.TrimSpace(spec.Objective),
		KPIs: func() []CampaignKPIForValidation {
			if len(spec.KPIs) == 0 {
				return nil
			}
			out := make([]CampaignKPIForValidation, 0, len(spec.KPIs))
			for _, kpi := range spec.KPIs {
				out = append(out, CampaignKPIForValidation{
					ID:             strings.TrimSpace(kpi.ID),
					Name:           strings.TrimSpace(kpi.Name),
					Priority:       strings.TrimSpace(kpi.Priority),
					Unit:           strings.TrimSpace(kpi.Unit),
					Baseline:       kpi.Baseline,
					Target:         kpi.Target,
					TimeWindowDays: kpi.TimeWindowDays,
					Direction:      strings.TrimSpace(kpi.Direction),
					Source:         strings.TrimSpace(kpi.Source),
				})
			}
			return out
		}(),
		ProjectURL:         spec.ProjectURL,
		ProjectGitHubToken: spec.ProjectGitHubToken,
		Version:            spec.Version,
		Workflows:          spec.Workflows,
		AllowedRepos:       spec.AllowedRepos,
		AllowedOrgs:        spec.AllowedOrgs,
		MemoryPaths:        spec.MemoryPaths,
		MetricsGlob:        spec.MetricsGlob,
		CursorGlob:         spec.CursorGlob,
		Owners:             spec.Owners,
		ExecutiveSponsors:  spec.ExecutiveSponsors,
		RiskLevel:          spec.RiskLevel,
		State:              spec.State,
		Tags:               spec.Tags,
		AllowedSafeOutputs: spec.AllowedSafeOutputs,
		Governance: func() *CampaignGovernancePolicyForValidation {
			if spec.Governance == nil {
				return nil
			}
			return &CampaignGovernancePolicyForValidation{
				MaxNewItemsPerRun:       spec.Governance.MaxNewItemsPerRun,
				MaxDiscoveryItemsPerRun: spec.Governance.MaxDiscoveryItemsPerRun,
				MaxDiscoveryPagesPerRun: spec.Governance.MaxDiscoveryPagesPerRun,
				OptOutLabels:            spec.Governance.OptOutLabels,
				DoNotDowngradeDoneItems: spec.Governance.DoNotDowngradeDoneItems,
				MaxProjectUpdatesPerRun: spec.Governance.MaxProjectUpdatesPerRun,
				MaxCommentsPerRun:       spec.Governance.MaxCommentsPerRun,
			}
		}(),
		ApprovalPolicy: spec.ApprovalPolicy,
	}

	// Marshal spec to JSON then unmarshal to any for validation
	// This is necessary because the jsonschema library validates against the JSON representation
	specJSON, err := json.Marshal(validationSpec)
	if err != nil {
		return []string{fmt.Sprintf("failed to marshal spec to JSON: %v", err)}
	}

	var specData any
	if err := json.Unmarshal(specJSON, &specData); err != nil {
		return []string{fmt.Sprintf("failed to unmarshal spec data: %v", err)}
	}

	// Validate the spec against the schema
	if err := schema.Validate(specData); err != nil {
		return formatValidationErrors(err)
	}

	return nil
}

// formatValidationErrors converts jsonschema validation errors to a list of human-readable messages
func formatValidationErrors(err error) []string {
	var problems []string

	ve, ok := err.(*jsonschema.ValidationError)
	if !ok {
		// Not a validation error, return as-is
		return []string{err.Error()}
	}

	// Process the main error and all causes
	var collectErrors func(*jsonschema.ValidationError)
	collectErrors = func(e *jsonschema.ValidationError) {
		// Skip collecting if there are causes - we'll collect those instead
		// to avoid duplicate/redundant messages
		if len(e.Causes) > 0 {
			for _, cause := range e.Causes {
				collectErrors(cause)
			}
			return
		}

		// Format the error message with field path
		field := "root"
		if len(e.InstanceLocation) > 0 {
			field = strings.Join(e.InstanceLocation, ".")
		}

		// Use the error's Error() method to get the message
		msg := e.Error()

		problems = append(problems, fmt.Sprintf("%s: %s", field, msg))
	}

	collectErrors(ve)
	return problems
}

// ValidateSpecFromFile validates a campaign spec file by loading and validating it.
// This is useful for validation commands that operate on files directly.
func ValidateSpecFromFile(filePath string) (*CampaignSpec, []string, error) {
	validationLog.Printf("Validating campaign spec from file: %s", filePath)

	// Read the campaign spec file content first, then extract frontmatter
	content, err := os.ReadFile(filePath)
	if err != nil {
		validationLog.Printf("Failed to read campaign spec file: %s", err)
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
	// YAML tags so kebab-case keys (e.g. project-url) map correctly.
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
	validationLog.Printf("Validating workflow existence for campaign '%s': checking %d workflows in %s",
		spec.ID, len(spec.Workflows), workflowsDir)
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
			problems = append(problems, fmt.Sprintf("workflow '%s' not found", workflowID))
		} else if !mdExists {
			problems = append(problems, fmt.Sprintf("workflow '%s' missing source .md file", workflowID))
		}
	}

	return problems
}
