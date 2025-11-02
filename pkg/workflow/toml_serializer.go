package workflow

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/BurntSushi/toml"
)

// TOMLConfig represents the top-level TOML configuration structure for Codex
type TOMLConfig struct {
	History    HistoryConfig              `toml:"history"`
	MCPServers map[string]MCPServerConfig `toml:"mcp_servers"`
}

// HistoryConfig represents the history configuration section
type HistoryConfig struct {
	Persistence string `toml:"persistence"`
}

// MCPServerConfig represents a single MCP server configuration
type MCPServerConfig struct {
	// Common fields - using toml struct tags with omitempty
	UserAgent         string            `toml:"user_agent,omitempty"`
	StartupTimeoutSec int               `toml:"startup_timeout_sec,omitempty"`
	ToolTimeoutSec    int               `toml:"tool_timeout_sec,omitempty"`
	URL               string            `toml:"url,omitempty"`
	BearerTokenEnvVar string            `toml:"bearer_token_env_var,omitempty"`
	Command           string            `toml:"command,omitempty"`
	Args              []string          `toml:"args,omitempty"`
	Env               map[string]string `toml:"env,omitempty"`

	// Internal field not serialized to TOML
	UseInlineEnv bool `toml:"-"` // If true, use inline format for env instead of section format

	// HTTP-specific fields
	Headers map[string]string `toml:"headers,omitempty"`
}

// SerializeToTOML serializes a TOMLConfig to TOML format with proper indentation
// Uses the BurntSushi/toml encoder with custom post-processing for formatting
func SerializeToTOML(config *TOMLConfig, indent string) (string, error) {
	// First, handle servers that need inline env formatting
	// We need to separate them and handle them specially after encoding
	inlineEnvServers := make(map[string]MCPServerConfig)
	regularServers := make(map[string]MCPServerConfig)

	for name, server := range config.MCPServers {
		if server.UseInlineEnv && len(server.Env) > 0 {
			inlineEnvServers[name] = server
		} else {
			regularServers[name] = server
		}
	}

	var buf bytes.Buffer

	// Encode the regular config using TOML encoder
	regularConfig := &TOMLConfig{
		History:    config.History,
		MCPServers: regularServers,
	}

	encoder := toml.NewEncoder(&buf)
	if err := encoder.Encode(regularConfig); err != nil {
		return "", fmt.Errorf("failed to encode TOML: %w", err)
	}

	output := buf.String()

	// Post-process the TOML output to fix formatting
	output = postProcessTOML(output)

	// Post-process to add servers with inline env
	if len(inlineEnvServers) > 0 {
		output = addInlineEnvServers(output, inlineEnvServers)
	}

	// Apply indentation if needed
	if indent != "" {
		output = applyIndentation(output, indent)
	}

	// Remove trailing blank lines
	output = strings.TrimRight(output, "\n") + "\n"

	return output, nil
}

// addInlineEnvServers adds servers with inline env formatting to the TOML output
func addInlineEnvServers(output string, servers map[string]MCPServerConfig) string {
	// Sort server names for consistent output
	names := make([]string, 0, len(servers))
	for name := range servers {
		names = append(names, name)
	}
	sort.Strings(names)

	var additions bytes.Buffer
	for _, name := range names {
		server := servers[name]
		additions.WriteString("\n")

		// Quote the server name if it contains a hyphen
		if containsHyphen(name) {
			additions.WriteString(fmt.Sprintf("[mcp_servers.\"%s\"]\n", name))
		} else {
			additions.WriteString(fmt.Sprintf("[mcp_servers.%s]\n", name))
		}

		// Write non-env fields
		if server.Command != "" {
			additions.WriteString(fmt.Sprintf("command = \"%s\"\n", server.Command))
		}
		if len(server.Args) > 0 {
			additions.WriteString("args = [\n")
			for i, arg := range server.Args {
				additions.WriteString(fmt.Sprintf("  \"%s\"", arg))
				if i < len(server.Args)-1 {
					additions.WriteString(",")
				}
				additions.WriteString("\n")
			}
			additions.WriteString("]\n")
		}

		// Write inline env
		if len(server.Env) > 0 {
			additions.WriteString("env = { ")
			envKeys := make([]string, 0, len(server.Env))
			for k := range server.Env {
				envKeys = append(envKeys, k)
			}
			sort.Strings(envKeys)

			for i, k := range envKeys {
				if i > 0 {
					additions.WriteString(", ")
				}
				additions.WriteString(fmt.Sprintf("\"%s\" = ", k))
				v := server.Env[k]
				additions.WriteString(fmt.Sprintf("\"%s\"", v))
			}
			additions.WriteString(" }\n")
		}
	}

	return output + additions.String()
}

