//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testLogParserScript = `
function parseLog(logContent) {
  const lines = logContent.split("\\n").filter(l => l.trim());
  return {
    markdown: "### Parsed " + lines.length + " lines",
    logEntries: lines.map((l, i) => ({ type: "message", index: i, text: l })),
    mcpFailures: [],
    maxTurnsHit: false,
  };
}
`

func newLogParserEngineDefinition() *EngineDefinition {
	return &EngineDefinition{
		ID:          "testparser",
		DisplayName: "TestParser",
		Description: "A test engine with a log-parser",
		Behaviors: &EngineBehaviorDefinition{
			Execution: &EngineExecutionDefinition{
				CommandName: "testparser-cli",
				Args:        []string{"run"},
				StepName:    "Execute TestParser CLI",
			},
			LogParser: testLogParserScript,
		},
	}
}

func TestBehaviorDefinedEngineLogParser_GetLogParserScriptId(t *testing.T) {
	t.Run("returns script ID when log-parser is set", func(t *testing.T) {
		def := newLogParserEngineDefinition()
		engine, err := NewBehaviorDefinedEngine(def)
		require.NoError(t, err)
		assert.Equal(t, "testparser_log_parser", engine.GetLogParserScriptId())
	})

	t.Run("returns empty when log-parser is not set", func(t *testing.T) {
		def := &EngineDefinition{
			ID:          "nologparser",
			DisplayName: "NoLogParser",
			Behaviors: &EngineBehaviorDefinition{
				Execution: &EngineExecutionDefinition{
					CommandName: "nologparser-cli",
					StepName:    "Execute",
				},
			},
		}
		engine, err := NewBehaviorDefinedEngine(def)
		require.NoError(t, err)
		assert.Empty(t, engine.GetLogParserScriptId())
	})
}

func TestBehaviorDefinedEngineLogParser_WriteStep(t *testing.T) {
	t.Run("write step is included in execution steps", func(t *testing.T) {
		def := newLogParserEngineDefinition()
		engine, err := NewBehaviorDefinedEngine(def)
		require.NoError(t, err)

		workflowData := &WorkflowData{Name: "test"}
		steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")

		// Steps: [log-parser write, execution]
		require.GreaterOrEqual(t, len(steps), 2, "should generate log-parser write step and execution step")

		logParserStepContent := strings.Join(steps[0], "\n")
		assert.Contains(t, logParserStepContent, "Write TestParser log parser script")
		assert.Contains(t, logParserStepContent, "testparser_log_parser.cjs")
		// Write step must run unconditionally so the parse step can always require() the file
		assert.Contains(t, logParserStepContent, "if: always()")
	})

	t.Run("wrapped script contains createEngineLogParser", func(t *testing.T) {
		def := newLogParserEngineDefinition()
		engine, err := NewBehaviorDefinedEngine(def)
		require.NoError(t, err)

		workflowData := &WorkflowData{Name: "test"}
		steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")
		require.GreaterOrEqual(t, len(steps), 2)

		logParserStepContent := strings.Join(steps[0], "\n")
		assert.Contains(t, logParserStepContent, "createEngineLogParser")
		assert.Contains(t, logParserStepContent, "parseFunction: parseLog")
		assert.Contains(t, logParserStepContent, `parserName: "TestParser"`)
	})

	t.Run("no write step when log-parser is empty", func(t *testing.T) {
		def := &EngineDefinition{
			ID:          "noparser",
			DisplayName: "NoParser",
			Behaviors: &EngineBehaviorDefinition{
				Execution: &EngineExecutionDefinition{
					CommandName: "noparser-cli",
					StepName:    "Execute",
				},
			},
		}
		engine, err := NewBehaviorDefinedEngine(def)
		require.NoError(t, err)

		workflowData := &WorkflowData{Name: "test"}
		steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")

		// Only execution step, no log-parser write step
		require.Len(t, steps, 1)
	})
}

func TestBehaviorDefinedEngineLogParser_WithHarnessAndLogParser(t *testing.T) {
	def := &EngineDefinition{
		ID:          "fullengine",
		DisplayName: "FullEngine",
		Behaviors: &EngineBehaviorDefinition{
			Execution: &EngineExecutionDefinition{
				CommandName: "full-cli",
				StepName:    "Execute FullEngine",
			},
			HarnessScript: `const { spawnSync } = require("child_process");`,
			LogParser:     testLogParserScript,
		},
	}
	engine, err := NewBehaviorDefinedEngine(def)
	require.NoError(t, err)

	workflowData := &WorkflowData{Name: "test"}
	steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")

	// Steps: [harness write, log-parser write, execution]
	require.Len(t, steps, 3, "should generate harness write, log-parser write, and execution steps")

	harnessContent := strings.Join(steps[0], "\n")
	assert.Contains(t, harnessContent, "harness script")

	logParserContent := strings.Join(steps[1], "\n")
	assert.Contains(t, logParserContent, "log parser script")

	execContent := strings.Join(steps[2], "\n")
	assert.Contains(t, execContent, "Execute FullEngine")
}

func TestBehaviorDefinedEngineLogParser_HeredocDelimiterSafety(t *testing.T) {
	def := &EngineDefinition{
		ID:          "badparser",
		DisplayName: "BadParser",
		Behaviors: &EngineBehaviorDefinition{
			Execution: &EngineExecutionDefinition{
				CommandName: "bad-cli",
				StepName:    "Execute",
			},
			// Inject the heredoc delimiter into the script
			LogParser: "function parseLog() {}\n" + logParserHeredocDelimiter + "\n// oops",
		},
	}
	engine, err := NewBehaviorDefinedEngine(def)
	require.NoError(t, err)

	// GetLogParserScriptId returns "" when the script contains the heredoc delimiter,
	// keeping it consistent with buildLogParserWriteStep (which also returns nil).
	assert.Empty(t, engine.GetLogParserScriptId(), "script ID should be empty when script contains heredoc delimiter")

	// The write step should also be nil due to heredoc safety check
	step := engine.buildLogParserWriteStep()
	assert.Nil(t, step, "write step should be nil when script contains heredoc delimiter")
}
