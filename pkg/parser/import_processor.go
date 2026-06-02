// Package parser provides functions for parsing and processing workflow markdown files.
// import_processor.go defines the public API and core types for the import processing system.
// The import system is implemented across multiple focused modules:
//   - import_bfs.go: BFS traversal core
//   - import_field_extractor.go: Field extraction and result accumulation
//   - import_remote.go: Remote origin types and workflowspec parsing
//   - import_cycle.go: Cycle detection
//   - import_topological.go: Topological ordering
package parser

import (
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/types"
)

var importLog = logger.New("parser:import_processor")

// ImportsResult holds the result of processing imports from frontmatter
type ImportsResult struct {
	MergedTools                   string                // Merged tools configuration from all imports
	MergedMCPServers              string                // Merged mcp-servers configuration from all imports
	MergedEngines                 []string              // Merged engine configurations from all imports
	MergedSafeOutputs             []string              // Merged safe-outputs configurations from all imports
	MergedMCPScripts              []string              // Merged mcp-scripts configurations from all imports
	MergedMarkdown                string                // Only contains imports WITH inputs (for compile-time substitution)
	ImportPaths                   []string              // List of import file paths for runtime-import macro generation (replaces MergedMarkdown)
	MergedSteps                   string                // Merged steps configuration from all imports (excluding copilot-setup-steps)
	CopilotSetupSteps             string                // Steps from copilot-setup-steps.yml (inserted at start)
	MergedPreSteps                string                // Merged pre-steps configuration from all imports (prepended in order)
	MergedPreAgentSteps           string                // Merged pre-agent-steps configuration from all imports (prepended in order)
	MergedRuntimes                string                // Merged runtimes configuration from all imports
	MergedRunInstallScripts       bool                  // true if any imported workflow sets runtimes.node.run-install-scripts: true
	MergedServices                string                // Merged services configuration from all imports
	MergedNetwork                 string                // Merged network configuration from all imports
	MergedPermissions             string                // Merged permissions configuration from all imports
	MergedSecretMasking           string                // Merged secret-masking steps from all imports
	MergedBots                    []string              // Merged bots list from all imports (union of bot names)
	MergedSkipRoles               []string              // Merged skip-roles list from all imports (union of role names)
	MergedSkipBots                []string              // Merged skip-bots list from all imports (union of usernames)
	MergedSkipIfMatch             string                // on.skip-if-match from first imported workflow that defines it (JSON-encoded)
	MergedSkipIfNoMatch           string                // on.skip-if-no-match from first imported workflow that defines it (JSON-encoded)
	MergedActivationGitHubToken   string                // GitHub token from on.github-token in first imported workflow that defines it
	MergedActivationGitHubApp     string                // JSON-encoded on.github-app from first imported workflow that defines it
	MergedTopLevelGitHubApp       string                // JSON-encoded top-level github-app from first imported workflow that defines it
	MergedCheckout                string                // JSON-encoded checkout configurations from imported workflows (one JSON value per line)
	MergedPostSteps               string                // Merged post-steps configuration from all imports (appended in order)
	MergedLabels                  []string              // Merged labels from all imports (union of label names)
	MergedCaches                  []string              // Merged cache configurations from all imports (appended in order)
	MergedJobs                    string                // Merged jobs from imported YAML workflows (JSON format)
	MergedEnv                     string                // Merged env configuration from all imports (JSON format)
	MergedEnvSources              map[string]string     // env var name → source import path (for conflict detection and lock file header listing)
	MergedFeatures                []map[string]any      // Merged features configuration from all imports (parsed YAML structures)
	MergedModels                  []map[string][]string // Merged model alias definitions from all imports (first import to define a key wins among imports)
	MergedObservability           string                // Merged observability config (JSON) from all imports as an endpoint array (deduped by URL)
	MergedEngineMCPToolTimeout    string                // First engine.mcp.tool-timeout found across all imports (Go duration string, e.g. "10m")
	MergedEngineMCPSessionTimeout string                // First engine.mcp.session-timeout found across all imports (Go duration string, e.g. "4h")
	MergedEngineModel             string                // First engine.model found in imports that have no engine.id (model preference without engine selection)
	MergedMaxTurns                string                // First max-turns value found across all imports (JSON-encoded, first-wins)
	MergedMaxRuns                 string                // First max-runs value found across all imports (JSON-encoded, first-wins)
	MergedMaxEffectiveTokens      string                // First max-effective-tokens value found across all imports (JSON-encoded, first-wins)
	MergedMaxDailyEffectiveTokens string                // First max-daily-effective-tokens value found across all imports (JSON-encoded, first-wins)
	ImportedFiles                 []string              // List of imported file paths (for manifest)
	AgentFile                     string                // Path to custom agent file (if imported)
	AgentImportSpec               string                // Original import specification for agent file (e.g., "owner/repo/path@ref")
	RepositoryImports             []string              // List of repository imports (format: "owner/repo@ref") for .github folder merging
	// ImportInputs uses map[string]any because input values can be different types (string, number, boolean).
	// This is parsed from YAML frontmatter where the structure is dynamic and not known at compile time.
	// This is an appropriate use of 'any' for dynamic YAML/JSON data.
	// See scratchpad/go-type-patterns.md for guidance on when to use map[string]any.
	ImportInputs map[string]any // Aggregated input values from all imports (key = input name, value = input value)
	// Warnings contains best-effort advisory messages collected while processing imports
	// (e.g. unknown frontmatter fields in inline sub-agent blocks). Callers should surface
	// these to the user but must not treat them as compilation failures.
	Warnings []string
}

// ImportInputDefinition defines an input parameter for a shared workflow import.
// Uses the same schema as workflow_dispatch inputs.
// NOTE: The parser package still uses map[string]any for actual parsing to avoid prematurely
// constraining dynamic YAML/JSON payloads during frontmatter extraction.
type ImportInputDefinition = types.InputDefinition

// ImportSpec represents a single import specification (either a string path or an object with path and inputs)
type ImportSpec struct {
	Path string // Import path (required)
	// Inputs uses map[string]any because input values can be different types (string, number, boolean).
	// This is parsed from YAML frontmatter and validated against the imported workflow's input definitions.
	// This is an appropriate use of 'any' for dynamic YAML data. See scratchpad/go-type-patterns.md.
	Inputs map[string]any // Optional input values to pass to the imported workflow (values are string, number, or boolean)
}

// ProcessImportsFromFrontmatterWithSource processes imports field from frontmatter with source tracking
// This version includes the workflow file path and YAML content for better error reporting
func ProcessImportsFromFrontmatterWithSource(frontmatter map[string]any, baseDir string, cache *ImportCache, workflowFilePath string, yamlContent string) (*ImportsResult, error) {
	importLog.Printf("Processing imports: workflowFile=%s, baseDir=%s", workflowFilePath, baseDir)
	result, err := processImportsFromFrontmatterWithManifestAndSource(frontmatter, baseDir, cache, workflowFilePath, yamlContent)
	if err != nil {
		importLog.Printf("Import processing failed for %s: %v", workflowFilePath, err)
		return result, err
	}
	if result != nil {
		importLog.Printf("Import processing complete: importedFiles=%d, mergedTools=%d bytes", len(result.ImportedFiles), len(result.MergedTools))
	}
	return result, nil
}
