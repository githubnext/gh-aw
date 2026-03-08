package workflow

import (
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var runtimeDefLog = logger.New("workflow:runtime_definitions")

// Runtime represents configuration for a runtime environment
type Runtime struct {
	ID              string            // Unique identifier (e.g., "node", "python")
	Name            string            // Display name (e.g., "Node.js", "Python")
	ActionRepo      string            // GitHub Actions repository (e.g., "actions/setup-node")
	ActionVersion   string            // Action version (e.g., "v4", without @ prefix)
	VersionField    string            // Field name for version in action (e.g., "node-version")
	DefaultVersion  string            // Default version to use
	Commands        []string          // Commands that indicate this runtime is needed
	ExtraWithFields map[string]string // Additional 'with' fields for the action
	ManifestFiles   []string          // Package manifest file names for this runtime (matched by filename, no path)
}

// RuntimeRequirement represents a detected runtime requirement
type RuntimeRequirement struct {
	Runtime     *Runtime
	Version     string         // Empty string means use default
	ExtraFields map[string]any // Additional 'with' fields from user's setup step (e.g., cache settings)
	GoModFile   string         // Path to go.mod file for Go runtime (Go-specific)
	IfCondition string         // Optional GitHub Actions if condition
}

// knownRuntimes is the list of all supported runtime configurations (alphabetically sorted by ID)
var knownRuntimes = []*Runtime{
	{
		ID:             "bun",
		Name:           "Bun",
		ActionRepo:     "oven-sh/setup-bun",
		ActionVersion:  "v2",
		VersionField:   "bun-version",
		DefaultVersion: string(constants.DefaultBunVersion),
		Commands:       []string{"bun", "bunx"},
		ManifestFiles:  []string{"package.json", "bun.lockb", "bunfig.toml"},
	},
	{
		ID:             "deno",
		Name:           "Deno",
		ActionRepo:     "denoland/setup-deno",
		ActionVersion:  "v2",
		VersionField:   "deno-version",
		DefaultVersion: string(constants.DefaultDenoVersion),
		Commands:       []string{"deno"},
		ManifestFiles:  []string{"deno.json", "deno.jsonc", "deno.lock"},
	},
	{
		ID:             "dotnet",
		Name:           ".NET",
		ActionRepo:     "actions/setup-dotnet",
		ActionVersion:  "v4",
		VersionField:   "dotnet-version",
		DefaultVersion: string(constants.DefaultDotNetVersion),
		Commands:       []string{"dotnet"},
		ManifestFiles:  []string{"global.json", "NuGet.Config", "Directory.Packages.props"},
	},
	{
		ID:             "elixir",
		Name:           "Elixir",
		ActionRepo:     "erlef/setup-beam",
		ActionVersion:  "v1",
		VersionField:   "elixir-version",
		DefaultVersion: string(constants.DefaultElixirVersion),
		Commands:       []string{"elixir", "mix", "iex"},
		ExtraWithFields: map[string]string{
			"otp-version": "27",
		},
		ManifestFiles: []string{"mix.exs", "mix.lock"},
	},
	{
		ID:             "go",
		Name:           "Go",
		ActionRepo:     "actions/setup-go",
		ActionVersion:  "v5",
		VersionField:   "go-version",
		DefaultVersion: string(constants.DefaultGoVersion),
		Commands:       []string{"go"},
		ExtraWithFields: map[string]string{
			"cache": "false", // Disable caching to prevent cache poisoning in agentic workflows
		},
		ManifestFiles: []string{"go.mod", "go.sum"},
	},
	{
		ID:             "haskell",
		Name:           "Haskell",
		ActionRepo:     "haskell-actions/setup",
		ActionVersion:  "v2",
		VersionField:   "ghc-version",
		DefaultVersion: string(constants.DefaultHaskellVersion),
		Commands:       []string{"ghc", "ghci", "cabal", "stack"},
		ManifestFiles:  []string{"stack.yaml", "stack.yaml.lock"},
	},
	{
		ID:             "java",
		Name:           "Java",
		ActionRepo:     "actions/setup-java",
		ActionVersion:  "v4",
		VersionField:   "java-version",
		DefaultVersion: string(constants.DefaultJavaVersion),
		Commands:       []string{"java", "javac", "mvn", "gradle"},
		ExtraWithFields: map[string]string{
			"distribution": "temurin",
		},
		ManifestFiles: []string{"pom.xml", "build.gradle", "build.gradle.kts", "settings.gradle", "settings.gradle.kts", "gradle.properties"},
	},
	{
		ID:             "node",
		Name:           "Node.js",
		ActionRepo:     "actions/setup-node",
		ActionVersion:  "v6",
		VersionField:   "node-version",
		DefaultVersion: string(constants.DefaultNodeVersion),
		Commands:       []string{"node", "npm", "npx", "yarn", "pnpm"},
		ExtraWithFields: map[string]string{
			"package-manager-cache": "false", // Disable caching by default to prevent cache poisoning in release workflows
		},
		ManifestFiles: []string{"package.json", "package-lock.json", "yarn.lock", "pnpm-lock.yaml", "npm-shrinkwrap.json"},
	},
	{
		ID:             "python",
		Name:           "Python",
		ActionRepo:     "actions/setup-python",
		ActionVersion:  "v5",
		VersionField:   "python-version",
		DefaultVersion: string(constants.DefaultPythonVersion),
		Commands:       []string{"python", "python3", "pip", "pip3"},
		ManifestFiles:  []string{"requirements.txt", "Pipfile", "Pipfile.lock", "pyproject.toml", "setup.py", "setup.cfg"},
	},
	{
		ID:             "ruby",
		Name:           "Ruby",
		ActionRepo:     "ruby/setup-ruby",
		ActionVersion:  "v1",
		VersionField:   "ruby-version",
		DefaultVersion: string(constants.DefaultRubyVersion),
		Commands:       []string{"ruby", "gem", "bundle"},
		ManifestFiles:  []string{"Gemfile", "Gemfile.lock"},
	},
	{
		ID:             "uv",
		Name:           "uv",
		ActionRepo:     "astral-sh/setup-uv",
		ActionVersion:  "v5",
		VersionField:   "version",
		DefaultVersion: "", // Uses latest
		Commands:       []string{"uv", "uvx"},
		ManifestFiles:  []string{"pyproject.toml", "uv.lock"},
	},
}

// commandToRuntime maps command patterns to runtime configurations
var commandToRuntime map[string]*Runtime

// actionRepoToRuntime maps action repository names to runtime configurations
var actionRepoToRuntime map[string]*Runtime

func init() {
	runtimeDefLog.Printf("Initializing runtime definitions: total_runtimes=%d", len(knownRuntimes))

	// Build the command to runtime mapping
	commandToRuntime = make(map[string]*Runtime)
	for _, runtime := range knownRuntimes {
		for _, cmd := range runtime.Commands {
			commandToRuntime[cmd] = runtime
		}
	}
	runtimeDefLog.Printf("Built command to runtime mapping: total_commands=%d", len(commandToRuntime))

	// Build the action repo to runtime mapping
	actionRepoToRuntime = make(map[string]*Runtime)
	for _, runtime := range knownRuntimes {
		actionRepoToRuntime[runtime.ActionRepo] = runtime
	}
	runtimeDefLog.Printf("Built action repo to runtime mapping: total_actions=%d", len(actionRepoToRuntime))
}

// getAllManifestFiles returns the deduplicated union of all manifest file names
// across all known runtimes, plus any additionally-provided filenames.
// These are matched by basename only (no path comparison).
func getAllManifestFiles(extra ...string) []string {
	var files []string
	for _, runtime := range knownRuntimes {
		files = append(files, runtime.ManifestFiles...)
	}
	return mergeUnique(files, extra...)
}

// getProtectedPathPrefixes returns path prefixes (relative to repo root) whose
// contents are always protected regardless of file basename.  Any file whose
// path in the diff starts with one of these prefixes is considered a protected
// file and will trigger the same manifest-file protection logic.
//
// ".github/" covers workflow definitions, CODEOWNERS, Dependabot config, and
// other repository-level security-sensitive configuration.
// ".agents/" covers generic agent instruction and configuration files.
func getProtectedPathPrefixes(extra ...string) []string {
	return mergeUnique([]string{".github/", ".agents/"}, extra...)
}

// mergeUnique returns a deduplicated slice that starts with base and appends any
// items from extra that are not already present in base.  Order is preserved.
func mergeUnique(base []string, extra ...string) []string {
	seen := make(map[string]bool, len(base)+len(extra))
	result := make([]string, 0, len(base)+len(extra))
	for _, v := range base {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	for _, v := range extra {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	return result
}

// findRuntimeByID finds a runtime configuration by its ID
func findRuntimeByID(id string) *Runtime {
	runtimeDefLog.Printf("Finding runtime by ID: %s", id)
	for _, runtime := range knownRuntimes {
		if runtime.ID == id {
			runtimeDefLog.Printf("Found runtime: %s (%s)", runtime.ID, runtime.Name)
			return runtime
		}
	}
	runtimeDefLog.Printf("Runtime not found: %s", id)
	return nil
}
