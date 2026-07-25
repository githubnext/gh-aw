//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractEngineConfig_InlineDriverSource(t *testing.T) {
	c := NewCompiler()

	_, config, _ := c.ExtractEngineConfig(map[string]any{
		"engine": map[string]any{
			"id": "copilot",
			"driver": map[string]any{
				"python": "print('hello')",
			},
		},
	})

	require.NotNil(t, config)
	require.NotNil(t, config.InlineDriver)
	assert.Equal(t, inlineCopilotSDKDriverWrapperPath, config.Driver)
	assert.Equal(t, "python", config.InlineDriver.Runtime)
	assert.Equal(t, "print('hello')", config.InlineDriver.Source)
	assert.True(t, config.CopilotSDK, "inline driver should enable copilot-sdk mode")
}

func TestExtractEngineConfig_InlineDriverEmptySourcePreserved(t *testing.T) {
	// An empty inline source must not be silently dropped; it must be extracted
	// so that validateInlineEngineDriver can reject it with a clear error.
	c := NewCompiler()

	_, config, _ := c.ExtractEngineConfig(map[string]any{
		"engine": map[string]any{
			"id": "copilot",
			"driver": map[string]any{
				"python": "",
			},
		},
	})

	require.NotNil(t, config)
	require.NotNil(t, config.InlineDriver, "empty source should still produce an InlineDriver so validation can catch it")
	assert.Equal(t, "python", config.InlineDriver.Runtime)
	assert.Empty(t, config.InlineDriver.Source)
}

func TestValidateEngineDriver_EmptyInlineSourceIsRejected(t *testing.T) {
	err := NewCompiler().validateEngineDriver(&WorkflowData{
		EngineConfig: &EngineConfig{
			ID: "copilot",
			InlineDriver: &InlineEngineDriver{
				Runtime: "python",
				Source:  "",
			},
			Driver: inlineCopilotSDKDriverWrapperPath,
		},
	})

	require.Error(t, err)
	require.ErrorContains(t, err, "must not be empty")
}

func TestValidateEngineDriver_InlineSourceRejectsNonCopilot(t *testing.T) {
	err := NewCompiler().validateEngineDriver(&WorkflowData{
		EngineConfig: &EngineConfig{
			ID: "claude",
			InlineDriver: &InlineEngineDriver{
				Runtime: "node",
				Source:  "console.log('hi')",
			},
			Driver: inlineCopilotSDKDriverWrapperPath,
		},
	})

	require.Error(t, err)
	require.ErrorContains(t, err, "only supported for the copilot engine")
}

func TestCopilotEngineInstallationWithInlineDriver(t *testing.T) {
	engine := NewCopilotEngine()

	tests := []struct {
		name       string
		runtime    string
		source     string
		expectPath string
		expectRun  string
	}{
		{
			name:       "node",
			runtime:    "node",
			source:     "console.log('hello')",
			expectPath: inlineCopilotSDKDriverNodePath,
			expectRun:  "npm install --ignore-scripts --no-save @github/copilot-sdk@",
		},
		{
			name:       "python",
			runtime:    "python",
			source:     "print('hello')",
			expectPath: inlineCopilotSDKDriverPythonPath,
			expectRun:  "python3 -m pip install --disable-pip-version-check --target",
		},
		{
			name:       "go",
			runtime:    "go",
			source:     "package main",
			expectPath: inlineCopilotSDKDriverGoPath,
			expectRun:  "go get github.com/github/copilot-sdk/go@v",
		},
		{
			name:       "java",
			runtime:    "java",
			source:     "class Main {}",
			expectPath: inlineCopilotSDKDriverJavaPath,
			expectRun:  "mvn -q dependency:build-classpath",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps := engine.GetInstallationSteps(&WorkflowData{
				EngineConfig: &EngineConfig{
					ID:           "copilot",
					CopilotSDK:   true,
					InlineDriver: &InlineEngineDriver{Runtime: tt.runtime, Source: tt.source},
					Driver:       inlineCopilotSDKDriverWrapperPath,
				},
			})

			allSteps := flattenStepText(steps)
			assert.Contains(t, allSteps, "Write Inline Copilot SDK Driver")
			assert.Contains(t, allSteps, tt.expectPath)
			assert.Contains(t, allSteps, inlineCopilotSDKDriverWrapperPath)
			assert.Contains(t, allSteps, tt.expectRun)
		})
	}
}

