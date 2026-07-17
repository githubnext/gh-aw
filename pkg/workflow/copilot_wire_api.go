package workflow

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
)

//go:embed data/wire_api_models.json
var wireAPIModelsJSON []byte

// wireAPIValue is one of the accepted values for COPILOT_PROVIDER_WIRE_API.
const (
	wireAPIResponses   = "responses"
	wireAPICompletions = "completions"
)

var (
	wireAPIModelsOnce sync.Once
	wireAPIModelsData map[string]string // modelName → wire_api (github-copilot provider)
	wireAPIModelsErr  error
)

type wireAPIModelsFile struct {
	GithubCopilot map[string]string `json:"github-copilot"`
}

func loadWireAPIModels() (map[string]string, error) {
	wireAPIModelsOnce.Do(func() {
		var data wireAPIModelsFile
		if err := json.Unmarshal(wireAPIModelsJSON, &data); err != nil {
			wireAPIModelsErr = fmt.Errorf("BUG: workflow: failed to parse embedded wire_api_models.json: %w", err)
			return
		}
		wireAPIModelsData = data.GithubCopilot
		if wireAPIModelsData == nil {
			wireAPIModelsData = make(map[string]string)
		}
	})
	return wireAPIModelsData, wireAPIModelsErr
}

// copilotWireAPIForModel returns the wire_api value for the given github-copilot model name.
// Returns an empty string when the model is not in the catalog or has no wire_api entry.
// The returned value is one of "responses" or "completions".
func copilotWireAPIForModel(modelName string) string {
	if modelName == "" {
		return ""
	}
	models, err := loadWireAPIModels()
	if err != nil || len(models) == 0 {
		return ""
	}
	// Case-insensitive lookup.
	normalized := strings.ToLower(strings.TrimSpace(modelName))
	for k, v := range models {
		if strings.ToLower(k) == normalized {
			return v
		}
	}
	return ""
}

// buildCopilotWireAPIResolutionScript returns a shell fragment that auto-configures
// COPILOT_PROVIDER_WIRE_API at runtime based on the $COPILOT_MODEL env var value.
//
// The script is a no-op when COPILOT_PROVIDER_WIRE_API is already set (user override).
// It uses a shell case statement built from the compile-time wire_api catalog so that
// no runtime JSON parsing is required.
//
// The script is intended to be emitted in the PathSetup (pre-engine shell) so that
// it runs before the Copilot CLI is launched, handling the case where the model name
// is only known at runtime (e.g. from a GitHub org variable).
func buildCopilotWireAPIResolutionScript() string {
	models, err := loadWireAPIModels()
	if err != nil || len(models) == 0 {
		return ""
	}

	// Collect models grouped by wire_api value.
	byAPI := make(map[string][]string)
	for model, api := range models {
		byAPI[api] = append(byAPI[api], model)
	}

	// Only emit case arms for values that need explicit configuration.
	// completions is the Copilot CLI default for custom providers, so we always
	// emit it to make the selection explicit and forward-compatible.
	var arms []string
	for _, api := range []string{wireAPIResponses, wireAPICompletions} {
		modelList, ok := byAPI[api]
		if !ok || len(modelList) == 0 {
			continue
		}
		sort.Strings(modelList)
		pattern := strings.Join(modelList, "|")
		arms = append(arms, fmt.Sprintf("    %s)\n      export COPILOT_PROVIDER_WIRE_API=%s\n      ;;", pattern, api))
	}

	if len(arms) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("if [ -z \"${COPILOT_PROVIDER_WIRE_API:-}\" ]; then\n")
	b.WriteString("  case \"${COPILOT_MODEL:-}\" in\n")
	for _, arm := range arms {
		b.WriteString(arm)
		b.WriteString("\n")
	}
	b.WriteString("  esac\n")
	b.WriteString("fi\n")
	return b.String()
}
