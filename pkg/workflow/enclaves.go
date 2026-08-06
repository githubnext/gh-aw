package workflow

import (
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
type EnclavesConfig struct {
	Enabled      bool                    `json:"enabled,omitempty"`
	PrivateRepos []*EnclavePrivateRepo   `json:"private-repos,omitempty"`
	Executors    *EnclaveExecutorsConfig `json:"executors,omitempty"`
}

type EnclavePrivateRepo struct {
	Repo        string `json:"repo"`
	Sensitivity string `json:"sensitivity"`
}

type EnclaveExecutorsConfig struct {
	Script *ScriptEnclaveExecutorConfig `json:"script,omitempty"`
	Agent  *AgentEnclaveExecutorConfig  `json:"agent,omitempty"`
}

type ScriptEnclaveExecutorConfig struct {
	Enabled        bool   `json:"enabled,omitempty"`
	Runtime        string `json:"runtime,omitempty"`
	Image          string `json:"image,omitempty"`
	Network        string `json:"network,omitempty"`
	Interpreter    string `json:"interpreter,omitempty"`
	Timeout        int    `json:"timeout,omitempty"`
	MemoryLimit    string `json:"memory-limit,omitempty"`
	CPULimit       string `json:"cpu-limit,omitempty"`
	PIDsLimit      int    `json:"pids-limit,omitempty"`
	TmpfsLimit     string `json:"tmpfs-limit,omitempty"`
	MaxOutputBytes int    `json:"max-output-bytes,omitempty"`
	MaxScriptBytes int    `json:"max-script-bytes,omitempty"`
	MaxInvocations int    `json:"max-invocations,omitempty"`
}

type AgentEnclaveExecutorConfig struct {
	Enabled          bool   `json:"enabled,omitempty"`
	Runtime          string `json:"runtime,omitempty"`
	Image            string `json:"image,omitempty"`
	Network          string `json:"network,omitempty"`
	Engine           string `json:"engine,omitempty"`
	Profile          string `json:"profile,omitempty"`
	Model            string `json:"model,omitempty"`
	Timeout          int    `json:"timeout,omitempty"`
	MemoryLimit      string `json:"memory-limit,omitempty"`
	CPULimit         string `json:"cpu-limit,omitempty"`
	PIDsLimit        int    `json:"pids-limit,omitempty"`
	TmpfsLimit       string `json:"tmpfs-limit,omitempty"`
	MaxOutputBytes   int    `json:"max-output-bytes,omitempty"`
	MaxTaskBytes     int    `json:"max-task-bytes,omitempty"`
	MaxInvocations   int    `json:"max-invocations,omitempty"`
	MaxModelRequests int    `json:"max-model-requests,omitempty"`
	MaxModelTokens   int    `json:"max-model-tokens,omitempty"`
}

func enclavesEnabled(workflowData *WorkflowData) bool {
	return workflowData != nil && workflowData.Enclaves != nil && workflowData.Enclaves.Enabled
}

func enabledEnclaveTools(workflowData *WorkflowData) []string {
	if !enclavesEnabled(workflowData) || workflowData.Enclaves.Executors == nil {
		return nil
	}
	var tools []string
	if script := workflowData.Enclaves.Executors.Script; script != nil && script.Enabled {
		tools = append(tools, "enclave_run_script")
	}
	if agent := workflowData.Enclaves.Executors.Agent; agent != nil && agent.Enabled {
		tools = append(tools, "enclave_run_agent")
	}
	return tools
}

func enclaveToolTimeout(workflowData *WorkflowData) int {
	maxTimeout := 0
	if !enclavesEnabled(workflowData) || workflowData.Enclaves.Executors == nil {
		return 0
	}
	if script := workflowData.Enclaves.Executors.Script; script != nil && script.Enabled {
		timeout := script.Timeout
		if timeout == 0 {
			timeout = defaultScriptEnclaveTimeout
		}
		maxTimeout = max(maxTimeout, timeout)
	}
	if agent := workflowData.Enclaves.Executors.Agent; agent != nil && agent.Enabled {
		timeout := agent.Timeout
		if timeout == 0 {
			timeout = defaultAgentEnclaveTimeout
		}
		maxTimeout = max(maxTimeout, timeout)
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
		return errors.New("enclaves cannot be combined with tools.github.bounded-queries")
	}
	config := workflowData.Enclaves
	if len(config.PrivateRepos) == 0 {
		return errors.New("enclaves.private-repos must contain at least one repository when enclaves is enabled")
	}
	seen := make(map[string]struct{}, len(config.PrivateRepos))
	for i, repo := range config.PrivateRepos {
		if repo == nil {
			return fmt.Errorf("enclaves.private-repos[%d] must be an object", i)
		}
		parts := strings.SplitN(repo.Repo, "/", 2)
		if !enclaveRepoPattern.MatchString(repo.Repo) || len(parts) != 2 || parts[1] == "." || parts[1] == ".." || strings.Contains(parts[1], "..") {
			return fmt.Errorf("enclaves.private-repos[%d].repo must be a bare owner/repository slug", i)
		}

		key := strings.ToLower(repo.Repo)
		if _, ok := seen[key]; ok {
			return fmt.Errorf("enclaves.private-repos contains duplicate repository %q", repo.Repo)
		}
		seen[key] = struct{}{}
		switch repo.Sensitivity {
		case "public", "internal", "confidential", "sealed":
		default:
			return fmt.Errorf("enclaves.private-repos[%d].sensitivity must be public, internal, confidential, or sealed", i)
		}
	}
	if len(enabledEnclaveTools(workflowData)) == 0 {
		return errors.New("enclaves.executors must enable at least one of script or agent")
	}
	if agent := config.Executors.Agent; agent != nil && agent.Enabled && agent.Model == "" {
		return errors.New("enclaves.executors.agent.model is required when the agent executor is enabled")
	}
	return nil
}

func buildAWFEnclavesConfig(config *EnclavesConfig) map[string]any {
	if config == nil || !config.Enabled {
		return nil
	}
	result := map[string]any{"enabled": true}
	privateRepos := make([]map[string]any, 0, len(config.PrivateRepos))
	for _, repo := range config.PrivateRepos {
		privateRepos = append(privateRepos, map[string]any{
			"repo": repo.Repo, "sensitivity": repo.Sensitivity,
		})
	}
	result["privateRepos"] = privateRepos
	executors := make(map[string]any)
	if config.Executors != nil {
		if script := config.Executors.Script; script != nil {
			values := map[string]any{"enabled": script.Enabled}
			addEnclaveString(values, "runtime", script.Runtime)
			addEnclaveString(values, "image", script.Image)
			addEnclaveString(values, "network", script.Network)
			addEnclaveString(values, "interpreter", script.Interpreter)
			addEnclaveInt(values, "timeout", script.Timeout)
			addEnclaveString(values, "memoryLimit", script.MemoryLimit)
			addEnclaveString(values, "cpuLimit", script.CPULimit)
			addEnclaveInt(values, "pidsLimit", script.PIDsLimit)
			addEnclaveString(values, "tmpfsLimit", script.TmpfsLimit)
			addEnclaveInt(values, "maxOutputBytes", script.MaxOutputBytes)
			addEnclaveInt(values, "maxScriptBytes", script.MaxScriptBytes)
			addEnclaveInt(values, "maxInvocations", script.MaxInvocations)
			executors["script"] = values
		}
		if agent := config.Executors.Agent; agent != nil {
			values := map[string]any{"enabled": agent.Enabled}
			addEnclaveString(values, "runtime", agent.Runtime)
			addEnclaveString(values, "image", agent.Image)
			addEnclaveString(values, "network", agent.Network)
			addEnclaveString(values, "engine", agent.Engine)
			addEnclaveString(values, "profile", agent.Profile)
			addEnclaveString(values, "model", agent.Model)
			addEnclaveInt(values, "timeout", agent.Timeout)
			addEnclaveString(values, "memoryLimit", agent.MemoryLimit)
			addEnclaveString(values, "cpuLimit", agent.CPULimit)
			addEnclaveInt(values, "pidsLimit", agent.PIDsLimit)
			addEnclaveString(values, "tmpfsLimit", agent.TmpfsLimit)
			addEnclaveInt(values, "maxOutputBytes", agent.MaxOutputBytes)
			addEnclaveInt(values, "maxTaskBytes", agent.MaxTaskBytes)
			addEnclaveInt(values, "maxInvocations", agent.MaxInvocations)
			addEnclaveInt(values, "maxModelRequests", agent.MaxModelRequests)
			addEnclaveInt(values, "maxModelTokens", agent.MaxModelTokens)
			executors["agent"] = values
		}
	}
	result["executors"] = executors
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
