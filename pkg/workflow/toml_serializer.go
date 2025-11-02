package workflow

import (
	"bytes"
	"fmt"
	"sort"
)

// TOMLConfig represents the top-level TOML configuration structure for Codex
type TOMLConfig struct {
	History    HistoryConfig
	MCPServers map[string]MCPServerConfig
}

// HistoryConfig represents the history configuration section
type HistoryConfig struct {
	Persistence string
}

// MCPServerConfig represents a single MCP server configuration
type MCPServerConfig struct {
	// Common fields
	Command            string
	Args               []string
	Env                map[string]string
	UserAgent          string
	StartupTimeoutSec  int
	ToolTimeoutSec     int
	BearerTokenEnvVar  string
	UseInlineEnv       bool // If true, use inline format for env instead of section format
	
	// HTTP-specific fields
	URL     string
	Headers map[string]string
}

// SerializeToTOML serializes a TOMLConfig to TOML format with proper indentation
// This uses manual formatting to match the expected output format for Codex
func SerializeToTOML(config *TOMLConfig, indent string) (string, error) {
	var buf bytes.Buffer
	
	// Write [history] section
	buf.WriteString(indent + "[history]\n")
	buf.WriteString(indent + "persistence = \"" + config.History.Persistence + "\"\n")
	
	// Get sorted server names for consistent output
	serverNames := make([]string, 0, len(config.MCPServers))
	for name := range config.MCPServers {
		serverNames = append(serverNames, name)
	}
	sort.Strings(serverNames)
	
	// Write each MCP server section
	for _, name := range serverNames {
		server := config.MCPServers[name]
		
		buf.WriteString(indent + "\n")
		// Quote the server name if it contains a hyphen
		if containsHyphen(name) {
			buf.WriteString(indent + "[mcp_servers.\"" + name + "\"]\n")
		} else {
			buf.WriteString(indent + "[mcp_servers." + name + "]\n")
		}
		
		// Write fields in a specific order for consistency
		// Order: user_agent, startup_timeout_sec, tool_timeout_sec, url, bearer_token_env_var, command, args, env
		
		if server.UserAgent != "" {
			buf.WriteString(indent + "user_agent = \"" + server.UserAgent + "\"\n")
		}
		
		if server.StartupTimeoutSec > 0 {
			buf.WriteString(indent + fmt.Sprintf("startup_timeout_sec = %d\n", server.StartupTimeoutSec))
		}
		
		if server.ToolTimeoutSec > 0 {
			buf.WriteString(indent + fmt.Sprintf("tool_timeout_sec = %d\n", server.ToolTimeoutSec))
		}
		
		if server.URL != "" {
			buf.WriteString(indent + "url = \"" + server.URL + "\"\n")
		}
		
		if server.BearerTokenEnvVar != "" {
			buf.WriteString(indent + "bearer_token_env_var = \"" + server.BearerTokenEnvVar + "\"\n")
		}
		
		if server.Command != "" {
			buf.WriteString(indent + "command = \"" + server.Command + "\"\n")
		}
		
		if len(server.Args) > 0 {
			buf.WriteString(indent + "args = [\n")
			for i, arg := range server.Args {
				buf.WriteString(indent + "  \"" + arg + "\"")
				// Only add comma if not the last element
				if i < len(server.Args)-1 {
					buf.WriteString(",")
				}
				buf.WriteString("\n")
			}
			buf.WriteString(indent + "]\n")
		}
		
		if len(server.Env) > 0 {
			if server.UseInlineEnv {
				// Use inline format for env (for safe-outputs and agentic-workflows)
				buf.WriteString(indent + "env = { ")
				envKeys := make([]string, 0, len(server.Env))
				for k := range server.Env {
					envKeys = append(envKeys, k)
				}
				sort.Strings(envKeys)
				
				for i, k := range envKeys {
					if i > 0 {
						buf.WriteString(", ")
					}
					buf.WriteString("\"" + k + "\" = ")
					// Check if value contains toJSON expression that should not be quoted
					v := server.Env[k]
					if k == "GH_AW_SAFE_OUTPUTS_CONFIG" && v == "${{ toJSON(env.GH_AW_SAFE_OUTPUTS_CONFIG) }}" {
						buf.WriteString(v)
					} else {
						buf.WriteString("\"" + v + "\"")
					}
				}
				buf.WriteString(" }\n")
			} else {
				// Use section format for env
				buf.WriteString(indent + "\n")
				// Quote the server name in env section if it contains a hyphen
				if containsHyphen(name) {
					buf.WriteString(indent + "[mcp_servers.\"" + name + "\".env]\n")
				} else {
					buf.WriteString(indent + "[mcp_servers." + name + ".env]\n")
				}
				
				envKeys := make([]string, 0, len(server.Env))
				for k := range server.Env {
					envKeys = append(envKeys, k)
				}
				sort.Strings(envKeys)
				
				for _, k := range envKeys {
					buf.WriteString(indent + k + " = \"" + server.Env[k] + "\"\n")
				}
			}
		}
	}
	
	return buf.String(), nil
}

// containsHyphen checks if a string contains a hyphen
func containsHyphen(s string) bool {
	for _, c := range s {
		if c == '-' {
			return true
		}
	}
	return false
}

// BuildTOMLConfig creates a TOMLConfig structure from workflow data
func BuildTOMLConfig() *TOMLConfig {
	return &TOMLConfig{
		History: HistoryConfig{
			Persistence: "none",
		},
		MCPServers: make(map[string]MCPServerConfig),
	}
}

// AddMCPServer adds an MCP server configuration to the TOMLConfig
func (c *TOMLConfig) AddMCPServer(name string, config MCPServerConfig) {
	c.MCPServers[name] = config
}
