package workflow

import (
	"fmt"
	"strings"
	"testing"
)

func TestExternalDetectorInheritsOpenAIBaseURL_Debug(t *testing.T) {
	compiler := NewCompiler()
	data := &WorkflowData{
		AI: "codex",
		EngineConfig: &EngineConfig{
			ID: "codex",
			Env: map[string]string{
				"OPENAI_BASE_URL": "https://llm-router.internal.example.com/v1",
			},
		},
		SafeOutputs: &SafeOutputsConfig{
			ThreatDetection: &ThreatDetectionConfig{
				EngineConfig: &EngineConfig{
					ID: "codex",
					Env: map[string]string{
						"CUSTOM_FLAG": "1",
					},
				},
			},
		},
	}

	steps := compiler.buildExternalDetectorExecutionStep(data)
	stepsContent := strings.Join(steps, "")
	// Print the relevant fragment around apiProxy
	idx := strings.Index(stepsContent, "apiProxy")
	if idx != -1 {
		start := idx - 20
		if start < 0 {
			start = 0
		}
		end := idx + 200
		if end > len(stepsContent) {
			end = len(stepsContent)
		}
		fmt.Printf("FRAGMENT: %s\n", stepsContent[start:end])
	}
}
