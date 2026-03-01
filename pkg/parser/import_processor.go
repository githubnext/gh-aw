// Package parser provides functions for parsing and processing workflow markdown files.
// import_processor.go defines the public API and core types for the import processing system.
// The import system is implemented across multiple focused modules:
//   - import_bfs.go: BFS traversal core
//   - import_field_extractor.go: Field extraction and result accumulation
//   - import_remote.go: Remote origin types and workflowspec parsing
//   - import_cycle.go: Cycle detection
//   - import_topological.go: Topological ordering
package parser

import "github.com/github/gh-aw/pkg/logger"

var importLog = logger.New("parser:import_processor")

// ImportsResult holds the result of processing imports from frontmatter
type ImportsResult struct {
	MergedTools         string           // Merged tools configuration from all imports
	MergedMCPServers    string           // Merged mcp-servers configuration from all imports
	MergedEngines       []string         // Merged engine configurations from all imports
	MergedSafeOutputs   []string         // Merged safe-outputs configurations from all imports
	MergedSafeInputs    []string         // Merged safe-inputs configurations from all imports
	MergedMarkdown      string           // Only contains imports WITH inputs (for compile-time substitution)
	ImportPaths         []string         // List of import file paths for runtime-import macro generation (replaces MergedMarkdown)
	MergedSteps         string           // Merged steps configuration from all imports (excluding copilot-setup-steps)
	CopilotSetupSteps   string           // Steps from copilot-setup-steps.yml (inserted at start)
	MergedRuntimes      string           // Merged runtimes configuration from all imports
	MergedServices      string           // Merged services configuration from all imports
	MergedNetwork       string           // Merged network configuration from all imports
	MergedPermissions   string           // Merged permissions configuration from all imports
	MergedSecretMasking string           // Merged secret-masking steps from all imports
	MergedBots          []string         // Merged bots list from all imports (union of bot names)
	MergedPlugins       []string         // Merged plugins list from all imports (union of plugin repos)
	MergedSkipRoles     []string         // Merged skip-roles list from all imports (union of role names)
	MergedSkipBots      []string         // Merged skip-bots list from all imports (union of usernames)
	MergedPostSteps     string           // Merged post-steps configuration from all imports (appended in order)
	MergedLabels        []string         // Merged labels from all imports (union of label names)
	MergedCaches        []string         // Merged cache configurations from all imports (appended in order)
	MergedJobs          string           // Merged jobs from imported YAML workflows (JSON format)
	MergedFeatures      []map[string]any // Merged features configuration from all imports (parsed YAML structures)
	ImportedFiles       []string         // List of imported file paths (for manifest)
	AgentFile           string           // Path to custom agent file (if imported)
	AgentImportSpec     string           // Original import specification for agent file (e.g., "owner/repo/path@ref")
	RepositoryImports   []string         // List of repository imports (format: "owner/repo@ref") for .github folder merging
	// ImportInputs uses map[string]any because input values can be different types (string, number, boolean).
	// This is parsed from YAML frontmatter where the structure is dynamic and not known at compile time.
	// This is an appropriate use of 'any' for dynamic YAML/JSON data.
	// See scratchpad/go-type-patterns.md for guidance on when to use map[string]any.
	ImportInputs map[string]any // Aggregated input values from all imports (key = input name, value = input value)
}

// ImportInputDefinition defines an input parameter for a shared workflow import.
// Uses the same schema as workflow_dispatch inputs.
// NOTE: This type matches workflow.InputDefinition which is the canonical type for input parameters.
// The parser package uses map[string]any for actual parsing to avoid circular dependencies.
type ImportInputDefinition struct {
	Description string   `yaml:"description,omitempty" json:"description,omitempty"`
	Required    bool     `yaml:"required,omitempty" json:"required,omitempty"`
	Default     any      `yaml:"default,omitempty" json:"default,omitempty"` // Can be string, number, or boolean (dynamic type from YAML)
	Type        string   `yaml:"type,omitempty" json:"type,omitempty"`       // "string", "choice", "boolean", "number"
	Options     []string `yaml:"options,omitempty" json:"options,omitempty"` // Options for choice type
}

// ImportSpec represents a single import specification (either a string path or an object with path and inputs)
type ImportSpec struct {
	Path string // Import path (required)
	// Inputs uses map[string]any because input values can be different types (string, number, boolean).
	// This is parsed from YAML frontmatter and validated against the imported workflow's input definitions.
	// This is an appropriate use of 'any' for dynamic YAML data. See scratchpad/go-type-patterns.md.
	Inputs map[string]any // Optional input values to pass to the imported workflow (values are string, number, or boolean)
}

// ProcessImportsFromFrontmatter processes imports field from frontmatter
// Returns merged tools and engines from imported files
//
// Type Pattern Note: frontmatter uses map[string]any because it represents parsed YAML with
// dynamic structure that varies by workflow. This is the appropriate pattern for parsing
// user-provided configuration files. See scratchpad/go-type-patterns.md for guidance.
func ProcessImportsFromFrontmatter(frontmatter map[string]any, baseDir string) (mergedTools string, mergedEngines []string, err error) {
	log.Printf("Processing imports from frontmatter: baseDir=%s", baseDir)
	result, err := ProcessImportsFromFrontmatterWithManifest(frontmatter, baseDir, nil)
	if err != nil {
		return "", nil, err
	}
	return result.MergedTools, result.MergedEngines, nil
}

// ProcessImportsFromFrontmatterWithManifest processes imports field from frontmatter
// Returns result containing merged tools, engines, markdown content, and list of imported files
// Uses BFS traversal with queues for deterministic ordering and cycle detection
func ProcessImportsFromFrontmatterWithManifest(frontmatter map[string]any, baseDir string, cache *ImportCache) (*ImportsResult, error) {
	return processImportsFromFrontmatterWithManifestAndSource(frontmatter, baseDir, cache, "", "")
}

// ProcessImportsFromFrontmatterWithSource processes imports field from frontmatter with source tracking
// This version includes the workflow file path and YAML content for better error reporting
func ProcessImportsFromFrontmatterWithSource(frontmatter map[string]any, baseDir string, cache *ImportCache, workflowFilePath string, yamlContent string) (*ImportsResult, error) {
	return processImportsFromFrontmatterWithManifestAndSource(frontmatter, baseDir, cache, workflowFilePath, yamlContent)
}
