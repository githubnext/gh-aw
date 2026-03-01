package workflow

import (
	"github.com/github/gh-aw/pkg/logger"
)

var serenaConfigLog = logger.New("workflow:mcp_serena_config")

// isSerenaInLocalMode checks if Serena tool is configured with local mode
func isSerenaInLocalMode(tools *ToolsConfig) bool {
	if tools == nil || tools.Serena == nil {
		return false
	}
	serenaConfigLog.Printf("Serena tool mode: %s", tools.Serena.Mode)
	return tools.Serena.Mode == "local"
}
