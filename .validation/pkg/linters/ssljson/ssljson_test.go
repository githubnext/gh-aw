//go:build !integration

package ssljson_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/github/gh-aw/pkg/linters/ssljson"
)

// TestSSJAnalyzer_NoFalsePositives confirms the analyzer produces no diagnostics
// when run against a non-anchor package (the testdata fixture).
func TestSSJAnalyzer_NoFalsePositives(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, ssljson.Analyzer, "ssljson")
}

// TestValidateDoc_Valid confirms no errors on a well-formed SSL document.
func TestValidateDoc_Valid(t *testing.T) {
	doc := ssljson.SSLDoc{
		Scheduling: ssljson.SSLScheduling{ID: "example", EntryScene: "scene_act"},
		Scenes: []ssljson.SSLScene{
			{
				ID:             "scene_act",
				Type:           "ACT",
				EntryLogicStep: "step_write",
				NextSceneRules: []ssljson.SSLSceneRule{{Condition: "ok", Target: "END_SUCCESS"}},
			},
		},
		LogicSteps: []ssljson.SSLStep{
			{ID: "step_write", ActionType: "WRITE", ResourceScope: "MEMORY", Next: "YIELD_SUCCESS"},
		},
	}
	assert.Empty(t, ssljson.ValidateDoc(doc), "expected no validation errors for a valid document")
}

// TestValidateDoc_InvalidEntryScene flags a missing entry_scene reference.
func TestValidateDoc_InvalidEntryScene(t *testing.T) {
	doc := ssljson.SSLDoc{
		Scheduling: ssljson.SSLScheduling{ID: "x", EntryScene: "ghost"},
		Scenes: []ssljson.SSLScene{
			{ID: "s", Type: "ACT", EntryLogicStep: "ls",
				NextSceneRules: []ssljson.SSLSceneRule{{Target: "END_SUCCESS"}}},
		},
		LogicSteps: []ssljson.SSLStep{
			{ID: "ls", ActionType: "READ", ResourceScope: "MEMORY", Next: "YIELD_SUCCESS"},
		},
	}
	errs := ssljson.ValidateDoc(doc)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0], `entry_scene "ghost" not found`)
}

// TestValidateDoc_InvalidSceneType flags a scene with an unknown type.
func TestValidateDoc_InvalidSceneType(t *testing.T) {
	doc := ssljson.SSLDoc{
		Scheduling: ssljson.SSLScheduling{ID: "x", EntryScene: "s"},
		Scenes: []ssljson.SSLScene{
			{ID: "s", Type: "UNKNOWN_TYPE", EntryLogicStep: "ls",
				NextSceneRules: []ssljson.SSLSceneRule{{Target: "END_SUCCESS"}}},
		},
		LogicSteps: []ssljson.SSLStep{
			{ID: "ls", ActionType: "READ", ResourceScope: "MEMORY", Next: "YIELD_SUCCESS"},
		},
	}
	errs := ssljson.ValidateDoc(doc)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0], `has invalid type "UNKNOWN_TYPE"`)
}

// TestValidateDoc_InvalidEntryLogicStep flags a scene whose entry_logic_step
// does not reference any existing logic step.
func TestValidateDoc_InvalidEntryLogicStep(t *testing.T) {
	doc := ssljson.SSLDoc{
		Scheduling: ssljson.SSLScheduling{ID: "x", EntryScene: "s"},
		Scenes: []ssljson.SSLScene{
			{ID: "s", Type: "ACT", EntryLogicStep: "missing_step",
				NextSceneRules: []ssljson.SSLSceneRule{{Target: "END_SUCCESS"}}},
		},
		LogicSteps: []ssljson.SSLStep{
			{ID: "ls", ActionType: "READ", ResourceScope: "MEMORY", Next: "YIELD_SUCCESS"},
		},
	}
	errs := ssljson.ValidateDoc(doc)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0], `entry_logic_step "missing_step" not found`)
}

// TestValidateDoc_InvalidTransitionTarget flags a scene with a transition to
// an unknown target.
func TestValidateDoc_InvalidTransitionTarget(t *testing.T) {
	doc := ssljson.SSLDoc{
		Scheduling: ssljson.SSLScheduling{ID: "x", EntryScene: "s"},
		Scenes: []ssljson.SSLScene{
			{ID: "s", Type: "ACT", EntryLogicStep: "ls",
				NextSceneRules: []ssljson.SSLSceneRule{{Target: "nonexistent"}}},
		},
		LogicSteps: []ssljson.SSLStep{
			{ID: "ls", ActionType: "READ", ResourceScope: "MEMORY", Next: "YIELD_SUCCESS"},
		},
	}
	errs := ssljson.ValidateDoc(doc)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0], `transition target "nonexistent"`)
}

// TestValidateDoc_InvalidActionType flags a logic step with an unknown action type.
func TestValidateDoc_InvalidActionType(t *testing.T) {
	doc := ssljson.SSLDoc{
		Scheduling: ssljson.SSLScheduling{ID: "x", EntryScene: "s"},
		Scenes: []ssljson.SSLScene{
			{ID: "s", Type: "ACT", EntryLogicStep: "ls",
				NextSceneRules: []ssljson.SSLSceneRule{{Target: "END_SUCCESS"}}},
		},
		LogicSteps: []ssljson.SSLStep{
			{ID: "ls", ActionType: "DO_MAGIC", ResourceScope: "MEMORY", Next: "YIELD_SUCCESS"},
		},
	}
	errs := ssljson.ValidateDoc(doc)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0], `has invalid action_type "DO_MAGIC"`)
}

// TestValidateDoc_InvalidResourceScope flags a logic step with an unknown
// resource scope.
func TestValidateDoc_InvalidResourceScope(t *testing.T) {
	doc := ssljson.SSLDoc{
		Scheduling: ssljson.SSLScheduling{ID: "x", EntryScene: "s"},
		Scenes: []ssljson.SSLScene{
			{ID: "s", Type: "ACT", EntryLogicStep: "ls",
				NextSceneRules: []ssljson.SSLSceneRule{{Target: "END_SUCCESS"}}},
		},
		LogicSteps: []ssljson.SSLStep{
			{ID: "ls", ActionType: "READ", ResourceScope: "UNIVERSE", Next: "YIELD_SUCCESS"},
		},
	}
	errs := ssljson.ValidateDoc(doc)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0], `has invalid resource_scope "UNIVERSE"`)
}

// TestValidateDoc_InvalidStepNext flags a logic step whose next pointer does
// not resolve to a known step ID or terminal target.
func TestValidateDoc_InvalidStepNext(t *testing.T) {
	doc := ssljson.SSLDoc{
		Scheduling: ssljson.SSLScheduling{ID: "x", EntryScene: "s"},
		Scenes: []ssljson.SSLScene{
			{ID: "s", Type: "ACT", EntryLogicStep: "ls",
				NextSceneRules: []ssljson.SSLSceneRule{{Target: "END_SUCCESS"}}},
		},
		LogicSteps: []ssljson.SSLStep{
			{ID: "ls", ActionType: "READ", ResourceScope: "MEMORY", Next: "nowhere"},
		},
	}
	errs := ssljson.ValidateDoc(doc)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0], `next "nowhere"`)
}
