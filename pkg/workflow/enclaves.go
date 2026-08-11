package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

const (
	enclaveMCPServerName          = "awf-enclave"
	enclaveMCPUpstreamURL         = "http://awf-enclave-mcp:8080/mcp"
	enclaveMCPCapabilityEnv       = "AWF_ENCLAVE_MCP_CAPABILITY"
	enclaveMCPGatewayContainerEnv = "AWF_ENCLAVE_MCP_GATEWAY_CONTAINER"
	enclaveMCPGatewayEndpointEnv  = "AWF_ENCLAVE_MCP_GATEWAY_ENDPOINT"
	enclaveMCPGatewayIdentityEnv  = "AWF_ENCLAVE_MCP_GATEWAY_IDENTITY"
	enclaveMCPReadinessTimeoutEnv = "AWF_ENCLAVE_MCP_READINESS_TIMEOUT_MS"
	enclaveMCPGatewayRunLabel     = "com.github.gh-aw.mcpg.run"
	enclaveMCPGatewayContainer    = "awmg-mcpg"
	enclaveMCPConnectTimeout      = 120
	enclaveMCPReadinessTimeoutMS  = 120000
	defaultScriptEnclaveTimeout   = 30
	defaultAgentEnclaveTimeout    = 120
)

var enclaveRepoPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]{0,38}/[A-Za-z0-9._-]{1,100}$`)

// EnclavesConfig configures AWF-owned, finite-disclosure private repository executors.
// Each executor type may appear at most once.
type EnclavesConfig []*EnclaveConfig

type EnclaveRepository struct {
	Repo        string `json:"repo"`
	Sensitivity string `json:"sensitivity"`
}

type EnclaveConfig struct {
	Script         *ScriptEnclaveConfig `json:"script,omitempty"`
	Agent          *AgentEnclaveConfig  `json:"agent,omitempty"`
	Repos          []*EnclaveRepository `json:"repos"`
	Runtime        string               `json:"runtime,omitempty"`
	Image          string               `json:"image,omitempty"`
	Timeout        int                  `json:"timeout,omitempty"`
	MemoryLimit    string               `json:"memory-limit,omitempty"`
	CPULimit       string               `json:"cpu-limit,omitempty"`
	PIDsLimit      int                  `json:"pids-limit,omitempty"`
	TmpfsLimit     string               `json:"tmpfs-limit,omitempty"`
	MaxOutputBytes int                  `json:"max-output-bytes,omitempty"`
	MaxInvocations int                  `json:"max-invocations,omitempty"`
}

type ScriptEnclaveConfig struct {
	MaxScriptBytes int `json:"max-script-bytes,omitempty"`
}

type AgentEnclaveConfig struct {
	Engine           string `json:"engine,omitempty"`
	Profile          string `json:"profile,omitempty"`
	Model            string `json:"model,omitempty"`
	MaxTaskBytes     int    `json:"max-task-bytes,omitempty"`
	MaxModelRequests int    `json:"max-model-requests,omitempty"`
	MaxModelTokens   int    `json:"max-model-tokens,omitempty"`
}

// UnmarshalJSON preserves the explicit null marker produced by YAML `script:`.
func (e *EnclaveConfig) UnmarshalJSON(data []byte) error {
	type enclaveAlias EnclaveConfig
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	var decoded enclaveAlias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*e = EnclaveConfig(decoded)
	if script, ok := raw["script"]; ok && string(script) == "null" {
		e.Script = &ScriptEnclaveConfig{}
	}
	return nil
}

func enclavesEnabled(workflowData *WorkflowData) bool {
	return workflowData != nil && len(workflowData.Enclaves) > 0
}

func enabledEnclaveTools(workflowData *WorkflowData) []string {
	var tools []string
	for _, enclave := range workflowData.Enclaves {
		if enclave == nil {
			continue
		}
		if enclave.Script != nil {
			tools = append(tools, "enclave_run_script")
		}
		if enclave.Agent != nil {
			tools = append(tools, "enclave_run_agent")
		}
	}
	return tools
}

func enclaveToolTimeout(workflowData *WorkflowData) int {
	maxTimeout := 0
	for _, enclave := range workflowData.Enclaves {
		if enclave == nil {
			continue
		}
		if enclave.Script != nil {
			timeout := enclave.Timeout
			if timeout == 0 {
				timeout = defaultScriptEnclaveTimeout
			}
			maxTimeout = max(maxTimeout, timeout)
		}
		if enclave.Agent != nil {
			timeout := enclave.Timeout
			if timeout == 0 {
				timeout = defaultAgentEnclaveTimeout
			}
			maxTimeout = max(maxTimeout, timeout)
		}
	}
	if maxTimeout == 0 {
		return 0
	}
	return maxTimeout + 30
}

func validateEnclavesConfig(workflowData *WorkflowData) error {
	if !enclavesEnabled(workflowData) {
		return nil
	}
	if !isAWFNetworkIsolationEnabled(workflowData) {
		return errors.New("enclaves requires AWF network isolation; set sandbox.agent.sudo: false or use sandbox.agent.runtime: docker-sbx")
	}
	if workflowData.ParsedTools != nil &&
		workflowData.ParsedTools.GitHub != nil &&
		workflowData.ParsedTools.GitHub.BoundedQueries != nil {
		return errors.New("enclaves cannot be combined with tools.github.bounded-queries; remove tools.github.bounded-queries to use enclaves")
	}
	seenTypes := make(map[string]struct{}, len(workflowData.Enclaves))
	repositorySensitivities := make(map[string]string)
	for i, enclave := range workflowData.Enclaves {
		if enclave == nil {
			return fmt.Errorf("enclaves[%d] must be an object", i)
		}
		enclaveType, ok := enclaveExecutor(enclave)
		if !ok {
			return fmt.Errorf("enclaves[%d] must contain exactly one of script or agent", i)
		}
		if _, ok := seenTypes[enclaveType]; ok {
			return fmt.Errorf("enclaves contains duplicate executor type %q; each type may appear at most once", enclaveType)
		}
		seenTypes[enclaveType] = struct{}{}
		if enclaveType == "agent" && enclave.Agent.Model == "" {
			return fmt.Errorf("enclaves[%d].agent.model is required", i)
		}
		if len(enclave.Repos) == 0 {
			return fmt.Errorf("enclaves[%d].repos must contain at least one repository", i)
		}
		seenInEnclave := make(map[string]struct{}, len(enclave.Repos))
		for j, repo := range enclave.Repos {
			if repo == nil {
				return fmt.Errorf("enclaves[%d].repos[%d] must be an object", i, j)
			}
			parts := strings.SplitN(repo.Repo, "/", 2)
			if !enclaveRepoPattern.MatchString(repo.Repo) || len(parts) != 2 || parts[1] == "." || parts[1] == ".." || strings.Contains(parts[1], "..") {
				return fmt.Errorf("enclaves[%d].repos[%d].repo must be a bare owner/repository slug (e.g. org/my-repo)", i, j)
			}
			key := strings.ToLower(repo.Repo)
			if _, ok := seenInEnclave[key]; ok {
				return fmt.Errorf("enclaves[%d].repos contains duplicate repository %q", i, repo.Repo)
			}
			seenInEnclave[key] = struct{}{}
			switch repo.Sensitivity {
			case "public", "internal", "confidential", "sealed":
			default:
				return fmt.Errorf("enclaves[%d].repos[%d].sensitivity must be public, internal, confidential, or sealed", i, j)
			}
			if sensitivity, ok := repositorySensitivities[key]; ok && sensitivity != repo.Sensitivity {
				return fmt.Errorf("repository %q must use the same sensitivity across enclave types", repo.Repo)
			}
			repositorySensitivities[key] = repo.Sensitivity
		}
	}
	return nil
}

func enclaveExecutor(enclave *EnclaveConfig) (string, bool) {
	if enclave.Script != nil && enclave.Agent == nil {
		return "script", true
	}
	if enclave.Agent != nil && enclave.Script == nil {
		return "agent", true
	}
	return "", false
}

func buildAWFEnclavesConfig(config EnclavesConfig) []map[string]any {
	if len(config) == 0 {
		return nil
	}
	result := make([]map[string]any, 0, len(config))
	for _, enclave := range config {
		enclaveType, ok := enclaveExecutor(enclave)
		if !ok {
			continue
		}
		values := make(map[string]any)
		repos := make([]map[string]any, 0, len(enclave.Repos))
		for _, repo := range enclave.Repos {
			repos = append(repos, map[string]any{"repo": repo.Repo, "sensitivity": repo.Sensitivity})
		}
		values["repos"] = repos
		addEnclaveString(values, "runtime", enclave.Runtime)
		addEnclaveString(values, "image", enclave.Image)
		addEnclaveInt(values, "timeout", enclave.Timeout)
		addEnclaveString(values, "memoryLimit", enclave.MemoryLimit)
		addEnclaveString(values, "cpuLimit", enclave.CPULimit)
		addEnclaveInt(values, "pidsLimit", enclave.PIDsLimit)
		addEnclaveString(values, "tmpfsLimit", enclave.TmpfsLimit)
		addEnclaveInt(values, "maxOutputBytes", enclave.MaxOutputBytes)
		addEnclaveInt(values, "maxInvocations", enclave.MaxInvocations)
		if enclaveType == "script" {
			script := make(map[string]any)
			addEnclaveInt(script, "maxScriptBytes", enclave.Script.MaxScriptBytes)
			values["script"] = script
		} else {
			agent := make(map[string]any)
			addEnclaveString(agent, "engine", enclave.Agent.Engine)
			addEnclaveString(agent, "profile", enclave.Agent.Profile)
			addEnclaveString(agent, "model", enclave.Agent.Model)
			addEnclaveInt(agent, "maxTaskBytes", enclave.Agent.MaxTaskBytes)
			addEnclaveInt(agent, "maxModelRequests", enclave.Agent.MaxModelRequests)
			addEnclaveInt(agent, "maxModelTokens", enclave.Agent.MaxModelTokens)
			values["agent"] = agent
		}
		result = append(result, values)
	}
	return result
}

func addEnclaveString(values map[string]any, key, value string) {
	if value != "" {
		values[key] = value
	}
}

func addEnclaveInt(values map[string]any, key string, value int) {
	if value != 0 {
		values[key] = value
	}
}

func writeEnclaveMCPJSON(yaml *strings.Builder, workflowData *WorkflowData, isLast bool) {
	fmt.Fprintf(yaml, "              %q: {\n", enclaveMCPServerName)
	yaml.WriteString("                \"type\": \"http\",\n")
	fmt.Fprintf(yaml, "                \"url\": %q,\n", enclaveMCPUpstreamURL)
	fmt.Fprintf(yaml, "                \"headers\": {\"Authorization\": \"Bearer \\${%s}\"},\n", enclaveMCPCapabilityEnv)
	fmt.Fprintf(yaml, "                \"tools\": [")
	for i, tool := range enabledEnclaveTools(workflowData) {
		if i > 0 {
			yaml.WriteString(", ")
		}
		fmt.Fprintf(yaml, "%q", tool)
	}
	yaml.WriteString("],\n")
	fmt.Fprintf(yaml, "                \"connectTimeout\": %d,\n", enclaveMCPConnectTimeout)
	fmt.Fprintf(yaml, "                \"toolTimeout\": %d\n", enclaveToolTimeout(workflowData))
	yaml.WriteString("              }")
	if !isLast {
		yaml.WriteString(",")
	}
	yaml.WriteString("\n")
}

func writeEnclaveMCPTOML(yaml *strings.Builder, workflowData *WorkflowData) {
	yaml.WriteString("          \n")
	fmt.Fprintf(yaml, "          [mcp_servers.%s]\n", enclaveMCPServerName)
	yaml.WriteString("          type = \"http\"\n")
	fmt.Fprintf(yaml, "          url = %q\n", enclaveMCPUpstreamURL)
	fmt.Fprintf(yaml, "          headers = { Authorization = \"Bearer $%s\" }\n", enclaveMCPCapabilityEnv)
	fmt.Fprintf(yaml, "          tools = [")
	for i, tool := range enabledEnclaveTools(workflowData) {
		if i > 0 {
			yaml.WriteString(", ")
		}
		fmt.Fprintf(yaml, "%q", tool)
	}
	yaml.WriteString("]\n")
	fmt.Fprintf(yaml, "          connectTimeout = %d\n", enclaveMCPConnectTimeout)
	fmt.Fprintf(yaml, "          toolTimeout = %d\n", enclaveToolTimeout(workflowData))
}
