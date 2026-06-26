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
	normalized := make(map[string]LSPServerConfig, len(servers))
	for key, value := range servers {
		language := strings.TrimSpace(strings.ToLower(key))
		if language == "" {
			lspManagerLog.Printf("Skipping invalid LSP language key: %q", key)
			continue
		}
		config := value
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

type lspInstallSpec struct {
	StepName string
	Command  string
}

var lspInstallSpecs = map[string]lspInstallSpec{
	"bash": {
		StepName: "Install Bash LSP dependencies",
		Command:  "npm install -g bash-language-server",
	},
	"go": {
		StepName: "Install Go LSP dependencies",
		Command:  "go install golang.org/x/tools/gopls@latest",
	},
	"php": {
		StepName: "Install PHP LSP dependencies",
		Command:  "npm install -g intelephense",
	},
	"python": {
		StepName: "Install Python LSP dependencies",
		Command:  "npm install -g pyright",
	},
	"ruby": {
		StepName: "Install Ruby LSP dependencies",
		Command:  "gem install solargraph",
	},
	"rust": {
		StepName: "Install Rust LSP dependencies",
		Command:  "rustup component add rust-analyzer",
	},
	"typescript": {
		StepName: "Install TypeScript LSP dependencies",
		Command:  "npm install -g typescript typescript-language-server",
	},
	"yaml": {
		StepName: "Install YAML LSP dependencies",
		Command:  "npm install -g yaml-language-server",
	},
}
