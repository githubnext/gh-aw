//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func formalRuntimeGateSequenceValid(gates []string) bool {
	expected := []string{
		"concurrency",
		"freshness",
		"repository trust",
		"actor authorization",
		"credential",
		"network boundary",
		"output integrity",
		"termination and audit",
	}
	return len(gates) == len(expected) && strings.Join(gates, "\n") == strings.Join(expected, "\n")
}

func TestFormalCC01_ConcurrencyGroupAlwaysConfigured(t *testing.T) {
	for name, on := range map[string]string{
		"issue":        "on:\n  issues:",
		"pull request": "on:\n  pull_request:",
		"push":         "on:\n  push:",
		"dispatch":     "on:\n  workflow_dispatch:",
	} {
		t.Run(name, func(t *testing.T) {
			config := GenerateConcurrencyConfig(&WorkflowData{On: on}, false)
			assert.NotEmpty(t, config)
			assert.Contains(t, config, "concurrency:\n  group: ")
		})
	}
}

func TestFormalCC02_ConcurrencyGroupIncludesWorkflowIdentity(t *testing.T) {
	config := GenerateConcurrencyConfig(&WorkflowData{On: "on:\n  push:"}, false)

	assert.Contains(t, config, "${{ github.workflow }}",
		"the runtime workflow identity must distinguish concurrency groups")
	assert.NotEqual(t,
		strings.Replace(config, "${{ github.workflow }}", "first-workflow", 1),
		strings.Replace(config, "${{ github.workflow }}", "second-workflow", 1),
	)
}

func TestFormalCC03_CommandTriggerNeverCancelsInProgress(t *testing.T) {
	for name, on := range map[string]string{
		"pull request": "on:\n  pull_request:",
		"issue":        "on:\n  issues:",
		"push":         "on:\n  push:",
	} {
		t.Run(name, func(t *testing.T) {
			assert.False(t, shouldEnableCancelInProgress(&WorkflowData{On: on}, true))
		})
	}
}

func TestFormalCC04_PullRequestWorkflowEnablesCancelInProgress(t *testing.T) {
	workflow := &WorkflowData{On: "on:\n  pull_request:"}

	assert.True(t, shouldEnableCancelInProgress(workflow, false))
	assert.Contains(t, GenerateConcurrencyConfig(workflow, false), "cancel-in-progress: true")
}

func TestFormalCC05_NonPRNonCommandOmitsCancelInProgress(t *testing.T) {
	for name, on := range map[string]string{
		"schedule": "on:\n  schedule:",
		"push":     "on:\n  push:",
		"dispatch": "on:\n  workflow_dispatch:",
	} {
		t.Run(name, func(t *testing.T) {
			workflow := &WorkflowData{On: on}
			assert.False(t, shouldEnableCancelInProgress(workflow, false))
			assert.NotContains(t, GenerateConcurrencyConfig(workflow, false), "cancel-in-progress:")
		})
	}
}

func TestFormalCC06_BotSelfCancelRiskDetection(t *testing.T) {
	appOutputs := &SafeOutputsConfig{GitHubApp: &GitHubAppConfig{}}

	assert.True(t, hasBotSelfCancelRisk(&WorkflowData{
		On:          "on:\n  issue_comment:",
		SafeOutputs: appOutputs,
	}))
	assert.False(t, hasBotSelfCancelRisk(&WorkflowData{
		On:          "on:\n  push:",
		SafeOutputs: appOutputs,
	}))
}

func TestFormalCC07_ConcurrencyGateIsFirstInSequence(t *testing.T) {
	gates := []string{
		"concurrency",
		"freshness",
		"repository trust",
		"actor authorization",
		"credential",
		"network boundary",
		"output integrity",
		"termination and audit",
	}

	assert.True(t, formalRuntimeGateSequenceValid(gates))
	gates[0], gates[1] = gates[1], gates[0]
	assert.False(t, formalRuntimeGateSequenceValid(gates))
}

func TestFormalCC08_GeneratedConcurrencyYAMLShape(t *testing.T) {
	config := GenerateConcurrencyConfig(&WorkflowData{On: "on:\n  issue_comment:"}, true)
	lines := strings.Split(config, "\n")

	assert.Equal(t, "concurrency:", lines[0])
	assert.True(t, strings.HasPrefix(lines[1], "  group: \""))
	assert.True(t, strings.HasSuffix(lines[1], "\""))
	assert.NotContains(t, config, "cancel-in-progress:")
}
