package workflow

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/github/gh-aw/pkg/logger"
)

var githubToolToToolsetLog = logger.New("workflow:github_tool_to_toolset")

//go:embed data/github_tool_to_toolset.json
var githubToolToToolsetJSON []byte

var getGitHubToolToToolsetMap = sync.OnceValues(func() (map[string]string, error) {
	var toolToToolsetMap map[string]string
	if err := json.Unmarshal(githubToolToToolsetJSON, &toolToToolsetMap); err != nil {
		return nil, fmt.Errorf("failed to load GitHub tool to toolset mapping: %w", err)
	}
	githubToolToToolsetLog.Printf("Loaded GitHub tool-to-toolset mapping: %d entries", len(toolToToolsetMap))
	return toolToToolsetMap, nil
})
