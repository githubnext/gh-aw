package workflowcontract

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
)

var (
	compiledAWFCandidateModelsPattern = regexp.MustCompile(`\\"candidateModels\\":\[([^\]]*)\]`)
	compiledAWFModelPattern           = regexp.MustCompile(`\\"([^"\\]+)\\"`)
	detectionAllowedModelsPattern     = regexp.MustCompile(`allowedModels:\s*(\[[^\]]*\])`)
	detectionModelPattern             = regexp.MustCompile(`detectionModel:\s*"([^"]+)"`)
)

type lockWorkflow struct {
	Jobs map[string]struct {
		Steps []struct {
			Name string            `yaml:"name"`
			Run  string            `yaml:"run"`
			Env  map[string]string `yaml:"env"`
			With map[string]any    `yaml:"with"`
		} `yaml:"steps"`
	} `yaml:"jobs"`
}

// TestRoutedSmokeDetectionChecksMatchCompiledRouting verifies that the explicit
// lists passed to the threat-detection smoke assertions (D2) match what the
// workflow compiles to. The detection job has no runner-written copy of the
// agent's routing candidates (it overwrites /tmp/gh-aw/awf-config.json with its
// own config), so the lists stay explicit and this check keeps them in sync with
// the compiled agent AWF routing.candidateModels and the compiled detection model.
func TestRoutedSmokeDetectionChecksMatchCompiledRouting(t *testing.T) {
	t.Parallel()

	lockFiles, err := filepath.Glob(filepath.Join("..", "..", ".github", "workflows", "*.lock.yml"))
	if err != nil {
		t.Fatalf("failed to list lock files: %v", err)
	}
	checked := 0
	for _, lockFile := range lockFiles {
		content, err := os.ReadFile(lockFile)
		if err != nil {
			t.Fatalf("failed to read %s: %v", lockFile, err)
		}
		if !strings.Contains(string(content), "detectionMain(") {
			continue
		}
		checked++
		t.Run(filepath.Base(lockFile), func(t *testing.T) {
			t.Parallel()
			var workflow lockWorkflow
			if err := yaml.Unmarshal(content, &workflow); err != nil {
				t.Fatalf("failed to parse %s: %v", lockFile, err)
			}

			var compiledAllowed []string
			for _, step := range workflow.Jobs["agent"].Steps {
				if match := compiledAWFCandidateModelsPattern.FindStringSubmatch(step.Run); match != nil {
					for _, model := range compiledAWFModelPattern.FindAllStringSubmatch(match[1], -1) {
						compiledAllowed = append(compiledAllowed, servedModelName(model[1]))
					}
				}
			}
			if len(compiledAllowed) == 0 {
				t.Fatalf("agent job has no compiled AWF routing.candidateModels to check the detection assertions against")
			}

			var detectionScript, compiledDetectionModel string
			for _, step := range workflow.Jobs["detection"].Steps {
				if script, ok := step.With["script"].(string); ok && strings.Contains(script, "detectionMain(") {
					detectionScript = script
				}
				if model := step.Env["COPILOT_MODEL"]; model != "" {
					compiledDetectionModel = model
				}
			}
			if detectionScript == "" {
				t.Fatalf("detection job has no detectionMain post-step")
			}

			match := detectionAllowedModelsPattern.FindStringSubmatch(detectionScript)
			if match == nil {
				t.Fatalf("detectionMain post-step has no allowedModels list")
			}
			var detectionAllowed []string
			if err := json.Unmarshal([]byte(match[1]), &detectionAllowed); err != nil {
				t.Fatalf("detectionMain allowedModels %s is not a JSON string array: %v", match[1], err)
			}
			for i, model := range detectionAllowed {
				detectionAllowed[i] = servedModelName(model)
			}
			slices.Sort(compiledAllowed)
			slices.Sort(detectionAllowed)
			if !slices.Equal(compiledAllowed, detectionAllowed) {
				t.Errorf("detectionMain allowedModels %v differs from compiled engine.model-routing.allowed-models %v", detectionAllowed, compiledAllowed)
			}

			modelMatch := detectionModelPattern.FindStringSubmatch(detectionScript)
			if modelMatch == nil {
				t.Fatalf("detectionMain post-step has no detectionModel")
			}
			if compiledDetectionModel == "" {
				t.Fatalf("detection job has no compiled COPILOT_MODEL")
			}
			if servedModelName(modelMatch[1]) != servedModelName(compiledDetectionModel) {
				t.Errorf("detectionMain detectionModel %q differs from compiled detection COPILOT_MODEL %q", modelMatch[1], compiledDetectionModel)
			}
		})
	}
	if checked == 0 {
		t.Fatal("no workflow runs detectionMain; update or remove this contract test")
	}
}

func servedModelName(model string) string {
	model = strings.TrimSpace(model)
	model = strings.TrimPrefix(model, "github-copilot/")
	return strings.TrimPrefix(model, "copilot/")
}
