package workflow

import (
	"fmt"
	"maps"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var lspManagerLog = logger.New("workflow:lsp_manager")

// LSPServerConfig defines a single language server entry under top-level frontmatter "lsp:".
type LSPServerConfig struct {
	Command        string            `json:"command,omitempty"`
	Args           []string          `json:"args,omitempty"`
	FileExtensions map[string]string `json:"fileExtensions,omitempty"`
}

// LSPManager handles LSP configuration normalization, validation, and generation.
type LSPManager struct {
	servers map[string]LSPServerConfig
}

func NewLSPManager(servers map[string]LSPServerConfig) *LSPManager {
	// Sort keys for deterministic normalization order so that when two keys
	// collapse to the same lowercase value (e.g. "TypeScript" and "typescript"),
	// the lexicographically first original key always wins and the duplicate is
	// logged rather than silently lost.
	keys := make([]string, 0, len(servers))
	for k := range servers {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	normalized := make(map[string]LSPServerConfig, len(servers))
	for _, key := range keys {
		language := strings.TrimSpace(strings.ToLower(key))
		if language == "" {
			lspManagerLog.Printf("Skipping invalid LSP language key: %q", key)
			continue
		}
		if _, exists := normalized[language]; exists {
			lspManagerLog.Printf("Duplicate LSP language key %q (normalizes to %q): entry ignored", key, language)
			continue
		}
		config := servers[key]
		config.Command = strings.TrimSpace(config.Command)
		normalized[language] = config
	}
	return &LSPManager{servers: normalized}
}

func (m *LSPManager) HasServers() bool {
	return m != nil && len(m.servers) > 0
}

func (m *LSPManager) Validate() error {
	if !m.HasServers() {
		return nil
	}
	for language, config := range m.servers {
		if config.Command == "" {
			return fmt.Errorf("lsp.%s.command is required", language)
		}
		if len(config.FileExtensions) == 0 {
			return fmt.Errorf("lsp.%s.fileExtensions must define at least one file extension mapping", language)
		}
	}
	return nil
}

func (m *LSPManager) CopilotLSPServers() map[string]LSPServerConfig {
	if !m.HasServers() {
		return nil
	}
	result := make(map[string]LSPServerConfig, len(m.servers))
	maps.Copy(result, m.servers)
	return result
}

func (m *LSPManager) GenerateInstallSteps() []GitHubActionStep {
	if !m.HasServers() {
		return nil
	}

	langs := make([]string, 0, len(m.servers))
	for language := range m.servers {
		langs = append(langs, language)
	}
	sort.Strings(langs)

	var steps []GitHubActionStep
	for _, language := range langs {
		spec, ok := lspInstallSpecs[language]
		if !ok {
			continue
		}
		step := GitHubActionStep{
			"      - name: " + spec.StepName,
			"        run: " + spec.Command,
			"        timeout-minutes: 10",
		}
		steps = append(steps, step)
	}

	return steps
}

// RuntimeRequirements returns the set of runtime requirements for all configured LSP
// servers. These are returned as [RuntimeRequirement] values so that the caller can
// feed them into the standard runtime manager (DetectRuntimeRequirements /
// GenerateRuntimeSetupSteps), which emits properly SHA-pinned setup actions.
//
// Only languages that have a matching entry in knownRuntimes are included; languages
// whose runtime is not tracked by the runtime manager (e.g. "rust") are silently
// skipped — their install commands still appear in GenerateInstallSteps.
//
// Note: Node.js-based LSP servers (bash, php, python, typescript, yaml) map to the
// "node" runtime, but the Copilot engine already sets up Node.js unconditionally via
// BuildNpmEngineInstallStepsWithAWF. Returning "node" here is correct and harmless:
// DetectRuntimeRequirements deduplicates by runtime ID, so at most one Node.js setup
// step is emitted regardless of how many node-based LSP servers are configured.
func (m *LSPManager) RuntimeRequirements() []RuntimeRequirement {
	if !m.HasServers() {
		return nil
	}

	seen := make(map[string]bool)
	var result []RuntimeRequirement

	langs := make([]string, 0, len(m.servers))
	for language := range m.servers {
		langs = append(langs, language)
	}
	sort.Strings(langs)

	for _, language := range langs {
		spec, ok := lspInstallSpecs[language]
		if !ok || spec.RuntimeID == "" {
			continue
		}
		if seen[spec.RuntimeID] {
			continue
		}
		seen[spec.RuntimeID] = true
		runtime := findRuntimeByID(spec.RuntimeID)
		if runtime == nil {
			lspManagerLog.Printf("LSP language %q specifies unknown runtime ID %q; skipping runtime requirement", language, spec.RuntimeID)
			continue
		}
		result = append(result, RuntimeRequirement{
			Runtime:  runtime,
			Version:  "",
			Cooldown: true,
		})
	}
	return result
}

type lspInstallSpec struct {
	StepName  string
	Command   string
	RuntimeID string // runtime manager ID for the runtime needed to run this LSP server
}

var lspInstallSpecs = map[string]lspInstallSpec{
	"bash": {
		StepName:  "Install Bash LSP dependencies",
		Command:   "npm install -g --ignore-scripts bash-language-server",
		RuntimeID: "node",
	},
	"go": {
		StepName:  "Install Go LSP dependencies",
		Command:   "go install golang.org/x/tools/gopls@latest",
		RuntimeID: "go",
	},
	"php": {
		StepName:  "Install PHP LSP dependencies",
		Command:   "npm install -g --ignore-scripts intelephense",
		RuntimeID: "node",
	},
	"python": {
		StepName:  "Install Python LSP dependencies",
		Command:   "npm install -g --ignore-scripts pyright",
		RuntimeID: "node",
	},
	"ruby": {
		StepName:  "Install Ruby LSP dependencies",
		Command:   "gem install solargraph",
		RuntimeID: "ruby",
	},
	"rust": {
		StepName:  "Install Rust LSP dependencies",
		Command:   "rustup component add rust-analyzer",
		RuntimeID: "", // Rust is not in knownRuntimes; runtime setup is done via rustup
	},
	"typescript": {
		StepName:  "Install TypeScript LSP dependencies",
		Command:   "npm install -g --ignore-scripts typescript typescript-language-server",
		RuntimeID: "node",
	},
	"yaml": {
		StepName:  "Install YAML LSP dependencies",
		Command:   "npm install -g --ignore-scripts yaml-language-server",
		RuntimeID: "node",
	},
}
