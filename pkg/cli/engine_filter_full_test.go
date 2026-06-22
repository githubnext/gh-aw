//go:build !integration

package cli_test

import (
    "encoding/json"
    "os"
    "path/filepath"
    "testing"
    
    "github.com/github/gh-aw/pkg/constants"
    "github.com/github/gh-aw/pkg/workflow"
)

func TestEngineFilterWithFile(t *testing.T) {
    // Simulate what the orchestrator does for each run
    cases := []struct {
        name           string
        awInfoContent  string
        filterEngine   string
        expectMatch    bool
    }{
        {
            name:          "copilot run should NOT match claude filter",
            awInfoContent: `{"engine_id": "copilot"}`,
            filterEngine:  "claude",
            expectMatch:   false,
        },
        {
            name:          "claude run should match claude filter",
            awInfoContent: `{"engine_id": "claude"}`,
            filterEngine:  "claude",
            expectMatch:   true,
        },
        {
            name:          "copilot run should match copilot filter",
            awInfoContent: `{"engine_id": "copilot"}`,
            filterEngine:  "copilot",
            expectMatch:   true,
        },
        {
            name:          "missing aw_info.json should NOT match any filter",
            awInfoContent: "",
            filterEngine:  "claude",
            expectMatch:   false,
        },
        {
            name:          "empty engine_id should NOT match any filter",
            awInfoContent: `{"engine_id": ""}`,
            filterEngine:  "claude",
            expectMatch:   false,
        },
    }
    
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            // Create a temp directory
            tmpDir, err := os.MkdirTemp("", "engine-filter-test-*")
            if err != nil {
                t.Fatal("Failed to create temp dir:", err)
            }
            defer os.RemoveAll(tmpDir)
            
            // Create aw_info.json if content is provided
            awInfoPath := filepath.Join(tmpDir, "aw_info.json")
            if tc.awInfoContent != "" {
                if err := os.WriteFile(awInfoPath, []byte(tc.awInfoContent), 0644); err != nil {
                    t.Fatal("Failed to write aw_info.json:", err)
                }
            }
            
            // Simulate the engine filter logic from logs_orchestrator.go
            engine := tc.filterEngine
            detectedEngine := extractEngineFromAwInfoForTest(awInfoPath)
            
            var engineMatches bool
            if detectedEngine != nil {
                registry := workflow.GetGlobalEngineRegistry()
                for _, supportedEngine := range constants.AgenticEngines {
                    if testEngine, err := registry.GetEngine(supportedEngine); err == nil && testEngine == detectedEngine {
                        engineMatches = (supportedEngine == engine)
                        break
                    }
                }
            }
            
            t.Logf("engine=%s, detectedEngine=%v, engineMatches=%v", engine, detectedEngine, engineMatches)
            
            if engineMatches != tc.expectMatch {
                t.Errorf("engineMatches=%v, want %v", engineMatches, tc.expectMatch)
            }
        })
    }
}

// extractEngineFromAwInfoForTest mimics the production code
func extractEngineFromAwInfoForTest(infoFilePath string) workflow.CodingAgentEngine {
    data, err := os.ReadFile(infoFilePath)
    if err != nil {
        return nil
    }
    
    var info struct {
        EngineID string `json:"engine_id"`
    }
    if err := json.Unmarshal(data, &info); err != nil {
        return nil
    }
    
    if info.EngineID == "" {
        return nil
    }
    
    registry := workflow.GetGlobalEngineRegistry()
    engine, err := registry.GetEngine(info.EngineID)
    if err != nil {
        return nil
    }
    
    return engine
}