func TestCopilotEngineExecutionStepsWithInlineDriver(t *testing.T) {
	engine := NewCopilotEngine()
	steps := engine.GetExecutionSteps(&WorkflowData{
		Name: "inline-driver-test",
		EngineConfig: &EngineConfig{
			ID:           "copilot",
			CopilotSDK:   true,
			InlineDriver: &InlineEngineDriver{Runtime: "python", Source: "print('hello')"},
			Driver:       inlineCopilotSDKDriverWrapperPath,
		},
	}, "/tmp/gh-aw/test.log")

	require.Len(t, steps, 1)
	stepContent := strings.Join(steps[0], "\n")
	assert.Contains(t, stepContent, `${GITHUB_WORKSPACE}/`+inlineCopilotSDKDriverWrapperPath)
	assert.Contains(t, stepContent, "PYTHONPATH: ${{ github.workspace }}/.gh-aw/copilot-sdk/python")
	assert.NotContains(t, stepContent, "copilot_sdk_driver.cjs")
}

func TestDetectRuntimeRequirements_InlineDriver(t *testing.T) {
	for _, runtimeID := range []string{"node", "python", "go", "java"} {
		t.Run(runtimeID, func(t *testing.T) {
			requirements := DetectRuntimeRequirements(&WorkflowData{
				EngineConfig: &EngineConfig{
					ID:           "copilot",
					InlineDriver: &InlineEngineDriver{Runtime: runtimeID, Source: "inline"},
				},
			})

			found := false
			for _, req := range requirements {
				if req.Runtime != nil && req.Runtime.ID == runtimeID {
					found = true
					break
				}
			}
			assert.True(t, found, "expected runtime requirement for %s", runtimeID)
		})
	}
}

func TestInlineJavaInstallStep_BootstrapsMaven(t *testing.T) {
	// The Java inline install step must bootstrap Maven when it is not pre-installed,
	// rather than unconditionally invoking mvn and failing on self-hosted runners.
	engine := NewCopilotEngine()

	steps := engine.GetInstallationSteps(&WorkflowData{
		EngineConfig: &EngineConfig{
			ID:           "copilot",
			CopilotSDK:   true,
			InlineDriver: &InlineEngineDriver{Runtime: "java", Source: "class Main {}"},
			Driver:       inlineCopilotSDKDriverWrapperPath,
		},
	})

	allSteps := flattenStepText(steps)
	assert.Contains(t, allSteps, "command -v mvn", "step must check for existing mvn before bootstrapping")
	assert.Contains(t, allSteps, "repo.maven.apache.org", "bootstrap must download from Maven Central")
	assert.Contains(t, allSteps, inlineMavenVersion, "bootstrap must use pinned Maven version")
	assert.Contains(t, allSteps, "mvn -q dependency:build-classpath", "step must run classpath resolution")
}

func TestInlineGoDriverWriteStep_UsesEffectiveGoVersion(t *testing.T) {
	// The generated go.mod must reflect an explicitly pinned runtimes.go.version
	// rather than always emitting the repository-default Go version.
	step := buildInlineCopilotSDKDriverWriteStep(&WorkflowData{
		EngineConfig: &EngineConfig{
			ID:           "copilot",
			CopilotSDK:   true,
			InlineDriver: &InlineEngineDriver{Runtime: "go", Source: "package main"},
			Driver:       inlineCopilotSDKDriverWrapperPath,
		},
		ParsedFrontmatter: &FrontmatterConfig{
			RuntimesTyped: &RuntimesConfig{
				Go: &RuntimeConfig{Version: "1.22"},
			},
		},
	})

	content := strings.Join(step, "\n")
	assert.Contains(t, content, "go 1.22", "go.mod should use the explicitly pinned Go version")
	assert.NotContains(t, content, "go 1.26", "go.mod must not use the default version when an explicit pin is set")
}

func flattenStepText(steps []GitHubActionStep) string {
	var parts []string
	for _, step := range steps {
		parts = append(parts, strings.Join(step, "\n"))
	}
	return strings.Join(parts, "\n")
}