// applyIndentation adds indentation to each non-empty line
func applyIndentation(output string, indent string) string {
	lines := strings.Split(output, "\n")
	var result bytes.Buffer
	for _, line := range lines {
		if len(line) > 0 {
			result.WriteString(indent + line + "\n")
		} else {
			result.WriteString("\n")
		}
	}
	return result.String()
}

// postProcessTOML fixes formatting issues from the TOML encoder
// Also strips the encoder's indentation so we can apply our own later
func postProcessTOML(output string) string {
	lines := strings.Split(output, "\n")
	var result []string
	
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		
		// Skip the [mcp_servers] header added by encoder
		if trimmed == "[mcp_servers]" {
			continue
		}
		
		// Strip encoder's indentation - we'll apply our own later
		line = trimmed
		
		// Add quotes around hyphenated server names in section headers
		if strings.HasPrefix(line, "[mcp_servers.") && !strings.Contains(line, `"`) {
			// Check if the server name contains a hyphen
			start := strings.Index(line, "[mcp_servers.") + len("[mcp_servers.")
			end := strings.Index(line, "]")
			if end > start {
				serverName := line[start:end]
				if containsHyphen(serverName) {
					line = `[mcp_servers."` + serverName + `"]`
				}
			}
		}
		
		// Add blank line before env subsections
		if strings.Contains(line, "[mcp_servers.") && strings.Contains(line, ".env]") {
			result = append(result, "")
		}
		
		// Handle array formatting - convert compact arrays to multi-line
		if strings.Contains(line, "args = [") && strings.Contains(line, "]") {
			// Compact array on one line
			reformatted := reformatCompactArray(line)
			result = append(result, reformatted...)
			continue
		}
		
		result = append(result, line)
	}
	
	return strings.Join(result, "\n")
}

// reformatCompactArray converts a compact array to multi-line format
// Input line should already have indentation stripped
func reformatCompactArray(line string) []string {
	// Extract array content
	start := strings.Index(line, "[")
	end := strings.LastIndex(line, "]")
	if start == -1 || end == -1 {
		return []string{line}
	}
	
	content := line[start+1 : end]
	elements := parseArrayElements(content)
	
	if len(elements) == 0 {
		return []string{line}
	}
	
	// Reformat to multi-line without indentation (will be added later)
	var result []string
	result = append(result, "args = [")
	for i, elem := range elements {
		if i < len(elements)-1 {
			result = append(result, "  "+elem+",")
		} else {
			result = append(result, "  "+elem)
		}
	}
	result = append(result, "]")
	return result
}

// parseArrayElements parses array elements from a compact TOML array string
func parseArrayElements(content string) []string {
	var elements []string
	var current bytes.Buffer
	inQuotes := false
	
	for i := 0; i < len(content); i++ {
		ch := content[i]
		
		if ch == '"' {
			inQuotes = !inQuotes
			current.WriteByte(ch)
		} else if ch == ',' && !inQuotes {
			elem := strings.TrimSpace(current.String())
			if elem != "" {
				elements = append(elements, elem)
			}
			current.Reset()
		} else if !unicode.IsSpace(rune(ch)) || inQuotes {
			current.WriteByte(ch)
		}
	}
	
	// Add last element
	elem := strings.TrimSpace(current.String())
	if elem != "" {
		elements = append(elements, elem)
	}
	
	return elements
}

// getIndent extracts the indentation from a line
func getIndent(line string) string {
	for i, ch := range line {
		if ch != ' ' && ch != '\t' {
			return line[:i]
		}
	}
	return line
}

// containsHyphen checks if a string contains a hyphen
func containsHyphen(s string) bool {
	return strings.Contains(s, "-")
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
