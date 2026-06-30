package workflow

import (
	"fmt"

	"github.com/github/gh-aw/pkg/logger"
)

var runnerConfigLog = logger.New("workflow:runner_config")

// extractRunnerConfig extracts runner topology configuration from frontmatter.
// Returns nil when no runner configuration is present.
func extractRunnerConfig(frontmatter map[string]any) *RunnerConfig {
	runner, exists := frontmatter["runner"]
	if !exists {
		return nil
	}

	runnerObj, ok := runner.(map[string]any)
	if !ok {
		runnerConfigLog.Printf("runner field has unexpected type %T, expected object", runner)
		return nil
	}

	config := &RunnerConfig{}
	if topology, ok := runnerObj["topology"].(string); ok {
		config.Topology = topology
		runnerConfigLog.Printf("Runner topology: %s", topology)
	}

	if config.Topology == "" {
		return nil
	}

	return config
}

// validateRunnerConfig validates the runner topology configuration.
// It returns an error if the topology value is not recognized.
func validateRunnerConfig(config *RunnerConfig) error {
	if config == nil {
		return nil
	}

	switch config.Topology {
	case RunnerTopologyArcDind:
		return nil
	case "":
		return nil
	default:
		return fmt.Errorf("unsupported runner.topology value %q; supported values: %q", config.Topology, RunnerTopologyArcDind)
	}
}
