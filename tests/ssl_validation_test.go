//go:build !integration

package ssl_validation_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Allowed SSL enum values per .github/skills/ssl/SKILL.md

var allowedSceneTypes = map[string]bool{
	"PREPARE":  true,
	"ACQUIRE":  true,
	"REASON":   true,
	"ACT":      true,
	"VERIFY":   true,
	"RECOVER":  true,
	"FINALIZE": true,
}

var allowedActionTypes = map[string]bool{
	"READ":         true,
	"SELECT":       true,
	"COMPARE":      true,
	"VALIDATE":     true,
	"INFER":        true,
	"WRITE":        true,
	"UPDATE_STATE": true,
	"CALL_TOOL":    true,
	"REQUEST":      true,
	"TRANSFER":     true,
	"NOTIFY":       true,
	"TERMINATE":    true,
}

var allowedResourceScopes = map[string]bool{
	"MEMORY":      true,
	"LOCAL_FS":    true,
	"CODEBASE":    true,
	"PROCESS":     true,
	"USER_DATA":   true,
	"CREDENTIALS": true,
	"NETWORK":     true,
	"OTHER":       true,
}

// Terminal targets for scene transitions.
var sceneTerminalTargets = map[string]bool{
	"END_SUCCESS": true,
	"END_FAIL":    true,
}

// Terminal targets for logic-step transitions.
var stepTerminalTargets = map[string]bool{
	"YIELD_SUCCESS": true,
	"YIELD_FAIL":    true,
}

// sslSkill is the parsed representation of an ssl.json file.
type sslSkill struct {
	Scheduling struct {
		ID         string `json:"id"`
		EntryScene string `json:"entry_scene"`
	} `json:"scheduling"`
	Scenes []struct {
		ID             string `json:"id"`
		Type           string `json:"type"`
		EntryLogicStep string `json:"entry_logic_step"`
		NextSceneRules []struct {
			Condition string `json:"condition"`
			Target    string `json:"target"`
		} `json:"next_scene_rules"`
	} `json:"scenes"`
	LogicSteps []struct {
		ID            string `json:"id"`
		SceneID       string `json:"scene_id"`
		ActionType    string `json:"action_type"`
		ResourceScope string `json:"resource_scope"`
		Next          string `json:"next"`
	} `json:"logic_steps"`
}

// repoRoot walks up from the test file's location to find the repository root.
// NOTE: pkg/parser contains a similar unexported findRepoRoot helper, but it
// cannot be imported here; this is an intentional local copy of the pattern.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "runtime.Caller should succeed")
	dir := filepath.Dir(thisFile)
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repository root")
		}
		dir = parent
	}
}

// loadSSL reads and unmarshals an ssl.json file relative to the repo root.
func loadSSL(t *testing.T, root, relPath string) sslSkill {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, relPath))
	require.NoErrorf(t, err, "reading %s", relPath)
	var skill sslSkill
	require.NoErrorf(t, json.Unmarshal(data, &skill), "parsing %s", relPath)
	return skill
}

// validateSSL runs all four SSL validation rules against a parsed skill.
func validateSSL(t *testing.T, name string, skill sslSkill) {
	t.Helper()

	// Build lookup sets.
	sceneIDs := make(map[string]bool, len(skill.Scenes))
	for _, s := range skill.Scenes {
		sceneIDs[s.ID] = true
	}
	stepIDs := make(map[string]bool, len(skill.LogicSteps))
	for _, ls := range skill.LogicSteps {
		stepIDs[ls.ID] = true
	}

	// Rule 1: entry_scene must reference an existing scene. Fail fast because an
	// invalid entry scene makes the rest of the graph validation unreliable.
	require.Truef(t, sceneIDs[skill.Scheduling.EntryScene],
		"%s: entry_scene %q not found in scenes", name, skill.Scheduling.EntryScene)

	for _, scene := range skill.Scenes {
		// Rule 2: scene type must be an allowed value.
		assert.Truef(t, allowedSceneTypes[scene.Type],
			"%s: scene %q has invalid type %q", name, scene.ID, scene.Type)

		// Rule 3: entry_logic_step must reference an existing logic step. Fail fast
		// because a missing entry step indicates a fundamental structural problem.
		require.Truef(t, stepIDs[scene.EntryLogicStep],
			"%s: scene %q entry_logic_step %q not found in logic_steps", name, scene.ID, scene.EntryLogicStep)

		// Rule 4: scene transition targets must be a scene ID or a terminal target.
		for _, rule := range scene.NextSceneRules {
			validTarget := sceneIDs[rule.Target] || sceneTerminalTargets[rule.Target]
			assert.Truef(t, validTarget,
				"%s: scene %q transition target %q is not a scene ID or END_SUCCESS/END_FAIL", name, scene.ID, rule.Target)
		}
	}

	for _, step := range skill.LogicSteps {
		// Rule 5: action_type must be an allowed value.
		assert.Truef(t, allowedActionTypes[step.ActionType],
			"%s: logic step %q has invalid action_type %q", name, step.ID, step.ActionType)

		// Rule 6: resource_scope must be an allowed value.
		assert.Truef(t, allowedResourceScopes[step.ResourceScope],
			"%s: logic step %q has invalid resource_scope %q", name, step.ID, step.ResourceScope)

		// Rule 7: logic-step next must be a step ID or a terminal target.
		validNext := stepIDs[step.Next] || stepTerminalTargets[step.Next]
		assert.Truef(t, validNext,
			"%s: logic step %q next %q is not a step ID or YIELD_SUCCESS/YIELD_FAIL", name, step.ID, step.Next)
	}
}

// TestSSLSkills_ValidateEnumsAndGraph loads each converted ssl.json and asserts
// that all SSL spec rules (Pass 4) are satisfied:
//  1. Scene types are from the restricted enum.
//  2. Logic-step action types are from the restricted enum.
//  3. Scene transition targets resolve to a known scene ID, END_SUCCESS, or END_FAIL.
//  4. entry_scene and entry_logic_step pointers reference valid IDs.
func TestSSLSkills_ValidateEnumsAndGraph(t *testing.T) {
	root := repoRoot(t)

	skills := []struct {
		name    string
		relPath string
	}{
		{"reporting", ".github/skills/reporting/ssl.json"},
		{"error-messages", ".github/skills/error-messages/ssl.json"},
		{"jqschema", ".github/skills/jqschema/ssl.json"},
	}

	for _, tc := range skills {
		t.Run(tc.name, func(t *testing.T) {
			skill := loadSSL(t, root, tc.relPath)
			validateSSL(t, tc.name, skill)
		})
	}
}
