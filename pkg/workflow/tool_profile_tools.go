package workflow

import (
	"encoding/json"
	"fmt"
	"strings"
)

func (c *Compiler) mergeToolProfileDefaults(topTools, mcpServers map[string]any, includedTools, profileTools string) (map[string]any, error) {
	tools, err := c.mergeToolsAndMCPServers(topTools, mcpServers, includedTools)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(profileTools) == "" {
		return tools, nil
	}
	var defaults map[string]any
	if err := json.Unmarshal([]byte(profileTools), &defaults); err != nil {
		return nil, fmt.Errorf("failed to parse tool profile defaults: %w", err)
	}
	if _, hasExplicitBash := tools["bash"]; hasExplicitBash {
		delete(defaults, "bash")
	}
	defaultsJSON, err := json.Marshal(defaults)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize tool profile defaults: %w", err)
	}
	return c.MergeTools(tools, string(defaultsJSON))
}
