//go:build !integration

package cli_test

import (
    "testing"
    
    "github.com/github/gh-aw/pkg/constants"
    "github.com/github/gh-aw/pkg/workflow"
)

func TestEngineFilterComparison(t *testing.T) {
    registry := workflow.GetGlobalEngineRegistry()
    
    // Simulate what extractEngineFromAwInfo does when engine_id is "copilot"
    detectedEngine, err := registry.GetEngine("copilot")
    if err != nil {
        t.Fatal("Failed to get copilot engine:", err)
    }
    
    // Simulate the filter loop with engine = "claude"
    filterEngine := "claude"
    var engineMatches bool
    for _, supportedEngine := range constants.AgenticEngines {
        testEngine, err := registry.GetEngine(supportedEngine)
        if err != nil {
            continue
        }
        t.Logf("Comparing supportedEngine=%s: testEngine type=%T, detectedEngine type=%T, equal=%v", supportedEngine, testEngine, detectedEngine, testEngine == detectedEngine)
        if testEngine == detectedEngine {
            engineMatches = (supportedEngine == filterEngine)
            t.Logf("Found match: supportedEngine=%s, filterEngine=%s, engineMatches=%v", supportedEngine, filterEngine, engineMatches)
            break
        }
    }
    
    t.Logf("Final engineMatches=%v (should be false for copilot vs claude filter)", engineMatches)
    
    if engineMatches {
        t.Error("FAIL: copilot run incorrectly matches claude filter!")
    } else {
        t.Log("PASS: copilot run correctly filtered out")
    }
}
